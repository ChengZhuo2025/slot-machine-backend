// Package admin 提供管理后台服务
package admin

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// HotelAdminService 酒店管理服务
type HotelAdminService struct {
	db               *gorm.DB
	hotelRepo        *repository.HotelRepository
	roomRepo         *repository.RoomRepository
	bookingRepo      *repository.BookingRepository
	timeSlotRepo     *repository.RoomTimeSlotRepository
}

// NewHotelAdminService 创建酒店管理服务
func NewHotelAdminService(
	db *gorm.DB,
	hotelRepo *repository.HotelRepository,
	roomRepo *repository.RoomRepository,
	bookingRepo *repository.BookingRepository,
	timeSlotRepo *repository.RoomTimeSlotRepository,
) *HotelAdminService {
	return &HotelAdminService{
		db:           db,
		hotelRepo:    hotelRepo,
		roomRepo:     roomRepo,
		bookingRepo:  bookingRepo,
		timeSlotRepo: timeSlotRepo,
	}
}

// CreateHotelRequest 创建酒店请求
type CreateHotelRequest struct {
	Name           string   `json:"name" binding:"required"`
	StarRating     *int     `json:"star_rating"`
	Province       string   `json:"province" binding:"required"`
	City           string   `json:"city" binding:"required"`
	District       string   `json:"district" binding:"required"`
	Address        string   `json:"address" binding:"required"`
	Phone          string   `json:"phone" binding:"required"`
	Longitude      *float64 `json:"longitude"`
	Latitude       *float64 `json:"latitude"`
	Images         []string `json:"images"`
	Facilities     []string `json:"facilities"`
	Description    *string  `json:"description"`
	CheckInTime    string   `json:"check_in_time"`
	CheckOutTime   string   `json:"check_out_time"`
	CommissionRate float64  `json:"commission_rate"`
}

// UpdateHotelRequest 更新酒店请求
type UpdateHotelRequest struct {
	Name           *string   `json:"name"`
	StarRating     *int      `json:"star_rating"`
	Province       *string   `json:"province"`
	City           *string   `json:"city"`
	District       *string   `json:"district"`
	Address        *string   `json:"address"`
	Phone          *string   `json:"phone"`
	Longitude      *float64  `json:"longitude"`
	Latitude       *float64  `json:"latitude"`
	Images         []string  `json:"images"`
	Facilities     []string  `json:"facilities"`
	Description    *string   `json:"description"`
	CheckInTime    *string   `json:"check_in_time"`
	CheckOutTime   *string   `json:"check_out_time"`
	CommissionRate *float64  `json:"commission_rate"`
	Status         *int8     `json:"status"`
}

// CreateRoomRequest 创建房间请求
type CreateRoomRequest struct {
	HotelID     int64    `json:"hotel_id" binding:"required"`
	RoomNo      string   `json:"room_no" binding:"required"`
	RoomType    string   `json:"room_type" binding:"required"`
	DeviceID    *int64   `json:"device_id"`
	Images      []string `json:"images"`
	Facilities  []string `json:"facilities"`
	Area        *int     `json:"area"`
	BedType     *string  `json:"bed_type"`
	MaxGuests   int      `json:"max_guests"`
	HourlyPrice float64  `json:"hourly_price" binding:"required"`
	DailyPrice  float64  `json:"daily_price" binding:"required"`
}

// UpdateRoomRequest 更新房间请求
type UpdateRoomRequest struct {
	RoomNo      *string   `json:"room_no"`
	RoomType    *string   `json:"room_type"`
	DeviceID    *int64    `json:"device_id"`
	Images      []string  `json:"images"`
	Facilities  []string  `json:"facilities"`
	Area        *int      `json:"area"`
	BedType     *string   `json:"bed_type"`
	MaxGuests   *int      `json:"max_guests"`
	HourlyPrice *float64  `json:"hourly_price"`
	DailyPrice  *float64  `json:"daily_price"`
	Status      *int8     `json:"status"`
}

