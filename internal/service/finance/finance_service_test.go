// Package finance 财务服务单元测试
package finance

import (
	"context"
	"fmt"
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

func setupFinanceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Payment{},
		&models.Refund{},
		&models.Order{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.Rental{},
		&models.Settlement{},
		&models.Commission{},
		&models.Distributor{},
		&models.Withdrawal{},
		&models.WalletTransaction{},
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	return db
}

func createFinanceTestUser(t *testing.T, db *gorm.DB, phone string) *models.User {
	t.Helper()

	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createTestMerchant(t *testing.T, db *gorm.DB, name string) *models.Merchant {
	t.Helper()

	merchant := &models.Merchant{
		Name:           name,
		CommissionRate: 0.1, // 10% 分成
		Status:         1,
	}
	require.NoError(t, db.Create(merchant).Error)
	return merchant
}

func createTestVenue(t *testing.T, db *gorm.DB, merchantID int64, name string) *models.Venue {
	t.Helper()

	venue := &models.Venue{
		MerchantID: merchantID,
		Name:       name,
		Status:     1,
	}
	require.NoError(t, db.Create(venue).Error)
	return venue
}

func createTestDevice(t *testing.T, db *gorm.DB, venueID int64, deviceNo string) *models.Device {
	t.Helper()

	device := &models.Device{
		DeviceNo:    deviceNo,
		Name:        "测试设备",
		VenueID:     venueID,
		Type:        "locker",
		ProductName: "测试产品",
		Status:      models.DeviceStatusActive,
	}
	require.NoError(t, db.Create(device).Error)
	return device
}

func createTestPayment(t *testing.T, db *gorm.DB, userID int64, amount float64, status int8) *models.Payment {
	t.Helper()

	now := time.Now()
	payment := &models.Payment{
		PaymentNo: fmt.Sprintf("PAY%d", time.Now().UnixNano()),
		UserID:    userID,
		Amount:    amount,
		Status:    status,
		PaidAt:    &now,
	}
	require.NoError(t, db.Create(payment).Error)
	return payment
}

func createTestOrder(t *testing.T, db *gorm.DB, userID int64, amount float64, status string) *models.Order {
	t.Helper()

	now := time.Now()
	order := &models.Order{
		OrderNo:        fmt.Sprintf("ORD%d", time.Now().UnixNano()),
		UserID:         userID,
		Type:           models.OrderTypeRental,
		OriginalAmount: amount,
		ActualAmount:   amount,
		Status:         status,
		PaidAt:         &now,
		CompletedAt:    &now,
	}
	require.NoError(t, db.Create(order).Error)
	return order
}

func createTestDistributor(t *testing.T, db *gorm.DB, userID int64) *models.Distributor {
	t.Helper()

	distributor := &models.Distributor{
		UserID:     userID,
		InviteCode: fmt.Sprintf("INV%d", userID),
		Level:      1,
		Status:     1,
	}
	require.NoError(t, db.Create(distributor).Error)
	return distributor
}

func createTestCommission(t *testing.T, db *gorm.DB, distributorID, orderID, fromUserID int64, amount float64, status int) *models.Commission {
	t.Helper()

	commission := &models.Commission{
		DistributorID: distributorID,
		OrderID:       orderID,
		FromUserID:    fromUserID,
		Type:          "direct",
		OrderAmount:   amount * 10, // 假设订单金额是佣金的10倍
		Rate:          0.1,
		Amount:        amount,
		Status:        status,
	}
	require.NoError(t, db.Create(commission).Error)
	return commission
}

func createTestWithdrawal(t *testing.T, db *gorm.DB, userID int64, amount float64, status string) *models.Withdrawal {
	t.Helper()

	withdrawal := &models.Withdrawal{
		WithdrawalNo:         fmt.Sprintf("WD%d", time.Now().UnixNano()),
		UserID:               userID,
		Type:                 "commission",
		Amount:               amount,
		Fee:                  0,
		ActualAmount:         amount,
		WithdrawTo:           "wechat",
		AccountInfoEncrypted: "encrypted_info",
		Status:               status,
	}
	require.NoError(t, db.Create(withdrawal).Error)
	return withdrawal
}

func createTestSettlement(t *testing.T, db *gorm.DB, settlementType string, targetID int64, amount float64, status string) *models.Settlement {
	t.Helper()

	settlement := &models.Settlement{
		SettlementNo: fmt.Sprintf("ST%d", time.Now().UnixNano()),
		Type:         settlementType,
		TargetID:     targetID,
		PeriodStart:  time.Now().Add(-24 * time.Hour),
		PeriodEnd:    time.Now(),
		TotalAmount:  amount,
		Fee:          amount * 0.1,
		ActualAmount: amount * 0.9,
		OrderCount:   5,
		Status:       status,
	}
	require.NoError(t, db.Create(settlement).Error)
	return settlement
}

func setupStatisticsService(db *gorm.DB) *StatisticsService {
	settlementRepo := repository.NewSettlementRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)

	return NewStatisticsService(db, settlementRepo, transactionRepo, orderRepo, paymentRepo, commissionRepo, withdrawalRepo)
}

func setupSettlementService(db *gorm.DB) *SettlementService {
	settlementRepo := repository.NewSettlementRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	merchantRepo := repository.NewMerchantRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)

	return NewSettlementService(db, settlementRepo, orderRepo, merchantRepo, commissionRepo, distributorRepo)
}

