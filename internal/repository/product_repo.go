// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// ProductRepository 商品仓储
type ProductRepository struct {
	db *gorm.DB
}

// NewProductRepository 创建商品仓储
func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Create 创建商品
func (r *ProductRepository) Create(ctx context.Context, product *models.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

// GetByID 根据 ID 获取商品
func (r *ProductRepository) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetByIDWithCategory 根据 ID 获取商品（包含分类）
func (r *ProductRepository) GetByIDWithCategory(ctx context.Context, id int64) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).Preload("Category").First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetByIDWithSkus 根据 ID 获取商品（包含 SKU）
func (r *ProductRepository) GetByIDWithSkus(ctx context.Context, id int64) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).Preload("Skus", func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true).Order("id ASC")
	}).First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetByIDFull 根据 ID 获取商品（包含分类和 SKU）
func (r *ProductRepository) GetByIDFull(ctx context.Context, id int64) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Skus", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("id ASC")
		}).
		First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// Update 更新商品
func (r *ProductRepository) Update(ctx context.Context, product *models.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

// UpdateFields 更新指定字段
func (r *ProductRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Product{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除商品
func (r *ProductRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Product{}, id).Error
}

// ProductListParams 商品列表查询参数
type ProductListParams struct {
	Offset     int
	Limit      int
	CategoryID int64
	Keyword    string
	IsOnSale   *bool
	IsHot      *bool
	IsNew      *bool
	MinPrice   *float64
	MaxPrice   *float64
	SortBy     string // price_asc, price_desc, sales_desc, newest
}

// List 获取商品列表
func (r *ProductRepository) List(ctx context.Context, params ProductListParams) ([]*models.Product, int64, error) {
	var products []*models.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Product{})

	// 过滤条件
	if params.CategoryID > 0 {
		query = query.Where("category_id = ?", params.CategoryID)
	}
	if params.Keyword != "" {
		query = query.Where("name LIKE ? OR subtitle LIKE ?", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	if params.IsOnSale != nil {
		query = query.Where("is_on_sale = ?", *params.IsOnSale)
	}
	if params.IsHot != nil {
		query = query.Where("is_hot = ?", *params.IsHot)
	}
	if params.IsNew != nil {
		query = query.Where("is_new = ?", *params.IsNew)
	}
	if params.MinPrice != nil {
		query = query.Where("price >= ?", *params.MinPrice)
	}
	if params.MaxPrice != nil {
		query = query.Where("price <= ?", *params.MaxPrice)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	switch params.SortBy {
	case "price_asc":
		query = query.Order("price ASC")
	case "price_desc":
		query = query.Order("price DESC")
	case "sales_desc":
		query = query.Order("sales DESC")
	case "newest":
		query = query.Order("created_at DESC")
	default:
		query = query.Order("sort DESC, id DESC")
	}

	// 查询列表
	if err := query.Offset(params.Offset).Limit(params.Limit).Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// ListOnSale 获取上架商品列表
func (r *ProductRepository) ListOnSale(ctx context.Context, offset, limit int) ([]*models.Product, int64, error) {
	isOnSale := true
	return r.List(ctx, ProductListParams{
		Offset:   offset,
		Limit:    limit,
		IsOnSale: &isOnSale,
	})
}

// ListByCategory 根据分类获取商品列表
func (r *ProductRepository) ListByCategory(ctx context.Context, categoryID int64, offset, limit int) ([]*models.Product, int64, error) {
	isOnSale := true
	return r.List(ctx, ProductListParams{
		Offset:     offset,
		Limit:      limit,
		CategoryID: categoryID,
		IsOnSale:   &isOnSale,
	})
}

// ListHot 获取热门商品
func (r *ProductRepository) ListHot(ctx context.Context, limit int) ([]*models.Product, error) {
	var products []*models.Product
	err := r.db.WithContext(ctx).
		Where("is_on_sale = ? AND is_hot = ?", true, true).
		Order("sales DESC, sort DESC").
		Limit(limit).
		Find(&products).Error
	return products, err
}

// ListNew 获取新品列表
func (r *ProductRepository) ListNew(ctx context.Context, limit int) ([]*models.Product, error) {
	var products []*models.Product
	err := r.db.WithContext(ctx).
		Where("is_on_sale = ? AND is_new = ?", true, true).
		Order("created_at DESC, sort DESC").
		Limit(limit).
		Find(&products).Error
	return products, err
}

// IncreaseSales 增加销量
func (r *ProductRepository) IncreaseSales(ctx context.Context, id int64, quantity int) error {
	return r.db.WithContext(ctx).Model(&models.Product{}).
		Where("id = ?", id).
		UpdateColumn("sales", gorm.Expr("sales + ?", quantity)).
		Error
}

// DecreaseStock 减少库存
func (r *ProductRepository) DecreaseStock(ctx context.Context, id int64, quantity int) error {
	result := r.db.WithContext(ctx).Model(&models.Product{}).
		Where("id = ? AND stock >= ?", id, quantity).
		UpdateColumn("stock", gorm.Expr("stock - ?", quantity))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IncreaseStock 增加库存
func (r *ProductRepository) IncreaseStock(ctx context.Context, id int64, quantity int) error {
	return r.db.WithContext(ctx).Model(&models.Product{}).
		Where("id = ?", id).
		UpdateColumn("stock", gorm.Expr("stock + ?", quantity)).
		Error
}

// ProductSkuRepository 商品 SKU 仓储
type ProductSkuRepository struct {
	db *gorm.DB
}

// NewProductSkuRepository 创建 SKU 仓储
func NewProductSkuRepository(db *gorm.DB) *ProductSkuRepository {
	return &ProductSkuRepository{db: db}
}

// Create 创建 SKU
func (r *ProductSkuRepository) Create(ctx context.Context, sku *models.ProductSku) error {
	return r.db.WithContext(ctx).Create(sku).Error
}

// CreateBatch 批量创建 SKU
func (r *ProductSkuRepository) CreateBatch(ctx context.Context, skus []*models.ProductSku) error {
	return r.db.WithContext(ctx).CreateInBatches(skus, 100).Error
}

// GetByID 根据 ID 获取 SKU
func (r *ProductSkuRepository) GetByID(ctx context.Context, id int64) (*models.ProductSku, error) {
	var sku models.ProductSku
	err := r.db.WithContext(ctx).First(&sku, id).Error
	if err != nil {
		return nil, err
	}
	return &sku, nil
}

// GetBySkuCode 根据 SKU 编码获取
func (r *ProductSkuRepository) GetBySkuCode(ctx context.Context, skuCode string) (*models.ProductSku, error) {
	var sku models.ProductSku
	err := r.db.WithContext(ctx).Where("sku_code = ?", skuCode).First(&sku).Error
	if err != nil {
		return nil, err
	}
	return &sku, nil
}

// ListByProductID 根据商品 ID 获取 SKU 列表
func (r *ProductSkuRepository) ListByProductID(ctx context.Context, productID int64) ([]*models.ProductSku, error) {
	var skus []*models.ProductSku
	err := r.db.WithContext(ctx).
		Where("product_id = ? AND is_active = ?", productID, true).
		Order("id ASC").
		Find(&skus).Error
	return skus, err
}

// Update 更新 SKU
func (r *ProductSkuRepository) Update(ctx context.Context, sku *models.ProductSku) error {
	return r.db.WithContext(ctx).Save(sku).Error
}

// UpdateFields 更新指定字段
func (r *ProductSkuRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.ProductSku{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除 SKU
func (r *ProductSkuRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.ProductSku{}, id).Error
}

// DeleteByProductID 根据商品 ID 删除所有 SKU
func (r *ProductSkuRepository) DeleteByProductID(ctx context.Context, productID int64) error {
	return r.db.WithContext(ctx).Where("product_id = ?", productID).Delete(&models.ProductSku{}).Error
}

// DecreaseStock 减少 SKU 库存
func (r *ProductSkuRepository) DecreaseStock(ctx context.Context, id int64, quantity int) error {
	result := r.db.WithContext(ctx).Model(&models.ProductSku{}).
		Where("id = ? AND stock >= ?", id, quantity).
		UpdateColumn("stock", gorm.Expr("stock - ?", quantity))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IncreaseStock 增加 SKU 库存
func (r *ProductSkuRepository) IncreaseStock(ctx context.Context, id int64, quantity int) error {
	return r.db.WithContext(ctx).Model(&models.ProductSku{}).
		Where("id = ?", id).
		UpdateColumn("stock", gorm.Expr("stock + ?", quantity)).
		Error
}
