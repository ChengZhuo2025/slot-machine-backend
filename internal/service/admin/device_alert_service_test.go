package admin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func TestDeviceAlertService_CheckDeviceHealth_GeneratesAlerts(t *testing.T) {
	db := setupAdminServiceTestDB(t)
	ctx := context.Background()

	merchant := &models.Merchant{Name: "商户A", ContactName: "联系人", ContactPhone: "13900139000", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "场地A", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)

	battery := 5
	signal := 10
	temp := 60.0
	lastHeartbeat := time.Now().Add(-2 * time.Hour)

	device := &models.Device{
		DeviceNo:        "DEV_ALERT_001",
		Name:            "告警设备",
		Type:            models.DeviceTypeStandard,
		VenueID:         venue.ID,
		ProductName:     "测试产品",
		SlotCount:       1,
		AvailableSlots:  1,
		OnlineStatus:    models.DeviceOffline,
		LockStatus:      models.DeviceLocked,
		RentalStatus:    models.DeviceRentalFree,
		NetworkType:     "WiFi",
		Status:          models.DeviceStatusFault,
		BatteryLevel:    &battery,
		SignalStrength:  &signal,
		Temperature:     &temp,
		LastHeartbeatAt: &lastHeartbeat,
	}
	require.NoError(t, db.Create(device).Error)

	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	alertRepo := repository.NewDeviceAlertRepository(db)
	svc := NewDeviceAlertService(deviceRepo, deviceLogRepo, alertRepo)

	alerts, err := svc.CheckDeviceHealth(ctx, device.ID)
	require.NoError(t, err)
	require.NotEmpty(t, alerts)

	types := make(map[AlertType]bool)
	for _, a := range alerts {
		types[a.Type] = true
	}

	assert.True(t, types[AlertTypeLowBattery])
	assert.True(t, types[AlertTypeLowSignal])
	assert.True(t, types[AlertTypeHighTemperature])
	assert.True(t, types[AlertTypeFault])
	assert.True(t, types[AlertTypeOffline])
}

func TestDeviceAlertService_CreateAndResolveAlert_PersistsAndLogs(t *testing.T) {
	db := setupAdminServiceTestDB(t)
	ctx := context.Background()

	merchant := &models.Merchant{Name: "商户A", ContactName: "联系人", ContactPhone: "13900139000", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "场地A", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)
	device := &models.Device{
		DeviceNo:       "DEV_ALERT_002",
		Name:           "告警设备2",
		Type:           models.DeviceTypeStandard,
		VenueID:        venue.ID,
		ProductName:    "测试产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	require.NoError(t, db.Create(device).Error)

	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	alertRepo := repository.NewDeviceAlertRepository(db)
	svc := NewDeviceAlertService(deviceRepo, deviceLogRepo, alertRepo)

	created, err := svc.CreateAlert(ctx, device.ID, AlertTypeFault, AlertLevelCritical, "设备故障", "设备处于故障状态")
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotZero(t, created.ID)

	var dbAlert models.DeviceAlert
	require.NoError(t, db.First(&dbAlert, created.ID).Error)
	assert.Equal(t, device.ID, dbAlert.DeviceID)
	assert.Equal(t, string(AlertTypeFault), dbAlert.Type)
	assert.Equal(t, string(AlertLevelCritical), dbAlert.Level)
	assert.Equal(t, "设备故障", dbAlert.Title)

	var logCount int64
	require.NoError(t, db.Model(&models.DeviceLog{}).Where("device_id = ? AND type = ?", device.ID, "alert").Count(&logCount).Error)
	assert.Equal(t, int64(1), logCount)

	operatorID := int64(100)
	require.NoError(t, svc.ResolveAlert(ctx, created.ID, operatorID))
	require.NoError(t, db.First(&dbAlert, created.ID).Error)
	assert.True(t, dbAlert.IsResolved)
	require.NotNil(t, dbAlert.ResolvedBy)
	assert.Equal(t, operatorID, *dbAlert.ResolvedBy)
	require.NotNil(t, dbAlert.ResolvedAt)
}

