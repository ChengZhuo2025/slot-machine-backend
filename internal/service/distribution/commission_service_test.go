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

// setupCommissionTestDB 创建测试数据库
func setupCommissionTestDB(t *testing.T) *gorm.DB {
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
		&models.Order{},
		&models.Distributor{},
		&models.Commission{},
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

// createTestUser 创建测试用户
func createTestUser(db *gorm.DB, referrerID *int64) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		ReferrerID:    referrerID,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	return user
}

// createTestDistributor 创建测试分销商
func createTestDistributor(db *gorm.DB, userID int64, parentID *int64, status int) *models.Distributor {
	distributor := &models.Distributor{
		UserID:              userID,
		ParentID:            parentID,
		Level:               models.DistributorLevelDirect,
		InviteCode:          fmt.Sprintf("INV%d", time.Now().UnixNano()%1000000),
		TotalCommission:     0,
		AvailableCommission: 0,
		FrozenCommission:    0,
		WithdrawnCommission: 0,
		TeamCount:           0,
		DirectCount:         0,
		Status:              status,
	}
	if parentID != nil {
		distributor.Level = models.DistributorLevelIndirect
	}
	db.Create(distributor)
	return distributor
}

// createTestOrder 创建测试订单
func createTestOrder(db *gorm.DB, userID int64, amount float64) *models.Order {
	order := &models.Order{
		OrderNo:        fmt.Sprintf("O%s%06d", time.Now().Format("20060102150405"), time.Now().UnixNano()%1000000),
		UserID:         userID,
		Type:           models.OrderTypeMall,
		OriginalAmount: amount,
		DiscountAmount: 0,
		ActualAmount:   amount,
		Status:         models.OrderStatusCompleted,
	}
	db.Create(order)
	return order
}

