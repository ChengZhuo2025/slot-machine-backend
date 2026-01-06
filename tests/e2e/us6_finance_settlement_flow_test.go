//go:build e2e

// Package e2e 财务模块端到端测试
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	adminHandler "github.com/dumeirei/smart-locker-backend/internal/handler/admin"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	financeService "github.com/dumeirei/smart-locker-backend/internal/service/finance"
)

// setupFinanceE2ETestDB 创建 E2E 测试数据库
func setupFinanceE2ETestDB(t *testing.T) *gorm.DB {
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
		&models.Admin{},
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

// setupFinanceE2ERouter 创建 E2E 测试路由
func setupFinanceE2ERouter(db *gorm.DB, jwtManager *jwt.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 初始化仓储
	settlementRepo := repository.NewSettlementRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	merchantRepo := repository.NewMerchantRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)

	// 初始化服务
	settlementSvc := financeService.NewSettlementService(db, settlementRepo, orderRepo, merchantRepo, commissionRepo, distributorRepo)
	statisticsSvc := financeService.NewStatisticsService(db, settlementRepo, transactionRepo, orderRepo, paymentRepo, commissionRepo, withdrawalRepo)
	withdrawalSvc := financeService.NewWithdrawalAuditService(db, withdrawalRepo, distributorRepo)
	exportSvc := financeService.NewExportService(db, settlementRepo, transactionRepo, orderRepo, withdrawalRepo)

	// 初始化处理器
	financeH := adminHandler.NewFinanceHandler(settlementSvc, statisticsSvc, withdrawalSvc, exportSvc)

	// 注册路由
	admin := r.Group("/api/admin")
	adminAuth := admin.Group("")
	adminAuth.Use(middleware.AdminAuth(jwtManager))
	{
		finance := adminAuth.Group("/finance")
		{
			finance.GET("/overview", financeH.GetOverview)
			finance.GET("/revenue/statistics", financeH.GetRevenueStatistics)
			finance.GET("/revenue/daily", financeH.GetDailyRevenueReport)
			finance.GET("/settlements", financeH.ListSettlements)
			finance.POST("/settlements", financeH.CreateSettlement)
			finance.GET("/settlements/summary", financeH.GetSettlementSummary)
			finance.POST("/settlements/generate", financeH.GenerateSettlements)
			finance.GET("/settlements/:id", financeH.GetSettlement)
			finance.POST("/settlements/:id/process", financeH.ProcessSettlement)
			finance.GET("/withdrawals", financeH.ListWithdrawals)
			finance.GET("/withdrawals/summary", financeH.GetWithdrawalSummary)
			finance.POST("/withdrawals/batch", financeH.BatchHandleWithdrawals)
			finance.GET("/withdrawals/:id", financeH.GetWithdrawal)
			finance.POST("/withdrawals/:id/handle", financeH.HandleWithdrawal)
			finance.GET("/export/settlements", financeH.ExportSettlements)
			finance.GET("/export/withdrawals", financeH.ExportWithdrawals)
			finance.GET("/export/daily-revenue", financeH.ExportDailyRevenue)
		}
	}

	return r
}

// E2ETestContext E2E 测试上下文
type E2ETestContext struct {
	DB         *gorm.DB
	Router     *gin.Engine
	JWTManager *jwt.Manager
	AdminToken string
	Admin      *models.Admin
}

// setupE2EContext 初始化 E2E 测试上下文
func setupE2EContext(t *testing.T) *E2ETestContext {
	db := setupFinanceE2ETestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "e2e-test-secret-key-for-finance",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "e2e-test",
	})
	router := setupFinanceE2ERouter(db, jwtManager)

	// 创建管理员
	admin := &models.Admin{
		Username: "e2e_admin",
		PasswordHash: "hashed_password",
		Name:     "E2E测试管理员",
		Status:   models.AdminStatusActive,
	}
	err := db.Create(admin).Error
	require.NoError(t, err)

	token, _, _ := jwtManager.GenerateAccessToken(admin.ID, jwt.UserTypeAdmin, "")

	return &E2ETestContext{
		DB:         db,
		Router:     router,
		JWTManager: jwtManager,
		AdminToken: token,
		Admin:      admin,
	}
}

// makeRequest 发起 HTTP 请求
func (ctx *E2ETestContext) makeRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(bodyBytes)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Authorization", "Bearer "+ctx.AdminToken)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ctx.Router.ServeHTTP(w, req)
	return w
}

