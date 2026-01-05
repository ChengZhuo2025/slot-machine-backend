// Package integration 酒店预订流程集成测试
package integration

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
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	hotelService "github.com/dumeirei/smart-locker-backend/internal/service/hotel"
)

// setupHotelTestDB 创建酒店测试数据库
func setupHotelTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Admin{},
		&models.Role{},
		&models.Order{},
		&models.Hotel{},
		&models.Room{},
		&models.RoomTimeSlot{},
		&models.Booking{},
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

// hotelTestContext 酒店测试上下文
type hotelTestContext struct {
	db             *gorm.DB
	bookingService *hotelService.BookingService
	hotelService   *hotelService.HotelService
	codeService    *hotelService.CodeService
	user           *models.User
	admin          *models.Admin
	hotel          *models.Hotel
	room           *models.Room
	timeSlot       *models.RoomTimeSlot
}

// setupHotelTestContext 创建酒店测试上下文
func setupHotelTestContext(t *testing.T) *hotelTestContext {
	db := setupHotelTestDB(t)

	// 创建仓储
	bookingRepo := repository.NewBookingRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	hotelRepo := repository.NewHotelRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	timeSlotRepo := repository.NewRoomTimeSlotRepository(db)

	// 创建服务
	codeService := hotelService.NewCodeService()
	bookingSvc := hotelService.NewBookingService(db, bookingRepo, roomRepo, hotelRepo, orderRepo, timeSlotRepo, codeService, nil, nil)
	hotelSvc := hotelService.NewHotelService(db, hotelRepo, roomRepo, timeSlotRepo)

	// 创建测试用户
	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 1000.0,
	}
	db.Create(wallet)

		// 创建角色
		roleDesc := "酒店前台"
		role := &models.Role{
			Code:        "hotel_frontdesk",
			Name:        "前台",
			Description: &roleDesc,
		}
		db.Create(role)

		// 创建管理员（前台）
		admin := &models.Admin{
			Username:     "admin001",
			PasswordHash: "hashedpassword",
			Name:         "前台人员",
			RoleID:       role.ID,
			Status:       models.AdminStatusActive,
		}
		db.Create(admin)

	// 创建酒店
	description := "测试酒店描述"
	starRating := 4
	hotel := &models.Hotel{
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
	db.Create(hotel)

	// 创建房间
	area := 25
	bedType := "大床"
	room := &models.Room{
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
	db.Create(room)

	// 创建时段价格
	timeSlot := &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         100.0,
		IsActive:      true,
		Sort:          1,
	}
	db.Create(timeSlot)

	return &hotelTestContext{
		db:             db,
		bookingService: bookingSvc,
		hotelService:   hotelSvc,
		codeService:    codeService,
		user:           user,
		admin:          admin,
		hotel:          hotel,
		room:           room,
		timeSlot:       timeSlot,
	}
}