func TestCommissionService_Calculate(t *testing.T) {
	t.Run("用户无推荐人_不计算佣金", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建无推荐人的用户
		user := createTestUser(db, nil)
		order := createTestOrder(db, user.ID, 100.0)

		// 计算佣金
		req := &CalculateRequest{
			OrderID:     order.ID,
			UserID:      user.ID,
			OrderAmount: order.ActualAmount,
		}
		resp, err := svc.Calculate(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, float64(0), resp.TotalAmount)
		assert.Nil(t, resp.DirectCommission)
		assert.Nil(t, resp.IndirectCommission)
	})

	t.Run("推荐人非分销商_不计算佣金", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建推荐人（非分销商）
		referrer := createTestUser(db, nil)
		// 创建被推荐人
		user := createTestUser(db, &referrer.ID)
		order := createTestOrder(db, user.ID, 100.0)

		// 计算佣金
		req := &CalculateRequest{
			OrderID:     order.ID,
			UserID:      user.ID,
			OrderAmount: order.ActualAmount,
		}
		resp, err := svc.Calculate(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, float64(0), resp.TotalAmount)
		assert.Nil(t, resp.DirectCommission)
		assert.Nil(t, resp.IndirectCommission)
	})

	t.Run("推荐人是待审核分销商_不计算佣金", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建推荐人（待审核分销商）
		referrer := createTestUser(db, nil)
		createTestDistributor(db, referrer.ID, nil, models.DistributorStatusPending)

		// 创建被推荐人
		user := createTestUser(db, &referrer.ID)
		order := createTestOrder(db, user.ID, 100.0)

		// 计算佣金
		req := &CalculateRequest{
			OrderID:     order.ID,
			UserID:      user.ID,
			OrderAmount: order.ActualAmount,
		}
		resp, err := svc.Calculate(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, float64(0), resp.TotalAmount)
	})

	t.Run("推荐人是已通过分销商_计算直推佣金", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建推荐人（已审核通过分销商）
		referrer := createTestUser(db, nil)
		directDistributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

		// 创建被推荐人
		user := createTestUser(db, &referrer.ID)
		order := createTestOrder(db, user.ID, 100.0)

		// 计算佣金
		req := &CalculateRequest{
			OrderID:     order.ID,
			UserID:      user.ID,
			OrderAmount: order.ActualAmount,
		}
		resp, err := svc.Calculate(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)

		// 验证直推佣金 100 * 10% = 10
		expectedDirectAmount := 100.0 * DefaultDirectRate
		assert.Equal(t, expectedDirectAmount, resp.TotalAmount)
		assert.NotNil(t, resp.DirectCommission)
		assert.Equal(t, directDistributor.ID, resp.DirectCommission.DistributorID)
		assert.Equal(t, models.CommissionTypeDirect, resp.DirectCommission.Type)
		assert.Equal(t, expectedDirectAmount, resp.DirectCommission.Amount)
		assert.Equal(t, DefaultDirectRate, resp.DirectCommission.Rate)
		assert.Equal(t, models.CommissionStatusPending, resp.DirectCommission.Status)
		assert.Nil(t, resp.IndirectCommission)

		// 验证佣金记录已保存
		var commissions []*models.Commission
		db.Where("order_id = ?", order.ID).Find(&commissions)
		assert.Len(t, commissions, 1)
	})

	t.Run("推荐人有上级分销商_计算直推和间推佣金", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建间推分销商（上级）
		indirectReferrer := createTestUser(db, nil)
		indirectDistributor := createTestDistributor(db, indirectReferrer.ID, nil, models.DistributorStatusApproved)

		// 创建直推分销商（下级，有上级）
		directReferrer := createTestUser(db, &indirectReferrer.ID)
		directDistributor := createTestDistributor(db, directReferrer.ID, &indirectDistributor.ID, models.DistributorStatusApproved)

		// 创建被推荐人
		user := createTestUser(db, &directReferrer.ID)
		order := createTestOrder(db, user.ID, 100.0)

		// 计算佣金
		req := &CalculateRequest{
			OrderID:     order.ID,
			UserID:      user.ID,
			OrderAmount: order.ActualAmount,
		}
		resp, err := svc.Calculate(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)

		// 验证直推佣金 100 * 10% = 10
		expectedDirectAmount := 100.0 * DefaultDirectRate
		// 验证间推佣金 100 * 5% = 5
		expectedIndirectAmount := 100.0 * DefaultIndirectRate
		expectedTotalAmount := expectedDirectAmount + expectedIndirectAmount

		assert.Equal(t, expectedTotalAmount, resp.TotalAmount)

		// 验证直推佣金
		assert.NotNil(t, resp.DirectCommission)
		assert.Equal(t, directDistributor.ID, resp.DirectCommission.DistributorID)
		assert.Equal(t, models.CommissionTypeDirect, resp.DirectCommission.Type)
		assert.Equal(t, expectedDirectAmount, resp.DirectCommission.Amount)

		// 验证间推佣金
		assert.NotNil(t, resp.IndirectCommission)
		assert.Equal(t, indirectDistributor.ID, resp.IndirectCommission.DistributorID)
		assert.Equal(t, models.CommissionTypeIndirect, resp.IndirectCommission.Type)
		assert.Equal(t, expectedIndirectAmount, resp.IndirectCommission.Amount)

		// 验证佣金记录已保存
		var commissions []*models.Commission
		db.Where("order_id = ?", order.ID).Find(&commissions)
		assert.Len(t, commissions, 2)
	})

	t.Run("订单金额无效_返回错误", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createTestUser(db, nil)

		req := &CalculateRequest{
			OrderID:     1,
			UserID:      user.ID,
			OrderAmount: 0,
		}
		resp, err := svc.Calculate(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "订单金额无效")
	})

	t.Run("负数订单金额_返回错误", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createTestUser(db, nil)

		req := &CalculateRequest{
			OrderID:     1,
			UserID:      user.ID,
			OrderAmount: -100,
		}
		resp, err := svc.Calculate(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestCommissionService_Settle(t *testing.T) {
	t.Run("正常结算待结算佣金", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建分销商
		referrer := createTestUser(db, nil)
		distributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

		// 创建佣金记录
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    2,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          DefaultDirectRate,
			Amount:        10.0,
			Status:        models.CommissionStatusPending,
		}
		db.Create(commission)

		// 结算佣金
		err := svc.Settle(ctx, commission.ID)
		require.NoError(t, err)

		// 验证佣金状态
		var updatedCommission models.Commission
		db.First(&updatedCommission, commission.ID)
		assert.Equal(t, models.CommissionStatusSettled, updatedCommission.Status)
		assert.NotNil(t, updatedCommission.SettledAt)

		// 验证分销商佣金增加
		var updatedDistributor models.Distributor
		db.First(&updatedDistributor, distributor.ID)
		assert.Equal(t, 10.0, updatedDistributor.TotalCommission)
		assert.Equal(t, 10.0, updatedDistributor.AvailableCommission)
	})

	t.Run("已结算佣金不能重复结算", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		referrer := createTestUser(db, nil)
		distributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

		now := time.Now()
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    2,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          DefaultDirectRate,
			Amount:        10.0,
			Status:        models.CommissionStatusSettled,
			SettledAt:     &now,
		}
		db.Create(commission)

		err := svc.Settle(ctx, commission.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "该佣金已处理")
	})
}

