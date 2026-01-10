// Package payment 提供支付服务
package payment

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/utils"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	"github.com/dumeirei/smart-locker-backend/pkg/wechatpay"
)

// PaymentService 支付服务
type PaymentService struct {
	db          *gorm.DB
	paymentRepo *repository.PaymentRepository
	refundRepo  *repository.RefundRepository
	rentalRepo  *repository.RentalRepository
	wechatPay   *wechatpay.Client
}

// NewPaymentService 创建支付服务
func NewPaymentService(
	db *gorm.DB,
	paymentRepo *repository.PaymentRepository,
	refundRepo *repository.RefundRepository,
	rentalRepo *repository.RentalRepository,
	wechatPay *wechatpay.Client,
) *PaymentService {
	return &PaymentService{
		db:          db,
		paymentRepo: paymentRepo,
		refundRepo:  refundRepo,
		rentalRepo:  rentalRepo,
		wechatPay:   wechatPay,
	}
}

// CreatePaymentRequest 创建支付请求
type CreatePaymentRequest struct {
	OrderID        int64   `json:"order_id" binding:"required"`
	OrderNo        string  `json:"order_no" binding:"required"`
	OrderType      string  `json:"order_type" binding:"required"` // rental, order
	Amount         float64 `json:"amount" binding:"required"`
	PaymentMethod  string  `json:"payment_method" binding:"required"`
	PaymentChannel string  `json:"payment_channel" binding:"required"`
	OpenID         string  `json:"openid,omitempty"`
	Description    string  `json:"description,omitempty"`
}

// CreatePaymentResponse 创建支付响应
type CreatePaymentResponse struct {
	PaymentNo string                          `json:"payment_no"`
	PayParams *wechatpay.UnifiedOrderResponse `json:"pay_params,omitempty"`
	ExpiredAt time.Time                       `json:"expired_at"`
}

// CreatePayment 创建支付
func (s *PaymentService) CreatePayment(ctx context.Context, userID int64, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	paymentNo := utils.GenerateOrderNo("P")
	expiredAt := time.Now().Add(30 * time.Minute)

	payment := &models.Payment{
		PaymentNo:      paymentNo,
		OrderID:        req.OrderID,
		OrderNo:        req.OrderNo,
		UserID:         userID,
		Amount:         req.Amount,
		PaymentMethod:  req.PaymentMethod,
		PaymentChannel: req.PaymentChannel,
		Status:         models.PaymentStatusPending,
		ExpiredAt:      &expiredAt,
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	resp := &CreatePaymentResponse{
		PaymentNo: paymentNo,
		ExpiredAt: expiredAt,
	}

	// 调用微信支付创建订单
	if req.PaymentMethod == models.PaymentMethodWechat && s.wechatPay != nil {
		amount := int64(req.Amount * 100) // 转换为分
		description := req.Description
		if description == "" {
			description = fmt.Sprintf("订单支付-%s", req.OrderNo)
		}

		wechatReq := &wechatpay.UnifiedOrderRequest{
			OutTradeNo:  paymentNo,
			Description: description,
			Amount:      amount,
			OpenID:      req.OpenID,
		}

		var payParams *wechatpay.UnifiedOrderResponse
		var err error

		switch req.PaymentChannel {
		case models.PaymentChannelMiniProgram:
			payParams, err = s.wechatPay.CreateOrder(ctx, wechatReq)
		case models.PaymentChannelNative:
			payParams, err = s.wechatPay.CreateNativeOrder(ctx, wechatReq)
		case models.PaymentChannelH5:
			payParams, err = s.wechatPay.CreateH5Order(ctx, wechatReq)
		default:
			payParams, err = s.wechatPay.CreateOrder(ctx, wechatReq)
		}

		if err != nil {
			return nil, errors.ErrPaymentFailed.WithError(err)
		}

		resp.PayParams = payParams
	}

	return resp, nil
}

// HandlePaymentCallback 处理支付回调
func (s *PaymentService) HandlePaymentCallback(ctx context.Context, payload []byte) error {
	if s.wechatPay == nil {
		return errors.ErrPaymentCallbackError.WithMessage("微信支付客户端未初始化")
	}

	resource, err := s.wechatPay.ParseNotify(payload)
	if err != nil {
		return errors.ErrPaymentCallbackError.WithError(err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取支付记录（在事务内使用 tx，确保一致性）
		var payment models.Payment
		if err := tx.WithContext(ctx).Where("payment_no = ?", resource.OutTradeNo).First(&payment).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.ErrPaymentNotFound
			}
			return errors.ErrDatabaseError.WithError(err)
		}

		// 检查是否已处理
		if payment.Status != models.PaymentStatusPending {
			return nil
		}

		// 验证金额
		callbackAmount := float64(resource.Amount.Total) / 100
		if callbackAmount != payment.Amount {
			return errors.ErrPaymentCallbackError.WithMessage("金额不匹配")
		}

		// 更新支付状态
		now := time.Now()
		transactionID := resource.TransactionID
		if resource.TradeState == wechatpay.TradeStateSuccess {
			payment.Status = models.PaymentStatusSuccess
			payment.TransactionID = &transactionID
			payment.PaidAt = &now
		} else {
			payment.Status = models.PaymentStatusFailed
			errMsg := resource.TradeState
			payment.ErrorMessage = &errMsg
		}

		if err := tx.Save(&payment).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 如果支付成功，更新订单状态
		if payment.Status == models.PaymentStatusSuccess {
			// 更新租借订单状态
			if err := tx.Model(&models.Rental{}).
				Where("order_id = ?", payment.OrderID).
				Update("status", models.RentalStatusPaid).Error; err != nil {
				return errors.ErrDatabaseError.WithError(err)
			}
		}

		return nil
	})
}

