package models

import (
	"time"
)

// Admin 管理员模型
type Admin struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	Name         string     `gorm:"type:varchar(50);not null" json:"name"`
	Phone        *string    `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Email        *string    `gorm:"type:varchar(100)" json:"email,omitempty"`
	RoleID       int64      `gorm:"not null" json:"role_id"`
	MerchantID   *int64     `json:"merchant_id,omitempty"`
	Status       int8       `gorm:"type:smallint;not null;default:1" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP  *string    `gorm:"type:varchar(45)" json:"last_login_ip,omitempty"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Role     *Role     `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Merchant *Merchant `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
}

// TableName 表名
func (Admin) TableName() string {
	return "admins"
}

// AdminStatus 管理员状态
const (
	AdminStatusDisabled = 0 // 禁用
	AdminStatusActive   = 1 // 正常
)

// Role 角色模型
type Role struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name        string    `gorm:"type:varchar(50);not null" json:"name"`
	Description *string   `gorm:"type:varchar(255)" json:"description,omitempty"`
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
}

// TableName 表名
func (Role) TableName() string {
	return "roles"
}

// RoleCode 预置角色编码
const (
	RoleCodeSuperAdmin     = "super_admin"     // 超级管理员
	RoleCodePlatformAdmin  = "platform_admin"  // 平台管理员
	RoleCodeOperationAdmin = "operation_admin" // 运营管理员
	RoleCodeFinanceAdmin   = "finance_admin"   // 财务管理员
	RoleCodePartner        = "partner"         // 合作商
	RoleCodeCustomerService = "customer_service" // 客服
)

// Permission 权限模型
type Permission struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Type      string    `gorm:"type:varchar(20);not null" json:"type"`
	ParentID  *int64    `gorm:"index" json:"parent_id,omitempty"`
	Path      *string   `gorm:"type:varchar(255)" json:"path,omitempty"`
	Method    *string   `gorm:"type:varchar(10)" json:"method,omitempty"`
	Sort      int       `gorm:"not null;default:0" json:"sort"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Parent   *Permission  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Permission `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

// TableName 表名
func (Permission) TableName() string {
	return "permissions"
}

// PermissionType 权限类型
const (
	PermissionTypeMenu = "menu" // 菜单
	PermissionTypeAPI  = "api"  // API
)

// RolePermission 角色权限关联表
type RolePermission struct {
	RoleID       int64 `gorm:"primaryKey" json:"role_id"`
	PermissionID int64 `gorm:"primaryKey" json:"permission_id"`
}

// TableName 表名
func (RolePermission) TableName() string {
	return "role_permissions"
}

// OperationLog 操作日志
type OperationLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID    int64     `gorm:"index;not null" json:"admin_id"`
	Module     string    `gorm:"type:varchar(50);not null" json:"module"`
	Action     string    `gorm:"type:varchar(50);not null" json:"action"`
	TargetType *string   `gorm:"type:varchar(50)" json:"target_type,omitempty"`
	TargetID   *int64    `json:"target_id,omitempty"`
	BeforeData JSON      `gorm:"type:jsonb" json:"before_data,omitempty"`
	AfterData  JSON      `gorm:"type:jsonb" json:"after_data,omitempty"`
	IP         string    `gorm:"type:varchar(45);not null" json:"ip"`
	UserAgent  *string   `gorm:"type:varchar(255)" json:"user_agent,omitempty"`
	CreatedAt  time.Time `gorm:"autoCreateTime;index" json:"created_at"`

	// 关联
	Admin *Admin `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}

// TableName 表名
func (OperationLog) TableName() string {
	return "operation_logs"
}
