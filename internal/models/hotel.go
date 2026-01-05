package models

import (
	"time"
)

// Hotel 酒店模型
type Hotel struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name           string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	StarRating     *int      `gorm:"column:star_rating;type:smallint" json:"star_rating,omitempty"`
	Province       string    `gorm:"column:province;type:varchar(50);not null" json:"province"`
	City           string    `gorm:"column:city;type:varchar(50);not null" json:"city"`
	District       string    `gorm:"column:district;type:varchar(50);not null" json:"district"`
	Address        string    `gorm:"column:address;type:varchar(255);not null" json:"address"`
	Longitude      *float64  `gorm:"column:longitude;type:decimal(10,7)" json:"longitude,omitempty"`
	Latitude       *float64  `gorm:"column:latitude;type:decimal(10,7)" json:"latitude,omitempty"`
	Phone          string    `gorm:"column:phone;type:varchar(20);not null" json:"phone"`
	Images         JSON      `gorm:"column:images;type:jsonb" json:"images,omitempty"`
	Facilities     JSON      `gorm:"column:facilities;type:jsonb" json:"facilities,omitempty"`
	Description    *string   `gorm:"column:description;type:text" json:"description,omitempty"`
	CheckInTime    string    `gorm:"column:check_in_time;type:time;not null;default:'14:00'" json:"check_in_time"`
	CheckOutTime   string    `gorm:"column:check_out_time;type:time;not null;default:'12:00'" json:"check_out_time"`
	CommissionRate float64   `gorm:"column:commission_rate;type:decimal(5,4);not null;default:0.1500" json:"commission_rate"`
	Status         int8      `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	Rooms []Room `gorm:"foreignKey:HotelID" json:"rooms,omitempty"`
}

// TableName 表名
func (Hotel) TableName() string {
	return "hotels"
}

// HotelStatus 酒店状态
const (
	HotelStatusDisabled = 0 // 下架
	HotelStatusActive   = 1 // 上架
)

// Room 房间模型
type Room struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	HotelID     int64     `gorm:"column:hotel_id;index;not null" json:"hotel_id"`
	RoomNo      string    `gorm:"column:room_no;type:varchar(20);not null" json:"room_no"`
	RoomType    string    `gorm:"column:room_type;type:varchar(50);not null" json:"room_type"`
	DeviceID    *int64    `gorm:"column:device_id" json:"device_id,omitempty"`
	Images      JSON      `gorm:"column:images;type:jsonb" json:"images,omitempty"`
	Facilities  JSON      `gorm:"column:facilities;type:jsonb" json:"facilities,omitempty"`
	Area        *int      `gorm:"column:area" json:"area,omitempty"`
	BedType     *string   `gorm:"column:bed_type;type:varchar(50)" json:"bed_type,omitempty"`
	MaxGuests   int       `gorm:"column:max_guests;not null;default:2" json:"max_guests"`
	HourlyPrice float64   `gorm:"column:hourly_price;type:decimal(10,2);not null" json:"hourly_price"`
	DailyPrice  float64   `gorm:"column:daily_price;type:decimal(10,2);not null" json:"daily_price"`
	Status      int8      `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	Hotel     *Hotel          `gorm:"foreignKey:HotelID" json:"hotel,omitempty"`
	Device    *Device         `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	TimeSlots []RoomTimeSlot  `gorm:"foreignKey:RoomID" json:"time_slots,omitempty"`
	Bookings  []Booking       `gorm:"foreignKey:RoomID" json:"bookings,omitempty"`
}

// TableName 表名
func (Room) TableName() string {
	return "rooms"
}

// RoomType 房间类型
const (
	RoomTypeStandard = "standard" // 标准间
	RoomTypeBusiness = "business" // 商务间
	RoomTypeDeluxe   = "deluxe"   // 豪华间
	RoomTypeSuite    = "suite"    // 套房
)

// RoomStatus 房间状态
const (
	RoomStatusDisabled = 0 // 停用
	RoomStatusActive   = 1 // 可用
	RoomStatusBooked   = 2 // 已预订
	RoomStatusInUse    = 3 // 使用中
)

// RoomTimeSlot 房间时段价格
type RoomTimeSlot struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RoomID        int64      `gorm:"column:room_id;index;not null" json:"room_id"`
	DurationHours int        `gorm:"column:duration_hours;not null" json:"duration_hours"`
	Price         float64    `gorm:"column:price;type:decimal(10,2);not null" json:"price"`
	StartTime     *string    `gorm:"column:start_time;type:time" json:"start_time,omitempty"`
	EndTime       *string    `gorm:"column:end_time;type:time" json:"end_time,omitempty"`
	IsActive      bool       `gorm:"column:is_active;not null;default:true" json:"is_active"`
	Sort          int        `gorm:"column:sort;not null;default:0" json:"sort"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	Room *Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

// TableName 表名
func (RoomTimeSlot) TableName() string {
	return "room_time_slots"
}

// Booking 预订记录模型
type Booking struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BookingNo        string     `gorm:"column:booking_no;type:varchar(64);uniqueIndex;not null" json:"booking_no"`
	OrderID          int64      `gorm:"column:order_id;uniqueIndex;not null" json:"order_id"`
	UserID           int64      `gorm:"column:user_id;index;not null" json:"user_id"`
	HotelID          int64      `gorm:"column:hotel_id;index;not null" json:"hotel_id"`
	RoomID           int64      `gorm:"column:room_id;index;not null" json:"room_id"`
	DeviceID         *int64     `gorm:"column:device_id" json:"device_id,omitempty"`
	CheckInTime      time.Time  `gorm:"column:check_in_time;not null" json:"check_in_time"`
	CheckOutTime     time.Time  `gorm:"column:check_out_time;not null" json:"check_out_time"`
	DurationHours    int        `gorm:"column:duration_hours;not null" json:"duration_hours"`
	Amount           float64    `gorm:"column:amount;type:decimal(10,2);not null" json:"amount"`
	VerificationCode string     `gorm:"column:verification_code;type:varchar(20);not null" json:"verification_code"`
	UnlockCode       string     `gorm:"column:unlock_code;type:varchar(10);not null" json:"unlock_code"`
	QRCode           string     `gorm:"column:qr_code;type:varchar(255);not null" json:"qr_code"`
	Status           string     `gorm:"column:status;type:varchar(20);not null" json:"status"`
	VerifiedAt       *time.Time `gorm:"column:verified_at" json:"verified_at,omitempty"`
	VerifiedBy       *int64     `gorm:"column:verified_by" json:"verified_by,omitempty"`
	UnlockedAt       *time.Time `gorm:"column:unlocked_at" json:"unlocked_at,omitempty"`
	CompletedAt      *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	Order    *Order  `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	User     *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Hotel    *Hotel  `gorm:"foreignKey:HotelID" json:"hotel,omitempty"`
	Room     *Room   `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Device   *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	Verifier *Admin  `gorm:"foreignKey:VerifiedBy" json:"verifier,omitempty"`
}

// TableName 表名
func (Booking) TableName() string {
	return "bookings"
}

// BookingStatus 预订状态
const (
	BookingStatusPending   = "pending"   // 待支付
	BookingStatusPaid      = "paid"      // 已支付/待核销
	BookingStatusVerified  = "verified"  // 已核销/待使用
	BookingStatusInUse     = "in_use"    // 使用中
	BookingStatusCompleted = "completed" // 已完成
	BookingStatusCancelled = "cancelled" // 已取消
	BookingStatusRefunded  = "refunded"  // 已退款
	BookingStatusExpired   = "expired"   // 已过期
)
