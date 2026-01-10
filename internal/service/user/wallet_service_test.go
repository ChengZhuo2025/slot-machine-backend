// Package user 钱包服务单元测试
package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupWalletTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.WalletTransaction{},
		&models.MemberLevel{},
	))

	// 创建默认会员等级
	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	return db
}

func createWalletTestUser(t *testing.T, db *gorm.DB, phone string, balance float64) (*models.User, *models.UserWallet) {
	t.Helper()

	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: balance,
	}
	require.NoError(t, db.Create(wallet).Error)

	return user, wallet
}

func setupWalletService(db *gorm.DB) *WalletService {
	userRepo := repository.NewUserRepository(db)
	return NewWalletService(db, userRepo)
}

func TestWalletService_GetWallet(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("获取已存在的钱包", func(t *testing.T) {
		user, wallet := createWalletTestUser(t, db, "13800138000", 100.0)

		info, err := svc.GetWallet(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, wallet.Balance, info.Balance)
		assert.Equal(t, wallet.FrozenBalance, info.FrozenBalance)
	})

	t.Run("钱包不存在时自动创建", func(t *testing.T) {
		user := &models.User{
			Nickname:      "无钱包用户",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		require.NoError(t, db.Create(user).Error)

		info, err := svc.GetWallet(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, float64(0), info.Balance)

		// 验证钱包已创建
		var wallet models.UserWallet
		err = db.Where("user_id = ?", user.ID).First(&wallet).Error
		require.NoError(t, err)
	})
}

func TestWalletService_Recharge(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("充值成功", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138001", 100.0)

		err := svc.Recharge(ctx, user.ID, 50.0, "ORDER001")
		require.NoError(t, err)

		// 验证余额
		var wallet models.UserWallet
		require.NoError(t, db.Where("user_id = ?", user.ID).First(&wallet).Error)
		assert.Equal(t, 150.0, wallet.Balance)
		assert.Equal(t, 50.0, wallet.TotalRecharged)

		// 验证交易记录
		var tx models.WalletTransaction
		require.NoError(t, db.Where("user_id = ? AND type = ?", user.ID, models.WalletTxTypeRecharge).First(&tx).Error)
		assert.Equal(t, 50.0, tx.Amount)
		assert.Equal(t, 100.0, tx.BalanceBefore)
		assert.Equal(t, 150.0, tx.BalanceAfter)
	})

	t.Run("充值金额为0或负数", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138002", 100.0)

		err := svc.Recharge(ctx, user.ID, 0, "ORDER002")
		assert.Error(t, err)

		err = svc.Recharge(ctx, user.ID, -10, "ORDER003")
		assert.Error(t, err)
	})
}

func TestWalletService_Consume(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("消费成功", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138003", 100.0)

		err := svc.Consume(ctx, user.ID, 30.0, "ORDER004")
		require.NoError(t, err)

		// 验证余额
		var wallet models.UserWallet
		require.NoError(t, db.Where("user_id = ?", user.ID).First(&wallet).Error)
		assert.Equal(t, 70.0, wallet.Balance)
		assert.Equal(t, 30.0, wallet.TotalConsumed)

		// 验证交易记录
		var tx models.WalletTransaction
		require.NoError(t, db.Where("user_id = ? AND type = ?", user.ID, models.WalletTxTypeConsume).First(&tx).Error)
		assert.Equal(t, -30.0, tx.Amount)
	})

	t.Run("余额不足", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138004", 50.0)

		err := svc.Consume(ctx, user.ID, 100.0, "ORDER005")
		assert.Error(t, err)
	})

	t.Run("消费金额为0或负数", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138005", 100.0)

		err := svc.Consume(ctx, user.ID, 0, "ORDER006")
		assert.Error(t, err)
	})
}

func TestWalletService_Refund(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("退款成功", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138006", 100.0)

		err := svc.Refund(ctx, user.ID, 20.0, "ORDER007")
		require.NoError(t, err)

		// 验证余额
		var wallet models.UserWallet
		require.NoError(t, db.Where("user_id = ?", user.ID).First(&wallet).Error)
		assert.Equal(t, 120.0, wallet.Balance)

		// 验证交易记录
		var tx models.WalletTransaction
		require.NoError(t, db.Where("user_id = ? AND type = ?", user.ID, models.WalletTxTypeRefund).First(&tx).Error)
		assert.Equal(t, 20.0, tx.Amount)
	})

	t.Run("退款金额为0或负数", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138007", 100.0)

		err := svc.Refund(ctx, user.ID, 0, "ORDER008")
		assert.Error(t, err)
	})
}

func TestWalletService_FreezeDeposit(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("冻结押金成功", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138008", 100.0)

		err := svc.FreezeDeposit(ctx, user.ID, 30.0, "RENTAL001")
		require.NoError(t, err)

		// 验证余额和冻结金额
		var wallet models.UserWallet
		require.NoError(t, db.Where("user_id = ?", user.ID).First(&wallet).Error)
		assert.Equal(t, 70.0, wallet.Balance)
		assert.Equal(t, 30.0, wallet.FrozenBalance)

		// 验证交易记录
		var tx models.WalletTransaction
		require.NoError(t, db.Where("user_id = ? AND type = ?", user.ID, models.WalletTxTypeDeposit).First(&tx).Error)
		assert.Equal(t, -30.0, tx.Amount)
	})

	t.Run("余额不足冻结", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138009", 20.0)

		err := svc.FreezeDeposit(ctx, user.ID, 50.0, "RENTAL002")
		assert.Error(t, err)
	})
}

