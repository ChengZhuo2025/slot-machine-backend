//go:build api

// Package api 财务模块 API 测试
package api

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

// setupFinanceAPITestDB 创建测试数据库
func setupFinanceAPITestDB(t *testing.T) *gorm.DB {
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
		&models.Distributor{},
		&models.Order{},
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

// setupFinanceAPITestRouter 创建测试路由
func setupFinanceAPITestRouter(db *gorm.DB, jwtManager *jwt.Manager) *gin.Engine {
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
			// 概览和统计
			finance.GET("/overview", financeH.GetOverview)
			finance.GET("/revenue/statistics", financeH.GetRevenueStatistics)
			finance.GET("/revenue/daily", financeH.GetDailyRevenueReport)
			finance.GET("/revenue/by-type", financeH.GetOrderRevenueByType)
			finance.GET("/transactions/statistics", financeH.GetTransactionStatistics)

			// 结算管理
			finance.GET("/settlements", financeH.ListSettlements)
			finance.POST("/settlements", financeH.CreateSettlement)
			finance.GET("/settlements/summary", financeH.GetSettlementSummary)
			finance.POST("/settlements/generate", financeH.GenerateSettlements)
			finance.GET("/settlements/:id", financeH.GetSettlement)
			finance.POST("/settlements/:id/process", financeH.ProcessSettlement)

			// 提现管理
			finance.GET("/withdrawals", financeH.ListWithdrawals)
			finance.GET("/withdrawals/summary", financeH.GetWithdrawalSummary)
			finance.POST("/withdrawals/batch", financeH.BatchHandleWithdrawals)
			finance.GET("/withdrawals/:id", financeH.GetWithdrawal)
			finance.POST("/withdrawals/:id/handle", financeH.HandleWithdrawal)

			// 报表
			finance.GET("/reports/merchant-settlement", financeH.GetMerchantSettlementReport)

			// 导出
			finance.GET("/export/settlements", financeH.ExportSettlements)
			finance.GET("/export/withdrawals", financeH.ExportWithdrawals)
			finance.GET("/export/daily-revenue", financeH.ExportDailyRevenue)
			finance.GET("/export/merchant-settlement", financeH.ExportMerchantSettlement)
			finance.GET("/export/transactions", financeH.ExportTransactions)
		}
	}

	return r
}

// generateAdminTestToken 生成管理员测试 Token
func generateAdminTestToken(jwtManager *jwt.Manager, adminID int64) string {
	token, _, _ := jwtManager.GenerateAccessToken(adminID, jwt.UserTypeAdmin, "")
	return token
}

// createFinanceTestAdmin 创建测试管理员
func createFinanceTestAdmin(t *testing.T, db *gorm.DB) *models.Admin {
	admin := &models.Admin{
		Username: fmt.Sprintf("admin_%d", time.Now().UnixNano()),
		Password: "hashed_password",
		Name:     "测试管理员",
		Status:   models.AdminStatusActive,
	}
	err := db.Create(admin).Error
	require.NoError(t, err)
	return admin
}

// createFinanceTestMerchant 创建测试商户
func createFinanceTestMerchant(t *testing.T, db *gorm.DB) *models.Merchant {
	merchant := &models.Merchant{
		Name:           "测试商户",
		ContactName:    "张三",
		ContactPhone:   "13800138000",
		Status:         models.MerchantStatusActive,
		CommissionRate: 0.1,
	}
	err := db.Create(merchant).Error
	require.NoError(t, err)
	return merchant
}

// createFinanceTestUser 创建测试用户
func createFinanceTestUser(t *testing.T, db *gorm.DB) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Nickname: "测试用户",
		Phone:    &phone,
		Status:   models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 1000.00,
	}
	err = db.Create(wallet).Error
	require.NoError(t, err)

	return user
}

// createFinanceTestOrder 创建测试订单
func createFinanceTestOrder(t *testing.T, db *gorm.DB, userID int64, merchantID int64, amount float64, orderType string) *models.Order {
	now := time.Now()
	order := &models.Order{
		OrderNo:        fmt.Sprintf("ORD%d%d", time.Now().UnixNano(), userID),
		UserID:         userID,
		Type:           orderType,
		Status:         models.OrderStatusCompleted,
		OriginalAmount: amount,
		ActualAmount:   amount,
		PaidAt:         &now,
	}
	err := db.Create(order).Error
	require.NoError(t, err)
	return order
}