// TestHotelBookingFlow_CreateToComplete 测试预订创建到完成的完整流程
func TestHotelBookingFlow_CreateToComplete(t *testing.T) {
	tc := setupHotelTestContext(t)
	ctx := context.Background()

	t.Run("完整预订流程：创建 -> 支付 -> 核销 -> 完成", func(t *testing.T) {
		// 1. 创建预订
		checkInTime := time.Now().Add(1 * time.Hour)
		createReq := &hotelService.CreateBookingRequest{
			RoomID:        tc.room.ID,
			DurationHours: 2,
			CheckInTime:   checkInTime,
		}

		booking, err := tc.bookingService.CreateBooking(ctx, tc.user.ID, createReq)
		require.NoError(t, err)
		assert.NotNil(t, booking)
		assert.Equal(t, models.BookingStatusPending, booking.Status)
		assert.NotEmpty(t, booking.BookingNo)
		assert.NotEmpty(t, booking.VerificationCode)
		assert.NotEmpty(t, booking.UnlockCode)
		assert.Equal(t, tc.timeSlot.Price, booking.Amount)

		t.Logf("预订创建成功: BookingNo=%s, Amount=%.2f", booking.BookingNo, booking.Amount)

		// 2. 模拟支付成功（通过订单ID触发）
		err = tc.bookingService.OnPaymentSuccess(ctx, booking.ID)
		require.NoError(t, err)

		// 验证状态已更新为已支付
		bookingAfterPay, err := tc.bookingService.GetBookingByID(ctx, booking.ID, tc.user.ID)
		require.NoError(t, err)
		assert.Equal(t, models.BookingStatusPaid, bookingAfterPay.Status)
		// 已支付状态应该显示核销码和开锁码
		assert.NotEmpty(t, bookingAfterPay.VerificationCode)
		assert.NotEmpty(t, bookingAfterPay.UnlockCode)

		t.Logf("支付成功: Status=%s, VerificationCode=%s", bookingAfterPay.Status, bookingAfterPay.VerificationCode)

		// 3. 酒店前台核销
		verifiedBooking, err := tc.bookingService.VerifyBooking(ctx, bookingAfterPay.VerificationCode, tc.admin.ID)
		require.NoError(t, err)
		assert.Equal(t, models.BookingStatusVerified, verifiedBooking.Status)
		assert.NotNil(t, verifiedBooking.VerifiedAt)

		t.Logf("核销成功: Status=%s, VerifiedAt=%v", verifiedBooking.Status, verifiedBooking.VerifiedAt)

		// 4. 完成预订
		err = tc.bookingService.CompleteBooking(ctx, booking.ID)
		require.NoError(t, err)

		// 验证最终状态
		var finalBooking models.Booking
		tc.db.First(&finalBooking, booking.ID)
		assert.Equal(t, models.BookingStatusCompleted, finalBooking.Status)
		assert.NotNil(t, finalBooking.CompletedAt)

		t.Logf("预订完成: Status=%s, CompletedAt=%v", finalBooking.Status, finalBooking.CompletedAt)
	})
}

// TestHotelBookingFlow_CancelBeforePayment 测试支付前取消预订
func TestHotelBookingFlow_CancelBeforePayment(t *testing.T) {
	tc := setupHotelTestContext(t)
	ctx := context.Background()

	t.Run("支付前取消预订", func(t *testing.T) {
		// 1. 创建预订
		checkInTime := time.Now().Add(1 * time.Hour)
		createReq := &hotelService.CreateBookingRequest{
			RoomID:        tc.room.ID,
			DurationHours: 2,
			CheckInTime:   checkInTime,
		}

		booking, err := tc.bookingService.CreateBooking(ctx, tc.user.ID, createReq)
		require.NoError(t, err)
		assert.Equal(t, models.BookingStatusPending, booking.Status)

		// 2. 取消预订
		err = tc.bookingService.CancelBooking(ctx, booking.ID, tc.user.ID)
		require.NoError(t, err)

		// 验证状态
		var cancelledBooking models.Booking
		tc.db.First(&cancelledBooking, booking.ID)
		assert.Equal(t, models.BookingStatusCancelled, cancelledBooking.Status)

		t.Logf("预订已取消: Status=%s", cancelledBooking.Status)
	})
}

