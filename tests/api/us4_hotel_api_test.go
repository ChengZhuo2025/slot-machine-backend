//go:build api
// +build api

// Package api 酒店预订 API 测试
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	hotelHandler "github.com/dumeirei/smart-locker-backend/internal/handler/hotel"
	userMiddleware "github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	hotelService "github.com/dumeirei/smart-locker-backend/internal/service/hotel"
)

func setupUS4APIRouter(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)

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

	// 默认会员等级
	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-us4-api",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	// 创建 repositories
	bookingRepo := repository.NewBookingRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	hotelRepo := repository.NewHotelRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	timeSlotRepo := repository.NewRoomTimeSlotRepository(db)

	// 创建 services
	codeService := hotelService.NewCodeService()
	hotelSvc := hotelService.NewHotelService(db, hotelRepo, roomRepo, timeSlotRepo)
	bookingSvc := hotelService.NewBookingService(db, bookingRepo, roomRepo, hotelRepo, orderRepo, timeSlotRepo, codeService, nil, nil)

	// 创建 handlers
	hotelH := hotelHandler.NewHandler(hotelSvc)
	bookingH := hotelHandler.NewBookingHandler(bookingSvc)

	v1 := r.Group("/api/v1")
	{
		// 公开接口 - 注意：静态路由必须在动态路由之前
		v1.GET("/hotels", hotelH.GetHotelList)
		v1.GET("/hotels/cities", hotelH.GetCities)

			// 酒店路由组
			hotels := v1.Group("/hotels")
			{
				hotels.GET("/:id", hotelH.GetHotelDetail)
				hotels.GET("/:id/rooms", hotelH.GetRoomList)
			}

		// 房间路由
		v1.GET("/rooms/:id", hotelH.GetRoomDetail)
		v1.GET("/rooms/:id/availability", hotelH.CheckRoomAvailability)
		v1.GET("/rooms/:id/time-slots", hotelH.GetRoomTimeSlots)

		// 需要认证的接口
		user := v1.Group("")
		user.Use(userMiddleware.UserAuth(jwtManager))
		{
			user.POST("/bookings", bookingH.CreateBooking)
			user.GET("/bookings", bookingH.GetMyBookings)
			user.GET("/bookings/:id", bookingH.GetBookingDetail)
			user.GET("/bookings/no/:booking_no", bookingH.GetBookingByNo)
			user.POST("/bookings/:id/cancel", bookingH.CancelBooking)
			user.POST("/bookings/unlock", bookingH.UnlockByCode)
		}
	}

	return r, db, jwtManager
}

func seedUS4TestData(t *testing.T, db *gorm.DB) (*models.User, *models.Hotel, *models.Room, *models.RoomTimeSlot) {
	t.Helper()

	// 创建用户
	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&models.UserWallet{UserID: user.ID, Balance: 1000.0}).Error)

	// 创建酒店
	description := "API测试酒店"
	starRating := 4
	longitude := 114.0579
	latitude := 22.5431
	hotel := &models.Hotel{
		Name:           "API测试酒店",
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
	require.NoError(t, db.Create(hotel).Error)

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
	require.NoError(t, db.Create(room).Error)

	// 创建时段价格
	timeSlot := &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 2,
		Price:         100.0,
		IsActive:      true,
		Sort:          1,
	}
	require.NoError(t, db.Create(timeSlot).Error)

	return user, hotel, room, timeSlot
}

// ==================== 酒店 API 测试 ====================

func TestUS4API_GetHotelList(t *testing.T) {
	router, db, _ := setupUS4APIRouter(t)
	_, _, _, _ = seedUS4TestData(t, db)

	// 创建更多酒店
	for i := 0; i < 3; i++ {
		starRating := 3 + i
		hotel := &models.Hotel{
			Name:         "测试酒店" + strconv.Itoa(i),
			StarRating:   &starRating,
			Province:     "广东省",
			City:         "广州市",
			District:     "天河区",
			Address:      "测试路" + strconv.Itoa(i) + "号",
			Phone:        "020-12345678",
			CheckInTime:  "14:00",
			CheckOutTime: "12:00",
			Status:       models.HotelStatusActive,
		}
		db.Create(hotel)
	}

	t.Run("获取酒店列表", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/hotels?page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.GreaterOrEqual(t, len(list), 4)
	})

	t.Run("按城市筛选", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/hotels?city=广州市&page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.Len(t, list, 3)
	})

	t.Run("关键词搜索", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/hotels?keyword=API&page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.Len(t, list, 1)
	})
}

