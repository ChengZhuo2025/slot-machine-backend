// Package order 订单服务
package order

import (
	"context"
	"log"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// PointsHook 积分钩子
// 用于在订单完成时给用户添加积分，退款时扣减积分
type PointsHook struct {
	db          *gorm.DB
	pointsAdder pointsAdder
}

// pointsAdder 积分增加接口
type pointsAdder interface {
	AddConsumePointsTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error
	RefundPointsTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error
}

// NewPointsHook 创建积分钩子
func NewPointsHook(db *gorm.DB, pointsAdder pointsAdder) *PointsHook {
	return &PointsHook{
		db:          db,
		pointsAdder: pointsAdder,
	}
}

// OnOrderCompleted 订单完成时触发，给用户添加积分
func (h *PointsHook) OnOrderCompleted(ctx context.Context, order *models.Order) error {
	// 只处理已完成的订单
	if order.Status != models.OrderStatusCompleted {
		return nil
	}

	if h.pointsAdder == nil {
		return nil
	}

	// 会员套餐购买订单不再额外计算积分（套餐本身包含赠送积分）
	if order.Type == "member_package" {
		return nil
	}

	// 使用实付金额计算积分
	amount := order.ActualAmount
	if amount <= 0 {
		return nil
	}

	// 在事务中添加积分
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return h.pointsAdder.AddConsumePointsTx(ctx, tx, order.UserID, amount, order.OrderNo)
	})

	if err != nil {
		// 积分添加失败不应该影响订单完成
		// 记录错误日志，后续可以通过定时任务补偿
		log.Printf("订单完成添加积分失败: orderID=%d, orderNo=%s, error=%v", order.ID, order.OrderNo, err)
		return nil
	}

	log.Printf("订单 %s 完成，用户 %d 获得 %d 积分", order.OrderNo, order.UserID, int(amount))
	return nil
}

// OnOrderRefunded 订单退款时触发，扣减用户积分
func (h *PointsHook) OnOrderRefunded(ctx context.Context, order *models.Order) error {
	if h.pointsAdder == nil {
		return nil
	}

	// 会员套餐不处理积分退还
	if order.Type == "member_package" {
		return nil
	}

	// 使用实付金额计算需扣减的积分
	amount := order.ActualAmount
	if amount <= 0 {
		return nil
	}

	// 在事务中扣减积分
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return h.pointsAdder.RefundPointsTx(ctx, tx, order.UserID, amount, order.OrderNo)
	})

	if err != nil {
		// 积分扣减失败记录日志
		log.Printf("订单退款扣减积分失败: orderID=%d, orderNo=%s, error=%v", order.ID, order.OrderNo, err)
		return nil
	}

	log.Printf("订单 %s 退款，用户 %d 扣减 %d 积分", order.OrderNo, order.UserID, int(amount))
	return nil
}

// Ensure PointsHook implements OrderEventHandler
var _ OrderEventHandler = (*PointsHook)(nil)

// CompositeOrderEventHandler 组合订单事件处理器
// 用于组合多个事件处理器，按顺序执行
type CompositeOrderEventHandler struct {
	handlers []OrderEventHandler
}

// NewCompositeOrderEventHandler 创建组合订单事件处理器
func NewCompositeOrderEventHandler(handlers ...OrderEventHandler) *CompositeOrderEventHandler {
	return &CompositeOrderEventHandler{
		handlers: handlers,
	}
}

// OnOrderCompleted 订单完成时触发
func (h *CompositeOrderEventHandler) OnOrderCompleted(ctx context.Context, order *models.Order) error {
	for _, handler := range h.handlers {
		if handler == nil {
			continue
		}
		if err := handler.OnOrderCompleted(ctx, order); err != nil {
			// 记录错误但继续执行其他处理器
			log.Printf("订单完成处理器执行失败: %v", err)
		}
	}
	return nil
}

// OnOrderRefunded 订单退款时触发
func (h *CompositeOrderEventHandler) OnOrderRefunded(ctx context.Context, order *models.Order) error {
	for _, handler := range h.handlers {
		if handler == nil {
			continue
		}
		if err := handler.OnOrderRefunded(ctx, order); err != nil {
			// 记录错误但继续执行其他处理器
			log.Printf("订单退款处理器执行失败: %v", err)
		}
	}
	return nil
}

// AddHandler 添加处理器
func (h *CompositeOrderEventHandler) AddHandler(handler OrderEventHandler) {
	if handler != nil {
		h.handlers = append(h.handlers, handler)
	}
}

// Ensure CompositeOrderEventHandler implements OrderEventHandler
var _ OrderEventHandler = (*CompositeOrderEventHandler)(nil)
