//go:build integration

// Package integration 财务模块集成测试
package integration

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
	financeService "github.com/dumeirei/smart-locker-backend/internal/service/finance"
)

// setupFinanceIntegrationDB 创建测试数据库
func setupFinanceIntegrationDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	// 迁移所需模型
	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.Distributor{},
		&models.Order{},
		&models.Rental{},
		&models.Payment{},
		&models.Refund{},
		&models.Settlement{},
		&models.WalletTransaction{},
		&models.Withdrawal{},
		&models.Commission{},
	)
	require.NoError(t, err)

	return db
}

// setupFinanceServices 初始化财务服务
func setupFinanceServices(db *gorm.DB) (*financeService.SettlementService, *financeService.StatisticsService, *financeService.WithdrawalAuditService, *financeService.ExportService) {
	settlementRepo := repository.NewSettlementRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	merchantRepo := repository.NewMerchantRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)

	settlementSvc := financeService.NewSettlementService(db, settlementRepo, orderRepo, merchantRepo, commissionRepo, distributorRepo)
	statisticsSvc := financeService.NewStatisticsService(db, settlementRepo, transactionRepo, orderRepo, paymentRepo, commissionRepo, withdrawalRepo)
	withdrawalAuditSvc := financeService.NewWithdrawalAuditService(db, withdrawalRepo, distributorRepo)
	exportSvc := financeService.NewExportService(db, settlementRepo, transactionRepo, orderRepo, withdrawalRepo)

	return settlementSvc, statisticsSvc, withdrawalAuditSvc, exportSvc
}

// createTestMerchant 创建测试商户
func createTestMerchant(t *testing.T, db *gorm.DB) *models.Merchant {
	merchant := &models.Merchant{
		Name:           "测试商户",
		ContactName:    "张三",
		ContactPhone:   "13800138000",
		Status:         models.MerchantStatusActive,
		CommissionRate: 0.1, // 10% 平台抽成
	}
	err := db.Create(merchant).Error
	require.NoError(t, err)
	return merchant
}

// createTestDistributor 创建测试分销商
func createTestDistributor(t *testing.T, db *gorm.DB, userID int64) *models.Distributor {
	distributor := &models.Distributor{
		UserID:              userID,
		Level:               1,
		Status:              models.DistributorStatusApproved,
		AvailableCommission: 500.00,
		TotalCommission:     1000.00,
	}
	err := db.Create(distributor).Error
	require.NoError(t, err)
	return distributor
}

// createTestOrder 创建测试订单
func createTestOrder(t *testing.T, db *gorm.DB, userID int64, merchantID int64, amount float64, orderType string) *models.Order {
	now := time.Now()
	order := &models.Order{
		OrderNo:        fmt.Sprintf("ORD%d%d", time.Now().UnixNano(), userID),
		UserID:         userID,
		Type:           orderType,
		Status:         models.OrderStatusCompleted,
		OriginalAmount: amount,
		ActualAmount:   amount,
		PaidAt:         &now,
		CompletedAt:    &now,
	}
	err := db.Create(order).Error
	require.NoError(t, err)
	return order
}

// createTestUser 创建测试用户
func createTestUser(t *testing.T, db *gorm.DB) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Nickname: "测试用户",
		Phone:    &phone,
		Status:   models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 1000.00,
	}
	err = db.Create(wallet).Error
	require.NoError(t, err)

	return user
}

// createTestWithdrawal 创建测试提现申请
func createTestWithdrawal(t *testing.T, db *gorm.DB, userID int64, amount float64, withdrawalType string) *models.Withdrawal {
	withdrawal := &models.Withdrawal{
		WithdrawalNo: fmt.Sprintf("WD%d", time.Now().UnixNano()),
		UserID:       userID,
		Type:         withdrawalType,
		Amount:       amount,
		Fee:          amount * 0.01, // 1% 手续费
		ActualAmount: amount * 0.99,
		Status:       models.WithdrawalStatusPending,
		WithdrawTo:   models.WithdrawToWechat,
	}
	err := db.Create(withdrawal).Error
	require.NoError(t, err)
	return withdrawal
}

