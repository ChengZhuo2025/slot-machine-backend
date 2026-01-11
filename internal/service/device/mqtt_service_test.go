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
	"github.com/dumeirei/smart-locker-backend/pkg/mqtt"
)

func setupMQTTServiceTestDB(t *testing.T) *gorm.DB {
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

func seedDeviceForMQTT(t *testing.T, db *gorm.DB, deviceNo string) *models.Device {
	t.Helper()
	merchant := &models.Merchant{Name: "测试商户", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "测试场地", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
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
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	require.NoError(t, db.Create(device).Error)
	return device
}

func TestMQTTService_OnStatus_UpdatesDeviceFields(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	device := seedDeviceForMQTT(t, db, "DEV_MQTT_STATUS")

	payload := &mqtt.StatusPayload{OnlineStatus: models.DeviceOnline, LockStatus: models.DeviceUnlocked, RentalStatus: models.DeviceRentalInUse, AvailableSlots: 7}
	err := mqttSvc.OnStatus(context.Background(), device.DeviceNo, payload)
	require.NoError(t, err)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceUnlocked), updated.LockStatus)
	assert.Equal(t, int8(models.DeviceRentalInUse), updated.RentalStatus)
	assert.Equal(t, 7, updated.AvailableSlots)
}

func TestMQTTService_OnEvent_Unlocked_CreatesLogAndUpdatesLockStatus(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	device := seedDeviceForMQTT(t, db, "DEV_MQTT_EVENT_1")

	payload := &mqtt.EventPayload{EventType: mqtt.EventUnlocked, Data: map[string]interface{}{"message": "unlock ok"}}
	err := mqttSvc.OnEvent(context.Background(), device.DeviceNo, payload)
	require.NoError(t, err)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceUnlocked), updated.LockStatus)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, models.DeviceLogTypeUnlock).Find(&logs).Error)
	require.Len(t, logs, 1)
	require.NotNil(t, logs[0].OperatorType)
	assert.Equal(t, models.DeviceLogOperatorSystem, *logs[0].OperatorType)
	require.NotNil(t, logs[0].Content)
	assert.Equal(t, "unlock ok", *logs[0].Content)
}

func TestMQTTService_OnEvent_Error_SetsDeviceFault(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	device := seedDeviceForMQTT(t, db, "DEV_MQTT_EVENT_2")

	payload := &mqtt.EventPayload{EventType: mqtt.EventError, Data: map[string]interface{}{"message": "motor error"}}
	err := mqttSvc.OnEvent(context.Background(), device.DeviceNo, payload)
	require.NoError(t, err)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceStatusFault), updated.Status)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, models.DeviceLogTypeError).Find(&logs).Error)
	assert.Len(t, logs, 1)
}

func TestMQTTService_OnAck_NoSender_DoesNotFail(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	err := mqttSvc.OnAck(context.Background(), "DEV_ACK", &mqtt.AckPayload{CommandID: "cmd1", Success: true})
	require.NoError(t, err)
}

func TestMQTTService_OnHeartbeat_UpdatesDeviceInfo(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	device := seedDeviceForMQTT(t, db, "DEV_HEARTBEAT")

	payload := &mqtt.HeartbeatPayload{
		SignalStrength:  -65,
		BatteryLevel:    85,
		Temperature:     25.5,
		Humidity:        60.0,
		FirmwareVersion: "v1.2.3",
		LockStatus:      int8(models.DeviceLocked),
	}

	err := mqttSvc.OnHeartbeat(context.Background(), device.DeviceNo, payload)
	require.NoError(t, err)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	require.NotNil(t, updated.SignalStrength)
	assert.Equal(t, -65, *updated.SignalStrength)
	require.NotNil(t, updated.BatteryLevel)
	assert.Equal(t, 85, *updated.BatteryLevel)
	require.NotNil(t, updated.FirmwareVersion)
	assert.Equal(t, "v1.2.3", *updated.FirmwareVersion)
}

