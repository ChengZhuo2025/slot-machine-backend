// Package api 管理员认证 API 测试
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/common/crypto"
	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// setupAdminAPITestDB 创建测试数据库
func setupAdminAPITestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Admin{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
	)
	require.NoError(t, err)

	return db
}

// setupAdminAPIRouter 创建测试路由
func setupAdminAPIRouter(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	db := setupAdminAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-admin-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	adminRepo := repository.NewAdminRepository(db)
	authService := adminService.NewAdminAuthService(adminRepo, jwtManager)

	api := r.Group("/api/v1/admin")

	// 登录接口
	api.POST("/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}

		loginReq := &adminService.LoginRequest{
			Username: req.Username,
			Password: req.Password,
			IP:       c.ClientIP(),
		}

		resp, err := authService.Login(c, loginReq)
		if err != nil {
			switch err {
			case adminService.ErrAdminNotFound:
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
			case adminService.ErrInvalidPassword:
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
			case adminService.ErrAdminDisabled:
				c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "账号已禁用"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务器错误"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    resp,
		})
	})

	// 获取当前管理员信息接口
	api.GET("/auth/info", func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权"})
			return
		}

		// 去掉 Bearer 前缀
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "令牌无效"})
			return
		}

		resp, err := authService.GetAdminWithPermissions(c, claims.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取信息失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    resp,
		})
	})

	// 刷新令牌接口
	api.POST("/auth/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}

		tokenPair, err := authService.RefreshToken(c, req.RefreshToken)
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

	// 修改密码接口
	api.PUT("/auth/password", func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权"})
			return
		}

		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "令牌无效"})
			return
		}

		var req struct {
			OldPassword string `json:"old_password" binding:"required"`
			NewPassword string `json:"new_password" binding:"required,min=6"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}

		changeReq := &adminService.ChangePasswordRequest{
			OldPassword: req.OldPassword,
			NewPassword: req.NewPassword,
		}

		err = authService.ChangePassword(c, claims.UserID, changeReq)
		if err != nil {
			if err == adminService.ErrOldPasswordInvalid {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "原密码错误"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "修改密码失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
		})
	})

	return r, db, jwtManager
}

// createAPITestAdmin 创建测试管理员
func createAPITestAdmin(t *testing.T, db *gorm.DB, username, password string) *models.Admin {
	role := &models.Role{
		Code:     "test_api_role",
		Name:     "测试角色",
		IsSystem: false,
	}
	// 检查角色是否存在
	var existingRole models.Role
	if err := db.Where("code = ?", role.Code).First(&existingRole).Error; err == nil {
		role = &existingRole
	} else {
		err := db.Create(role).Error
		require.NoError(t, err)
	}

	permission := &models.Permission{
		Code: "device:read",
		Name: "设备查看",
		Type: models.PermissionTypeAPI,
	}
	var existingPerm models.Permission
	if err := db.Where("code = ?", permission.Code).First(&existingPerm).Error; err == nil {
		permission = &existingPerm
	} else {
		err := db.Create(permission).Error
		require.NoError(t, err)
	}

	var existingRP models.RolePermission
	if err := db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).First(&existingRP).Error; err != nil {
		rolePermission := &models.RolePermission{
			RoleID:       role.ID,
			PermissionID: permission.ID,
		}
		err := db.Create(rolePermission).Error
		require.NoError(t, err)
	}

	passwordHash, err := crypto.HashPassword(password)
	require.NoError(t, err)

	admin := &models.Admin{
		Username:     username,
		PasswordHash: passwordHash,
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	err = db.Create(admin).Error
	require.NoError(t, err)

	return admin
}

func TestAdminAuthAPI_Login_Success(t *testing.T) {
	router, db, _ := setupAdminAPIRouter(t)

	// 创建测试管理员
	createAPITestAdmin(t, db, "apitest_admin", "password123")

	body := map[string]interface{}{
		"username": "apitest_admin",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.NotNil(t, data["admin"])
	assert.NotNil(t, data["token"])
	assert.NotNil(t, data["permissions"])

	token := data["token"].(map[string]interface{})
	assert.NotEmpty(t, token["access_token"])
	assert.NotEmpty(t, token["refresh_token"])
}

func TestAdminAuthAPI_Login_WrongPassword(t *testing.T) {
	router, db, _ := setupAdminAPIRouter(t)

	createAPITestAdmin(t, db, "wrongpw_admin", "correctpassword")

	body := map[string]interface{}{
		"username": "wrongpw_admin",
		"password": "wrongpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuthAPI_Login_UserNotFound(t *testing.T) {
	router, _, _ := setupAdminAPIRouter(t)

	body := map[string]interface{}{
		"username": "nonexistent_admin",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuthAPI_Login_DisabledAdmin(t *testing.T) {
	router, db, _ := setupAdminAPIRouter(t)

	admin := createAPITestAdmin(t, db, "disabled_api_admin", "password123")
	db.Model(admin).Update("status", models.AdminStatusDisabled)

	body := map[string]interface{}{
		"username": "disabled_api_admin",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminAuthAPI_Login_MissingParams(t *testing.T) {
	router, _, _ := setupAdminAPIRouter(t)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "缺少用户名",
			body: map[string]interface{}{
				"password": "password123",
			},
		},
		{
			name: "缺少密码",
			body: map[string]interface{}{
				"username": "testadmin",
			},
		},
		{
			name: "空请求体",
			body: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestAdminAuthAPI_GetInfo_Success(t *testing.T) {
	router, db, _ := setupAdminAPIRouter(t)

	createAPITestAdmin(t, db, "info_api_admin", "password123")

	// 先登录获取 token
	loginBody := map[string]interface{}{
		"username": "info_api_admin",
		"password": "password123",
	}
	loginJsonBody, _ := json.Marshal(loginBody)
	loginReq, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(loginJsonBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	data := loginResp["data"].(map[string]interface{})
	token := data["token"].(map[string]interface{})
	accessToken := token["access_token"].(string)

	// 获取管理员信息
	req, _ := http.NewRequest("GET", "/api/v1/admin/auth/info", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	respData := resp["data"].(map[string]interface{})
	assert.NotNil(t, respData["admin"])
	assert.NotNil(t, respData["permissions"])
}

func TestAdminAuthAPI_GetInfo_NoToken(t *testing.T) {
	router, _, _ := setupAdminAPIRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/admin/auth/info", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuthAPI_GetInfo_InvalidToken(t *testing.T) {
	router, _, _ := setupAdminAPIRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/admin/auth/info", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuthAPI_RefreshToken_Success(t *testing.T) {
	router, db, _ := setupAdminAPIRouter(t)

	createAPITestAdmin(t, db, "refresh_api_admin", "password123")

	// 先登录获取 token
	loginBody := map[string]interface{}{
		"username": "refresh_api_admin",
		"password": "password123",
	}
	loginJsonBody, _ := json.Marshal(loginBody)
	loginReq, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(loginJsonBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	data := loginResp["data"].(map[string]interface{})
	token := data["token"].(map[string]interface{})
	refreshToken := token["refresh_token"].(string)

	// 刷新 token
	body := map[string]interface{}{
		"refresh_token": refreshToken,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	newToken := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, newToken["access_token"])
	assert.NotEmpty(t, newToken["refresh_token"])
}

func TestAdminAuthAPI_RefreshToken_InvalidToken(t *testing.T) {
	router, _, _ := setupAdminAPIRouter(t)

	body := map[string]interface{}{
		"refresh_token": "invalid-refresh-token",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuthAPI_ChangePassword_Success(t *testing.T) {
	router, db, _ := setupAdminAPIRouter(t)

	createAPITestAdmin(t, db, "changepw_api_admin", "oldpassword")

	// 先登录获取 token
	loginBody := map[string]interface{}{
		"username": "changepw_api_admin",
		"password": "oldpassword",
	}
	loginJsonBody, _ := json.Marshal(loginBody)
	loginReq, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(loginJsonBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	data := loginResp["data"].(map[string]interface{})
	token := data["token"].(map[string]interface{})
	accessToken := token["access_token"].(string)

	// 修改密码
	body := map[string]interface{}{
		"old_password": "oldpassword",
		"new_password": "newpassword123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/v1/admin/auth/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证新密码可以登录
	newLoginBody := map[string]interface{}{
		"username": "changepw_api_admin",
		"password": "newpassword123",
	}
	newLoginJsonBody, _ := json.Marshal(newLoginBody)
	newLoginReq, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(newLoginJsonBody))
	newLoginReq.Header.Set("Content-Type", "application/json")

	newLoginW := httptest.NewRecorder()
	router.ServeHTTP(newLoginW, newLoginReq)

	assert.Equal(t, http.StatusOK, newLoginW.Code)
}

func TestAdminAuthAPI_ChangePassword_WrongOldPassword(t *testing.T) {
	router, db, _ := setupAdminAPIRouter(t)

	createAPITestAdmin(t, db, "wrongold_api_admin", "correctpassword")

	// 先登录获取 token
	loginBody := map[string]interface{}{
		"username": "wrongold_api_admin",
		"password": "correctpassword",
	}
	loginJsonBody, _ := json.Marshal(loginBody)
	loginReq, _ := http.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBuffer(loginJsonBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	data := loginResp["data"].(map[string]interface{})
	token := data["token"].(map[string]interface{})
	accessToken := token["access_token"].(string)

	// 使用错误的原密码修改密码
	body := map[string]interface{}{
		"old_password": "wrongpassword",
		"new_password": "newpassword123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/v1/admin/auth/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminAuthAPI_ChangePassword_NoToken(t *testing.T) {
	router, _, _ := setupAdminAPIRouter(t)

	body := map[string]interface{}{
		"old_password": "oldpassword",
		"new_password": "newpassword123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/v1/admin/auth/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
