//go:build api
// +build api

// Package api 分销 API 测试
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
	distributionHandler "github.com/dumeirei/smart-locker-backend/internal/handler/distribution"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	distributionService "github.com/dumeirei/smart-locker-backend/internal/service/distribution"
)

// setupDistributionAPITestDB 创建 API 测试数据库
func setupDistributionAPITestDB(t *testing.T) *gorm.DB {
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

// setupDistributionAPITestRouter 创建测试路由
func setupDistributionAPITestRouter(db *gorm.DB, jwtManager *jwt.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 初始化仓储和服务
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	userRepo := repository.NewUserRepository(db)

	distributorSvc := distributionService.NewDistributorService(distributorRepo, userRepo, db)
	commissionSvc := distributionService.NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
	inviteSvc := distributionService.NewInviteService(distributorRepo, "https://test.example.com")
	withdrawSvc := distributionService.NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)

	handler := distributionHandler.NewHandler(distributorSvc, commissionSvc, inviteSvc, withdrawSvc)

	// 注册路由
	api := r.Group("/api/v1")
	api.Use(middleware.UserAuth(jwtManager))
	{
		distribution := api.Group("/distribution")
		{
			distribution.GET("/check", handler.CheckStatus)
			distribution.POST("/apply", handler.Apply)
			distribution.GET("/info", handler.GetInfo)
			distribution.GET("/dashboard", handler.GetDashboard)
			distribution.GET("/team/stats", handler.GetTeamStats)
			distribution.GET("/team/members", handler.GetTeamMembers)
			distribution.GET("/invite", handler.GetInviteInfo)
			distribution.GET("/invite/validate", handler.ValidateInviteCode)
			distribution.GET("/commissions", handler.GetCommissions)
			distribution.GET("/commissions/stats", handler.GetCommissionStats)
			distribution.POST("/withdraw", handler.ApplyWithdraw)
			distribution.GET("/withdrawals", handler.GetWithdrawals)
			distribution.GET("/withdraw/config", handler.GetWithdrawConfig)
			distribution.GET("/ranking", handler.GetRanking)
		}
	}

	return r
}

// createAPITestJWTManager 创建测试 JWT 管理器
func createAPITestJWTManager() *jwt.Manager {
	return jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-distribution-api",
		AccessExpireTime:  24 * time.Hour,
		RefreshExpireTime: 7 * 24 * time.Hour,
		Issuer:            "test",
	})
}

// createAPITestUser 创建 API 测试用户
func createAPITestUser(db *gorm.DB) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 1000.0,
	}
	db.Create(wallet)

	return user
}

// createAPITestDistributor 创建 API 测试分销商
func createAPITestDistributor(db *gorm.DB, userID int64, status int) *models.Distributor {
	distributor := &models.Distributor{
		UserID:              userID,
		Level:               models.DistributorLevelDirect,
		InviteCode:          fmt.Sprintf("API%d", time.Now().UnixNano()%1000000),
		TotalCommission:     100.0,
		AvailableCommission: 50.0,
		FrozenCommission:    0,
		WithdrawnCommission: 50.0,
		TeamCount:           5,
		DirectCount:         3,
		Status:              status,
	}
	db.Create(distributor)
	return distributor
}

// generateTestToken 生成测试 Token
func generateTestToken(jwtManager *jwt.Manager, userID int64) string {
	token, _ := jwtManager.GenerateToken(userID, jwt.UserTypeUser, "")
	return token.AccessToken
}

func TestDistributionAPI_CheckStatus(t *testing.T) {
	t.Run("用户非分销商_返回false", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/check", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])
		data := response["data"].(map[string]interface{})
		assert.False(t, data["is_distributor"].(bool))
	})

	t.Run("用户是分销商_返回true", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		createAPITestDistributor(db, user.ID, models.DistributorStatusApproved)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/check", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		data := response["data"].(map[string]interface{})
		assert.True(t, data["is_distributor"].(bool))
	})
}

func TestDistributionAPI_Apply(t *testing.T) {
	t.Run("正常申请成为分销商", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		token := generateTestToken(jwtManager, user.ID)

		reqBody := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/distribution/apply", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])
	})

	t.Run("使用邀请码申请", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		// 创建上级分销商
		parentUser := createAPITestUser(db)
		parentDistributor := createAPITestDistributor(db, parentUser.ID, models.DistributorStatusApproved)

		// 新用户申请
		user := createAPITestUser(db)
		token := generateTestToken(jwtManager, user.ID)

		reqBody := fmt.Sprintf(`{"invite_code":"%s"}`, parentDistributor.InviteCode)
		req, _ := http.NewRequest("POST", "/api/v1/distribution/apply", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("重复申请返回错误", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		createAPITestDistributor(db, user.ID, models.DistributorStatusPending)
		token := generateTestToken(jwtManager, user.ID)

		reqBody := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/distribution/apply", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 应该返回错误
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotEqual(t, float64(0), response["code"])
	})
}

