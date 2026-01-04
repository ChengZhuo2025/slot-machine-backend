package models

import (
	"encoding/json"
	"time"
)

// Category 商品分类
type Category struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ParentID  *int64    `gorm:"column:parent_id;index" json:"parent_id,omitempty"`
	Name      string    `gorm:"column:name;type:varchar(50);not null" json:"name"`
	Icon      *string   `gorm:"column:icon;type:varchar(255)" json:"icon,omitempty"`
	Sort      int       `gorm:"column:sort;not null;default:0" json:"sort"`
	Level     int16     `gorm:"column:level;type:smallint;not null;default:1" json:"level"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// 关联
	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

// TableName 表名
func (Category) TableName() string {
	return "categories"
}

// Product 商品模型
type Product struct {
	ID            int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CategoryID    int64            `gorm:"column:category_id;index;not null" json:"category_id"`
	Name          string           `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Subtitle      *string          `gorm:"column:subtitle;type:varchar(255)" json:"subtitle,omitempty"`
	Images        json.RawMessage  `gorm:"column:images;type:jsonb;not null" json:"images"`
	Description   *string          `gorm:"column:description;type:text" json:"description,omitempty"`
	Price         float64          `gorm:"column:price;type:decimal(10,2);not null" json:"price"`
	OriginalPrice *float64         `gorm:"column:original_price;type:decimal(10,2)" json:"original_price,omitempty"`
	Stock         int              `gorm:"column:stock;not null;default:0" json:"stock"`
	Sales         int              `gorm:"column:sales;not null;default:0" json:"sales"`
	Unit          string           `gorm:"column:unit;type:varchar(20);not null;default:'件'" json:"unit"`
	IsOnSale      bool             `gorm:"column:is_on_sale;not null;default:true" json:"is_on_sale"`
	IsHot         bool             `gorm:"column:is_hot;not null;default:false" json:"is_hot"`
	IsNew         bool             `gorm:"column:is_new;not null;default:false" json:"is_new"`
	Sort          int              `gorm:"column:sort;not null;default:0" json:"sort"`
	CreatedAt     time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// 关联
	Category *Category    `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Skus     []ProductSku `gorm:"foreignKey:ProductID" json:"skus,omitempty"`
	Reviews  []Review     `gorm:"foreignKey:ProductID" json:"reviews,omitempty"`
}

// TableName 表名
func (Product) TableName() string {
	return "products"
}

// ProductSku SKU 模型
type ProductSku struct {
	ID         int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID  int64           `gorm:"column:product_id;index;not null" json:"product_id"`
	SkuCode    string          `gorm:"column:sku_code;type:varchar(64);uniqueIndex;not null" json:"sku_code"`
	Attributes json.RawMessage `gorm:"column:attributes;type:jsonb;not null" json:"attributes"`
	Price      float64         `gorm:"column:price;type:decimal(10,2);not null" json:"price"`
	Stock      int             `gorm:"column:stock;not null;default:0" json:"stock"`
	Image      *string         `gorm:"column:image;type:varchar(255)" json:"image,omitempty"`
	IsActive   bool            `gorm:"column:is_active;not null;default:true" json:"is_active"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// 关联
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// TableName 表名
func (ProductSku) TableName() string {
	return "product_skus"
}

// CartItem 购物车项
type CartItem struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"column:user_id;index;not null" json:"user_id"`
	ProductID int64     `gorm:"column:product_id;not null" json:"product_id"`
	SkuID     *int64    `gorm:"column:sku_id" json:"sku_id,omitempty"`
	Quantity  int       `gorm:"column:quantity;not null" json:"quantity"`
	Selected  bool      `gorm:"column:selected;not null;default:true" json:"selected"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

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
	ID          int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID     int64           `gorm:"column:order_id;index;not null" json:"order_id"`
	ProductID   int64           `gorm:"column:product_id;index;not null" json:"product_id"`
	UserID      int64           `gorm:"column:user_id;index;not null" json:"user_id"`
	Rating      int16           `gorm:"column:rating;type:smallint;not null" json:"rating"`
	Content     *string         `gorm:"column:content;type:text" json:"content,omitempty"`
	Images      json.RawMessage `gorm:"column:images;type:jsonb" json:"images,omitempty"`
	IsAnonymous bool            `gorm:"column:is_anonymous;not null;default:false" json:"is_anonymous"`
	Reply       *string         `gorm:"column:reply;type:text" json:"reply,omitempty"`
	RepliedAt   *time.Time      `gorm:"column:replied_at" json:"replied_at,omitempty"`
	Status      int16           `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// 关联
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Order   *Order   `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

// TableName 表名
func (Review) TableName() string {
	return "reviews"
}

// ReviewStatus 评价状态
const (
	ReviewStatusHidden  = 0 // 隐藏
	ReviewStatusVisible = 1 // 显示
)
