// Package hotel 酒店服务单元测试
package hotel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// testHotelService 测试用酒店服务
type testHotelService struct {
	*HotelService
	db *gorm.DB
}

// setupTestHotelService 创建测试用的 HotelService
func setupTestHotelService(t *testing.T) *testHotelService {
	db := setupTestDB(t)
	hotelRepo := repository.NewHotelRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	timeSlotRepo := repository.NewRoomTimeSlotRepository(db)

	service := NewHotelService(db, hotelRepo, roomRepo, timeSlotRepo)

	return &testHotelService{
		HotelService: service,
		db:           db,
	}
}

// createTestHotelData 创建酒店测试数据
func createTestHotelData(t *testing.T, db *gorm.DB) (hotel *models.Hotel, room *models.Room, timeSlots []*models.RoomTimeSlot) {
	// 创建酒店
	description := "测试酒店描述"
	starRating := 4
	longitude := 114.0579
	latitude := 22.5431
	hotel = &models.Hotel{
		Name:         "测试酒店",
		StarRating:   &starRating,
		Province:     "广东省",
		City:         "深圳市",
		District:     "南山区",
		Address:      "科技园路1号",
		Longitude:    &longitude,
		Latitude:     &latitude,
		Phone:        "0755-12345678",
		Description:  &description,
		CheckInTime:  "14:00",
		CheckOutTime: "12:00",
		Status:       models.HotelStatusActive,
	}
	err := db.Create(hotel).Error
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
	timeSlots = []*models.RoomTimeSlot{
		{RoomID: room.ID, DurationHours: 2, Price: 100.0, IsActive: true, Sort: 1},
		{RoomID: room.ID, DurationHours: 4, Price: 180.0, IsActive: true, Sort: 2},
		{RoomID: room.ID, DurationHours: 6, Price: 250.0, IsActive: true, Sort: 3},
	}
	for _, slot := range timeSlots {
		err = db.Create(slot).Error
		require.NoError(t, err)
	}

	return hotel, room, timeSlots
}

func TestHotelService_GetHotelList(t *testing.T) {
	svc := setupTestHotelService(t)
	ctx := context.Background()

	// 创建多个酒店
	cities := []string{"深圳市", "广州市", "东莞市"}
	for i, city := range cities {
		description := "测试酒店描述"
		starRating := 3 + (i % 3)
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
		svc.db.Create(hotel)
	}

	// 创建一个下架的酒店（先创建再更新状态，避免 default:1 覆盖）
	disabledHotel := &models.Hotel{
		Name:         "已下架酒店",
		Province:     "广东省",
		City:         "深圳市",
		District:     "南山区",
		Address:      "测试路2号",
		Phone:        "0755-12345679",
		CheckInTime:  "14:00",
		CheckOutTime: "12:00",
		Status:       models.HotelStatusActive,
	}
	svc.db.Create(disabledHotel)
	svc.db.Model(disabledHotel).Update("status", models.HotelStatusDisabled)

	t.Run("获取酒店列表", func(t *testing.T) {
		req := &HotelListRequest{
			Page:     1,
			PageSize: 10,
		}
		hotels, total, err := svc.GetHotelList(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total) // 只有3个上架酒店
		assert.Len(t, hotels, 3)
	})

	t.Run("按城市过滤", func(t *testing.T) {
		req := &HotelListRequest{
			Page:     1,
			PageSize: 10,
			City:     "深圳市",
		}
		hotels, total, err := svc.GetHotelList(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, hotels, 1)
		assert.Equal(t, "深圳市", hotels[0].City)
	})

	t.Run("关键词搜索", func(t *testing.T) {
		req := &HotelListRequest{
			Page:     1,
			PageSize: 10,
			Keyword:  "广州",
		}
		hotels, total, err := svc.GetHotelList(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Contains(t, hotels[0].Name, "广州")
	})

	t.Run("分页测试", func(t *testing.T) {
		req := &HotelListRequest{
			Page:     1,
			PageSize: 2,
		}
		hotels, total, err := svc.GetHotelList(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, hotels, 2)

		// 获取第二页
		req.Page = 2
		hotels2, _, err := svc.GetHotelList(ctx, req)
		require.NoError(t, err)
		assert.Len(t, hotels2, 1)
	})

	t.Run("默认分页参数", func(t *testing.T) {
		req := &HotelListRequest{
			Page:     0, // 无效值应该被设为1
			PageSize: 0, // 无效值应该被设为10
		}
		hotels, _, err := svc.GetHotelList(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, hotels)
	})
}

func Test_jsonToStringSlice(t *testing.T) {
	t.Run("nil 返回 nil", func(t *testing.T) {
		assert.Nil(t, jsonToStringSlice(nil))
	})

	t.Run("只提取 string 值", func(t *testing.T) {
		j := models.JSON{
			"a": "wifi",
			"b": 123,
			"c": "parking",
			"d": true,
		}
		assert.ElementsMatch(t, []string{"wifi", "parking"}, jsonToStringSlice(j))
	})
}

func TestHotelService_GetHotelDetail(t *testing.T) {
	svc := setupTestHotelService(t)
	ctx := context.Background()

	hotel, _, _ := createTestHotelData(t, svc.db)

	t.Run("获取酒店详情成功", func(t *testing.T) {
		info, err := svc.GetHotelDetail(ctx, hotel.ID)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, hotel.ID, info.ID)
		assert.Equal(t, hotel.Name, info.Name)
		assert.Equal(t, hotel.City, info.City)
		assert.NotEmpty(t, info.FullAddress)
	})

	t.Run("酒店不存在", func(t *testing.T) {
		_, err := svc.GetHotelDetail(ctx, 999999)
		assert.Error(t, err)
	})

	t.Run("下架酒店不可查看", func(t *testing.T) {
		disabledHotel := &models.Hotel{
			Name:         "已下架酒店",
			Province:     "广东省",
			City:         "深圳市",
			District:     "南山区",
			Address:      "测试路2号",
			Phone:        "0755-12345679",
			CheckInTime:  "14:00",
			CheckOutTime: "12:00",
			Status:       models.HotelStatusActive, // 先以Active状态创建
		}
		svc.db.Create(disabledHotel)
		// 使用 Updates 强制设置 Status 为 0（避免 default:1 覆盖）
		svc.db.Model(disabledHotel).Update("status", models.HotelStatusDisabled)

		_, err := svc.GetHotelDetail(ctx, disabledHotel.ID)
		assert.Error(t, err)
	})
}

