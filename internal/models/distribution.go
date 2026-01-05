package models

import (
	"time"
)

// Distributor 分销商模型
// 对应数据库表: distributors
type Distributor struct {
	ID                  int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID              int64      `gorm:"column:user_id;uniqueIndex;not null" json:"user_id"`
	ParentID            *int64     `gorm:"column:parent_id;index" json:"parent_id,omitempty"`
	Level               int        `gorm:"column:level;not null;default:1" json:"level"` // 层级: 1直推 2间推
	InviteCode          string     `gorm:"column:invite_code;type:varchar(20);uniqueIndex;not null" json:"invite_code"`
	TotalCommission     float64    `gorm:"column:total_commission;type:decimal(12,2);not null;default:0" json:"total_commission"`
	AvailableCommission float64    `gorm:"column:available_commission;type:decimal(12,2);not null;default:0" json:"available_commission"`
	FrozenCommission    float64    `gorm:"column:frozen_commission;type:decimal(12,2);not null;default:0" json:"frozen_commission"`
	WithdrawnCommission float64    `gorm:"column:withdrawn_commission;type:decimal(12,2);not null;default:0" json:"withdrawn_commission"`
	TeamCount           int        `gorm:"column:team_count;not null;default:0" json:"team_count"`
	DirectCount         int        `gorm:"column:direct_count;not null;default:0" json:"direct_count"`
	Status              int        `gorm:"column:status;type:smallint;not null;default:0" json:"status"` // 0待审核 1已通过 2已拒绝
	ApprovedAt          *time.Time `gorm:"column:approved_at" json:"approved_at,omitempty"`
	ApprovedBy          *int64     `gorm:"column:approved_by" json:"approved_by,omitempty"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	User           *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Parent         *Distributor  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children       []Distributor `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	ApprovedByUser *Admin        `gorm:"foreignKey:ApprovedBy" json:"approved_by_admin,omitempty"`
}

// TableName 表名
func (Distributor) TableName() string {
	return "distributors"
}

// DistributorStatus 分销商状态
const (
	DistributorStatusPending  = 0 // 待审核
	DistributorStatusApproved = 1 // 已通过
	DistributorStatusRejected = 2 // 已拒绝
)

// DistributorLevel 分销层级
const (
	DistributorLevelDirect   = 1 // 直推
	DistributorLevelIndirect = 2 // 间推
)

// Commission 佣金记录
// 对应数据库表: commissions
type Commission struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DistributorID int64      `gorm:"column:distributor_id;index;not null" json:"distributor_id"`
	OrderID       int64      `gorm:"column:order_id;index;not null" json:"order_id"`
	FromUserID    int64      `gorm:"column:from_user_id;not null" json:"from_user_id"` // 消费用户ID
	Type          string     `gorm:"column:type;type:varchar(20);not null" json:"type"` // direct/indirect
	OrderAmount   float64    `gorm:"column:order_amount;type:decimal(12,2);not null" json:"order_amount"`
	Rate          float64    `gorm:"column:rate;type:decimal(5,4);not null" json:"rate"`
	Amount        float64    `gorm:"column:amount;type:decimal(12,2);not null" json:"amount"`
	Status        int        `gorm:"column:status;type:smallint;not null;default:0" json:"status"` // 0待结算 1已结算 2已失效
	SettledAt     *time.Time `gorm:"column:settled_at" json:"settled_at,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// 关联
	Distributor *Distributor `gorm:"foreignKey:DistributorID" json:"distributor,omitempty"`
	Order       *Order       `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	FromUser    *User        `gorm:"foreignKey:FromUserID" json:"from_user,omitempty"`
}

// TableName 表名
func (Commission) TableName() string {
	return "commissions"
}

// CommissionType 佣金类型
const (
	CommissionTypeDirect   = "direct"   // 直接推荐
	CommissionTypeIndirect = "indirect" // 间接推荐
)

// CommissionStatus 佣金状态
const (
	CommissionStatusPending   = 0 // 待结算
	CommissionStatusSettled   = 1 // 已结算
	CommissionStatusCancelled = 2 // 已失效
)

// Withdrawal 提现申请
// 对应数据库表: withdrawals
type Withdrawal struct {
	ID                   int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WithdrawalNo         string     `gorm:"column:withdrawal_no;type:varchar(64);uniqueIndex;not null" json:"withdrawal_no"`
	UserID               int64      `gorm:"column:user_id;index;not null" json:"user_id"`
	Type                 string     `gorm:"column:type;type:varchar(20);not null" json:"type"` // wallet/commission
	Amount               float64    `gorm:"column:amount;type:decimal(12,2);not null" json:"amount"`
	Fee                  float64    `gorm:"column:fee;type:decimal(10,2);not null;default:0" json:"fee"`
	ActualAmount         float64    `gorm:"column:actual_amount;type:decimal(12,2);not null" json:"actual_amount"`
	WithdrawTo           string     `gorm:"column:withdraw_to;type:varchar(20);not null" json:"withdraw_to"` // wechat/alipay/bank
	AccountInfoEncrypted string     `gorm:"column:account_info_encrypted;type:text;not null" json:"-"`
	Status               string     `gorm:"column:status;type:varchar(20);not null" json:"status"` // pending/approved/processing/success/rejected
	OperatorID           *int64     `gorm:"column:operator_id" json:"operator_id,omitempty"`
	ProcessedAt          *time.Time `gorm:"column:processed_at" json:"processed_at,omitempty"`
	RejectReason         *string    `gorm:"column:reject_reason;type:varchar(255)" json:"reject_reason,omitempty"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	User     *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Operator *Admin `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
}

// TableName 表名
func (Withdrawal) TableName() string {
	return "withdrawals"
}

// WithdrawalType 提现类型
const (
	WithdrawalTypeWallet     = "wallet"     // 钱包余额提现
	WithdrawalTypeCommission = "commission" // 佣金提现
)

// WithdrawTo 提现方式
const (
	WithdrawToWechat = "wechat" // 微信
	WithdrawToAlipay = "alipay" // 支付宝
	WithdrawToBank   = "bank"   // 银行卡
)

// WithdrawalStatus 提现状态
const (
	WithdrawalStatusPending    = "pending"    // 待审核
	WithdrawalStatusApproved   = "approved"   // 已通过
	WithdrawalStatusProcessing = "processing" // 打款中
	WithdrawalStatusSuccess    = "success"    // 已完成
	WithdrawalStatusRejected   = "rejected"   // 已拒绝
)

// CommissionSetting 佣金设置
type CommissionSetting struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DirectRate    float64   `gorm:"column:direct_rate;type:decimal(5,4);not null" json:"direct_rate"`     // 直推佣金比例
	IndirectRate  float64   `gorm:"column:indirect_rate;type:decimal(5,4);not null" json:"indirect_rate"` // 间推佣金比例
	MinWithdraw   float64   `gorm:"column:min_withdraw;type:decimal(10,2);not null" json:"min_withdraw"`  // 最低提现金额
	WithdrawFee   float64   `gorm:"column:withdraw_fee;type:decimal(5,4);not null" json:"withdraw_fee"`   // 提现手续费比例
	SettleDelay   int       `gorm:"column:settle_delay;not null" json:"settle_delay"`                     // 结算延迟天数
	IsActive      bool      `gorm:"column:is_active;not null;default:true" json:"is_active"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (CommissionSetting) TableName() string {
	return "commission_settings"
}
