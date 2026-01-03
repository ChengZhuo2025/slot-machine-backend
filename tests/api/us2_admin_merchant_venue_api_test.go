//go:build api
// +build api

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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
	adminHandler "github.com/dumeirei/smart-locker-backend/internal/handler/admin"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

func setupMerchantVenueAPITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.Admin{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
	))

	return db
}

func setupMerchantVenueAPIRouter(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	db := setupMerchantVenueAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-merchant-venue-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	merchantRepo := repository.NewMerchantRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)

	merchantSvc := adminService.NewMerchantAdminService(merchantRepo, nil)
	venueSvc := adminService.NewVenueAdminService(venueRepo, merchantRepo, deviceRepo)

	merchantH := adminHandler.NewMerchantHandler(merchantSvc)
	venueH := adminHandler.NewVenueHandler(venueSvc)

	api := r.Group("/api/v1/admin")

	// 模拟认证中间件
	api.Use(func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.Next()
			return
		}
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}
		claims, err := jwtManager.ParseToken(token)
		if err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("user_type", claims.UserType)
		}
		c.Next()
	})

	merchantH.RegisterRoutes(api)
	venueH.RegisterRoutes(api)

	return r, db, jwtManager
}

func createMerchantVenueAdminToken(t *testing.T, db *gorm.DB, jwtManager *jwt.Manager, username string) string {
	t.Helper()
	role := &models.Role{Code: "mv_api_role_" + username, Name: "测试角色", IsSystem: false}
	require.NoError(t, db.Create(role).Error)

	passwordHash, err := crypto.HashPassword("password123")
	require.NoError(t, err)
	admin := &models.Admin{Username: username, PasswordHash: passwordHash, Name: "测试管理员", RoleID: role.ID, Status: models.AdminStatusActive}
	require.NoError(t, db.Create(admin).Error)

	tokens, err := jwtManager.GenerateTokenPair(admin.ID, jwt.UserTypeAdmin, role.Code)
	require.NoError(t, err)
	return tokens.AccessToken
}

func TestUS2API_MerchantVenue_CRUD_AndDeleteConstraints(t *testing.T) {
	router, db, jwtManager := setupMerchantVenueAPIRouter(t)
	token := createMerchantVenueAdminToken(t, db, jwtManager, "mv_admin")

	// 1) 创建商户
	merchantBody, _ := json.Marshal(map[string]interface{}{
		"name":            "商户A",
		"contact_name":    "联系人",
		"contact_phone":   "13900139000",
		"commission_rate": 0.2,
		"settlement_type": "monthly",
	})
	createMerchantReq, _ := http.NewRequest("POST", "/api/v1/admin/merchants", bytes.NewBuffer(merchantBody))
	createMerchantReq.Header.Set("Content-Type", "application/json")
	createMerchantReq.Header.Set("Authorization", "Bearer "+token)
	createMerchantW := httptest.NewRecorder()
	router.ServeHTTP(createMerchantW, createMerchantReq)
	require.Equal(t, http.StatusOK, createMerchantW.Code)

	var createMerchantResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createMerchantW.Body.Bytes(), &createMerchantResp))
	require.Equal(t, float64(0), createMerchantResp["code"])
	merchantID := int64(createMerchantResp["data"].(map[string]interface{})["id"].(float64))
	require.NotZero(t, merchantID)

	// 2) 创建场地
	venueBody, _ := json.Marshal(map[string]interface{}{
		"merchant_id": merchantID,
		"name":        "场地A",
		"type":        "mall",
		"province":    "广东省",
		"city":        "深圳市",
		"district":    "南山区",
		"address":     "科技园路1号",
	})
	createVenueReq, _ := http.NewRequest("POST", "/api/v1/admin/venues", bytes.NewBuffer(venueBody))
	createVenueReq.Header.Set("Content-Type", "application/json")
	createVenueReq.Header.Set("Authorization", "Bearer "+token)
	createVenueW := httptest.NewRecorder()
	router.ServeHTTP(createVenueW, createVenueReq)
	require.Equal(t, http.StatusOK, createVenueW.Code)

	var createVenueResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createVenueW.Body.Bytes(), &createVenueResp))
	require.Equal(t, float64(0), createVenueResp["code"])
	venueID := int64(createVenueResp["data"].(map[string]interface{})["id"].(float64))
	require.NotZero(t, venueID)

	// 3) 尝试删除商户（应失败：商户下有场地）
	delMerchantReq, _ := http.NewRequest("DELETE", "/api/v1/admin/merchants/"+strconv.FormatInt(merchantID, 10), nil)
	delMerchantReq.Header.Set("Authorization", "Bearer "+token)
	delMerchantW := httptest.NewRecorder()
	router.ServeHTTP(delMerchantW, delMerchantReq)
	require.Equal(t, http.StatusBadRequest, delMerchantW.Code)

	// 4) 删除场地
	delVenueReq, _ := http.NewRequest("DELETE", "/api/v1/admin/venues/"+strconv.FormatInt(venueID, 10), nil)
	delVenueReq.Header.Set("Authorization", "Bearer "+token)
	delVenueW := httptest.NewRecorder()
	router.ServeHTTP(delVenueW, delVenueReq)
	require.Equal(t, http.StatusOK, delVenueW.Code)

	// 5) 再次删除商户（应成功）
	delMerchantReq2, _ := http.NewRequest("DELETE", "/api/v1/admin/merchants/"+strconv.FormatInt(merchantID, 10), nil)
	delMerchantReq2.Header.Set("Authorization", "Bearer "+token)
	delMerchantW2 := httptest.NewRecorder()
	router.ServeHTTP(delMerchantW2, delMerchantReq2)
	require.Equal(t, http.StatusOK, delMerchantW2.Code)

	// 6) 列表查询（返回 code=0，数据结构为分页）
	listMerchantsReq, _ := http.NewRequest("GET", "/api/v1/admin/merchants?page=1&page_size=10", nil)
	listMerchantsReq.Header.Set("Authorization", "Bearer "+token)
	listMerchantsW := httptest.NewRecorder()
	router.ServeHTTP(listMerchantsW, listMerchantsReq)
	require.Equal(t, http.StatusOK, listMerchantsW.Code)
	var listResp map[string]interface{}
	require.NoError(t, json.Unmarshal(listMerchantsW.Body.Bytes(), &listResp))
	assert.Equal(t, float64(0), listResp["code"])
}
