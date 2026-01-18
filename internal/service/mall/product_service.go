// Package mall 提供商城服务
package mall

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// ProductService 商品服务
type ProductService struct {
	db           *gorm.DB
	productRepo  *repository.ProductRepository
	categoryRepo *repository.CategoryRepository
	skuRepo      *repository.ProductSkuRepository
}

// NewProductService 创建商品服务
func NewProductService(
	db *gorm.DB,
	productRepo *repository.ProductRepository,
	categoryRepo *repository.CategoryRepository,
	skuRepo *repository.ProductSkuRepository,
) *ProductService {
	return &ProductService{
		db:           db,
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		skuRepo:      skuRepo,
	}
}

// ProductInfo 商品信息
type ProductInfo struct {
	ID            int64      `json:"id"`
	CategoryID    int64      `json:"category_id"`
	CategoryName  string     `json:"category_name,omitempty"`
	Name          string     `json:"name"`
	Subtitle      string     `json:"subtitle,omitempty"`
	Images        []string   `json:"images"`
	Description   string     `json:"description,omitempty"`
	Price         float64    `json:"price"`
	OriginalPrice float64    `json:"original_price,omitempty"`
	Stock         int        `json:"stock"`
	Sales         int        `json:"sales"`
	Unit          string     `json:"unit"`
	IsOnSale      bool       `json:"is_on_sale"`
	IsHot         bool       `json:"is_hot"`
	IsNew         bool       `json:"is_new"`
	Skus          []*SkuInfo `json:"skus,omitempty"`
}

// SkuInfo SKU 信息
type SkuInfo struct {
	ID         int64             `json:"id"`
	SkuCode    string            `json:"sku_code"`
	Attributes map[string]string `json:"attributes"`
	Price      float64           `json:"price"`
	Stock      int               `json:"stock"`
	Image      string            `json:"image,omitempty"`
}

// CategoryInfo 分类信息
type CategoryInfo struct {
	ID       int64           `json:"id"`
	ParentID *int64          `json:"parent_id,omitempty"`
	Name     string          `json:"name"`
	Icon     string          `json:"icon,omitempty"`
	Level    int             `json:"level"`
	Children []*CategoryInfo `json:"children,omitempty"`
}

// ProductListRequest 商品列表请求
type ProductListRequest struct {
	Page       int     `form:"page" binding:"min=1"`
	PageSize   int     `form:"page_size" binding:"min=1,max=100"`
	CategoryID int64   `form:"category_id"`
	Keyword    string  `form:"keyword"`
	IsHot      *bool   `form:"is_hot"`
	IsNew      *bool   `form:"is_new"`
	MinPrice   float64 `form:"min_price"`
	MaxPrice   float64 `form:"max_price"`
	SortBy     string  `form:"sort_by"` // price_asc, price_desc, sales_desc, newest
}

