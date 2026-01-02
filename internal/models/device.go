package models

import (
	"time"
)

// Device 智能柜设备模型
type Device struct {
	ID               int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceNo         string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"device_no"`
	Name             string     `gorm:"type:varchar(100);not null" json:"name"`
	Type             string     `gorm:"type:varchar(20);not null" json:"type"`
	Model            *string    `gorm:"type:varchar(50)" json:"model,omitempty"`
	VenueID          int64      `gorm:"index;not null" json:"venue_id"`
	QRCode           string     `gorm:"type:varchar(255);not null" json:"qr_code"`
	ProductName      string     `gorm:"type:varchar(100);not null" json:"product_name"`
	ProductImage     *string    `gorm:"type:varchar(255)" json:"product_image,omitempty"`
	SlotCount        int        `gorm:"not null;default:1" json:"slot_count"`
	AvailableSlots   int        `gorm:"not null;default:1" json:"available_slots"`
	OnlineStatus     int8       `gorm:"type:smallint;not null;default:0" json:"online_status"`
	LockStatus       int8       `gorm:"type:smallint;not null;default:0" json:"lock_status"`
	RentalStatus     int8       `gorm:"type:smallint;not null;default:0" json:"rental_status"`
	CurrentRentalID  *int64     `json:"current_rental_id,omitempty"`
	FirmwareVersion  *string    `gorm:"type:varchar(20)" json:"firmware_version,omitempty"`
	NetworkType      string     `gorm:"type:varchar(20);default:'WiFi'" json:"network_type"`
	SignalStrength   *int       `json:"signal_strength,omitempty"`
	BatteryLevel     *int       `json:"battery_level,omitempty"`
	Temperature      *float64   `gorm:"type:decimal(5,2)" json:"temperature,omitempty"`
	Humidity         *float64   `gorm:"type:decimal(5,2)" json:"humidity,omitempty"`
	LastHeartbeatAt  *time.Time `json:"last_heartbeat_at,omitempty"`
	LastOnlineAt     *time.Time `json:"last_online_at,omitempty"`
	LastOfflineAt    *time.Time `json:"last_offline_at,omitempty"`
	InstallTime      *time.Time `json:"install_time,omitempty"`
	Status           int8       `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Venue         *Venue  `gorm:"foreignKey:VenueID" json:"venue,omitempty"`
	CurrentRental *Rental `gorm:"foreignKey:CurrentRentalID" json:"current_rental,omitempty"`
}

// TableName 表名
func (Device) TableName() string {
	return "devices"
}

// DeviceType 设备类型
const (
	DeviceTypeStandard = "standard" // 标准柜
	DeviceTypeMini     = "mini"     // 迷你柜
	DeviceTypePremium  = "premium"  // 豪华柜
)

// DeviceOnlineStatus 设备在线状态
const (
	DeviceOffline = 0 // 离线
	DeviceOnline  = 1 // 在线
)

// DeviceLockStatus 设备锁状态
const (
	DeviceLocked   = 0 // 锁定
	DeviceUnlocked = 1 // 打开
)

// DeviceRentalStatus 设备租借状态
const (
	DeviceRentalFree  = 0 // 空闲
	DeviceRentalInUse = 1 // 使用中
)

// DeviceStatus 设备状态
const (
	DeviceStatusDisabled    = 0 // 禁用
	DeviceStatusActive      = 1 // 正常
	DeviceStatusMaintenance = 2 // 维护中
	DeviceStatusFault       = 3 // 故障
)

// DeviceLog 设备日志
type DeviceLog struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID     int64     `gorm:"index;not null" json:"device_id"`
	Type         string    `gorm:"type:varchar(20);not null" json:"type"`
	Content      *string   `gorm:"type:text" json:"content,omitempty"`
	OperatorID   *int64    `json:"operator_id,omitempty"`
	OperatorType *string   `gorm:"type:varchar(10)" json:"operator_type,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime;index" json:"created_at"`

	// 关联
	Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// TableName 表名
func (DeviceLog) TableName() string {
	return "device_logs"
}

// DeviceLogType 设备日志类型
const (
	DeviceLogTypeOnline    = "online"    // 上线
	DeviceLogTypeOffline   = "offline"   // 离线
	DeviceLogTypeUnlock    = "unlock"    // 开锁
	DeviceLogTypeLock      = "lock"      // 锁定
	DeviceLogTypeError     = "error"     // 错误
	DeviceLogTypeHeartbeat = "heartbeat" // 心跳
)

// DeviceLogOperatorType 设备日志操作人类型
const (
	DeviceLogOperatorUser   = "user"   // 用户
	DeviceLogOperatorAdmin  = "admin"  // 管理员
	DeviceLogOperatorSystem = "system" // 系统
)

// DeviceMaintenance 设备维护记录
type DeviceMaintenance struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID     int64      `gorm:"index;not null" json:"device_id"`
	Type         string     `gorm:"type:varchar(20);not null" json:"type"`
	Description  string     `gorm:"type:text;not null" json:"description"`
	BeforeImages JSON       `gorm:"type:jsonb" json:"before_images,omitempty"`
	AfterImages  JSON       `gorm:"type:jsonb" json:"after_images,omitempty"`
	Cost         float64    `gorm:"type:decimal(10,2);not null;default:0" json:"cost"`
	OperatorID   int64      `gorm:"not null" json:"operator_id"`
	Status       int8       `gorm:"type:smallint;not null;default:0" json:"status"`
	StartedAt    time.Time  `gorm:"not null" json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Device   *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	Operator *Admin  `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
}

// TableName 表名
func (DeviceMaintenance) TableName() string {
	return "device_maintenances"
}

// MaintenanceType 维护类型
const (
	MaintenanceTypeRepair  = "repair"  // 维修
	MaintenanceTypeClean   = "clean"   // 清洁
	MaintenanceTypeReplace = "replace" // 更换
	MaintenanceTypeInspect = "inspect" // 巡检
)

// MaintenanceStatus 维护状态
const (
	MaintenanceStatusInProgress = 0 // 进行中
	MaintenanceStatusCompleted  = 1 // 已完成
)
