//go:build api
// +build api

// Package api 设备管理 API 测试
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

// setupDeviceAPITestDB 创建测试数据库
func setupDeviceAPITestDB(t *testing.T) *gorm.DB {
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
		&models.Admin{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.Device{},
		&models.DeviceLog{},
		&models.DeviceMaintenance{},
		&models.Venue{},
		&models.Merchant{},
	)
	require.NoError(t, err)

	return db
}

// setupDeviceAPIRouter 创建测试路由
func setupDeviceAPIRouter(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	db := setupDeviceAPITestDB(t)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-device-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	// 创建设备服务和处理器
	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	deviceMaintenanceRepo := repository.NewDeviceMaintenanceRepository(db)
	venueRepo := repository.NewVenueRepository(db)

	deviceService := adminService.NewDeviceAdminService(deviceRepo, deviceLogRepo, deviceMaintenanceRepo, venueRepo, nil)
	deviceHandler := adminHandler.NewDeviceHandler(deviceService)

	api := r.Group("/api/v1/admin")

	// 模拟认证中间件
	api.Use(func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.Next()
			return
		}

		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_type", claims.UserType)
		c.Next()
	})

	deviceHandler.RegisterRoutes(api)

	return r, db, jwtManager
}

// createDeviceAPITestAdmin 创建测试管理员并返回 token
func createDeviceAPITestAdmin(t *testing.T, db *gorm.DB, jwtManager *jwt.Manager, username string) string {
	role := &models.Role{
		Code:     "device_api_test_role_" + username,
		Name:     "测试角色",
		IsSystem: false,
	}
	err := db.Create(role).Error
	require.NoError(t, err)

	passwordHash, err := crypto.HashPassword("password123")
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

	tokenPair, err := jwtManager.GenerateTokenPair(admin.ID, jwt.UserTypeAdmin, role.Code)
	require.NoError(t, err)

	return tokenPair.AccessToken
}

// createDeviceAPITestVenue 创建测试场地
func createDeviceAPITestVenue(t *testing.T, db *gorm.DB) *models.Venue {
	merchant := &models.Merchant{
		Name:   "测试商户",
		Status: models.MerchantStatusActive,
	}
	err := db.Create(merchant).Error
	require.NoError(t, err)

	venue := &models.Venue{
		MerchantID: merchant.ID,
		Name:       "测试场地",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园路1号",
		Status:     models.VenueStatusActive,
	}
	err = db.Create(venue).Error
	require.NoError(t, err)

	return venue
}

// createDeviceAPITestDevice 创建测试设备
func createDeviceAPITestDevice(t *testing.T, db *gorm.DB, deviceNo string, venueID int64) *models.Device {
	device := &models.Device{
		DeviceNo:       deviceNo,
		Name:           "测试设备",
		Type:           "standard",
		VenueID:        venueID,
		ProductName:    "测试产品",
		SlotCount:      10,
		AvailableSlots: 10,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	err := db.Create(device).Error
	require.NoError(t, err)

	return device
}

func TestDeviceAPI_Create_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "create_device_admin")
	venue := createDeviceAPITestVenue(t, db)

	body := map[string]interface{}{
		"device_no":    "DEV_API_001",
		"name":         "API测试设备",
		"type":         "standard",
		"venue_id":     venue.ID,
		"product_name": "测试产品",
		"slot_count":   5,
		"network_type": "WiFi",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "DEV_API_001", data["device_no"])
	assert.Equal(t, "API测试设备", data["name"])
}

