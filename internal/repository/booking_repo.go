// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// BookingRepository 预订仓储
type BookingRepository struct {
	db *gorm.DB
}

// NewBookingRepository 创建预订仓储
func NewBookingRepository(db *gorm.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// Create 创建预订
func (r *BookingRepository) Create(ctx context.Context, booking *models.Booking) error {
	return r.db.WithContext(ctx).Create(booking).Error
}

// GetByID 根据 ID 获取预订
func (r *BookingRepository) GetByID(ctx context.Context, id int64) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).First(&booking, id).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// GetByIDWithDetails 根据 ID 获取预订（包含关联信息）
func (r *BookingRepository) GetByIDWithDetails(ctx context.Context, id int64) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Hotel").
		Preload("Room").
		Preload("Device").
		First(&booking, id).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// GetByBookingNo 根据预订号获取预订
func (r *BookingRepository) GetByBookingNo(ctx context.Context, bookingNo string) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).
		Where("booking_no = ?", bookingNo).
		First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// GetByBookingNoWithDetails 根据预订号获取预订（包含关联信息）
func (r *BookingRepository) GetByBookingNoWithDetails(ctx context.Context, bookingNo string) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Hotel").
		Preload("Room").
		Preload("Device").
		Where("booking_no = ?", bookingNo).
		First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// GetByOrderID 根据订单 ID 获取预订
func (r *BookingRepository) GetByOrderID(ctx context.Context, orderID int64) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// GetByVerificationCode 根据核销码获取预订
func (r *BookingRepository) GetByVerificationCode(ctx context.Context, code string) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Hotel").
		Preload("Room").
		Where("verification_code = ?", code).
		First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// GetByUnlockCode 根据开锁码获取预订
func (r *BookingRepository) GetByUnlockCode(ctx context.Context, code string, deviceID int64) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).
		Where("unlock_code = ?", code).
		Where("device_id = ?", deviceID).
		Where("status IN ?", []string{
			models.BookingStatusVerified,
			models.BookingStatusInUse,
		}).
		First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// Update 更新预订
func (r *BookingRepository) Update(ctx context.Context, booking *models.Booking) error {
	return r.db.WithContext(ctx).Save(booking).Error
}

