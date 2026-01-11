// Package repository 预订仓储单元测试
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

func setupBookingTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Hotel{}, &models.Room{}, &models.Booking{}, &models.User{}, &models.Device{})
	require.NoError(t, err)

	return db
}

func TestBookingRepository_Create(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	checkOut := checkIn.Add(2 * time.Hour)

	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkOut,
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPending,
	}

	err := repo.Create(ctx, booking)
	require.NoError(t, err)
	assert.NotZero(t, booking.ID)
}

func TestBookingRepository_GetByID(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	found, err := repo.GetByID(ctx, booking.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.ID, found.ID)
	assert.Equal(t, "BK001", found.BookingNo)
}

func TestBookingRepository_GetByIDWithDetails(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
	ctx := context.Background()

	user := &models.User{
		Phone:    stringPtr("13800138000"),
		Nickname: "测试用户",
	}
	db.Create(user)

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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           user.ID,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	found, err := repo.GetByIDWithDetails(ctx, booking.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.User)
	assert.NotNil(t, found.Hotel)
	assert.NotNil(t, found.Room)
	assert.Equal(t, user.ID, found.User.ID)
	assert.Equal(t, hotel.ID, found.Hotel.ID)
}

func TestBookingRepository_GetByBookingNo(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	found, err := repo.GetByBookingNo(ctx, "BK001")
	require.NoError(t, err)
	assert.Equal(t, booking.ID, found.ID)
}

func TestBookingRepository_GetByOrderID(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          123,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	found, err := repo.GetByOrderID(ctx, 123)
	require.NoError(t, err)
	assert.Equal(t, booking.ID, found.ID)
}

func TestBookingRepository_GetByVerificationCode(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
	ctx := context.Background()

	user := &models.User{
		Phone:    stringPtr("13800138000"),
		Nickname: "测试用户",
	}
	db.Create(user)

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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           user.ID,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC123",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	found, err := repo.GetByVerificationCode(ctx, "VC123")
	require.NoError(t, err)
	assert.Equal(t, booking.ID, found.ID)
	assert.NotNil(t, found.User)
	assert.NotNil(t, found.Hotel)
	assert.NotNil(t, found.Room)
}

func TestBookingRepository_GetByUnlockCode(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		DeviceID:         &device.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC123",
		QRCode:           "QR001",
		Status:           models.BookingStatusVerified,
	}
	db.Create(booking)

	found, err := repo.GetByUnlockCode(ctx, "UC123", device.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.ID, found.ID)
}

func TestBookingRepository_Update(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPending,
	}
	db.Create(booking)

	booking.Status = models.BookingStatusPaid
	err := repo.Update(ctx, booking)
	require.NoError(t, err)

	var found models.Booking
	db.First(&found, booking.ID)
	assert.Equal(t, models.BookingStatusPaid, found.Status)
}

func TestBookingRepository_UpdateStatus(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPending,
	}
	db.Create(booking)

	err := repo.UpdateStatus(ctx, booking.ID, models.BookingStatusPaid)
	require.NoError(t, err)

	var found models.Booking
	db.First(&found, booking.ID)
	assert.Equal(t, models.BookingStatusPaid, found.Status)
}