func TestDeviceAPI_Create_DuplicateDeviceNo(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "dup_device_admin")
	venue := createDeviceAPITestVenue(t, db)
	createDeviceAPITestDevice(t, db, "DEV_API_DUP", venue.ID)

	body := map[string]interface{}{
		"device_no":    "DEV_API_DUP",
		"name":         "重复设备",
		"type":         "standard",
		"venue_id":     venue.ID,
		"product_name": "测试产品",
		"slot_count":   5,
		"network_type": "WiFi",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeviceAPI_Create_VenueNotFound(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "venue_not_found_admin")

	body := map[string]interface{}{
		"device_no":    "DEV_API_002",
		"name":         "测试设备",
		"type":         "standard",
		"venue_id":     99999,
		"product_name": "测试产品",
		"slot_count":   5,
		"network_type": "WiFi",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeviceAPI_Create_NoToken(t *testing.T) {
	router, _, _ := setupDeviceAPIRouter(t)

	body := map[string]interface{}{
		"device_no":    "DEV_API_003",
		"name":         "测试设备",
		"type":         "standard",
		"venue_id":     1,
		"product_name": "测试产品",
		"slot_count":   5,
		"network_type": "WiFi",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeviceAPI_Get_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "get_device_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_GET", venue.ID)

	req, _ := http.NewRequest("GET", "/api/v1/admin/devices/"+string(rune(device.ID+'0')), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// 使用正确的 ID 格式
	req, _ = http.NewRequest("GET", "/api/v1/admin/devices/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
}

func TestDeviceAPI_Get_NotFound(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "get_not_found_admin")

	req, _ := http.NewRequest("GET", "/api/v1/admin/devices/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeviceAPI_Get_InvalidID(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "get_invalid_admin")

	req, _ := http.NewRequest("GET", "/api/v1/admin/devices/invalid", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeviceAPI_List_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "list_device_admin")
	venue := createDeviceAPITestVenue(t, db)
	createDeviceAPITestDevice(t, db, "DEV_API_LIST1", venue.ID)
	createDeviceAPITestDevice(t, db, "DEV_API_LIST2", venue.ID)
	createDeviceAPITestDevice(t, db, "DEV_API_LIST3", venue.ID)

	req, _ := http.NewRequest("GET", "/api/v1/admin/devices?page=1&page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["total"])
}


func TestDeviceAPI_List_WithFilters(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "list_filter_admin")
	venue := createDeviceAPITestVenue(t, db)
	device1 := createDeviceAPITestDevice(t, db, "DEV_API_FILTER1", venue.ID)
	device2 := createDeviceAPITestDevice(t, db, "DEV_API_FILTER2", venue.ID)

	// 设置一个设备离线
	db.Model(device2).Update("online_status", models.DeviceOffline)
	_ = device1

	req, _ := http.NewRequest("GET", "/api/v1/admin/devices?online_status=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])
}

func TestDeviceAPI_Update_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "update_device_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_UPDATE", venue.ID)

	body := map[string]interface{}{
		"name":         "更新后的设备",
		"type":         "premium",
		"venue_id":     venue.ID,
		"product_name": "更新后的产品",
		"slot_count":   20,
		"network_type": "4G",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/v1/admin/devices/"+strconv.FormatInt(device.ID, 10), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证更新
	var updatedDevice models.Device
	db.First(&updatedDevice, device.ID)
	assert.Equal(t, "更新后的设备", updatedDevice.Name)
	assert.Equal(t, "premium", updatedDevice.Type)
}

func TestDeviceAPI_Update_NotFound(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "update_not_found_admin")
	venue := createDeviceAPITestVenue(t, db)

	body := map[string]interface{}{
		"name":         "更新设备",
		"type":         "standard",
		"venue_id":     venue.ID,
		"product_name": "产品",
		"slot_count":   10,
		"network_type": "WiFi",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/v1/admin/devices/99999", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeviceAPI_UpdateStatus_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "status_device_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_STATUS", venue.ID)

	body := map[string]interface{}{
		"status": models.DeviceStatusMaintenance,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/v1/admin/devices/"+strconv.FormatInt(device.ID, 10)+"/status", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证状态更新
	var updatedDevice models.Device
	db.First(&updatedDevice, device.ID)
	assert.EqualValues(t, models.DeviceStatusMaintenance, updatedDevice.Status)
}

func TestDeviceAPI_UpdateStatus_DeviceInUse(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "status_inuse_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_INUSE", venue.ID)

	// 设置设备为使用中
	db.Model(device).Update("rental_status", models.DeviceRentalInUse)

	body := map[string]interface{}{
		"status": models.DeviceStatusDisabled,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/v1/admin/devices/"+strconv.FormatInt(device.ID, 10)+"/status", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeviceAPI_GetStatistics_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "stats_device_admin")
	venue := createDeviceAPITestVenue(t, db)
	createDeviceAPITestDevice(t, db, "DEV_API_STAT1", venue.ID)
	device2 := createDeviceAPITestDevice(t, db, "DEV_API_STAT2", venue.ID)
	device3 := createDeviceAPITestDevice(t, db, "DEV_API_STAT3", venue.ID)

	// 设置不同状态
	db.Model(device2).Updates(map[string]interface{}{
		"online_status": models.DeviceOffline,
		"status":        models.DeviceStatusMaintenance,
	})
	db.Model(device3).Update("rental_status", models.DeviceRentalInUse)

	req, _ := http.NewRequest("GET", "/api/v1/admin/devices/statistics", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["total"])
}


func TestDeviceAPI_GetLogs_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "logs_device_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_LOGS", venue.ID)

	// 创建日志
	for i := 0; i < 3; i++ {
		content := "测试日志"
		log := &models.DeviceLog{
			DeviceID: device.ID,
			Type:     models.DeviceLogTypeOnline,
			Content:  &content,
		}
		db.Create(log)
	}

	req, _ := http.NewRequest("GET", "/api/v1/admin/devices/"+strconv.FormatInt(device.ID, 10)+"/logs", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["total"])
}


func TestDeviceAPI_CreateMaintenance_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "maint_device_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_MAINT", venue.ID)

	body := map[string]interface{}{
		"device_id":   device.ID,
		"type":        "repair",
		"description": "更换零件",
		"cost":        100.0,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices/maintenance", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	// 验证设备状态已变为维护中
	var updatedDevice models.Device
	db.First(&updatedDevice, device.ID)
	assert.EqualValues(t, models.DeviceStatusMaintenance, updatedDevice.Status)
}

func TestDeviceAPI_CreateMaintenance_DeviceInUse(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "maint_inuse_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_MAINT_INUSE", venue.ID)

	// 设置设备为使用中
	db.Model(device).Update("rental_status", models.DeviceRentalInUse)

	body := map[string]interface{}{
		"device_id":   device.ID,
		"type":        "repair",
		"description": "更换零件",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices/maintenance", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeviceAPI_CompleteMaintenance_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "complete_maint_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_COMPLETE", venue.ID)

	// 创建维护记录
	maintenance := &models.DeviceMaintenance{
		DeviceID:    device.ID,
		Type:        "repair",
		Description: "更换零件",
		OperatorID:  1,
		Status:      models.MaintenanceStatusInProgress,
	}
	db.Create(maintenance)
	db.Model(device).Update("status", models.DeviceStatusMaintenance)

	body := map[string]interface{}{
		"cost": 150.0,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices/maintenance/"+strconv.FormatInt(maintenance.ID, 10)+"/complete", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证维护状态
	var updatedMaintenance models.DeviceMaintenance
	db.First(&updatedMaintenance, maintenance.ID)
	assert.EqualValues(t, models.MaintenanceStatusCompleted, updatedMaintenance.Status)

	// 验证设备状态恢复
	var updatedDevice models.Device
	db.First(&updatedDevice, device.ID)
	assert.EqualValues(t, models.DeviceStatusActive, updatedDevice.Status)
}

func TestDeviceAPI_CompleteMaintenance_NotFound(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "complete_not_found_admin")

	body := map[string]interface{}{
		"cost": 150.0,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices/maintenance/99999/complete", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeviceAPI_CompleteMaintenance_AlreadyCompleted(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "already_complete_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_ALREADY", venue.ID)

	// 创建已完成的维护记录
	now := time.Now()
	maintenance := &models.DeviceMaintenance{
		DeviceID:    device.ID,
		Type:        "repair",
		Description: "更换零件",
		OperatorID:  1,
		Status:      models.MaintenanceStatusCompleted,
		CompletedAt: &now,
	}
	db.Create(maintenance)

	body := map[string]interface{}{
		"cost": 150.0,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices/maintenance/"+strconv.FormatInt(maintenance.ID, 10)+"/complete", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeviceAPI_ListMaintenance_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "list_maint_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_LIST_MAINT", venue.ID)

	// 创建多个维护记录
	for i := 0; i < 3; i++ {
		maintenance := &models.DeviceMaintenance{
			DeviceID:    device.ID,
			Type:        "repair",
			Description: "测试维护",
			OperatorID:  1,
			Status:      models.MaintenanceStatusInProgress,
		}
		db.Create(maintenance)
	}

	req, _ := http.NewRequest("GET", "/api/v1/admin/devices/maintenance?page=1&page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["total"])
}

func TestDeviceAPI_RemoteUnlock_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "unlock_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_UNLOCK", venue.ID)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices/"+strconv.FormatInt(device.ID, 10)+"/unlock", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceUnlocked), updated.LockStatus)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, models.DeviceLogTypeUnlock).Find(&logs).Error)
	assert.Len(t, logs, 1)
}

func TestDeviceAPI_RemoteUnlock_DeviceOffline(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "unlock_offline_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_UNLOCK_OFF", venue.ID)
	require.NoError(t, db.Model(device).Update("online_status", models.DeviceOffline).Error)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices/"+strconv.FormatInt(device.ID, 10)+"/unlock", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceLocked), updated.LockStatus)
}

func TestDeviceAPI_RemoteLock_Success(t *testing.T) {
	router, db, jwtManager := setupDeviceAPIRouter(t)

	token := createDeviceAPITestAdmin(t, db, jwtManager, "lock_admin")
	venue := createDeviceAPITestVenue(t, db)
	device := createDeviceAPITestDevice(t, db, "DEV_API_LOCK", venue.ID)
	require.NoError(t, db.Model(device).Update("lock_status", models.DeviceUnlocked).Error)

	req, _ := http.NewRequest("POST", "/api/v1/admin/devices/"+strconv.FormatInt(device.ID, 10)+"/lock", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceLocked), updated.LockStatus)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, models.DeviceLogTypeLock).Find(&logs).Error)
	assert.Len(t, logs, 1)
}

