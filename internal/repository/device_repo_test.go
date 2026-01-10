// Package repository 设备仓储单元测试
package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// setupDeviceTestDB 创建设备测试数据库
func setupDeviceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Device{},
		&models.DeviceLog{},
		&models.Venue{},
		&models.Merchant{},
		&models.RentalPricing{},
	)
	require.NoError(t, err)

	return db
}

func createDeviceTestVenue(t *testing.T, db *gorm.DB) *models.Venue {
	t.Helper()

	merchant := &models.Merchant{
		Name:           "测试商户",
		CommissionRate: 0.1,
		Status:         1,
	}
	require.NoError(t, db.Create(merchant).Error)

	venue := &models.Venue{
		MerchantID: merchant.ID,
		Name:       "测试场地",
		Status:     1,
	}
	require.NoError(t, db.Create(venue).Error)

	return venue
}

func createTestDeviceForRepo(t *testing.T, db *gorm.DB, venueID int64, deviceNo string) *models.Device {
	t.Helper()

	device := &models.Device{
		DeviceNo:       deviceNo,
		Name:           "测试设备",
		VenueID:        venueID,
		Type:           "locker",
		ProductName:    "测试产品",
		QRCode:         fmt.Sprintf("QR_%s", deviceNo),
		Status:         models.DeviceStatusActive,
		OnlineStatus:   models.DeviceOnline,
		RentalStatus:   models.DeviceRentalFree,
		SlotCount:      10,
		AvailableSlots: 10,
	}
	require.NoError(t, db.Create(device).Error)
	return device
}

func TestDeviceRepository_Create(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)

	device := &models.Device{
		DeviceNo:       "DEV001",
		Name:           "测试设备",
		VenueID:        venue.ID,
		Type:           "locker",
		ProductName:    "测试产品",
		Status:         models.DeviceStatusActive,
		SlotCount:      10,
		AvailableSlots: 10,
	}

	err := repo.Create(ctx, device)
	require.NoError(t, err)
	assert.NotZero(t, device.ID)

	// 验证设备已创建
	var found models.Device
	db.First(&found, device.ID)
	assert.Equal(t, "DEV001", found.DeviceNo)
}

