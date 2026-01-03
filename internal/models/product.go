package models

import (
	"time"
)

// Category 商品分类
type Category struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID  *int64    `gorm:"index" json:"parent_id,omitempty"`
	Name      string    `gorm:"type:varchar(50);not null" json:"name"`
	Icon      *string   `gorm:"type:varchar(255)" json:"icon,omitempty"`
	Image     *string   `gorm:"type:varchar(255)" json:"image,omitempty"`
	Sort      int       `gorm:"not null;default:0" json:"sort"`
	Level     int       `gorm:"not null;default:1" json:"level"`
	Status    int8      `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

// TableName 表名
func (Category) TableName() string {
	return "categories"
}

// CategoryStatus 分类状态
const (
	CategoryStatusDisabled = 0 // 禁用
	CategoryStatusActive   = 1 // 启用
)

// Product 商品模型
type Product struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryID      int64     `gorm:"index;not null" json:"category_id"`
	Name            string    `gorm:"type:varchar(200);not null" json:"name"`
	Subtitle        *string   `gorm:"type:varchar(255)" json:"subtitle,omitempty"`
	MainImage       string    `gorm:"type:varchar(255);not null" json:"main_image"`
	Images          JSON      `gorm:"type:jsonb" json:"images,omitempty"`
	Video           *string   `gorm:"type:varchar(255)" json:"video,omitempty"`
	Description     *string   `gorm:"type:text" json:"description,omitempty"`
	Detail          *string   `gorm:"type:text" json:"detail,omitempty"`
	Brand           *string   `gorm:"type:varchar(50)" json:"brand,omitempty"`
	Specs           JSON      `gorm:"type:jsonb" json:"specs,omitempty"`
	MinPrice        float64   `gorm:"type:decimal(10,2);not null" json:"min_price"`
	MaxPrice        float64   `gorm:"type:decimal(10,2);not null" json:"max_price"`
	OriginalPrice   *float64  `gorm:"type:decimal(10,2)" json:"original_price,omitempty"`
	TotalStock      int       `gorm:"not null;default:0" json:"total_stock"`
	TotalSales      int       `gorm:"not null;default:0" json:"total_sales"`
	ViewCount       int       `gorm:"not null;default:0" json:"view_count"`
	IsHot           bool      `gorm:"not null;default:false" json:"is_hot"`
	IsNew           bool      `gorm:"not null;default:false" json:"is_new"`
	IsRecommend     bool      `gorm:"not null;default:false" json:"is_recommend"`
	Sort            int       `gorm:"not null;default:0" json:"sort"`
	Status          int8      `gorm:"type:smallint;not null;default:1" json:"status"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Category *Category    `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Skus     []ProductSku `gorm:"foreignKey:ProductID" json:"skus,omitempty"`
	Reviews  []Review     `gorm:"foreignKey:ProductID" json:"reviews,omitempty"`
}

// TableName 表名
func (Product) TableName() string {
	return "products"
}

// ProductStatus 商品状态
const (
	ProductStatusDraft    = 0 // 草稿
	ProductStatusOnSale   = 1 // 上架
	ProductStatusOffSale  = 2 // 下架
)

// ProductSku SKU 模型
type ProductSku struct {
	ID            int64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID     int64    `gorm:"index;not null" json:"product_id"`
	SkuCode       string   `gorm:"type:varchar(64);uniqueIndex;not null" json:"sku_code"`
	Name          string   `gorm:"type:varchar(100);not null" json:"name"`
	Image         *string  `gorm:"type:varchar(255)" json:"image,omitempty"`
	Specs         JSON     `gorm:"type:jsonb" json:"specs,omitempty"`
	Price         float64  `gorm:"type:decimal(10,2);not null" json:"price"`
	OriginalPrice *float64 `gorm:"type:decimal(10,2)" json:"original_price,omitempty"`
	CostPrice     *float64 `gorm:"type:decimal(10,2)" json:"cost_price,omitempty"`
	Stock         int      `gorm:"not null;default:0" json:"stock"`
	Sales         int      `gorm:"not null;default:0" json:"sales"`
	Weight        *float64 `gorm:"type:decimal(10,3)" json:"weight,omitempty"`
	Status        int8     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// TableName 表名
func (ProductSku) TableName() string {
	return "product_skus"
}

// SkuStatus SKU状态
const (
	SkuStatusDisabled = 0 // 禁用
	SkuStatusActive   = 1 // 启用
)

// CartItem 购物车项
type CartItem struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"index;not null" json:"user_id"`
	ProductID int64     `gorm:"not null" json:"product_id"`
	SkuID     int64     `gorm:"not null" json:"sku_id"`
	Quantity  int       `gorm:"not null;default:1" json:"quantity"`
	Selected  bool      `gorm:"not null;default:true" json:"selected"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	User    *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Product *Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Sku     *ProductSku `gorm:"foreignKey:SkuID" json:"sku,omitempty"`
}

// TableName 表名
func (CartItem) TableName() string {
	return "cart_items"
}

// Review 评价模型
type Review struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID    int64      `gorm:"index;not null" json:"product_id"`
	SkuID        int64      `gorm:"not null" json:"sku_id"`
	UserID       int64      `gorm:"index;not null" json:"user_id"`
	OrderID      int64      `gorm:"not null" json:"order_id"`
	Rating       int        `gorm:"not null" json:"rating"`
	Content      *string    `gorm:"type:text" json:"content,omitempty"`
	Images       JSON       `gorm:"type:jsonb" json:"images,omitempty"`
	IsAnonymous  bool       `gorm:"not null;default:false" json:"is_anonymous"`
	Reply        *string    `gorm:"type:text" json:"reply,omitempty"`
	RepliedAt    *time.Time `json:"replied_at,omitempty"`
	Status       int8       `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Product *Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Sku     *ProductSku `gorm:"foreignKey:SkuID" json:"sku,omitempty"`
	User    *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Order   *Order      `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

// TableName 表名
func (Review) TableName() string {
	return "reviews"
}

// ReviewStatus 评价状态
const (
	ReviewStatusPending  = 0 // 待审核
	ReviewStatusApproved = 1 // 已通过
	ReviewStatusRejected = 2 // 已拒绝
)
