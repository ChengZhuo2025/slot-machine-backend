package distribution

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupWithdrawTestDB 创建提现测试数据库
func setupWithdrawTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Distributor{},
		&models.Withdrawal{},
		&models.Admin{},
	)
	require.NoError(t, err)

	// 创建默认会员等级
	level := &models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
	}
	db.Create(level)

	return db
}

// createWithdrawTestUser 创建提现测试用户
func createWithdrawTestUser(db *gorm.DB) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	return user
}

// createWithdrawTestDistributor 创建提现测试分销商
func createWithdrawTestDistributor(db *gorm.DB, userID int64, availableCommission float64) *models.Distributor {
	distributor := &models.Distributor{
		UserID:              userID,
		Level:               models.DistributorLevelDirect,
		InviteCode:          fmt.Sprintf("W%d", time.Now().UnixNano()%1000000),
		TotalCommission:     availableCommission,
		AvailableCommission: availableCommission,
		FrozenCommission:    0,
		WithdrawnCommission: 0,
		Status:              models.DistributorStatusApproved,
	}
	db.Create(distributor)
	return distributor
}

// createWithdrawTestWallet 创建提现测试钱包
func createWithdrawTestWallet(db *gorm.DB, userID int64, balance float64) *models.UserWallet {
	wallet := &models.UserWallet{
		UserID:        userID,
		Balance:       balance,
		FrozenBalance: 0,
	}
	db.Create(wallet)
	return wallet
}

func TestWithdrawService_Apply(t *testing.T) {
	t.Run("正常申请佣金提现", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		createWithdrawTestDistributor(db, user.ID, 100.0)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      50.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{"openid":"test_openid"}`,
		}
		resp, err := svc.Apply(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Withdrawal)
		assert.Equal(t, 50.0, resp.Withdrawal.Amount)
		assert.Equal(t, models.WithdrawalStatusPending, resp.Withdrawal.Status)

		// 验证手续费计算 50 * 0.006 = 0.3
		expectedFee := 50.0 * DefaultWithdrawFee
		assert.Equal(t, expectedFee, resp.Fee)
		assert.Equal(t, 50.0-expectedFee, resp.ActualAmount)

		// 验证分销商余额被冻结
		var distributor models.Distributor
		db.Where("user_id = ?", user.ID).First(&distributor)
		assert.Equal(t, 50.0, distributor.AvailableCommission)
		assert.Equal(t, 50.0, distributor.FrozenCommission)
	})

	t.Run("正常申请钱包提现", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		createWithdrawTestWallet(db, user.ID, 100.0)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeWallet,
			Amount:      50.0,
			WithdrawTo:  models.WithdrawToAlipay,
			AccountInfo: `{"account":"test@alipay.com"}`,
		}
		resp, err := svc.Apply(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)

		// 验证钱包余额被冻结
		var wallet models.UserWallet
		db.Where("user_id = ?", user.ID).First(&wallet)
		assert.Equal(t, 50.0, wallet.Balance)
		assert.Equal(t, 50.0, wallet.FrozenBalance)
	})

	t.Run("无效提现类型_返回错误", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        "invalid_type",
			Amount:      50.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{}`,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "无效的提现类型")
	})

	t.Run("无效提现方式_返回错误", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      50.0,
			WithdrawTo:  "invalid_method",
			AccountInfo: `{}`,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "无效的提现方式")
	})

	t.Run("金额低于最低提现额_返回错误", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		createWithdrawTestDistributor(db, user.ID, 100.0)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      5.0, // 低于默认最低提现额 10.0
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{}`,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "最低提现金额为")
	})

	t.Run("待处理提现过多_返回错误", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		createWithdrawTestDistributor(db, user.ID, 1000.0)

		// 创建多个待处理提现
		for i := 0; i < MaxPendingWithdraw; i++ {
			withdrawal := &models.Withdrawal{
				WithdrawalNo: fmt.Sprintf("W%d%d", time.Now().UnixNano(), i),
				UserID:       user.ID,
				Type:         models.WithdrawalTypeCommission,
				Amount:       20.0,
				Fee:          0.12,
				ActualAmount: 19.88,
				WithdrawTo:   models.WithdrawToWechat,
				Status:       models.WithdrawalStatusPending,
			}
			db.Create(withdrawal)
		}

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      20.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{}`,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "太多待处理的提现申请")
	})

	t.Run("佣金提现_用户不是分销商_返回错误", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      50.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{}`,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "您还不是分销商")
	})

	t.Run("佣金提现_分销商未审核通过_返回错误", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		// 创建待审核分销商
		distributor := &models.Distributor{
			UserID:              user.ID,
			Level:               models.DistributorLevelDirect,
			InviteCode:          "PEND0002",
			AvailableCommission: 100.0,
			Status:              models.DistributorStatusPending,
		}
		db.Create(distributor)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      50.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{}`,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "分销商尚未审核通过")
	})

	t.Run("佣金余额不足_返回错误", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		createWithdrawTestDistributor(db, user.ID, 30.0)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      50.0, // 超过可用佣金
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{}`,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "可提现余额不足")
	})

	t.Run("钱包余额不足_返回错误", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		createWithdrawTestWallet(db, user.ID, 30.0)

		req := &WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeWallet,
			Amount:      50.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{}`,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "可提现余额不足")
	})
}

