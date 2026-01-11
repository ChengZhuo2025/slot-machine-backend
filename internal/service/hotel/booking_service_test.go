// Package hotel 预订服务单元测试
package hotel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Order{},
		&models.Hotel{},
		&models.Room{},
		&models.RoomTimeSlot{},
		&models.Booking{},
		&models.Device{},
	)
	require.NoError(t, err)

	// 创建默认会员等级
	level := &models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
	}
	db.Create(level)

	return db
}

// testBookingService 测试用预订服务
type testBookingService struct {
	*BookingService
	db *gorm.DB
}

// setupTestBookingService 创建测试用的 BookingService
func setupTestBookingService(t *testing.T) *testBookingService {
	db := setupTestDB(t)
	bookingRepo := repository.NewBookingRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	hotelRepo := repository.NewHotelRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	timeSlotRepo := repository.NewRoomTimeSlotRepository(db)
	codeService := NewCodeService()

	service := NewBookingService(db, bookingRepo, roomRepo, hotelRepo, orderRepo, timeSlotRepo, codeService, nil, nil)

	return &testBookingService{
		BookingService: service,
		db:             db,
	}
}

// createTestBookingData 创建预订测试数据
func createTestBookingData(t *testing.T, db *gorm.DB) (user *models.User, hotel *models.Hotel, room *models.Room, timeSlot *models.RoomTimeSlot) {
	// 创建用户
	phone := "13800138000"
	user = &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 500.0,
	}
	err = db.Create(wallet).Error
	require.NoError(t, err)

	// 创建酒店
	description := "测试酒店描述"
	starRating := 4
	hotel = &models.Hotel{
		Name:         "测试酒店",
		StarRating:   &starRating,
		Province:     "广东省",
		City:         "深圳市",
		District:     "南山区",
		Address:      "科技园路1号",
		Phone:        "0755-12345678",
		Description:  &description,
		CheckInTime:  "14:00",
		CheckOutTime: "12:00",
		Status:       models.HotelStatusActive,
	}
	err = db.Create(hotel).Error
	require.NoError(t, err)

	// 创建房间
	area := 25
	bedType := "大床"
	room = &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		Area:        &area,
		BedType:     &bedType,
		MaxGuests:   2,
		HourlyPrice: 60.0,
		DailyPrice:  288.0,
		Status:      models.RoomStatusActive,
	}
	err = db.Create(room).Error
	require.NoError(t, err)

	// 创建时段价格
	timeSlot = &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         100.0,
		IsActive:      true,
		Sort:          1,
	}
	err = db.Create(timeSlot).Error
	require.NoError(t, err)

	return user, hotel, room, timeSlot
}

func TestBookingService_CreateBooking(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, _, room, _ := createTestBookingData(t, svc.db)

	t.Run("创建预订成功", func(t *testing.T) {
		checkInTime := time.Now().Add(1 * time.Hour)
		req := &CreateBookingRequest{
			RoomID:        room.ID,
			DurationHours: 2,
			CheckInTime:   checkInTime,
		}

		bookingInfo, err := svc.CreateBooking(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, bookingInfo)
		assert.NotEmpty(t, bookingInfo.BookingNo)
		assert.Equal(t, models.BookingStatusPending, bookingInfo.Status)
		assert.Equal(t, "待支付", bookingInfo.StatusName)
		assert.Equal(t, 100.0, bookingInfo.Amount)
		assert.Equal(t, 2, bookingInfo.DurationHours)
		assert.NotEmpty(t, bookingInfo.VerificationCode)
		assert.NotEmpty(t, bookingInfo.UnlockCode)
		assert.NotEmpty(t, bookingInfo.QRCode)
	})

	t.Run("房间不存在创建失败", func(t *testing.T) {
		checkInTime := time.Now().Add(1 * time.Hour)
		req := &CreateBookingRequest{
			RoomID:        999999,
			DurationHours: 2,
			CheckInTime:   checkInTime,
		}

		_, err := svc.CreateBooking(ctx, user.ID, req)
		assert.Error(t, err)
	})

	t.Run("入住时间是过去创建失败", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		req := &CreateBookingRequest{
			RoomID:        room.ID,
			DurationHours: 2,
			CheckInTime:   pastTime,
		}

		_, err := svc.CreateBooking(ctx, user.ID, req)
		assert.Error(t, err)
	})
}

