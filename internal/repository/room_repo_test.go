// Package repository 房间仓储单元测试
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupRoomTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Hotel{}, &models.Room{}, &models.RoomTimeSlot{}, &models.Device{}, &models.Booking{})
	require.NoError(t, err)

	return db
}

func TestRoomRepository_Create(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	// 创建酒店
	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}

	err := repo.Create(ctx, room)
	require.NoError(t, err)
	assert.NotZero(t, room.ID)
}

func TestRoomRepository_GetByID(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	found, err := repo.GetByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, room.ID, found.ID)
	assert.Equal(t, "101", found.RoomNo)
}

func TestRoomRepository_GetByIDWithHotel(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	found, err := repo.GetByIDWithHotel(ctx, room.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.Hotel)
	assert.Equal(t, hotel.ID, found.Hotel.ID)
	assert.Equal(t, "测试酒店", found.Hotel.Name)
}

func TestRoomRepository_GetByIDWithTimeSlots(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	// 创建时段 - 活跃
	db.Create(&models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         200,
		IsActive:      true,
	})

	// 创建时段 - 停用，使用map
	db.Model(&models.RoomTimeSlot{}).Create(map[string]interface{}{
		"room_id":        room.ID,
		"duration_hours": 4,
		"price":          400,
		"is_active":      false,
	})

	found, err := repo.GetByIDWithTimeSlots(ctx, room.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.Hotel)
	assert.Equal(t, 1, len(found.TimeSlots)) // 只加载活跃时段
}

func TestRoomRepository_GetByIDWithDevice(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	device := &models.Device{
		DeviceNo: "DEV001",
		Name:     "测试设备",
		VenueID:  1,
		QRCode:   "QR001",
		ProductName: "测试产品",
		Status:   models.DeviceStatusActive,
	}
	db.Create(device)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
		DeviceID:    &device.ID,
	}
	db.Create(room)

	found, err := repo.GetByIDWithDevice(ctx, room.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.Device)
	assert.Equal(t, device.ID, found.Device.ID)
}

func TestRoomRepository_Update(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	room.RoomNo = "102"
	room.HourlyPrice = 150
	err := repo.Update(ctx, room)
	require.NoError(t, err)

	var found models.Room
	db.First(&found, room.ID)
	assert.Equal(t, "102", found.RoomNo)
	assert.Equal(t, 150.0, found.HourlyPrice)
}

func TestRoomRepository_UpdateFields(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	err := repo.UpdateFields(ctx, room.ID, map[string]interface{}{
		"hourly_price": 120,
		"daily_price":  600,
	})
	require.NoError(t, err)

	var found models.Room
	db.First(&found, room.ID)
	assert.Equal(t, 120.0, found.HourlyPrice)
	assert.Equal(t, 600.0, found.DailyPrice)
}

func TestRoomRepository_UpdateStatus(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
		Status:      models.RoomStatusActive,
	}
	db.Create(room)

	err := repo.UpdateStatus(ctx, room.ID, models.RoomStatusDisabled)
	require.NoError(t, err)

	var found models.Room
	db.First(&found, room.ID)
	assert.Equal(t, int8(models.RoomStatusDisabled), found.Status)
}

func TestRoomRepository_List(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	rooms := []*models.Room{
		{HotelID: hotel.ID, RoomNo: "101", RoomType: models.RoomTypeStandard, HourlyPrice: 100, DailyPrice: 500, Status: models.RoomStatusActive},
		{HotelID: hotel.ID, RoomNo: "102", RoomType: models.RoomTypeBusiness, HourlyPrice: 150, DailyPrice: 800, Status: models.RoomStatusActive},
	}
	for _, r := range rooms {
		db.Create(r)
	}

	// 按酒店过滤
	list, total, err := repo.List(ctx, 0, 10, map[string]interface{}{
		"hotel_id": hotel.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))

	// 按房间类型过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"hotel_id":  hotel.ID,
		"room_type": models.RoomTypeStandard,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))

	// 按价格过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"hotel_id":  hotel.ID,
		"max_price": 120.0,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}