func TestWithdrawService_Approve(t *testing.T) {
	t.Run("正常审核通过", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeCommission,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusPending,
		}
		db.Create(withdrawal)

		err := svc.Approve(ctx, withdrawal.ID, 1)
		require.NoError(t, err)

		var updated models.Withdrawal
		db.First(&updated, withdrawal.ID)
		assert.Equal(t, models.WithdrawalStatusApproved, updated.Status)
	})

	t.Run("已处理的提现不能再审核", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeCommission,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusApproved,
		}
		db.Create(withdrawal)

		err := svc.Approve(ctx, withdrawal.ID, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "该提现申请已处理")
	})
}

func TestWithdrawService_Reject(t *testing.T) {
	t.Run("拒绝佣金提现_解冻余额", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		// 创建分销商，模拟已冻结状态
		distributor := &models.Distributor{
			UserID:              user.ID,
			Level:               models.DistributorLevelDirect,
			InviteCode:          "REJECT01",
			TotalCommission:     100.0,
			AvailableCommission: 50.0,
			FrozenCommission:    50.0,
			Status:              models.DistributorStatusApproved,
		}
		db.Create(distributor)

		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeCommission,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusPending,
		}
		db.Create(withdrawal)

		err := svc.Reject(ctx, withdrawal.ID, 1, "不符合条件")
		require.NoError(t, err)

		// 验证状态
		var updated models.Withdrawal
		db.First(&updated, withdrawal.ID)
		assert.Equal(t, models.WithdrawalStatusRejected, updated.Status)
		assert.NotNil(t, updated.RejectReason)
		assert.Equal(t, "不符合条件", *updated.RejectReason)

		// 验证余额已解冻
		var updatedDistributor models.Distributor
		db.First(&updatedDistributor, distributor.ID)
		assert.Equal(t, 100.0, updatedDistributor.AvailableCommission)
		assert.Equal(t, 0.0, updatedDistributor.FrozenCommission)
	})

	t.Run("拒绝钱包提现_解冻余额", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		// 创建钱包，模拟已冻结状态
		wallet := &models.UserWallet{
			UserID:        user.ID,
			Balance:       50.0,
			FrozenBalance: 50.0,
		}
		db.Create(wallet)

		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeWallet,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusPending,
		}
		db.Create(withdrawal)

		err := svc.Reject(ctx, withdrawal.ID, 1, "风控拒绝")
		require.NoError(t, err)

		// 验证余额已解冻
		var updatedWallet models.UserWallet
		db.Where("user_id = ?", user.ID).First(&updatedWallet)
		assert.Equal(t, 100.0, updatedWallet.Balance)
		assert.Equal(t, 0.0, updatedWallet.FrozenBalance)
	})
}

