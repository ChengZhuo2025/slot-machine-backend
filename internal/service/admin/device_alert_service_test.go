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

func TestDeviceAlertService_CheckAllDevices(t *testing.T) {
	db := setupAdminServiceTestDB(t)
	ctx := context.Background()

	merchant := &models.Merchant{Name: "商户A", ContactName: "联系人", ContactPhone: "13900139000", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "场地A", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)

	// 创建多个设备
	for i := 0; i < 3; i++ {
		battery := 5
		device := &models.Device{
			DeviceNo:       "DEV_CHECK_" + string(rune('A'+i)),
			Name:           "检测设备",
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
			BatteryLevel:   &battery,
		}
		require.NoError(t, db.Create(device).Error)
	}

	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	alertRepo := repository.NewDeviceAlertRepository(db)
	svc := NewDeviceAlertService(deviceRepo, deviceLogRepo, alertRepo)

	alerts, err := svc.CheckAllDevices(ctx)
	require.NoError(t, err)
	assert.NotNil(t, alerts)
}

func TestDeviceAlertService_ListAlerts(t *testing.T) {
	db := setupAdminServiceTestDB(t)
	ctx := context.Background()

	merchant := &models.Merchant{Name: "商户A", ContactName: "联系人", ContactPhone: "13900139000", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "场地A", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)
	device := &models.Device{
		DeviceNo:       "DEV_LIST_ALERT",
		Name:           "告警列表设备",
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

	// 创建一些告警记录
	for i := 0; i < 3; i++ {
		alert := &models.DeviceAlert{
			DeviceID: device.ID,
			Type:     string(AlertTypeFault),
			Level:    string(AlertLevelWarning),
			Title:    "告警" + string(rune('0'+i)),
			Content:  "告警消息",
		}
		require.NoError(t, db.Create(alert).Error)
	}

	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	alertRepo := repository.NewDeviceAlertRepository(db)
	svc := NewDeviceAlertService(deviceRepo, deviceLogRepo, alertRepo)

	alerts, total, err := svc.ListAlerts(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, alerts, 3)
}

func TestDeviceAlertService_GetUnresolvedCount(t *testing.T) {
	db := setupAdminServiceTestDB(t)
	ctx := context.Background()

	merchant := &models.Merchant{Name: "商户A", ContactName: "联系人", ContactPhone: "13900139000", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "场地A", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)
	device := &models.Device{
		DeviceNo:       "DEV_UNRESOLVED",
		Name:           "未解决告警设备",
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

	// 创建未解决的告警
	unresolvedAlert := &models.DeviceAlert{
		DeviceID:   device.ID,
		Type:       string(AlertTypeFault),
		Level:      string(AlertLevelCritical),
		Title:      "未解决告警",
		Content:    "告警消息",
		IsResolved: false,
	}
	require.NoError(t, db.Create(unresolvedAlert).Error)

	// 创建已解决的告警
	resolvedAlert := &models.DeviceAlert{
		DeviceID:   device.ID,
		Type:       string(AlertTypeFault),
		Level:      string(AlertLevelWarning),
		Title:      "已解决告警",
		Content:    "告警消息",
		IsResolved: true,
	}
	require.NoError(t, db.Create(resolvedAlert).Error)

	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	alertRepo := repository.NewDeviceAlertRepository(db)
	svc := NewDeviceAlertService(deviceRepo, deviceLogRepo, alertRepo)

	count, err := svc.GetUnresolvedCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestDeviceAlertService_GetAlertStatistics(t *testing.T) {
	db := setupAdminServiceTestDB(t)
	ctx := context.Background()

	merchant := &models.Merchant{Name: "商户A", ContactName: "联系人", ContactPhone: "13900139000", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "场地A", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)
	device := &models.Device{
		DeviceNo:       "DEV_STATS",
		Name:           "统计设备",
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

	// 创建不同类型的告警
	alerts := []models.DeviceAlert{
		{DeviceID: device.ID, Type: string(AlertTypeFault), Level: string(AlertLevelCritical), Title: "故障", Content: "消息"},
		{DeviceID: device.ID, Type: string(AlertTypeLowBattery), Level: string(AlertLevelWarning), Title: "低电量", Content: "消息"},
		{DeviceID: device.ID, Type: string(AlertTypeOffline), Level: string(AlertLevelCritical), Title: "离线", Content: "消息"},
	}
	for i := range alerts {
		require.NoError(t, db.Create(&alerts[i]).Error)
	}

	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	alertRepo := repository.NewDeviceAlertRepository(db)
	svc := NewDeviceAlertService(deviceRepo, deviceLogRepo, alertRepo)

	stats, err := svc.GetAlertStatistics(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

