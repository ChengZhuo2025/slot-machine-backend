package device

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
)

func setupDeviceServiceTestDB(t *testing.T) *gorm.DB {
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
	))

	return db
}

func seedMerchantVenueDevice(t *testing.T, db *gorm.DB, deviceNo string, onlineStatus int8) (*models.Venue, *models.Device) {
	t.Helper()

	merchant := &models.Merchant{Name: "测试商户", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)

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
	require.NoError(t, db.Create(venue).Error)

	device := &models.Device{
		DeviceNo:       deviceNo,
		Name:           "测试设备",
		Type:           models.DeviceTypeStandard,
		VenueID:        venue.ID,
		QRCode:         "QR_" + deviceNo,
		ProductName:    "测试产品",
		SlotCount:      10,
		AvailableSlots: 10,
		OnlineStatus:   onlineStatus,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	require.NoError(t, db.Create(device).Error)

	return venue, device
}

func TestDeviceService_UpdateDeviceHeartbeat_OfflineToOnline_CreatesOnlineLog(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	_, device := seedMerchantVenueDevice(t, db, "DEV_HB_1", models.DeviceOffline)

	signal := 80
	battery := 90
	firmware := "v1.2.3"
	data := &HeartbeatData{SignalStrength: &signal, BatteryLevel: &battery, FirmwareVersion: &firmware}

	err := svc.UpdateDeviceHeartbeat(context.Background(), device.DeviceNo, data)
	require.NoError(t, err)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceOnline), updated.OnlineStatus)
	assert.NotNil(t, updated.LastHeartbeatAt)
	assert.NotNil(t, updated.LastOnlineAt)
	require.NotNil(t, updated.SignalStrength)
	assert.Equal(t, 80, *updated.SignalStrength)
	require.NotNil(t, updated.BatteryLevel)
	assert.Equal(t, 90, *updated.BatteryLevel)
	require.NotNil(t, updated.FirmwareVersion)
	assert.Equal(t, "v1.2.3", *updated.FirmwareVersion)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, models.DeviceLogTypeOnline).Find(&logs).Error)
	require.Len(t, logs, 1)
	require.NotNil(t, logs[0].OperatorType)
	assert.Equal(t, models.DeviceLogOperatorSystem, *logs[0].OperatorType)
}

func TestDeviceService_UpdateDeviceHeartbeat_AlreadyOnline_DoesNotCreateOnlineLog(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	_, device := seedMerchantVenueDevice(t, db, "DEV_HB_2", models.DeviceOnline)

	err := svc.UpdateDeviceHeartbeat(context.Background(), device.DeviceNo, &HeartbeatData{})
	require.NoError(t, err)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, models.DeviceLogTypeOnline).Find(&logs).Error)
	assert.Len(t, logs, 0)
}

func TestDeviceService_SetDeviceOffline_UpdatesFieldsAndCreatesOfflineLog(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	_, device := seedMerchantVenueDevice(t, db, "DEV_OFF_1", models.DeviceOnline)

	err := svc.SetDeviceOffline(context.Background(), device.ID)
	require.NoError(t, err)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceOffline), updated.OnlineStatus)
	assert.NotNil(t, updated.LastOfflineAt)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, models.DeviceLogTypeOffline).Find(&logs).Error)
	require.Len(t, logs, 1)
	require.NotNil(t, logs[0].OperatorType)
	assert.Equal(t, models.DeviceLogOperatorSystem, *logs[0].OperatorType)
}