// createFinanceTestSettlement 创建测试结算记录
func createFinanceTestSettlement(t *testing.T, db *gorm.DB, merchantID int64) *models.Settlement {
	settlement := &models.Settlement{
		SettlementNo: fmt.Sprintf("SET%d", time.Now().UnixNano()),
		Type:         models.SettlementTypeMerchant,
		TargetID:     merchantID,
		PeriodStart:  time.Now().AddDate(0, 0, -7),
		PeriodEnd:    time.Now(),
		TotalAmount:  1000.00,
		Fee:          100.00,
		ActualAmount: 900.00,
		OrderCount:   10,
		Status:       models.SettlementStatusPending,
	}
	err := db.Create(settlement).Error
	require.NoError(t, err)
	return settlement
}

// createFinanceTestWithdrawal 创建测试提现记录
func createFinanceTestWithdrawal(t *testing.T, db *gorm.DB, userID int64) *models.Withdrawal {
	withdrawal := &models.Withdrawal{
		WithdrawalNo: fmt.Sprintf("WD%d", time.Now().UnixNano()),
		UserID:       userID,
		Type:         models.WithdrawalTypeWallet,
		Amount:       100.00,
		Fee:          1.00,
		ActualAmount: 99.00,
		Status:       models.WithdrawalStatusPending,
		WithdrawTo:   models.WithdrawToWechat,
	}
	err := db.Create(withdrawal).Error
	require.NoError(t, err)
	return withdrawal
}