func TestRoomRepository_ListByHotel(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
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

	db.Model(&models.Room{}).Create(map[string]interface{}{
		"hotel_id":     hotel.ID,
		"room_no":      "102",
		"room_type":    models.RoomTypeStandard,
		"hourly_price": 100,
		"daily_price":  500,
		"status":       models.RoomStatusDisabled,
	})

	// 获取所有房间
	list, err := repo.ListByHotel(ctx, hotel.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))

	// 只获取活跃房间
	status := int8(models.RoomStatusActive)
	list, err = repo.ListByHotel(ctx, hotel.ID, &status)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
}

func TestRoomRepository_ListAvailableByHotel(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
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

	db.Model(&models.Room{}).Create(map[string]interface{}{
		"hotel_id":     hotel.ID,
		"room_no":      "102",
		"room_type":    models.RoomTypeStandard,
		"hourly_price": 100,
		"daily_price":  500,
		"status":       models.RoomStatusDisabled,
	})

	list, err := repo.ListAvailableByHotel(ctx, hotel.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
	assert.Equal(t, "101", list[0].RoomNo)
}

func TestRoomRepository_ExistsByRoomNo(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
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
	})

	exists, err := repo.ExistsByRoomNo(ctx, hotel.ID, "101")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByRoomNo(ctx, hotel.ID, "999")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRoomRepository_Delete(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	err := repo.Delete(ctx, room.ID)
	require.NoError(t, err)

	err = db.First(&models.Room{}, room.ID).Error
	assert.Error(t, err)
}

func TestRoomRepository_GetByDeviceID(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	device := &models.Device{
		DeviceNo: "DEV001",
		Name:     "测试设备",
		VenueID:  1,
		QRCode:   "QR001",
		ProductName: "测试产品",
		Status:   models.DeviceStatusActive,
	}
	db.Create(device)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
		DeviceID:    &device.ID,
	}
	db.Create(room)

	found, err := repo.GetByDeviceID(ctx, device.ID)
	require.NoError(t, err)
	assert.Equal(t, room.ID, found.ID)
	assert.NotNil(t, found.Hotel)
}

func TestRoomRepository_BindDevice(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	device := &models.Device{
		DeviceNo: "DEV001",
		Name:     "测试设备",
		VenueID:  1,
		QRCode:   "QR001",
		ProductName: "测试产品",
		Status:   models.DeviceStatusActive,
	}
	db.Create(device)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	err := repo.BindDevice(ctx, room.ID, device.ID)
	require.NoError(t, err)

	var found models.Room
	db.First(&found, room.ID)
	assert.NotNil(t, found.DeviceID)
	assert.Equal(t, device.ID, *found.DeviceID)
}

func TestRoomRepository_UnbindDevice(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	device := &models.Device{
		DeviceNo: "DEV001",
		Name:     "测试设备",
		VenueID:  1,
		QRCode:   "QR001",
		ProductName: "测试产品",
		Status:   models.DeviceStatusActive,
	}
	db.Create(device)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
		DeviceID:    &device.ID,
	}
	db.Create(room)

	err := repo.UnbindDevice(ctx, room.ID)
	require.NoError(t, err)

	var found models.Room
	db.First(&found, room.ID)
	assert.Nil(t, found.DeviceID)
}

func TestRoomRepository_CheckAvailability(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	// 创建已支付的预订
	checkIn := time.Now().Add(1 * time.Hour)
	checkOut := checkIn.Add(3 * time.Hour)
	db.Create(&models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkOut,
		DurationHours:    3,
		Amount:           300,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPaid,
	})

	// 检查冲突时段
	available, err := repo.CheckAvailability(ctx, room.ID, checkIn.Add(1*time.Hour), checkOut.Add(1*time.Hour))
	require.NoError(t, err)
	assert.False(t, available)

	// 检查不冲突时段
	available, err = repo.CheckAvailability(ctx, room.ID, checkOut.Add(1*time.Hour), checkOut.Add(4*time.Hour))
	require.NoError(t, err)
	assert.True(t, available)
}

// RoomTimeSlot repository tests

func TestRoomTimeSlotRepository_Create(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slot := &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         200,
	}

	err := repo.Create(ctx, slot)
	require.NoError(t, err)
	assert.NotZero(t, slot.ID)
}

