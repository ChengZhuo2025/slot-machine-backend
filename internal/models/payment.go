package models

import (
	"time"
)

// Payment 支付记录
type Payment struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	PaymentNo       string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"payment_no"`
	OrderID         int64      `gorm:"index;not null" json:"order_id"`
	OrderNo         string     `gorm:"type:varchar(64);not null" json:"order_no"`
	UserID          int64      `gorm:"index;not null" json:"user_id"`
	Amount          float64    `gorm:"type:decimal(12,2);not null" json:"amount"`
	PaymentMethod   string     `gorm:"type:varchar(20);not null" json:"payment_method"`
	PaymentChannel  string     `gorm:"type:varchar(20);not null" json:"payment_channel"`
	TransactionID   *string    `gorm:"type:varchar(64)" json:"transaction_id,omitempty"`
	Status          int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	ExpiredAt       *time.Time `json:"expired_at,omitempty"`
	CallbackData    JSON       `gorm:"type:jsonb" json:"callback_data,omitempty"`
	ErrorMessage    *string    `gorm:"type:varchar(255)" json:"error_message,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Order *Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	User  *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 表名
func (Payment) TableName() string {
	return "payments"
}

// PaymentMethod 支付方式
const (
	PaymentMethodWechat  = "wechat"  // 微信支付
	PaymentMethodAlipay  = "alipay"  // 支付宝
	PaymentMethodBalance = "balance" // 余额支付
)

// PaymentChannel 支付渠道
const (
	PaymentChannelMiniProgram = "miniprogram" // 小程序
	PaymentChannelH5          = "h5"          // H5
	PaymentChannelNative      = "native"      // 扫码
	PaymentChannelApp         = "app"         // APP
)

// PaymentStatus 支付状态
const (
	PaymentStatusPending  = 0 // 待支付
	PaymentStatusSuccess  = 1 // 支付成功
	PaymentStatusFailed   = 2 // 支付失败
	PaymentStatusClosed   = 3 // 已关闭
	PaymentStatusRefunded = 4 // 已退款
)

// Refund 退款记录
type Refund struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	RefundNo       string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"refund_no"`
	OrderID        int64      `gorm:"index;not null" json:"order_id"`
	OrderNo        string     `gorm:"type:varchar(64);not null" json:"order_no"`
	PaymentID      int64      `gorm:"not null" json:"payment_id"`
	PaymentNo      string     `gorm:"type:varchar(64);not null" json:"payment_no"`
	UserID         int64      `gorm:"index;not null" json:"user_id"`
	Amount         float64    `gorm:"type:decimal(12,2);not null" json:"amount"`
	Reason         string     `gorm:"type:varchar(255);not null" json:"reason"`
	Status         int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	TransactionID  *string    `gorm:"type:varchar(64)" json:"transaction_id,omitempty"`
	RefundedAt     *time.Time `json:"refunded_at,omitempty"`
	RejectedAt     *time.Time `json:"rejected_at,omitempty"`
	RejectReason   *string    `gorm:"type:varchar(255)" json:"reject_reason,omitempty"`
	CallbackData   JSON       `gorm:"type:jsonb" json:"callback_data,omitempty"`
	OperatorID     *int64     `json:"operator_id,omitempty"`
	OperatorType   *string    `gorm:"type:varchar(10)" json:"operator_type,omitempty"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Order   *Order   `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Payment *Payment `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 表名
func (Refund) TableName() string {
	return "refunds"
}

// RefundStatus 退款状态
const (
	RefundStatusPending   = 0 // 待处理
	RefundStatusApproved  = 1 // 已批准
	RefundStatusProcessing = 2 // 退款中
	RefundStatusSuccess   = 3 // 退款成功
	RefundStatusRejected  = 4 // 已拒绝
	RefundStatusFailed    = 5 // 退款失败
)

// RefundOperatorType 退款操作人类型
const (
	RefundOperatorUser   = "user"   // 用户
	RefundOperatorAdmin  = "admin"  // 管理员
	RefundOperatorSystem = "system" // 系统
)

// Settlement 结算记录
type Settlement struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	SettlementNo    string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"settlement_no"`
	MerchantID      int64      `gorm:"index;not null" json:"merchant_id"`
	PeriodStart     time.Time  `gorm:"not null" json:"period_start"`
	PeriodEnd       time.Time  `gorm:"not null" json:"period_end"`
	TotalAmount     float64    `gorm:"type:decimal(12,2);not null" json:"total_amount"`
	CommissionAmount float64   `gorm:"type:decimal(12,2);not null" json:"commission_amount"`
	SettlementAmount float64   `gorm:"type:decimal(12,2);not null" json:"settlement_amount"`
	OrderCount      int        `gorm:"not null;default:0" json:"order_count"`
	Status          int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	SettledAt       *time.Time `json:"settled_at,omitempty"`
	Remark          *string    `gorm:"type:varchar(255)" json:"remark,omitempty"`
	OperatorID      *int64     `json:"operator_id,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Merchant *Merchant `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
}

// TableName 表名
func (Settlement) TableName() string {
	return "settlements"
}

// SettlementStatus 结算状态
const (
	SettlementStatusPending   = 0 // 待结算
	SettlementStatusSettled   = 1 // 已结算
	SettlementStatusFailed    = 2 // 结算失败
)