func TestUS4API_GetHotelDetail(t *testing.T) {
	router, db, _ := setupUS4APIRouter(t)
	_, hotel, _, _ := seedUS4TestData(t, db)

	t.Run("获取酒店详情成功", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/hotels/"+strconv.FormatInt(hotel.ID, 10), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "API测试酒店", data["name"])
		assert.Equal(t, "深圳市", data["city"])
	})

	t.Run("酒店不存在", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/hotels/99999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, float64(0), resp["code"])
	})
}

func TestUS4API_GetCities(t *testing.T) {
	router, db, _ := setupUS4APIRouter(t)
	seedUS4TestData(t, db)

	// 创建不同城市的酒店
	cities := []string{"广州市", "东莞市"}
	for _, city := range cities {
		hotel := &models.Hotel{
			Name:         "测试酒店" + city,
			Province:     "广东省",
			City:         city,
			District:     "中心区",
			Address:      "测试路1号",
			Phone:        "020-12345678",
			CheckInTime:  "14:00",
			CheckOutTime: "12:00",
			Status:       models.HotelStatusActive,
		}
		db.Create(hotel)
	}

	req, _ := http.NewRequest("GET", "/api/v1/hotels/cities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].([]interface{})
	assert.Len(t, data, 3) // 深圳、广州、东莞
}

func TestUS4API_GetRoomList(t *testing.T) {
	router, db, _ := setupUS4APIRouter(t)
	_, hotel, _, _ := seedUS4TestData(t, db)

	// 创建更多房间
	for i := 2; i <= 5; i++ {
		room := &models.Room{
			HotelID:     hotel.ID,
			RoomNo:      "10" + strconv.Itoa(i),
			RoomType:    models.RoomTypeStandard,
			MaxGuests:   2,
			HourlyPrice: 60.0,
			DailyPrice:  288.0,
			Status:      models.RoomStatusActive,
		}
		db.Create(room)
	}

		req, _ := http.NewRequest("GET", "/api/v1/hotels/"+strconv.FormatInt(hotel.ID, 10)+"/rooms", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].([]interface{})
	assert.Len(t, data, 5)
}

func TestUS4API_GetRoomDetail(t *testing.T) {
	router, db, _ := setupUS4APIRouter(t)
	_, _, room, _ := seedUS4TestData(t, db)

	t.Run("获取房间详情成功", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/rooms/"+strconv.FormatInt(room.ID, 10), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "101", data["room_no"])
		// 应该包含时段价格
		slots := data["time_slots"].([]interface{})
		assert.Len(t, slots, 1)
	})

	t.Run("房间不存在", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/rooms/99999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, float64(0), resp["code"])
	})
}

func TestUS4API_GetRoomTimeSlots(t *testing.T) {
	router, db, _ := setupUS4APIRouter(t)
	_, _, room, _ := seedUS4TestData(t, db)

	// 创建更多时段
	slots := []struct {
		duration int
		price    float64
	}{
		{4, 180.0},
		{6, 250.0},
	}
	for _, s := range slots {
		db.Create(&models.RoomTimeSlot{
			RoomID:        room.ID,
			DurationHours: s.duration,
			Price:         s.price,
			IsActive:      true,
			Sort:          s.duration,
		})
	}

	req, _ := http.NewRequest("GET", "/api/v1/rooms/"+strconv.FormatInt(room.ID, 10)+"/time-slots", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].([]interface{})
	assert.Len(t, data, 3)
}