// ProductListResponse 商品列表响应
type ProductListResponse struct {
	List       []*ProductInfo `json:"list"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// GetCategoryTree 获取分类树
func (s *ProductService) GetCategoryTree(ctx context.Context) ([]*CategoryInfo, error) {
	categories, err := s.categoryRepo.GetCategoryTree(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.buildCategoryTree(categories), nil
}

// buildCategoryTree 构建分类树
func (s *ProductService) buildCategoryTree(categories []*models.Category) []*CategoryInfo {
	var result []*CategoryInfo
	for _, cat := range categories {
		info := &CategoryInfo{
			ID:       cat.ID,
			ParentID: cat.ParentID,
			Name:     cat.Name,
			Level:    int(cat.Level),
		}
		if cat.Icon != nil {
			info.Icon = *cat.Icon
		}
		if len(cat.Children) > 0 {
			childCategories := make([]*models.Category, len(cat.Children))
			for i := range cat.Children {
				childCategories[i] = &cat.Children[i]
			}
			info.Children = s.buildCategoryTree(childCategories)
		}
		result = append(result, info)
	}
	return result
}

// GetCategoryList 获取分类列表
func (s *ProductService) GetCategoryList(ctx context.Context, parentID *int64) ([]*CategoryInfo, error) {
	filters := map[string]interface{}{
		"is_active": true,
	}
	if parentID != nil {
		filters["parent_id"] = *parentID
	} else {
		filters["parent_id"] = nil
	}

	categories, err := s.categoryRepo.List(ctx, filters)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var result []*CategoryInfo
	for _, cat := range categories {
		info := &CategoryInfo{
			ID:       cat.ID,
			ParentID: cat.ParentID,
			Name:     cat.Name,
			Level:    int(cat.Level),
		}
		if cat.Icon != nil {
			info.Icon = *cat.Icon
		}
		result = append(result, info)
	}
	return result, nil
}

// GetProductList 获取商品列表
func (s *ProductService) GetProductList(ctx context.Context, req *ProductListRequest) (*ProductListResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize
	isOnSale := true

	params := repository.ProductListParams{
		Offset:     offset,
		Limit:      req.PageSize,
		CategoryID: req.CategoryID,
		Keyword:    req.Keyword,
		IsOnSale:   &isOnSale,
		IsHot:      req.IsHot,
		IsNew:      req.IsNew,
		SortBy:     req.SortBy,
	}

	if req.MinPrice > 0 {
		params.MinPrice = &req.MinPrice
	}
	if req.MaxPrice > 0 {
		params.MaxPrice = &req.MaxPrice
	}

	products, total, err := s.productRepo.List(ctx, params)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*ProductInfo, len(products))
	for i, p := range products {
		list[i] = s.toProductInfo(p)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &ProductListResponse{
		List:       list,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetProductDetail 获取商品详情
func (s *ProductService) GetProductDetail(ctx context.Context, productID int64) (*ProductInfo, error) {
	product, err := s.productRepo.GetByIDFull(ctx, productID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrResourceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if !product.IsOnSale {
		return nil, errors.ErrResourceNotFound
	}

	info := s.toProductInfo(product)

	// 添加分类名称
	if product.Category != nil {
		info.CategoryName = product.Category.Name
	}

	// 添加 SKU 信息
	if len(product.Skus) > 0 {
		info.Skus = make([]*SkuInfo, len(product.Skus))
		for i, sku := range product.Skus {
			info.Skus[i] = s.toSkuInfo(&sku)
		}
	}

	return info, nil
}

// GetHotProducts 获取热门商品
func (s *ProductService) GetHotProducts(ctx context.Context, limit int) ([]*ProductInfo, error) {
	if limit <= 0 {
		limit = 10
	}

	products, err := s.productRepo.ListHot(ctx, limit)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*ProductInfo, len(products))
	for i, p := range products {
		list[i] = s.toProductInfo(p)
	}
	return list, nil
}

// GetNewProducts 获取新品列表
func (s *ProductService) GetNewProducts(ctx context.Context, limit int) ([]*ProductInfo, error) {
	if limit <= 0 {
		limit = 10
	}

	products, err := s.productRepo.ListNew(ctx, limit)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*ProductInfo, len(products))
	for i, p := range products {
		list[i] = s.toProductInfo(p)
	}
	return list, nil
}

// GetSelectedProducts 获取精选商品
func (s *ProductService) GetSelectedProducts(ctx context.Context, limit int) ([]*ProductInfo, error) {
	if limit <= 0 {
		limit = 6
	}

	isOnSale := true
	isHot := true

	params := repository.ProductListParams{
		Offset:   0,
		Limit:    limit,
		IsOnSale: &isOnSale,
		IsHot:    &isHot,
		SortBy:   "sales_desc", // 按销量排序
	}

	products, _, err := s.productRepo.List(ctx, params)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var result []*ProductInfo
	for _, p := range products {
		result = append(result, s.toProductInfo(p))
	}

	return result, nil
}

// GetProductsByCategory 根据分类获取商品
func (s *ProductService) GetProductsByCategory(ctx context.Context, categoryID int64, page, pageSize int) (*ProductListResponse, error) {
	return s.GetProductList(ctx, &ProductListRequest{
		Page:       page,
		PageSize:   pageSize,
		CategoryID: categoryID,
	})
}

// GetSkusByProductID 获取商品的 SKU 列表
func (s *ProductService) GetSkusByProductID(ctx context.Context, productID int64) ([]*SkuInfo, error) {
	skus, err := s.skuRepo.ListByProductID(ctx, productID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*SkuInfo, len(skus))
	for i, sku := range skus {
		list[i] = s.toSkuInfo(sku)
	}
	return list, nil
}

// GetSkuByID 根据 ID 获取 SKU
func (s *ProductService) GetSkuByID(ctx context.Context, skuID int64) (*SkuInfo, error) {
	sku, err := s.skuRepo.GetByID(ctx, skuID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrResourceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if !sku.IsActive {
		return nil, errors.ErrResourceNotFound
	}

	return s.toSkuInfo(sku), nil
}

// CheckStock 检查库存
func (s *ProductService) CheckStock(ctx context.Context, productID int64, skuID *int64, quantity int) error {
	if skuID != nil && *skuID > 0 {
		sku, err := s.skuRepo.GetByID(ctx, *skuID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.ErrResourceNotFound
			}
			return errors.ErrDatabaseError.WithError(err)
		}
		if sku.Stock < quantity {
			return errors.ErrStockInsufficient
		}
	} else {
		product, err := s.productRepo.GetByID(ctx, productID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.ErrResourceNotFound
			}
			return errors.ErrDatabaseError.WithError(err)
		}
		if product.Stock < quantity {
			return errors.ErrStockInsufficient
		}
	}
	return nil
}

// DeductStock 扣减库存
func (s *ProductService) DeductStock(ctx context.Context, productID int64, skuID *int64, quantity int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 扣减商品总库存
		if err := s.productRepo.DecreaseStock(ctx, productID, quantity); err != nil {
			return err
		}

		// 如果有 SKU，同时扣减 SKU 库存
		if skuID != nil && *skuID > 0 {
			if err := s.skuRepo.DecreaseStock(ctx, *skuID, quantity); err != nil {
				return err
			}
		}

		return nil
	})
}

// RestoreStock 恢复库存
func (s *ProductService) RestoreStock(ctx context.Context, productID int64, skuID *int64, quantity int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 恢复商品总库存
		if err := s.productRepo.IncreaseStock(ctx, productID, quantity); err != nil {
			return err
		}

		// 如果有 SKU，同时恢复 SKU 库存
		if skuID != nil && *skuID > 0 {
			if err := s.skuRepo.IncreaseStock(ctx, *skuID, quantity); err != nil {
				return err
			}
		}

		return nil
	})
}

// IncreaseSales 增加销量
func (s *ProductService) IncreaseSales(ctx context.Context, productID int64, quantity int) error {
	return s.productRepo.IncreaseSales(ctx, productID, quantity)
}

// toProductInfo 转换为商品信息
func (s *ProductService) toProductInfo(p *models.Product) *ProductInfo {
	info := &ProductInfo{
		ID:         p.ID,
		CategoryID: p.CategoryID,
		Name:       p.Name,
		Price:      p.Price,
		Stock:      p.Stock,
		Sales:      p.Sales,
		Unit:       p.Unit,
		IsOnSale:   p.IsOnSale,
		IsHot:      p.IsHot,
		IsNew:      p.IsNew,
	}

	if p.Subtitle != nil {
		info.Subtitle = *p.Subtitle
	}
	if p.Description != nil {
		info.Description = *p.Description
	}
	if p.OriginalPrice != nil {
		info.OriginalPrice = *p.OriginalPrice
	}

	// 解析图片 JSON
	if p.Images != nil {
		_ = json.Unmarshal(p.Images, &info.Images)
	}

	return info
}

// toSkuInfo 转换为 SKU 信息
func (s *ProductService) toSkuInfo(sku *models.ProductSku) *SkuInfo {
	info := &SkuInfo{
		ID:      sku.ID,
		SkuCode: sku.SkuCode,
		Price:   sku.Price,
		Stock:   sku.Stock,
	}

	if sku.Image != nil {
		info.Image = *sku.Image
	}

	// 解析属性 JSON
	if sku.Attributes != nil {
		_ = json.Unmarshal(sku.Attributes, &info.Attributes)
	}

	return info
}
