// Package repository 场地仓储单元测试
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

func setupVenueTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Merchant{}, &models.Venue{}, &models.Device{})
	require.NoError(t, err)

	return db
}

func TestVenueRepository_Create(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1,
		Name:       "测试场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
	}

	err := repo.Create(ctx, venue)
	require.NoError(t, err)
	assert.NotZero(t, venue.ID)
}

func TestVenueRepository_GetByID(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1,
		Name:       "测试场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
	}
	db.Create(venue)

	found, err := repo.GetByID(ctx, venue.ID)
	require.NoError(t, err)
	assert.Equal(t, venue.ID, found.ID)
	assert.Equal(t, "测试场地", found.Name)
}

func TestVenueRepository_GetByIDWithDevices(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1,
		Name:       "测试场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
	}
	db.Create(venue)

	// 创建活跃设备
	db.Create(&models.Device{
		VenueID:     venue.ID,
		DeviceNo:    "DEV001",
		Name:        "设备1",
		Type:        "智能柜",
		QRCode:      "QR001",
		ProductName: "商品",
		Status:      models.DeviceStatusActive,
	})

	// 创建禁用设备
	db.Model(&models.Device{}).Create(map[string]interface{}{
		"venue_id":     venue.ID,
		"device_no":    "DEV002",
		"name":         "设备2",
		"type":         "智能柜",
		"qr_code":      "QR002",
		"product_name": "商品",
		"status":       models.DeviceStatusDisabled,
	})

	found, err := repo.GetByIDWithDevices(ctx, venue.ID)
	require.NoError(t, err)
	assert.Equal(t, venue.ID, found.ID)
	assert.Equal(t, 1, len(found.Devices)) // 只加载活跃设备
}

func TestVenueRepository_Update(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1,
		Name:       "原场地名",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "原地址",
	}
	db.Create(venue)

	venue.Name = "新场地名"
	venue.Address = "新地址"
	err := repo.Update(ctx, venue)
	require.NoError(t, err)

	var found models.Venue
	db.First(&found, venue.ID)
	assert.Equal(t, "新场地名", found.Name)
	assert.Equal(t, "新地址", found.Address)
}

func TestVenueRepository_UpdateFields(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1,
		Name:       "测试场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
	}
	db.Create(venue)

	err := repo.UpdateFields(ctx, venue.ID, map[string]interface{}{
		"name":    "更新后的场地",
		"address": "更新后的地址",
	})
	require.NoError(t, err)

	var found models.Venue
	db.First(&found, venue.ID)
	assert.Equal(t, "更新后的场地", found.Name)
	assert.Equal(t, "更新后的地址", found.Address)
}

func TestVenueRepository_UpdateStatus(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1,
		Name:       "测试场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
		Status:     models.VenueStatusActive,
	}
	db.Create(venue)

	err := repo.UpdateStatus(ctx, venue.ID, models.VenueStatusDisabled)
	require.NoError(t, err)

	var found models.Venue
	db.First(&found, venue.ID)
	assert.Equal(t, int8(models.VenueStatusDisabled), found.Status)
}

