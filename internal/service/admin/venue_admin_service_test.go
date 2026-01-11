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

func setupVenueAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(&models.Merchant{}, &models.Venue{}, &models.Device{}))
	return db
}

func TestVenueAdminService_CRUD(t *testing.T) {
	db := setupVenueAdminTestDB(t)
	svc := NewVenueAdminService(
		repository.NewVenueRepository(db),
		repository.NewMerchantRepository(db),
		repository.NewDeviceRepository(db),
	)
	ctx := context.Background()

	t.Run("CreateVenue 商户不存在", func(t *testing.T) {
		_, err := svc.CreateVenue(ctx, &CreateVenueRequest{
			MerchantID: 99999,
			Name:       "场地",
			Type:       models.VenueTypeMall,
			Province:   "广东省",
			City:       "深圳市",
			District:   "南山区",
			Address:    "科技园",
		})
		require.Error(t, err)
		assert.Equal(t, ErrMerchantNotFound, err)
	})

	merchant := &models.Merchant{Name: "M1", ContactName: "C", ContactPhone: "138", CommissionRate: 0.2, SettlementType: models.SettlementTypeMonthly, Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)

	venue, err := svc.CreateVenue(ctx, &CreateVenueRequest{
		MerchantID: merchant.ID,
		Name:       "场地1",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
	})
	require.NoError(t, err)
	require.NotNil(t, venue)

	t.Run("DeleteVenue 场地下有设备不允许删除", func(t *testing.T) {
		device := &models.Device{
			DeviceNo:     "D1",
			Name:         "设备1",
			Type:         models.DeviceTypeStandard,
			VenueID:      venue.ID,
			QRCode:       "qr",
			ProductName:  "商品",
			OnlineStatus: models.DeviceOnline,
			Status:       models.DeviceStatusActive,
		}
		require.NoError(t, db.Create(device).Error)

		err := svc.DeleteVenue(ctx, venue.ID)
		require.Error(t, err)
		assert.Equal(t, ErrVenueHasDevices, err)
	})

	t.Run("UpdateVenue 场地不存在", func(t *testing.T) {
		err := svc.UpdateVenue(ctx, 99999, &UpdateVenueRequest{
			MerchantID: merchant.ID,
			Name:       "场地2",
			Type:       models.VenueTypeMall,
			Province:   "广东省",
			City:       "深圳市",
			District:   "南山区",
			Address:    "科技园",
		})
		require.Error(t, err)
		assert.Equal(t, ErrVenueNotFound, err)
	})
}

func TestVenueAdminService_AdditionalOperations(t *testing.T) {
	db := setupVenueAdminTestDB(t)
	svc := NewVenueAdminService(
		repository.NewVenueRepository(db),
		repository.NewMerchantRepository(db),
		repository.NewDeviceRepository(db),
	)
	ctx := context.Background()

	merchant := &models.Merchant{Name: "M2", ContactName: "C", ContactPhone: "138", CommissionRate: 0.2, SettlementType: models.SettlementTypeMonthly, Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)

	venue, _ := svc.CreateVenue(ctx, &CreateVenueRequest{
		MerchantID: merchant.ID,
		Name:       "测试场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
	})

	t.Run("UpdateVenueStatus 更新场地状态", func(t *testing.T) {
		err := svc.UpdateVenueStatus(ctx, venue.ID, models.VenueStatusDisabled)
		require.NoError(t, err)

		var updated models.Venue
		db.First(&updated, venue.ID)
		assert.Equal(t, int8(models.VenueStatusDisabled), updated.Status)
	})

	t.Run("GetVenue 获取场地详情", func(t *testing.T) {
		result, err := svc.GetVenue(ctx, venue.ID)
		require.NoError(t, err)
		assert.Equal(t, venue.ID, result.ID)
	})

	t.Run("GetVenue 场地不存在", func(t *testing.T) {
		_, err := svc.GetVenue(ctx, 99999)
		assert.Error(t, err)
	})

	t.Run("ListVenues 获取场地列表", func(t *testing.T) {
		list, total, err := svc.ListVenues(ctx, 0, 10, nil)
		require.NoError(t, err)
		assert.True(t, total >= 1)
		assert.NotEmpty(t, list)
	})

	t.Run("ListVenuesByMerchant 按商户筛选场地", func(t *testing.T) {
		list, err := svc.ListVenuesByMerchant(ctx, merchant.ID)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})
}

