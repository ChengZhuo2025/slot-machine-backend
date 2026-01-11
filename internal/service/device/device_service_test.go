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
		&models.RentalPricing{},
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
func TestDeviceService_GetDeviceByQRCode(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	_, device := seedMerchantVenueDevice(t, db, "DEV_QR_1", models.DeviceOnline)

	t.Run("成功获取设备信息", func(t *testing.T) {
		info, err := svc.GetDeviceByQRCode(context.Background(), device.QRCode)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, device.ID, info.ID)
		assert.Equal(t, device.DeviceNo, info.DeviceNo)
		assert.NotNil(t, info.Venue)
	})

	t.Run("设备不存在", func(t *testing.T) {
		_, err := svc.GetDeviceByQRCode(context.Background(), "INVALID_QR")
		assert.Error(t, err)
	})

	t.Run("设备已禁用", func(t *testing.T) {
		db.Model(&models.Device{}).Where("id = ?", device.ID).Update("status", models.DeviceStatusDisabled)
		_, err := svc.GetDeviceByQRCode(context.Background(), device.QRCode)
		assert.Error(t, err)
		// Restore
		db.Model(&models.Device{}).Where("id = ?", device.ID).Update("status", models.DeviceStatusActive)
	})
}

func TestDeviceService_GetDeviceByNo(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	_, device := seedMerchantVenueDevice(t, db, "DEV_NO_1", models.DeviceOnline)

	t.Run("成功获取设备信息", func(t *testing.T) {
		info, err := svc.GetDeviceByNo(context.Background(), device.DeviceNo)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, device.ID, info.ID)
		assert.NotNil(t, info.Venue)
	})

	t.Run("设备不存在", func(t *testing.T) {
		_, err := svc.GetDeviceByNo(context.Background(), "INVALID_NO")
		assert.Error(t, err)
	})
}

func TestDeviceService_GetDeviceByID(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	_, device := seedMerchantVenueDevice(t, db, "DEV_ID_1", models.DeviceOnline)

	t.Run("成功获取设备信息", func(t *testing.T) {
		info, err := svc.GetDeviceByID(context.Background(), device.ID)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, device.ID, info.ID)
	})

	t.Run("设备不存在", func(t *testing.T) {
		_, err := svc.GetDeviceByID(context.Background(), 99999)
		assert.Error(t, err)
	})
}

func TestDeviceService_CheckDeviceAvailable(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	_, device := seedMerchantVenueDevice(t, db, "DEV_CHK_1", models.DeviceOnline)

	t.Run("设备可用", func(t *testing.T) {
		err := svc.CheckDeviceAvailable(context.Background(), device.ID)
		assert.NoError(t, err)
	})

	t.Run("设备不存在", func(t *testing.T) {
		err := svc.CheckDeviceAvailable(context.Background(), 99999)
		assert.Error(t, err)
	})

	t.Run("设备已禁用", func(t *testing.T) {
		db.Model(&models.Device{}).Where("id = ?", device.ID).Update("status", models.DeviceStatusDisabled)
		err := svc.CheckDeviceAvailable(context.Background(), device.ID)
		assert.Error(t, err)
		db.Model(&models.Device{}).Where("id = ?", device.ID).Update("status", models.DeviceStatusActive)
	})

	t.Run("设备离线", func(t *testing.T) {
		db.Model(&models.Device{}).Where("id = ?", device.ID).Update("online_status", models.DeviceOffline)
		err := svc.CheckDeviceAvailable(context.Background(), device.ID)
		assert.Error(t, err)
		db.Model(&models.Device{}).Where("id = ?", device.ID).Update("online_status", models.DeviceOnline)
	})

	t.Run("无可用槽位", func(t *testing.T) {
		db.Model(&models.Device{}).Where("id = ?", device.ID).Update("available_slots", 0)
		err := svc.CheckDeviceAvailable(context.Background(), device.ID)
		assert.Error(t, err)
		db.Model(&models.Device{}).Where("id = ?", device.ID).Update("available_slots", 10)
	})
}

func TestDeviceService_GetPricing(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	venue, _ := seedMerchantVenueDevice(t, db, "DEV_PR_1", models.DeviceOnline)

	pricing := &models.RentalPricing{
		VenueID:       &venue.ID,
		DurationHours: 1,
		Price:         10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		IsActive:      true,
	}
	require.NoError(t, db.Create(pricing).Error)

	t.Run("成功获取定价", func(t *testing.T) {
		info, err := svc.GetPricing(context.Background(), pricing.ID)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, pricing.ID, info.ID)
		assert.Equal(t, pricing.Price, info.Price)
	})

	t.Run("定价不存在", func(t *testing.T) {
		_, err := svc.GetPricing(context.Background(), 99999)
		assert.Error(t, err)
	})
}

func TestDeviceService_GetDevicePricings(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	_, device := seedMerchantVenueDevice(t, db, "DEV_PRS_1", models.DeviceOnline)

	t.Run("成功获取定价列表", func(t *testing.T) {
		list, err := svc.GetDevicePricings(context.Background(), device.ID)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})

	t.Run("设备不存在", func(t *testing.T) {
		_, err := svc.GetDevicePricings(context.Background(), 99999)
		assert.Error(t, err)
	})
}

func TestDeviceService_ListVenueDevices(t *testing.T) {
	db := setupDeviceServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	svc := NewDeviceService(db, deviceRepo, venueRepo)

	venue, _ := seedMerchantVenueDevice(t, db, "DEV_LST_1", models.DeviceOnline)

	t.Run("成功获取场地设备列表", func(t *testing.T) {
		list, err := svc.ListVenueDevices(context.Background(), venue.ID)
		require.NoError(t, err)
		assert.NotNil(t, list)
		assert.Greater(t, len(list), 0)
	})

	t.Run("场地不存在", func(t *testing.T) {
		_, err := svc.ListVenueDevices(context.Background(), 99999)
		assert.Error(t, err)
	})
}
