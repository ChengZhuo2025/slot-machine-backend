// Package repository 酒店仓储单元测试
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupHotelTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Hotel{}, &models.Room{})
	require.NoError(t, err)

	return db
}

func TestHotelRepository_Create(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name:     "测试酒店",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-12345678",
	}

	err := repo.Create(ctx, hotel)
	require.NoError(t, err)
	assert.NotZero(t, hotel.ID)
}

func TestHotelRepository_GetByID(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name:     "测试酒店",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-12345678",
	}
	db.Create(hotel)

	found, err := repo.GetByID(ctx, hotel.ID)
	require.NoError(t, err)
	assert.Equal(t, hotel.ID, found.ID)
	assert.Equal(t, "测试酒店", found.Name)
}

func TestHotelRepository_GetByIDWithRooms(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name:     "测试酒店",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-12345678",
	}
	db.Create(hotel)

	// 创建房间
	db.Create(&models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
		Status:      models.RoomStatusActive,
	})

	db.Model(&models.Room{}).Create(map[string]interface{}{
		"hotel_id":     hotel.ID,
		"room_no":      "102",
		"room_type":    models.RoomTypeStandard,
		"hourly_price": 100,
		"daily_price":  500,
		"status":       models.RoomStatusDisabled,
	})

	found, err := repo.GetByIDWithRooms(ctx, hotel.ID)
	require.NoError(t, err)
	assert.Equal(t, hotel.ID, found.ID)
	assert.Equal(t, 1, len(found.Rooms)) // 只加载 active 状态的房间
}

func TestHotelRepository_Update(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name:     "原酒店名",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "原地址",
		Phone:    "0755-12345678",
	}
	db.Create(hotel)

	hotel.Name = "新酒店名"
	hotel.Address = "新地址"
	err := repo.Update(ctx, hotel)
	require.NoError(t, err)

	var found models.Hotel
	db.First(&found, hotel.ID)
	assert.Equal(t, "新酒店名", found.Name)
	assert.Equal(t, "新地址", found.Address)
}

func TestHotelRepository_UpdateFields(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name:     "测试酒店",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-12345678",
	}
	db.Create(hotel)

	err := repo.UpdateFields(ctx, hotel.ID, map[string]interface{}{
		"phone": "0755-87654321",
		"name":  "更新后的酒店",
	})
	require.NoError(t, err)

	var found models.Hotel
	db.First(&found, hotel.ID)
	assert.Equal(t, "0755-87654321", found.Phone)
	assert.Equal(t, "更新后的酒店", found.Name)
}

func TestHotelRepository_UpdateStatus(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name:     "测试酒店",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-12345678",
		Status:   models.HotelStatusActive,
	}
	db.Create(hotel)

	err := repo.UpdateStatus(ctx, hotel.ID, models.HotelStatusDisabled)
	require.NoError(t, err)

	var found models.Hotel
	db.First(&found, hotel.ID)
	assert.Equal(t, int8(models.HotelStatusDisabled), found.Status)
}

