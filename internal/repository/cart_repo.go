// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// CartRepository 购物车仓储
type CartRepository struct {
	db *gorm.DB
}

// NewCartRepository 创建购物车仓储
func NewCartRepository(db *gorm.DB) *CartRepository {
	return &CartRepository{db: db}
}

// Create 创建购物车项
func (r *CartRepository) Create(ctx context.Context, item *models.CartItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// GetByID 根据 ID 获取购物车项
func (r *CartRepository) GetByID(ctx context.Context, id int64) (*models.CartItem, error) {
	var item models.CartItem
	err := r.db.WithContext(ctx).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByUserIDAndProductSku 根据用户ID、商品ID、SKU ID获取购物车项
func (r *CartRepository) GetByUserIDAndProductSku(ctx context.Context, userID, productID int64, skuID *int64) (*models.CartItem, error) {
	var item models.CartItem
	query := r.db.WithContext(ctx).Where("user_id = ? AND product_id = ?", userID, productID)

	if skuID != nil {
		query = query.Where("sku_id = ?", *skuID)
	} else {
		query = query.Where("sku_id IS NULL")
	}

	err := query.First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Update 更新购物车项
func (r *CartRepository) Update(ctx context.Context, item *models.CartItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// UpdateQuantity 更新数量
func (r *CartRepository) UpdateQuantity(ctx context.Context, id int64, quantity int) error {
	return r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("id = ?", id).
		Update("quantity", quantity).Error
}

// UpdateSelected 更新选中状态
func (r *CartRepository) UpdateSelected(ctx context.Context, id int64, selected bool) error {
	return r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("id = ?", id).
		Update("selected", selected).Error
}

// UpdateAllSelected 更新用户所有购物车项的选中状态
func (r *CartRepository) UpdateAllSelected(ctx context.Context, userID int64, selected bool) error {
	return r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("user_id = ?", userID).
		Update("selected", selected).Error
}

// Delete 删除购物车项
func (r *CartRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.CartItem{}, id).Error
}

// DeleteByIDs 批量删除购物车项
func (r *CartRepository) DeleteByIDs(ctx context.Context, userID int64, ids []int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND id IN ?", userID, ids).
		Delete(&models.CartItem{}).Error
}

// DeleteByUserID 删除用户所有购物车项
func (r *CartRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.CartItem{}).Error
}

// DeleteSelected 删除用户选中的购物车项
func (r *CartRepository) DeleteSelected(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND selected = ?", userID, true).
		Delete(&models.CartItem{}).Error
}

// ListByUserID 获取用户购物车列表
func (r *CartRepository) ListByUserID(ctx context.Context, userID int64) ([]*models.CartItem, error) {
	var items []*models.CartItem
	err := r.db.WithContext(ctx).
		Preload("Product").
		Preload("Sku").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

// ListSelectedByUserID 获取用户选中的购物车项
func (r *CartRepository) ListSelectedByUserID(ctx context.Context, userID int64) ([]*models.CartItem, error) {
	var items []*models.CartItem
	err := r.db.WithContext(ctx).
		Preload("Product").
		Preload("Sku").
		Where("user_id = ? AND selected = ?", userID, true).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

// CountByUserID 获取用户购物车数量
func (r *CartRepository) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// SumQuantityByUserID 获取用户购物车商品总数量
func (r *CartRepository) SumQuantityByUserID(ctx context.Context, userID int64) (int, error) {
	var sum int
	err := r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&sum).Error
	return sum, err
}

// IncrementQuantity 增加数量
func (r *CartRepository) IncrementQuantity(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("id = ?", id).
		UpdateColumn("quantity", gorm.Expr("quantity + ?", delta)).
		Error
}