// TestHotelBookingFlow_RoomConflict 测试房间时段冲突
func TestHotelBookingFlow_RoomConflict(t *testing.T) {
	tc := setupHotelTestContext(t)
	ctx := context.Background()

	t.Run("同一时段不能重复预订", func(t *testing.T) {
		// 1. 第一个用户创建预订
		checkInTime := time.Now().Add(10 * time.Hour)
		createReq := &hotelService.CreateBookingRequest{
			RoomID:        tc.room.ID,
			DurationHours: 2,
			CheckInTime:   checkInTime,
		}

		booking1, err := tc.bookingService.CreateBooking(ctx, tc.user.ID, createReq)
		require.NoError(t, err)

		// 模拟支付成功
		err = tc.bookingService.OnPaymentSuccess(ctx, booking1.ID)
		require.NoError(t, err)

		// 2. 创建第二个用户
		phone2 := "13800138001"
		user2 := &models.User{
			Phone:         &phone2,
			Nickname:      "测试用户2",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		tc.db.Create(user2)

		// 3. 第二个用户尝试预订同一时段
		_, err = tc.bookingService.CreateBooking(ctx, user2.ID, createReq)
		assert.Error(t, err, "同一时段应该无法预订")

		t.Logf("时段冲突检测成功")
	})
}

// TestHotelBookingFlow_MultipleBookings 测试用户预订列表
func TestHotelBookingFlow_MultipleBookings(t *testing.T) {
	tc := setupHotelTestContext(t)
	ctx := context.Background()

	t.Run("获取用户预订列表", func(t *testing.T) {
		// 创建多个预订（不同时段）
		for i := 0; i < 3; i++ {
			checkInTime := time.Now().Add(time.Duration(24+i*3) * time.Hour)
			createReq := &hotelService.CreateBookingRequest{
				RoomID:        tc.room.ID,
				DurationHours: 2,
				CheckInTime:   checkInTime,
			}

			_, err := tc.bookingService.CreateBooking(ctx, tc.user.ID, createReq)
			require.NoError(t, err)
		}

		// 获取预订列表
		bookings, total, err := tc.bookingService.GetUserBookings(ctx, tc.user.ID, 1, 10, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, bookings, 3)

		t.Logf("用户预订列表: total=%d", total)
	})
}

// TestHotelService_Integration 测试酒店服务集成
func TestHotelService_Integration(t *testing.T) {
	tc := setupHotelTestContext(t)
	ctx := context.Background()

	t.Run("获取酒店和房间信息", func(t *testing.T) {
		// 获取酒店详情
		hotelInfo, err := tc.hotelService.GetHotelDetail(ctx, tc.hotel.ID)
		require.NoError(t, err)
		assert.Equal(t, tc.hotel.Name, hotelInfo.Name)

		// 获取房间列表
		rooms, err := tc.hotelService.GetRoomList(ctx, tc.hotel.ID)
		require.NoError(t, err)
		assert.Len(t, rooms, 1)

		// 获取房间详情
		roomInfo, err := tc.hotelService.GetRoomDetail(ctx, tc.room.ID)
		require.NoError(t, err)
		assert.Equal(t, tc.room.RoomNo, roomInfo.RoomNo)
		assert.Len(t, roomInfo.TimeSlots, 1)

		// 检查房间可用性
		checkIn := time.Now().Add(1 * time.Hour)
		checkOut := checkIn.Add(2 * time.Hour)
		available, err := tc.hotelService.CheckRoomAvailability(ctx, tc.room.ID, checkIn, checkOut)
		require.NoError(t, err)
		assert.True(t, available)

		t.Logf("酒店服务集成测试通过")
	})
}

// TestCodeService_Integration 测试验证码服务集成
func TestCodeService_IntegrationFlow(t *testing.T) {
	tc := setupHotelTestContext(t)

	t.Run("验证码生成和验证", func(t *testing.T) {
		// 生成核销码
		verificationCode := tc.codeService.GenerateVerificationCode()
		assert.True(t, tc.codeService.ValidateVerificationCode(verificationCode))

		// 生成开锁码
		unlockCode := tc.codeService.GenerateUnlockCode()
		assert.True(t, tc.codeService.ValidateUnlockCode(unlockCode))

		// 检查时段有效性
		checkInTime := time.Now().Add(-1 * time.Hour)
		checkOutTime := time.Now().Add(1 * time.Hour)
		isValid := tc.codeService.IsUnlockCodeValid(checkInTime, checkOutTime)
		assert.True(t, isValid)

		t.Logf("验证码服务集成测试通过")
	})
}
