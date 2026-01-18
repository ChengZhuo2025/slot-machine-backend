package models

import (
	"time"
)

// Coupon 优惠券模型
type Coupon struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string     `gorm:"type:varchar(100);not null" json:"name"`
	Type            string     `gorm:"type:varchar(20);not null" json:"type"`
	Value           float64    `gorm:"type:decimal(10,2);not null" json:"value"`
	MinAmount       float64    `gorm:"type:decimal(10,2);not null;default:0" json:"min_amount"`
	MaxDiscount     *float64   `gorm:"type:decimal(10,2)" json:"max_discount,omitempty"`
	TotalCount      int        `gorm:"not null" json:"total_count"`
	UsedCount       int        `gorm:"not null;default:0" json:"used_count"`
	ReceivedCount   int        `gorm:"column:issued_count;not null;default:0" json:"received_count"`
	PerUserLimit    int        `gorm:"not null;default:1" json:"per_user_limit"`
	ApplicableScope string     `gorm:"type:varchar(20);not null;default:'all'" json:"applicable_scope"`
	ApplicableIDs   JSON       `gorm:"type:jsonb" json:"applicable_ids,omitempty"`
	StartTime       time.Time  `gorm:"not null" json:"start_time"`
	EndTime         time.Time  `gorm:"not null" json:"end_time"`
	ValidDays       *int       `json:"valid_days,omitempty"`
	Description     *string    `gorm:"type:varchar(255)" json:"description,omitempty"`
	Status          int8       `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	UserCoupons []UserCoupon `gorm:"foreignKey:CouponID" json:"user_coupons,omitempty"`
}

// TableName 表名
func (Coupon) TableName() string {
	return "coupons"
}

// CouponType 优惠券类型
const (
	CouponTypeFixed   = "fixed"   // 固定金额
	CouponTypePercent = "percent" // 百分比折扣
)

// CouponScope 适用范围
const (
	CouponScopeAll      = "all"      // 全场通用
	CouponScopeCategory = "category" // 指定分类
	CouponScopeProduct  = "product"  // 指定商品
)

// CouponStatus 优惠券状态
const (
	CouponStatusDisabled = 0 // 禁用
	CouponStatusActive   = 1 // 启用
)

// UserCoupon 用户优惠券
type UserCoupon struct {
	ID         int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64      `gorm:"index;not null" json:"user_id"`
	CouponID   int64      `gorm:"index;not null" json:"coupon_id"`
	OrderID    *int64     `json:"order_id,omitempty"`
	Status     int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	ExpiredAt  time.Time  `gorm:"not null" json:"expired_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
	ReceivedAt time.Time  `gorm:"autoCreateTime" json:"received_at"`

	// 关联
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Coupon *Coupon `gorm:"foreignKey:CouponID" json:"coupon,omitempty"`
	Order  *Order  `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

// TableName 表名
func (UserCoupon) TableName() string {
	return "user_coupons"
}

// UserCouponStatus 用户优惠券状态
const (
	UserCouponStatusUnused  = 0 // 未使用
	UserCouponStatusUsed    = 1 // 已使用
	UserCouponStatusExpired = 2 // 已过期
)

// Campaign 活动模型
type Campaign struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Type        string    `gorm:"type:varchar(20);not null" json:"type"`
	Description *string   `gorm:"type:text" json:"description,omitempty"`
	Image       *string   `gorm:"type:varchar(255)" json:"image,omitempty"`
	Rules       JSON      `gorm:"type:jsonb" json:"rules,omitempty"`
	StartTime   time.Time `gorm:"not null" json:"start_time"`
	EndTime     time.Time `gorm:"not null" json:"end_time"`
	Status      int8      `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (Campaign) TableName() string {
	return "campaigns"
}

// CampaignType 活动类型
const (
	CampaignTypeDiscount  = "discount"  // 满减
	CampaignTypeGift      = "gift"      // 满赠
	CampaignTypeFlashSale = "flashsale" // 秒杀
	CampaignTypeGroupBuy  = "groupbuy"  // 团购
)

// CampaignStatus 活动状态
const (
	CampaignStatusDisabled = 0 // 禁用
	CampaignStatusActive   = 1 // 启用
)

// MemberPackage 会员套餐
type MemberPackage struct {
	ID             int64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string   `gorm:"type:varchar(50);not null" json:"name"`
	MemberLevelID  int64    `gorm:"not null" json:"member_level_id"`
	Duration       int      `gorm:"not null" json:"duration"`
	DurationUnit   string   `gorm:"type:varchar(10);not null;default:'month'" json:"duration_unit"`
	Price          float64  `gorm:"type:decimal(10,2);not null" json:"price"`
	OriginalPrice  *float64 `gorm:"type:decimal(10,2)" json:"original_price,omitempty"`
	GiftPoints     int      `gorm:"not null;default:0" json:"gift_points"`
	GiftCouponIDs  JSON     `gorm:"type:jsonb" json:"gift_coupon_ids,omitempty"`
	Description    *string  `gorm:"type:text" json:"description,omitempty"`
	Benefits       JSON     `gorm:"type:jsonb" json:"benefits,omitempty"`
	Sort           int      `gorm:"not null;default:0" json:"sort"`
	IsRecommend    bool     `gorm:"not null;default:false" json:"is_recommend"`
	Status         int8     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	MemberLevel *MemberLevel `gorm:"foreignKey:MemberLevelID" json:"member_level,omitempty"`
}

// TableName 表名
func (MemberPackage) TableName() string {
	return "member_packages"
}

// PackageDurationUnit 套餐时长单位
const (
	PackageDurationUnitDay   = "day"   // 天
	PackageDurationUnitMonth = "month" // 月
	PackageDurationUnitYear  = "year"  // 年
)

// MemberPackageStatus 套餐状态
const (
	MemberPackageStatusDisabled = 0 // 禁用
	MemberPackageStatusActive   = 1 // 启用
)