func TestHotelService_GetRoomList(t *testing.T) {
	svc := setupTestHotelService(t)
	ctx := context.Background()

	hotel, _, _ := createTestHotelData(t, svc.db)

	// 创建更多房间
	for i := 2; i <= 5; i++ {
		area := 25
		room := &models.Room{
			HotelID:     hotel.ID,
			RoomNo:      "10" + string(rune('0'+i)),
			RoomType:    models.RoomTypeStandard,
			Area:        &area,
			MaxGuests:   2,
			HourlyPrice: 60.0,
			DailyPrice:  288.0,
			Status:      models.RoomStatusActive,
		}
		svc.db.Create(room)
	}

	// 创建一个停用的房间（先创建再更新状态，避免 default:1 覆盖）
	disabledRoom := &models.Room{
		HotelID:     hotel.ID,
		RoomNo:      "999",
		RoomType:    models.RoomTypeStandard,
		MaxGuests:   2,
		HourlyPrice: 60.0,
		DailyPrice:  288.0,
		Status:      models.RoomStatusActive,
	}
	svc.db.Create(disabledRoom)
	svc.db.Model(disabledRoom).Update("status", models.RoomStatusDisabled)

	t.Run("获取房间列表", func(t *testing.T) {
		rooms, err := svc.GetRoomList(ctx, hotel.ID)
		require.NoError(t, err)
		assert.Len(t, rooms, 5) // 只有5个可用房间
		// 不应包含停用的房间
		for _, room := range rooms {
			assert.NotEqual(t, models.RoomStatusDisabled, room.Status)
		}
	})

	t.Run("酒店不存在", func(t *testing.T) {
		_, err := svc.GetRoomList(ctx, 999999)
		assert.Error(t, err)
	})
}

func TestHotelService_GetRoomDetail(t *testing.T) {
	svc := setupTestHotelService(t)
	ctx := context.Background()

	_, room, timeSlots := createTestHotelData(t, svc.db)

	t.Run("获取房间详情成功", func(t *testing.T) {
		info, err := svc.GetRoomDetail(ctx, room.ID)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, room.ID, info.ID)
		assert.Equal(t, room.RoomNo, info.RoomNo)
		assert.Equal(t, room.RoomType, info.RoomType)
		assert.Equal(t, room.HourlyPrice, info.HourlyPrice)
		// 应该包含时段价格
		assert.Len(t, info.TimeSlots, len(timeSlots))
	})

	t.Run("房间不存在", func(t *testing.T) {
		_, err := svc.GetRoomDetail(ctx, 999999)
		assert.Error(t, err)
	})

	t.Run("停用房间不可查看", func(t *testing.T) {
		disabledRoom := &models.Room{
			HotelID:     room.HotelID,
			RoomNo:      "998",
			RoomType:    models.RoomTypeStandard,
			MaxGuests:   2,
			HourlyPrice: 60.0,
			DailyPrice:  288.0,
			Status:      models.RoomStatusActive,
		}
		svc.db.Create(disabledRoom)
		svc.db.Model(disabledRoom).Update("status", models.RoomStatusDisabled)

		_, err := svc.GetRoomDetail(ctx, disabledRoom.ID)
		assert.Error(t, err)
	})
}