// TestSettlementFlow_MerchantSettlement 测试商户结算流程
func TestSettlementFlow_MerchantSettlement(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	settlementSvc, _, _, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	merchant := createTestMerchant(t, db)

	// 创建商户场地和设备（结算统计依赖 rentals -> devices -> venues -> merchants 链路）
	venue := &models.Venue{
		MerchantID: merchant.ID,
		Name:       "测试场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
		Status:     models.VenueStatusActive,
	}
	require.NoError(t, db.Create(venue).Error)

	device := &models.Device{
		DeviceNo:       fmt.Sprintf("DEV%d", time.Now().UnixNano()),
		Name:           "测试设备",
		Type:           models.DeviceTypeStandard,
		VenueID:        venue.ID,
		QRCode:         "test-qrcode",
		ProductName:    "智能柜",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		Status:         models.DeviceStatusActive,
	}
	require.NoError(t, db.Create(device).Error)

	// 创建多个已完成订单
	periodStart := time.Now().AddDate(0, 0, -7)
	periodEnd := time.Now()

	for i := 0; i < 5; i++ {
		order := createTestOrder(t, db, user.ID, merchant.ID, 100.00, models.OrderTypeRental)
		rental := &models.Rental{
			OrderID:       order.ID,
			UserID:        user.ID,
			DeviceID:      device.ID,
			DurationHours: 24,
			RentalFee:     100.00,
			Deposit:       0,
			OvertimeRate:  0,
			Status:        models.RentalStatusCompleted,
		}
		require.NoError(t, db.Create(rental).Error)
	}

	// 1. 创建结算
	req := &financeService.CreateSettlementRequest{
		Type:        models.SettlementTypeMerchant,
		TargetID:    merchant.ID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	}
	settlement, err := settlementSvc.CreateSettlement(ctx, req, 1)
	require.NoError(t, err)
	assert.NotNil(t, settlement)
	assert.Equal(t, models.SettlementStatusPending, settlement.Status)
	assert.Equal(t, models.SettlementTypeMerchant, settlement.Type)
	assert.Equal(t, merchant.ID, settlement.TargetID)

	// 2. 获取结算详情
	detail, err := settlementSvc.GetSettlementDetail(ctx, settlement.ID)
	require.NoError(t, err)
	assert.Equal(t, settlement.SettlementNo, detail.SettlementNo)

	// 3. 处理结算
	err = settlementSvc.ProcessSettlement(ctx, settlement.ID, 1)
	require.NoError(t, err)

	// 验证状态更新
	var updatedSettlement models.Settlement
	err = db.First(&updatedSettlement, settlement.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.SettlementStatusCompleted, updatedSettlement.Status)
}

// TestSettlementFlow_DistributorSettlement 测试分销商结算流程
func TestSettlementFlow_DistributorSettlement(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	settlementSvc, _, _, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	distributor := createTestDistributor(t, db, user.ID)

	periodStart := time.Now().AddDate(0, 0, -7)
	periodEnd := time.Now()

	// 创建佣金记录
	for i := 0; i < 3; i++ {
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       int64(i + 1),
			FromUserID:    user.ID,
			Type:          "direct",
			OrderAmount:   500.00,
			Amount:        50.00,
			Rate:          0.1,
			Status:        models.CommissionStatusSettled,
			SettledAt:     &periodEnd,
		}
		err := db.Create(commission).Error
		require.NoError(t, err)
	}

	// 1. 创建分销商结算
	req := &financeService.CreateSettlementRequest{
		Type:        models.SettlementTypeDistributor,
		TargetID:    distributor.ID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	}
	settlement, err := settlementSvc.CreateSettlement(ctx, req, 1)
	require.NoError(t, err)
	assert.NotNil(t, settlement)
	assert.Equal(t, models.SettlementTypeDistributor, settlement.Type)

	// 2. 处理结算
	err = settlementSvc.ProcessSettlement(ctx, settlement.ID, 1)
	require.NoError(t, err)

	// 验证状态
	var updatedSettlement models.Settlement
	err = db.First(&updatedSettlement, settlement.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.SettlementStatusCompleted, updatedSettlement.Status)
}

