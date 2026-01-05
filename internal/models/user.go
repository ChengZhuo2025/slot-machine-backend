// Package models 定义数据模型
package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// User 用户模型
type User struct {
	ID                int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Phone             *string    `gorm:"type:varchar(20);uniqueIndex" json:"phone,omitempty"`
	OpenID            *string    `gorm:"column:openid;type:varchar(64);uniqueIndex" json:"openid,omitempty"`
	UnionID           *string    `gorm:"column:unionid;type:varchar(64)" json:"unionid,omitempty"`
	Nickname          string     `gorm:"type:varchar(50);not null;default:''" json:"nickname"`
	Avatar            *string    `gorm:"type:varchar(255)" json:"avatar,omitempty"`
	Gender            int8       `gorm:"type:smallint;not null;default:0" json:"gender"`
	Birthday          *time.Time `gorm:"type:date" json:"birthday,omitempty"`
	MemberLevelID     int64      `gorm:"not null;default:1" json:"member_level_id"`
	Points            int        `gorm:"not null;default:0" json:"points"`
	IsVerified        bool       `gorm:"not null;default:false" json:"is_verified"`
	RealNameEncrypted *string    `gorm:"type:text" json:"-"`
	IDCardEncrypted   *string    `gorm:"type:text" json:"-"`
	ReferrerID        *int64     `gorm:"index" json:"referrer_id,omitempty"`
	Status            int8       `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	MemberLevel *MemberLevel `gorm:"foreignKey:MemberLevelID" json:"member_level,omitempty"`
	Referrer    *User        `gorm:"foreignKey:ReferrerID" json:"referrer,omitempty"`
	Wallet      *UserWallet  `gorm:"foreignKey:UserID" json:"wallet,omitempty"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

// UserStatus 用户状态
const (
	UserStatusDisabled = 0 // 禁用
	UserStatusActive   = 1 // 正常
)

// Gender 性别
const (
	GenderUnknown = 0 // 未知
	GenderMale    = 1 // 男
	GenderFemale  = 2 // 女
)

// UserWallet 用户钱包
type UserWallet struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	Balance        float64   `gorm:"type:decimal(12,2);not null;default:0" json:"balance"`
	FrozenBalance  float64   `gorm:"type:decimal(12,2);not null;default:0" json:"frozen_balance"`
	TotalRecharged float64   `gorm:"type:decimal(12,2);not null;default:0" json:"total_recharged"`
	TotalConsumed  float64   `gorm:"type:decimal(12,2);not null;default:0" json:"total_consumed"`
	TotalWithdrawn float64   `gorm:"type:decimal(12,2);not null;default:0" json:"total_withdrawn"`
	Version        int       `gorm:"not null;default:0" json:"-"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (UserWallet) TableName() string {
	return "user_wallets"
}

// MemberLevel 会员等级
type MemberLevel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"type:varchar(50);not null" json:"name"`
	Level     int       `gorm:"uniqueIndex;not null" json:"level"`
	MinPoints int       `gorm:"not null;default:0" json:"min_points"`
	Discount  float64   `gorm:"type:decimal(3,2);not null;default:1.00" json:"discount"`
	Benefits  JSON      `gorm:"type:jsonb" json:"benefits,omitempty"`
	Icon      *string   `gorm:"type:varchar(255)" json:"icon,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 表名
func (MemberLevel) TableName() string {
	return "member_levels"
}

// Address 用户收货地址
type Address struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64     `gorm:"index;not null" json:"user_id"`
	ReceiverName  string    `gorm:"type:varchar(50);not null" json:"receiver_name"`
	ReceiverPhone string    `gorm:"type:varchar(20);not null" json:"receiver_phone"`
	Province      string    `gorm:"type:varchar(50);not null" json:"province"`
	City          string    `gorm:"type:varchar(50);not null" json:"city"`
	District      string    `gorm:"type:varchar(50);not null" json:"district"`
	Detail        string    `gorm:"type:varchar(255);not null" json:"detail"`
	PostalCode    *string   `gorm:"type:varchar(10)" json:"postal_code,omitempty"`
	IsDefault     bool      `gorm:"not null;default:false" json:"is_default"`
	Tag           *string   `gorm:"type:varchar(20)" json:"tag,omitempty"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (Address) TableName() string {
	return "addresses"
}

// UserFeedback 用户反馈
type UserFeedback struct {
	ID        int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64      `gorm:"index;not null" json:"user_id"`
	Type      string     `gorm:"type:varchar(20);not null" json:"type"`
	Content   string     `gorm:"type:text;not null" json:"content"`
	Images    JSON       `gorm:"type:jsonb" json:"images,omitempty"`
	Contact   *string    `gorm:"type:varchar(100)" json:"contact,omitempty"`
	Status    int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	Reply     *string    `gorm:"type:text" json:"reply,omitempty"`
	RepliedBy *int64     `json:"replied_by,omitempty"`
	RepliedAt *time.Time `json:"replied_at,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 表名
func (UserFeedback) TableName() string {
	return "user_feedbacks"
}

// FeedbackType 反馈类型
const (
	FeedbackTypeSuggestion = "suggestion" // 建议
	FeedbackTypeBug        = "bug"        // 问题
	FeedbackTypeComplaint  = "complaint"  // 投诉
	FeedbackTypeOther      = "other"      // 其他
)

// FeedbackStatus 反馈状态
const (
	FeedbackStatusPending    = 0 // 待处理
	FeedbackStatusProcessing = 1 // 处理中
	FeedbackStatusProcessed  = 2 // 已处理
)

// WalletTransaction 钱包交易记录
type WalletTransaction struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64     `gorm:"index;not null" json:"user_id"`
	Type          string    `gorm:"type:varchar(20);not null" json:"type"`
	Amount        float64   `gorm:"type:decimal(12,2);not null" json:"amount"`
	BalanceBefore float64   `gorm:"type:decimal(12,2);not null" json:"balance_before"`
	BalanceAfter  float64   `gorm:"type:decimal(12,2);not null" json:"balance_after"`
	OrderNo       *string   `gorm:"type:varchar(64);index" json:"order_no,omitempty"`
	Remark        *string   `gorm:"type:varchar(255)" json:"remark,omitempty"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 表名
func (WalletTransaction) TableName() string {
	return "wallet_transactions"
}

// WalletTransactionType 钱包交易类型
const (
	WalletTxTypeRecharge      = "recharge"       // 充值
	WalletTxTypeConsume       = "consume"        // 消费
	WalletTxTypeRefund        = "refund"         // 退款
	WalletTxTypeWithdraw      = "withdraw"       // 提现
	WalletTxTypeDeposit       = "deposit"        // 押金冻结
	WalletTxTypeReturnDeposit = "return_deposit" // 押金退还
)

// JSON 自定义 JSON 类型
type JSON map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现 driver.Valuer 接口
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Unmarshal 将 JSON 值反序列化到目标结构（便于业务层使用）
func (j JSON) Unmarshal(target interface{}) error {
	if j == nil {
		return nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
