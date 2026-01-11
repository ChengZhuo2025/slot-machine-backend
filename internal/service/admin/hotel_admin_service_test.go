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

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupHotelAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.Hotel{},
		&models.Room{},
		&models.Booking{},
		&models.RoomTimeSlot{},
	))
	return db
}

func TestHotelAdminService_CreateHotelAndRoom(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	hotel, err := svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店A",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})
	require.NoError(t, err)
	require.NotNil(t, hotel)
	assert.Equal(t, int8(models.HotelStatusActive), hotel.Status)
	assert.Equal(t, "14:00", hotel.CheckInTime)
	assert.Equal(t, "12:00", hotel.CheckOutTime)

	t.Run("CreateHotel 名称重复", func(t *testing.T) {
		_, err := svc.CreateHotel(ctx, &CreateHotelRequest{
			Name:     "酒店A",
			Province: "广东省",
			City:     "深圳市",
			District: "南山区",
			Address:  "科技园",
			Phone:    "0755-123456",
		})
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrAlreadyExists.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "酒店名称已存在")
	})

	t.Run("CreateRoom 酒店不存在", func(t *testing.T) {
		_, err := svc.CreateRoom(ctx, &CreateRoomRequest{
			HotelID:     99999,
			RoomNo:      "101",
			RoomType:    models.RoomTypeStandard,
			HourlyPrice: 60,
			DailyPrice:  288,
		})
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrHotelNotFound.Code, appErr.Code)
	})

	room, err := svc.CreateRoom(ctx, &CreateRoomRequest{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 60,
		DailyPrice:  288,
	})
	require.NoError(t, err)
	require.NotNil(t, room)

	t.Run("CreateRoom 房间号重复", func(t *testing.T) {
		_, err := svc.CreateRoom(ctx, &CreateRoomRequest{
			HotelID:     hotel.ID,
			RoomNo:      "101",
			RoomType:    models.RoomTypeStandard,
			HourlyPrice: 60,
			DailyPrice:  288,
		})
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrAlreadyExists.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "房间号已存在")
	})

	t.Run("DeleteHotel 先删除房间", func(t *testing.T) {
		err := svc.DeleteHotel(ctx, hotel.ID)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrOperationFailed.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "请先删除所有房间")

		// 清理房间后可删除
		require.NoError(t, db.Delete(&models.Room{}, room.ID).Error)
		require.NoError(t, svc.DeleteHotel(ctx, hotel.ID))
	})
}

func TestHotelAdminService_GetHotelList(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	// 创建酒店
	svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店列表1",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})

	hotels, total, err := svc.GetHotelList(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.True(t, total >= 1)
	assert.NotEmpty(t, hotels)
}

func TestHotelAdminService_GetHotelByID(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	hotel, _ := svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店详情",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})

	result, err := svc.GetHotelByID(ctx, hotel.ID)
	require.NoError(t, err)
	assert.Equal(t, hotel.ID, result.ID)
}

func TestHotelAdminService_UpdateHotel(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	hotel, _ := svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店更新",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})

	updated, err := svc.UpdateHotel(ctx, hotel.ID, &UpdateHotelRequest{
		Name:    "酒店更新后",
		Address: "新地址",
	})
	require.NoError(t, err)
	assert.Equal(t, "酒店更新后", updated.Name)
}

func TestHotelAdminService_UpdateHotelStatus(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	hotel, _ := svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店状态",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})

	err := svc.UpdateHotelStatus(ctx, hotel.ID, models.HotelStatusClosed)
	require.NoError(t, err)
}

func TestHotelAdminService_UpdateRoom(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	hotel, _ := svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店房间更新",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})

	room, _ := svc.CreateRoom(ctx, &CreateRoomRequest{
		HotelID:     hotel.ID,
		RoomNo:      "201",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 60,
		DailyPrice:  288,
	})

	updated, err := svc.UpdateRoom(ctx, room.ID, &UpdateRoomRequest{
		RoomNo:      "202",
		HourlyPrice: 80,
	})
	require.NoError(t, err)
	assert.Equal(t, "202", updated.RoomNo)
}

func TestHotelAdminService_GetRoomByID(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	hotel, _ := svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店房间详情",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})

	room, _ := svc.CreateRoom(ctx, &CreateRoomRequest{
		HotelID:     hotel.ID,
		RoomNo:      "301",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 60,
		DailyPrice:  288,
	})

	result, err := svc.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, room.ID, result.ID)
}

func TestHotelAdminService_GetRoomList(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	hotel, _ := svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店房间列表",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})

	svc.CreateRoom(ctx, &CreateRoomRequest{
		HotelID:     hotel.ID,
		RoomNo:      "401",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 60,
		DailyPrice:  288,
	})

	rooms, total, err := svc.GetRoomList(ctx, hotel.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, rooms, 1)
}

func TestHotelAdminService_DeleteRoom(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	hotel, _ := svc.CreateHotel(ctx, &CreateHotelRequest{
		Name:     "酒店房间删除",
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Address:  "科技园",
		Phone:    "0755-123456",
	})

	room, _ := svc.CreateRoom(ctx, &CreateRoomRequest{
		HotelID:     hotel.ID,
		RoomNo:      "501",
		RoomType:    models.RoomTypeStandard,
		HourlyPrice: 60,
		DailyPrice:  288,
	})

	err := svc.DeleteRoom(ctx, room.ID)
	require.NoError(t, err)
}

func TestHotelAdminService_GetBookingList(t *testing.T) {
	db := setupHotelAdminTestDB(t)
	svc := NewHotelAdminService(
		db,
		repository.NewHotelRepository(db),
		repository.NewRoomRepository(db),
		repository.NewBookingRepository(db),
		repository.NewRoomTimeSlotRepository(db),
	)
	ctx := context.Background()

	bookings, total, err := svc.GetBookingList(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, bookings)
}

