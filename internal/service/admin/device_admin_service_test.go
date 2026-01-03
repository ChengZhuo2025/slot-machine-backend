// Package admin 设备管理服务单元测试
package admin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// MockMQTTService MQTT 服务 mock
type MockMQTTService struct {
	mock.Mock
}

func (m *MockMQTTService) SendUnlockCommand(ctx context.Context, deviceNo string, slotNo *int) (bool, error) {
	args := m.Called(ctx, deviceNo, slotNo)
	return args.Bool(0), args.Error(1)
}

func (m *MockMQTTService) SendLockCommand(ctx context.Context, deviceNo string, slotNo *int) (bool, error) {
	args := m.Called(ctx, deviceNo, slotNo)
	return args.Bool(0), args.Error(1)
}

// setupDeviceAdminTestDB 创建测试数据库
func setupDeviceAdminTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Device{},
		&models.DeviceLog{},
		&models.DeviceMaintenance{},
		&models.Venue{},
		&models.Merchant{},
	)
	require.NoError(t, err)

	return db
}

// createTestVenue 创建测试场地
func createTestVenue(t *testing.T, db *gorm.DB) *models.Venue {
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

// createTestDevice 创建测试设备
func createTestDevice(t *testing.T, db *gorm.DB, deviceNo string, venue *models.Venue) *models.Device {
	device := &models.Device{
		DeviceNo:       deviceNo,
		Name:           "测试设备",
		Type:           "standard",
		VenueID:        venue.ID,
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

// setupDeviceAdminService 创建测试用的 DeviceAdminService
func setupDeviceAdminService(t *testing.T) (*DeviceAdminService, *gorm.DB, *MockMQTTService) {
	db := setupDeviceAdminTestDB(t)
	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	deviceMaintenanceRepo := repository.NewDeviceMaintenanceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	mockMQTT := new(MockMQTTService)

	service := NewDeviceAdminService(deviceRepo, deviceLogRepo, deviceMaintenanceRepo, venueRepo, nil)

	return service, db, mockMQTT
}

func TestDeviceAdminService_CreateDevice_Success(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地
	venue := createTestVenue(t, db)

	// 创建设备
	req := &CreateDeviceRequest{
		DeviceNo:    "DEV001",
		Name:        "新设备",
		Type:        "standard",
		VenueID:     venue.ID,
		ProductName: "测试产品",
		SlotCount:   5,
		NetworkType: "WiFi",
	}

	device, err := service.CreateDevice(ctx, req, 1)

	require.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, "DEV001", device.DeviceNo)
	assert.Equal(t, "新设备", device.Name)
	assert.Equal(t, venue.ID, device.VenueID)
	assert.Equal(t, 5, device.SlotCount)
	assert.Equal(t, 5, device.AvailableSlots)
}

func TestDeviceAdminService_CreateDevice_DuplicateDeviceNo(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地
	venue := createTestVenue(t, db)

	// 创建第一个设备
	createTestDevice(t, db, "DEV_DUP", venue)

	// 尝试创建重复设备编号
	req := &CreateDeviceRequest{
		DeviceNo:    "DEV_DUP",
		Name:        "重复设备",
		Type:        "standard",
		VenueID:     venue.ID,
		ProductName: "测试产品",
		SlotCount:   5,
		NetworkType: "WiFi",
	}

	_, err := service.CreateDevice(ctx, req, 1)

	assert.Error(t, err)
	assert.Equal(t, ErrDeviceNoExists, err)
}

func TestDeviceAdminService_CreateDevice_VenueNotFound(t *testing.T) {
	service, _, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	req := &CreateDeviceRequest{
		DeviceNo:    "DEV002",
		Name:        "新设备",
		Type:        "standard",
		VenueID:     99999, // 不存在的场地
		ProductName: "测试产品",
		SlotCount:   5,
		NetworkType: "WiFi",
	}

	_, err := service.CreateDevice(ctx, req, 1)

	assert.Error(t, err)
	assert.Equal(t, ErrVenueNotFound, err)
}

func TestDeviceAdminService_UpdateDevice_Success(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_UPDATE", venue)

	// 更新设备
	req := &UpdateDeviceRequest{
		Name:        "更新后的设备",
		Type:        "premium",
		VenueID:     venue.ID,
		ProductName: "更新后的产品",
		SlotCount:   20,
		NetworkType: "4G",
	}

	err := service.UpdateDevice(ctx, device.ID, req)

	require.NoError(t, err)

	// 验证更新
	var updatedDevice models.Device
	err = db.First(&updatedDevice, device.ID).Error
	require.NoError(t, err)

	assert.Equal(t, "更新后的设备", updatedDevice.Name)
	assert.Equal(t, "premium", updatedDevice.Type)
	assert.Equal(t, 20, updatedDevice.SlotCount)
	assert.Equal(t, "4G", updatedDevice.NetworkType)
}

func TestDeviceAdminService_UpdateDevice_NotFound(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	venue := createTestVenue(t, db)

	req := &UpdateDeviceRequest{
		Name:        "更新后的设备",
		Type:        "premium",
		VenueID:     venue.ID,
		ProductName: "更新后的产品",
		SlotCount:   20,
		NetworkType: "4G",
	}

	err := service.UpdateDevice(ctx, 99999, req)

	assert.Error(t, err)
	assert.Equal(t, ErrDeviceNotFound, err)
}

func TestDeviceAdminService_UpdateDeviceStatus_Success(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_STATUS", venue)

	// 更新状态为维护中
	err := service.UpdateDeviceStatus(ctx, device.ID, models.DeviceStatusMaintenance, 1)

	require.NoError(t, err)

	// 验证状态已更新
	var updatedDevice models.Device
	err = db.First(&updatedDevice, device.ID).Error
	require.NoError(t, err)

	assert.Equal(t, int8(models.DeviceStatusMaintenance), updatedDevice.Status)
}

func TestDeviceAdminService_UpdateDeviceStatus_DeviceInUse(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_INUSE", venue)

	// 设置设备为使用中
	err := db.Model(device).Update("rental_status", models.DeviceRentalInUse).Error
	require.NoError(t, err)

	// 尝试禁用设备
	err = service.UpdateDeviceStatus(ctx, device.ID, models.DeviceStatusDisabled, 1)

	assert.Error(t, err)
	assert.Equal(t, ErrDeviceInUse, err)
}

func TestDeviceAdminService_GetDevice_Success(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_GET", venue)

	// 获取设备
	info, err := service.GetDevice(ctx, device.ID)

	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, device.ID, info.ID)
	assert.Equal(t, "DEV_GET", info.DeviceNo)
}

func TestDeviceAdminService_GetDevice_NotFound(t *testing.T) {
	service, _, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	_, err := service.GetDevice(ctx, 99999)

	assert.Error(t, err)
	assert.Equal(t, ErrDeviceNotFound, err)
}

func TestDeviceAdminService_ListDevices(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和多个设备
	venue := createTestVenue(t, db)
	createTestDevice(t, db, "DEV_LIST1", venue)
	createTestDevice(t, db, "DEV_LIST2", venue)
	createTestDevice(t, db, "DEV_LIST3", venue)

	// 获取列表
	devices, total, err := service.ListDevices(ctx, 0, 10, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, devices, 3)
}

func TestDeviceAdminService_ListDevices_WithFilters(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和多个设备
	venue := createTestVenue(t, db)
	createTestDevice(t, db, "DEV_FILTER1", venue)
	device2 := createTestDevice(t, db, "DEV_FILTER2", venue)

	// 设置一个设备离线
	err := db.Model(device2).Update("online_status", models.DeviceOffline).Error
	require.NoError(t, err)

	// 按在线状态过滤
	filters := map[string]interface{}{
		"online_status": int8(models.DeviceOnline),
	}
	devices, total, err := service.ListDevices(ctx, 0, 10, filters)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, devices, 1)
	assert.Equal(t, "DEV_FILTER1", devices[0].DeviceNo)
}