func TestUS4API_CheckRoomAvailability(t *testing.T) {
	router, db, _ := setupUS4APIRouter(t)
	_, _, room, _ := seedUS4TestData(t, db)

	t.Run("房间可用", func(t *testing.T) {
		checkIn := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04")
		checkOut := time.Now().Add(4 * time.Hour).Format("2006-01-02 15:04")

		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/rooms/%d/availability?check_in=%s&check_out=%s",
			room.ID, checkIn, checkOut), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.True(t, data["available"].(bool))
	})

	t.Run("缺少参数", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/rooms/"+strconv.FormatInt(room.ID, 10)+"/availability", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 缺少参数时返回 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ==================== 预订 API 测试 ====================

func TestUS4API_CreateBooking(t *testing.T) {
	router, db, jwtManager := setupUS4APIRouter(t)
	user, _, room, _ := seedUS4TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	t.Run("创建预订成功", func(t *testing.T) {
		checkInTime := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04:05")
		body, _ := json.Marshal(map[string]interface{}{
			"room_id":        room.ID,
			"duration_hours": 2,
			"check_in_time":  checkInTime,
		})

		req, _ := http.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.NotEmpty(t, data["booking_no"])
		assert.Equal(t, models.BookingStatusPending, data["status"])
		assert.NotEmpty(t, data["verification_code"])
		assert.NotEmpty(t, data["unlock_code"])
	})

	t.Run("未登录创建预订失败", func(t *testing.T) {
		checkInTime := time.Now().Add(3 * time.Hour).Format("2006-01-02 15:04:05")
		body, _ := json.Marshal(map[string]interface{}{
			"room_id":        room.ID,
			"duration_hours": 2,
			"check_in_time":  checkInTime,
		})

		req, _ := http.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("房间不存在创建失败", func(t *testing.T) {
		checkInTime := time.Now().Add(4 * time.Hour).Format("2006-01-02 15:04:05")
		body, _ := json.Marshal(map[string]interface{}{
			"room_id":        99999,
			"duration_hours": 2,
			"check_in_time":  checkInTime,
		})

		req, _ := http.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, float64(0), resp["code"])
	})

	t.Run("入住时间是过去创建失败", func(t *testing.T) {
		// 使用 RFC3339 格式确保时区正确传递
		pastTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		body, _ := json.Marshal(map[string]interface{}{
			"room_id":        room.ID,
			"duration_hours": 2,
			"check_in_time":  pastTime,
		})

		req, _ := http.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, float64(0), resp["code"])
	})
}

func TestUS4API_GetMyBookings(t *testing.T) {
	router, db, jwtManager := setupUS4APIRouter(t)
	user, hotel, room, _ := seedUS4TestData(t, db)

	// 创建多个预订
	for i := 0; i < 3; i++ {
		order := &models.Order{
			OrderNo:        "OAPI" + strconv.Itoa(i),
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPending,
		}
		db.Create(order)

		checkInTime := time.Now().Add(time.Duration(i+1) * 24 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BAPI" + strconv.Itoa(i),
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkInTime.Add(2 * time.Hour),
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VAPI" + strconv.Itoa(i) + "XXXXXXXXXX",
			UnlockCode:       "12345" + strconv.Itoa(i),
			QRCode:           "/qr/api" + strconv.Itoa(i),
			Status:           models.BookingStatusPending,
		}
		db.Create(booking)
	}

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	t.Run("获取预订列表", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/bookings?page=1&page_size=10", nil)
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.Len(t, list, 3)
	})

	t.Run("分页获取", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/bookings?page=1&page_size=2", nil)
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.Len(t, list, 2)
		assert.Equal(t, float64(3), data["total"])
	})

	t.Run("按状态筛选", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/bookings?status=pending&page=1&page_size=10", nil)
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])
	})
}

func TestUS4API_GetBookingDetail(t *testing.T) {
	router, db, jwtManager := setupUS4APIRouter(t)
	user, hotel, room, _ := seedUS4TestData(t, db)

	// 创建预订
	order := &models.Order{
		OrderNo:        "ODETAIL001",
		UserID:         user.ID,
		Type:           models.OrderTypeHotel,
		OriginalAmount: 100.0,
		ActualAmount:   100.0,
		Status:         models.OrderStatusPaid,
	}
	db.Create(order)

	checkInTime := time.Now().Add(2 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BDETAIL001",
		OrderID:          order.ID,
		UserID:           user.ID,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkInTime,
		CheckOutTime:     checkInTime.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           100.0,
		VerificationCode: "VDETAIL001XXXXXXXXX",
		UnlockCode:       "123456",
		QRCode:           "/qr/detail001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	t.Run("获取预订详情成功", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/bookings/"+strconv.FormatInt(booking.ID, 10), nil)
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, booking.BookingNo, data["booking_no"])
		// 已支付状态应该显示敏感信息
		assert.NotEmpty(t, data["verification_code"])
		assert.NotEmpty(t, data["unlock_code"])
	})

	t.Run("获取不属于自己的预订失败", func(t *testing.T) {
		// 创建另一个用户
		phone2 := "13900139000"
		user2 := &models.User{
			Phone:         &phone2,
			Nickname:      "测试用户2",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		db.Create(user2)

		tokenPair2, _ := jwtManager.GenerateTokenPair(user2.ID, jwt.UserTypeUser, "")
		authz2 := "Bearer " + tokenPair2.AccessToken

		req, _ := http.NewRequest("GET", "/api/v1/bookings/"+strconv.FormatInt(booking.ID, 10), nil)
		req.Header.Set("Authorization", authz2)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, float64(0), resp["code"])
	})
}