func TestRoomTimeSlotRepository_CreateBatch(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slots := []*models.RoomTimeSlot{
		{RoomID: room.ID, DurationHours: 2, Price: 200},
		{RoomID: room.ID, DurationHours: 4, Price: 380},
		{RoomID: room.ID, DurationHours: 8, Price: 720},
	}

	err := repo.CreateBatch(ctx, slots)
	require.NoError(t, err)

	var count int64
	db.Model(&models.RoomTimeSlot{}).Where("room_id = ?", room.ID).Count(&count)
	assert.Equal(t, int64(3), count)
}

func TestRoomTimeSlotRepository_GetByID(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slot := &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         200,
	}
	db.Create(slot)

	found, err := repo.GetByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, slot.ID, found.ID)
	assert.Equal(t, 2, found.DurationHours)
}

func TestRoomTimeSlotRepository_GetByRoomAndDuration(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	db.Create(&models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         200,
		IsActive:      true,
	})

	found, err := repo.GetByRoomAndDuration(ctx, room.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, found.DurationHours)
	assert.Equal(t, 200.0, found.Price)
}

func TestRoomTimeSlotRepository_Update(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slot := &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         200,
	}
	db.Create(slot)

	slot.Price = 250
	err := repo.Update(ctx, slot)
	require.NoError(t, err)

	var found models.RoomTimeSlot
	db.First(&found, slot.ID)
	assert.Equal(t, 250.0, found.Price)
}

func TestRoomTimeSlotRepository_UpdateFields(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slot := &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         200,
	}
	db.Create(slot)

	err := repo.UpdateFields(ctx, slot.ID, map[string]interface{}{
		"price": 260,
		"sort":  10,
	})
	require.NoError(t, err)

	var found models.RoomTimeSlot
	db.First(&found, slot.ID)
	assert.Equal(t, 260.0, found.Price)
	assert.Equal(t, 10, found.Sort)
}

func TestRoomTimeSlotRepository_ListByRoom(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slots := []*models.RoomTimeSlot{
		{RoomID: room.ID, DurationHours: 2, Price: 200, IsActive: true},
		{RoomID: room.ID, DurationHours: 4, Price: 380, IsActive: true},
	}
	for _, s := range slots {
		db.Create(s)
	}

	// 使用 map 创建 is_active=false 的时段
	db.Model(&models.RoomTimeSlot{}).Create(map[string]interface{}{
		"room_id":        room.ID,
		"duration_hours": 8,
		"price":          720,
		"is_active":      false,
	})

	list, err := repo.ListByRoom(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(list)) // 包含所有时段
}

func TestRoomTimeSlotRepository_ListActiveByRoom(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slots := []*models.RoomTimeSlot{
		{RoomID: room.ID, DurationHours: 2, Price: 200, IsActive: true},
		{RoomID: room.ID, DurationHours: 4, Price: 380, IsActive: true},
	}
	for _, s := range slots {
		db.Create(s)
	}

	// 使用 map 创建 is_active=false 的时段
	db.Model(&models.RoomTimeSlot{}).Create(map[string]interface{}{
		"room_id":        room.ID,
		"duration_hours": 8,
		"price":          720,
		"is_active":      false,
	})

	list, err := repo.ListActiveByRoom(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list)) // 只包含活跃时段
}

func TestRoomTimeSlotRepository_Delete(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slot := &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         200,
	}
	db.Create(slot)

	err := repo.Delete(ctx, slot.ID)
	require.NoError(t, err)

	err = db.First(&models.RoomTimeSlot{}, slot.ID).Error
	assert.Error(t, err)
}

func TestRoomTimeSlotRepository_DeleteByRoom(t *testing.T) {
	db := setupRoomTestDB(t)
	repo := NewRoomTimeSlotRepository(db)
	ctx := context.Background()

	hotel := &models.Hotel{
		Name: "测试酒店", Province: "广东省", City: "深圳市", District: "南山区",
		Address: "addr1", Phone: "123",
	}
	db.Create(hotel)

	room := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 100,
		DailyPrice:  500,
	}
	db.Create(room)

	slots := []*models.RoomTimeSlot{
		{RoomID: room.ID, DurationHours: 2, Price: 200},
		{RoomID: room.ID, DurationHours: 4, Price: 380},
	}
	for _, s := range slots {
		db.Create(s)
	}

	err := repo.DeleteByRoom(ctx, room.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.RoomTimeSlot{}).Where("room_id = ?", room.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}
