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
