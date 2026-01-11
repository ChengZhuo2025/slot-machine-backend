// Package repository 商户仓储单元测试
package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupMerchantTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Merchant{}, &models.Venue{}, &models.Device{})
	require.NoError(t, err)

	return db
}

func TestMerchantRepository_Create(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name:         "测试商户",
		ContactName:  "张三",
		ContactPhone: "13800138000",
	}

	err := repo.Create(ctx, merchant)
	require.NoError(t, err)
	assert.NotZero(t, merchant.ID)
}

func TestMerchantRepository_GetByID(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name:         "测试商户",
		ContactName:  "张三",
		ContactPhone: "13800138000",
	}
	db.Create(merchant)

	found, err := repo.GetByID(ctx, merchant.ID)
	require.NoError(t, err)
	assert.Equal(t, merchant.ID, found.ID)
	assert.Equal(t, "测试商户", found.Name)
	assert.Equal(t, "张三", found.ContactName)
}

func TestMerchantRepository_Update(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name:         "原商户名",
		ContactName:  "张三",
		ContactPhone: "13800138000",
	}
	db.Create(merchant)

	merchant.Name = "新商户名"
	merchant.ContactPhone = "13900139000"
	err := repo.Update(ctx, merchant)
	require.NoError(t, err)

	var found models.Merchant
	db.First(&found, merchant.ID)
	assert.Equal(t, "新商户名", found.Name)
	assert.Equal(t, "13900139000", found.ContactPhone)
}

func TestMerchantRepository_UpdateFields(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name:         "测试商户",
		ContactName:  "张三",
		ContactPhone: "13800138000",
	}
	db.Create(merchant)

	err := repo.UpdateFields(ctx, merchant.ID, map[string]interface{}{
		"contact_phone": "13900139000",
		"contact_name":  "李四",
	})
	require.NoError(t, err)

	var found models.Merchant
	db.First(&found, merchant.ID)
	assert.Equal(t, "13900139000", found.ContactPhone)
	assert.Equal(t, "李四", found.ContactName)
}

func TestMerchantRepository_UpdateStatus(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name:         "测试商户",
		ContactName:  "张三",
		ContactPhone: "13800138000",
		Status:       models.MerchantStatusActive,
	}
	db.Create(merchant)

	err := repo.UpdateStatus(ctx, merchant.ID, models.MerchantStatusDisabled)
	require.NoError(t, err)

	var found models.Merchant
	db.First(&found, merchant.ID)
	assert.Equal(t, int8(models.MerchantStatusDisabled), found.Status)
}

func TestMerchantRepository_Delete(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name:         "待删除商户",
		ContactName:  "张三",
		ContactPhone: "13800138000",
	}
	db.Create(merchant)

	err := repo.Delete(ctx, merchant.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Count(&count)
	assert.Equal(t, int64(0), count) // 软删除
}

func TestMerchantRepository_List(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchants := []*models.Merchant{
		{Name: "深圳商户", ContactName: "张三", ContactPhone: "13800138000", Status: models.MerchantStatusActive},
		{Name: "广州商户", ContactName: "李四", ContactPhone: "13900139000", Status: models.MerchantStatusActive},
	}
	for _, m := range merchants {
		db.Create(m)
	}

	db.Model(&models.Merchant{}).Create(map[string]interface{}{
		"name":          "禁用商户",
		"contact_name":  "王五",
		"contact_phone": "13700137000",
		"status":        models.MerchantStatusDisabled,
	})

	// 测试获取所有商户
	list, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 3, len(list))

	// 按名称过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"name": "深圳",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))

	// 按联系人过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"contact_name": "张三",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按电话过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"contact_phone": "13800138000",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按状态过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": int8(models.MerchantStatusActive),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestMerchantRepository_ListAll(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	db.Create(&models.Merchant{
		Name: "活跃商户", ContactName: "张三", ContactPhone: "13800138000", Status: models.MerchantStatusActive,
	})

	db.Model(&models.Merchant{}).Create(map[string]interface{}{
		"name": "禁用商户", "contact_name": "李四", "contact_phone": "13900139000", "status": models.MerchantStatusDisabled,
	})

	list, err := repo.ListAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list)) // 只返回活跃商户
	assert.Equal(t, "活跃商户", list[0].Name)
}