// TestE2E_MerchantSettlementCompleteFlow 测试商户结算完整流程
// 流程：创建订单 → 创建结算 → 查看结算详情 → 处理结算 → 导出结算报表
func TestE2E_MerchantSettlementCompleteFlow(t *testing.T) {
	ctx := setupE2EContext(t)

	// Step 1: 准备测试数据 - 创建商户
	merchant := &models.Merchant{
		Name:           "E2E测试商户",
		ContactName:    "张三",
		ContactPhone:   "13800138000",
		Status:         models.MerchantStatusActive,
		CommissionRate: 0.1,
	}
	err := ctx.DB.Create(merchant).Error
	require.NoError(t, err)

	// Step 2: 创建用户和订单
	phone1 := "13900139000"
	user := &models.User{
		Nickname: "E2E测试用户",
		Phone:    &phone1,
		Status:   models.UserStatusActive,
	}
	err = ctx.DB.Create(user).Error
	require.NoError(t, err)

	// 创建多个已完成订单
	now := time.Now()
	for i := 0; i < 5; i++ {
		order := &models.Order{
			OrderNo:        fmt.Sprintf("E2EORD%d%d", time.Now().UnixNano(), i),
			UserID:         user.ID,
			Type:           models.OrderTypeRental,
			Status:         models.OrderStatusCompleted,
			OriginalAmount: 100.00,
			ActualAmount:   100.00,
			PaidAt:         &now,
		}
		err = ctx.DB.Create(order).Error
		require.NoError(t, err)
	}

	// Step 3: 创建结算
	periodStart := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	periodEnd := time.Now().Format("2006-01-02")

	createReq := map[string]interface{}{
		"type":         "merchant",
		"target_id":    merchant.ID,
		"period_start": periodStart,
		"period_end":   periodEnd,
	}

	w := ctx.makeRequest("POST", "/api/admin/finance/settlements", createReq)
	assert.Equal(t, http.StatusOK, w.Code)

	var createResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), createResp["code"])

	// 提取结算 ID
	data := createResp["data"].(map[string]interface{})
	settlementID := int64(data["id"].(float64))

	// Step 4: 查看结算详情
	w = ctx.makeRequest("GET", fmt.Sprintf("/api/admin/finance/settlements/%d", settlementID), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var detailResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &detailResp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), detailResp["code"])

	// Step 5: 处理结算
	w = ctx.makeRequest("POST", fmt.Sprintf("/api/admin/finance/settlements/%d/process", settlementID), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证结算状态已更新
	var settlement models.Settlement
	err = ctx.DB.First(&settlement, settlementID).Error
	require.NoError(t, err)
	assert.Equal(t, models.SettlementStatusCompleted, settlement.Status)

	// Step 6: 获取结算列表
	w = ctx.makeRequest("GET", "/api/admin/finance/settlements?page=1&page_size=10&type=merchant", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &listResp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), listResp["code"])

	// Step 7: 获取结算汇总
	w = ctx.makeRequest("GET", "/api/admin/finance/settlements/summary?type=merchant", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 8: 导出结算记录
	w = ctx.makeRequest("GET", "/api/admin/finance/export/settlements?type=merchant", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

// TestE2E_WithdrawalAuditCompleteFlow 测试提现审核完整流程
// 流程：创建提现申请 → 获取待审核列表 → 审核通过 → 处理打款 → 完成提现 → 导出提现报表
func TestE2E_WithdrawalAuditCompleteFlow(t *testing.T) {
	ctx := setupE2EContext(t)

	// Step 1: 创建用户和钱包
	phone2 := "13800138001"
	user := &models.User{
		Nickname: "E2E提现测试用户",
		Phone:    &phone2,
		Status:   models.UserStatusActive,
	}
	err := ctx.DB.Create(user).Error
	require.NoError(t, err)

	wallet := &models.UserWallet{
		UserID:        user.ID,
		Balance:       500.00,
		FrozenBalance: 100.00, // 冻结金额（用于提现）
	}
	err = ctx.DB.Create(wallet).Error
	require.NoError(t, err)

	// Step 2: 创建提现申请
	withdrawal := &models.Withdrawal{
		WithdrawalNo: fmt.Sprintf("E2EWD%d", time.Now().UnixNano()),
		UserID:       user.ID,
		Type:         models.WithdrawalTypeWallet,
		Amount:       100.00,
		Fee:          1.00,
		ActualAmount: 99.00,
		Status:       models.WithdrawalStatusPending,
		WithdrawTo:   models.WithdrawToWechat,
	}
	err = ctx.DB.Create(withdrawal).Error
	require.NoError(t, err)

	// Step 3: 获取待审核提现列表
	w := ctx.makeRequest("GET", "/api/admin/finance/withdrawals?status=pending&page=1&page_size=10", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &listResp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), listResp["code"])

	// Step 4: 查看提现详情
	w = ctx.makeRequest("GET", fmt.Sprintf("/api/admin/finance/withdrawals/%d", withdrawal.ID), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 5: 审核通过
	approveReq := map[string]interface{}{
		"action": "approve",
	}
	w = ctx.makeRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), approveReq)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证状态已更新为已审核
	var approvedWithdrawal models.Withdrawal
	err = ctx.DB.First(&approvedWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusApproved, approvedWithdrawal.Status)

	// Step 6: 处理打款
	processReq := map[string]interface{}{
		"action": "process",
	}
	w = ctx.makeRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), processReq)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证状态已更新为打款中
	var processingWithdrawal models.Withdrawal
	err = ctx.DB.First(&processingWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusProcessing, processingWithdrawal.Status)

	// Step 7: 完成提现
	completeReq := map[string]interface{}{
		"action": "complete",
	}
	w = ctx.makeRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), completeReq)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证状态已更新为已完成
	var completedWithdrawal models.Withdrawal
	err = ctx.DB.First(&completedWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusSuccess, completedWithdrawal.Status)
	assert.NotNil(t, completedWithdrawal.ProcessedAt)

	// Step 8: 获取提现汇总
	w = ctx.makeRequest("GET", "/api/admin/finance/withdrawals/summary", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 9: 导出提现记录
	w = ctx.makeRequest("GET", "/api/admin/finance/export/withdrawals", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

// TestE2E_WithdrawalRejectFlow 测试提现拒绝流程
// 流程：创建提现申请 → 拒绝提现 → 验证余额退还
func TestE2E_WithdrawalRejectFlow(t *testing.T) {
	ctx := setupE2EContext(t)

	// Step 1: 创建用户和钱包
	phone3 := "13800138002"
	user := &models.User{
		Nickname: "E2E拒绝测试用户",
		Phone:    &phone3,
		Status:   models.UserStatusActive,
	}
	err := ctx.DB.Create(user).Error
	require.NoError(t, err)

	initialBalance := 500.00
	frozenAmount := 100.00
	wallet := &models.UserWallet{
		UserID:        user.ID,
		Balance:       initialBalance,
		FrozenBalance: frozenAmount,
	}
	err = ctx.DB.Create(wallet).Error
	require.NoError(t, err)

	// Step 2: 创建提现申请
	withdrawal := &models.Withdrawal{
		WithdrawalNo: fmt.Sprintf("E2EWDREJ%d", time.Now().UnixNano()),
		UserID:       user.ID,
		Type:         models.WithdrawalTypeWallet,
		Amount:       frozenAmount,
		Fee:          1.00,
		ActualAmount: 99.00,
		Status:       models.WithdrawalStatusPending,
		WithdrawTo:   models.WithdrawToWechat,
	}
	err = ctx.DB.Create(withdrawal).Error
	require.NoError(t, err)

	// Step 3: 拒绝提现
	rejectReq := map[string]interface{}{
		"action": "reject",
		"reason": "金额超出每日提现限制",
	}
	w := ctx.makeRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), rejectReq)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证状态已更新为已拒绝
	var rejectedWithdrawal models.Withdrawal
	err = ctx.DB.First(&rejectedWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusRejected, rejectedWithdrawal.Status)
	assert.NotNil(t, rejectedWithdrawal.RejectReason)
	assert.Equal(t, "金额超出每日提现限制", *rejectedWithdrawal.RejectReason)

	// 验证钱包余额已退还
	var updatedWallet models.UserWallet
	err = ctx.DB.Where("user_id = ?", user.ID).First(&updatedWallet).Error
	require.NoError(t, err)
	assert.Equal(t, initialBalance+frozenAmount, updatedWallet.Balance)
	assert.Equal(t, 0.0, updatedWallet.FrozenBalance)
}

