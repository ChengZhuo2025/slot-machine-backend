package models

import (
	"time"
)

// Order 订单模型
type Order struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderNo         string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"order_no"`
	UserID          int64      `gorm:"index;not null" json:"user_id"`
	Type            string     `gorm:"type:varchar(20);not null" json:"type"`
	Status          int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	TotalAmount     float64    `gorm:"type:decimal(12,2);not null" json:"total_amount"`
	DiscountAmount  float64    `gorm:"type:decimal(12,2);not null;default:0" json:"discount_amount"`
	ActualAmount    float64    `gorm:"type:decimal(12,2);not null" json:"actual_amount"`
	CouponID        *int64     `json:"coupon_id,omitempty"`
	AddressID       *int64     `json:"address_id,omitempty"`
	ShippingFee     float64    `gorm:"type:decimal(10,2);not null;default:0" json:"shipping_fee"`
	ShippingNo      *string    `gorm:"type:varchar(64)" json:"shipping_no,omitempty"`
	ShippingCompany *string    `gorm:"type:varchar(50)" json:"shipping_company,omitempty"`
	Remark          *string    `gorm:"type:varchar(255)" json:"remark,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	ShippedAt       *time.Time `json:"shipped_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	CancelReason    *string    `gorm:"type:varchar(255)" json:"cancel_reason,omitempty"`
	ExpiredAt       *time.Time `json:"expired_at,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	User      *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Coupon    *Coupon     `gorm:"foreignKey:CouponID" json:"coupon,omitempty"`
	Address   *Address    `gorm:"foreignKey:AddressID" json:"address,omitempty"`
	Items     []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Payments  []Payment   `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
}

// TableName 表名
func (Order) TableName() string {
	return "orders"
}

// OrderType 订单类型
const (
	OrderTypeMall   = "mall"   // 商城订单
	OrderTypeRental = "rental" // 租借订单
	OrderTypeHotel  = "hotel"  // 酒店预订
)

// OrderStatus 订单状态
const (
	OrderStatusPending    = 0 // 待支付
	OrderStatusPaid       = 1 // 已支付
	OrderStatusShipping   = 2 // 配送中
	OrderStatusDelivered  = 3 // 已送达
	OrderStatusCompleted  = 4 // 已完成
	OrderStatusCancelled  = 5 // 已取消
	OrderStatusRefunding  = 6 // 退款中
	OrderStatusRefunded   = 7 // 已退款
)

// OrderItem 订单项
type OrderItem struct {
	ID           int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID      int64   `gorm:"index;not null" json:"order_id"`
	ProductID    int64   `gorm:"not null" json:"product_id"`
	SkuID        int64   `gorm:"not null" json:"sku_id"`
	ProductName  string  `gorm:"type:varchar(200);not null" json:"product_name"`
	SkuName      *string `gorm:"type:varchar(100)" json:"sku_name,omitempty"`
	ProductImage *string `gorm:"type:varchar(255)" json:"product_image,omitempty"`
	Price        float64 `gorm:"type:decimal(10,2);not null" json:"price"`
	Quantity     int     `gorm:"not null" json:"quantity"`
	TotalAmount  float64 `gorm:"type:decimal(12,2);not null" json:"total_amount"`

	// 关联
	Order   *Order      `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Product *Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Sku     *ProductSku `gorm:"foreignKey:SkuID" json:"sku,omitempty"`
}

// TableName 表名
func (OrderItem) TableName() string {
	return "order_items"
}

// Rental 租借订单
type Rental struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	RentalNo       string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"rental_no"`
	UserID         int64      `gorm:"index;not null" json:"user_id"`
	DeviceID       int64      `gorm:"index;not null" json:"device_id"`
	SlotNo         *int       `json:"slot_no,omitempty"`
	PricingID      int64      `gorm:"not null" json:"pricing_id"`
	Status         int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	StartTime      *time.Time `json:"start_time,omitempty"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	Duration       *int       `json:"duration,omitempty"`
	UnitPrice      float64    `gorm:"type:decimal(10,2);not null" json:"unit_price"`
	DepositAmount  float64    `gorm:"type:decimal(10,2);not null;default:0" json:"deposit_amount"`
	RentalAmount   float64    `gorm:"type:decimal(10,2);not null;default:0" json:"rental_amount"`
	DiscountAmount float64    `gorm:"type:decimal(10,2);not null;default:0" json:"discount_amount"`
	ActualAmount   float64    `gorm:"type:decimal(10,2);not null;default:0" json:"actual_amount"`
	RefundAmount   float64    `gorm:"type:decimal(10,2);not null;default:0" json:"refund_amount"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	ReturnedAt     *time.Time `json:"returned_at,omitempty"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	User    *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Device  *Device        `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	Pricing *RentalPricing `gorm:"foreignKey:PricingID" json:"pricing,omitempty"`
}

// TableName 表名
func (Rental) TableName() string {
	return "rentals"
}

// RentalStatus 租借状态
const (
	RentalStatusPending   = 0 // 待支付
	RentalStatusPaid      = 1 // 已支付(待取货)
	RentalStatusInUse     = 2 // 使用中
	RentalStatusReturned  = 3 // 已归还
	RentalStatusCompleted = 4 // 已完成
	RentalStatusCancelled = 5 // 已取消
	RentalStatusOverdue   = 6 // 超时未还
)

// RentalPricing 租借定价
type RentalPricing struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID     int64     `gorm:"index;not null" json:"device_id"`
	Name         string    `gorm:"type:varchar(50);not null" json:"name"`
	Duration     int       `gorm:"not null" json:"duration"`
	DurationUnit string    `gorm:"type:varchar(10);not null;default:'hour'" json:"duration_unit"`
	Price        float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	OriginalPrice *float64 `gorm:"type:decimal(10,2)" json:"original_price,omitempty"`
	Deposit      float64   `gorm:"type:decimal(10,2);not null;default:0" json:"deposit"`
	IsDefault    bool      `gorm:"not null;default:false" json:"is_default"`
	Sort         int       `gorm:"not null;default:0" json:"sort"`
	Status       int8      `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// TableName 表名
func (RentalPricing) TableName() string {
	return "rental_pricings"
}

// DurationUnit 时长单位
const (
	DurationUnitMinute = "minute" // 分钟
	DurationUnitHour   = "hour"   // 小时
	DurationUnitDay    = "day"    // 天
)

// RentalPricingStatus 定价状态
const (
	RentalPricingStatusDisabled = 0 // 禁用
	RentalPricingStatusActive   = 1 // 启用
)