func TestWalletService_UnfreezeDeposit(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("解冻押金成功", func(t *testing.T) {
		user, wallet := createWalletTestUser(t, db, "13800138010", 70.0)
		wallet.FrozenBalance = 30.0
		require.NoError(t, db.Save(wallet).Error)

		err := svc.UnfreezeDeposit(ctx, user.ID, 30.0, "RENTAL003")
		require.NoError(t, err)

		// 验证余额和冻结金额
		var updatedWallet models.UserWallet
		require.NoError(t, db.Where("user_id = ?", user.ID).First(&updatedWallet).Error)
		assert.Equal(t, 100.0, updatedWallet.Balance)
		assert.Equal(t, 0.0, updatedWallet.FrozenBalance)

		// 验证交易记录
		var tx models.WalletTransaction
		require.NoError(t, db.Where("user_id = ? AND type = ?", user.ID, models.WalletTxTypeReturnDeposit).First(&tx).Error)
		assert.Equal(t, 30.0, tx.Amount)
	})

	t.Run("冻结余额不足", func(t *testing.T) {
		user, wallet := createWalletTestUser(t, db, "13800138011", 100.0)
		wallet.FrozenBalance = 10.0
		require.NoError(t, db.Save(wallet).Error)

		err := svc.UnfreezeDeposit(ctx, user.ID, 50.0, "RENTAL004")
		assert.Error(t, err)
	})
}

func TestWalletService_CheckBalance(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("余额充足", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138012", 100.0)

		sufficient, err := svc.CheckBalance(ctx, user.ID, 50.0)
		require.NoError(t, err)
		assert.True(t, sufficient)
	})

	t.Run("余额不足", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138013", 30.0)

		sufficient, err := svc.CheckBalance(ctx, user.ID, 50.0)
		require.NoError(t, err)
		assert.False(t, sufficient)
	})

	t.Run("用户钱包不存在", func(t *testing.T) {
		sufficient, err := svc.CheckBalance(ctx, 99999, 10.0)
		require.NoError(t, err)
		assert.False(t, sufficient)
	})
}

func TestWalletService_GetBalance(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("获取余额", func(t *testing.T) {
		user, _ := createWalletTestUser(t, db, "13800138014", 88.88)

		balance, err := svc.GetBalance(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, 88.88, balance)
	})

	t.Run("用户钱包不存在", func(t *testing.T) {
		balance, err := svc.GetBalance(ctx, 99999)
		require.NoError(t, err)
		assert.Equal(t, float64(0), balance)
	})
}

func TestWalletService_DeductFrozenToConsume(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	t.Run("从冻结余额扣款消费成功", func(t *testing.T) {
		user, wallet := createWalletTestUser(t, db, "13800138015", 50.0)
		wallet.FrozenBalance = 30.0
		require.NoError(t, db.Save(wallet).Error)

		err := svc.DeductFrozenToConsume(ctx, user.ID, 20.0, "RENTAL005", "租借费用")
		require.NoError(t, err)

		// 验证冻结余额和消费总额
		var updatedWallet models.UserWallet
		require.NoError(t, db.Where("user_id = ?", user.ID).First(&updatedWallet).Error)
		assert.Equal(t, 50.0, updatedWallet.Balance) // 可用余额不变
		assert.Equal(t, 10.0, updatedWallet.FrozenBalance)
		assert.Equal(t, 20.0, updatedWallet.TotalConsumed)
	})

	t.Run("扣款金额为0或负数时跳过", func(t *testing.T) {
		user, wallet := createWalletTestUser(t, db, "13800138016", 50.0)
		wallet.FrozenBalance = 30.0
		require.NoError(t, db.Save(wallet).Error)

		err := svc.DeductFrozenToConsume(ctx, user.ID, 0, "RENTAL006", "")
		require.NoError(t, err)

		err = svc.DeductFrozenToConsume(ctx, user.ID, -10, "RENTAL007", "")
		require.NoError(t, err)
	})

	t.Run("冻结余额不足", func(t *testing.T) {
		user, wallet := createWalletTestUser(t, db, "13800138017", 50.0)
		wallet.FrozenBalance = 10.0
		require.NoError(t, db.Save(wallet).Error)

		err := svc.DeductFrozenToConsume(ctx, user.ID, 20.0, "RENTAL008", "租借费用")
		assert.Error(t, err)
	})
}

func TestWalletService_GetTransactions(t *testing.T) {
	db := setupWalletTestDB(t)
	svc := setupWalletService(db)
	ctx := context.Background()

	user, _ := createWalletTestUser(t, db, "13800138018", 100.0)

	// 创建多笔交易记录
	transactions := []*models.WalletTransaction{
		{UserID: user.ID, Type: models.WalletTxTypeRecharge, Amount: 100, BalanceBefore: 0, BalanceAfter: 100},
		{UserID: user.ID, Type: models.WalletTxTypeConsume, Amount: -30, BalanceBefore: 100, BalanceAfter: 70},
		{UserID: user.ID, Type: models.WalletTxTypeRefund, Amount: 20, BalanceBefore: 70, BalanceAfter: 90},
	}
	for _, tx := range transactions {
		require.NoError(t, db.Create(tx).Error)
	}

	t.Run("获取所有交易记录", func(t *testing.T) {
		records, total, err := svc.GetTransactions(ctx, user.ID, 0, 10, "")
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, records, 3)
	})

	t.Run("按类型筛选", func(t *testing.T) {
		records, total, err := svc.GetTransactions(ctx, user.ID, 0, 10, models.WalletTxTypeRecharge)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, records, 1)
		assert.Equal(t, models.WalletTxTypeRecharge, records[0].Type)
	})

	t.Run("分页获取", func(t *testing.T) {
		records, total, err := svc.GetTransactions(ctx, user.ID, 0, 2, "")
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, records, 2)
	})
}
