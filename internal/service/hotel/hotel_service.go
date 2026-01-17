// Package hotel 提供酒店服务
package hotel

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// HotelService 酒店服务
type HotelService struct {
	db                 *gorm.DB
	hotelRepo          *repository.HotelRepository
	roomRepo           *repository.RoomRepository
	roomTimeSlotRepo   *repository.RoomTimeSlotRepository
}

// NewHotelService 创建酒店服务
func NewHotelService(
	db *gorm.DB,
	hotelRepo *repository.HotelRepository,
	roomRepo *repository.RoomRepository,
	roomTimeSlotRepo *repository.RoomTimeSlotRepository,
) *HotelService {
	return &HotelService{
		db:               db,
		hotelRepo:        hotelRepo,
		roomRepo:         roomRepo,
		roomTimeSlotRepo: roomTimeSlotRepo,
	}
}

// HotelListRequest 酒店列表请求
type HotelListRequest struct {
	Page       int     `form:"page" json:"page"`
	PageSize   int     `form:"page_size" json:"page_size"`
	City       string  `form:"city" json:"city"`
	District   string  `form:"district" json:"district"`
	StarRating int     `form:"star_rating" json:"star_rating"`
	Keyword    string  `form:"keyword" json:"keyword"`
	Longitude  float64 `form:"longitude" json:"longitude"`
	Latitude   float64 `form:"latitude" json:"latitude"`
	RadiusKm   float64 `form:"radius_km" json:"radius_km"`
}

// HotelInfo 酒店信息
type HotelInfo struct {
	ID             int64              `json:"id"`
	Name           string             `json:"name"`
	StarRating     *int               `json:"star_rating,omitempty"`
	Province       string             `json:"province"`
	City           string             `json:"city"`
	District       string             `json:"district"`
	Address        string             `json:"address"`
	FullAddress    string             `json:"full_address"`
	Longitude      *float64           `json:"longitude,omitempty"`
	Latitude       *float64           `json:"latitude,omitempty"`
	Phone          string             `json:"phone"`
	Images         []string           `json:"images"`
	Facilities     []string           `json:"facilities"`
	Description    string             `json:"description"`
	CheckInTime    string             `json:"check_in_time"`
	CheckOutTime   string             `json:"check_out_time"`
	MinPrice       float64            `json:"min_price"`
	RoomCount      int64              `json:"room_count"`
	Distance       float64            `json:"distance,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

// RoomInfo 房间信息
type RoomInfo struct {
	ID          int64              `json:"id"`
	HotelID     int64              `json:"hotel_id"`
	RoomNo      string             `json:"room_no"`
	RoomType    string             `json:"room_type"`
	Images      []string           `json:"images"`
	Facilities  []string           `json:"facilities"`
	Area        *int               `json:"area,omitempty"`
	BedType     *string            `json:"bed_type,omitempty"`
	MaxGuests   int                `json:"max_guests"`
	HourlyPrice float64            `json:"hourly_price"`
	DailyPrice  float64            `json:"daily_price"`
	Status      int8               `json:"status"`
	StatusName  string             `json:"status_name"`
	TimeSlots   []TimeSlotInfo     `json:"time_slots,omitempty"`
	DeviceID    *int64             `json:"device_id,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

// TimeSlotInfo 时段信息
type TimeSlotInfo struct {
	ID            int64   `json:"id"`
	DurationHours int     `json:"duration_hours"`
	Price         float64 `json:"price"`
	StartTime     *string `json:"start_time,omitempty"`
	EndTime       *string `json:"end_time,omitempty"`
	IsActive      bool    `json:"is_active"`
}

// GetHotelList 获取酒店列表
func (s *HotelService) GetHotelList(ctx context.Context, req *HotelListRequest) ([]*HotelInfo, int64, error) {
	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}

	offset := (req.Page - 1) * req.PageSize

	// 附近搜索
	if req.Longitude > 0 && req.Latitude > 0 {
		radiusKm := req.RadiusKm
		if radiusKm <= 0 {
			radiusKm = 10 // 默认10公里
		}
		hotels, err := s.hotelRepo.ListNearby(ctx, req.Longitude, req.Latitude, radiusKm, req.PageSize)
		if err != nil {
			return nil, 0, errors.ErrDatabaseError.WithError(err)
		}
		return s.convertHotelList(hotels), int64(len(hotels)), nil
	}

	// 关键词搜索
	if req.Keyword != "" {
		hotels, total, err := s.hotelRepo.Search(ctx, req.Keyword, offset, req.PageSize)
		if err != nil {
			return nil, 0, errors.ErrDatabaseError.WithError(err)
		}
		return s.convertHotelList(hotels), total, nil
	}

	// 普通列表
	filters := map[string]interface{}{
		"status": int8(models.HotelStatusActive),
	}
	if req.City != "" {
		filters["city"] = req.City
	}
	if req.District != "" {
		filters["district"] = req.District
	}
	if req.StarRating > 0 {
		filters["star_rating"] = req.StarRating
	}

	hotels, total, err := s.hotelRepo.List(ctx, offset, req.PageSize, filters)
	if err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	return s.convertHotelList(hotels), total, nil
}

