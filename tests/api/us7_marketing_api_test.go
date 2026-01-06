//go:build api
// +build api

// Package api 营销 API 测试
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
	marketingHandler "github.com/dumeirei/smart-locker-backend/internal/handler/marketing"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	marketingService "github.com/dumeirei/smart-locker-backend/internal/service/marketing"
)

// setupMarketingAPITestDB 创建 API 测试数据库
func setupMarketingAPITestDB(t *testing.T) *gorm.DB {
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
		&models.Coupon{},
		&models.UserCoupon{},
		&models.Campaign{},
		&models.Order{},
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

// setupMarketingAPITestRouter 创建测试路由
func setupMarketingAPITestRouter(db *gorm.DB, jwtManager *jwt.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 初始化仓储和服务
	couponRepo := repository.NewCouponRepository(db)
	userCouponRepo := repository.NewUserCouponRepository(db)

	couponSvc := marketingService.NewCouponService(db, couponRepo, userCouponRepo)
	userCouponSvc := marketingService.NewUserCouponService(db, couponRepo, userCouponRepo)

	handler := marketingHandler.NewCouponHandler(couponSvc, userCouponSvc)

	// 注册路由
	api := r.Group("/api/v1")
	api.Use(middleware.UserAuth(jwtManager))
	{
		marketing := api.Group("/marketing")
		{
			// 可领取的优惠券
			marketing.GET("/coupons", handler.GetCouponList)
			marketing.GET("/coupons/:id", handler.GetCouponDetail)
			marketing.POST("/coupons/:id/receive", handler.ReceiveCoupon)

			// 用户优惠券
			marketing.GET("/user-coupons", handler.GetUserCoupons)
			marketing.GET("/user-coupons/available", handler.GetAvailableCoupons)
			marketing.GET("/user-coupons/for-order", handler.GetAvailableCouponsForOrder)
			marketing.GET("/user-coupons/count", handler.GetCouponCountByStatus)
			marketing.GET("/user-coupons/:id", handler.GetUserCouponDetail)
		}
	}

	return r
}

// createMarketingAPITestJWTManager 创建测试 JWT 管理器
func createMarketingAPITestJWTManager() *jwt.Manager {
	return jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-marketing-api",
		AccessExpireTime:  24 * time.Hour,
		RefreshExpireTime: 7 * 24 * time.Hour,
		Issuer:            "test",
	})
}

// createMarketingAPITestUser 创建 API 测试用户
func createMarketingAPITestUser(db *gorm.DB) *models.User {
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

// createMarketingAPITestCoupon 创建 API 测试优惠券
func createMarketingAPITestCoupon(db *gorm.DB, opts ...func(*models.Coupon)) *models.Coupon {
	coupon := &models.Coupon{
		Name:            "测试优惠券",
		Type:            models.CouponTypeFixed,
		Value:           10.0,
		MinAmount:       50.0,
		TotalCount:      100,
		ReceivedCount:   0,
		UsedCount:       0,
		PerUserLimit:    3,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       time.Now().Add(-time.Hour),
		EndTime:         time.Now().Add(24 * time.Hour),
		Status:          models.CouponStatusActive,
	}

	for _, opt := range opts {
		opt(coupon)
	}

	db.Create(coupon)
	return coupon
}

// generateMarketingTestToken 生成测试 Token
func generateMarketingTestToken(jwtManager *jwt.Manager, userID int64) string {
	token, _, _ := jwtManager.GenerateAccessToken(userID, jwt.UserTypeUser, "")
	return token
}

func TestMarketingAPI_GetCouponList(t *testing.T) {
	t.Run("正常获取优惠券列表", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		// 创建优惠券
		createMarketingAPITestCoupon(db, func(c *models.Coupon) { c.Name = "优惠券1" })
		createMarketingAPITestCoupon(db, func(c *models.Coupon) { c.Name = "优惠券2" })

		req, _ := http.NewRequest("GET", "/api/v1/marketing/coupons", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.Len(t, list, 2)
		assert.Equal(t, float64(2), data["total"])
	})

	t.Run("分页获取优惠券列表", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		// 创建5个优惠券
		for i := 0; i < 5; i++ {
			createMarketingAPITestCoupon(db, func(c *models.Coupon) {
				c.Name = fmt.Sprintf("优惠券%d", i+1)
			})
		}

		req, _ := http.NewRequest("GET", "/api/v1/marketing/coupons?page=1&page_size=2", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.Len(t, list, 2)
		assert.Equal(t, float64(5), data["total"])
	})
}

func TestMarketingAPI_GetCouponDetail(t *testing.T) {
	t.Run("正常获取优惠券详情", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		coupon := createMarketingAPITestCoupon(db, func(c *models.Coupon) {
			c.Name = "测试详情优惠券"
		})

		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/marketing/coupons/%d", coupon.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "测试详情优惠券", data["name"])
		assert.Equal(t, float64(coupon.ID), data["id"])
		assert.True(t, data["can_receive"].(bool))
	})

	t.Run("获取不存在的优惠券", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/marketing/coupons/99999", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMarketingAPI_ReceiveCoupon(t *testing.T) {
	t.Run("正常领取优惠券", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		coupon := createMarketingAPITestCoupon(db)

		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/marketing/coupons/%d/receive", coupon.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])
		assert.Equal(t, "领取成功", response["message"])

		data := response["data"].(map[string]interface{})
		assert.NotNil(t, data["user_coupon_id"])

		// 验证数据库
		var updatedCoupon models.Coupon
		db.First(&updatedCoupon, coupon.ID)
		assert.Equal(t, 1, updatedCoupon.ReceivedCount)
	})

	t.Run("超过领取限制返回错误", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		coupon := createMarketingAPITestCoupon(db, func(c *models.Coupon) {
			c.PerUserLimit = 1
		})

		// 第一次领取
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/marketing/coupons/%d/receive", coupon.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 第二次领取应失败
		req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/marketing/coupons/%d/receive", coupon.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// API 返回 HTTP 200，错误码在 JSON body 中
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotEqual(t, float64(0), response["code"])
	})

	t.Run("领取已领完优惠券返回错误", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		coupon := createMarketingAPITestCoupon(db, func(c *models.Coupon) {
			c.TotalCount = 1
			c.ReceivedCount = 1 // 已领完
		})

		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/marketing/coupons/%d/receive", coupon.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// API 返回 HTTP 200，错误码在 JSON body 中
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotEqual(t, float64(0), response["code"])
	})
}

