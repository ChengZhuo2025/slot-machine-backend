// Package order 提供订单相关服务
package order

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/utils"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// RefundService 退款服务
type RefundService struct {
	db          *gorm.DB
	refundRepo  *repository.RefundRepository
	orderRepo   *repository.OrderRepository
	paymentRepo *repository.PaymentRepository
}

// NewRefundService 创建退款服务
func NewRefundService(
	db *gorm.DB,
	refundRepo *repository.RefundRepository,
	orderRepo *repository.OrderRepository,
	paymentRepo *repository.PaymentRepository,
) *RefundService {
	return &RefundService{
		db:          db,
		refundRepo:  refundRepo,
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
	}
}

// RefundInfo 退款信息
type RefundInfo struct {
	ID            int64   `json:"id"`
	RefundNo      string  `json:"refund_no"`
	OrderID       int64   `json:"order_id"`
	OrderNo       string  `json:"order_no"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
	Status        int8    `json:"status"`
	StatusName    string  `json:"status_name"`
	TransactionID string  `json:"transaction_id,omitempty"`
	RefundedAt    string  `json:"refunded_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// RefundListResponse 退款列表响应
type RefundListResponse struct {
	List       []*RefundInfo `json:"list"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// CreateRefundRequest 创建退款请求
type CreateRefundRequest struct {
	OrderID int64   `json:"order_id" binding:"required"`
	Amount  float64 `json:"amount" binding:"required,gt=0"`
	Reason  string  `json:"reason" binding:"required"`
}

// CreateRefund 创建退款申请
func (s *RefundService) CreateRefund(ctx context.Context, userID int64, req *CreateRefundRequest) (*RefundInfo, error) {
	// 检查订单是否存在且属于该用户
	order, err := s.orderRepo.GetByID(ctx, req.OrderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrOrderNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if order.UserID != userID {
		return nil, errors.ErrOrderNotFound
	}

	// 检查订单状态是否允许退款
	if order.Status != models.OrderStatusPaid &&
		order.Status != models.OrderStatusPendingShip &&
		order.Status != models.OrderStatusShipped {
		return nil, errors.ErrOrderStatusError.WithMessage("订单状态不允许申请退款")
	}

	// 检查是否已存在待处理的退款
	exists, err := s.refundRepo.ExistsPendingByOrderID(ctx, req.OrderID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithMessage("已存在待处理的退款申请")
	}

	// 检查退款金额
	if req.Amount > order.ActualAmount {
		return nil, errors.ErrRefundAmountExceed
	}

	// 获取支付记录
	payment, err := s.paymentRepo.GetByOrder(ctx, req.OrderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrPaymentNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 创建退款记录
	operatorType := models.RefundOperatorUser
	refund := &models.Refund{
		RefundNo:     utils.GenerateOrderNo("R"),
		OrderID:      req.OrderID,
		OrderNo:      order.OrderNo,
		PaymentID:    payment.ID,
		PaymentNo:    payment.PaymentNo,
		UserID:       userID,
		Amount:       req.Amount,
		Reason:       req.Reason,
		Status:       models.RefundStatusPending,
		OperatorID:   &userID,
		OperatorType: &operatorType,
	}

	if err := s.refundRepo.Create(ctx, refund); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 更新订单状态为退款中
	if err := s.orderRepo.UpdateFields(ctx, req.OrderID, map[string]interface{}{
		"status": models.OrderStatusRefunding,
	}); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toRefundInfo(refund), nil
}

// GetRefundDetail 获取退款详情
func (s *RefundService) GetRefundDetail(ctx context.Context, userID int64, refundID int64) (*RefundInfo, error) {
	refund, err := s.refundRepo.GetByIDWithRelations(ctx, refundID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrRefundNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if refund.UserID != userID {
		return nil, errors.ErrRefundNotFound
	}

	return s.toRefundInfo(refund), nil
}

// GetUserRefunds 获取用户退款列表
func (s *RefundService) GetUserRefunds(ctx context.Context, userID int64, page, pageSize int) (*RefundListResponse, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	refunds, total, err := s.refundRepo.ListByUserID(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*RefundInfo, len(refunds))
	for i, r := range refunds {
		list[i] = s.toRefundInfo(r)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &RefundListResponse{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// CancelRefund 取消退款申请
func (s *RefundService) CancelRefund(ctx context.Context, userID int64, refundID int64) error {
	refund, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrRefundNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if refund.UserID != userID {
		return errors.ErrRefundNotFound
	}

	if refund.Status != models.RefundStatusPending {
		return errors.ErrOperationFailed.WithMessage("退款申请状态不允许取消")
	}

	now := time.Now()
	// 更新退款状态
	if err := s.refundRepo.UpdateFields(ctx, refundID, map[string]interface{}{
		"status":      models.RefundStatusRejected,
		"rejected_at": now,
	}); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	// 恢复订单状态
	order, err := s.orderRepo.GetByID(ctx, refund.OrderID)
	if err == nil && order.Status == models.OrderStatusRefunding {
		// 根据之前的状态恢复
		newStatus := models.OrderStatusPaid
		if order.ShippedAt != nil {
			newStatus = models.OrderStatusShipped
		}
		_ = s.orderRepo.UpdateFields(ctx, refund.OrderID, map[string]interface{}{
			"status": newStatus,
		})
	}

	return nil
}

// ApproveRefund 批准退款（管理端）
func (s *RefundService) ApproveRefund(ctx context.Context, operatorID int64, refundID int64) error {
	refund, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrRefundNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if refund.Status != models.RefundStatusPending {
		return errors.ErrOperationFailed.WithMessage("退款申请状态不允许审批")
	}

	operatorType := models.RefundOperatorAdmin
	return s.refundRepo.UpdateFields(ctx, refundID, map[string]interface{}{
		"status":        models.RefundStatusApproved,
		"operator_id":   operatorID,
		"operator_type": operatorType,
	})
}

// RejectRefund 拒绝退款（管理端）
func (s *RefundService) RejectRefund(ctx context.Context, operatorID int64, refundID int64, reason string) error {
	refund, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrRefundNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if refund.Status != models.RefundStatusPending {
		return errors.ErrOperationFailed.WithMessage("退款申请状态不允许拒绝")
	}

	now := time.Now()
	operatorType := models.RefundOperatorAdmin
	if err := s.refundRepo.UpdateFields(ctx, refundID, map[string]interface{}{
		"status":        models.RefundStatusRejected,
		"operator_id":   operatorID,
		"operator_type": operatorType,
		"rejected_at":   now,
		"reject_reason": reason,
	}); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	// 恢复订单状态
	order, err := s.orderRepo.GetByID(ctx, refund.OrderID)
	if err == nil && order.Status == models.OrderStatusRefunding {
		newStatus := models.OrderStatusPaid
		if order.ShippedAt != nil {
			newStatus = models.OrderStatusShipped
		}
		_ = s.orderRepo.UpdateFields(ctx, refund.OrderID, map[string]interface{}{
			"status": newStatus,
		})
	}

	return nil
}

// toRefundInfo 转换为退款信息
func (s *RefundService) toRefundInfo(r *models.Refund) *RefundInfo {
	info := &RefundInfo{
		ID:         r.ID,
		RefundNo:   r.RefundNo,
		OrderID:    r.OrderID,
		OrderNo:    r.OrderNo,
		Amount:     r.Amount,
		Reason:     r.Reason,
		Status:     r.Status,
		StatusName: s.getStatusName(r.Status),
		CreatedAt:  r.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if r.TransactionID != nil {
		info.TransactionID = *r.TransactionID
	}
	if r.RefundedAt != nil {
		info.RefundedAt = r.RefundedAt.Format("2006-01-02 15:04:05")
	}

	return info
}

// getStatusName 获取状态名称
func (s *RefundService) getStatusName(status int8) string {
	switch status {
	case models.RefundStatusPending:
		return "待处理"
	case models.RefundStatusApproved:
		return "已批准"
	case models.RefundStatusRejected:
		return "已拒绝"
	case models.RefundStatusProcessing:
		return "处理中"
	case models.RefundStatusSuccess:
		return "退款成功"
	case models.RefundStatusFailed:
		return "退款失败"
	default:
		return fmt.Sprintf("未知状态(%d)", status)
	}
}
