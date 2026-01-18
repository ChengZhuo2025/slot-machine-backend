// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// CouponRepository 优惠券仓储
type CouponRepository struct {
	db *gorm.DB
}

// NewCouponRepository 创建优惠券仓储
func NewCouponRepository(db *gorm.DB) *CouponRepository {
	return &CouponRepository{db: db}
}

// Create 创建优惠券
func (r *CouponRepository) Create(ctx context.Context, coupon *models.Coupon) error {
	return r.db.WithContext(ctx).Create(coupon).Error
}

// GetByID 根据 ID 获取优惠券
func (r *CouponRepository) GetByID(ctx context.Context, id int64) (*models.Coupon, error) {
	var coupon models.Coupon
	err := r.db.WithContext(ctx).First(&coupon, id).Error
	if err != nil {
		return nil, err
	}
	return &coupon, nil
}

// Update 更新优惠券
func (r *CouponRepository) Update(ctx context.Context, coupon *models.Coupon) error {
	return r.db.WithContext(ctx).Save(coupon).Error
}

// UpdateFields 更新指定字段
func (r *CouponRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Coupon{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除优惠券
func (r *CouponRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Coupon{}, id).Error
}

// CouponListParams 优惠券列表查询参数
type CouponListParams struct {
	Offset         int
	Limit          int
	Status         *int8
	Type           string
	ApplicableType string
	Keyword        string
	StartTimeFrom  *time.Time
	StartTimeTo    *time.Time
	EndTimeFrom    *time.Time
	EndTimeTo      *time.Time
}

// List 获取优惠券列表
func (r *CouponRepository) List(ctx context.Context, params CouponListParams) ([]*models.Coupon, int64, error) {
	var coupons []*models.Coupon
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Coupon{})

	// 过滤条件
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.ApplicableType != "" {
		query = query.Where("applicable_scope = ?", params.ApplicableType)
	}
	if params.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+params.Keyword+"%")
	}
	if params.StartTimeFrom != nil {
		query = query.Where("start_time >= ?", *params.StartTimeFrom)
	}
	if params.StartTimeTo != nil {
		query = query.Where("start_time <= ?", *params.StartTimeTo)
	}
	if params.EndTimeFrom != nil {
		query = query.Where("end_time >= ?", *params.EndTimeFrom)
	}
	if params.EndTimeTo != nil {
		query = query.Where("end_time <= ?", *params.EndTimeTo)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("created_at DESC").Offset(params.Offset).Limit(params.Limit).Find(&coupons).Error; err != nil {
		return nil, 0, err
	}

	return coupons, total, nil
}

// ListActive 获取有效优惠券列表（用户端）
func (r *CouponRepository) ListActive(ctx context.Context, offset, limit int) ([]*models.Coupon, int64, error) {
	var coupons []*models.Coupon
	var total int64
	now := time.Now()

	query := r.db.WithContext(ctx).Model(&models.Coupon{}).
		Where("status = ?", models.CouponStatusActive).
		Where("start_time <= ?", now).
		Where("end_time >= ?", now).
		Where("total_count > issued_count")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&coupons).Error; err != nil {
		return nil, 0, err
	}

	return coupons, total, nil
}

// ListAvailableForUser 获取用户可领取的优惠券列表
func (r *CouponRepository) ListAvailableForUser(ctx context.Context, userID int64, offset, limit int) ([]*models.Coupon, int64, error) {
	var coupons []*models.Coupon
	var total int64
	now := time.Now()

	// 获取用户已领取优惠券数量的子查询
	subQuery := r.db.Model(&models.UserCoupon{}).
		Select("coupon_id, COUNT(*) as count").
		Where("user_id = ?", userID).
		Group("coupon_id")

	query := r.db.WithContext(ctx).Model(&models.Coupon{}).
		Joins("LEFT JOIN (?) AS uc ON uc.coupon_id = coupons.id", subQuery).
		Where("coupons.status = ?", models.CouponStatusActive).
		Where("coupons.start_time <= ?", now).
		Where("coupons.end_time >= ?", now).
		Where("coupons.total_count > coupons.issued_count").
		Where("(uc.count IS NULL OR uc.count < coupons.per_user_limit)")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("coupons.created_at DESC").Offset(offset).Limit(limit).Find(&coupons).Error; err != nil {
		return nil, 0, err
	}

	return coupons, total, nil
}

// IncrementIssuedCount 增加已发放数量
func (r *CouponRepository) IncrementIssuedCount(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Model(&models.Coupon{}).
		Where("id = ? AND total_count > issued_count", id).
		UpdateColumn("issued_count", gorm.Expr("issued_count + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IncrementUsedCount 增加已使用数量
func (r *CouponRepository) IncrementUsedCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Coupon{}).
		Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

// DecrementIssuedCount 减少已发放数量（用于退券）
func (r *CouponRepository) DecrementIssuedCount(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Model(&models.Coupon{}).
		Where("id = ? AND issued_count > 0", id).
		UpdateColumn("issued_count", gorm.Expr("issued_count - 1"))
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DecrementUsedCount 减少已使用数量（用于退款）
func (r *CouponRepository) DecrementUsedCount(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Model(&models.Coupon{}).
		Where("id = ? AND used_count > 0", id).
		UpdateColumn("used_count", gorm.Expr("used_count - 1"))
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetUserReceivedCount 获取用户已领取该优惠券的数量
func (r *CouponRepository) GetUserReceivedCount(ctx context.Context, userID, couponID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("user_id = ? AND coupon_id = ?", userID, couponID).
		Count(&count).Error
	return count, err
}

// ListByApplicableType 根据适用类型获取有效优惠券
func (r *CouponRepository) ListByApplicableType(ctx context.Context, applicableType string) ([]*models.Coupon, error) {
	var coupons []*models.Coupon
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("status = ?", models.CouponStatusActive).
		Where("start_time <= ?", now).
		Where("end_time >= ?", now).
		Where("applicable_scope = ? OR applicable_scope = ?", applicableType, models.CouponScopeAll).
		Order("value DESC").
		Find(&coupons).Error

	return coupons, err
}