// TestE2E_BatchWithdrawalApproveFlow 测试批量审核提现流程
func TestE2E_BatchWithdrawalApproveFlow(t *testing.T) {
	ctx := setupE2EContext(t)

	// Step 1: 创建用户
	phone4 := "13800138003"
	user := &models.User{
		Nickname: "E2E批量测试用户",
		Phone:    &phone4,
		Status:   models.UserStatusActive,
	}
	err := ctx.DB.Create(user).Error
	require.NoError(t, err)

	// Step 2: 创建多个提现申请
	var withdrawalIDs []int64
	for i := 0; i < 3; i++ {
		withdrawal := &models.Withdrawal{
			WithdrawalNo: fmt.Sprintf("E2EWDBATCH%d%d", time.Now().UnixNano(), i),
			UserID:       user.ID,
			Type:         models.WithdrawalTypeWallet,
			Amount:       50.00,
			Fee:          0.50,
			ActualAmount: 49.50,
			Status:       models.WithdrawalStatusPending,
			WithdrawTo:   models.WithdrawToWechat,
		}
		err = ctx.DB.Create(withdrawal).Error
		require.NoError(t, err)
		withdrawalIDs = append(withdrawalIDs, withdrawal.ID)
	}

	// Step 3: 批量审核通过
	batchReq := map[string]interface{}{
		"ids":    withdrawalIDs,
		"action": "approve",
	}
	w := ctx.makeRequest("POST", "/api/admin/finance/withdrawals/batch", batchReq)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证所有提现状态已更新
	for _, id := range withdrawalIDs {
		var withdrawal models.Withdrawal
		err = ctx.DB.First(&withdrawal, id).Error
		require.NoError(t, err)
		assert.Equal(t, models.WithdrawalStatusApproved, withdrawal.Status)
	}

	// Step 4: 批量完成
	batchCompleteReq := map[string]interface{}{
		"ids":    withdrawalIDs,
		"action": "complete",
	}
	w = ctx.makeRequest("POST", "/api/admin/finance/withdrawals/batch", batchCompleteReq)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证所有提现状态已完成
	for _, id := range withdrawalIDs {
		var withdrawal models.Withdrawal
		err = ctx.DB.First(&withdrawal, id).Error
		require.NoError(t, err)
		assert.Equal(t, models.WithdrawalStatusSuccess, withdrawal.Status)
	}
}