func TestBookingService_CreateBooking_RoomConflict(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, _, room, _ := createTestBookingData(t, svc.db)

	// 首先创建一个预订
	checkInTime := time.Now().Add(1 * time.Hour)
	req1 := &CreateBookingRequest{
		RoomID:        room.ID,
		DurationHours: 2,
		CheckInTime:   checkInTime,
	}
	booking1, err := svc.CreateBooking(ctx, user.ID, req1)
	require.NoError(t, err)

	// 模拟支付成功，更新状态为已支付
	err = svc.db.Model(&models.Booking{}).Where("id = ?", booking1.ID).Update("status", models.BookingStatusPaid).Error
	require.NoError(t, err)

	// 尝试创建时间冲突的预订
	t.Run("时段冲突创建失败", func(t *testing.T) {
		// 创建新用户
		phone2 := "13800138001"
		user2 := &models.User{
			Phone:         &phone2,
			Nickname:      "测试用户2",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		svc.db.Create(user2)

		// 创建完全重叠的时段
		conflictReq := &CreateBookingRequest{
			RoomID:        room.ID,
			DurationHours: 2,
			CheckInTime:   checkInTime,
		}

		_, err := svc.CreateBooking(ctx, user2.ID, conflictReq)
		assert.Error(t, err) // 应该返回冲突错误
	})
}

func TestBookingService_GetBookingByID(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	// 创建预订记录
	order := &models.Order{
		OrderNo:        "TEST123456",
		UserID:         user.ID,
		Type:           models.OrderTypeHotel,
		OriginalAmount: 100.0,
		ActualAmount:   100.0,
		Status:         models.OrderStatusPending,
	}
	svc.db.Create(order)

	checkInTime := time.Now().Add(1 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "B202401010001",
		OrderID:          order.ID,
		UserID:           user.ID,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkInTime,
		CheckOutTime:     checkInTime.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           100.0,
		VerificationCode: "V1234567890123456789",
		UnlockCode:       "123456",
		QRCode:           "/api/v1/hotel/verify/B202401010001?code=V1234567890123456789",
		Status:           models.BookingStatusPaid,
	}
	err := svc.db.Create(booking).Error
	require.NoError(t, err)

	t.Run("获取预订成功", func(t *testing.T) {
		info, err := svc.GetBookingByID(ctx, booking.ID, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, booking.BookingNo, info.BookingNo)
		assert.Equal(t, models.BookingStatusPaid, info.Status)
		// 已支付状态应该显示敏感信息
		assert.NotEmpty(t, info.VerificationCode)
		assert.NotEmpty(t, info.UnlockCode)
	})

	t.Run("获取不属于自己的预订失败", func(t *testing.T) {
		_, err := svc.GetBookingByID(ctx, booking.ID, 999999)
		assert.Error(t, err)
	})

	t.Run("预订不存在", func(t *testing.T) {
		_, err := svc.GetBookingByID(ctx, 999999, user.ID)
		assert.Error(t, err)
	})
}

func TestBookingService_GetUserBookings(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	// 创建多个预订记录
	for i := 0; i < 5; i++ {
		order := &models.Order{
			OrderNo:        "ORDER" + string(rune('A'+i)),
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPending,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(time.Duration(i+1) * time.Hour)
		booking := &models.Booking{
			BookingNo:        "B" + string(rune('A'+i)),
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "V" + string(rune('A'+i)),
			UnlockCode:       "12345" + string(rune('0'+i)),
			QRCode:           "/qr/" + string(rune('A'+i)),
			Status:           models.BookingStatusPending,
		}
		svc.db.Create(booking)
	}

	t.Run("获取用户预订列表", func(t *testing.T) {
		bookings, total, err := svc.GetUserBookings(ctx, user.ID, 1, 10, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, bookings, 5)
	})

	t.Run("分页获取", func(t *testing.T) {
		bookings, total, err := svc.GetUserBookings(ctx, user.ID, 1, 2, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, bookings, 2)
	})

	t.Run("按状态过滤", func(t *testing.T) {
		status := models.BookingStatusPending
		bookings, total, err := svc.GetUserBookings(ctx, user.ID, 1, 10, &status)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, bookings, 5)
		for _, b := range bookings {
			assert.Equal(t, models.BookingStatusPending, b.Status)
		}
	})
}

func TestBookingService_CancelBooking(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	t.Run("取消待支付预订成功", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "CANCEL001",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPending,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BCANCEL001",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VCANCEL001",
			UnlockCode:       "111111",
			QRCode:           "/qr/cancel001",
			Status:           models.BookingStatusPending,
		}
		svc.db.Create(booking)

		err := svc.CancelBooking(ctx, booking.ID, user.ID)
		require.NoError(t, err)

		// 验证状态已更新
		var updatedBooking models.Booking
		svc.db.First(&updatedBooking, booking.ID)
		assert.Equal(t, models.BookingStatusCancelled, updatedBooking.Status)
	})

	t.Run("取消已支付预订失败", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "CANCEL002",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BCANCEL002",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VCANCEL002",
			UnlockCode:       "222222",
			QRCode:           "/qr/cancel002",
			Status:           models.BookingStatusPaid,
		}
		svc.db.Create(booking)

		err := svc.CancelBooking(ctx, booking.ID, user.ID)
		assert.Error(t, err)
	})

	t.Run("取消别人的预订失败", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "CANCEL003",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPending,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BCANCEL003",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VCANCEL003",
			UnlockCode:       "333333",
			QRCode:           "/qr/cancel003",
			Status:           models.BookingStatusPending,
		}
		svc.db.Create(booking)

		err := svc.CancelBooking(ctx, booking.ID, 999999)
		assert.Error(t, err)
	})
}

