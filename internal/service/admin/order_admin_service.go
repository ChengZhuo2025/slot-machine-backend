// Package admin 管理端服务
package admin

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// OrderAdminService 订单管理服务
type OrderAdminService struct {
	orderRepo *repository.OrderRepository
	db        *gorm.DB
}

// NewOrderAdminService 创建订单管理服务
func NewOrderAdminService(db *gorm.DB, orderRepo *repository.OrderRepository) *OrderAdminService {
	return &OrderAdminService{
		orderRepo: orderRepo,
		db:        db,
	}
}

// OrderListFilters 订单列表筛选条件
type OrderListFilters struct {
	OrderNo   string
	UserID    int64
	Type      string
	Status    string
	StartDate *time.Time
	EndDate   *time.Time
}

// OrderListResponse 订单列表响应
type OrderListResponse struct {
	ID             int64      `json:"id"`
	OrderNo        string     `json:"order_no"`
	UserID         int64      `json:"user_id"`
	UserPhone      string     `json:"user_phone"`
	Type           string     `json:"type"`
	OriginalAmount float64    `json:"original_amount"`
	DiscountAmount float64    `json:"discount_amount"`
	ActualAmount   float64    `json:"actual_amount"`
	Status         string     `json:"status"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// List 获取订单列表
func (s *OrderAdminService) List(ctx context.Context, page, pageSize int, filters *OrderListFilters) ([]*OrderListResponse, int64, error) {
	offset := (page - 1) * pageSize

	query := s.db.WithContext(ctx).Model(&models.Order{})

	if filters != nil {
		if filters.OrderNo != "" {
			query = query.Where("order_no LIKE ?", "%"+filters.OrderNo+"%")
		}
		if filters.UserID > 0 {
			query = query.Where("user_id = ?", filters.UserID)
		}
		if filters.Type != "" {
			query = query.Where("type = ?", filters.Type)
		}
		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}
		if filters.StartDate != nil {
			query = query.Where("created_at >= ?", *filters.StartDate)
		}
		if filters.EndDate != nil {
			query = query.Where("created_at <= ?", *filters.EndDate)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orders []*models.Order
	if err := query.Preload("User").
		Order("id DESC").Offset(offset).Limit(pageSize).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	results := make([]*OrderListResponse, len(orders))
	for i, order := range orders {
		results[i] = s.toOrderListResponse(order)
	}

	return results, total, nil
}

// GetByID 根据 ID 获取订单详情
func (s *OrderAdminService) GetByID(ctx context.Context, id int64) (*models.Order, error) {
	var order models.Order
	err := s.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByOrderNo 根据订单号获取订单详情
func (s *OrderAdminService) GetByOrderNo(ctx context.Context, orderNo string) (*models.Order, error) {
	var order models.Order
	err := s.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		Where("order_no = ?", orderNo).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// CancelOrder 取消订单
func (s *OrderAdminService) CancelOrder(ctx context.Context, id int64, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}

		// 只能取消待支付的订单
		if order.Status != models.OrderStatusPending {
			return fmt.Errorf("只能取消待支付的订单")
		}

		now := time.Now()
		return tx.Model(&order).Updates(map[string]interface{}{
			"status":        models.OrderStatusCancelled,
			"cancelled_at":  now,
			"cancel_reason": reason,
		}).Error
	})
}

// ShipOrder 发货（商城订单）
func (s *OrderAdminService) ShipOrder(ctx context.Context, id int64, expressCompany, expressNo string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}

		// 检查订单类型和状态
		if order.Type != models.OrderTypeMall {
			return fmt.Errorf("只能对商城订单执行发货操作")
		}
		if order.Status != models.OrderStatusPendingShip {
			return fmt.Errorf("只能对待发货订单执行发货操作")
		}

		now := time.Now()
		return tx.Model(&order).Updates(map[string]interface{}{
			"status":          models.OrderStatusShipped,
			"express_company": expressCompany,
			"express_no":      expressNo,
			"shipped_at":      now,
		}).Error
	})
}

// ConfirmReceipt 确认收货（商城订单）
func (s *OrderAdminService) ConfirmReceipt(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}

		if order.Status != models.OrderStatusShipped {
			return fmt.Errorf("只能对已发货订单确认收货")
		}

		now := time.Now()
		return tx.Model(&order).Updates(map[string]interface{}{
			"status":       models.OrderStatusCompleted,
			"received_at":  now,
			"completed_at": now,
		}).Error
	})
}

// AddRemark 添加备注
func (s *OrderAdminService) AddRemark(ctx context.Context, id int64, remark string) error {
	return s.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ?", id).
		Update("remark", remark).Error
}

// OrderStatistics 订单统计
type OrderStatistics struct {
	TotalOrders     int64                `json:"total_orders"`
	TodayOrders     int64                `json:"today_orders"`
	TotalRevenue    float64              `json:"total_revenue"`
	TodayRevenue    float64              `json:"today_revenue"`
	StatusCounts    map[string]int64     `json:"status_counts"`
	TypeCounts      map[string]int64     `json:"type_counts"`
}

// GetStatistics 获取订单统计
func (s *OrderAdminService) GetStatistics(ctx context.Context) (*OrderStatistics, error) {
	stats := &OrderStatistics{
		StatusCounts: make(map[string]int64),
		TypeCounts:   make(map[string]int64),
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 总订单数
	s.db.WithContext(ctx).Model(&models.Order{}).Count(&stats.TotalOrders)

	// 今日订单
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("created_at >= ?", today).
		Count(&stats.TodayOrders)

	// 总收入（已完成订单）
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("status = ?", models.OrderStatusCompleted).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&stats.TotalRevenue)

	// 今日收入
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("status = ? AND completed_at >= ?", models.OrderStatusCompleted, today).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&stats.TodayRevenue)

	// 状态统计
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	s.db.WithContext(ctx).Model(&models.Order{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts)

	for _, sc := range statusCounts {
		stats.StatusCounts[sc.Status] = sc.Count
	}

	// 类型统计
	type TypeCount struct {
		Type  string
		Count int64
	}
	var typeCounts []TypeCount
	s.db.WithContext(ctx).Model(&models.Order{}).
		Select("type, COUNT(*) as count").
		Group("type").
		Find(&typeCounts)

	for _, tc := range typeCounts {
		stats.TypeCounts[tc.Type] = tc.Count
	}

	return stats, nil
}

// toOrderListResponse 转换为列表响应
func (s *OrderAdminService) toOrderListResponse(order *models.Order) *OrderListResponse {
	resp := &OrderListResponse{
		ID:             order.ID,
		OrderNo:        order.OrderNo,
		UserID:         order.UserID,
		Type:           order.Type,
		OriginalAmount: order.OriginalAmount,
		DiscountAmount: order.DiscountAmount,
		ActualAmount:   order.ActualAmount,
		Status:         order.Status,
		PaidAt:         order.PaidAt,
		CompletedAt:    order.CompletedAt,
		CreatedAt:      order.CreatedAt,
	}

	if order.User != nil && order.User.Phone != nil {
		resp.UserPhone = *order.User.Phone
	}

	return resp
}

// ExportOrders 导出订单
func (s *OrderAdminService) ExportOrders(ctx context.Context, filters *OrderListFilters) ([]*models.Order, error) {
	query := s.db.WithContext(ctx).Model(&models.Order{})

	if filters != nil {
		if filters.OrderNo != "" {
			query = query.Where("order_no LIKE ?", "%"+filters.OrderNo+"%")
		}
		if filters.UserID > 0 {
			query = query.Where("user_id = ?", filters.UserID)
		}
		if filters.Type != "" {
			query = query.Where("type = ?", filters.Type)
		}
		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}
		if filters.StartDate != nil {
			query = query.Where("created_at >= ?", *filters.StartDate)
		}
		if filters.EndDate != nil {
			query = query.Where("created_at <= ?", *filters.EndDate)
		}
	}

	var orders []*models.Order
	if err := query.Preload("User").Preload("Items").
		Order("id DESC").
		Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}
