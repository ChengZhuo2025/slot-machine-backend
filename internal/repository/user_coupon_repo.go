// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// UserCouponRepository 用户优惠券仓储
type UserCouponRepository struct {
	db *gorm.DB
}

// NewUserCouponRepository 创建用户优惠券仓储
func NewUserCouponRepository(db *gorm.DB) *UserCouponRepository {
	return &UserCouponRepository{db: db}
}

// Create 创建用户优惠券
func (r *UserCouponRepository) Create(ctx context.Context, userCoupon *models.UserCoupon) error {
	return r.db.WithContext(ctx).Create(userCoupon).Error
}

// GetByID 根据 ID 获取用户优惠券
func (r *UserCouponRepository) GetByID(ctx context.Context, id int64) (*models.UserCoupon, error) {
	var userCoupon models.UserCoupon
	err := r.db.WithContext(ctx).First(&userCoupon, id).Error
	if err != nil {
		return nil, err
	}
	return &userCoupon, nil
}

// GetByIDWithCoupon 根据 ID 获取用户优惠券（包含优惠券详情）
func (r *UserCouponRepository) GetByIDWithCoupon(ctx context.Context, id int64) (*models.UserCoupon, error) {
	var userCoupon models.UserCoupon
	err := r.db.WithContext(ctx).Preload("Coupon").First(&userCoupon, id).Error
	if err != nil {
		return nil, err
	}
	return &userCoupon, nil
}

// GetByUserIDAndCouponID 根据用户ID和优惠券ID获取用户优惠券
func (r *UserCouponRepository) GetByUserIDAndCouponID(ctx context.Context, userID, couponID int64) (*models.UserCoupon, error) {
	var userCoupon models.UserCoupon
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND coupon_id = ?", userID, couponID).
		First(&userCoupon).Error
	if err != nil {
		return nil, err
	}
	return &userCoupon, nil
}

// Update 更新用户优惠券
func (r *UserCouponRepository) Update(ctx context.Context, userCoupon *models.UserCoupon) error {
	return r.db.WithContext(ctx).Save(userCoupon).Error
}

// UpdateFields 更新指定字段
func (r *UserCouponRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.UserCoupon{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除用户优惠券
func (r *UserCouponRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.UserCoupon{}, id).Error
}

// UserCouponListParams 用户优惠券列表查询参数
type UserCouponListParams struct {
	Offset   int
	Limit    int
	UserID   int64
	CouponID int64
	Status   *int8
}

// List 获取用户优惠券列表
func (r *UserCouponRepository) List(ctx context.Context, params UserCouponListParams) ([]*models.UserCoupon, int64, error) {
	var userCoupons []*models.UserCoupon
	var total int64

	query := r.db.WithContext(ctx).Model(&models.UserCoupon{})

	// 过滤条件
	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.CouponID > 0 {
		query = query.Where("coupon_id = ?", params.CouponID)
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Preload("Coupon").Order("received_at DESC").Offset(params.Offset).Limit(params.Limit).Find(&userCoupons).Error; err != nil {
		return nil, 0, err
	}

	return userCoupons, total, nil
}

// ListByUserID 获取用户的优惠券列表
func (r *UserCouponRepository) ListByUserID(ctx context.Context, userID int64, offset, limit int) ([]*models.UserCoupon, int64, error) {
	return r.List(ctx, UserCouponListParams{
		UserID: userID,
		Offset: offset,
		Limit:  limit,
	})
}

// ListByUserIDAndStatus 根据用户ID和状态获取优惠券列表
func (r *UserCouponRepository) ListByUserIDAndStatus(ctx context.Context, userID int64, status int8, offset, limit int) ([]*models.UserCoupon, int64, error) {
	return r.List(ctx, UserCouponListParams{
		UserID: userID,
		Status: &status,
		Offset: offset,
		Limit:  limit,
	})
}

// ListAvailableByUserID 获取用户可用的优惠券列表
func (r *UserCouponRepository) ListAvailableByUserID(ctx context.Context, userID int64, offset, limit int) ([]*models.UserCoupon, int64, error) {
	var userCoupons []*models.UserCoupon
	var total int64
	now := time.Now()

	query := r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("user_id = ?", userID).
		Where("status = ?", models.UserCouponStatusUnused).
		Where("expired_at > ?", now)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("Coupon").Order("expired_at ASC").Offset(offset).Limit(limit).Find(&userCoupons).Error; err != nil {
		return nil, 0, err
	}

	return userCoupons, total, nil
}

// ListAvailableForOrder 获取用户可用于指定类型订单的优惠券
func (r *UserCouponRepository) ListAvailableForOrder(ctx context.Context, userID int64, orderType string, orderAmount float64) ([]*models.UserCoupon, error) {
	var userCoupons []*models.UserCoupon
	now := time.Now()

	err := r.db.WithContext(ctx).
		Preload("Coupon").
		Joins("INNER JOIN coupons ON coupons.id = user_coupons.coupon_id").
		Where("user_coupons.user_id = ?", userID).
		Where("user_coupons.status = ?", models.UserCouponStatusUnused).
		Where("user_coupons.expired_at > ?", now).
		Where("coupons.min_amount <= ?", orderAmount).
		Where("coupons.applicable_scope = ? OR coupons.applicable_scope = ?", models.CouponScopeAll, orderType).
		Order("coupons.value DESC").
		Find(&userCoupons).Error

	return userCoupons, err
}

// MarkAsUsed 标记为已使用
func (r *UserCouponRepository) MarkAsUsed(ctx context.Context, id int64, orderID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("id = ? AND status = ?", id, models.UserCouponStatusUnused).
		Updates(map[string]interface{}{
			"status":   models.UserCouponStatusUsed,
			"order_id": orderID,
			"used_at":  now,
		}).Error
}

// MarkAsUnused 标记为未使用（退款时恢复）
func (r *UserCouponRepository) MarkAsUnused(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":   models.UserCouponStatusUnused,
			"order_id": nil,
			"used_at":  nil,
		}).Error
}

// MarkAsExpired 标记为已过期
func (r *UserCouponRepository) MarkAsExpired(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("id = ? AND status = ?", id, models.UserCouponStatusUnused).
		Update("status", models.UserCouponStatusExpired).Error
}

// BatchMarkAsExpired 批量标记过期优惠券
func (r *UserCouponRepository) BatchMarkAsExpired(ctx context.Context) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("status = ? AND expired_at <= ?", models.UserCouponStatusUnused, now).
		Update("status", models.UserCouponStatusExpired)
	return result.RowsAffected, result.Error
}

// CountByUserIDAndCouponID 统计用户已领取某优惠券的数量
func (r *UserCouponRepository) CountByUserIDAndCouponID(ctx context.Context, userID, couponID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("user_id = ? AND coupon_id = ?", userID, couponID).
		Count(&count).Error
	return count, err
}

// CountByUserID 统计用户优惠券数量
func (r *UserCouponRepository) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// CountAvailableByUserID 统计用户可用优惠券数量
func (r *UserCouponRepository) CountAvailableByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("user_id = ? AND status = ? AND expired_at > ?", userID, models.UserCouponStatusUnused, now).
		Count(&count).Error
	return count, err
}

// GetByOrderID 根据订单ID获取使用的优惠券
func (r *UserCouponRepository) GetByOrderID(ctx context.Context, orderID int64) (*models.UserCoupon, error) {
	var userCoupon models.UserCoupon
	err := r.db.WithContext(ctx).Preload("Coupon").Where("order_id = ?", orderID).First(&userCoupon).Error
	if err != nil {
		return nil, err
	}
	return &userCoupon, nil
}