func TestCommissionService_CancelByOrderID(t *testing.T) {
	t.Run("取消待结算佣金", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		referrer := createTestUser(db, nil)
		distributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    2,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          DefaultDirectRate,
			Amount:        10.0,
			Status:        models.CommissionStatusPending,
		}
		db.Create(commission)

		err := svc.CancelByOrderID(ctx, 1)
		require.NoError(t, err)

		// 验证佣金状态
		var updatedCommission models.Commission
		db.First(&updatedCommission, commission.ID)
		assert.Equal(t, models.CommissionStatusCancelled, updatedCommission.Status)
	})

	t.Run("取消已结算佣金_扣减分销商余额", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		referrer := createTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:              referrer.ID,
			Level:               models.DistributorLevelDirect,
			InviteCode:          "INV001",
			TotalCommission:     50.0,
			AvailableCommission: 50.0,
			FrozenCommission:    0,
			WithdrawnCommission: 0,
			Status:              models.DistributorStatusApproved,
		}
		db.Create(distributor)

		now := time.Now()
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    2,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          DefaultDirectRate,
			Amount:        10.0,
			Status:        models.CommissionStatusSettled,
			SettledAt:     &now,
		}
		db.Create(commission)

		err := svc.CancelByOrderID(ctx, 1)
		require.NoError(t, err)

		// 验证分销商余额被扣减
		var updatedDistributor models.Distributor
		db.First(&updatedDistributor, distributor.ID)
		assert.Equal(t, 40.0, updatedDistributor.TotalCommission)
		assert.Equal(t, 40.0, updatedDistributor.AvailableCommission)
	})

	t.Run("订单无佣金记录_正常返回", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		err := svc.CancelByOrderID(ctx, 999)
		require.NoError(t, err)
	})
}

func TestCommissionService_SetRates(t *testing.T) {
	t.Run("设置自定义佣金比例", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		// 设置自定义比例
		svc.SetRates(0.15, 0.08, 14)

		// 创建分销商
		referrer := createTestUser(db, nil)
		createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

		// 创建消费用户
		user := createTestUser(db, &referrer.ID)
		order := createTestOrder(db, user.ID, 100.0)

		req := &CalculateRequest{
			OrderID:     order.ID,
			UserID:      user.ID,
			OrderAmount: order.ActualAmount,
		}
		resp, err := svc.Calculate(ctx, req)

		require.NoError(t, err)
		// 验证使用自定义比例 100 * 15% = 15
		assert.Equal(t, 15.0, resp.TotalAmount)
		assert.Equal(t, 0.15, resp.DirectCommission.Rate)
	})
}

