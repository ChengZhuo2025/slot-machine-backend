package models

import (
	"encoding/json"
	"time"
)

// Order 订单模型
type Order struct {
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderNo         string          `gorm:"column:order_no;type:varchar(64);uniqueIndex;not null" json:"order_no"`
	UserID          int64           `gorm:"column:user_id;index;not null" json:"user_id"`
	Type            string          `gorm:"column:type;type:varchar(20);not null" json:"type"`
	OriginalAmount  float64         `gorm:"column:original_amount;type:decimal(12,2);not null" json:"original_amount"`
	DiscountAmount  float64         `gorm:"column:discount_amount;type:decimal(12,2);not null;default:0" json:"discount_amount"`
	ActualAmount    float64         `gorm:"column:actual_amount;type:decimal(12,2);not null" json:"actual_amount"`
	DepositAmount   float64         `gorm:"column:deposit_amount;type:decimal(12,2);not null;default:0" json:"deposit_amount"`
	Status          string          `gorm:"column:status;type:varchar(20);not null" json:"status"`
	CouponID        *int64          `gorm:"column:coupon_id" json:"coupon_id,omitempty"`
	Remark          *string         `gorm:"column:remark;type:varchar(255)" json:"remark,omitempty"`
	AddressID       *int64          `gorm:"column:address_id" json:"address_id,omitempty"`
	AddressSnapshot json.RawMessage `gorm:"column:address_snapshot;type:jsonb" json:"address_snapshot,omitempty"`
	ExpressCompany  *string         `gorm:"column:express_company;type:varchar(50)" json:"express_company,omitempty"`
	ExpressNo       *string         `gorm:"column:express_no;type:varchar(64)" json:"express_no,omitempty"`
	ShippedAt       *time.Time      `gorm:"column:shipped_at" json:"shipped_at,omitempty"`
	ReceivedAt      *time.Time      `gorm:"column:received_at" json:"received_at,omitempty"`
	PaidAt          *time.Time      `gorm:"column:paid_at" json:"paid_at,omitempty"`
	CompletedAt     *time.Time      `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CancelledAt     *time.Time      `gorm:"column:cancelled_at" json:"cancelled_at,omitempty"`
	CancelReason    *string         `gorm:"column:cancel_reason;type:varchar(255)" json:"cancel_reason,omitempty"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	User     *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Coupon   *Coupon      `gorm:"foreignKey:CouponID" json:"coupon,omitempty"`
	Address  *Address     `gorm:"foreignKey:AddressID" json:"address,omitempty"`
	Items    []*OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Payments []Payment    `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
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
	OrderStatusPending     = "pending"      // 待支付
	OrderStatusPaid        = "paid"         // 已支付
	OrderStatusPendingShip = "pending_ship" // 待发货
	OrderStatusShipping    = "shipping"     // 配送中
	OrderStatusShipped     = "shipped"      // 已发货
	OrderStatusDelivered   = "delivered"    // 已送达
	OrderStatusCompleted   = "completed"    // 已完成
	OrderStatusCancelled   = "cancelled"    // 已取消
	OrderStatusRefunding   = "refunding"    // 退款中
	OrderStatusRefunded    = "refunded"     // 已退款
)

// OrderItem 订单项
type OrderItem struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID      int64     `gorm:"column:order_id;index;not null" json:"order_id"`
	ProductID    *int64    `gorm:"column:product_id" json:"product_id,omitempty"`
	ProductName  string    `gorm:"column:product_name;type:varchar(100);not null" json:"product_name"`
	ProductImage *string   `gorm:"column:product_image;type:varchar(255)" json:"product_image,omitempty"`
	SkuInfo      *string   `gorm:"column:sku_info;type:varchar(255)" json:"sku_info,omitempty"`
	Price        float64   `gorm:"column:price;type:decimal(12,2);not null" json:"price"`
	Quantity     int       `gorm:"column:quantity;not null" json:"quantity"`
	Subtotal     float64   `gorm:"column:subtotal;type:decimal(12,2);not null" json:"subtotal"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// 关联
	Order   *Order   `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// TableName 表名
func (OrderItem) TableName() string {
	return "order_items"
}

// Rental 租借订单
type Rental struct {
	ID                int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID           int64      `gorm:"column:order_id;uniqueIndex;not null" json:"order_id"`
	UserID            int64      `gorm:"column:user_id;index;not null" json:"user_id"`
	DeviceID          int64      `gorm:"column:device_id;index;not null" json:"device_id"`
	DurationHours     int        `gorm:"column:duration_hours;not null" json:"duration_hours"`
	RentalFee         float64    `gorm:"column:rental_fee;type:decimal(10,2);not null" json:"rental_fee"`
	Deposit           float64    `gorm:"column:deposit;type:decimal(10,2);not null" json:"deposit"`
	OvertimeRate      float64    `gorm:"column:overtime_rate;type:decimal(10,2);not null" json:"overtime_rate"`
	OvertimeFee       float64    `gorm:"column:overtime_fee;type:decimal(10,2);not null;default:0" json:"overtime_fee"`
	Status            string     `gorm:"column:status;type:varchar(20);not null" json:"status"`
	UnlockedAt        *time.Time `gorm:"column:unlocked_at" json:"unlocked_at,omitempty"`
	ExpectedReturnAt  *time.Time `gorm:"column:expected_return_at" json:"expected_return_at,omitempty"`
	ReturnedAt        *time.Time `gorm:"column:returned_at" json:"returned_at,omitempty"`
	IsPurchased       bool       `gorm:"column:is_purchased;not null;default:false" json:"is_purchased"`
	PurchasedAt       *time.Time `gorm:"column:purchased_at" json:"purchased_at,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	Order  *Order  `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// TableName 表名
func (Rental) TableName() string {
	return "rentals"
}

// RentalStatus 租借状态(字符串)
const (
	RentalStatusPending   = "pending"    // 待支付
	RentalStatusPaid      = "paid"       // 已支付(待取货)
	RentalStatusInUse     = "in_use"     // 使用中
	RentalStatusOverdue   = "overdue"    // 超时未还
	RentalStatusReturned  = "returned"   // 已归还
	RentalStatusCompleted = "completed"  // 已完成
	RentalStatusCancelled = "cancelled"  // 已取消
	RentalStatusRefunding = "refunding"  // 退款中
	RentalStatusRefunded  = "refunded"   // 已退款
)

// RentalPricing 租借定价
type RentalPricing struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	VenueID       *int64    `gorm:"column:venue_id;index" json:"venue_id,omitempty"`
	DurationHours int       `gorm:"column:duration_hours;not null" json:"duration_hours"`
	Price         float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	Deposit       float64   `gorm:"type:decimal(10,2);not null" json:"deposit"`
	OvertimeRate  float64   `gorm:"column:overtime_rate;type:decimal(10,2);not null" json:"overtime_rate"`
	IsActive      bool      `gorm:"column:is_active;not null;default:true" json:"is_active"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	Venue *Venue `gorm:"foreignKey:VenueID" json:"venue,omitempty"`
}

// TableName 表名
func (RentalPricing) TableName() string {
	return "rental_pricings"
}