func TestDeviceAdminService_CreateMaintenance_Success(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_MAINT", venue)

	// 创建维护记录
	req := &CreateMaintenanceRequest{
		DeviceID:    device.ID,
		Type:        "repair",
		Description: "更换零件",
		Cost:        100.0,
	}

	maintenance, err := service.CreateMaintenance(ctx, req, 1)

	require.NoError(t, err)
	assert.NotNil(t, maintenance)
	assert.Equal(t, device.ID, maintenance.DeviceID)
	assert.Equal(t, "repair", maintenance.Type)
	assert.Equal(t, int8(models.MaintenanceStatusInProgress), maintenance.Status)

	// 验证设备状态已变为维护中
	var updatedDevice models.Device
	err = db.First(&updatedDevice, device.ID).Error
	require.NoError(t, err)
	assert.Equal(t, int8(models.DeviceStatusMaintenance), updatedDevice.Status)
}

func TestDeviceAdminService_CreateMaintenance_DeviceInUse(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_MAINT_INUSE", venue)

	// 设置设备为使用中
	err := db.Model(device).Update("rental_status", models.DeviceRentalInUse).Error
	require.NoError(t, err)

	// 尝试创建维护记录
	req := &CreateMaintenanceRequest{
		DeviceID:    device.ID,
		Type:        "repair",
		Description: "更换零件",
	}

	_, err = service.CreateMaintenance(ctx, req, 1)

	assert.Error(t, err)
	assert.Equal(t, ErrDeviceInUse, err)
}