// GetHotelDetail 获取酒店详情
func (s *HotelService) GetHotelDetail(ctx context.Context, hotelID int64) (*HotelInfo, error) {
	hotel, err := s.hotelRepo.GetByIDWithRooms(ctx, hotelID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrHotelNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if hotel.Status != int8(models.HotelStatusActive) {
		return nil, errors.ErrHotelNotFound
	}

	return s.convertHotelInfo(hotel), nil
}

// GetRoomList 获取房间列表
func (s *HotelService) GetRoomList(ctx context.Context, hotelID int64) ([]*RoomInfo, error) {
	// 验证酒店存在
	hotel, err := s.hotelRepo.GetByID(ctx, hotelID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrHotelNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if hotel.Status != int8(models.HotelStatusActive) {
		return nil, errors.ErrHotelNotFound
	}

	rooms, err := s.roomRepo.ListAvailableByHotel(ctx, hotelID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.convertRoomList(rooms), nil
}

// GetRoomDetail 获取房间详情
func (s *HotelService) GetRoomDetail(ctx context.Context, roomID int64) (*RoomInfo, error) {
	room, err := s.roomRepo.GetByIDWithTimeSlots(ctx, roomID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrRoomNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if room.Status == int8(models.RoomStatusDisabled) {
		return nil, errors.ErrRoomNotFound
	}

	return s.convertRoomInfo(room), nil
}

// CheckRoomAvailability 检查房间可用性
func (s *HotelService) CheckRoomAvailability(ctx context.Context, roomID int64, checkIn, checkOut time.Time) (bool, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, errors.ErrRoomNotFound
		}
		return false, errors.ErrDatabaseError.WithError(err)
	}

	if room.Status != int8(models.RoomStatusActive) {
		return false, nil
	}

	available, err := s.roomRepo.CheckAvailability(ctx, roomID, checkIn, checkOut)
	if err != nil {
		return false, errors.ErrDatabaseError.WithError(err)
	}

	return available, nil
}

// GetCities 获取城市列表
func (s *HotelService) GetCities(ctx context.Context) ([]string, error) {
	cities, err := s.hotelRepo.GetCities(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	return cities, nil
}

// GetTimeSlotsByRoom 获取房间的时段价格
func (s *HotelService) GetTimeSlotsByRoom(ctx context.Context, roomID int64) ([]TimeSlotInfo, error) {
	slots, err := s.roomTimeSlotRepo.ListActiveByRoom(ctx, roomID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var result []TimeSlotInfo
	for _, slot := range slots {
		result = append(result, TimeSlotInfo{
			ID:            slot.ID,
			DurationHours: slot.DurationHours,
			Price:         slot.Price,
			StartTime:     slot.StartTime,
			EndTime:       slot.EndTime,
			IsActive:      slot.IsActive,
		})
	}

	return result, nil
}

// convertHotelList 转换酒店列表
func (s *HotelService) convertHotelList(hotels []*models.Hotel) []*HotelInfo {
	var result []*HotelInfo
	for _, hotel := range hotels {
		result = append(result, s.convertHotelInfo(hotel))
	}
	return result
}

// convertHotelInfo 转换酒店信息
func (s *HotelService) convertHotelInfo(hotel *models.Hotel) *HotelInfo {
	info := &HotelInfo{
		ID:           hotel.ID,
		Name:         hotel.Name,
		StarRating:   hotel.StarRating,
		Province:     hotel.Province,
		City:         hotel.City,
		District:     hotel.District,
		Address:      hotel.Address,
		FullAddress:  hotel.Province + hotel.City + hotel.District + hotel.Address,
		Longitude:    hotel.Longitude,
		Latitude:     hotel.Latitude,
		Phone:        hotel.Phone,
		CheckInTime:  hotel.CheckInTime,
		CheckOutTime: hotel.CheckOutTime,
		CreatedAt:    hotel.CreatedAt,
	}

	// 解析图片
	if hotel.Images != nil {
		info.Images = jsonToStringSlice(hotel.Images)
	}

	// 解析设施
	if hotel.Facilities != nil {
		info.Facilities = jsonToStringSlice(hotel.Facilities)
	}

	// 描述
	if hotel.Description != nil {
		info.Description = *hotel.Description
	}

	// 计算房间数和最低价
	if len(hotel.Rooms) > 0 {
		info.RoomCount = int64(len(hotel.Rooms))
		minPrice := hotel.Rooms[0].HourlyPrice
		for _, room := range hotel.Rooms {
			if room.HourlyPrice < minPrice {
				minPrice = room.HourlyPrice
			}
		}
		info.MinPrice = minPrice
	}

	return info
}

// convertRoomList 转换房间列表
func (s *HotelService) convertRoomList(rooms []*models.Room) []*RoomInfo {
	var result []*RoomInfo
	for _, room := range rooms {
		result = append(result, s.convertRoomInfo(room))
	}
	return result
}

// convertRoomInfo 转换房间信息
func (s *HotelService) convertRoomInfo(room *models.Room) *RoomInfo {
	info := &RoomInfo{
		ID:          room.ID,
		HotelID:     room.HotelID,
		RoomNo:      room.RoomNo,
		RoomType:    room.RoomType,
		Area:        room.Area,
		BedType:     room.BedType,
		MaxGuests:   room.MaxGuests,
		HourlyPrice: room.HourlyPrice,
		DailyPrice:  room.DailyPrice,
		Status:      room.Status,
		StatusName:  s.getRoomStatusName(room.Status),
		DeviceID:    room.DeviceID,
		CreatedAt:   room.CreatedAt,
	}

	// 解析图片
	if room.Images != nil {
		info.Images = jsonToStringSlice(room.Images)
	}

	// 解析设施
	if room.Facilities != nil {
		info.Facilities = jsonToStringSlice(room.Facilities)
	}

	// 时段价格
	if len(room.TimeSlots) > 0 {
		for _, slot := range room.TimeSlots {
			info.TimeSlots = append(info.TimeSlots, TimeSlotInfo{
				ID:            slot.ID,
				DurationHours: slot.DurationHours,
				Price:         slot.Price,
				StartTime:     slot.StartTime,
				EndTime:       slot.EndTime,
				IsActive:      slot.IsActive,
			})
		}
	}

	return info
}

// getRoomStatusName 获取房间状态名称
func (s *HotelService) getRoomStatusName(status int8) string {
	switch status {
	case models.RoomStatusDisabled:
		return "停用"
	case models.RoomStatusActive:
		return "可用"
	case models.RoomStatusBooked:
		return "已预订"
	case models.RoomStatusInUse:
		return "使用中"
	default:
		return "未知"
	}
}

// jsonToStringSlice 将 JSON map 转换为字符串切片
func jsonToStringSlice(j models.JSON) []string {
	if j == nil {
		return nil
	}
	var result []string
	for _, v := range j {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// RecommendedHotelInfo 推荐酒店信息
type RecommendedHotelInfo struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	StarRating     *int     `json:"star_rating,omitempty"`
	City           string   `json:"city"`
	District       string   `json:"district"`
	Address        string   `json:"address"`
	Images         []string `json:"images"`
	MinPrice       float64  `json:"min_price"`
	RecommendScore int      `json:"recommend_score"`
}

// GetRecommendedHotels 获取推荐酒店列表
func (s *HotelService) GetRecommendedHotels(ctx context.Context, limit int) ([]*RecommendedHotelInfo, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	hotels, err := s.hotelRepo.ListRecommended(ctx, limit)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var result []*RecommendedHotelInfo
	for _, hotel := range hotels {
		info := &RecommendedHotelInfo{
			ID:             hotel.ID,
			Name:           hotel.Name,
			StarRating:     hotel.StarRating,
			City:           hotel.City,
			District:       hotel.District,
			Address:        hotel.Address,
			RecommendScore: hotel.RecommendScore,
		}
		if hotel.Images != nil {
			info.Images = jsonToStringSlice(hotel.Images)
		}
		// 获取最低价格
		if len(hotel.Rooms) > 0 {
			minPrice := hotel.Rooms[0].HourlyPrice
			for _, room := range hotel.Rooms {
				if room.HourlyPrice < minPrice {
					minPrice = room.HourlyPrice
				}
			}
			info.MinPrice = minPrice
		}
		result = append(result, info)
	}

	return result, nil
}

// HotRoomInfo 热门房型信息
type HotRoomInfo struct {
	ID            int64    `json:"id"`
	HotelID       int64    `json:"hotel_id"`
	HotelName     string   `json:"hotel_name"`
	RoomNo        string   `json:"room_no"`
	RoomType      string   `json:"room_type"`
	Images        []string `json:"images"`
	HourlyPrice   float64  `json:"hourly_price"`
	DailyPrice    float64  `json:"daily_price"`
	BookingCount  int      `json:"booking_count"`
	AverageRating float64  `json:"average_rating"`
	ReviewCount   int      `json:"review_count"`
	HotScore      float64  `json:"hot_score"`
}

// GetHotRooms 获取全站热门房型
func (s *HotelService) GetHotRooms(ctx context.Context, limit int) ([]*HotRoomInfo, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	rooms, err := s.roomRepo.ListHotRooms(ctx, limit)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var result []*HotRoomInfo
	for _, room := range rooms {
		info := &HotRoomInfo{
			ID:            room.ID,
			HotelID:       room.HotelID,
			RoomNo:        room.RoomNo,
			RoomType:      room.RoomType,
			HourlyPrice:   room.HourlyPrice,
			DailyPrice:    room.DailyPrice,
			BookingCount:  room.BookingCount,
			AverageRating: room.AverageRating,
			ReviewCount:   room.ReviewCount,
			HotScore:      room.HotScore,
		}
		if room.Hotel != nil {
			info.HotelName = room.Hotel.Name
		}
		if room.Images != nil {
			info.Images = jsonToStringSlice(room.Images)
		}
		result = append(result, info)
	}

	return result, nil
}

// GetHotRoomsByHotel 获取酒店内热门房型
func (s *HotelService) GetHotRoomsByHotel(ctx context.Context, hotelID int64, limit int) ([]*HotRoomInfo, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	// 验证酒店存在
	hotel, err := s.hotelRepo.GetByID(ctx, hotelID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrHotelNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if hotel.Status != int8(models.HotelStatusActive) {
		return nil, errors.ErrHotelNotFound
	}

	rooms, err := s.roomRepo.ListHotRoomsByHotel(ctx, hotelID, limit)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var result []*HotRoomInfo
	for _, room := range rooms {
		info := &HotRoomInfo{
			ID:            room.ID,
			HotelID:       room.HotelID,
			HotelName:     hotel.Name,
			RoomNo:        room.RoomNo,
			RoomType:      room.RoomType,
			HourlyPrice:   room.HourlyPrice,
			DailyPrice:    room.DailyPrice,
			BookingCount:  room.BookingCount,
			AverageRating: room.AverageRating,
			ReviewCount:   room.ReviewCount,
			HotScore:      room.HotScore,
		}
		if room.Images != nil {
			info.Images = jsonToStringSlice(room.Images)
		}
		result = append(result, info)
	}

	return result, nil
}