func TestHotelService_CheckRoomAvailability(t *testing.T) {
	svc := setupTestHotelService(t)
	ctx := context.Background()

	user := &models.User{
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	svc.db.Create(user)

	hotel, room, _ := createTestHotelData(t, svc.db)

	t.Run("无预订时房间可用", func(t *testing.T) {
		checkIn := time.Now().Add(1 * time.Hour)
		checkOut := checkIn.Add(2 * time.Hour)

		available, err := svc.CheckRoomAvailability(ctx, room.ID, checkIn, checkOut)
		require.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("有预订时房间不可用", func(t *testing.T) {
		// 先创建一个订单和预订
		order := &models.Order{
			OrderNo:        "AVAIL001",
			UserID:         user.ID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusPaid,
		}
		svc.db.Create(order)

		checkIn := time.Now().Add(10 * time.Hour)
		checkOut := checkIn.Add(2 * time.Hour)
		booking := &models.Booking{
			BookingNo:        "BAVAIL001",
			OrderID:          order.ID,
			UserID:           user.ID,
			HotelID:          hotel.ID,
			RoomID:           room.ID,
			CheckInTime:      checkIn,
			CheckOutTime:     checkOut,
			DurationHours:    2,
			Amount:           100.0,
			VerificationCode: "VAVAIL001XXXXXXXXXX",
			UnlockCode:       "123456",
			QRCode:           "/qr/avail001",
			Status:           models.BookingStatusPaid,
		}
		svc.db.Create(booking)

		// 查询同一时段
		available, err := svc.CheckRoomAvailability(ctx, room.ID, checkIn, checkOut)
		require.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("停用房间不可用", func(t *testing.T) {
		disabledRoom := &models.Room{
			HotelID:     hotel.ID,
			RoomNo:      "997",
			RoomType:    models.RoomTypeStandard,
			MaxGuests:   2,
			HourlyPrice: 60.0,
			DailyPrice:  288.0,
			Status:      models.RoomStatusActive,
		}
		svc.db.Create(disabledRoom)
		svc.db.Model(disabledRoom).Update("status", models.RoomStatusDisabled)

		checkIn := time.Now().Add(1 * time.Hour)
		checkOut := checkIn.Add(2 * time.Hour)

		available, err := svc.CheckRoomAvailability(ctx, disabledRoom.ID, checkIn, checkOut)
		require.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("房间不存在", func(t *testing.T) {
		checkIn := time.Now().Add(1 * time.Hour)
		checkOut := checkIn.Add(2 * time.Hour)

		_, err := svc.CheckRoomAvailability(ctx, 999999, checkIn, checkOut)
		assert.Error(t, err)
	})
}

func TestHotelService_GetCities(t *testing.T) {
	svc := setupTestHotelService(t)
	ctx := context.Background()

	// 创建不同城市的酒店
	cities := []string{"深圳市", "广州市", "东莞市"}
	for _, city := range cities {
		hotel := &models.Hotel{
			Name:         "测试酒店" + city,
			Province:     "广东省",
			City:         city,
			District:     "中心区",
			Address:      "测试路1号",
			Phone:        "0755-12345678",
			CheckInTime:  "14:00",
			CheckOutTime: "12:00",
			Status:       models.HotelStatusActive,
		}
		svc.db.Create(hotel)
	}

	// 创建一个下架酒店的城市（先创建再更新状态，避免 default:1 覆盖）
	disabledHotel := &models.Hotel{
		Name:         "已下架酒店",
		Province:     "广东省",
		City:         "珠海市",
		District:     "中心区",
		Address:      "测试路1号",
		Phone:        "0755-12345678",
		CheckInTime:  "14:00",
		CheckOutTime: "12:00",
		Status:       models.HotelStatusActive,
	}
	svc.db.Create(disabledHotel)
	svc.db.Model(disabledHotel).Update("status", models.HotelStatusDisabled)

	t.Run("获取城市列表", func(t *testing.T) {
		citiesList, err := svc.GetCities(ctx)
		require.NoError(t, err)
		assert.Len(t, citiesList, 3) // 只返回有上架酒店的城市
		assert.NotContains(t, citiesList, "珠海市")
	})
}

func TestHotelService_GetTimeSlotsByRoom(t *testing.T) {
	svc := setupTestHotelService(t)
	ctx := context.Background()

	_, room, expectedSlots := createTestHotelData(t, svc.db)

	// 创建一个停用的时段（先创建再更新状态，避免 default:true 覆盖）
	disabledSlot := &models.RoomTimeSlot{
		RoomID:        room.ID,
		DurationHours: 8,
		Price:         300.0,
		IsActive:      true,
		Sort:          4,
	}
	svc.db.Create(disabledSlot)
	svc.db.Model(disabledSlot).Update("is_active", false)

	t.Run("获取房间时段价格", func(t *testing.T) {
		slots, err := svc.GetTimeSlotsByRoom(ctx, room.ID)
		require.NoError(t, err)
		assert.Len(t, slots, len(expectedSlots)) // 只返回启用的时段
		// 验证排序
		for i := 0; i < len(slots)-1; i++ {
			assert.LessOrEqual(t, slots[i].DurationHours, slots[i+1].DurationHours)
		}
	})
}

func TestHotelService_getRoomStatusName(t *testing.T) {
	svc := setupTestHotelService(t)

	tests := []struct {
		status   int8
		expected string
	}{
		{models.RoomStatusDisabled, "停用"},
		{models.RoomStatusActive, "可用"},
		{models.RoomStatusBooked, "已预订"},
		{models.RoomStatusInUse, "使用中"},
		{99, "未知"},
	}

	for _, tt := range tests {
		name := svc.getRoomStatusName(tt.status)
		assert.Equal(t, tt.expected, name)
	}
}

func TestHotelService_convertHotelInfo(t *testing.T) {
	svc := setupTestHotelService(t)

	description := "测试酒店描述"
	starRating := 4
	longitude := 114.0579
	latitude := 22.5431

	hotel := &models.Hotel{
		ID:           1,
		Name:         "测试酒店",
		StarRating:   &starRating,
		Province:     "广东省",
		City:         "深圳市",
		District:     "南山区",
		Address:      "科技园路1号",
		Longitude:    &longitude,
		Latitude:     &latitude,
		Phone:        "0755-12345678",
		Description:  &description,
		CheckInTime:  "14:00",
		CheckOutTime: "12:00",
		Status:       models.HotelStatusActive,
		CreatedAt:    time.Now(),
	}

	info := svc.convertHotelInfo(hotel)

	assert.Equal(t, hotel.ID, info.ID)
	assert.Equal(t, hotel.Name, info.Name)
	assert.Equal(t, *hotel.StarRating, *info.StarRating)
	assert.Equal(t, hotel.Province, info.Province)
	assert.Equal(t, hotel.City, info.City)
	assert.Equal(t, hotel.District, info.District)
	assert.Equal(t, hotel.Address, info.Address)
	assert.Equal(t, "广东省深圳市南山区科技园路1号", info.FullAddress)
	assert.Equal(t, *hotel.Longitude, *info.Longitude)
	assert.Equal(t, *hotel.Latitude, *info.Latitude)
	assert.Equal(t, hotel.Phone, info.Phone)
	assert.Equal(t, *hotel.Description, info.Description)
	assert.Equal(t, hotel.CheckInTime, info.CheckInTime)
	assert.Equal(t, hotel.CheckOutTime, info.CheckOutTime)
}

func TestHotelService_convertRoomInfo(t *testing.T) {
	svc := setupTestHotelService(t)

	area := 25
	bedType := "大床"

	room := &models.Room{
		ID:          1,
		HotelID:     1,
		RoomNo:      "101",
		RoomType:    models.RoomTypeStandard,
		Area:        &area,
		BedType:     &bedType,
		MaxGuests:   2,
		HourlyPrice: 60.0,
		DailyPrice:  288.0,
		Status:      models.RoomStatusActive,
		CreatedAt:   time.Now(),
	}

	info := svc.convertRoomInfo(room)

	assert.Equal(t, room.ID, info.ID)
	assert.Equal(t, room.HotelID, info.HotelID)
	assert.Equal(t, room.RoomNo, info.RoomNo)
	assert.Equal(t, room.RoomType, info.RoomType)
	assert.Equal(t, *room.Area, *info.Area)
	assert.Equal(t, *room.BedType, *info.BedType)
	assert.Equal(t, room.MaxGuests, info.MaxGuests)
	assert.Equal(t, room.HourlyPrice, info.HourlyPrice)
	assert.Equal(t, room.DailyPrice, info.DailyPrice)
	assert.Equal(t, room.Status, info.Status)
	assert.Equal(t, "可用", info.StatusName)
}