func TestBookingRepository_List(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	bookings := []*models.Booking{
		{
			BookingNo: "BK001", OrderID: 1, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
			CheckInTime: checkIn, CheckOutTime: checkIn.Add(2 * time.Hour), DurationHours: 2,
			Amount: 200, VerificationCode: "VC001", UnlockCode: "UC001", QRCode: "QR001",
			Status: models.BookingStatusPaid,
		},
		{
			BookingNo: "BK002", OrderID: 2, UserID: 2, HotelID: hotel.ID, RoomID: room.ID,
			CheckInTime: checkIn.Add(3 * time.Hour), CheckOutTime: checkIn.Add(5 * time.Hour), DurationHours: 2,
			Amount: 200, VerificationCode: "VC002", UnlockCode: "UC002", QRCode: "QR002",
			Status: models.BookingStatusVerified,
		},
	}
	for _, b := range bookings {
		db.Create(b)
	}

	// 测试获取所有预订
	list, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))

	// 按用户过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"user_id": int64(1),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))

	// 按状态过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": models.BookingStatusPaid,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))

	// 按酒店过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"hotel_id": hotel.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestBookingRepository_ListByUser(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	bookings := []*models.Booking{
		{
			BookingNo: "BK001", OrderID: 1, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
			CheckInTime: checkIn, CheckOutTime: checkIn.Add(2 * time.Hour), DurationHours: 2,
			Amount: 200, VerificationCode: "VC001", UnlockCode: "UC001", QRCode: "QR001",
			Status: models.BookingStatusPaid,
		},
		{
			BookingNo: "BK002", OrderID: 2, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
			CheckInTime: checkIn.Add(3 * time.Hour), CheckOutTime: checkIn.Add(5 * time.Hour), DurationHours: 2,
			Amount: 200, VerificationCode: "VC002", UnlockCode: "UC002", QRCode: "QR002",
			Status: models.BookingStatusCompleted,
		},
		{
			BookingNo: "BK003", OrderID: 3, UserID: 2, HotelID: hotel.ID, RoomID: room.ID,
			CheckInTime: checkIn, CheckOutTime: checkIn.Add(2 * time.Hour), DurationHours: 2,
			Amount: 200, VerificationCode: "VC003", UnlockCode: "UC003", QRCode: "QR003",
			Status: models.BookingStatusPaid,
		},
	}
	for _, b := range bookings {
		db.Create(b)
	}

	list, total, err := repo.ListByUser(ctx, 1, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))

	status := models.BookingStatusPaid
	list, total, err = repo.ListByUser(ctx, 1, 0, 10, &status)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}

func TestBookingRepository_Verify(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	err := repo.Verify(ctx, booking.ID, 123)
	require.NoError(t, err)

	var found models.Booking
	db.First(&found, booking.ID)
	assert.Equal(t, models.BookingStatusVerified, found.Status)
	assert.NotNil(t, found.VerifiedAt)
	assert.NotNil(t, found.VerifiedBy)
	assert.Equal(t, int64(123), *found.VerifiedBy)
}

func TestBookingRepository_Unlock(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusVerified,
	}
	db.Create(booking)

	err := repo.Unlock(ctx, booking.ID)
	require.NoError(t, err)

	var found models.Booking
	db.First(&found, booking.ID)
	assert.Equal(t, models.BookingStatusInUse, found.Status)
	assert.NotNil(t, found.UnlockedAt)
}

func TestBookingRepository_Complete(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusInUse,
	}
	db.Create(booking)

	err := repo.Complete(ctx, booking.ID)
	require.NoError(t, err)

	var found models.Booking
	db.First(&found, booking.ID)
	assert.Equal(t, models.BookingStatusCompleted, found.Status)
	assert.NotNil(t, found.CompletedAt)
}

func TestBookingRepository_Cancel(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BK001",
		OrderID:          1,
		UserID:           1,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkIn,
		CheckOutTime:     checkIn.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           200,
		VerificationCode: "VC001",
		UnlockCode:       "UC001",
		QRCode:           "QR001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	err := repo.Cancel(ctx, booking.ID)
	require.NoError(t, err)

	var found models.Booking
	db.First(&found, booking.ID)
	assert.Equal(t, models.BookingStatusCancelled, found.Status)
}

func TestBookingRepository_ListActiveBookings(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	checkOut := checkIn.Add(3 * time.Hour)

	// 活跃预订
	db.Create(&models.Booking{
		BookingNo: "BK001", OrderID: 1, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
		CheckInTime: checkIn, CheckOutTime: checkOut, DurationHours: 3,
		Amount: 300, VerificationCode: "VC001", UnlockCode: "UC001", QRCode: "QR001",
		Status: models.BookingStatusPaid,
	})

	// 已完成预订
	db.Create(&models.Booking{
		BookingNo: "BK002", OrderID: 2, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
		CheckInTime: checkIn, CheckOutTime: checkOut, DurationHours: 3,
		Amount: 300, VerificationCode: "VC002", UnlockCode: "UC002", QRCode: "QR002",
		Status: models.BookingStatusCompleted,
	})

	list, err := repo.ListActiveBookings(ctx, room.ID, checkIn.Add(1*time.Hour), checkOut.Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, len(list)) // 只返回活跃状态的预订
	assert.Equal(t, models.BookingStatusPaid, list[0].Status)
}

