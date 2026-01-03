package admin

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

func setupAdminServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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
		&models.DeviceAlert{},
	))

	return db
}

func TestDeviceQRCodeService_GenerateQRCode_UpdatesDevice(t *testing.T) {
	db := setupAdminServiceTestDB(t)
	ctx := context.Background()

	merchant := &models.Merchant{Name: "商户A", ContactName: "联系人", ContactPhone: "13900139000", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "场地A", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)

	device := &models.Device{
		DeviceNo:       "DEV_QR_001",
		Name:           "二维码设备",
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
	svc := NewDeviceQRCodeService(deviceRepo, "https://example.com")

	info, err := svc.GenerateQRCode(ctx, device.ID)
	require.NoError(t, err)

	assert.Equal(t, device.ID, info.DeviceID)
	assert.Equal(t, device.DeviceNo, info.DeviceNo)
	assert.Equal(t, device.Name, info.DeviceName)
	assert.Equal(t, "https://example.com/scan/"+device.DeviceNo, info.QRCodeURL)
	assert.True(t, strings.HasPrefix(info.DataURL, "data:image/png;base64,"))

	var updated models.Device
	require.NoError(t, db.First(&updated, device.ID).Error)
	assert.Equal(t, info.QRCodeURL, updated.QRCode)
}

func TestDeviceQRCodeService_GetQRCodeDataURL_UsesExistingQRCode(t *testing.T) {
	db := setupAdminServiceTestDB(t)
	ctx := context.Background()

	device := &models.Device{
		DeviceNo:       "DEV_QR_002",
		Name:           "二维码设备2",
		Type:           models.DeviceTypeStandard,
		VenueID:        1,
		QRCode:         "https://qr.example.com/DEV_QR_002",
		ProductName:    "测试产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	require.NoError(t, db.Create(&models.Merchant{Name: "M", ContactName: "C", ContactPhone: "139", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}).Error)
	require.NoError(t, db.Create(&models.Venue{MerchantID: 1, Name: "V", Type: "mall", Province: "P", City: "C", District: "D", Address: "A", Status: models.VenueStatusActive}).Error)
	require.NoError(t, db.Create(device).Error)

	deviceRepo := repository.NewDeviceRepository(db)
	svc := NewDeviceQRCodeService(deviceRepo, "https://example.com")

	dataURL, err := svc.GetQRCodeDataURL(ctx, device.ID)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(dataURL, "data:image/png;base64,"))
}