func TestVenueRepository_Delete(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1,
		Name:       "待删除场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园",
	}
	db.Create(venue)

	err := repo.Delete(ctx, venue.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Venue{}).Where("id = ?", venue.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestVenueRepository_List(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venues := []*models.Venue{
		{MerchantID: 1, Name: "深圳商场", Type: models.VenueTypeMall, Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园", Status: models.VenueStatusActive},
		{MerchantID: 1, Name: "广州酒店", Type: models.VenueTypeHotel, Province: "广东省", City: "广州市", District: "天河区", Address: "天河路", Status: models.VenueStatusActive},
		{MerchantID: 2, Name: "北京社区", Type: models.VenueTypeCommunity, Province: "北京市", City: "北京市", District: "朝阳区", Address: "朝阳路", Status: models.VenueStatusActive},
	}
	for _, v := range venues {
		db.Create(v)
	}

	db.Model(&models.Venue{}).Create(map[string]interface{}{
		"merchant_id": 1,
		"name":        "禁用场地",
		"type":        models.VenueTypeMall,
		"province":    "广东省",
		"city":        "深圳市",
		"district":    "福田区",
		"address":     "福华路",
		"status":      models.VenueStatusDisabled,
	})

	// 测试获取所有场地
	list, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Equal(t, 4, len(list))

	// 按商户过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"merchant_id": int64(1),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按名称过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"name": "深圳",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按类型过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"type": models.VenueTypeMall,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按城市过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"city": "深圳市",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按区域过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"district": "南山区",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按状态过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": int8(models.VenueStatusActive),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
}

func TestVenueRepository_ListByMerchant(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	db.Create(&models.Venue{
		MerchantID: 1, Name: "商户1场地1", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
		Status: models.VenueStatusActive,
	})

	db.Create(&models.Venue{
		MerchantID: 1, Name: "商户1场地2", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "福田区", Address: "福华路",
		Status: models.VenueStatusActive,
	})

	db.Model(&models.Venue{}).Create(map[string]interface{}{
		"merchant_id": 1,
		"name":        "商户1禁用场地",
		"type":        models.VenueTypeMall,
		"province":    "广东省",
		"city":        "深圳市",
		"district":    "宝安区",
		"address":     "宝安路",
		"status":      models.VenueStatusDisabled,
	})

	db.Create(&models.Venue{
		MerchantID: 2, Name: "商户2场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "广州市", District: "天河区", Address: "天河路",
		Status: models.VenueStatusActive,
	})

	// 获取商户1所有场地
	list, err := repo.ListByMerchant(ctx, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, len(list))

	// 获取商户1活跃场地
	activeStatus := int8(models.VenueStatusActive)
	list, err = repo.ListByMerchant(ctx, 1, &activeStatus)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
}

func TestVenueRepository_ListNearby(t *testing.T) {
	// Skip this test for SQLite as it doesn't support trigonometric functions
	// This test would work with PostgreSQL in production
	t.Skip("Skipping ListNearby test - SQLite doesn't support acos/sin/cos functions")

	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	// 深圳位置附近的场地
	lng1 := 114.0579
	lat1 := 22.5431
	db.Create(&models.Venue{
		MerchantID: 1, Name: "附近场地1", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
		Longitude: &lng1, Latitude: &lat1,
		Status: models.VenueStatusActive,
	})

	// 远距离场地
	lng2 := 113.2644
	lat2 := 23.1291
	db.Create(&models.Venue{
		MerchantID: 1, Name: "远距离场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "广州市", District: "天河区", Address: "天河路",
		Longitude: &lng2, Latitude: &lat2,
		Status: models.VenueStatusActive,
	})

	// 没有经纬度的场地
	db.Create(&models.Venue{
		MerchantID: 1, Name: "无经纬度场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "福田区", Address: "福华路",
		Status: models.VenueStatusActive,
	})

	// 搜索深圳附近10公里内的场地
	list, err := repo.ListNearby(ctx, lng1, lat1, 10, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list)) // 只有附近场地1
}

func TestVenueRepository_ListByCity(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	db.Create(&models.Venue{
		MerchantID: 1, Name: "深圳场地1", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
		Status: models.VenueStatusActive,
	})

	db.Create(&models.Venue{
		MerchantID: 1, Name: "深圳场地2", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "福田区", Address: "福华路",
		Status: models.VenueStatusActive,
	})

	db.Create(&models.Venue{
		MerchantID: 1, Name: "广州场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "广州市", District: "天河区", Address: "天河路",
		Status: models.VenueStatusActive,
	})

	db.Model(&models.Venue{}).Create(map[string]interface{}{
		"merchant_id": 1,
		"name":        "深圳禁用场地",
		"type":        models.VenueTypeMall,
		"province":    "广东省",
		"city":        "深圳市",
		"district":    "宝安区",
		"address":     "宝安路",
		"status":      models.VenueStatusDisabled,
	})

	list, total, err := repo.ListByCity(ctx, "深圳市", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total) // 只返回活跃场地
	assert.Equal(t, 2, len(list))
}

func TestVenueRepository_GetDeviceCount(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1, Name: "测试场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
	}
	db.Create(venue)

	// 创建活跃设备
	for i := 0; i < 3; i++ {
		db.Create(&models.Device{
			VenueID:     venue.ID,
			DeviceNo:    fmt.Sprintf("DEV%03d", i+1),
			Name:        "设备",
			Type:        "智能柜",
			QRCode:      fmt.Sprintf("QR%03d", i+1),
			ProductName: "商品",
			Status:      models.DeviceStatusActive,
		})
	}

	// 创建禁用设备
	db.Model(&models.Device{}).Create(map[string]interface{}{
		"venue_id":     venue.ID,
		"device_no":    "DEV099",
		"name":         "设备",
		"type":         "智能柜",
		"qr_code":      "QR099",
		"product_name": "商品",
		"status":       models.DeviceStatusDisabled,
	})

	count, err := repo.GetDeviceCount(ctx, venue.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count) // 只统计活跃设备
}

func TestVenueRepository_GetAvailableDeviceCount(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1, Name: "测试场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
	}
	db.Create(venue)

	// 创建可用设备（在线+空闲+有可用槽位）
	for i := 0; i < 2; i++ {
		db.Create(&models.Device{
			VenueID:        venue.ID,
			DeviceNo:       fmt.Sprintf("DEV%03d", i+1),
			Name:           "设备",
			Type:           "智能柜",
			QRCode:         fmt.Sprintf("QR%03d", i+1),
			ProductName:    "商品",
			Status:         models.DeviceStatusActive,
			OnlineStatus:   models.DeviceOnline,
			RentalStatus:   models.DeviceRentalFree,
			AvailableSlots: 5,
		})
	}

	// 创建离线设备
	db.Model(&models.Device{}).Create(map[string]interface{}{
		"venue_id":        venue.ID,
		"device_no":       "DEV098",
		"name":            "设备",
		"type":            "智能柜",
		"qr_code":         "QR098",
		"product_name":    "商品",
		"status":          models.DeviceStatusActive,
		"online_status":   models.DeviceOffline,
		"rental_status":   models.DeviceRentalFree,
		"available_slots": 5,
	})

	// 创建租用中设备
	db.Model(&models.Device{}).Create(map[string]interface{}{
		"venue_id":        venue.ID,
		"device_no":       "DEV099",
		"name":            "设备",
		"type":            "智能柜",
		"qr_code":         "QR099",
		"product_name":    "商品",
		"status":          models.DeviceStatusActive,
		"online_status":   models.DeviceOnline,
		"rental_status":   models.DeviceRentalInUse,
		"available_slots": 0,
	})

	count, err := repo.GetAvailableDeviceCount(ctx, venue.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count) // 只统计可用设备
}

