// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"smart-locker-backend/internal/models"
)

// OrderRepository 订单仓储
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建订单仓储
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create 创建订单
func (r *OrderRepository) Create(ctx context.Context, order *models.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// GetByID 根据 ID 获取订单
func (r *OrderRepository) GetByID(ctx context.Context, id int64) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByIDWithItems 根据 ID 获取订单（包含订单项）
func (r *OrderRepository) GetByIDWithItems(ctx context.Context, id int64) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Items.Product").
		First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByOrderNo 根据订单号获取订单
func (r *OrderRepository) GetByOrderNo(ctx context.Context, orderNo string) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// Update 更新订单
func (r *OrderRepository) Update(ctx context.Context, order *models.Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}

// UpdateFields 更新指定字段
func (r *OrderRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Order{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新订单状态
func (r *OrderRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Order{}).Where("id = ?", id).Update("status", status).Error
}

// ListByUser 获取用户订单列表
func (r *OrderRepository) ListByUser(ctx context.Context, userID int64, offset, limit int, orderType string, status *int8) ([]*models.Order, int64, error) {
	var orders []*models.Order
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID)

	if orderType != "" {
		query = query.Where("type = ?", orderType)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("Items").
		Order("id DESC").Offset(offset).Limit(limit).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// List 获取订单列表（管理端）
func (r *OrderRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Order, int64, error) {
	var orders []*models.Order
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Order{})

	if userID, ok := filters["user_id"].(int64); ok && userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if orderType, ok := filters["type"].(string); ok && orderType != "" {
		query = query.Where("type = ?", orderType)
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}
	if orderNo, ok := filters["order_no"].(string); ok && orderNo != "" {
		query = query.Where("order_no LIKE ?", "%"+orderNo+"%")
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

	if err := query.Preload("User").Preload("Items").
		Order("id DESC").Offset(offset).Limit(limit).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetExpiredPending 获取过期的待支付订单
func (r *OrderRepository) GetExpiredPending(ctx context.Context, limit int) ([]*models.Order, error) {
	var orders []*models.Order
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("status = ?", models.OrderStatusPending).
		Where("expired_at < ?", now).
		Limit(limit).
		Find(&orders).Error
	return orders, err
}

// GetForUpdate 获取订单（加锁）
func (r *OrderRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, id int64) (*models.Order, error) {
	var order models.Order
	err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// CreateOrderItem 创建订单项
func (r *OrderRepository) CreateOrderItem(ctx context.Context, item *models.OrderItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// CreateOrderItems 批量创建订单项
func (r *OrderRepository) CreateOrderItems(ctx context.Context, items []*models.OrderItem) error {
	return r.db.WithContext(ctx).Create(&items).Error
}

// CountByStatus 统计各状态订单数量
func (r *OrderRepository) CountByStatus(ctx context.Context, userID int64) (map[int8]int64, error) {
	type Result struct {
		Status int8
		Count  int64
	}

	var results []Result
	query := r.db.WithContext(ctx).Model(&models.Order{}).
		Select("status, count(*) as count")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Group("status").Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[int8]int64)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}