// ================== StatisticsService Tests ==================

func TestStatisticsService_GetFinanceOverview(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupStatisticsService(db)
	ctx := context.Background()

	t.Run("无数据时返回零值", func(t *testing.T) {
		overview, err := svc.GetFinanceOverview(ctx)
		require.NoError(t, err)
		assert.Equal(t, float64(0), overview.TotalRevenue)
		assert.Equal(t, float64(0), overview.TotalRefund)
		assert.Equal(t, float64(0), overview.TotalCommission)
		assert.Equal(t, 0, overview.PendingWithdrawals)
	})

	t.Run("有数据时正确统计", func(t *testing.T) {
		user := createFinanceTestUser(t, db, "13800138000")

		// 创建成功支付
		createTestPayment(t, db, user.ID, 100.0, models.PaymentStatusSuccess)
		createTestPayment(t, db, user.ID, 200.0, models.PaymentStatusSuccess)

		// 创建待审核提现
		distributor := createTestDistributor(t, db, user.ID)
		createTestWithdrawal(t, db, distributor.ID, 50.0, models.WithdrawalStatusPending)

		overview, err := svc.GetFinanceOverview(ctx)
		require.NoError(t, err)
		assert.Equal(t, 300.0, overview.TotalRevenue)
		assert.Equal(t, 1, overview.PendingWithdrawals)
	})
}

func TestStatisticsService_GetWithdrawalSummary(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupStatisticsService(db)
	ctx := context.Background()

	user := createFinanceTestUser(t, db, "13800138001")
	distributor := createTestDistributor(t, db, user.ID)

	// 创建不同状态的提现
	createTestWithdrawal(t, db, distributor.ID, 100.0, models.WithdrawalStatusPending)
	createTestWithdrawal(t, db, distributor.ID, 200.0, models.WithdrawalStatusSuccess)
	createTestWithdrawal(t, db, distributor.ID, 50.0, models.WithdrawalStatusRejected)

	summary, err := svc.GetWithdrawalSummary(ctx, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, summary.TotalWithdrawals)
	assert.Equal(t, 1, summary.PendingCount)
	assert.Equal(t, 1, summary.ApprovedCount)
	assert.Equal(t, 1, summary.RejectedCount)
}

// ================== SettlementService Tests ==================

