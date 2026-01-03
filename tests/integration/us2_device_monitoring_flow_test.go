//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
)

func setupUS2DeviceMonitoringIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.DeviceLog{},
		&models.DeviceMaintenance{},
	))

	return db
}

func TestUS2DeviceMonitoringFlow_HeartbeatThenAdminManage(t *testing.T) {
	db := setupUS2DeviceMonitoringIntegrationDB(t)
	ctx := context.Background()

	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	deviceMaintenanceRepo := repository.NewDeviceMaintenanceRepository(db)
	venueRepo := repository.NewVenueRepository(db)

	adminSvc := adminService.NewDeviceAdminService(deviceRepo, deviceLogRepo, deviceMaintenanceRepo, venueRepo, nil)
	deviceSvc := deviceService.NewDeviceService(db, deviceRepo, venueRepo)

	merchant := &models.Merchant{Name: "测试商户", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "测试场地", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)

	device := &models.Device{DeviceNo: "D_US2_001", Name: "US2设备", Type: models.DeviceTypeStandard, VenueID: venue.ID, QRCode: "QR_D_US2_001", ProductName: "测试产品", SlotCount: 10, AvailableSlots: 10, OnlineStatus: models.DeviceOffline, LockStatus: models.DeviceLocked, RentalStatus: models.DeviceRentalFree, NetworkType: "WiFi", Status: models.DeviceStatusActive}
	require.NoError(t, db.Create(device).Error)

	// 1) 设备心跳 -> 离线变在线 + 写上线日志
	signal := 75
	battery := 66
	firmware := "v9.9.9"
	require.NoError(t, deviceSvc.UpdateDeviceHeartbeat(ctx, device.DeviceNo, &deviceService.HeartbeatData{SignalStrength: &signal, BatteryLevel: &battery, FirmwareVersion: &firmware}))

	// 2) 管理员统计应看到在线数=1
	stats, err := adminSvc.GetDeviceStatistics(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Total)
	assert.Equal(t, int64(1), stats.Online)
	assert.Equal(t, int64(0), stats.Offline)

	// 3) 远程开锁（不依赖 mqttService）
	require.NoError(t, adminSvc.RemoteUnlock(ctx, device.ID, 1001))

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceUnlocked), updated.LockStatus)

	// 4) 管理员查看日志：应至少包含上线 + 开锁
	logs, total, err := adminSvc.GetDeviceLogs(ctx, device.ID, 0, 20, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(2))
	assert.GreaterOrEqual(t, len(logs), 2)
}