func TestWithdrawService_Process(t *testing.T) {
	t.Run("正常处理提现", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeCommission,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusApproved,
		}
		db.Create(withdrawal)

		err := svc.Process(ctx, withdrawal.ID)
		require.NoError(t, err)

		var updated models.Withdrawal
		db.First(&updated, withdrawal.ID)
		assert.Equal(t, models.WithdrawalStatusProcessing, updated.Status)
	})

	t.Run("非已审核状态不能处理", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeCommission,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusPending,
		}
		db.Create(withdrawal)

		err := svc.Process(ctx, withdrawal.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "状态不正确")
	})
}

func TestWithdrawService_Complete(t *testing.T) {
	t.Run("完成佣金提现", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		// 创建分销商，模拟已冻结状态
		distributor := &models.Distributor{
			UserID:              user.ID,
			Level:               models.DistributorLevelDirect,
			InviteCode:          "COMP0001",
			TotalCommission:     100.0,
			AvailableCommission: 50.0,
			FrozenCommission:    50.0,
			WithdrawnCommission: 0,
			Status:              models.DistributorStatusApproved,
		}
		db.Create(distributor)

		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeCommission,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusProcessing,
		}
		db.Create(withdrawal)

		err := svc.Complete(ctx, withdrawal.ID)
		require.NoError(t, err)

		// 验证状态
		var updated models.Withdrawal
		db.First(&updated, withdrawal.ID)
		assert.Equal(t, models.WithdrawalStatusSuccess, updated.Status)

		// 验证分销商余额变化
		var updatedDistributor models.Distributor
		db.First(&updatedDistributor, distributor.ID)
		assert.Equal(t, 0.0, updatedDistributor.FrozenCommission)
		assert.Equal(t, 50.0, updatedDistributor.WithdrawnCommission)
	})

	t.Run("完成钱包提现", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		// 创建钱包，模拟已冻结状态
		wallet := &models.UserWallet{
			UserID:         user.ID,
			Balance:        50.0,
			FrozenBalance:  50.0,
			TotalWithdrawn: 0,
		}
		db.Create(wallet)

		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeWallet,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusProcessing,
		}
		db.Create(withdrawal)

		err := svc.Complete(ctx, withdrawal.ID)
		require.NoError(t, err)

		// 验证钱包余额变化
		var updatedWallet models.UserWallet
		db.Where("user_id = ?", user.ID).First(&updatedWallet)
		assert.Equal(t, 0.0, updatedWallet.FrozenBalance)
		assert.Equal(t, 49.7, updatedWallet.TotalWithdrawn) // 实际到账金额
	})

	t.Run("非处理中状态不能完成", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createWithdrawTestUser(db)
		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeCommission,
			Amount:       50.0,
			Fee:          0.3,
			ActualAmount: 49.7,
			WithdrawTo:   models.WithdrawToWechat,
			Status:       models.WithdrawalStatusApproved,
		}
		db.Create(withdrawal)

		err := svc.Complete(ctx, withdrawal.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "状态不正确")
	})
}

func TestWithdrawService_SetConfig(t *testing.T) {
	t.Run("设置自定义配置", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)

		svc.SetConfig(50.0, 0.01)

		config := svc.GetConfig()
		assert.Equal(t, 50.0, config["min_withdraw"])
		assert.Equal(t, 0.01, config["withdraw_fee"])
	})
}

func TestWithdrawService_GetConfig(t *testing.T) {
	t.Run("获取默认配置", func(t *testing.T) {
		db := setupWithdrawTestDB(t)
		withdrawalRepo := repository.NewWithdrawalRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)

		config := svc.GetConfig()

		assert.Equal(t, DefaultMinWithdraw, config["min_withdraw"])
		assert.Equal(t, DefaultWithdrawFee, config["withdraw_fee"])
		assert.Equal(t, MaxPendingWithdraw, config["max_pending"])
		assert.NotNil(t, config["support_methods"])
	})
}
