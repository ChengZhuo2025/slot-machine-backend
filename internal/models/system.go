package models

import (
	"time"
)

// Article 文章/公告
type Article struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Type        string     `gorm:"type:varchar(20);not null;index" json:"type"`
	Title       string     `gorm:"type:varchar(200);not null" json:"title"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Cover       *string    `gorm:"type:varchar(255)" json:"cover,omitempty"`
	Author      *string    `gorm:"type:varchar(50)" json:"author,omitempty"`
	ViewCount   int        `gorm:"not null;default:0" json:"view_count"`
	Sort        int        `gorm:"not null;default:0" json:"sort"`
	Status      int8       `gorm:"type:smallint;not null;default:1" json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (Article) TableName() string {
	return "articles"
}

// ArticleType 文章类型
const (
	ArticleTypeAnnouncement = "announcement" // 公告
	ArticleTypeHelp         = "help"         // 帮助
	ArticleTypeAgreement    = "agreement"    // 协议
	ArticleTypeAbout        = "about"        // 关于
)

// ArticleStatus 文章状态
const (
	ArticleStatusDraft     = 0 // 草稿
	ArticleStatusPublished = 1 // 已发布
)

// Notification 通知消息
type Notification struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64      `gorm:"index;not null" json:"user_id"`
	Type        string     `gorm:"type:varchar(20);not null" json:"type"`
	Title       string     `gorm:"type:varchar(100);not null" json:"title"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	ExtraData   JSON       `gorm:"type:jsonb" json:"extra_data,omitempty"`
	IsRead      bool       `gorm:"not null;default:false" json:"is_read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 表名
func (Notification) TableName() string {
	return "notifications"
}

// NotificationType 通知类型
const (
	NotificationTypeSystem  = "system"  // 系统通知
	NotificationTypeOrder   = "order"   // 订单通知
	NotificationTypeRental  = "rental"  // 租借通知
	NotificationTypePromo   = "promo"   // 促销通知
)

// MessageTemplate 消息模板
type MessageTemplate struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Type      string    `gorm:"type:varchar(20);not null" json:"type"`
	Title     *string   `gorm:"type:varchar(200)" json:"title,omitempty"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Variables JSON      `gorm:"type:jsonb" json:"variables,omitempty"`
	Status    int8      `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (MessageTemplate) TableName() string {
	return "message_templates"
}

// MessageTemplateType 模板类型
const (
	MessageTemplateTypeSMS      = "sms"      // 短信
	MessageTemplateTypeWechat   = "wechat"   // 微信
	MessageTemplateTypeEmail    = "email"    // 邮件
	MessageTemplateTypeInApp    = "in_app"   // 站内信
)

// MessageTemplateStatus 模板状态
const (
	MessageTemplateStatusDisabled = 0 // 禁用
	MessageTemplateStatusActive   = 1 // 启用
)

// SystemConfig 系统配置
type SystemConfig struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Type        string    `gorm:"type:varchar(20);not null;default:'string'" json:"type"`
	Group       string    `gorm:"type:varchar(50);not null;default:'general'" json:"group"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description *string   `gorm:"type:varchar(255)" json:"description,omitempty"`
	Options     JSON      `gorm:"type:jsonb" json:"options,omitempty"`
	Sort        int       `gorm:"not null;default:0" json:"sort"`
	IsPublic    bool      `gorm:"not null;default:false" json:"is_public"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// ConfigValueType 配置值类型
const (
	ConfigTypeString  = "string"  // 字符串
	ConfigTypeNumber  = "number"  // 数字
	ConfigTypeBoolean = "boolean" // 布尔
	ConfigTypeJSON    = "json"    // JSON
	ConfigTypeText    = "text"    // 长文本
)

// ConfigGroup 配置分组
const (
	ConfigGroupGeneral  = "general"  // 通用
	ConfigGroupPayment  = "payment"  // 支付
	ConfigGroupSMS      = "sms"      // 短信
	ConfigGroupWechat   = "wechat"   // 微信
	ConfigGroupStorage  = "storage"  // 存储
)

// Banner 轮播图
type Banner struct {
	ID        int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Position  string     `gorm:"type:varchar(50);not null;index" json:"position"`
	Title     string     `gorm:"type:varchar(100);not null" json:"title"`
	Image     string     `gorm:"type:varchar(255);not null" json:"image"`
	LinkType  string     `gorm:"type:varchar(20);not null;default:'none'" json:"link_type"`
	LinkValue *string    `gorm:"type:varchar(255)" json:"link_value,omitempty"`
	Sort      int        `gorm:"not null;default:0" json:"sort"`
	Status    int8       `gorm:"type:smallint;not null;default:1" json:"status"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (Banner) TableName() string {
	return "banners"
}

// BannerPosition 轮播图位置
const (
	BannerPositionHome    = "home"    // 首页
	BannerPositionMall    = "mall"    // 商城
	BannerPositionRental  = "rental"  // 租借
)

// BannerLinkType 链接类型
const (
	BannerLinkTypeNone    = "none"    // 无链接
	BannerLinkTypeProduct = "product" // 商品
	BannerLinkTypePage    = "page"    // 页面
	BannerLinkTypeURL     = "url"     // 外部链接
)

// BannerStatus 轮播图状态
const (
	BannerStatusDisabled = 0 // 禁用
	BannerStatusActive   = 1 // 启用
)

// SmsCode 短信验证码
type SmsCode struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Phone     string    `gorm:"type:varchar(20);not null;index" json:"phone"`
	Code      string    `gorm:"type:varchar(10);not null" json:"code"`
	Type      string    `gorm:"type:varchar(20);not null" json:"type"`
	IP        string    `gorm:"type:varchar(45);not null" json:"ip"`
	IsUsed    bool      `gorm:"not null;default:false" json:"is_used"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiredAt time.Time `gorm:"not null" json:"expired_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 表名
func (SmsCode) TableName() string {
	return "sms_codes"
}

// SmsCodeType 验证码类型
const (
	SmsCodeTypeLogin    = "login"    // 登录
	SmsCodeTypeRegister = "register" // 注册
	SmsCodeTypeBind     = "bind"     // 绑定
	SmsCodeTypeReset    = "reset"    // 重置密码
)