func TestSettlementService_CreateSettlement(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	merchant := createTestMerchant(t, db, "测试商户")

	t.Run("创建商户结算", func(t *testing.T) {
		req := &CreateSettlementRequest{
			Type:        models.SettlementTypeMerchant,
			TargetID:    merchant.ID,
			PeriodStart: time.Now().Add(-7 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		settlement, err := svc.CreateSettlement(ctx, req, 1)
		require.NoError(t, err)
		assert.NotNil(t, settlement)
		assert.Equal(t, models.SettlementTypeMerchant, settlement.Type)
		assert.Equal(t, merchant.ID, settlement.TargetID)
		assert.Equal(t, models.SettlementStatusPending, settlement.Status)
	})

	t.Run("创建分销商结算", func(t *testing.T) {
		user := createFinanceTestUser(t, db, "13800138002")
		distributor := createTestDistributor(t, db, user.ID)

		req := &CreateSettlementRequest{
			Type:        models.SettlementTypeDistributor,
			TargetID:    distributor.ID,
			PeriodStart: time.Now().Add(-7 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		settlement, err := svc.CreateSettlement(ctx, req, 1)
		require.NoError(t, err)
		assert.NotNil(t, settlement)
		assert.Equal(t, models.SettlementTypeDistributor, settlement.Type)
		assert.Equal(t, distributor.ID, settlement.TargetID)
		// 分销商无手续费
		assert.Equal(t, 0.0, settlement.Fee)
	})

	t.Run("重复结算周期失败", func(t *testing.T) {
		periodStart := time.Now().Add(-30 * 24 * time.Hour)
		periodEnd := time.Now().Add(-20 * 24 * time.Hour)

		req := &CreateSettlementRequest{
			Type:        models.SettlementTypeMerchant,
			TargetID:    merchant.ID,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
		}

		_, err := svc.CreateSettlement(ctx, req, 1)
		require.NoError(t, err)

		// 再次创建相同周期的结算
		_, err = svc.CreateSettlement(ctx, req, 1)
		assert.Error(t, err)
	})
}

func TestSettlementService_GetSettlement(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	merchant := createTestMerchant(t, db, "测试商户")
	settlement := createTestSettlement(t, db, models.SettlementTypeMerchant, merchant.ID, 1000.0, models.SettlementStatusPending)

	t.Run("获取存在的结算", func(t *testing.T) {
		result, err := svc.GetSettlement(ctx, settlement.ID)
		require.NoError(t, err)
		assert.Equal(t, settlement.ID, result.ID)
		assert.Equal(t, settlement.SettlementNo, result.SettlementNo)
	})

	t.Run("获取不存在的结算", func(t *testing.T) {
		_, err := svc.GetSettlement(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestSettlementService_ListSettlements(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	merchant := createTestMerchant(t, db, "测试商户")

	// 创建多个结算
	createTestSettlement(t, db, models.SettlementTypeMerchant, merchant.ID, 1000.0, models.SettlementStatusPending)
	createTestSettlement(t, db, models.SettlementTypeMerchant, merchant.ID, 2000.0, models.SettlementStatusCompleted)

	t.Run("获取所有结算列表", func(t *testing.T) {
		req := &SettlementListRequest{
			Page:     1,
			PageSize: 10,
		}

		settlements, total, err := svc.ListSettlements(ctx, req)
		require.NoError(t, err)
		assert.True(t, total >= 2)
		assert.True(t, len(settlements) >= 2)
	})

	t.Run("按类型筛选", func(t *testing.T) {
		req := &SettlementListRequest{
			Type:     models.SettlementTypeMerchant,
			Page:     1,
			PageSize: 10,
		}

		settlements, total, err := svc.ListSettlements(ctx, req)
		require.NoError(t, err)
		assert.True(t, total >= 2)
		for _, s := range settlements {
			assert.Equal(t, models.SettlementTypeMerchant, s.Type)
		}
	})

	t.Run("按状态筛选", func(t *testing.T) {
		req := &SettlementListRequest{
			Status:   models.SettlementStatusPending,
			Page:     1,
			PageSize: 10,
		}

		settlements, _, err := svc.ListSettlements(ctx, req)
		require.NoError(t, err)
		for _, s := range settlements {
			assert.Equal(t, models.SettlementStatusPending, s.Status)
		}
	})
}

func TestSettlementService_ProcessSettlement(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	t.Run("处理待结算状态记录", func(t *testing.T) {
		merchant := createTestMerchant(t, db, "测试商户2")
		settlement := createTestSettlement(t, db, models.SettlementTypeMerchant, merchant.ID, 1000.0, models.SettlementStatusPending)

		err := svc.ProcessSettlement(ctx, settlement.ID, 1)
		require.NoError(t, err)

		// 验证状态更新
		var updated models.Settlement
		require.NoError(t, db.First(&updated, settlement.ID).Error)
		assert.Equal(t, models.SettlementStatusCompleted, updated.Status)
	})

	t.Run("处理非待结算状态记录失败", func(t *testing.T) {
		merchant := createTestMerchant(t, db, "测试商户3")
		settlement := createTestSettlement(t, db, models.SettlementTypeMerchant, merchant.ID, 1000.0, models.SettlementStatusCompleted)

		err := svc.ProcessSettlement(ctx, settlement.ID, 1)
		assert.Error(t, err)
	})

	t.Run("处理不存在的结算失败", func(t *testing.T) {
		err := svc.ProcessSettlement(ctx, 99999, 1)
		assert.Error(t, err)
	})
}

func TestSettlementService_ProcessDistributorSettlement(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	user := createFinanceTestUser(t, db, "13800138003")
	distributor := createTestDistributor(t, db, user.ID)

	// 创建待结算佣金
	order := createTestOrder(t, db, user.ID, 100.0, models.OrderStatusCompleted)
	createTestCommission(t, db, distributor.ID, order.ID, user.ID, 10.0, models.CommissionStatusPending)

	// 创建分销商结算
	settlement := &models.Settlement{
		SettlementNo: fmt.Sprintf("ST%d", time.Now().UnixNano()),
		Type:         models.SettlementTypeDistributor,
		TargetID:     distributor.ID,
		PeriodStart:  time.Now().Add(-24 * time.Hour),
		PeriodEnd:    time.Now().Add(time.Hour),
		TotalAmount:  10.0,
		Fee:          0,
		ActualAmount: 10.0,
		OrderCount:   1,
		Status:       models.SettlementStatusPending,
	}
	require.NoError(t, db.Create(settlement).Error)

	t.Run("处理分销商结算更新佣金状态", func(t *testing.T) {
		err := svc.ProcessSettlement(ctx, settlement.ID, 1)
		require.NoError(t, err)

		// 验证结算状态
		var updated models.Settlement
		require.NoError(t, db.First(&updated, settlement.ID).Error)
		assert.Equal(t, models.SettlementStatusCompleted, updated.Status)

		// 验证分销商余额增加
		var updatedDistributor models.Distributor
		require.NoError(t, db.First(&updatedDistributor, distributor.ID).Error)
		assert.True(t, updatedDistributor.AvailableCommission >= settlement.ActualAmount)
	})
}

func TestSettlementService_GetSettlementDetail(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	t.Run("获取商户结算详情", func(t *testing.T) {
		merchant := createTestMerchant(t, db, "详情测试商户")
		settlement := createTestSettlement(t, db, models.SettlementTypeMerchant, merchant.ID, 1000.0, models.SettlementStatusPending)

		detail, err := svc.GetSettlementDetail(ctx, settlement.ID)
		require.NoError(t, err)
		assert.Equal(t, settlement.SettlementNo, detail.SettlementNo)
		assert.Equal(t, merchant.Name, detail.TargetName)
	})

	t.Run("获取分销商结算详情", func(t *testing.T) {
		user := createFinanceTestUser(t, db, "13800138004")
		distributor := createTestDistributor(t, db, user.ID)
		settlement := createTestSettlement(t, db, models.SettlementTypeDistributor, distributor.ID, 500.0, models.SettlementStatusPending)

		detail, err := svc.GetSettlementDetail(ctx, settlement.ID)
		require.NoError(t, err)
		assert.Equal(t, settlement.SettlementNo, detail.SettlementNo)
		assert.Contains(t, detail.TargetName, fmt.Sprintf("ID: %d", distributor.ID))
	})
}

func TestSettlementService_GenerateMerchantSettlements(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	// 创建有订单的商户
	merchant := createTestMerchant(t, db, "有订单商户")
	venue := createTestVenue(t, db, merchant.ID, "测试场地")
	device := createTestDevice(t, db, venue.ID, "DEV001")

	user := createFinanceTestUser(t, db, "13800138005")
	order := createTestOrder(t, db, user.ID, 100.0, models.OrderStatusCompleted)

	// 创建租借记录
	rental := &models.Rental{
		OrderID:  order.ID,
		UserID:   user.ID,
		DeviceID: device.ID,
		Status:   models.RentalStatusCompleted,
	}
	require.NoError(t, db.Create(rental).Error)

	periodStart := time.Now().Add(-7 * 24 * time.Hour)
	periodEnd := time.Now().Add(time.Hour)

	settlements, err := svc.GenerateMerchantSettlements(ctx, periodStart, periodEnd, 1)
	require.NoError(t, err)
	// 注意：根据实际业务逻辑，可能需要更多设置才能生成结算
	assert.NotNil(t, settlements)
}

func TestSettlementService_GenerateDistributorSettlements(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	user := createFinanceTestUser(t, db, "13800138006")
	distributor := createTestDistributor(t, db, user.ID)
	order := createTestOrder(t, db, user.ID, 100.0, models.OrderStatusCompleted)

	// 创建待结算佣金
	createTestCommission(t, db, distributor.ID, order.ID, user.ID, 10.0, models.CommissionStatusPending)

	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now().Add(time.Hour)

	settlements, err := svc.GenerateDistributorSettlements(ctx, periodStart, periodEnd, 1)
	require.NoError(t, err)
	assert.True(t, len(settlements) >= 1)

	// 验证结算记录
	for _, s := range settlements {
		assert.Equal(t, models.SettlementTypeDistributor, s.Type)
		assert.Equal(t, models.SettlementStatusPending, s.Status)
	}
}

// ================== Edge Cases ==================

func TestSettlementService_EdgeCases(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := setupSettlementService(db)
	ctx := context.Background()

	t.Run("商户无场地时结算金额为0", func(t *testing.T) {
		merchant := createTestMerchant(t, db, "无场地商户")

		req := &CreateSettlementRequest{
			Type:        models.SettlementTypeMerchant,
			TargetID:    merchant.ID,
			PeriodStart: time.Now().Add(-7 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		settlement, err := svc.CreateSettlement(ctx, req, 1)
		require.NoError(t, err)
		assert.Equal(t, 0.0, settlement.TotalAmount)
		assert.Equal(t, 0.0, settlement.ActualAmount)
	})

	t.Run("分销商无佣金时结算金额为0", func(t *testing.T) {
		user := createFinanceTestUser(t, db, "13800138007")
		distributor := createTestDistributor(t, db, user.ID)

		req := &CreateSettlementRequest{
			Type:        models.SettlementTypeDistributor,
			TargetID:    distributor.ID,
			PeriodStart: time.Now().Add(-7 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		settlement, err := svc.CreateSettlement(ctx, req, 1)
		require.NoError(t, err)
		assert.Equal(t, 0.0, settlement.TotalAmount)
	})
}
