package models

import (
	"time"
)

// Distributor 分销员模型
type Distributor struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          int64      `gorm:"uniqueIndex;not null" json:"user_id"`
	Level           int        `gorm:"not null;default:1" json:"level"`
	ParentID        *int64     `gorm:"index" json:"parent_id,omitempty"`
	InviteCode      string     `gorm:"type:varchar(20);uniqueIndex;not null" json:"invite_code"`
	TotalCommission float64    `gorm:"type:decimal(12,2);not null;default:0" json:"total_commission"`
	PaidCommission  float64    `gorm:"type:decimal(12,2);not null;default:0" json:"paid_commission"`
	FrozenCommission float64   `gorm:"type:decimal(12,2);not null;default:0" json:"frozen_commission"`
	ChildCount      int        `gorm:"not null;default:0" json:"child_count"`
	OrderCount      int        `gorm:"not null;default:0" json:"order_count"`
	Status          int8       `gorm:"type:smallint;not null;default:1" json:"status"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	User     *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Parent   *Distributor `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Distributor `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

// TableName 表名
func (Distributor) TableName() string {
	return "distributors"
}

// DistributorStatus 分销员状态
const (
	DistributorStatusPending  = 0 // 待审核
	DistributorStatusActive   = 1 // 正常
	DistributorStatusDisabled = 2 // 禁用
)

// DistributorLevel 分销员等级
const (
	DistributorLevelPrimary   = 1 // 初级
	DistributorLevelSenior    = 2 // 高级
	DistributorLevelExpert    = 3 // 专家
)

// Commission 佣金记录
type Commission struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	DistributorID  int64     `gorm:"index;not null" json:"distributor_id"`
	OrderID        int64     `gorm:"index;not null" json:"order_id"`
	OrderNo        string    `gorm:"type:varchar(64);not null" json:"order_no"`
	SourceUserID   int64     `gorm:"not null" json:"source_user_id"`
	Level          int       `gorm:"not null" json:"level"`
	OrderAmount    float64   `gorm:"type:decimal(12,2);not null" json:"order_amount"`
	CommissionRate float64   `gorm:"type:decimal(5,4);not null" json:"commission_rate"`
	Amount         float64   `gorm:"type:decimal(12,2);not null" json:"amount"`
	Status         int8      `gorm:"type:smallint;not null;default:0" json:"status"`
	SettledAt      *time.Time `json:"settled_at,omitempty"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Distributor *Distributor `gorm:"foreignKey:DistributorID" json:"distributor,omitempty"`
	Order       *Order       `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	SourceUser  *User        `gorm:"foreignKey:SourceUserID" json:"source_user,omitempty"`
}

// TableName 表名
func (Commission) TableName() string {
	return "commissions"
}

// CommissionStatus 佣金状态
const (
	CommissionStatusPending  = 0 // 待结算
	CommissionStatusSettled  = 1 // 已结算
	CommissionStatusCancelled = 2 // 已取消
)

// CommissionLevel 佣金层级
const (
	CommissionLevelDirect = 1 // 直接推荐
	CommissionLevelSecond = 2 // 二级推荐
)

// Withdrawal 提现记录
type Withdrawal struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	WithdrawalNo    string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"withdrawal_no"`
	DistributorID   int64      `gorm:"index;not null" json:"distributor_id"`
	Amount          float64    `gorm:"type:decimal(12,2);not null" json:"amount"`
	Fee             float64    `gorm:"type:decimal(10,2);not null;default:0" json:"fee"`
	ActualAmount    float64    `gorm:"type:decimal(12,2);not null" json:"actual_amount"`
	WithdrawMethod  string     `gorm:"type:varchar(20);not null" json:"withdraw_method"`
	AccountName     *string    `gorm:"type:varchar(50)" json:"account_name,omitempty"`
	AccountNo       *string    `gorm:"type:varchar(64)" json:"account_no,omitempty"`
	BankName        *string    `gorm:"type:varchar(100)" json:"bank_name,omitempty"`
	Status          int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	TransactionID   *string    `gorm:"type:varchar(64)" json:"transaction_id,omitempty"`
	RejectReason    *string    `gorm:"type:varchar(255)" json:"reject_reason,omitempty"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	OperatorID      *int64     `json:"operator_id,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Distributor *Distributor `gorm:"foreignKey:DistributorID" json:"distributor,omitempty"`
	Operator    *Admin       `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
}

// TableName 表名
func (Withdrawal) TableName() string {
	return "withdrawals"
}

// WithdrawMethod 提现方式
const (
	WithdrawMethodWechat  = "wechat"  // 微信
	WithdrawMethodAlipay  = "alipay"  // 支付宝
	WithdrawMethodBank    = "bank"    // 银行卡
)

// WithdrawalStatus 提现状态
const (
	WithdrawalStatusPending   = 0 // 待处理
	WithdrawalStatusApproved  = 1 // 已批准
	WithdrawalStatusProcessing = 2 // 处理中
	WithdrawalStatusCompleted = 3 // 已完成
	WithdrawalStatusRejected  = 4 // 已拒绝
	WithdrawalStatusFailed    = 5 // 失败
)