// CreateTimeSlotRequest 创建时段请求
type CreateTimeSlotRequest struct {
	RoomID        int64   `json:"room_id" binding:"required"`
	DurationHours int     `json:"duration_hours" binding:"required,min=1"`
	Price         float64 `json:"price" binding:"required"`
	StartTime     *string `json:"start_time"`
	EndTime       *string `json:"end_time"`
	Sort          int     `json:"sort"`
}

// CreateHotel 创建酒店
func (s *HotelAdminService) CreateHotel(ctx context.Context, req *CreateHotelRequest) (*models.Hotel, error) {
	// 检查名称是否重复
	exists, err := s.hotelRepo.ExistsByName(ctx, req.Name)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithMessage("酒店名称已存在")
	}

	hotel := &models.Hotel{
		Name:           req.Name,
		StarRating:     req.StarRating,
		Province:       req.Province,
		City:           req.City,
		District:       req.District,
		Address:        req.Address,
		Phone:          req.Phone,
		Longitude:      req.Longitude,
		Latitude:       req.Latitude,
		Description:    req.Description,
		CommissionRate: req.CommissionRate,
		Status:         int8(models.HotelStatusActive),
	}

	// 设置默认入住/退房时间
	if req.CheckInTime != "" {
		hotel.CheckInTime = req.CheckInTime
	} else {
		hotel.CheckInTime = "14:00"
	}
	if req.CheckOutTime != "" {
		hotel.CheckOutTime = req.CheckOutTime
	} else {
		hotel.CheckOutTime = "12:00"
	}

	// 设置图片和设施
	if len(req.Images) > 0 {
		hotel.Images = stringSliceToJSON(req.Images)
	}
	if len(req.Facilities) > 0 {
		hotel.Facilities = stringSliceToJSON(req.Facilities)
	}

	if err := s.hotelRepo.Create(ctx, hotel); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return hotel, nil
}

// UpdateHotel 更新酒店
func (s *HotelAdminService) UpdateHotel(ctx context.Context, id int64, req *UpdateHotelRequest) (*models.Hotel, error) {
	hotel, err := s.hotelRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrHotelNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 检查名称重复（如果修改了名称）
	if req.Name != nil && *req.Name != hotel.Name {
		exists, err := s.hotelRepo.ExistsByName(ctx, *req.Name)
		if err != nil {
			return nil, errors.ErrDatabaseError.WithError(err)
		}
		if exists {
			return nil, errors.ErrAlreadyExists.WithMessage("酒店名称已存在")
		}
		hotel.Name = *req.Name
	}

	// 更新字段
	if req.StarRating != nil {
		hotel.StarRating = req.StarRating
	}
	if req.Province != nil {
		hotel.Province = *req.Province
	}
	if req.City != nil {
		hotel.City = *req.City
	}
	if req.District != nil {
		hotel.District = *req.District
	}
	if req.Address != nil {
		hotel.Address = *req.Address
	}
	if req.Phone != nil {
		hotel.Phone = *req.Phone
	}
	if req.Longitude != nil {
		hotel.Longitude = req.Longitude
	}
	if req.Latitude != nil {
		hotel.Latitude = req.Latitude
	}
	if req.Description != nil {
		hotel.Description = req.Description
	}
	if req.CheckInTime != nil {
		hotel.CheckInTime = *req.CheckInTime
	}
	if req.CheckOutTime != nil {
		hotel.CheckOutTime = *req.CheckOutTime
	}
	if req.CommissionRate != nil {
		hotel.CommissionRate = *req.CommissionRate
	}
	if req.Status != nil {
		hotel.Status = *req.Status
	}
	if len(req.Images) > 0 {
		hotel.Images = stringSliceToJSON(req.Images)
	}
	if len(req.Facilities) > 0 {
		hotel.Facilities = stringSliceToJSON(req.Facilities)
	}

	if err := s.hotelRepo.Update(ctx, hotel); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return hotel, nil
}

