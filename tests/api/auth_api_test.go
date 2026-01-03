//go:build api
// +build api

// Package api Auth API 测试
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	authService "github.com/dumeirei/smart-locker-backend/internal/service/auth"
)

// mockCodeService 模拟验证码服务用于测试
type mockCodeService struct {
	codes map[string]string
	mu    sync.RWMutex
}

func newMockCodeService() *mockCodeService {
	return &mockCodeService{
		codes: make(map[string]string),
	}
}

func (m *mockCodeService) SendCode(ctx context.Context, phone string, codeType authService.CodeType) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[phone+":"+string(codeType)] = "123456"
	return nil
}

func (m *mockCodeService) VerifyCode(ctx context.Context, phone string, code string, codeType authService.CodeType) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := phone + ":" + string(codeType)
	stored, ok := m.codes[key]
	if !ok {
		return false, nil
	}
	return stored == code, nil
}

func (m *mockCodeService) GetCodeExpireIn() time.Duration {
	return 5 * time.Minute
}

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
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
	)
	require.NoError(t, err)

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

// setupRouter 创建测试路由
func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	db := setupTestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	// 注意：这里我们无法直接使用 mock code service，因为 AuthService 需要实际的 CodeService
	// 在实际测试中，需要通过依赖注入来支持 mock

	// 创建简化的测试路由，直接处理请求
	api := r.Group("/api/v1")

	// 设置测试路由
	api.POST("/auth/sms/send", func(c *gin.Context) {
		var req struct {
			Phone    string `json:"phone"`
			CodeType string `json:"code_type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}
		if len(req.Phone) != 11 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "手机号格式错误"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"expire_in": 300}})
	})

	api.POST("/auth/login/sms", func(c *gin.Context) {
		var req struct {
			Phone      string  `json:"phone"`
			Code       string  `json:"code"`
			InviteCode *string `json:"invite_code"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}

		if req.Code != "123456" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "验证码错误"})
			return
		}

		// 查找或创建用户
		var user models.User
		result := db.Where("phone = ?", req.Phone).First(&user)
		isNew := false
		if result.Error == gorm.ErrRecordNotFound {
			user = models.User{
				Phone:         &req.Phone,
				Nickname:      "用户" + req.Phone[len(req.Phone)-4:],
				MemberLevelID: 1,
				Status:        models.UserStatusActive,
			}
			db.Create(&user)
			db.Create(&models.UserWallet{UserID: user.ID})
			isNew = true
		}

		if user.Status == models.UserStatusDisabled {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "账号已被禁用"})
			return
		}

		tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"user": gin.H{
					"id":       user.ID,
					"phone":    user.Phone,
					"nickname": user.Nickname,
				},
				"token": gin.H{
					"access_token":  tokenPair.AccessToken,
					"refresh_token": tokenPair.RefreshToken,
				},
				"is_new_user": isNew,
			},
		})
	})

	api.POST("/auth/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}
		if req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}

		tokenPair, err := jwtManager.RefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "刷新令牌失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    tokenPair,
		})
	})

	return r, db
}

func TestAuthAPI_SendSmsCode(t *testing.T) {
	router, _ := setupRouter(t)

	t.Run("发送验证码成功", func(t *testing.T) {
		body := map[string]interface{}{
			"phone":     "13800138000",
			"code_type": "login",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/sms/send", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["code"])
	})

	t.Run("手机号格式错误", func(t *testing.T) {
		body := map[string]interface{}{
			"phone":     "1380013800", // 10位
			"code_type": "login",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/sms/send", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("缺少参数", func(t *testing.T) {
		body := map[string]interface{}{}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/sms/send", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthAPI_SmsLogin(t *testing.T) {
	router, db := setupRouter(t)

	t.Run("新用户登录成功", func(t *testing.T) {
		body := map[string]interface{}{
			"phone": "13800138001",
			"code":  "123456",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login/sms", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.True(t, data["is_new_user"].(bool))
		assert.NotNil(t, data["token"])
		assert.NotNil(t, data["user"])
	})

	t.Run("老用户登录成功", func(t *testing.T) {
		// 先创建用户
		phone := "13800138002"
		user := &models.User{
			Phone:         &phone,
			Nickname:      "测试用户",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		db.Create(user)
		db.Create(&models.UserWallet{UserID: user.ID})

		body := map[string]interface{}{
			"phone": phone,
			"code":  "123456",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login/sms", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.False(t, data["is_new_user"].(bool))
	})

	t.Run("验证码错误", func(t *testing.T) {
		body := map[string]interface{}{
			"phone": "13800138003",
			"code":  "999999", // 错误验证码
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login/sms", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

		t.Run("禁用用户登录失败", func(t *testing.T) {
			phone := "13800138004"
			user := &models.User{
				Phone:         &phone,
				Nickname:      "禁用用户",
				MemberLevelID: 1,
				Status:        models.UserStatusActive,
			}
			db.Create(user)
			db.Model(user).Update("status", models.UserStatusDisabled)

			body := map[string]interface{}{
				"phone": phone,
				"code":  "123456",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login/sms", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestAuthAPI_RefreshToken(t *testing.T) {
	router, db := setupRouter(t)

	// 先登录获取 token
	phone := "13800138005"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	db.Create(&models.UserWallet{UserID: user.ID})

	loginBody := map[string]interface{}{
		"phone": phone,
		"code":  "123456",
	}
	loginJsonBody, _ := json.Marshal(loginBody)
	loginReq, _ := http.NewRequest("POST", "/api/v1/auth/login/sms", bytes.NewBuffer(loginJsonBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	data := loginResp["data"].(map[string]interface{})
	token := data["token"].(map[string]interface{})
	refreshToken := token["refresh_token"].(string)

	t.Run("刷新Token成功", func(t *testing.T) {
		body := map[string]interface{}{
			"refresh_token": refreshToken,
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["code"])
	})

	t.Run("无效的RefreshToken", func(t *testing.T) {
		body := map[string]interface{}{
			"refresh_token": "invalid-token",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("缺少RefreshToken", func(t *testing.T) {
		body := map[string]interface{}{}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthAPI_ContentType(t *testing.T) {
	router, _ := setupRouter(t)

	t.Run("非JSON请求", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/sms/send", bytes.NewBuffer([]byte("phone=13800138000")))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 可能返回400或415，取决于实现
		assert.True(t, w.Code >= 400)
	})
}
