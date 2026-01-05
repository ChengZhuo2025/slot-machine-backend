// Package e2e 酒店预订完整流程 E2E 测试
package e2e

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

// hotelE2ETestContext E2E测试上下文
type hotelE2ETestContext struct {
	db             *gorm.DB
	bookingService *hotelService.BookingService
	hotelService   *hotelService.HotelService
	codeService    *hotelService.CodeService
}

// setupHotelE2ETestDB 创建E2E测试数据库
func setupHotelE2ETestDB(t *testing.T) *gorm.DB {
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

	// 初始化基础数据
	level := &models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0}
	db.Create(level)

	return db
}

// setupHotelE2ETestContext 创建E2E测试上下文
func setupHotelE2ETestContext(t *testing.T) *hotelE2ETestContext {
	db := setupHotelE2ETestDB(t)

	bookingRepo := repository.NewBookingRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	hotelRepo := repository.NewHotelRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	timeSlotRepo := repository.NewRoomTimeSlotRepository(db)

	codeService := hotelService.NewCodeService()
	bookingSvc := hotelService.NewBookingService(db, bookingRepo, roomRepo, hotelRepo, orderRepo, timeSlotRepo, codeService, nil, nil)
	hotelSvc := hotelService.NewHotelService(db, hotelRepo, roomRepo, timeSlotRepo)

	return &hotelE2ETestContext{
		db:             db,
		bookingService: bookingSvc,
		hotelService:   hotelSvc,
		codeService:    codeService,
	}
}

// createE2EHotelWithRooms 创建酒店和房间
func createE2EHotelWithRooms(t *testing.T, db *gorm.DB) (*models.Hotel, []*models.Room) {
	// 创建酒店
	description := "E2E测试酒店 - 深圳南山科技园"
	starRating := 4
	longitude := 114.0579
	latitude := 22.5431
	hotel := &models.Hotel{
		Name:           "E2E测试酒店",
		StarRating:     &starRating,
		Province:       "广东省",
		City:           "深圳市",
		District:       "南山区",
		Address:        "科技园路1号",
		Longitude:      &longitude,
		Latitude:       &latitude,
		Phone:          "0755-12345678",
		Description:    &description,
		CheckInTime:    "14:00",
		CheckOutTime:   "12:00",
		CommissionRate: 0.15,
		Status:         models.HotelStatusActive,
	}
	db.Create(hotel)

	// 创建多个房间
	roomTypes := []struct {
		roomNo      string
		roomType    string
		hourlyPrice float64
		dailyPrice  float64
	}{
		{"101", models.RoomTypeStandard, 60.0, 288.0},
		{"102", models.RoomTypeStandard, 60.0, 288.0},
		{"201", models.RoomTypeBusiness, 80.0, 388.0},
		{"301", models.RoomTypeDeluxe, 120.0, 588.0},
	}

	var rooms []*models.Room
	for _, rt := range roomTypes {
		area := 25
		bedType := "大床"
		room := &models.Room{
			HotelID:     hotel.ID,
			RoomNo:      rt.roomNo,
			RoomType:    rt.roomType,
			Area:        &area,
			BedType:     &bedType,
			MaxGuests:   2,
			HourlyPrice: rt.hourlyPrice,
			DailyPrice:  rt.dailyPrice,
			Status:      models.RoomStatusActive,
		}
		db.Create(room)
		rooms = append(rooms, room)

		// 创建时段价格
		timeSlots := []struct {
			duration int
			price    float64
		}{
			{2, rt.hourlyPrice * 2 * 0.9},
			{4, rt.hourlyPrice * 4 * 0.85},
			{6, rt.hourlyPrice * 6 * 0.8},
		}
		for _, ts := range timeSlots {
			slot := &models.RoomTimeSlot{
				RoomID:        room.ID,
				DurationHours: ts.duration,
				Price:         ts.price,
				IsActive:      true,
				Sort:          ts.duration,
			}
			db.Create(slot)
		}
	}

	return hotel, rooms
}

// createE2EUser 创建测试用户
func createE2EUser(t *testing.T, db *gorm.DB, phone string, balance float64) *models.User {
	user := &models.User{
		Phone:         &phone,
		Nickname:      "E2E测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: balance,
	}
	db.Create(wallet)

	return user
}