// TestSettlementFlow_ListSettlements 测试结算列表查询
func TestSettlementFlow_ListSettlements(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	settlementSvc, _, _, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 创建多个结算记录
	merchant := createTestMerchant(t, db)
	periodStart := time.Now().AddDate(0, 0, -7)
	periodEnd := time.Now()

	for i := 0; i < 3; i++ {
		settlement := &models.Settlement{
			SettlementNo: fmt.Sprintf("SET%d%d", time.Now().UnixNano(), i),
			Type:         models.SettlementTypeMerchant,
			TargetID:     merchant.ID,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
			TotalAmount:  1000.00,
			Fee:          100.00,
			ActualAmount: 900.00,
			OrderCount:   10,
			Status:       models.SettlementStatusPending,
		}
		err := db.Create(settlement).Error
		require.NoError(t, err)
	}

	// 查询列表
	req := &financeService.SettlementListRequest{
		Type:     models.SettlementTypeMerchant,
		Page:     1,
		PageSize: 10,
	}
	settlements, total, err := settlementSvc.ListSettlements(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, settlements, 3)
}

// TestWithdrawalAuditFlow_ApproveAndComplete 测试提现审核通过并完成流程
func TestWithdrawalAuditFlow_ApproveAndComplete(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, _, withdrawalSvc, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	withdrawal := createTestWithdrawal(t, db, user.ID, 100.00, models.WithdrawalTypeWallet)

	// 冻结用户钱包余额
	err := db.Model(&models.UserWallet{}).
		Where("user_id = ?", user.ID).
		Updates(map[string]interface{}{
			"balance":        gorm.Expr("balance - ?", withdrawal.Amount),
			"frozen_balance": gorm.Expr("frozen_balance + ?", withdrawal.Amount),
		}).Error
	require.NoError(t, err)

	// 1. 审核通过
	err = withdrawalSvc.ApproveWithdrawal(ctx, withdrawal.ID, 1)
	require.NoError(t, err)

	// 验证状态
	var approvedWithdrawal models.Withdrawal
	err = db.First(&approvedWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusApproved, approvedWithdrawal.Status)

	// 2. 处理打款
	err = withdrawalSvc.ProcessWithdrawal(ctx, withdrawal.ID, 1)
	require.NoError(t, err)

	var processingWithdrawal models.Withdrawal
	err = db.First(&processingWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusProcessing, processingWithdrawal.Status)

	// 3. 完成提现
	err = withdrawalSvc.CompleteWithdrawal(ctx, withdrawal.ID, 1)
	require.NoError(t, err)

	var completedWithdrawal models.Withdrawal
	err = db.First(&completedWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusSuccess, completedWithdrawal.Status)
	assert.NotNil(t, completedWithdrawal.ProcessedAt)
}

// TestWithdrawalAuditFlow_Reject 测试提现拒绝流程
func TestWithdrawalAuditFlow_Reject(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, _, withdrawalSvc, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	withdrawal := createTestWithdrawal(t, db, user.ID, 100.00, models.WithdrawalTypeWallet)

	// 冻结用户钱包余额
	err := db.Model(&models.UserWallet{}).
		Where("user_id = ?", user.ID).
		Updates(map[string]interface{}{
			"balance":        gorm.Expr("balance - ?", withdrawal.Amount),
			"frozen_balance": gorm.Expr("frozen_balance + ?", withdrawal.Amount),
		}).Error
	require.NoError(t, err)

	// 记录拒绝前的钱包余额
	var walletBefore models.UserWallet
	err = db.Where("user_id = ?", user.ID).First(&walletBefore).Error
	require.NoError(t, err)

	// 拒绝提现
	err = withdrawalSvc.RejectWithdrawal(ctx, withdrawal.ID, 1, "金额超出限制")
	require.NoError(t, err)

	// 验证状态
	var rejectedWithdrawal models.Withdrawal
	err = db.First(&rejectedWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusRejected, rejectedWithdrawal.Status)
	assert.NotNil(t, rejectedWithdrawal.RejectReason)
	assert.Equal(t, "金额超出限制", *rejectedWithdrawal.RejectReason)

	// 验证余额已退还
	var walletAfter models.UserWallet
	err = db.Where("user_id = ?", user.ID).First(&walletAfter).Error
	require.NoError(t, err)
	assert.Equal(t, walletBefore.Balance+withdrawal.Amount, walletAfter.Balance)
	assert.Equal(t, walletBefore.FrozenBalance-withdrawal.Amount, walletAfter.FrozenBalance)
}