// QueryPayment 查询支付状态
func (s *PaymentService) QueryPayment(ctx context.Context, paymentNo string) (*PaymentInfo, error) {
	payment, err := s.paymentRepo.GetByPaymentNo(ctx, paymentNo)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrPaymentNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toPaymentInfo(payment), nil
}

// PaymentInfo 支付信息
type PaymentInfo struct {
	ID             int64      `json:"id"`
	PaymentNo      string     `json:"payment_no"`
	OrderNo        string     `json:"order_no"`
	Amount         float64    `json:"amount"`
	PaymentMethod  string     `json:"payment_method"`
	PaymentChannel string     `json:"payment_channel"`
	Status         int8       `json:"status"`
	StatusName     string     `json:"status_name"`
	TransactionID  *string    `json:"transaction_id,omitempty"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	ExpiredAt      *time.Time `json:"expired_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// toPaymentInfo 转换为支付信息
func (s *PaymentService) toPaymentInfo(payment *models.Payment) *PaymentInfo {
	return &PaymentInfo{
		ID:             payment.ID,
		PaymentNo:      payment.PaymentNo,
		OrderNo:        payment.OrderNo,
		Amount:         payment.Amount,
		PaymentMethod:  payment.PaymentMethod,
		PaymentChannel: payment.PaymentChannel,
		Status:         payment.Status,
		StatusName:     s.getStatusName(payment.Status),
		TransactionID:  payment.TransactionID,
		PaidAt:         payment.PaidAt,
		ExpiredAt:      payment.ExpiredAt,
		CreatedAt:      payment.CreatedAt,
	}
}

// getStatusName 获取状态名称
func (s *PaymentService) getStatusName(status int8) string {
	switch status {
	case models.PaymentStatusPending:
		return "待支付"
	case models.PaymentStatusSuccess:
		return "支付成功"
	case models.PaymentStatusFailed:
		return "支付失败"
	case models.PaymentStatusClosed:
		return "已关闭"
	case models.PaymentStatusRefunded:
		return "已退款"
	default:
		return "未知"
	}
}

// CreateRefundRequest 创建退款请求
type CreateRefundRequest struct {
	PaymentNo string  `json:"payment_no" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
	Reason    string  `json:"reason" binding:"required"`
}

// CreateRefund 创建退款
func (s *PaymentService) CreateRefund(ctx context.Context, userID int64, req *CreateRefundRequest) error {
	payment, err := s.paymentRepo.GetByPaymentNo(ctx, req.PaymentNo)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrPaymentNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if payment.UserID != userID {
		return errors.ErrPermissionDenied
	}

	if payment.Status != models.PaymentStatusSuccess {
		return errors.ErrPaymentFailed.WithMessage("只有已支付的订单可以退款")
	}

	// 检查退款金额
	totalRefunded, err := s.refundRepo.GetTotalRefunded(ctx, payment.ID)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	if totalRefunded+req.Amount > payment.Amount {
		return errors.ErrRefundAmountExceed
	}

	// 创建退款记录
	refundNo := utils.GenerateOrderNo("RF")
	refund := &models.Refund{
		RefundNo:  refundNo,
		OrderID:   payment.OrderID,
		OrderNo:   payment.OrderNo,
		PaymentID: payment.ID,
		PaymentNo: payment.PaymentNo,
		UserID:    userID,
		Amount:    req.Amount,
		Reason:    req.Reason,
		Status:    models.RefundStatusPending,
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(refund).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 调用微信退款
		if s.wechatPay != nil && payment.PaymentMethod == models.PaymentMethodWechat {
			wechatReq := &wechatpay.RefundRequest{
				OutTradeNo:  payment.PaymentNo,
				OutRefundNo: refundNo,
				Reason:      req.Reason,
				Total:       int64(payment.Amount * 100),
				Refund:      int64(req.Amount * 100),
			}

			resp, err := s.wechatPay.Refund(ctx, wechatReq)
			if err != nil {
				return errors.ErrRefundFailed.WithError(err)
			}

			transactionID := resp.RefundID
			refund.TransactionID = &transactionID
			refund.Status = models.RefundStatusProcessing

			if err := tx.Save(refund).Error; err != nil {
				return errors.ErrDatabaseError.WithError(err)
			}
		}

		return nil
	})
}

// CloseExpiredPayments 关闭过期支付
func (s *PaymentService) CloseExpiredPayments(ctx context.Context) error {
	payments, err := s.paymentRepo.GetPendingExpired(ctx, time.Now(), 100)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	for _, payment := range payments {
		_ = s.paymentRepo.UpdateFields(ctx, payment.ID, map[string]interface{}{
			"status": models.PaymentStatusClosed,
		})
	}

	return nil
}
