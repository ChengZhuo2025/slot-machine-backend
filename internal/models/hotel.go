package models

import (
	"time"
)

// Hotel 酒店模型
type Hotel struct {
	ID           int64    `gorm:"primaryKey;autoIncrement" json:"id"`
	MerchantID   int64    `gorm:"index;not null" json:"merchant_id"`
	Name         string   `gorm:"type:varchar(100);not null" json:"name"`
	Stars        *int     `json:"stars,omitempty"`
	Province     string   `gorm:"type:varchar(50);not null" json:"province"`
	City         string   `gorm:"type:varchar(50);not null" json:"city"`
	District     string   `gorm:"type:varchar(50);not null" json:"district"`
	Address      string   `gorm:"type:varchar(255);not null" json:"address"`
	Longitude    *float64 `gorm:"type:decimal(10,7)" json:"longitude,omitempty"`
	Latitude     *float64 `gorm:"type:decimal(10,7)" json:"latitude,omitempty"`
	ContactName  *string  `gorm:"type:varchar(50)" json:"contact_name,omitempty"`
	ContactPhone *string  `gorm:"type:varchar(20)" json:"contact_phone,omitempty"`
	Description  *string  `gorm:"type:text" json:"description,omitempty"`
	Images       JSON     `gorm:"type:jsonb" json:"images,omitempty"`
	Facilities   JSON     `gorm:"type:jsonb" json:"facilities,omitempty"`
	Status       int8     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Merchant *Merchant `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
	Rooms    []Room    `gorm:"foreignKey:HotelID" json:"rooms,omitempty"`
}

// TableName 表名
func (Hotel) TableName() string {
	return "hotels"
}

// HotelStatus 酒店状态
const (
	HotelStatusDisabled = 0 // 禁用
	HotelStatusActive   = 1 // 正常
)

// Room 房间模型
type Room struct {
	ID              int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	HotelID         int64   `gorm:"index;not null" json:"hotel_id"`
	RoomNo          string  `gorm:"type:varchar(20);not null" json:"room_no"`
	Floor           *int    `json:"floor,omitempty"`
	Type            string  `gorm:"type:varchar(50);not null" json:"type"`
	Area            *float64 `gorm:"type:decimal(6,2)" json:"area,omitempty"`
	BedType         *string `gorm:"type:varchar(50)" json:"bed_type,omitempty"`
	MaxGuests       int     `gorm:"not null;default:2" json:"max_guests"`
	Description     *string `gorm:"type:text" json:"description,omitempty"`
	Images          JSON    `gorm:"type:jsonb" json:"images,omitempty"`
	Facilities      JSON    `gorm:"type:jsonb" json:"facilities,omitempty"`
	BasePrice       float64 `gorm:"type:decimal(10,2);not null" json:"base_price"`
	DeviceID        *int64  `json:"device_id,omitempty"`
	Status          int8    `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Hotel     *Hotel       `gorm:"foreignKey:HotelID" json:"hotel,omitempty"`
	Device    *Device      `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	TimeSlots []RoomTimeSlot `gorm:"foreignKey:RoomID" json:"time_slots,omitempty"`
}

// TableName 表名
func (Room) TableName() string {
	return "rooms"
}

// RoomType 房间类型
const (
	RoomTypeStandard  = "standard"  // 标准间
	RoomTypeBusiness  = "business"  // 商务间
	RoomTypeDeluxe    = "deluxe"    // 豪华间
	RoomTypeSuite     = "suite"     // 套房
)

// RoomStatus 房间状态
const (
	RoomStatusDisabled = 0 // 禁用
	RoomStatusActive   = 1 // 正常
	RoomStatusBooked   = 2 // 已预订
)

// RoomTimeSlot 房间时段
type RoomTimeSlot struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	RoomID    int64     `gorm:"index;not null" json:"room_id"`
	Date      time.Time `gorm:"type:date;not null;index" json:"date"`
	StartTime string    `gorm:"type:varchar(5);not null" json:"start_time"`
	EndTime   string    `gorm:"type:varchar(5);not null" json:"end_time"`
	Price     float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	Status    int8      `gorm:"type:smallint;not null;default:0" json:"status"`
	BookingID *int64    `json:"booking_id,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Room    *Room    `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Booking *Booking `gorm:"foreignKey:BookingID" json:"booking,omitempty"`
}

// TableName 表名
func (RoomTimeSlot) TableName() string {
	return "room_time_slots"
}

// TimeSlotStatus 时段状态
const (
	TimeSlotStatusAvailable = 0 // 可预订
	TimeSlotStatusBooked    = 1 // 已预订
	TimeSlotStatusUsing     = 2 // 使用中
	TimeSlotStatusCompleted = 3 // 已完成
)

// Booking 预订模型
type Booking struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	BookingNo      string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"booking_no"`
	UserID         int64      `gorm:"index;not null" json:"user_id"`
	RoomID         int64      `gorm:"index;not null" json:"room_id"`
	TimeSlotID     int64      `gorm:"not null" json:"time_slot_id"`
	CheckInDate    time.Time  `gorm:"type:date;not null" json:"check_in_date"`
	CheckInTime    string     `gorm:"type:varchar(5);not null" json:"check_in_time"`
	CheckOutTime   string     `gorm:"type:varchar(5);not null" json:"check_out_time"`
	GuestName      string     `gorm:"type:varchar(50);not null" json:"guest_name"`
	GuestPhone     string     `gorm:"type:varchar(20);not null" json:"guest_phone"`
	GuestCount     int        `gorm:"not null;default:1" json:"guest_count"`
	TotalAmount    float64    `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	DiscountAmount float64    `gorm:"type:decimal(10,2);not null;default:0" json:"discount_amount"`
	ActualAmount   float64    `gorm:"type:decimal(10,2);not null" json:"actual_amount"`
	Status         int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	Remark         *string    `gorm:"type:varchar(255)" json:"remark,omitempty"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	CheckedInAt    *time.Time `json:"checked_in_at,omitempty"`
	CheckedOutAt   *time.Time `json:"checked_out_at,omitempty"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
	CancelReason   *string    `gorm:"type:varchar(255)" json:"cancel_reason,omitempty"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	User     *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Room     *Room         `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	TimeSlot *RoomTimeSlot `gorm:"foreignKey:TimeSlotID" json:"time_slot,omitempty"`
}

// TableName 表名
func (Booking) TableName() string {
	return "bookings"
}

// BookingStatus 预订状态
const (
	BookingStatusPending    = 0 // 待支付
	BookingStatusPaid       = 1 // 已支付
	BookingStatusCheckedIn  = 2 // 已入住
	BookingStatusCheckedOut = 3 // 已退房
	BookingStatusCancelled  = 4 // 已取消
	BookingStatusRefunded   = 5 // 已退款
)
