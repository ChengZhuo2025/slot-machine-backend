// Package order 订单服务
package order

import (
	"context"
	"log"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/service/distribution"
)

// OrderCompleteHook 订单完成钩子
// 用于在订单完成时触发佣金计算
type OrderCompleteHook struct {
	commissionService commissionService
}

type commissionService interface {
	Calculate(ctx context.Context, req *distribution.CalculateRequest) (*distribution.CalculateResponse, error)
	CancelByOrderID(ctx context.Context, orderID int64) error
}

// NewOrderCompleteHook 创建订单完成钩子
func NewOrderCompleteHook(commissionService commissionService) *OrderCompleteHook {
	return &OrderCompleteHook{
		commissionService: commissionService,
	}
}

// OnOrderCompleted 订单完成时触发
// 应该在订单状态变为"已完成"时调用此方法
func (h *OrderCompleteHook) OnOrderCompleted(ctx context.Context, order *models.Order) error {
	// 只处理已完成的订单
	if order.Status != models.OrderStatusCompleted {
		return nil
	}

	if h.commissionService == nil {
		return nil
	}

	// 计算佣金
	req := &distribution.CalculateRequest{
		OrderID:     order.ID,
		UserID:      order.UserID,
		OrderAmount: order.ActualAmount, // 使用实付金额计算佣金
	}

	result, err := h.commissionService.Calculate(ctx, req)
	if err != nil {
		// 佣金计算失败不应该影响订单完成
		// 记录错误日志，后续可以通过定时任务补偿
		log.Printf("佣金计算失败: orderID=%d, error=%v", order.ID, err)
		return nil
	}

	if result.TotalAmount > 0 {
		log.Printf("订单 %s 佣金计算完成: 直推佣金=%.2f, 间推佣金=%.2f",
			order.OrderNo,
			getCommissionAmount(result.DirectCommission),
			getCommissionAmount(result.IndirectCommission),
		)
	}

	return nil
}

// getCommissionAmount 获取佣金金额
func getCommissionAmount(commission *models.Commission) float64 {
	if commission == nil {
		return 0
	}
	return commission.Amount
}

// OnOrderRefunded 订单退款时触发
// 应该在订单状态变为"已退款"时调用此方法
func (h *OrderCompleteHook) OnOrderRefunded(ctx context.Context, order *models.Order) error {
	if h.commissionService == nil {
		return nil
	}

	// 取消该订单相关的佣金
	err := h.commissionService.CancelByOrderID(ctx, order.ID)
	if err != nil {
		log.Printf("取消佣金失败: orderID=%d, error=%v", order.ID, err)
		return nil // 不影响退款流程
	}

	log.Printf("订单 %s 相关佣金已取消", order.OrderNo)
	return nil
}

// OrderEventHandler 订单事件处理器接口
type OrderEventHandler interface {
	OnOrderCompleted(ctx context.Context, order *models.Order) error
	OnOrderRefunded(ctx context.Context, order *models.Order) error
}

// Ensure OrderCompleteHook implements OrderEventHandler
var _ OrderEventHandler = (*OrderCompleteHook)(nil)
