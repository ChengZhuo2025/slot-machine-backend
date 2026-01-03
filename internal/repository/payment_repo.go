// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// PaymentRepository 支付仓储
type PaymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository 创建支付仓储
func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create 创建支付记录
func (r *PaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

// GetByID 根据 ID 获取支付记录
func (r *PaymentRepository) GetByID(ctx context.Context, id int64) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).First(&payment, id).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetByPaymentNo 根据支付单号获取
func (r *PaymentRepository) GetByPaymentNo(ctx context.Context, paymentNo string) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("payment_no = ?", paymentNo).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetByTransactionID 根据第三方交易号获取
func (r *PaymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("transaction_id = ?", transactionID).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetByOrder 获取订单的支付记录
func (r *PaymentRepository) GetByOrder(ctx context.Context, orderID int64) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Where("status = ?", models.PaymentStatusSuccess).
		First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// Update 更新支付记录
func (r *PaymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}

// UpdateFields 更新指定字段
func (r *PaymentRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Payment{}).Where("id = ?", id).Updates(fields).Error
}

// List 获取支付记录列表
func (r *PaymentRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Payment, int64, error) {
	var payments []*models.Payment
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Payment{})

	if userID, ok := filters["user_id"].(int64); ok && userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if orderID, ok := filters["order_id"].(int64); ok && orderID > 0 {
		query = query.Where("order_id = ?", orderID)
	}
	if method, ok := filters["payment_method"].(string); ok && method != "" {
		query = query.Where("payment_method = ?", method)
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
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

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&payments).Error; err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// GetForUpdate 获取支付记录（加锁）
func (r *PaymentRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, id int64) (*models.Payment, error) {
	var payment models.Payment
	err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").First(&payment, id).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetPendingExpired 获取过期的待支付记录
func (r *PaymentRepository) GetPendingExpired(ctx context.Context, expiredBefore time.Time, limit int) ([]*models.Payment, error) {
	var payments []*models.Payment
	err := r.db.WithContext(ctx).
		Where("status = ?", models.PaymentStatusPending).
		Where("expired_at < ?", expiredBefore).
		Limit(limit).
		Find(&payments).Error
	return payments, err
}

// RefundRepository 退款仓储
type RefundRepository struct {
	db *gorm.DB
}

// NewRefundRepository 创建退款仓储
func NewRefundRepository(db *gorm.DB) *RefundRepository {
	return &RefundRepository{db: db}
}

// Create 创建退款记录
func (r *RefundRepository) Create(ctx context.Context, refund *models.Refund) error {
	return r.db.WithContext(ctx).Create(refund).Error
}

// GetByID 根据 ID 获取退款记录
func (r *RefundRepository) GetByID(ctx context.Context, id int64) (*models.Refund, error) {
	var refund models.Refund
	err := r.db.WithContext(ctx).First(&refund, id).Error
	if err != nil {
		return nil, err
	}
	return &refund, nil
}

// GetByRefundNo 根据退款单号获取
func (r *RefundRepository) GetByRefundNo(ctx context.Context, refundNo string) (*models.Refund, error) {
	var refund models.Refund
	err := r.db.WithContext(ctx).Where("refund_no = ?", refundNo).First(&refund).Error
	if err != nil {
		return nil, err
	}
	return &refund, nil
}

// Update 更新退款记录
func (r *RefundRepository) Update(ctx context.Context, refund *models.Refund) error {
	return r.db.WithContext(ctx).Save(refund).Error
}

// UpdateFields 更新指定字段
func (r *RefundRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Refund{}).Where("id = ?", id).Updates(fields).Error
}

// ListByPayment 获取支付记录的退款列表
func (r *RefundRepository) ListByPayment(ctx context.Context, paymentID int64) ([]*models.Refund, error) {
	var refunds []*models.Refund
	err := r.db.WithContext(ctx).
		Where("payment_id = ?", paymentID).
		Order("id DESC").
		Find(&refunds).Error
	return refunds, err
}

// GetTotalRefunded 获取支付记录已退款总额
func (r *RefundRepository) GetTotalRefunded(ctx context.Context, paymentID int64) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).Model(&models.Refund{}).
		Where("payment_id = ?", paymentID).
		// 退款申请一旦创建就需要占用可退额度，避免并发/重复提交导致超额退款。
		Where("status IN ?", []int8{
			models.RefundStatusPending,
			models.RefundStatusApproved,
			models.RefundStatusProcessing,
			models.RefundStatusSuccess,
		}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}