func TestMQTTService_OnHeartbeat_DeviceNotFound(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)

	payload := &mqtt.HeartbeatPayload{
		SignalStrength: -65,
		BatteryLevel:   85,
	}

	err := mqttSvc.OnHeartbeat(context.Background(), "NON_EXISTENT", payload)
	assert.Error(t, err)
}

func TestMQTTService_SendUnlockCommand_NoSender(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	// No commandSender
	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)

	result, err := mqttSvc.SendUnlockCommand(context.Background(), "DEV001", nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMQTTService_SendLockCommand_NoSender(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	// No commandSender
	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)

	result, err := mqttSvc.SendLockCommand(context.Background(), "DEV001", nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMQTTService_Unlock_DeviceNotFound(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)

	err := mqttSvc.Unlock(context.Background(), 99999)
	assert.Error(t, err)
}

func TestMQTTService_Unlock_Success_NoSender(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	device := seedDeviceForMQTT(t, db, "DEV_UNLOCK_1")

	err := mqttSvc.Unlock(context.Background(), device.ID)
	require.NoError(t, err)
}

func TestMQTTService_SendUnlockCommandAsync_NoSender(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)

	commandID, err := mqttSvc.SendUnlockCommandAsync(context.Background(), "DEV001", nil)
	require.NoError(t, err)
	assert.Empty(t, commandID)
}

func TestMQTTService_OnEvent_Locked(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	device := seedDeviceForMQTT(t, db, "DEV_MQTT_LOCKED")

	// Set device to unlocked first
	db.Model(&models.Device{}).Where("id = ?", device.ID).Update("lock_status", models.DeviceUnlocked)

	payload := &mqtt.EventPayload{EventType: mqtt.EventLocked, Data: nil}
	err := mqttSvc.OnEvent(context.Background(), device.DeviceNo, payload)
	require.NoError(t, err)

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, int8(models.DeviceLocked), updated.LockStatus)
}

func TestMQTTService_OnEvent_Alarm(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	device := seedDeviceForMQTT(t, db, "DEV_MQTT_ALARM")

	payload := &mqtt.EventPayload{EventType: mqtt.EventAlarm, Data: map[string]interface{}{"message": "alarm triggered"}}
	err := mqttSvc.OnEvent(context.Background(), device.DeviceNo, payload)
	require.NoError(t, err)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, models.DeviceLogTypeError).Find(&logs).Error)
	assert.Len(t, logs, 1)
}

func TestMQTTService_OnEvent_UnknownType(t *testing.T) {
	db := setupMQTTServiceTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	deviceSvc := NewDeviceService(db, deviceRepo, venueRepo)

	mqttSvc := NewMQTTService(deviceRepo, deviceSvc, nil)
	device := seedDeviceForMQTT(t, db, "DEV_MQTT_UNKNOWN")

	payload := &mqtt.EventPayload{EventType: "custom_event", Data: nil}
	err := mqttSvc.OnEvent(context.Background(), device.DeviceNo, payload)
	require.NoError(t, err)

	var logs []models.DeviceLog
	require.NoError(t, db.Where("device_id = ? AND type = ?", device.ID, "custom_event").Find(&logs).Error)
	assert.Len(t, logs, 1)
}

func TestMQTTService_HelperFunctions(t *testing.T) {
	t.Run("intPtr", func(t *testing.T) {
		result := intPtr(42)
		require.NotNil(t, result)
		assert.Equal(t, 42, *result)
	})

	t.Run("int8Ptr", func(t *testing.T) {
		result := int8Ptr(8)
		require.NotNil(t, result)
		assert.Equal(t, int8(8), *result)
	})

	t.Run("float64Ptr with value", func(t *testing.T) {
		result := float64Ptr(3.14)
		require.NotNil(t, result)
		assert.Equal(t, 3.14, *result)
	})

	t.Run("float64Ptr with zero", func(t *testing.T) {
		result := float64Ptr(0)
		assert.Nil(t, result)
	})
}
