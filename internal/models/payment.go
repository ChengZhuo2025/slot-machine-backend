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
	PaidAt          *time.Time `gorm:"column:pay_time" json:"paid_at,omitempty"`
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
// 参考: migrations/000010_create_finance.up.sql
type Settlement struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	SettlementNo string     `gorm:"column:settlement_no;type:varchar(64);uniqueIndex;not null" json:"settlement_no"`
	Type         string     `gorm:"column:type;type:varchar(20);not null" json:"type"`
	TargetID     int64      `gorm:"column:target_id;index;not null" json:"target_id"`
	PeriodStart  time.Time  `gorm:"column:period_start;type:date;not null" json:"period_start"`
	PeriodEnd    time.Time  `gorm:"column:period_end;type:date;not null" json:"period_end"`
	TotalAmount  float64    `gorm:"column:total_amount;type:decimal(12,2);not null" json:"total_amount"`
	Fee          float64    `gorm:"column:fee;type:decimal(10,2);not null;default:0" json:"fee"`
	ActualAmount float64    `gorm:"column:actual_amount;type:decimal(12,2);not null" json:"actual_amount"`
	OrderCount   int        `gorm:"column:order_count;not null" json:"order_count"`
	Status       string     `gorm:"column:status;type:varchar(20);not null" json:"status"`
	OperatorID   *int64     `gorm:"column:operator_id" json:"operator_id,omitempty"`
	SettledAt    *time.Time `gorm:"column:settled_at" json:"settled_at,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// 关联
	Operator *Admin `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
}

// TableName 表名
func (Settlement) TableName() string {
	return "settlements"
}

// SettlementStatus 结算状态
const (
	SettlementStatusPending    = "pending"    // 待结算
	SettlementStatusProcessing = "processing" // 结算中
	SettlementStatusCompleted  = "completed"  // 已完成
	SettlementStatusFailed     = "failed"     // 结算失败
)