func TestBookingService_VerifyBooking(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	t.Run("核销已支付预订成功", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "VERIFY001",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(1 * time.Hour)
		verificationCode := "VVERIFY001XXXXXXXXX"
		booking := &models.Booking{
			BookingNo:        "BVERIFY001",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: verificationCode,
			UnlockCode:       "444444",
			QRCode:           "/qr/verify001",
			Status:           models.BookingStatusPaid,
		}
		svc.db.Create(booking)

		adminID := int64(1)
		info, err := svc.VerifyBooking(ctx, verificationCode, adminID)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, models.BookingStatusVerified, info.Status)
		assert.Equal(t, "已核销", info.StatusName)
		assert.NotNil(t, info.VerifiedAt)
	})

	t.Run("核销待支付预订失败", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "VERIFY002",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPending,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(1 * time.Hour)
		verificationCode := "VVERIFY002XXXXXXXXX"
		booking := &models.Booking{
			BookingNo:        "BVERIFY002",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: verificationCode,
			UnlockCode:       "555555",
			QRCode:           "/qr/verify002",
			Status:           models.BookingStatusPending,
		}
		svc.db.Create(booking)

		adminID := int64(1)
		_, err := svc.VerifyBooking(ctx, verificationCode, adminID)
		assert.Error(t, err)
	})

	t.Run("无效核销码", func(t *testing.T) {
		adminID := int64(1)
		_, err := svc.VerifyBooking(ctx, "INVALID_CODE", adminID)
		assert.Error(t, err)
	})
}