// TestWithdrawalAuditFlow_CommissionWithdraw 测试佣金提现流程
func TestWithdrawalAuditFlow_CommissionWithdraw(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, _, withdrawalSvc, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	distributor := createTestDistributor(t, db, user.ID)

	withdrawal := createTestWithdrawal(t, db, user.ID, 100.00, models.WithdrawalTypeCommission)

	// 冻结分销商佣金
	err := db.Model(&models.Distributor{}).
		Where("id = ?", distributor.ID).
		Updates(map[string]interface{}{
			"available_commission": gorm.Expr("available_commission - ?", withdrawal.Amount),
			"frozen_commission":    gorm.Expr("frozen_commission + ?", withdrawal.Amount),
		}).Error
	require.NoError(t, err)

	// 审核通过
	err = withdrawalSvc.ApproveWithdrawal(ctx, withdrawal.ID, 1)
	require.NoError(t, err)

	// 完成提现
	err = withdrawalSvc.CompleteWithdrawal(ctx, withdrawal.ID, 1)
	require.NoError(t, err)

	// 验证分销商佣金更新
	var updatedDistributor models.Distributor
	err = db.First(&updatedDistributor, distributor.ID).Error
	require.NoError(t, err)
	assert.Equal(t, 0.0, updatedDistributor.FrozenCommission)
}

// TestWithdrawalAuditFlow_BatchOperations 测试批量操作
func TestWithdrawalAuditFlow_BatchOperations(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, _, withdrawalSvc, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据：创建多个提现申请
	user := createTestUser(t, db)
	var withdrawalIDs []int64
	for i := 0; i < 3; i++ {
		withdrawal := createTestWithdrawal(t, db, user.ID, 50.00, models.WithdrawalTypeWallet)
		withdrawalIDs = append(withdrawalIDs, withdrawal.ID)
	}

	// 批量审核通过
	err := withdrawalSvc.BatchApprove(ctx, withdrawalIDs, 1)
	require.NoError(t, err)

	// 验证所有提现状态
	for _, id := range withdrawalIDs {
		var w models.Withdrawal
		err := db.First(&w, id).Error
		require.NoError(t, err)
		assert.Equal(t, models.WithdrawalStatusApproved, w.Status)
	}

	// 批量完成
	err = withdrawalSvc.BatchComplete(ctx, withdrawalIDs, 1)
	require.NoError(t, err)

	for _, id := range withdrawalIDs {
		var w models.Withdrawal
		err := db.First(&w, id).Error
		require.NoError(t, err)
		assert.Equal(t, models.WithdrawalStatusSuccess, w.Status)
	}
}

// TestStatisticsFlow_FinanceOverview 测试财务概览
func TestStatisticsFlow_FinanceOverview(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, statisticsSvc, _, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	merchant := createTestMerchant(t, db)

	// 创建订单
	for i := 0; i < 5; i++ {
		createTestOrder(t, db, user.ID, merchant.ID, 100.00, models.OrderTypeRental)
	}

	// 获取财务概览
	overview, err := statisticsSvc.GetFinanceOverview(ctx)
	require.NoError(t, err)
	assert.NotNil(t, overview)
}

