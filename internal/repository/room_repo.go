// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// RoomRepository 房间仓储
type RoomRepository struct {
	db *gorm.DB
}

// NewRoomRepository 创建房间仓储
func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

// Create 创建房间
func (r *RoomRepository) Create(ctx context.Context, room *models.Room) error {
	return r.db.WithContext(ctx).Create(room).Error
}

// GetByID 根据 ID 获取房间
func (r *RoomRepository) GetByID(ctx context.Context, id int64) (*models.Room, error) {
	var room models.Room
	err := r.db.WithContext(ctx).First(&room, id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// GetByIDWithHotel 根据 ID 获取房间（包含酒店信息）
func (r *RoomRepository) GetByIDWithHotel(ctx context.Context, id int64) (*models.Room, error) {
	var room models.Room
	err := r.db.WithContext(ctx).
		Preload("Hotel").
		First(&room, id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// GetByIDWithTimeSlots 根据 ID 获取房间（包含时段价格）
func (r *RoomRepository) GetByIDWithTimeSlots(ctx context.Context, id int64) (*models.Room, error) {
	var room models.Room
	err := r.db.WithContext(ctx).
		Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("sort ASC, duration_hours ASC")
		}).
		Preload("Hotel").
		First(&room, id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// GetByIDWithDevice 根据 ID 获取房间（包含设备信息）
func (r *RoomRepository) GetByIDWithDevice(ctx context.Context, id int64) (*models.Room, error) {
	var room models.Room
	err := r.db.WithContext(ctx).
		Preload("Device").
		Preload("Hotel").
		First(&room, id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// Update 更新房间
func (r *RoomRepository) Update(ctx context.Context, room *models.Room) error {
	return r.db.WithContext(ctx).Save(room).Error
}

// UpdateFields 更新指定字段
func (r *RoomRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Room{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新房间状态
func (r *RoomRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Room{}).Where("id = ?", id).Update("status", status).Error
}

// List 获取房间列表
func (r *RoomRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Room, int64, error) {
	var rooms []*models.Room
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Room{})

	// 应用过滤条件
	if hotelID, ok := filters["hotel_id"].(int64); ok && hotelID > 0 {
		query = query.Where("hotel_id = ?", hotelID)
	}
	if roomType, ok := filters["room_type"].(string); ok && roomType != "" {
		query = query.Where("room_type = ?", roomType)
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}
	if maxPrice, ok := filters["max_price"].(float64); ok && maxPrice > 0 {
		query = query.Where("hourly_price <= ?", maxPrice)
	}
	if minPrice, ok := filters["min_price"].(float64); ok && minPrice > 0 {
		query = query.Where("hourly_price >= ?", minPrice)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Preload("Hotel").Order("room_no ASC").Offset(offset).Limit(limit).Find(&rooms).Error; err != nil {
		return nil, 0, err
	}

	return rooms, total, nil
}

// ListByHotel 获取酒店下的房间列表
func (r *RoomRepository) ListByHotel(ctx context.Context, hotelID int64, status *int8) ([]*models.Room, error) {
	var rooms []*models.Room
	query := r.db.WithContext(ctx).Where("hotel_id = ?", hotelID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	err := query.Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true).Order("sort ASC, duration_hours ASC")
	}).Order("room_no ASC").Find(&rooms).Error
	return rooms, err
}

// ListAvailableByHotel 获取酒店下的可用房间列表
func (r *RoomRepository) ListAvailableByHotel(ctx context.Context, hotelID int64) ([]*models.Room, error) {
	status := int8(models.RoomStatusActive)
	return r.ListByHotel(ctx, hotelID, &status)
}

// ExistsByRoomNo 检查房间号是否存在
func (r *RoomRepository) ExistsByRoomNo(ctx context.Context, hotelID int64, roomNo string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Room{}).
		Where("hotel_id = ?", hotelID).
		Where("room_no = ?", roomNo).
		Count(&count).Error
	return count > 0, err
}

// Delete 删除房间（软删除）
func (r *RoomRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Room{}, id).Error
}

// GetByDeviceID 根据设备 ID 获取房间
func (r *RoomRepository) GetByDeviceID(ctx context.Context, deviceID int64) (*models.Room, error) {
	var room models.Room
	err := r.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Preload("Hotel").
		First(&room).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// BindDevice 绑定设备到房间
func (r *RoomRepository) BindDevice(ctx context.Context, roomID, deviceID int64) error {
	return r.db.WithContext(ctx).Model(&models.Room{}).
		Where("id = ?", roomID).
		Update("device_id", deviceID).Error
}

// UnbindDevice 解绑设备
func (r *RoomRepository) UnbindDevice(ctx context.Context, roomID int64) error {
	return r.db.WithContext(ctx).Model(&models.Room{}).
		Where("id = ?", roomID).
		Update("device_id", nil).Error
}

// CheckAvailability 检查房间在指定时段是否可用
func (r *RoomRepository) CheckAvailability(ctx context.Context, roomID int64, checkIn, checkOut time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("room_id = ?", roomID).
		Where("status IN ?", []string{
			models.BookingStatusPaid,
			models.BookingStatusVerified,
			models.BookingStatusInUse,
		}).
		Where("(check_in_time < ? AND check_out_time > ?)", checkOut, checkIn).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// RoomTimeSlotRepository 房间时段仓储
type RoomTimeSlotRepository struct {
	db *gorm.DB
}

// NewRoomTimeSlotRepository 创建房间时段仓储
func NewRoomTimeSlotRepository(db *gorm.DB) *RoomTimeSlotRepository {
	return &RoomTimeSlotRepository{db: db}
}

// Create 创建时段价格
func (r *RoomTimeSlotRepository) Create(ctx context.Context, slot *models.RoomTimeSlot) error {
	return r.db.WithContext(ctx).Create(slot).Error
}

// CreateBatch 批量创建时段价格
func (r *RoomTimeSlotRepository) CreateBatch(ctx context.Context, slots []*models.RoomTimeSlot) error {
	if len(slots) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&slots).Error
}

// GetByID 根据 ID 获取时段
func (r *RoomTimeSlotRepository) GetByID(ctx context.Context, id int64) (*models.RoomTimeSlot, error) {
	var slot models.RoomTimeSlot
	err := r.db.WithContext(ctx).First(&slot, id).Error
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

// GetByRoomAndDuration 根据房间 ID 和时长获取时段
func (r *RoomTimeSlotRepository) GetByRoomAndDuration(ctx context.Context, roomID int64, durationHours int) (*models.RoomTimeSlot, error) {
	var slot models.RoomTimeSlot
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Where("duration_hours = ?", durationHours).
		Where("is_active = ?", true).
		First(&slot).Error
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

// Update 更新时段
func (r *RoomTimeSlotRepository) Update(ctx context.Context, slot *models.RoomTimeSlot) error {
	return r.db.WithContext(ctx).Save(slot).Error
}

// UpdateFields 更新指定字段
func (r *RoomTimeSlotRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.RoomTimeSlot{}).Where("id = ?", id).Updates(fields).Error
}

// ListByRoom 获取房间的所有时段
func (r *RoomTimeSlotRepository) ListByRoom(ctx context.Context, roomID int64) ([]*models.RoomTimeSlot, error) {
	var slots []*models.RoomTimeSlot
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("sort ASC, duration_hours ASC").
		Find(&slots).Error
	return slots, err
}

// ListActiveByRoom 获取房间的启用时段
func (r *RoomTimeSlotRepository) ListActiveByRoom(ctx context.Context, roomID int64) ([]*models.RoomTimeSlot, error) {
	var slots []*models.RoomTimeSlot
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Where("is_active = ?", true).
		Order("sort ASC, duration_hours ASC").
		Find(&slots).Error
	return slots, err
}

// Delete 删除时段
func (r *RoomTimeSlotRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.RoomTimeSlot{}, id).Error
}

// DeleteByRoom 删除房间的所有时段
func (r *RoomTimeSlotRepository) DeleteByRoom(ctx context.Context, roomID int64) error {
	return r.db.WithContext(ctx).Where("room_id = ?", roomID).Delete(&models.RoomTimeSlot{}).Error
}

// ListHotRooms 获取全站热门房型
func (r *RoomRepository) ListHotRooms(ctx context.Context, limit int) ([]*models.Room, error) {
	var rooms []*models.Room
	err := r.db.WithContext(ctx).
		Where("is_hot = ?", true).
		Where("status = ?", models.RoomStatusActive).
		Preload("Hotel").
		Order("hot_score DESC, booking_count DESC, id DESC").
		Limit(limit).
		Find(&rooms).Error
	return rooms, err
}

// ListHotRoomsByHotel 获取酒店内热门房型
func (r *RoomRepository) ListHotRoomsByHotel(ctx context.Context, hotelID int64, limit int) ([]*models.Room, error) {
	var rooms []*models.Room
	err := r.db.WithContext(ctx).
		Where("hotel_id = ?", hotelID).
		Where("is_hot = ?", true).
		Where("status = ?", models.RoomStatusActive).
		Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("sort ASC, duration_hours ASC")
		}).
		Order("hot_score DESC, booking_count DESC, id DESC").
		Limit(limit).
		Find(&rooms).Error
	return rooms, err
}

// IncrementBookingCount 增加预订次数（使用 gorm.Expr 避免并发问题）
func (r *RoomRepository) IncrementBookingCount(ctx context.Context, roomID int64) error {
	return r.db.WithContext(ctx).Model(&models.Room{}).
		Where("id = ?", roomID).
		UpdateColumn("booking_count", gorm.Expr("booking_count + 1")).Error
}

// SetHotStatus 设置热门状态
func (r *RoomRepository) SetHotStatus(ctx context.Context, roomID int64, isHot bool, rank int) error {
	fields := map[string]interface{}{
		"is_hot":   isHot,
		"hot_rank": rank,
	}
	return r.db.WithContext(ctx).Model(&models.Room{}).Where("id = ?", roomID).Updates(fields).Error
}

// RecalculateHotScore 重新计算热门分数
// 分数计算公式：booking_count * 10 + average_rating * 20 + review_count * 5
func (r *RoomRepository) RecalculateHotScore(ctx context.Context, roomID int64) error {
	return r.db.WithContext(ctx).Model(&models.Room{}).
		Where("id = ?", roomID).
		UpdateColumn("hot_score", gorm.Expr("booking_count * 10 + average_rating * 20 + review_count * 5")).Error
}

// UpdateRating 更新评分信息
func (r *RoomRepository) UpdateRating(ctx context.Context, roomID int64, avgRating float64, reviewCount int) error {
	fields := map[string]interface{}{
		"average_rating": avgRating,
		"review_count":   reviewCount,
	}
	return r.db.WithContext(ctx).Model(&models.Room{}).Where("id = ?", roomID).Updates(fields).Error
}