func TestUS4API_GetBookingByNo(t *testing.T) {
	router, db, jwtManager := setupUS4APIRouter(t)
	user, hotel, room, _ := seedUS4TestData(t, db)

	// 创建预订
	order := &models.Order{
		OrderNo:        "OBYNO001",
		UserID:         user.ID,
		Type:           models.OrderTypeHotel,
		OriginalAmount: 100.0,
		ActualAmount:   100.0,
		Status:         models.OrderStatusPaid,
	}
	db.Create(order)

	checkInTime := time.Now().Add(2 * time.Hour)
	booking := &models.Booking{
		BookingNo:        "BBYNO001",
		OrderID:          order.ID,
		UserID:           user.ID,
		HotelID:          hotel.ID,
		RoomID:           room.ID,
		CheckInTime:      checkInTime,
		CheckOutTime:     checkInTime.Add(2 * time.Hour),
		DurationHours:    2,
		Amount:           100.0,
		VerificationCode: "VBYNO001XXXXXXXXXXX",
		UnlockCode:       "654321",
		QRCode:           "/qr/byno001",
		Status:           models.BookingStatusPaid,
	}
	db.Create(booking)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	t.Run("根据预订号获取预订成功", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/bookings/no/"+booking.BookingNo, nil)
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, booking.BookingNo, data["booking_no"])
	})

	t.Run("预订号不存在", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/bookings/no/NOTEXIST", nil)
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, float64(0), resp["code"])
	})
}

func TestUS4API_CancelBooking(t *testing.T) {
	router, db, jwtManager := setupUS4APIRouter(t)
	user, hotel, room, _ := seedUS4TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	t.Run("取消待支付预订成功", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "OCANCEL001",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPending,
		}
		db.Create(order)

		checkInTime := time.Now().Add(2 * time.Hour)
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
			VerificationCode: "VCANCEL001XXXXXXXXX",
			UnlockCode:       "111111",
			QRCode:           "/qr/cancel001",
			Status:           models.BookingStatusPending,
		}
		db.Create(booking)

		req, _ := http.NewRequest("POST", "/api/v1/bookings/"+strconv.FormatInt(booking.ID, 10)+"/cancel", nil)
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		// 验证状态已更新
		var updated models.Booking
		db.First(&updated, booking.ID)
		assert.Equal(t, models.BookingStatusCancelled, updated.Status)
	})

	t.Run("取消已支付预订失败", func(t *testing.T) {
		order := &models.Order{
			OrderNo:        "OCANCEL002",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		db.Create(order)

		checkInTime := time.Now().Add(3 * time.Hour)
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
			VerificationCode: "VCANCEL002XXXXXXXXX",
			UnlockCode:       "222222",
			QRCode:           "/qr/cancel002",
			Status:           models.BookingStatusPaid,
		}
		db.Create(booking)

		req, _ := http.NewRequest("POST", "/api/v1/bookings/"+strconv.FormatInt(booking.ID, 10)+"/cancel", nil)
		req.Header.Set("Authorization", authz)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, float64(0), resp["code"])
	})
}