func TestVenueRepository_GetCities(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	db.Create(&models.Venue{
		MerchantID: 1, Name: "深圳场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
		Status: models.VenueStatusActive,
	})

	db.Create(&models.Venue{
		MerchantID: 1, Name: "广州场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "广州市", District: "天河区", Address: "天河路",
		Status: models.VenueStatusActive,
	})

	db.Create(&models.Venue{
		MerchantID: 1, Name: "北京场地", Type: models.VenueTypeMall,
		Province: "北京市", City: "北京市", District: "朝阳区", Address: "朝阳路",
		Status: models.VenueStatusActive,
	})

	// 创建禁用场地，不应该被统计
	db.Model(&models.Venue{}).Create(map[string]interface{}{
		"merchant_id": 1,
		"name":        "上海场地",
		"type":        models.VenueTypeMall,
		"province":    "上海市",
		"city":        "上海市",
		"district":    "浦东新区",
		"address":     "陆家嘴",
		"status":      models.VenueStatusDisabled,
	})

	cities, err := repo.GetCities(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, len(cities))
	assert.Contains(t, cities, "深圳市")
	assert.Contains(t, cities, "广州市")
	assert.Contains(t, cities, "北京市")
}

func TestVenueRepository_ExistsByName(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	db.Create(&models.Venue{
		MerchantID: 1, Name: "已存在场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
	})

	exists, err := repo.ExistsByName(ctx, 1, "已存在场地")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByName(ctx, 1, "不存在场地")
	require.NoError(t, err)
	assert.False(t, exists)

	// 不同商户可以有同名场地
	exists, err = repo.ExistsByName(ctx, 2, "已存在场地")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestVenueRepository_GetByIDWithMerchant(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	merchant := &models.Merchant{
		Name:         "测试商户",
		ContactName:  "张三",
		ContactPhone: "13800138000",
	}
	db.Create(merchant)

	venue := &models.Venue{
		MerchantID: merchant.ID, Name: "测试场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
	}
	db.Create(venue)

	found, err := repo.GetByIDWithMerchant(ctx, venue.ID)
	require.NoError(t, err)
	assert.Equal(t, venue.ID, found.ID)
	assert.NotNil(t, found.Merchant)
	assert.Equal(t, merchant.ID, found.Merchant.ID)
	assert.Equal(t, "测试商户", found.Merchant.Name)
}

func TestVenueRepository_CountDevices(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	venue := &models.Venue{
		MerchantID: 1, Name: "测试场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
	}
	db.Create(venue)

	// 创建设备（包括各种状态）
	for i := 0; i < 3; i++ {
		db.Create(&models.Device{
			VenueID:     venue.ID,
			DeviceNo:    fmt.Sprintf("DEV%03d", i+1),
			Name:        "设备",
			Type:        "智能柜",
			QRCode:      fmt.Sprintf("QR%03d", i+1),
			ProductName: "商品",
			Status:      models.DeviceStatusActive,
		})
	}

	db.Model(&models.Device{}).Create(map[string]interface{}{
		"venue_id":     venue.ID,
		"device_no":    "DEV099",
		"name":         "设备",
		"type":         "智能柜",
		"qr_code":      "QR099",
		"product_name": "商品",
		"status":       models.DeviceStatusDisabled,
	})

	count, err := repo.CountDevices(ctx, venue.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(4), count) // 统计所有设备
}

func TestVenueRepository_ListByMerchantSimple(t *testing.T) {
	db := setupVenueTestDB(t)
	repo := NewVenueRepository(db)
	ctx := context.Background()

	db.Create(&models.Venue{
		MerchantID: 1, Name: "商户1场地1", Type: models.VenueTypeMall,
		Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园",
		Status: models.VenueStatusActive,
	})

	db.Model(&models.Venue{}).Create(map[string]interface{}{
		"merchant_id": 1,
		"name":        "商户1场地2",
		"type":        models.VenueTypeMall,
		"province":    "广东省",
		"city":        "深圳市",
		"district":    "福田区",
		"address":     "福华路",
		"status":      models.VenueStatusDisabled,
	})

	db.Create(&models.Venue{
		MerchantID: 2, Name: "商户2场地", Type: models.VenueTypeMall,
		Province: "广东省", City: "广州市", District: "天河区", Address: "天河路",
		Status: models.VenueStatusActive,
	})

	list, err := repo.ListByMerchantSimple(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list)) // 返回商户1的所有场地（包括禁用）
}