func TestDeviceRepository_GetByID(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV002")

	t.Run("获取存在的设备", func(t *testing.T) {
		found, err := repo.GetByID(ctx, device.ID)
		require.NoError(t, err)
		assert.Equal(t, device.ID, found.ID)
		assert.Equal(t, device.DeviceNo, found.DeviceNo)
	})

	t.Run("获取不存在的设备", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestDeviceRepository_GetByIDWithVenue(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV003")

	found, err := repo.GetByIDWithVenue(ctx, device.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.Venue)
	assert.Equal(t, venue.Name, found.Venue.Name)
}

func TestDeviceRepository_GetByDeviceNo(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV004")

	t.Run("根据设备编号获取设备", func(t *testing.T) {
		found, err := repo.GetByDeviceNo(ctx, device.DeviceNo)
		require.NoError(t, err)
		assert.Equal(t, device.ID, found.ID)
	})

	t.Run("获取不存在的设备编号", func(t *testing.T) {
		_, err := repo.GetByDeviceNo(ctx, "INVALID_NO")
		assert.Error(t, err)
	})
}

func TestDeviceRepository_GetByQRCode(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV005")

	t.Run("根据二维码获取设备", func(t *testing.T) {
		found, err := repo.GetByQRCode(ctx, device.QRCode)
		require.NoError(t, err)
		assert.Equal(t, device.ID, found.ID)
	})

	t.Run("获取不存在的二维码", func(t *testing.T) {
		_, err := repo.GetByQRCode(ctx, "INVALID_QR")
		assert.Error(t, err)
	})
}

func TestDeviceRepository_Update(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV006")

	device.DeviceNo = "DEV006_UPDATED"
	err := repo.Update(ctx, device)
	require.NoError(t, err)

	var found models.Device
	db.First(&found, device.ID)
	assert.Equal(t, "DEV006_UPDATED", found.DeviceNo)
}

func TestDeviceRepository_UpdateFields(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV007")

	err := repo.UpdateFields(ctx, device.ID, map[string]interface{}{
		"device_no": "DEV007_NEW",
	})
	require.NoError(t, err)

	var found models.Device
	db.First(&found, device.ID)
	assert.Equal(t, "DEV007_NEW", found.DeviceNo)
}

func TestDeviceRepository_UpdateStatus(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV008")

	err := repo.UpdateStatus(ctx, device.ID, models.DeviceStatusMaintenance)
	require.NoError(t, err)

	var found models.Device
	db.First(&found, device.ID)
	assert.Equal(t, int8(models.DeviceStatusMaintenance), found.Status)
}

func TestDeviceRepository_UpdateOnlineStatus(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV009")

	err := repo.UpdateOnlineStatus(ctx, device.ID, models.DeviceOffline)
	require.NoError(t, err)

	var found models.Device
	db.First(&found, device.ID)
	assert.Equal(t, int8(models.DeviceOffline), found.OnlineStatus)
}

func TestDeviceRepository_UpdateRentalStatus(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV010")

	rentalID := int64(123)
	err := repo.UpdateRentalStatus(ctx, device.ID, models.DeviceRentalInUse, &rentalID)
	require.NoError(t, err)

	var found models.Device
	db.First(&found, device.ID)
	assert.Equal(t, int8(models.DeviceRentalInUse), found.RentalStatus)
	assert.Equal(t, &rentalID, found.CurrentRentalID)
}

func TestDeviceRepository_List(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)

	// 创建多个设备
	for i := 0; i < 5; i++ {
		createTestDeviceForRepo(t, db, venue.ID, fmt.Sprintf("DEV_LIST_%d", i))
	}

	t.Run("获取设备列表", func(t *testing.T) {
		devices, total, err := repo.List(ctx, 0, 10, nil)
		require.NoError(t, err)
		assert.True(t, total >= 5)
		assert.True(t, len(devices) >= 5)
	})

	t.Run("分页获取", func(t *testing.T) {
		devices, total, err := repo.List(ctx, 0, 2, nil)
		require.NoError(t, err)
		assert.True(t, total >= 5)
		assert.Len(t, devices, 2)
	})

	t.Run("按场地筛选", func(t *testing.T) {
		devices, total, err := repo.List(ctx, 0, 10, map[string]interface{}{
			"venue_id": venue.ID,
		})
		require.NoError(t, err)
		assert.True(t, total >= 5)
		for _, d := range devices {
			assert.Equal(t, venue.ID, d.VenueID)
		}
	})

	t.Run("按状态筛选", func(t *testing.T) {
		devices, _, err := repo.List(ctx, 0, 10, map[string]interface{}{
			"status": int8(models.DeviceStatusActive),
		})
		require.NoError(t, err)
		for _, d := range devices {
			assert.Equal(t, int8(models.DeviceStatusActive), d.Status)
		}
	})
}

func TestDeviceRepository_ListByVenue(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	createTestDeviceForRepo(t, db, venue.ID, "DEV_VENUE_1")
	createTestDeviceForRepo(t, db, venue.ID, "DEV_VENUE_2")

	devices, err := repo.ListByVenue(ctx, venue.ID, nil)
	require.NoError(t, err)
	assert.True(t, len(devices) >= 2)
	for _, d := range devices {
		assert.Equal(t, venue.ID, d.VenueID)
	}
}

func TestDeviceRepository_ListAvailable(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)

	// 创建可租借设备
	availableDevice := createTestDeviceForRepo(t, db, venue.ID, "DEV_AVAIL_1")

	// 创建不可租借设备（已被租借）
	busyDevice := createTestDeviceForRepo(t, db, venue.ID, "DEV_BUSY_1")
	db.Model(&busyDevice).Update("rental_status", models.DeviceRentalInUse)

	devices, err := repo.ListAvailable(ctx, venue.ID)
	require.NoError(t, err)

	// 验证只返回可租借设备
	foundAvailable := false
	foundBusy := false
	for _, d := range devices {
		if d.ID == availableDevice.ID {
			foundAvailable = true
		}
		if d.ID == busyDevice.ID {
			foundBusy = true
		}
	}
	assert.True(t, foundAvailable)
	assert.False(t, foundBusy)
}

