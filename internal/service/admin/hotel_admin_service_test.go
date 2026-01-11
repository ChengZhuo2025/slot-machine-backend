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