func TestMarketingAPI_GetUserCoupons(t *testing.T) {
	t.Run("正常获取用户优惠券列表", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		// 创建用户优惠券
		coupon := createMarketingAPITestCoupon(db)
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})

		req, _ := http.NewRequest("GET", "/api/v1/marketing/user-coupons", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.Len(t, list, 1)
	})

	t.Run("按状态筛选用户优惠券", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		coupon := createMarketingAPITestCoupon(db)

		// 创建未使用优惠券
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})

		// 创建已使用优惠券
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUsed,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})

		// 筛选未使用
		req, _ := http.NewRequest("GET", "/api/v1/marketing/user-coupons?status=0", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["total"])
	})
}

func TestMarketingAPI_GetAvailableCoupons(t *testing.T) {
	t.Run("获取可用优惠券列表", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		coupon := createMarketingAPITestCoupon(db)

		// 创建可用优惠券
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})

		// 创建已过期优惠券（不应出现在可用列表）
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(-time.Hour),
			ReceivedAt: time.Now(),
		})

		req, _ := http.NewRequest("GET", "/api/v1/marketing/user-coupons/available", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.Len(t, list, 1)
	})
}

func TestMarketingAPI_GetAvailableCouponsForOrder(t *testing.T) {
	t.Run("获取订单可用优惠券", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		// 创建满足条件的优惠券
		coupon := createMarketingAPITestCoupon(db, func(c *models.Coupon) {
			c.MinAmount = 50.0
		})
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})

		// 创建不满足金额的优惠券
		coupon2 := createMarketingAPITestCoupon(db, func(c *models.Coupon) {
			c.MinAmount = 200.0
		})
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon2.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})

		req, _ := http.NewRequest("GET", "/api/v1/marketing/user-coupons/for-order?order_type=all&order_amount=100", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].([]interface{})
		assert.Len(t, data, 1)
	})

	t.Run("缺少必要参数返回错误", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		req, _ := http.NewRequest("GET", "/api/v1/marketing/user-coupons/for-order", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestMarketingAPI_GetCouponCountByStatus(t *testing.T) {
	t.Run("获取各状态优惠券数量", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		coupon := createMarketingAPITestCoupon(db)

		// 创建不同状态的优惠券
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUsed,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusExpired,
			ExpiredAt:  time.Now().Add(-time.Hour),
			ReceivedAt: time.Now(),
		})

		req, _ := http.NewRequest("GET", "/api/v1/marketing/user-coupons/count", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["unused"])
		assert.Equal(t, float64(1), data["used"])
		assert.Equal(t, float64(1), data["expired"])
	})
}

func TestMarketingAPI_GetUserCouponDetail(t *testing.T) {
	t.Run("正常获取用户优惠券详情", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user.ID)

		coupon := createMarketingAPITestCoupon(db, func(c *models.Coupon) {
			c.Name = "测试优惠券详情"
		})

		userCoupon := &models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		}
		db.Create(userCoupon)

		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/marketing/user-coupons/%d", userCoupon.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(userCoupon.ID), data["id"])
		assert.Equal(t, "测试优惠券详情", data["coupon_name"])
	})

	t.Run("获取其他用户优惠券返回404", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		user1 := createMarketingAPITestUser(db)
		user2 := createMarketingAPITestUser(db)
		token := generateMarketingTestToken(jwtManager, user1.ID)

		coupon := createMarketingAPITestCoupon(db)

		// 创建属于user2的优惠券
		userCoupon := &models.UserCoupon{
			UserID:     user2.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		}
		db.Create(userCoupon)

		// user1尝试获取
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/marketing/user-coupons/%d", userCoupon.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestMarketingAPI_Unauthorized(t *testing.T) {
	t.Run("无Token访问返回401", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		req, _ := http.NewRequest("GET", "/api/v1/marketing/coupons", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("无效Token访问返回401", func(t *testing.T) {
		db := setupMarketingAPITestDB(t)
		jwtManager := createMarketingAPITestJWTManager()
		router := setupMarketingAPITestRouter(db, jwtManager)

		req, _ := http.NewRequest("GET", "/api/v1/marketing/coupons", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// 使用 bytes 包（保持未使用的导入不报错）
var _ = bytes.Buffer{}
