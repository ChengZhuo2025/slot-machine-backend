//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
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
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
	"github.com/dumeirei/smart-locker-backend/pkg/mqtt"
)

func setupUS2AdminDeviceE2E(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager, *deviceService.MQTTService) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
		&models.DeviceLog{},
		&models.DeviceMaintenance{},
	))

	jwtManager := jwt.NewManager(&jwt.Config{Secret: "test-secret-key-us2-e2e", AccessExpireTime: time.Hour, RefreshExpireTime: 2 * time.Hour, Issuer: "test"})

	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	deviceMaintenanceRepo := repository.NewDeviceMaintenanceRepository(db)
	venueRepo := repository.NewVenueRepository(db)

	adminDeviceSvc := adminService.NewDeviceAdminService(deviceRepo, deviceLogRepo, deviceMaintenanceRepo, venueRepo, nil)
	adminDeviceH := adminHandler.NewDeviceHandler(adminDeviceSvc)

	deviceSvc := deviceService.NewDeviceService(db, deviceRepo, venueRepo)
	mqttSvc := deviceService.NewMQTTService(deviceRepo, deviceSvc, nil)

	api := engine.Group("/api/v1/admin")
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
	adminDeviceH.RegisterRoutes(api)

	_ = mqttSvc
	return engine, db, jwtManager, mqttSvc
}

func createUS2E2EAdminToken(t *testing.T, db *gorm.DB, jwtManager *jwt.Manager) string {
	t.Helper()
	role := &models.Role{Code: "us2_e2e_role", Name: "测试角色", IsSystem: false}
	require.NoError(t, db.Create(role).Error)
	passwordHash, err := crypto.HashPassword("password123")
	require.NoError(t, err)
	admin := &models.Admin{Username: "us2_admin", PasswordHash: passwordHash, Name: "管理员", RoleID: role.ID, Status: models.AdminStatusActive}
	require.NoError(t, db.Create(admin).Error)
	pair, err := jwtManager.GenerateTokenPair(admin.ID, jwt.UserTypeAdmin, role.Code)
	require.NoError(t, err)
	return pair.AccessToken
}

func TestUS2_E2E_AdminDeviceMonitorAndManage(t *testing.T) {
	router, db, jwtManager, mqttSvc := setupUS2AdminDeviceE2E(t)
	ctx := context.Background()

	authz := "Bearer " + createUS2E2EAdminToken(t, db, jwtManager)

	merchant := &models.Merchant{Name: "测试商户", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "测试场地", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)

	device := &models.Device{DeviceNo: "D_US2_E2E_001", Name: "US2设备", Type: models.DeviceTypeStandard, VenueID: venue.ID, QRCode: "QR_D_US2_E2E_001", ProductName: "测试产品", SlotCount: 10, AvailableSlots: 10, OnlineStatus: models.DeviceOffline, LockStatus: models.DeviceLocked, RentalStatus: models.DeviceRentalFree, NetworkType: "WiFi", Status: models.DeviceStatusActive}
	require.NoError(t, db.Create(device).Error)

	// 1) 模拟设备上报心跳（离线->在线）
	require.NoError(t, mqttSvc.OnHeartbeat(ctx, device.DeviceNo, &mqtt.HeartbeatPayload{SignalStrength: 88, BatteryLevel: 77, FirmwareVersion: "v1.0.0"}))

	// 2) 管理员查看统计（在线数应为 1）
	statReq, _ := http.NewRequest("GET", "/api/v1/admin/devices/statistics", nil)
	statReq.Header.Set("Authorization", authz)
	statW := httptest.NewRecorder()
	router.ServeHTTP(statW, statReq)
	require.Equal(t, http.StatusOK, statW.Code)

	var statResp apiResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(statW.Body.Bytes())).Decode(&statResp))
	require.Equal(t, 0, statResp.Code)
	var statData map[string]interface{}
	require.NoError(t, json.Unmarshal(statResp.Data, &statData))
	assert.Equal(t, float64(1), statData["online"])

	// 3) 管理员远程开锁
	unlockReq, _ := http.NewRequest("POST", "/api/v1/admin/devices/"+strconv.FormatInt(device.ID, 10)+"/unlock", nil)
	unlockReq.Header.Set("Authorization", authz)
	unlockW := httptest.NewRecorder()
	router.ServeHTTP(unlockW, unlockReq)
	require.Equal(t, http.StatusOK, unlockW.Code)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceUnlocked), updated.LockStatus)

	// 4) 创建设备维护记录（进入维护中）
	maintBody, _ := json.Marshal(map[string]interface{}{"device_id": device.ID, "type": "repair", "description": "例行维护", "cost": 12.5})
	maintReq, _ := http.NewRequest("POST", "/api/v1/admin/devices/maintenance", bytes.NewBuffer(maintBody))
	maintReq.Header.Set("Content-Type", "application/json")
	maintReq.Header.Set("Authorization", authz)
	maintW := httptest.NewRecorder()
	router.ServeHTTP(maintW, maintReq)
	require.Equal(t, http.StatusOK, maintW.Code)

	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceStatusMaintenance), updated.Status)

	// 5) 完成维护（恢复正常）
	var maint models.DeviceMaintenance
	require.NoError(t, db.Where("device_id = ?", device.ID).Order("id DESC").First(&maint).Error)
	completeBody, _ := json.Marshal(map[string]interface{}{"cost": 20.0})
	completeReq, _ := http.NewRequest("POST", "/api/v1/admin/devices/maintenance/"+strconv.FormatInt(maint.ID, 10)+"/complete", bytes.NewBuffer(completeBody))
	completeReq.Header.Set("Content-Type", "application/json")
	completeReq.Header.Set("Authorization", authz)
	completeW := httptest.NewRecorder()
	router.ServeHTTP(completeW, completeReq)
	require.Equal(t, http.StatusOK, completeW.Code)

	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceStatusActive), updated.Status)
}