func TestBookingService_CompleteBooking(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	t.Run("完成使用中的预订", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "COMPLETE001",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(-1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BCOMPLETE001",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VCOMPLETE001XXXXXXX",
			UnlockCode:       "666666",
			QRCode:           "/qr/complete001",
			Status:           models.BookingStatusInUse,
		}
		svc.db.Create(booking)

		err := svc.CompleteBooking(ctx, booking.ID)
		require.NoError(t, err)

		// 验证状态已更新
		var updatedBooking models.Booking
		svc.db.First(&updatedBooking, booking.ID)
		assert.Equal(t, models.BookingStatusCompleted, updatedBooking.Status)
		assert.NotNil(t, updatedBooking.CompletedAt)
	})

	t.Run("完成已核销的预订", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "COMPLETE002",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(-1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BCOMPLETE002",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VCOMPLETE002XXXXXXX",
			UnlockCode:       "777777",
			QRCode:           "/qr/complete002",
			Status:           models.BookingStatusVerified,
		}
		svc.db.Create(booking)

		err := svc.CompleteBooking(ctx, booking.ID)
		require.NoError(t, err)
	})

	t.Run("完成待支付预订失败", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "COMPLETE003",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPending,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BCOMPLETE003",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VCOMPLETE003XXXXXXX",
			UnlockCode:       "888888",
			QRCode:           "/qr/complete003",
			Status:           models.BookingStatusPending,
		}
		svc.db.Create(booking)

		err := svc.CompleteBooking(ctx, booking.ID)
		assert.Error(t, err)
	})
}

func TestBookingService_OnPaymentSuccess(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	t.Run("支付成功更新预订状态", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "PAY001",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPending,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BPAY001",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VPAY001XXXXXXXXXXXX",
			UnlockCode:       "999999",
			QRCode:           "/qr/pay001",
			Status:           models.BookingStatusPending,
		}
		svc.db.Create(booking)

		err := svc.OnPaymentSuccess(ctx, order.ID)
		require.NoError(t, err)

		// 验证状态已更新
		var updatedBooking models.Booking
		svc.db.First(&updatedBooking, booking.ID)
		assert.Equal(t, models.BookingStatusPaid, updatedBooking.Status)
	})

	t.Run("重复支付回调幂等", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "PAY002",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		svc.db.Create(order)

		checkInTime := time.Now().Add(1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BPAY002",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VPAY002XXXXXXXXXXXX",
			UnlockCode:       "000000",
			QRCode:           "/qr/pay002",
			Status:           models.BookingStatusPaid, // 已经是支付状态
		}
		svc.db.Create(booking)

		// 重复调用不应报错
		err := svc.OnPaymentSuccess(ctx, order.ID)
		require.NoError(t, err)
	})
}

func TestBookingService_GetBookingByNo(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	order := &models.Order{
		OrderNo:        "TEST_GET_BY_NO",
		UserID:         user.ID,
		Type:           models.OrderTypeHotel,
		OriginalAmount: 100.0,
		ActualAmount:   100.0,
		Status:         models.OrderStatusPending,
	}
	svc.db.Create(order)

	checkInTime := time.Now().Add(1 * time.Hour)
	bookingNo := "B_GET_BY_NO"
	booking := &models.Booking{
		BookingNo:        bookingNo,
		OrderID:          order.ID,
		UserID:           user.ID,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkInTime,
		CheckOutTime:     checkInTime.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           100.0,
		VerificationCode: "V_GET_BY_NO_XXXXXXXXX",
		UnlockCode:       "123456",
		QRCode:           "/api/v1/hotel/verify/B_GET_BY_NO?code=V_GET_BY_NO_XXXXXXXXX",
		Status:           models.BookingStatusPaid,
	}
	require.NoError(t, svc.db.Create(booking).Error)

	t.Run("获取预订成功", func(t *testing.T) {
		info, err := svc.GetBookingByNo(ctx, bookingNo, user.ID)
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, bookingNo, info.BookingNo)
		assert.Equal(t, models.BookingStatusPaid, info.Status)
		// 已支付状态应该显示敏感信息
		assert.NotEmpty(t, info.VerificationCode)
		assert.NotEmpty(t, info.UnlockCode)
	})

	t.Run("获取不属于自己的预订失败", func(t *testing.T) {
		_, err := svc.GetBookingByNo(ctx, bookingNo, 999999)
		assert.Error(t, err)
	})

	t.Run("预订不存在", func(t *testing.T) {
		_, err := svc.GetBookingByNo(ctx, "NOT_EXISTS", user.ID)
		assert.Error(t, err)
	})
}

