// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// RentalRepository 租借仓储
type RentalRepository struct {
	db *gorm.DB
}

// NewRentalRepository 创建租借仓储
func NewRentalRepository(db *gorm.DB) *RentalRepository {
	return &RentalRepository{db: db}
}

// Create 创建租借订单
func (r *RentalRepository) Create(ctx context.Context, rental *models.Rental) error {
	return r.db.WithContext(ctx).Create(rental).Error
}

// GetByID 根据 ID 获取租借订单
func (r *RentalRepository) GetByID(ctx context.Context, id int64) (*models.Rental, error) {
	var rental models.Rental
	err := r.db.WithContext(ctx).First(&rental, id).Error
	if err != nil {
		return nil, err
	}
	return &rental, nil
}

// GetByIDWithRelations 根据 ID 获取租借订单（包含关联）
func (r *RentalRepository) GetByIDWithRelations(ctx context.Context, id int64) (*models.Rental, error) {
	var rental models.Rental
	err := r.db.WithContext(ctx).
		Preload("Device").
		Preload("Device.Venue").
		Preload("Pricing").
		First(&rental, id).Error
	if err != nil {
		return nil, err
	}
	return &rental, nil
}

// GetByRentalNo 根据订单号获取租借订单
func (r *RentalRepository) GetByRentalNo(ctx context.Context, rentalNo string) (*models.Rental, error) {
	var rental models.Rental
	err := r.db.WithContext(ctx).Where("rental_no = ?", rentalNo).First(&rental).Error
	if err != nil {
		return nil, err
	}
	return &rental, nil
}

// GetByRentalNoWithRelations 根据订单号获取租借订单（包含关联）
func (r *RentalRepository) GetByRentalNoWithRelations(ctx context.Context, rentalNo string) (*models.Rental, error) {
	var rental models.Rental
	err := r.db.WithContext(ctx).
		Preload("Device").
		Preload("Device.Venue").
		Preload("Pricing").
		Where("rental_no = ?", rentalNo).
		First(&rental).Error
	if err != nil {
		return nil, err
	}
	return &rental, nil
}

// Update 更新租借订单
func (r *RentalRepository) Update(ctx context.Context, rental *models.Rental) error {
	return r.db.WithContext(ctx).Save(rental).Error
}

// UpdateFields 更新指定字段
func (r *RentalRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Rental{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新租借状态
func (r *RentalRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Rental{}).Where("id = ?", id).Update("status", status).Error
}

// ListByUser 获取用户的租借列表
func (r *RentalRepository) ListByUser(ctx context.Context, userID int64, offset, limit int, status *int8) ([]*models.Rental, int64, error) {
	var rentals []*models.Rental
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Rental{}).Where("user_id = ?", userID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("Device").Preload("Pricing").
		Order("id DESC").Offset(offset).Limit(limit).
		Find(&rentals).Error; err != nil {
		return nil, 0, err
	}

	return rentals, total, nil
}

// ListByDevice 获取设备的租借列表
func (r *RentalRepository) ListByDevice(ctx context.Context, deviceID int64, offset, limit int) ([]*models.Rental, int64, error) {
	var rentals []*models.Rental
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Rental{}).Where("device_id = ?", deviceID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&rentals).Error; err != nil {
		return nil, 0, err
	}

	return rentals, total, nil
}

// GetCurrentByDevice 获取设备当前租借
func (r *RentalRepository) GetCurrentByDevice(ctx context.Context, deviceID int64) (*models.Rental, error) {
	var rental models.Rental
	err := r.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Where("status IN ?", []int8{
			models.RentalStatusPaid,
			models.RentalStatusInUse,
		}).
		First(&rental).Error
	if err != nil {
		return nil, err
	}
	return &rental, nil
}

// GetActiveByUser 获取用户当前进行中的租借
func (r *RentalRepository) GetActiveByUser(ctx context.Context, userID int64) (*models.Rental, error) {
	var rental models.Rental
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("status IN ?", []int8{
			models.RentalStatusPending,
			models.RentalStatusPaid,
			models.RentalStatusInUse,
		}).
		First(&rental).Error
	if err != nil {
		return nil, err
	}
	return &rental, nil
}

// HasActiveRental 检查用户是否有进行中的租借
func (r *RentalRepository) HasActiveRental(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Rental{}).
		Where("user_id = ?", userID).
		Where("status IN ?", []int8{
			models.RentalStatusPending,
			models.RentalStatusPaid,
			models.RentalStatusInUse,
		}).
		Count(&count).Error
	return count > 0, err
}

// List 获取租借列表（管理端）
func (r *RentalRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Rental, int64, error) {
	var rentals []*models.Rental
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Rental{})

	if userID, ok := filters["user_id"].(int64); ok && userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if deviceID, ok := filters["device_id"].(int64); ok && deviceID > 0 {
		query = query.Where("device_id = ?", deviceID)
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}
	if rentalNo, ok := filters["rental_no"].(string); ok && rentalNo != "" {
		query = query.Where("rental_no LIKE ?", "%"+rentalNo+"%")
	}
	if startDate, ok := filters["start_date"].(time.Time); ok {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate, ok := filters["end_date"].(time.Time); ok {
		query = query.Where("created_at <= ?", endDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("User").Preload("Device").
		Order("id DESC").Offset(offset).Limit(limit).
		Find(&rentals).Error; err != nil {
		return nil, 0, err
	}

	return rentals, total, nil
}

// GetExpiredPending 获取过期的待支付租借
func (r *RentalRepository) GetExpiredPending(ctx context.Context, expiredBefore time.Time, limit int) ([]*models.Rental, error) {
	var rentals []*models.Rental
	err := r.db.WithContext(ctx).
		Where("status = ?", models.RentalStatusPending).
		Where("created_at < ?", expiredBefore).
		Limit(limit).
		Find(&rentals).Error
	return rentals, err
}

// GetOverdue 获取超时未还的租借
func (r *RentalRepository) GetOverdue(ctx context.Context, limit int) ([]*models.Rental, error) {
	var rentals []*models.Rental
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("status = ?", models.RentalStatusInUse).
		Where("end_time < ?", now).
		Limit(limit).
		Find(&rentals).Error
	return rentals, err
}

// CountByStatus 统计各状态租借数量
func (r *RentalRepository) CountByStatus(ctx context.Context) (map[int8]int64, error) {
	type Result struct {
		Status int8
		Count  int64
	}

	var results []Result
	err := r.db.WithContext(ctx).Model(&models.Rental{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[int8]int64)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}

// GetForUpdate 获取租借订单（加锁）
func (r *RentalRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, id int64) (*models.Rental, error) {
	var rental models.Rental
	err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").First(&rental, id).Error
	if err != nil {
		return nil, err
	}
	return &rental, nil
}