// UpdateFields 更新指定字段
func (r *BookingRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Booking{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新预订状态
func (r *BookingRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).Model(&models.Booking{}).Where("id = ?", id).Update("status", status).Error
}

// List 获取预订列表
func (r *BookingRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Booking, int64, error) {
	var bookings []*models.Booking
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Booking{})

	// 应用过滤条件
	if userID, ok := filters["user_id"].(int64); ok && userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if hotelID, ok := filters["hotel_id"].(int64); ok && hotelID > 0 {
		query = query.Where("hotel_id = ?", hotelID)
	}
	if roomID, ok := filters["room_id"].(int64); ok && roomID > 0 {
		query = query.Where("room_id = ?", roomID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if statuses, ok := filters["statuses"].([]string); ok && len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if bookingNo, ok := filters["booking_no"].(string); ok && bookingNo != "" {
		query = query.Where("booking_no LIKE ?", "%"+bookingNo+"%")
	}
	if startDate, ok := filters["start_date"].(time.Time); ok {
		query = query.Where("check_in_time >= ?", startDate)
	}
	if endDate, ok := filters["end_date"].(time.Time); ok {
		query = query.Where("check_in_time <= ?", endDate)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.
		Preload("Hotel").
		Preload("Room").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&bookings).Error; err != nil {
		return nil, 0, err
	}

	return bookings, total, nil
}

// ListByUser 获取用户的预订列表
func (r *BookingRepository) ListByUser(ctx context.Context, userID int64, offset, limit int, status *string) ([]*models.Booking, int64, error) {
	filters := map[string]interface{}{
		"user_id": userID,
	}
	if status != nil {
		filters["status"] = *status
	}
	return r.List(ctx, offset, limit, filters)
}

// ListByHotel 获取酒店的预订列表
func (r *BookingRepository) ListByHotel(ctx context.Context, hotelID int64, offset, limit int) ([]*models.Booking, int64, error) {
	filters := map[string]interface{}{
		"hotel_id": hotelID,
	}
	return r.List(ctx, offset, limit, filters)
}

// ListByRoom 获取房间的预订列表
func (r *BookingRepository) ListByRoom(ctx context.Context, roomID int64, offset, limit int) ([]*models.Booking, int64, error) {
	filters := map[string]interface{}{
		"room_id": roomID,
	}
	return r.List(ctx, offset, limit, filters)
}

// ListPendingVerification 获取待核销的预订列表
func (r *BookingRepository) ListPendingVerification(ctx context.Context, hotelID int64, offset, limit int) ([]*models.Booking, int64, error) {
	filters := map[string]interface{}{
		"hotel_id": hotelID,
		"status":   models.BookingStatusPaid,
	}
	return r.List(ctx, offset, limit, filters)
}

// ListActiveBookings 获取活跃的预订列表（已支付、已核销、使用中）
func (r *BookingRepository) ListActiveBookings(ctx context.Context, roomID int64, checkIn, checkOut time.Time) ([]*models.Booking, error) {
	var bookings []*models.Booking
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Where("status IN ?", []string{
			models.BookingStatusPaid,
			models.BookingStatusVerified,
			models.BookingStatusInUse,
		}).
		Where("(check_in_time < ? AND check_out_time > ?)", checkOut, checkIn).
		Find(&bookings).Error
	return bookings, err
}

// ListExpiredBookings 获取已过期的预订列表
func (r *BookingRepository) ListExpiredBookings(ctx context.Context, limit int) ([]*models.Booking, error) {
	var bookings []*models.Booking
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("status = ?", models.BookingStatusPaid).
		Where("check_in_time < ?", now).
		Limit(limit).
		Find(&bookings).Error
	return bookings, err
}

// ListToComplete 获取需要标记完成的预订列表
func (r *BookingRepository) ListToComplete(ctx context.Context, limit int) ([]*models.Booking, error) {
	var bookings []*models.Booking
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("status IN ?", []string{
			models.BookingStatusVerified,
			models.BookingStatusInUse,
		}).
		Where("check_out_time < ?", now).
		Limit(limit).
		Find(&bookings).Error
	return bookings, err
}

// Verify 核销预订
func (r *BookingRepository) Verify(ctx context.Context, id int64, verifiedBy int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.BookingStatusVerified,
			"verified_at": now,
			"verified_by": verifiedBy,
		}).Error
}

// Unlock 标记已开锁
func (r *BookingRepository) Unlock(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.BookingStatusInUse,
			"unlocked_at": now,
		}).Error
}

// Complete 标记完成
func (r *BookingRepository) Complete(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       models.BookingStatusCompleted,
			"completed_at": now,
		}).Error
}

// Cancel 取消预订
func (r *BookingRepository) Cancel(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("id = ?", id).
		Update("status", models.BookingStatusCancelled).Error
}

// MarkRefunded 标记已退款
func (r *BookingRepository) MarkRefunded(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("id = ?", id).
		Update("status", models.BookingStatusRefunded).Error
}

// MarkExpired 标记已过期
func (r *BookingRepository) MarkExpired(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("id = ?", id).
		Update("status", models.BookingStatusExpired).Error
}

// CountByUserAndStatus 统计用户指定状态的预订数量
func (r *BookingRepository) CountByUserAndStatus(ctx context.Context, userID int64, statuses []string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("user_id = ?", userID).
		Where("status IN ?", statuses).
		Count(&count).Error
	return count, err
}

// CountTodayBookings 统计今日预订数量
func (r *BookingRepository) CountTodayBookings(ctx context.Context, hotelID int64) (int64, error) {
	var count int64
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("hotel_id = ?", hotelID).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Count(&count).Error
	return count, err
}

// ExistsByRoomAndTimeRange 检查房间在指定时段是否有预订
func (r *BookingRepository) ExistsByRoomAndTimeRange(ctx context.Context, roomID int64, checkIn, checkOut time.Time) (bool, error) {
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
	return count > 0, err
}