func TestDistributionAPI_GetInfo(t *testing.T) {
	t.Run("获取分销商信息", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		createAPITestDistributor(db, user.ID, models.DistributorStatusApproved)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/info", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])
		assert.NotNil(t, response["data"])
	})

	t.Run("非分销商获取信息返回错误", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/info", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotEqual(t, float64(0), response["code"])
	})
}

func TestDistributionAPI_GetDashboard(t *testing.T) {
	t.Run("获取仪表盘数据", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		createAPITestDistributor(db, user.ID, models.DistributorStatusApproved)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "total_commission")
		assert.Contains(t, data, "available_commission")
		assert.Contains(t, data, "team_count")
		assert.Contains(t, data, "invite_code")
	})
}

func TestDistributionAPI_GetTeamStats(t *testing.T) {
	t.Run("获取团队统计", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		createAPITestDistributor(db, user.ID, models.DistributorStatusApproved)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/team/stats", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "team_count")
		assert.Contains(t, data, "direct_count")
	})
}

func TestDistributionAPI_ValidateInviteCode(t *testing.T) {
	t.Run("验证有效邀请码", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		// 创建分销商
		parentUser := createAPITestUser(db)
		parentDistributor := createAPITestDistributor(db, parentUser.ID, models.DistributorStatusApproved)

		user := createAPITestUser(db)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/invite/validate?code="+parentDistributor.InviteCode, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.True(t, data["valid"].(bool))
	})

	t.Run("验证无效邀请码", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/invite/validate?code=INVALID", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		assert.False(t, data["valid"].(bool))
	})
}

func TestDistributionAPI_ApplyWithdraw(t *testing.T) {
	t.Run("正常申请提现", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		createAPITestDistributor(db, user.ID, models.DistributorStatusApproved)
		token := generateTestToken(jwtManager, user.ID)

		reqBody := `{
			"type": "commission",
			"amount": 20.0,
			"withdraw_to": "wechat",
			"account_info": "{\"openid\":\"test\"}"
		}`
		req, _ := http.NewRequest("POST", "/api/v1/distribution/withdraw", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])
	})

	t.Run("金额不足返回错误", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		createAPITestDistributor(db, user.ID, models.DistributorStatusApproved)
		token := generateTestToken(jwtManager, user.ID)

		reqBody := `{
			"type": "commission",
			"amount": 100.0,
			"withdraw_to": "wechat",
			"account_info": "{\"openid\":\"test\"}"
		}`
		req, _ := http.NewRequest("POST", "/api/v1/distribution/withdraw", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotEqual(t, float64(0), response["code"])
	})
}

func TestDistributionAPI_GetWithdrawals(t *testing.T) {
	t.Run("获取提现记录", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		createAPITestDistributor(db, user.ID, models.DistributorStatusApproved)
		token := generateTestToken(jwtManager, user.ID)

		// 创建提现记录
		withdrawal := &models.Withdrawal{
			WithdrawalNo:         fmt.Sprintf("W%d", time.Now().UnixNano()),
			UserID:               user.ID,
			Type:                 models.WithdrawalTypeCommission,
			Amount:               20.0,
			Fee:                  0.12,
			ActualAmount:         19.88,
			WithdrawTo:           models.WithdrawToWechat,
			AccountInfoEncrypted: `{"openid":"test"}`,
			Status:               models.WithdrawalStatusPending,
		}
		db.Create(withdrawal)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/withdrawals", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])
	})
}

func TestDistributionAPI_GetWithdrawConfig(t *testing.T) {
	t.Run("获取提现配置", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		token := generateTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/withdraw/config", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "min_withdraw")
		assert.Contains(t, data, "withdraw_fee")
	})
}

func TestDistributionAPI_GetCommissions(t *testing.T) {
	t.Run("获取佣金记录", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		user := createAPITestUser(db)
		distributor := createAPITestDistributor(db, user.ID, models.DistributorStatusApproved)
		token := generateTestToken(jwtManager, user.ID)

		// 创建佣金记录
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    999,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          0.10,
			Amount:        10.0,
			Status:        models.CommissionStatusPending,
		}
		db.Create(commission)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/commissions", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])
	})
}

func TestDistributionAPI_Unauthorized(t *testing.T) {
	t.Run("无Token访问返回401", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("无效Token访问返回401", func(t *testing.T) {
		db := setupDistributionAPITestDB(t)
		jwtManager := createAPITestJWTManager()
		router := setupDistributionAPITestRouter(db, jwtManager)

		req, _ := http.NewRequest("GET", "/api/v1/distribution/check", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