// createE2EAdmin 创建测试管理员
func createE2EAdmin(t *testing.T, db *gorm.DB) *models.Admin {
	description := "酒店前台工作人员"
	role := &models.Role{Code: "hotel_frontdesk", Name: "酒店前台", Description: &description}
	db.Create(role)

	admin := &models.Admin{
		Username:     "hotel_staff",
		PasswordHash: "hashedpassword",
		Name:         "前台小王",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	return admin
}

// TestE2E_HotelBookingCompleteFlow 测试完整的酒店预订业务流程
func TestE2E_HotelBookingCompleteFlow(t *testing.T) {
	tc := setupHotelE2ETestContext(t)
	ctx := context.Background()

	// 准备测试数据
	hotel, rooms := createE2EHotelWithRooms(t, tc.db)
	user := createE2EUser(t, tc.db, "13800138000", 1000.0)
	admin := createE2EAdmin(t, tc.db)

	t.Run("场景1: 用户浏览酒店并预订房间", func(t *testing.T) {
		// Step 1: 用户查看酒店列表
		hotelList, total, err := tc.hotelService.GetHotelList(ctx, &hotelService.HotelListRequest{
			Page:     1,
			PageSize: 10,
			City:     "深圳市",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, hotel.Name, hotelList[0].Name)
		t.Logf("Step 1: 用户查看酒店列表，找到 %d 家酒店", total)

		// Step 2: 用户查看酒店详情
		hotelDetail, err := tc.hotelService.GetHotelDetail(ctx, hotel.ID)
		require.NoError(t, err)
		assert.Equal(t, hotel.Name, hotelDetail.Name)
		t.Logf("Step 2: 用户查看酒店详情: %s", hotelDetail.Name)

		// Step 3: 用户查看房间列表
		roomList, err := tc.hotelService.GetRoomList(ctx, hotel.ID)
		require.NoError(t, err)
		assert.Len(t, roomList, 4)
		t.Logf("Step 3: 酒店有 %d 个房间可选", len(roomList))

		// Step 4: 用户选择房间并查看详情
		selectedRoom := rooms[0] // 选择101房间
		roomDetail, err := tc.hotelService.GetRoomDetail(ctx, selectedRoom.ID)
		require.NoError(t, err)
		assert.Equal(t, selectedRoom.RoomNo, roomDetail.RoomNo)
		assert.Len(t, roomDetail.TimeSlots, 3)
		t.Logf("Step 4: 用户选择房间 %s，有 %d 个时段价格", roomDetail.RoomNo, len(roomDetail.TimeSlots))

		// Step 5: 检查房间可用性
		checkInTime := time.Now().Add(2 * time.Hour)
		checkOutTime := checkInTime.Add(2 * time.Hour)
		available, err := tc.hotelService.CheckRoomAvailability(ctx, selectedRoom.ID, checkInTime, checkOutTime)
		require.NoError(t, err)
		assert.True(t, available)
		t.Logf("Step 5: 房间 %s 在 %s 至 %s 可用", selectedRoom.RoomNo, checkInTime.Format("15:04"), checkOutTime.Format("15:04"))

		// Step 6: 创建预订
		booking, err := tc.bookingService.CreateBooking(ctx, user.ID, &hotelService.CreateBookingRequest{
			RoomID:        selectedRoom.ID,
			DurationHours: 2,
			CheckInTime:   checkInTime,
		})
		require.NoError(t, err)
		assert.Equal(t, models.BookingStatusPending, booking.Status)
		t.Logf("Step 6: 创建预订成功，预订号: %s，金额: %.2f", booking.BookingNo, booking.Amount)

		// Step 7: 支付成功回调
		err = tc.bookingService.OnPaymentSuccess(ctx, booking.ID)
		require.NoError(t, err)
		t.Logf("Step 7: 支付成功")

		// Step 8: 用户查看预订详情
		bookingDetail, err := tc.bookingService.GetBookingByID(ctx, booking.ID, user.ID)
		require.NoError(t, err)
		assert.Equal(t, models.BookingStatusPaid, bookingDetail.Status)
		assert.NotEmpty(t, bookingDetail.VerificationCode)
		assert.NotEmpty(t, bookingDetail.UnlockCode)
		t.Logf("Step 8: 预订详情 - 核销码: %s, 开锁码: %s", bookingDetail.VerificationCode, bookingDetail.UnlockCode)

		// Step 9: 酒店前台核销
		verifiedBooking, err := tc.bookingService.VerifyBooking(ctx, bookingDetail.VerificationCode, admin.ID)
		require.NoError(t, err)
		assert.Equal(t, models.BookingStatusVerified, verifiedBooking.Status)
		t.Logf("Step 9: 前台核销成功，核销人: %s", admin.Name)

		// Step 10: 完成预订
		err = tc.bookingService.CompleteBooking(ctx, booking.ID)
		require.NoError(t, err)

		// 验证最终状态
		var finalBooking models.Booking
		tc.db.First(&finalBooking, booking.ID)
		assert.Equal(t, models.BookingStatusCompleted, finalBooking.Status)
		assert.NotNil(t, finalBooking.CompletedAt)
		t.Logf("Step 10: 预订完成，状态: %s", finalBooking.Status)
	})

	t.Run("场景2: 用户取消未支付的预订", func(t *testing.T) {
		checkInTime := time.Now().Add(24 * time.Hour)
		booking, err := tc.bookingService.CreateBooking(ctx, user.ID, &hotelService.CreateBookingRequest{
			RoomID:        rooms[1].ID, // 使用102房间
			DurationHours: 2,
			CheckInTime:   checkInTime,
		})
		require.NoError(t, err)
		t.Logf("创建预订: %s", booking.BookingNo)

		// 取消预订
		err = tc.bookingService.CancelBooking(ctx, booking.ID, user.ID)
		require.NoError(t, err)

		// 验证取消状态
		var cancelled models.Booking
		tc.db.First(&cancelled, booking.ID)
		assert.Equal(t, models.BookingStatusCancelled, cancelled.Status)
		t.Logf("预订已取消: %s", cancelled.Status)

		// 验证房间仍可预订
		available, err := tc.hotelService.CheckRoomAvailability(ctx, rooms[1].ID, checkInTime, checkInTime.Add(2*time.Hour))
		require.NoError(t, err)
		assert.True(t, available, "取消后房间应该可用")
	})

	t.Run("场景3: 多用户同时预订同一房间", func(t *testing.T) {
		user2 := createE2EUser(t, tc.db, "13900139000", 1000.0)
		checkInTime := time.Now().Add(48 * time.Hour)
		targetRoom := rooms[2] // 使用201房间

		// 用户1预订
		booking1, err := tc.bookingService.CreateBooking(ctx, user.ID, &hotelService.CreateBookingRequest{
			RoomID:        targetRoom.ID,
			DurationHours: 4,
			CheckInTime:   checkInTime,
		})
		require.NoError(t, err)

		// 模拟用户1支付
		err = tc.bookingService.OnPaymentSuccess(ctx, booking1.ID)
		require.NoError(t, err)
		t.Logf("用户1预订成功并支付: %s", booking1.BookingNo)

		// 用户2尝试预订相同时段
		_, err = tc.bookingService.CreateBooking(ctx, user2.ID, &hotelService.CreateBookingRequest{
			RoomID:        targetRoom.ID,
			DurationHours: 4,
			CheckInTime:   checkInTime,
		})
		assert.Error(t, err, "用户2应该无法预订已被占用的时段")
		t.Logf("用户2预订失败（预期行为）: 时段已被占用")

		// 用户2预订不同时段
		differentTime := checkInTime.Add(5 * time.Hour)
		booking2, err := tc.bookingService.CreateBooking(ctx, user2.ID, &hotelService.CreateBookingRequest{
			RoomID:        targetRoom.ID,
			DurationHours: 2,
			CheckInTime:   differentTime,
		})
		require.NoError(t, err)
		t.Logf("用户2预订不同时段成功: %s", booking2.BookingNo)
	})

	t.Run("场景4: 用户查看预订历史", func(t *testing.T) {
		// 获取用户所有预订
		bookings, total, err := tc.bookingService.GetUserBookings(ctx, user.ID, 1, 10, nil)
		require.NoError(t, err)
		t.Logf("用户共有 %d 个预订记录", total)
		assert.Greater(t, total, int64(0))

		// 按状态过滤
		completedStatus := models.BookingStatusCompleted
		completedBookings, completedTotal, err := tc.bookingService.GetUserBookings(ctx, user.ID, 1, 10, &completedStatus)
		require.NoError(t, err)
		t.Logf("其中已完成的预订: %d 个", completedTotal)

		for _, b := range bookings {
			t.Logf("- %s: %s (%.2f元)", b.BookingNo, b.StatusName, b.Amount)
		}
		_ = completedBookings // 使用变量避免编译警告
	})
}

// TestE2E_HotelSearchAndFilter 测试酒店搜索和筛选功能
func TestE2E_HotelSearchAndFilter(t *testing.T) {
	tc := setupHotelE2ETestContext(t)
	ctx := context.Background()

	// 创建多个酒店
	cities := []string{"深圳市", "广州市", "东莞市"}
	for i, city := range cities {
		description := "测试酒店描述"
		starRating := 3 + i
		hotel := &models.Hotel{
			Name:         "测试酒店" + city,
			StarRating:   &starRating,
			Province:     "广东省",
			City:         city,
			District:     "中心区",
			Address:      "测试路1号",
			Phone:        "0755-12345678",
			Description:  &description,
			CheckInTime:  "14:00",
			CheckOutTime: "12:00",
			Status:       models.HotelStatusActive,
		}
		tc.db.Create(hotel)
	}

	t.Run("按城市筛选", func(t *testing.T) {
		hotels, total, err := tc.hotelService.GetHotelList(ctx, &hotelService.HotelListRequest{
			Page:     1,
			PageSize: 10,
			City:     "深圳市",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "深圳市", hotels[0].City)
	})

	t.Run("关键词搜索", func(t *testing.T) {
		hotels, _, err := tc.hotelService.GetHotelList(ctx, &hotelService.HotelListRequest{
			Page:     1,
			PageSize: 10,
			Keyword:  "广州",
		})
		require.NoError(t, err)
		assert.Len(t, hotels, 1)
		assert.Contains(t, hotels[0].Name, "广州")
	})

	t.Run("获取城市列表", func(t *testing.T) {
		cityList, err := tc.hotelService.GetCities(ctx)
		require.NoError(t, err)
		assert.Len(t, cityList, 3)
		for _, city := range cities {
			assert.Contains(t, cityList, city)
		}
	})
}