func TestBookingRepository_ListExpiredBookings(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	// 已过期（入住时间在过去）
	pastCheckIn := time.Now().Add(-2 * time.Hour)
	db.Create(&models.Booking{
		BookingNo: "BK001", OrderID: 1, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
		CheckInTime: pastCheckIn, CheckOutTime: pastCheckIn.Add(2 * time.Hour), DurationHours: 2,
		Amount: 200, VerificationCode: "VC001", UnlockCode: "UC001", QRCode: "QR001",
		Status: models.BookingStatusPaid,
	})

	// 未过期
	futureCheckIn := time.Now().Add(2 * time.Hour)
	db.Create(&models.Booking{
		BookingNo: "BK002", OrderID: 2, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
		CheckInTime: futureCheckIn, CheckOutTime: futureCheckIn.Add(2 * time.Hour), DurationHours: 2,
		Amount: 200, VerificationCode: "VC002", UnlockCode: "UC002", QRCode: "QR002",
		Status: models.BookingStatusPaid,
	})

	list, err := repo.ListExpiredBookings(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
}

func TestBookingRepository_ListToComplete(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	// 离店时间已过
	pastCheckOut := time.Now().Add(-1 * time.Hour)
	db.Create(&models.Booking{
		BookingNo: "BK001", OrderID: 1, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
		CheckInTime: pastCheckOut.Add(-2 * time.Hour), CheckOutTime: pastCheckOut, DurationHours: 2,
		Amount: 200, VerificationCode: "VC001", UnlockCode: "UC001", QRCode: "QR001",
		Status: models.BookingStatusInUse,
	})

	// 离店时间未到
	futureCheckOut := time.Now().Add(2 * time.Hour)
	db.Create(&models.Booking{
		BookingNo: "BK002", OrderID: 2, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
		CheckInTime: futureCheckOut.Add(-2 * time.Hour), CheckOutTime: futureCheckOut, DurationHours: 2,
		Amount: 200, VerificationCode: "VC002", UnlockCode: "UC002", QRCode: "QR002",
		Status: models.BookingStatusInUse,
	})

	list, err := repo.ListToComplete(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
}

func TestBookingRepository_CountByUserAndStatus(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	bookings := []*models.Booking{
		{
			BookingNo: "BK001", OrderID: 1, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
			CheckInTime: checkIn, CheckOutTime: checkIn.Add(2 * time.Hour), DurationHours: 2,
			Amount: 200, VerificationCode: "VC001", UnlockCode: "UC001", QRCode: "QR001",
			Status: models.BookingStatusPaid,
		},
		{
			BookingNo: "BK002", OrderID: 2, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
			CheckInTime: checkIn, CheckOutTime: checkIn.Add(2 * time.Hour), DurationHours: 2,
			Amount: 200, VerificationCode: "VC002", UnlockCode: "UC002", QRCode: "QR002",
			Status: models.BookingStatusVerified,
		},
		{
			BookingNo: "BK003", OrderID: 3, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
			CheckInTime: checkIn, CheckOutTime: checkIn.Add(2 * time.Hour), DurationHours: 2,
			Amount: 200, VerificationCode: "VC003", UnlockCode: "UC003", QRCode: "QR003",
			Status: models.BookingStatusCompleted,
		},
	}
	for _, b := range bookings {
		db.Create(b)
	}

	count, err := repo.CountByUserAndStatus(ctx, 1, []string{models.BookingStatusPaid, models.BookingStatusVerified})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestBookingRepository_ExistsByRoomAndTimeRange(t *testing.T) {
	db := setupBookingTestDB(t)
	repo := NewBookingRepository(db)
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

	checkIn := time.Now().Add(1 * time.Hour)
	checkOut := checkIn.Add(3 * time.Hour)
	db.Create(&models.Booking{
		BookingNo: "BK001", OrderID: 1, UserID: 1, HotelID: hotel.ID, RoomID: room.ID,
		CheckInTime: checkIn, CheckOutTime: checkOut, DurationHours: 3,
		Amount: 300, VerificationCode: "VC001", UnlockCode: "UC001", QRCode: "QR001",
		Status: models.BookingStatusPaid,
	})

	// 冲突时段
	exists, err := repo.ExistsByRoomAndTimeRange(ctx, room.ID, checkIn.Add(1*time.Hour), checkOut.Add(1*time.Hour))
	require.NoError(t, err)
	assert.True(t, exists)

	// 不冲突时段
	exists, err = repo.ExistsByRoomAndTimeRange(ctx, room.ID, checkOut.Add(1*time.Hour), checkOut.Add(4*time.Hour))
	require.NoError(t, err)
	assert.False(t, exists)
}