func TestBookingService_UnlockByCode(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)
	deviceID := int64(1)

	// 让 booking.DeviceID 非空，使 UnlockByCode 能匹配到记录
	require.NoError(t, svc.db.Model(&models.Room{}).Where("id = ?", room.ID).Update("device_id", deviceID).Error)

	createBooking := func(t *testing.T, status string, checkIn, checkOut time.Time, unlockCode string) *models.Booking {
		t.Helper()

		order := &models.Order{
			OrderNo:        "UNLOCK_" + status + "_" + unlockCode,
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		require.NoError(t, svc.db.Create(order).Error)

		booking := &models.Booking{
			BookingNo:        "B_UNLOCK_" + status + "_" + unlockCode,
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			DeviceID:         &deviceID,
			CheckInTime:      checkIn,
			CheckOutTime:     checkOut,
			DurationHours:    int(checkOut.Sub(checkIn).Hours()),
			Amount:           100.0,
			VerificationCode: "V_UNLOCK_" + unlockCode + "XXXXXX",
			UnlockCode:       unlockCode,
			QRCode:           "/qr/unlock",
			Status:           status,
		}
		require.NoError(t, svc.db.Create(booking).Error)
		return booking
	}

	t.Run("开锁码格式不正确", func(t *testing.T) {
		_, err := svc.UnlockByCode(ctx, deviceID, "bad")
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrUnlockCodeInvalid.Code, appErr.Code)
	})

	t.Run("找不到对应预订返回开锁码无效", func(t *testing.T) {
		_, err := svc.UnlockByCode(ctx, deviceID, "123456")
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrUnlockCodeInvalid.Code, appErr.Code)
	})

	t.Run("已开锁状态返回已开锁", func(t *testing.T) {
		checkIn := time.Now().Add(-time.Hour)
		checkOut := time.Now().Add(time.Hour)
		createBooking(t, models.BookingStatusInUse, checkIn, checkOut, "111111")

		_, err := svc.UnlockByCode(ctx, deviceID, "111111")
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrBookingStatusError.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "已开锁")
	})

	t.Run("未到入住时间", func(t *testing.T) {
		checkIn := time.Now().Add(2 * time.Hour)
		checkOut := time.Now().Add(3 * time.Hour)
		createBooking(t, models.BookingStatusVerified, checkIn, checkOut, "222222")

		_, err := svc.UnlockByCode(ctx, deviceID, "222222")
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrBookingTimeNotArrived.Code, appErr.Code)
	})

	t.Run("超过退房时间开锁码过期", func(t *testing.T) {
		checkIn := time.Now().Add(-3 * time.Hour)
		checkOut := time.Now().Add(-1 * time.Hour)
		createBooking(t, models.BookingStatusVerified, checkIn, checkOut, "333333")

		_, err := svc.UnlockByCode(ctx, deviceID, "333333")
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrUnlockCodeExpired.Code, appErr.Code)
	})

	t.Run("开锁成功更新为使用中", func(t *testing.T) {
		checkIn := time.Now().Add(-time.Hour)
		checkOut := time.Now().Add(time.Hour)
		booking := createBooking(t, models.BookingStatusVerified, checkIn, checkOut, "444444")

		info, err := svc.UnlockByCode(ctx, deviceID, "444444")
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, models.BookingStatusInUse, info.Status)

		var updated models.Booking
		require.NoError(t, svc.db.First(&updated, booking.ID).Error)
		assert.Equal(t, models.BookingStatusInUse, updated.Status)
		assert.NotNil(t, updated.UnlockedAt)
	})
}