func TestHotelRepository_List(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	// 创建测试数据
	hotels := []*models.Hotel{
		{Name: "深圳酒店", Province: "广东省", City: "深圳市", District: "南山区", Address: "addr1", Phone: "123", Status: models.HotelStatusActive},
		{Name: "广州酒店", Province: "广东省", City: "广州市", District: "天河区", Address: "addr2", Phone: "456", Status: models.HotelStatusActive},
	}
	for _, h := range hotels {
		db.Create(h)
	}

	db.Model(&models.Hotel{}).Create(map[string]interface{}{
		"name":     "停用酒店",
		"province": "广东省",
		"city":     "深圳市",
		"district": "福田区",
		"address":  "addr3",
		"phone":    "789",
		"status":   models.HotelStatusDisabled,
	})

	// 测试获取所有酒店
	list, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 3, len(list))

	// 按城市过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"city": "深圳市",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按状态过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": int8(models.HotelStatusActive),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按名称模糊查询
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"name": "深圳",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestHotelRepository_ListActive(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	// 创建测试数据
	db.Create(&models.Hotel{
		Name: "活跃酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123", Status: models.HotelStatusActive,
	})

	db.Model(&models.Hotel{}).Create(map[string]interface{}{
		"name": "停用酒店", "province": "广东省", "city": "深圳市", "district": "福田区",
		"address": "addr2", "phone": "456", "status": models.HotelStatusDisabled,
	})

	list, total, err := repo.ListActive(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}

func TestHotelRepository_ListByCity(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotels := []*models.Hotel{
		{Name: "深圳酒店1", Province: "广东省", City: "深圳市", District: "南山区", Address: "addr1", Phone: "123", Status: models.HotelStatusActive},
		{Name: "深圳酒店2", Province: "广东省", City: "深圳市", District: "福田区", Address: "addr2", Phone: "456", Status: models.HotelStatusActive},
		{Name: "广州酒店", Province: "广东省", City: "广州市", District: "天河区", Address: "addr3", Phone: "789", Status: models.HotelStatusActive},
	}
	for _, h := range hotels {
		db.Create(h)
	}

	list, total, err := repo.ListByCity(ctx, "深圳市", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestHotelRepository_GetCities(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotels := []*models.Hotel{
		{Name: "深圳酒店", Province: "广东省", City: "深圳市", District: "南山区", Address: "addr1", Phone: "123", Status: models.HotelStatusActive},
		{Name: "广州酒店", Province: "广东省", City: "广州市", District: "天河区", Address: "addr2", Phone: "456", Status: models.HotelStatusActive},
		{Name: "北京酒店", Province: "北京市", City: "北京市", District: "朝阳区", Address: "addr3", Phone: "789", Status: models.HotelStatusActive},
	}
	for _, h := range hotels {
		db.Create(h)
	}

	cities, err := repo.GetCities(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, len(cities))
	assert.Contains(t, cities, "深圳市")
	assert.Contains(t, cities, "广州市")
	assert.Contains(t, cities, "北京市")
}

func TestHotelRepository_ExistsByName(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	db.Create(&models.Hotel{
		Name: "已存在酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	})

	exists, err := repo.ExistsByName(ctx, "已存在酒店")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByName(ctx, "不存在酒店")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestHotelRepository_Delete(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "待删除酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	err := repo.Delete(ctx, hotel.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Hotel{}).Where("id = ?", hotel.ID).Count(&count)
	assert.Equal(t, int64(0), count) // 硬删除，记录不存在
}

func TestHotelRepository_GetRoomCount(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	// 创建活跃房间
	for i := 1; i <= 3; i++ {
		db.Create(&models.Room{
			HotelID:     hotel.ID,
			RoomNo:      string(rune('0' + i)),
			RoomType:    models.RoomTypeStandard,
			HourlyPrice: 100,
			DailyPrice:  500,
			Status:      models.RoomStatusActive,
		})
	}

	// 创建停用房间
	db.Model(&models.Room{}).Create(map[string]interface{}{
		"hotel_id":     hotel.ID,
		"room_no":      "4",
		"room_type":    models.RoomTypeStandard,
		"hourly_price": 100,
		"daily_price":  500,
		"status":       models.RoomStatusDisabled,
	})

	count, err := repo.GetRoomCount(ctx, hotel.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count) // 只统计活跃房间
}

func TestHotelRepository_GetAvailableRoomCount(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	db.Create(&models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
		Status:      models.RoomStatusActive,
	})

	db.Create(&models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "102",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
		Status:      models.RoomStatusActive,
	})

	count, err := repo.GetAvailableRoomCount(ctx, hotel.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestHotelRepository_Search(t *testing.T) {
	db := setupHotelTestDB(t)
	repo := NewHotelRepository(db)
	ctx := context.Background()

	hotels := []*models.Hotel{
		{Name: "豪华酒店", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园", Phone: "123", Status: models.HotelStatusActive},
		{Name: "商务酒店", Province: "广东省", City: "深圳市", District: "福田区", Address: "福华路", Phone: "456", Status: models.HotelStatusActive},
	}
	for _, h := range hotels {
		db.Create(h)
	}

	// 搜索名称
	list, total, err := repo.Search(ctx, "豪华", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))

	// 搜索地址
	list, total, err = repo.Search(ctx, "科技园", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}