// TestStatisticsFlow_RevenueStatistics 测试收入统计
func TestStatisticsFlow_RevenueStatistics(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, statisticsSvc, _, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	merchant := createTestMerchant(t, db)

	// 创建不同类型订单
	createTestOrder(t, db, user.ID, merchant.ID, 100.00, models.OrderTypeRental)
	createTestOrder(t, db, user.ID, merchant.ID, 200.00, models.OrderTypeHotel)
	createTestOrder(t, db, user.ID, merchant.ID, 150.00, models.OrderTypeMall)

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	// 获取收入统计
	stats, err := statisticsSvc.GetRevenueStatistics(ctx, startDate, endDate)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

// TestStatisticsFlow_DailyRevenueReport 测试每日收入报表
func TestStatisticsFlow_DailyRevenueReport(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, statisticsSvc, _, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	merchant := createTestMerchant(t, db)

	// 创建订单
	for i := 0; i < 3; i++ {
		createTestOrder(t, db, user.ID, merchant.ID, 100.00, models.OrderTypeRental)
	}

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	// 获取每日收入报表
	report, err := statisticsSvc.GetDailyRevenueReport(ctx, startDate, endDate)
	require.NoError(t, err)
	assert.NotNil(t, report)
}

// TestExportFlow_ExportSettlements 测试导出结算记录
func TestExportFlow_ExportSettlements(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, _, _, exportSvc := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	merchant := createTestMerchant(t, db)
	periodStart := time.Now().AddDate(0, 0, -7)
	periodEnd := time.Now()

	for i := 0; i < 3; i++ {
		settlement := &models.Settlement{
			SettlementNo: fmt.Sprintf("SET%d%d", time.Now().UnixNano(), i),
			Type:         models.SettlementTypeMerchant,
			TargetID:     merchant.ID,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
			TotalAmount:  1000.00,
			Fee:          100.00,
			ActualAmount: 900.00,
			OrderCount:   10,
			Status:       models.SettlementStatusCompleted,
		}
		err := db.Create(settlement).Error
		require.NoError(t, err)
	}

	// 导出
	req := &financeService.ExportSettlementsRequest{
		Type: models.SettlementTypeMerchant,
	}
	data, filename, err := exportSvc.ExportSettlements(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, filename, "settlements_")
	assert.Contains(t, filename, ".csv")
}

// TestExportFlow_ExportWithdrawals 测试导出提现记录
func TestExportFlow_ExportWithdrawals(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, _, _, exportSvc := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	for i := 0; i < 3; i++ {
		createTestWithdrawal(t, db, user.ID, 100.00, models.WithdrawalTypeWallet)
	}

	// 导出
	req := &financeService.ExportWithdrawalsRequest{
		Type: models.WithdrawalTypeWallet,
	}
	data, filename, err := exportSvc.ExportWithdrawals(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, filename, "withdrawals_")
	assert.Contains(t, filename, ".csv")
}

// TestExportFlow_ExportDailyRevenue 测试导出每日收入报表
func TestExportFlow_ExportDailyRevenue(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, _, _, exportSvc := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)
	merchant := createTestMerchant(t, db)

	for i := 0; i < 3; i++ {
		createTestOrder(t, db, user.ID, merchant.ID, 100.00, models.OrderTypeRental)
	}

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	// 导出
	data, filename, err := exportSvc.ExportDailyRevenue(ctx, startDate, endDate)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, filename, "daily_revenue_")
	assert.Contains(t, filename, ".csv")
}

// TestWithdrawalAuditFlow_InvalidStatusTransitions 测试无效状态转换
func TestWithdrawalAuditFlow_InvalidStatusTransitions(t *testing.T) {
	db := setupFinanceIntegrationDB(t)
	_, _, withdrawalSvc, _ := setupFinanceServices(db)
	ctx := context.Background()

	// 准备数据
	user := createTestUser(t, db)

	t.Run("审核已完成的提现应失败", func(t *testing.T) {
		withdrawal := createTestWithdrawal(t, db, user.ID, 100.00, models.WithdrawalTypeWallet)
		// 直接设置为已完成
		db.Model(withdrawal).Update("status", models.WithdrawalStatusSuccess)

		err := withdrawalSvc.ApproveWithdrawal(ctx, withdrawal.ID, 1)
		assert.Error(t, err)
	})

	t.Run("处理待审核的提现应失败", func(t *testing.T) {
		withdrawal := createTestWithdrawal(t, db, user.ID, 100.00, models.WithdrawalTypeWallet)

		err := withdrawalSvc.ProcessWithdrawal(ctx, withdrawal.ID, 1)
		assert.Error(t, err)
	})

	t.Run("完成待审核的提现应失败", func(t *testing.T) {
		withdrawal := createTestWithdrawal(t, db, user.ID, 100.00, models.WithdrawalTypeWallet)

		err := withdrawalSvc.CompleteWithdrawal(ctx, withdrawal.ID, 1)
		assert.Error(t, err)
	})
}
