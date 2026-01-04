// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// ReviewRepository 评价仓储
type ReviewRepository struct {
	db *gorm.DB
}

// NewReviewRepository 创建评价仓储
func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// Create 创建评价
func (r *ReviewRepository) Create(ctx context.Context, review *models.Review) error {
	return r.db.WithContext(ctx).Create(review).Error
}

// GetByID 根据 ID 获取评价
func (r *ReviewRepository) GetByID(ctx context.Context, id int64) (*models.Review, error) {
	var review models.Review
	err := r.db.WithContext(ctx).First(&review, id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// GetByIDWithUser 根据 ID 获取评价（包含用户信息）
func (r *ReviewRepository) GetByIDWithUser(ctx context.Context, id int64) (*models.Review, error) {
	var review models.Review
	err := r.db.WithContext(ctx).Preload("User").First(&review, id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// GetByOrderAndProduct 根据订单ID和商品ID获取评价
func (r *ReviewRepository) GetByOrderAndProduct(ctx context.Context, orderID, productID int64) (*models.Review, error) {
	var review models.Review
	err := r.db.WithContext(ctx).
		Where("order_id = ? AND product_id = ?", orderID, productID).
		First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// Update 更新评价
func (r *ReviewRepository) Update(ctx context.Context, review *models.Review) error {
	return r.db.WithContext(ctx).Save(review).Error
}

// UpdateFields 更新指定字段
func (r *ReviewRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Review{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除评价
func (r *ReviewRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Review{}, id).Error
}

// ReviewListParams 评价列表查询参数
type ReviewListParams struct {
	Offset    int
	Limit     int
	ProductID int64
	UserID    int64
	OrderID   int64
	Rating    *int16
	Status    *int16
}

// List 获取评价列表
func (r *ReviewRepository) List(ctx context.Context, params ReviewListParams) ([]*models.Review, int64, error) {
	var reviews []*models.Review
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Review{})

	// 过滤条件
	if params.ProductID > 0 {
		query = query.Where("product_id = ?", params.ProductID)
	}
	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.OrderID > 0 {
		query = query.Where("order_id = ?", params.OrderID)
	}
	if params.Rating != nil {
		query = query.Where("rating = ?", *params.Rating)
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Preload("User").Order("created_at DESC").Offset(params.Offset).Limit(params.Limit).Find(&reviews).Error; err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

// ListByProductID 根据商品ID获取评价列表
func (r *ReviewRepository) ListByProductID(ctx context.Context, productID int64, offset, limit int) ([]*models.Review, int64, error) {
	status := int16(models.ReviewStatusVisible)
	return r.List(ctx, ReviewListParams{
		Offset:    offset,
		Limit:     limit,
		ProductID: productID,
		Status:    &status,
	})
}

// ListByUserID 根据用户ID获取评价列表
func (r *ReviewRepository) ListByUserID(ctx context.Context, userID int64, offset, limit int) ([]*models.Review, int64, error) {
	return r.List(ctx, ReviewListParams{
		Offset: offset,
		Limit:  limit,
		UserID: userID,
	})
}

// CountByProductID 根据商品ID统计评价数量
func (r *ReviewRepository) CountByProductID(ctx context.Context, productID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Review{}).
		Where("product_id = ? AND status = ?", productID, models.ReviewStatusVisible).
		Count(&count).Error
	return count, err
}

// GetAverageRatingByProductID 根据商品ID获取平均评分
func (r *ReviewRepository) GetAverageRatingByProductID(ctx context.Context, productID int64) (float64, error) {
	var avg float64
	err := r.db.WithContext(ctx).Model(&models.Review{}).
		Where("product_id = ? AND status = ?", productID, models.ReviewStatusVisible).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&avg).Error
	return avg, err
}

// GetRatingDistribution 获取评分分布
func (r *ReviewRepository) GetRatingDistribution(ctx context.Context, productID int64) (map[int16]int64, error) {
	type RatingCount struct {
		Rating int16
		Count  int64
	}
	var results []RatingCount

	err := r.db.WithContext(ctx).Model(&models.Review{}).
		Where("product_id = ? AND status = ?", productID, models.ReviewStatusVisible).
		Select("rating, COUNT(*) as count").
		Group("rating").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	distribution := make(map[int16]int64)
	for _, r := range results {
		distribution[r.Rating] = r.Count
	}
	return distribution, nil
}

// ExistsByOrderAndProduct 检查订单商品是否已评价
func (r *ReviewRepository) ExistsByOrderAndProduct(ctx context.Context, orderID, productID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Review{}).
		Where("order_id = ? AND product_id = ?", orderID, productID).
		Count(&count).Error
	return count > 0, err
}
