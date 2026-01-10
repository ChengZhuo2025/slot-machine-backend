package models

import (
	"time"
)

// Article 文章/公告
type Article struct {
	ID          int64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Category    string     `gorm:"type:varchar(20);not null;index;column:category" json:"category"`
	Title       string     `gorm:"type:varchar(200);not null;column:title" json:"title"`
	Content     string     `gorm:"type:text;not null;column:content" json:"content"`
	CoverImage  *string    `gorm:"type:varchar(255);column:cover_image" json:"cover_image,omitempty"`
	Sort        int        `gorm:"not null;default:0;column:sort" json:"sort"`
	ViewCount   int        `gorm:"not null;default:0;column:view_count" json:"view_count"`
	IsPublished bool       `gorm:"not null;default:true;column:is_published" json:"is_published"`
	PublishedAt *time.Time `gorm:"column:published_at" json:"published_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`
}

// TableName 表名
func (Article) TableName() string {
	return "articles"
}

// ArticleCategory 文章分类
const (
	ArticleCategoryHelp   = "help"   // 帮助中心
	ArticleCategoryFAQ    = "faq"    // 常见问题
	ArticleCategoryNotice = "notice" // 公告通知
	ArticleCategoryAbout  = "about"  // 关于我们
)

// ArticleStatus 文章状态
const (
	ArticleStatusDraft     = 0 // 草稿
	ArticleStatusPublished = 1 // 已发布
)

// Notification 通知消息
type Notification struct {
	ID        int64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID    *int64     `gorm:"index;column:user_id" json:"user_id,omitempty"`
	Type      string     `gorm:"type:varchar(20);not null;column:type" json:"type"`
	Title     string     `gorm:"type:varchar(100);not null;column:title" json:"title"`
	Content   string     `gorm:"type:text;not null;column:content" json:"content"`
	Link      *string    `gorm:"type:varchar(255);column:link" json:"link,omitempty"`
	IsRead    bool       `gorm:"not null;default:false;column:is_read" json:"is_read"`
	ReadAt    *time.Time `gorm:"column:read_at" json:"read_at,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime;column:created_at" json:"created_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 表名
func (Notification) TableName() string {
	return "notifications"
}

// NotificationType 通知类型
const (
	NotificationTypeSystem    = "system"    // 系统通知
	NotificationTypeOrder     = "order"     // 订单通知
	NotificationTypeMarketing = "marketing" // 营销通知
)

// MessageTemplate 消息模板
type MessageTemplate struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Code      string    `gorm:"type:varchar(50);uniqueIndex;not null;column:code" json:"code"`
	Name      string    `gorm:"type:varchar(100);not null;column:name" json:"name"`
	Type      string    `gorm:"type:varchar(20);not null;column:type" json:"type"`
	Content   string    `gorm:"type:text;not null;column:content" json:"content"`
	Variables JSON      `gorm:"type:jsonb;column:variables" json:"variables,omitempty"`
	IsActive  bool      `gorm:"not null;default:true;column:is_active" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`
}

// TableName 表名
func (MessageTemplate) TableName() string {
	return "message_templates"
}

// MessageTemplateType 模板类型
const (
	MessageTemplateTypeSMS    = "sms"    // 短信
	MessageTemplateTypePush   = "push"   // 推送
	MessageTemplateTypeWechat = "wechat" // 微信
)

// SystemConfig 系统配置
type SystemConfig struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Group       string    `gorm:"type:varchar(50);not null;column:group" json:"group"`
	Key         string    `gorm:"type:varchar(100);not null;column:key" json:"key"`
	Value       string    `gorm:"type:text;not null;column:value" json:"value"`
	Type        string    `gorm:"type:varchar(20);not null;default:'string';column:type" json:"type"`
	Description *string   `gorm:"type:varchar(255);column:description" json:"description,omitempty"`
	IsPublic    bool      `gorm:"not null;default:false;column:is_public" json:"is_public"`
	CreatedAt   time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`
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
	ID         int64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Title      string     `gorm:"type:varchar(100);not null;column:title" json:"title"`
	Image      string     `gorm:"type:varchar(255);not null;column:image" json:"image"`
	LinkType   *string    `gorm:"type:varchar(20);column:link_type" json:"link_type,omitempty"`
	LinkValue  *string    `gorm:"type:varchar(255);column:link_value" json:"link_value,omitempty"`
	Position   string     `gorm:"type:varchar(20);not null;index;column:position" json:"position"`
	Sort       int        `gorm:"not null;default:0;column:sort" json:"sort"`
	StartTime  *time.Time `gorm:"column:start_time" json:"start_time,omitempty"`
	EndTime    *time.Time `gorm:"column:end_time" json:"end_time,omitempty"`
	IsActive   bool       `gorm:"not null;default:true;column:is_active" json:"is_active"`
	ClickCount int        `gorm:"not null;default:0;column:click_count" json:"click_count"`
	CreatedAt  time.Time  `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`
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
	BannerStatusInactive = false // 禁用
	BannerStatusActive   = true  // 启用
)

// SmsCode 短信验证码
type SmsCode struct {
	ID        int64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Phone     string     `gorm:"type:varchar(20);not null;index;column:phone" json:"phone"`
	Code      string     `gorm:"type:varchar(10);not null;column:code" json:"code"`
	Type      string     `gorm:"type:varchar(20);not null;column:type" json:"type"`
	ExpireAt  time.Time  `gorm:"not null;column:expire_at" json:"expire_at"`
	IsUsed    bool       `gorm:"not null;default:false;column:is_used" json:"is_used"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"used_at,omitempty"`
	IP        *string    `gorm:"type:varchar(45);column:ip" json:"ip,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime;column:created_at" json:"created_at"`
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