// TestE2E_FinanceOverviewAndStatistics 测试财务概览和统计
func TestE2E_FinanceOverviewAndStatistics(t *testing.T) {
	ctx := setupE2EContext(t)

	// Step 1: 准备测试数据
	merchant := &models.Merchant{
		Name:           "统计测试商户",
		ContactName:    "李四",
		ContactPhone:   "13800138004",
		Status:         models.MerchantStatusActive,
		CommissionRate: 0.1,
	}
	err := ctx.DB.Create(merchant).Error
	require.NoError(t, err)

	phone5 := "13800138005"
	user := &models.User{
		Nickname: "统计测试用户",
		Phone:    &phone5,
		Status:   models.UserStatusActive,
	}
	err = ctx.DB.Create(user).Error
	require.NoError(t, err)

	// 创建不同类型的订单
	now := time.Now()
	orderTypes := []string{models.OrderTypeRental, models.OrderTypeHotel, models.OrderTypeMall}
	for i, orderType := range orderTypes {
		order := &models.Order{
			OrderNo:        fmt.Sprintf("E2ESTAT%d%d", time.Now().UnixNano(), i),
			UserID:         user.ID,
			Type:           orderType,
			Status:         models.OrderStatusCompleted,
			OriginalAmount: float64(100 * (i + 1)),
			ActualAmount:   float64(100 * (i + 1)),
			PaidAt:         &now,
		}
		err = ctx.DB.Create(order).Error
		require.NoError(t, err)
	}

	// Step 2: 获取财务概览
	w := ctx.makeRequest("GET", "/api/admin/finance/overview", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var overviewResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &overviewResp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), overviewResp["code"])

	// Step 3: 获取收入统计
	startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	w = ctx.makeRequest("GET", fmt.Sprintf("/api/admin/finance/revenue/statistics?start_date=%s&end_date=%s", startDate, endDate), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 4: 获取每日收入报表
	w = ctx.makeRequest("GET", fmt.Sprintf("/api/admin/finance/revenue/daily?start_date=%s&end_date=%s", startDate, endDate), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 5: 导出每日收入报表
	w = ctx.makeRequest("GET", fmt.Sprintf("/api/admin/finance/export/daily-revenue?start_date=%s&end_date=%s", startDate, endDate), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

// TestE2E_DistributorSettlementFlow 测试分销商结算流程
func TestE2E_DistributorSettlementFlow(t *testing.T) {
	ctx := setupE2EContext(t)

	// Step 1: 创建用户和分销商
	phone6 := "13800138006"
	user := &models.User{
		Nickname: "分销商结算测试用户",
		Phone:    &phone6,
		Status:   models.UserStatusActive,
	}
	err := ctx.DB.Create(user).Error
	require.NoError(t, err)

	distributor := &models.Distributor{
		UserID:              user.ID,
		Level:               1,
		Status:              models.DistributorStatusApproved,
		AvailableCommission: 500.00,
		TotalCommission:     1000.00,
	}
	err = ctx.DB.Create(distributor).Error
	require.NoError(t, err)

	// Step 2: 创建佣金记录
	now := time.Now()
	for i := 0; i < 3; i++ {
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       int64(i + 1),
			FromUserID:    user.ID,
			Type:          "direct",
			OrderAmount:   1000.00,
			Amount:        100.00,
			Rate:          0.1,
			Status:        models.CommissionStatusSettled,
			SettledAt:     &now,
		}
		err = ctx.DB.Create(commission).Error
		require.NoError(t, err)
	}

	// Step 3: 创建分销商结算
	periodStart := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	periodEnd := time.Now().Format("2006-01-02")

	createReq := map[string]interface{}{
		"type":         "distributor",
		"target_id":    distributor.ID,
		"period_start": periodStart,
		"period_end":   periodEnd,
	}

	w := ctx.makeRequest("POST", "/api/admin/finance/settlements", createReq)
	assert.Equal(t, http.StatusOK, w.Code)

	var createResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), createResp["code"])

	data := createResp["data"].(map[string]interface{})
	settlementID := int64(data["id"].(float64))

	// Step 4: 处理结算
	w = ctx.makeRequest("POST", fmt.Sprintf("/api/admin/finance/settlements/%d/process", settlementID), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证结算状态
	var settlement models.Settlement
	err = ctx.DB.First(&settlement, settlementID).Error
	require.NoError(t, err)
	assert.Equal(t, models.SettlementStatusCompleted, settlement.Status)
}

// TestE2E_CommissionWithdrawalFlow 测试佣金提现流程
func TestE2E_CommissionWithdrawalFlow(t *testing.T) {
	ctx := setupE2EContext(t)

	// Step 1: 创建用户和分销商
	phone7 := "13800138007"
	user := &models.User{
		Nickname: "佣金提现测试用户",
		Phone:    &phone7,
		Status:   models.UserStatusActive,
	}
	err := ctx.DB.Create(user).Error
	require.NoError(t, err)

	distributor := &models.Distributor{
		UserID:              user.ID,
		Level:               1,
		Status:              models.DistributorStatusApproved,
		AvailableCommission: 400.00,
		FrozenCommission:    100.00, // 冻结用于提现
		TotalCommission:     500.00,
	}
	err = ctx.DB.Create(distributor).Error
	require.NoError(t, err)

	// Step 2: 创建佣金提现申请
	withdrawal := &models.Withdrawal{
		WithdrawalNo: fmt.Sprintf("E2EWDCOMM%d", time.Now().UnixNano()),
		UserID:       user.ID,
		Type:         models.WithdrawalTypeCommission,
		Amount:       100.00,
		Fee:          1.00,
		ActualAmount: 99.00,
		Status:       models.WithdrawalStatusPending,
		WithdrawTo:   models.WithdrawToWechat,
	}
	err = ctx.DB.Create(withdrawal).Error
	require.NoError(t, err)

	// Step 3: 审核通过
	approveReq := map[string]interface{}{
		"action": "approve",
	}
	w := ctx.makeRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), approveReq)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 4: 完成提现
	completeReq := map[string]interface{}{
		"action": "complete",
	}
	w = ctx.makeRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), completeReq)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证提现状态
	var completedWithdrawal models.Withdrawal
	err = ctx.DB.First(&completedWithdrawal, withdrawal.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.WithdrawalStatusSuccess, completedWithdrawal.Status)

	// 验证分销商佣金更新
	var updatedDistributor models.Distributor
	err = ctx.DB.First(&updatedDistributor, distributor.ID).Error
	require.NoError(t, err)
	assert.Equal(t, 0.0, updatedDistributor.FrozenCommission)
}