func TestMerchantRepository_ExistsByName(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	db.Create(&models.Merchant{
		Name: "已存在商户", ContactName: "张三", ContactPhone: "13800138000",
	})

	exists, err := repo.ExistsByName(ctx, "已存在商户")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByName(ctx, "不存在商户")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMerchantRepository_ExistsByNameExcludeID(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name: "已存在商户", ContactName: "张三", ContactPhone: "13800138000",
	}
	db.Create(merchant)

	// 排除自己，不存在同名
	exists, err := repo.ExistsByNameExcludeID(ctx, "已存在商户", merchant.ID)
	require.NoError(t, err)
	assert.False(t, exists)

	// 创建另一个商户
	other := &models.Merchant{
		Name: "另一个商户", ContactName: "李四", ContactPhone: "13900139000",
	}
	db.Create(other)

	// 检查是否有其他商户同名
	exists, err = repo.ExistsByNameExcludeID(ctx, "已存在商户", other.ID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMerchantRepository_CountVenues(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name: "测试商户", ContactName: "张三", ContactPhone: "13800138000",
	}
	db.Create(merchant)

	// 创建场地
	for i := 0; i < 3; i++ {
		db.Create(&models.Venue{
			MerchantID: merchant.ID,
			Name:       "场地",
			Province:   "广东省",
			City:       "深圳市",
			District:   "南山区",
			Address:    "地址",
		})
	}

	count, err := repo.CountVenues(ctx, merchant.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestMerchantRepository_CountDevices(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name: "测试商户", ContactName: "张三", ContactPhone: "13800138000",
	}
	db.Create(merchant)

	// 创建场地
	venue := &models.Venue{
		MerchantID: merchant.ID,
		Name:       "场地",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "地址",
	}
	db.Create(venue)

	// 创建设备
	for i := 0; i < 5; i++ {
		db.Create(&models.Device{
			VenueID:     venue.ID,
			DeviceNo:    fmt.Sprintf("DEV%03d", i+1),
			Name:        "设备",
			Type:        "智能柜",
			QRCode:      fmt.Sprintf("QR%03d", i+1),
			ProductName: "商品",
		})
	}

	count, err := repo.CountDevices(ctx, merchant.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestMerchantRepository_GetStatistics(t *testing.T) {
	db := setupMerchantTestDB(t)
	repo := NewMerchantRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name: "测试商户", ContactName: "张三", ContactPhone: "13800138000",
	}
	db.Create(merchant)

	// 创建场地
	venue := &models.Venue{
		MerchantID: merchant.ID,
		Name:       "场地",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "地址",
	}
	db.Create(venue)

	// 创建在线设备
	for i := 0; i < 3; i++ {
		db.Create(&models.Device{
			VenueID:      venue.ID,
			DeviceNo:     fmt.Sprintf("DEV%03d", i+1),
			Name:         "设备",
			Type:         "智能柜",
			QRCode:       fmt.Sprintf("QR%03d", i+1),
			ProductName:  "商品",
			OnlineStatus: models.DeviceOnline,
		})
	}

	// 创建离线设备
	db.Model(&models.Device{}).Create(map[string]interface{}{
		"venue_id":      venue.ID,
		"device_no":     "DEV099",
		"name":          "设备",
		"type":          "智能柜",
		"qr_code":       "QR099",
		"product_name":  "商品",
		"online_status": models.DeviceOffline,
	})

	stats, err := repo.GetStatistics(ctx, merchant.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.VenueCount)
	assert.Equal(t, int64(4), stats.DeviceCount)
	assert.Equal(t, int64(3), stats.OnlineDeviceCount)
}