func TestBookingService_ProcessExpiredBookings(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	createPaidBooking := func(t *testing.T, checkIn time.Time) *models.Booking {
		t.Helper()

		order := &models.Order{
			OrderNo:        "EXPIRED_" + checkIn.Format(time.RFC3339Nano),
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		require.NoError(t, svc.db.Create(order).Error)

		booking := &models.Booking{
			BookingNo:        "B_EXPIRED_" + order.OrderNo,
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkIn,
			CheckOutTime:     checkIn.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "V_EXPIRED_XXXXXXXXXXX",
			UnlockCode:       "555555",
			QRCode:           "/qr/expired",
			Status:           models.BookingStatusPaid,
		}
		require.NoError(t, svc.db.Create(booking).Error)
		return booking
	}

	expired := createPaidBooking(t, time.Now().Add(-2*time.Hour))
	notExpired := createPaidBooking(t, time.Now().Add(2*time.Hour))

	require.NoError(t, svc.ProcessExpiredBookings(ctx))

	var gotExpired models.Booking
	require.NoError(t, svc.db.First(&gotExpired, expired.ID).Error)
	assert.Equal(t, models.BookingStatusExpired, gotExpired.Status)

	var gotNotExpired models.Booking
	require.NoError(t, svc.db.First(&gotNotExpired, notExpired.ID).Error)
	assert.Equal(t, models.BookingStatusPaid, gotNotExpired.Status)
}

func TestBookingService_ProcessCompletedBookings(t *testing.T) {
	svc := setupTestBookingService(t)
	ctx := context.Background()

	user, hotel, room, _ := createTestBookingData(t, svc.db)

	createToComplete := func(t *testing.T, status string) *models.Booking {
		t.Helper()

		order := &models.Order{
			OrderNo:        "TO_COMPLETE_" + status + "_" + time.Now().Format(time.RFC3339Nano),
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		require.NoError(t, svc.db.Create(order).Error)

		checkIn := time.Now().Add(-3 * time.Hour)
		checkOut := time.Now().Add(-1 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "B_TO_COMPLETE_" + order.OrderNo,
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkIn,
			CheckOutTime:     checkOut,
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "V_TO_COMPLETE_XXXXXXXX",
			UnlockCode:       "666666",
			QRCode:           "/qr/to_complete",
			Status:           status,
		}
		require.NoError(t, svc.db.Create(booking).Error)
		return booking
	}

	verified := createToComplete(t, models.BookingStatusVerified)
	inUse := createToComplete(t, models.BookingStatusInUse)

	require.NoError(t, svc.ProcessCompletedBookings(ctx))

	var gotVerified models.Booking
	require.NoError(t, svc.db.First(&gotVerified, verified.ID).Error)
	assert.Equal(t, models.BookingStatusCompleted, gotVerified.Status)
	assert.NotNil(t, gotVerified.CompletedAt)

	var gotInUse models.Booking
	require.NoError(t, svc.db.First(&gotInUse, inUse.ID).Error)
	assert.Equal(t, models.BookingStatusCompleted, gotInUse.Status)
	assert.NotNil(t, gotInUse.CompletedAt)
}

func TestBookingService_getStatusName(t *testing.T) {
	svc := setupTestBookingService(t)

	tests := []struct {
		status   string
		expected string
	}{
		{models.BookingStatusPending, "待支付"},
		{models.BookingStatusPaid, "待核销"},
		{models.BookingStatusVerified, "已核销"},
		{models.BookingStatusInUse, "使用中"},
		{models.BookingStatusCompleted, "已完成"},
		{models.BookingStatusCancelled, "已取消"},
		{models.BookingStatusRefunded, "已退款"},
		{models.BookingStatusExpired, "已过期"},
		{"unknown", "未知"},
	}

	for _, tt := range tests {
		name := svc.getStatusName(tt.status)
		assert.Equal(t, tt.expected, name)
	}
}
