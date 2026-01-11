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

	"github.com/dumeirei/smart-locker-backend/internal/common/crypto"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupMerchantAdminTestDB(t *testing.T) *gorm.DB {
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
	))
	return db
}

func TestMerchantAdminService_CRUD(t *testing.T) {
	db := setupMerchantAdminTestDB(t)
	merchantRepo := repository.NewMerchantRepository(db)
	aes, err := crypto.NewAES("1234567890abcdef")
	require.NoError(t, err)
	svc := NewMerchantAdminService(merchantRepo, aes)
	ctx := context.Background()

	bankAccount := "6222021234567890"
	bankHolder := "张三"
	req := &CreateMerchantRequest{
		Name:         "商户A",
		ContactName:  "联系人",
		ContactPhone: "13800138000",
		BankAccount:  &bankAccount,
		BankHolder:   &bankHolder,
		// CommissionRate/SettlementType 留空，走默认值
	}

	merchant, err := svc.CreateMerchant(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, merchant)
	assert.Equal(t, "商户A", merchant.Name)
	assert.Equal(t, 0.2, merchant.CommissionRate)
	assert.Equal(t, models.SettlementTypeMonthly, merchant.SettlementType)
	require.NotNil(t, merchant.BankAccountEncrypted)
	require.NotNil(t, merchant.BankHolderEncrypted)

	t.Run("GetMerchant 解密脱敏并返回统计", func(t *testing.T) {
		venue := &models.Venue{
			MerchantID: merchant.ID,
			Name:       "场地1",
			Type:       models.VenueTypeMall,
			Province:   "广东省",
			City:       "深圳市",
			District:   "南山区",
			Address:    "科技园",
			Status:     models.VenueStatusActive,
		}
		require.NoError(t, db.Create(venue).Error)

		device := &models.Device{
			DeviceNo:     "D001",
			Name:         "设备1",
			Type:         models.DeviceTypeStandard,
			VenueID:      venue.ID,
			QRCode:       "qr",
			ProductName:  "商品",
			OnlineStatus: models.DeviceOnline,
			Status:       models.DeviceStatusActive,
		}
		require.NoError(t, db.Create(device).Error)

		info, err := svc.GetMerchant(ctx, merchant.ID)
		require.NoError(t, err)
		require.NotNil(t, info)
		require.NotNil(t, info.BankAccount)
		require.NotNil(t, info.BankHolder)
		assert.Equal(t, "6222 **** **** 7890", *info.BankAccount)
		assert.Equal(t, "张*", *info.BankHolder)
		require.NotNil(t, info.Stats)
		assert.Equal(t, int64(1), info.Stats.VenueCount)
		assert.Equal(t, int64(1), info.Stats.DeviceCount)
		assert.Equal(t, int64(1), info.Stats.OnlineDeviceCount)
	})

	t.Run("UpdateMerchant 名称冲突返回错误", func(t *testing.T) {
		_, err := svc.CreateMerchant(ctx, &CreateMerchantRequest{
			Name:         "商户B",
			ContactName:  "联系人",
			ContactPhone: "13800138001",
		})
		require.NoError(t, err)

		err = svc.UpdateMerchant(ctx, merchant.ID, &UpdateMerchantRequest{
			Name:         "商户B",
			ContactName:  "联系人",
			ContactPhone: "13800138000",
			CommissionRate: 0.2,
			SettlementType: models.SettlementTypeMonthly,
		})
		require.Error(t, err)
		assert.Equal(t, ErrMerchantNameExists, err)
	})

	t.Run("DeleteMerchant 商户下有场地不允许删除", func(t *testing.T) {
		err := svc.DeleteMerchant(ctx, merchant.ID)
		require.Error(t, err)
		assert.Equal(t, ErrMerchantHasVenues, err)
	})
}

func TestMerchantAdminService_AdditionalOperations(t *testing.T) {
	db := setupMerchantAdminTestDB(t)
	merchantRepo := repository.NewMerchantRepository(db)
	aes, err := crypto.NewAES("1234567890abcdef")
	require.NoError(t, err)
	svc := NewMerchantAdminService(merchantRepo, aes)
	ctx := context.Background()

	// 创建商户
	merchant, _ := svc.CreateMerchant(ctx, &CreateMerchantRequest{
		Name:         "状态测试商户",
		ContactName:  "联系人",
		ContactPhone: "13800138002",
	})

	t.Run("UpdateMerchantStatus 更新商户状态", func(t *testing.T) {
		err := svc.UpdateMerchantStatus(ctx, merchant.ID, models.MerchantStatusDisabled)
		require.NoError(t, err)

		var updated models.Merchant
		db.First(&updated, merchant.ID)
		assert.Equal(t, int8(models.MerchantStatusDisabled), updated.Status)
	})

	t.Run("ListMerchants 获取商户列表", func(t *testing.T) {
		list, total, err := svc.ListMerchants(ctx, 0, 10, nil)
		require.NoError(t, err)
		assert.True(t, total >= 1)
		assert.NotEmpty(t, list)
	})

	t.Run("ListAllMerchants 获取所有商户", func(t *testing.T) {
		list, err := svc.ListAllMerchants(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})
}

