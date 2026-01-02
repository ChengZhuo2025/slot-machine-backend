package models

import (
	"time"
)

// Merchant 商户模型
type Merchant struct {
	ID                   int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                 string    `gorm:"type:varchar(100);not null" json:"name"`
	ContactName          string    `gorm:"type:varchar(50);not null" json:"contact_name"`
	ContactPhone         string    `gorm:"type:varchar(20);not null" json:"contact_phone"`
	Address              *string   `gorm:"type:varchar(255)" json:"address,omitempty"`
	BusinessLicense      *string   `gorm:"type:varchar(255)" json:"business_license,omitempty"`
	CommissionRate       float64   `gorm:"type:decimal(5,4);not null;default:0.2" json:"commission_rate"`
	SettlementType       string    `gorm:"type:varchar(20);not null;default:'monthly'" json:"settlement_type"`
	BankName             *string   `gorm:"type:varchar(100)" json:"bank_name,omitempty"`
	BankAccountEncrypted *string   `gorm:"type:text" json:"-"`
	BankHolderEncrypted  *string   `gorm:"type:text" json:"-"`
	Status               int8      `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt            time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Venues []Venue `gorm:"foreignKey:MerchantID" json:"venues,omitempty"`
}

// TableName 表名
func (Merchant) TableName() string {
	return "merchants"
}

// MerchantStatus 商户状态
const (
	MerchantStatusDisabled = 0 // 禁用
	MerchantStatusActive   = 1 // 正常
)

// SettlementType 结算周期类型
const (
	SettlementTypeWeekly  = "weekly"  // 周结
	SettlementTypeMonthly = "monthly" // 月结
)

// Venue 场地模型
type Venue struct {
	ID           int64    `gorm:"primaryKey;autoIncrement" json:"id"`
	MerchantID   int64    `gorm:"index;not null" json:"merchant_id"`
	Name         string   `gorm:"type:varchar(100);not null" json:"name"`
	Type         string   `gorm:"type:varchar(20);not null" json:"type"`
	Province     string   `gorm:"type:varchar(50);not null" json:"province"`
	City         string   `gorm:"type:varchar(50);not null" json:"city"`
	District     string   `gorm:"type:varchar(50);not null" json:"district"`
	Address      string   `gorm:"type:varchar(255);not null" json:"address"`
	Longitude    *float64 `gorm:"type:decimal(10,7)" json:"longitude,omitempty"`
	Latitude     *float64 `gorm:"type:decimal(10,7)" json:"latitude,omitempty"`
	ContactName  *string  `gorm:"type:varchar(50)" json:"contact_name,omitempty"`
	ContactPhone *string  `gorm:"type:varchar(20)" json:"contact_phone,omitempty"`
	Status       int8     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Merchant *Merchant `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
	Devices  []Device  `gorm:"foreignKey:VenueID" json:"devices,omitempty"`
}

// TableName 表名
func (Venue) TableName() string {
	return "venues"
}

// VenueType 场地类型
const (
	VenueTypeMall      = "mall"      // 商场
	VenueTypeHotel     = "hotel"     // 酒店
	VenueTypeCommunity = "community" // 社区
	VenueTypeOffice    = "office"    // 写字楼
	VenueTypeOther     = "other"     // 其他
)

// VenueStatus 场地状态
const (
	VenueStatusDisabled = 0 // 禁用
	VenueStatusActive   = 1 // 正常
)