func TestCommissionService_GetByDistributorID(t *testing.T) {
	t.Run("获取分销商佣金记录", func(t *testing.T) {
		db := setupCommissionTestDB(t)
		commissionRepo := repository.NewCommissionRepository(db)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
		ctx := context.Background()

		referrer := createTestUser(db, nil)
		distributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

		// 创建多条佣金记录
		for i := 0; i < 5; i++ {
			commission := &models.Commission{
				DistributorID: distributor.ID,
				OrderID:       int64(i + 1),
				FromUserID:    int64(i + 100),
				Type:          models.CommissionTypeDirect,
				OrderAmount:   100.0,
				Rate:          DefaultDirectRate,
				Amount:        10.0,
				Status:        models.CommissionStatusPending,
			}
			db.Create(commission)
		}

		commissions, total, err := svc.GetByDistributorID(ctx, distributor.ID, 0, 10)
		require.NoError(t, err)
		assert.Len(t, commissions, 5)
		assert.Equal(t, int64(5), total)
	})
}

func TestCommissionService_GetStats(t *testing.T) {
	db := setupCommissionTestDB(t)
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
	ctx := context.Background()

	referrer := createTestUser(db, nil)
	distributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

	// 创建佣金记录
	commission := &models.Commission{
		DistributorID: distributor.ID,
		OrderID:       1,
		FromUserID:    2,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100.0,
		Rate:          DefaultDirectRate,
		Amount:        10.0,
		Status:        models.CommissionStatusSettled,
	}
	db.Create(commission)

	stats, err := svc.GetStats(ctx, distributor.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestCommissionService_List(t *testing.T) {
	db := setupCommissionTestDB(t)
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
	ctx := context.Background()

	referrer := createTestUser(db, nil)
	distributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

	// 创建佣金记录
	for i := 0; i < 3; i++ {
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       int64(i + 1),
			FromUserID:    int64(i + 100),
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          DefaultDirectRate,
			Amount:        10.0,
			Status:        models.CommissionStatusPending,
		}
		db.Create(commission)
	}

	commissions, total, err := svc.List(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Len(t, commissions, 3)
	assert.Equal(t, int64(3), total)
}

func TestCommissionService_GetByOrderID(t *testing.T) {
	db := setupCommissionTestDB(t)
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
	ctx := context.Background()

	referrer := createTestUser(db, nil)
	distributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

	// 创建佣金记录
	commission := &models.Commission{
		DistributorID: distributor.ID,
		OrderID:       123,
		FromUserID:    456,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100.0,
		Rate:          DefaultDirectRate,
		Amount:        10.0,
		Status:        models.CommissionStatusPending,
	}
	db.Create(commission)

	commissions, err := svc.GetByOrderID(ctx, 123)
	require.NoError(t, err)
	assert.Len(t, commissions, 1)
	assert.Equal(t, int64(123), commissions[0].OrderID)
}

func TestCommissionService_SettlePendingCommissions(t *testing.T) {
	db := setupCommissionTestDB(t)
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
	ctx := context.Background()

	// 设置结算延迟为0天（便于测试）
	svc.SetRates(DefaultDirectRate, DefaultIndirectRate, 0)

	referrer := createTestUser(db, nil)
	distributor := createTestDistributor(db, referrer.ID, nil, models.DistributorStatusApproved)

	// 创建待结算的佣金记录（创建时间在过去）
	commission := &models.Commission{
		DistributorID: distributor.ID,
		OrderID:       1,
		FromUserID:    2,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100.0,
		Rate:          DefaultDirectRate,
		Amount:        10.0,
		Status:        models.CommissionStatusPending,
	}
	db.Create(commission)
	// 手动设置创建时间为过去
	db.Model(commission).Update("created_at", time.Now().AddDate(0, 0, -1))

	count, err := svc.SettlePendingCommissions(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// 验证佣金状态
	var updated models.Commission
	db.First(&updated, commission.ID)
	assert.Equal(t, models.CommissionStatusSettled, updated.Status)
}