// GetHotelList 获取酒店列表
func (s *HotelAdminService) GetHotelList(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*models.Hotel, int64, error) {
	offset := (page - 1) * pageSize
	return s.hotelRepo.List(ctx, offset, pageSize, filters)
}

// GetHotelByID 获取酒店详情
func (s *HotelAdminService) GetHotelByID(ctx context.Context, id int64) (*models.Hotel, error) {
	hotel, err := s.hotelRepo.GetByIDWithRooms(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrHotelNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	return hotel, nil
}

// DeleteHotel 删除酒店
func (s *HotelAdminService) DeleteHotel(ctx context.Context, id int64) error {
	// 检查是否有房间
	count, err := s.hotelRepo.GetRoomCount(ctx, id)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}
	if count > 0 {
		return errors.ErrOperationFailed.WithMessage("请先删除所有房间")
	}

	return s.hotelRepo.Delete(ctx, id)
}

// UpdateHotelStatus 更新酒店状态
func (s *HotelAdminService) UpdateHotelStatus(ctx context.Context, id int64, status int8) error {
	_, err := s.hotelRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrHotelNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}
	return s.hotelRepo.UpdateStatus(ctx, id, status)
}

// CreateRoom 创建房间
func (s *HotelAdminService) CreateRoom(ctx context.Context, req *CreateRoomRequest) (*models.Room, error) {
	// 验证酒店存在
	_, err := s.hotelRepo.GetByID(ctx, req.HotelID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrHotelNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 检查房间号是否重复
	exists, err := s.roomRepo.ExistsByRoomNo(ctx, req.HotelID, req.RoomNo)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithMessage("房间号已存在")
	}

	room := &models.Room{
		HotelID:     req.HotelID,
		RoomNo:      req.RoomNo,
		RoomType:    req.RoomType,
		DeviceID:    req.DeviceID,
		Area:        req.Area,
		BedType:     req.BedType,
		MaxGuests:   req.MaxGuests,
		HourlyPrice: req.HourlyPrice,
		DailyPrice:  req.DailyPrice,
		Status:      int8(models.RoomStatusActive),
	}

	if req.MaxGuests == 0 {
		room.MaxGuests = 2
	}

	if len(req.Images) > 0 {
		room.Images = stringSliceToJSON(req.Images)
	}
	if len(req.Facilities) > 0 {
		room.Facilities = stringSliceToJSON(req.Facilities)
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return room, nil
}

// UpdateRoom 更新房间
func (s *HotelAdminService) UpdateRoom(ctx context.Context, id int64, req *UpdateRoomRequest) (*models.Room, error) {
	room, err := s.roomRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrRoomNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 检查房间号重复
	if req.RoomNo != nil && *req.RoomNo != room.RoomNo {
		exists, err := s.roomRepo.ExistsByRoomNo(ctx, room.HotelID, *req.RoomNo)
		if err != nil {
			return nil, errors.ErrDatabaseError.WithError(err)
		}
		if exists {
			return nil, errors.ErrAlreadyExists.WithMessage("房间号已存在")
		}
		room.RoomNo = *req.RoomNo
	}

	if req.RoomType != nil {
		room.RoomType = *req.RoomType
	}
	if req.DeviceID != nil {
		room.DeviceID = req.DeviceID
	}
	if req.Area != nil {
		room.Area = req.Area
	}
	if req.BedType != nil {
		room.BedType = req.BedType
	}
	if req.MaxGuests != nil {
		room.MaxGuests = *req.MaxGuests
	}
	if req.HourlyPrice != nil {
		room.HourlyPrice = *req.HourlyPrice
	}
	if req.DailyPrice != nil {
		room.DailyPrice = *req.DailyPrice
	}
	if req.Status != nil {
		room.Status = *req.Status
	}
	if len(req.Images) > 0 {
		room.Images = stringSliceToJSON(req.Images)
	}
	if len(req.Facilities) > 0 {
		room.Facilities = stringSliceToJSON(req.Facilities)
	}

	if err := s.roomRepo.Update(ctx, room); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return room, nil
}

// GetRoomByID 获取房间详情
func (s *HotelAdminService) GetRoomByID(ctx context.Context, id int64) (*models.Room, error) {
	room, err := s.roomRepo.GetByIDWithTimeSlots(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrRoomNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	return room, nil
}

// GetRoomList 获取房间列表
func (s *HotelAdminService) GetRoomList(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*models.Room, int64, error) {
	offset := (page - 1) * pageSize
	return s.roomRepo.List(ctx, offset, pageSize, filters)
}

// DeleteRoom 删除房间
func (s *HotelAdminService) DeleteRoom(ctx context.Context, id int64) error {
	// 先删除时段
	if err := s.timeSlotRepo.DeleteByRoom(ctx, id); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}
	return s.roomRepo.Delete(ctx, id)
}

// CreateTimeSlot 创建时段价格
func (s *HotelAdminService) CreateTimeSlot(ctx context.Context, req *CreateTimeSlotRequest) (*models.RoomTimeSlot, error) {
	// 验证房间存在
	_, err := s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrRoomNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	slot := &models.RoomTimeSlot{
		RoomID:        req.RoomID,
		DurationHours: req.DurationHours,
		Price:         req.Price,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Sort:          req.Sort,
		IsActive:      true,
	}

	if err := s.timeSlotRepo.Create(ctx, slot); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return slot, nil
}

// UpdateTimeSlot 更新时段价格
func (s *HotelAdminService) UpdateTimeSlot(ctx context.Context, id int64, fields map[string]interface{}) error {
	return s.timeSlotRepo.UpdateFields(ctx, id, fields)
}

// DeleteTimeSlot 删除时段
func (s *HotelAdminService) DeleteTimeSlot(ctx context.Context, id int64) error {
	return s.timeSlotRepo.Delete(ctx, id)
}

// GetBookingList 获取预订列表
func (s *HotelAdminService) GetBookingList(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*models.Booking, int64, error) {
	offset := (page - 1) * pageSize
	return s.bookingRepo.List(ctx, offset, pageSize, filters)
}

// GetBookingByID 获取预订详情
func (s *HotelAdminService) GetBookingByID(ctx context.Context, id int64) (*models.Booking, error) {
	booking, err := s.bookingRepo.GetByIDWithDetails(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrBookingNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	return booking, nil
}

// stringSliceToJSON 将字符串切片转换为 JSON 格式
func stringSliceToJSON(slice []string) models.JSON {
	result := make(models.JSON)
	for i, v := range slice {
		result[fmt.Sprintf("%d", i)] = v
	}
	return result
}

// SetHotelRecommendedRequest 设置酒店推荐请求
type SetHotelRecommendedRequest struct {
	IsRecommended bool `json:"is_recommended"`
	Score         int  `json:"score"`
}

// SetHotelRecommended 设置酒店推荐状态
func (s *HotelAdminService) SetHotelRecommended(ctx context.Context, id int64, req *SetHotelRecommendedRequest) error {
	// 验证酒店存在
	_, err := s.hotelRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrHotelNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	return s.hotelRepo.SetRecommended(ctx, id, req.IsRecommended, req.Score)
}

// SetRoomHotRequest 设置房间热门请求
type SetRoomHotRequest struct {
	IsHot bool `json:"is_hot"`
	Rank  int  `json:"rank"`
}

// SetRoomHot 设置房间热门状态
func (s *HotelAdminService) SetRoomHot(ctx context.Context, id int64, req *SetRoomHotRequest) error {
	// 验证房间存在
	_, err := s.roomRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrRoomNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	return s.roomRepo.SetHotStatus(ctx, id, req.IsHot, req.Rank)
}