func TestUS4API_Booking_Unauthorized(t *testing.T) {
	router, _, _ := setupUS4APIRouter(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/bookings"},
		{"GET", "/api/v1/bookings/1"},
		{"POST", "/api/v1/bookings"},
		{"POST", "/api/v1/bookings/1/cancel"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, ep.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// ==================== 完整酒店预订流程测试 ====================

func TestUS4API_FullBookingFlow(t *testing.T) {
	router, db, jwtManager := setupUS4APIRouter(t)
	user, hotel, room, _ := seedUS4TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 1. 获取城市列表
	citiesReq, _ := http.NewRequest("GET", "/api/v1/hotels/cities", nil)
	citiesW := httptest.NewRecorder()
	router.ServeHTTP(citiesW, citiesReq)
	require.Equal(t, http.StatusOK, citiesW.Code)
	t.Log("Step 1: 获取城市列表成功")

	// 2. 获取酒店列表
	hotelsReq, _ := http.NewRequest("GET", "/api/v1/hotels?city=深圳市&page=1&page_size=10", nil)
	hotelsW := httptest.NewRecorder()
	router.ServeHTTP(hotelsW, hotelsReq)
	require.Equal(t, http.StatusOK, hotelsW.Code)
	t.Log("Step 2: 获取酒店列表成功")

	// 3. 获取酒店详情
	hotelDetailReq, _ := http.NewRequest("GET", "/api/v1/hotels/"+strconv.FormatInt(hotel.ID, 10), nil)
	hotelDetailW := httptest.NewRecorder()
	router.ServeHTTP(hotelDetailW, hotelDetailReq)
	require.Equal(t, http.StatusOK, hotelDetailW.Code)
	t.Log("Step 3: 获取酒店详情成功")

	// 4. 获取房间列表
	roomsReq, _ := http.NewRequest("GET", "/api/v1/hotels/"+strconv.FormatInt(hotel.ID, 10)+"/rooms", nil)
	roomsW := httptest.NewRecorder()
	router.ServeHTTP(roomsW, roomsReq)
	require.Equal(t, http.StatusOK, roomsW.Code)
	t.Log("Step 4: 获取房间列表成功")

	// 5. 获取房间详情
	roomDetailReq, _ := http.NewRequest("GET", "/api/v1/rooms/"+strconv.FormatInt(room.ID, 10), nil)
	roomDetailW := httptest.NewRecorder()
	router.ServeHTTP(roomDetailW, roomDetailReq)
	require.Equal(t, http.StatusOK, roomDetailW.Code)
	t.Log("Step 5: 获取房间详情成功")

	// 6. 检查房间可用性
	checkInTime := time.Now().Add(2 * time.Hour)
	checkOutTime := checkInTime.Add(2 * time.Hour)
	availReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/rooms/%d/availability?check_in=%s&check_out=%s",
		room.ID, checkInTime.Format("2006-01-02 15:04"), checkOutTime.Format("2006-01-02 15:04")), nil)
	availW := httptest.NewRecorder()
	router.ServeHTTP(availW, availReq)
	require.Equal(t, http.StatusOK, availW.Code)
	t.Log("Step 6: 检查房间可用性成功")

	// 7. 创建预订
	createBookingBody, _ := json.Marshal(map[string]interface{}{
		"room_id":        room.ID,
		"duration_hours": 2,
		"check_in_time":  checkInTime.Format("2006-01-02 15:04:05"),
	})
	createBookingReq, _ := http.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(createBookingBody))
	createBookingReq.Header.Set("Content-Type", "application/json")
	createBookingReq.Header.Set("Authorization", authz)
	createBookingW := httptest.NewRecorder()
	router.ServeHTTP(createBookingW, createBookingReq)
	require.Equal(t, http.StatusOK, createBookingW.Code)

	var createBookingResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createBookingW.Body.Bytes(), &createBookingResp))
	bookingData := createBookingResp["data"].(map[string]interface{})
	bookingID := int64(bookingData["id"].(float64))
	t.Logf("Step 7: 创建预订成功，预订号: %s", bookingData["booking_no"])

	// 8. 获取预订详情
	bookingDetailReq, _ := http.NewRequest("GET", "/api/v1/bookings/"+strconv.FormatInt(bookingID, 10), nil)
	bookingDetailReq.Header.Set("Authorization", authz)
	bookingDetailW := httptest.NewRecorder()
	router.ServeHTTP(bookingDetailW, bookingDetailReq)
	require.Equal(t, http.StatusOK, bookingDetailW.Code)
	t.Log("Step 8: 获取预订详情成功")

	// 9. 获取我的预订列表
	myBookingsReq, _ := http.NewRequest("GET", "/api/v1/bookings", nil)
	myBookingsReq.Header.Set("Authorization", authz)
	myBookingsW := httptest.NewRecorder()
	router.ServeHTTP(myBookingsW, myBookingsReq)
	require.Equal(t, http.StatusOK, myBookingsW.Code)

	var myBookingsResp map[string]interface{}
	require.NoError(t, json.Unmarshal(myBookingsW.Body.Bytes(), &myBookingsResp))
	bookingsData := myBookingsResp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), bookingsData["total"])
	t.Log("Step 9: 获取我的预订列表成功")

	// 10. 取消预订
	cancelReq, _ := http.NewRequest("POST", "/api/v1/bookings/"+strconv.FormatInt(bookingID, 10)+"/cancel", nil)
	cancelReq.Header.Set("Authorization", authz)
	cancelW := httptest.NewRecorder()
	router.ServeHTTP(cancelW, cancelReq)
	require.Equal(t, http.StatusOK, cancelW.Code)
	t.Log("Step 10: 取消预订成功")

	// 验证预订已取消
	var cancelledBooking models.Booking
	db.First(&cancelledBooking, bookingID)
	assert.Equal(t, models.BookingStatusCancelled, cancelledBooking.Status)
	t.Log("完整酒店预订流程测试通过！")
}