// TestFinanceAPI_GetOverview 测试获取财务概览
func TestFinanceAPI_GetOverview(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	// 准备测试数据
	user := createFinanceTestUser(t, db)
	merchant := createFinanceTestMerchant(t, db)
	createFinanceTestOrder(t, db, user.ID, merchant.ID, 100.00, models.OrderTypeRental)

	req, _ := http.NewRequest("GET", "/api/admin/finance/overview", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
}

// TestFinanceAPI_GetRevenueStatistics 测试获取收入统计
func TestFinanceAPI_GetRevenueStatistics(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/admin/finance/revenue/statistics?start_date=%s&end_date=%s", startDate, endDate), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_GetRevenueStatistics_MissingParams 测试缺少参数
func TestFinanceAPI_GetRevenueStatistics_MissingParams(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/revenue/statistics", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFinanceAPI_ListSettlements 测试获取结算列表
func TestFinanceAPI_ListSettlements(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	merchant := createFinanceTestMerchant(t, db)
	createFinanceTestSettlement(t, db, merchant.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/settlements?page=1&page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
}

// TestFinanceAPI_GetSettlement 测试获取结算详情
func TestFinanceAPI_GetSettlement(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	merchant := createFinanceTestMerchant(t, db)
	settlement := createFinanceTestSettlement(t, db, merchant.ID)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/admin/finance/settlements/%d", settlement.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_CreateSettlement 测试创建结算
func TestFinanceAPI_CreateSettlement(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	merchant := createFinanceTestMerchant(t, db)

	reqBody := map[string]interface{}{
		"type":         "merchant",
		"target_id":    merchant.ID,
		"period_start": time.Now().AddDate(0, 0, -7).Format("2006-01-02"),
		"period_end":   time.Now().Format("2006-01-02"),
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/admin/finance/settlements", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_ProcessSettlement 测试处理结算
func TestFinanceAPI_ProcessSettlement(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	merchant := createFinanceTestMerchant(t, db)
	settlement := createFinanceTestSettlement(t, db, merchant.ID)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/admin/finance/settlements/%d/process", settlement.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_ListWithdrawals 测试获取提现列表
func TestFinanceAPI_ListWithdrawals(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	user := createFinanceTestUser(t, db)
	createFinanceTestWithdrawal(t, db, user.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/withdrawals?page=1&page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_GetWithdrawal 测试获取提现详情
func TestFinanceAPI_GetWithdrawal(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	user := createFinanceTestUser(t, db)
	withdrawal := createFinanceTestWithdrawal(t, db, user.ID)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/admin/finance/withdrawals/%d", withdrawal.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_HandleWithdrawal_Approve 测试审核通过提现
func TestFinanceAPI_HandleWithdrawal_Approve(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	user := createFinanceTestUser(t, db)
	withdrawal := createFinanceTestWithdrawal(t, db, user.ID)

	reqBody := map[string]interface{}{
		"action": "approve",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_HandleWithdrawal_Reject 测试拒绝提现
func TestFinanceAPI_HandleWithdrawal_Reject(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	user := createFinanceTestUser(t, db)
	withdrawal := createFinanceTestWithdrawal(t, db, user.ID)

	reqBody := map[string]interface{}{
		"action": "reject",
		"reason": "金额超出限制",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_HandleWithdrawal_RejectWithoutReason 测试拒绝但无原因
func TestFinanceAPI_HandleWithdrawal_RejectWithoutReason(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	user := createFinanceTestUser(t, db)
	withdrawal := createFinanceTestWithdrawal(t, db, user.ID)

	reqBody := map[string]interface{}{
		"action": "reject",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/admin/finance/withdrawals/%d/handle", withdrawal.ID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFinanceAPI_BatchHandleWithdrawals 测试批量处理提现
func TestFinanceAPI_BatchHandleWithdrawals(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	user := createFinanceTestUser(t, db)
	var ids []int64
	for i := 0; i < 3; i++ {
		w := createFinanceTestWithdrawal(t, db, user.ID)
		ids = append(ids, w.ID)
	}

	reqBody := map[string]interface{}{
		"ids":    ids,
		"action": "approve",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/admin/finance/withdrawals/batch", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_GetWithdrawalSummary 测试获取提现汇总
func TestFinanceAPI_GetWithdrawalSummary(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/withdrawals/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_GetSettlementSummary 测试获取结算汇总
func TestFinanceAPI_GetSettlementSummary(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/settlements/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_ExportSettlements 测试导出结算记录
func TestFinanceAPI_ExportSettlements(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	merchant := createFinanceTestMerchant(t, db)
	createFinanceTestSettlement(t, db, merchant.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/export/settlements", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

// TestFinanceAPI_ExportWithdrawals 测试导出提现记录
func TestFinanceAPI_ExportWithdrawals(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	user := createFinanceTestUser(t, db)
	createFinanceTestWithdrawal(t, db, user.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/export/withdrawals", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

// TestFinanceAPI_ExportDailyRevenue 测试导出每日收入报表
func TestFinanceAPI_ExportDailyRevenue(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/admin/finance/export/daily-revenue?start_date=%s&end_date=%s", startDate, endDate), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

// TestFinanceAPI_Unauthorized 测试未授权访问
func TestFinanceAPI_Unauthorized(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	req, _ := http.NewRequest("GET", "/api/admin/finance/overview", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestFinanceAPI_GetDailyRevenueReport 测试获取每日收入报表
func TestFinanceAPI_GetDailyRevenueReport(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/admin/finance/revenue/daily?start_date=%s&end_date=%s", startDate, endDate), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_GetOrderRevenueByType 测试按类型获取订单收入
func TestFinanceAPI_GetOrderRevenueByType(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/revenue/by-type", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_GetTransactionStatistics 测试获取交易统计
func TestFinanceAPI_GetTransactionStatistics(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/transactions/statistics", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFinanceAPI_GetMerchantSettlementReport 测试获取商户结算报表
func TestFinanceAPI_GetMerchantSettlementReport(t *testing.T) {
	db := setupFinanceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-finance-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: time.Hour * 24,
		Issuer:            "test",
	})
	router := setupFinanceAPITestRouter(db, jwtManager)

	admin := createFinanceTestAdmin(t, db)
	token := generateAdminTestToken(jwtManager, admin.ID)

	req, _ := http.NewRequest("GET", "/api/admin/finance/reports/merchant-settlement", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