func TestDeviceAdminService_CompleteMaintenance_Success(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_MAINT_COMPLETE", venue)

	// 创建维护记录
	createReq := &CreateMaintenanceRequest{
		DeviceID:    device.ID,
		Type:        "repair",
		Description: "更换零件",
		Cost:        100.0,
	}
	maintenance, err := service.CreateMaintenance(ctx, createReq, 1)
	require.NoError(t, err)

	// 完成维护
	completeReq := &CompleteMaintenanceRequest{
		Cost: 150.0,
	}
	err = service.CompleteMaintenance(ctx, maintenance.ID, completeReq, 1)

	require.NoError(t, err)

	// 验证维护记录已完成
	var updatedMaintenance models.DeviceMaintenance
	err = db.First(&updatedMaintenance, maintenance.ID).Error
	require.NoError(t, err)
	assert.Equal(t, int8(models.MaintenanceStatusCompleted), updatedMaintenance.Status)
	assert.NotNil(t, updatedMaintenance.CompletedAt)
	assert.Equal(t, 150.0, updatedMaintenance.Cost)

	// 验证设备状态已恢复正常
	var updatedDevice models.Device
	err = db.First(&updatedDevice, device.ID).Error
	require.NoError(t, err)
	assert.Equal(t, int8(models.DeviceStatusActive), updatedDevice.Status)
}

func TestDeviceAdminService_CompleteMaintenance_AlreadyCompleted(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_MAINT_DONE", venue)

	// 创建并完成维护记录
	createReq := &CreateMaintenanceRequest{
		DeviceID:    device.ID,
		Type:        "repair",
		Description: "更换零件",
	}
	maintenance, err := service.CreateMaintenance(ctx, createReq, 1)
	require.NoError(t, err)

	err = service.CompleteMaintenance(ctx, maintenance.ID, &CompleteMaintenanceRequest{}, 1)
	require.NoError(t, err)

	// 再次尝试完成
	err = service.CompleteMaintenance(ctx, maintenance.ID, &CompleteMaintenanceRequest{}, 1)

	assert.Error(t, err)
	assert.Equal(t, ErrMaintenanceCompleted, err)
}

func TestDeviceAdminService_GetDeviceLogs(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地和设备
	venue := createTestVenue(t, db)
	device := createTestDevice(t, db, "DEV_LOGS", venue)

	// 创建日志记录
	for i := 0; i < 3; i++ {
		content := "测试日志"
		log := &models.DeviceLog{
			DeviceID: device.ID,
			Type:     models.DeviceLogTypeOnline,
			Content:  &content,
		}
		err := db.Create(log).Error
		require.NoError(t, err)
	}

	// 获取日志
	logs, total, err := service.GetDeviceLogs(ctx, device.ID, 0, 10, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 3)
}

func TestDeviceAdminService_GetDeviceStatistics(t *testing.T) {
	service, db, _ := setupDeviceAdminService(t)
	ctx := context.Background()

	// 创建场地
	venue := createTestVenue(t, db)

	// 创建不同状态的设备
	device1 := createTestDevice(t, db, "DEV_STAT1", venue) // 在线、空闲、正常
	device2 := createTestDevice(t, db, "DEV_STAT2", venue)
	device3 := createTestDevice(t, db, "DEV_STAT3", venue)

	// 设置不同状态
	db.Model(device2).Updates(map[string]interface{}{
		"online_status": models.DeviceOffline,
		"status":        models.DeviceStatusMaintenance,
	})
	db.Model(device3).Updates(map[string]interface{}{
		"rental_status": models.DeviceRentalInUse,
	})
	_ = device1

	// 获取统计
	stats, err := service.GetDeviceStatistics(ctx, nil)

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(3), stats.Total)
	assert.Equal(t, int64(2), stats.Online)
	assert.Equal(t, int64(1), stats.Offline)
	assert.Equal(t, int64(1), stats.InUse)
	assert.Equal(t, int64(2), stats.Free)
	assert.Equal(t, int64(1), stats.Maintenance)
}

func TestDeviceAdminService_toDeviceInfo(t *testing.T) {
	service, _, _ := setupDeviceAdminService(t)

	now := time.Now()
	firmwareVersion := "v1.0.0"
	signalStrength := 85
	batteryLevel := 90

	device := &models.Device{
		ID:              1,
		DeviceNo:        "DEV001",
		Name:            "测试设备",
		Type:            "standard",
		VenueID:         1,
		ProductName:     "测试产品",
		SlotCount:       10,
		AvailableSlots:  8,
		OnlineStatus:    models.DeviceOnline,
		LockStatus:      models.DeviceLocked,
		RentalStatus:    models.DeviceRentalFree,
		FirmwareVersion: &firmwareVersion,
		NetworkType:     "WiFi",
		SignalStrength:  &signalStrength,
		BatteryLevel:    &batteryLevel,
		LastHeartbeatAt: &now,
		Status:          models.DeviceStatusActive,
		CreatedAt:       now,
		Venue: &models.Venue{
			Name: "测试场地",
		},
	}

	info := service.toDeviceInfo(device)

	assert.Equal(t, int64(1), info.ID)
	assert.Equal(t, "DEV001", info.DeviceNo)
	assert.Equal(t, "测试设备", info.Name)
	assert.Equal(t, "测试场地", info.VenueName)
	assert.Equal(t, 10, info.SlotCount)
	assert.Equal(t, 8, info.AvailableSlots)
	assert.Equal(t, &firmwareVersion, info.FirmwareVersion)
	assert.Equal(t, &signalStrength, info.SignalStrength)
	assert.Equal(t, &batteryLevel, info.BatteryLevel)
}