func TestDeviceRepository_ExistsByDeviceNo(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV_EXISTS")

	t.Run("设备编号存在", func(t *testing.T) {
		exists, err := repo.ExistsByDeviceNo(ctx, device.DeviceNo)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("设备编号不存在", func(t *testing.T) {
		exists, err := repo.ExistsByDeviceNo(ctx, "NONEXISTENT")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestDeviceRepository_CreateLog(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV_LOG")

	content := "状态变更"
	log := &models.DeviceLog{
		DeviceID: device.ID,
		Type:     "status_change",
		Content:  &content,
	}

	err := repo.CreateLog(ctx, log)
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
}

func TestDeviceRepository_ListLogs(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV_LOGS")

	// 创建多个日志
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("日志%d", i)
		log := &models.DeviceLog{
			DeviceID: device.ID,
			Type:     "status_change",
			Content:  &content,
		}
		db.Create(log)
	}

	logs, total, err := repo.ListLogs(ctx, device.ID, 0, 10, "")
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 5)
}

func TestDeviceRepository_IncrementDecrementSlots(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV_SLOTS")

	t.Run("减少可用槽位", func(t *testing.T) {
		err := repo.DecrementAvailableSlots(ctx, device.ID)
		require.NoError(t, err)

		var found models.Device
		db.First(&found, device.ID)
		assert.Equal(t, 9, found.AvailableSlots)
	})

	t.Run("增加可用槽位", func(t *testing.T) {
		err := repo.IncrementAvailableSlots(ctx, device.ID)
		require.NoError(t, err)

		var found models.Device
		db.First(&found, device.ID)
		assert.Equal(t, 10, found.AvailableSlots)
	})

	t.Run("槽位为0时减少失败", func(t *testing.T) {
		// 将槽位设置为0
		db.Model(&device).Update("available_slots", 0)

		err := repo.DecrementAvailableSlots(ctx, device.ID)
		assert.Error(t, err)
	})
}

func TestDeviceRepository_UpdateQRCode(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV_QR")

	newQRCode := "NEW_QR_CODE"
	err := repo.UpdateQRCode(ctx, device.ID, newQRCode)
	require.NoError(t, err)

	var found models.Device
	db.First(&found, device.ID)
	assert.Equal(t, newQRCode, found.QRCode)
}

func TestDeviceRepository_GetPricingsByDevice(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV_PRICING")

	venueIDPtr := &venue.ID
	// 创建定价
	pricing1 := &models.RentalPricing{
		VenueID:       venueIDPtr,
		DurationHours: 1,
		Price:         10.0,
		Deposit:       50.0,
		OvertimeRate:  5.0,
		IsActive:      true,
	}
	db.Create(pricing1)

	pricing2 := &models.RentalPricing{
		VenueID:       venueIDPtr,
		DurationHours: 2,
		Price:         18.0,
		Deposit:       50.0,
		OvertimeRate:  5.0,
		IsActive:      true,
	}
	db.Create(pricing2)

	pricings, err := repo.GetPricingsByDevice(ctx, device.ID)
	require.NoError(t, err)
	assert.Len(t, pricings, 2)
	// 验证按时长排序
	assert.Equal(t, 1, pricings[0].DurationHours)
	assert.Equal(t, 2, pricings[1].DurationHours)
}

func TestDeviceRepository_GetDefaultPricing(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV_DEFAULT_PRICING")

	venueIDPtr := &venue.ID
	// 创建定价
	pricing := &models.RentalPricing{
		VenueID:       venueIDPtr,
		DurationHours: 1,
		Price:         10.0,
		Deposit:       50.0,
		OvertimeRate:  5.0,
		IsActive:      true,
	}
	db.Create(pricing)

	defaultPricing, err := repo.GetDefaultPricing(ctx, device.ID)
	require.NoError(t, err)
	assert.Equal(t, pricing.ID, defaultPricing.ID)
}

func TestDeviceRepository_UpdateHeartbeat(t *testing.T) {
	db := setupDeviceTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	venue := createDeviceTestVenue(t, db)
	device := createTestDeviceForRepo(t, db, venue.ID, "DEV_HEARTBEAT")

	now := time.Now()
	err := repo.UpdateHeartbeat(ctx, device.ID, map[string]interface{}{
		"last_heartbeat_at": now,
		"online_status":     models.DeviceOnline,
	})
	require.NoError(t, err)

	var found models.Device
	db.First(&found, device.ID)
	assert.Equal(t, int8(models.DeviceOnline), found.OnlineStatus)
}
