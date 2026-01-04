// Package admin 提供管理后台服务
package admin

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// ProductAdminService 商品管理服务
type ProductAdminService struct {
	db           *gorm.DB
	categoryRepo *repository.CategoryRepository
	productRepo  *repository.ProductRepository
	skuRepo      *repository.ProductSkuRepository
}

// NewProductAdminService 创建商品管理服务
func NewProductAdminService(
	db *gorm.DB,
	categoryRepo *repository.CategoryRepository,
	productRepo *repository.ProductRepository,
	skuRepo *repository.ProductSkuRepository,
) *ProductAdminService {
	return &ProductAdminService{
		db:           db,
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		skuRepo:      skuRepo,
	}
}

// CategoryAdminInfo 分类管理信息
type CategoryAdminInfo struct {
	ID        int64                `json:"id"`
	ParentID  *int64               `json:"parent_id,omitempty"`
	Name      string               `json:"name"`
	Icon      string               `json:"icon,omitempty"`
	Sort      int                  `json:"sort"`
	Level     int                  `json:"level"`
	IsActive  bool                 `json:"is_active"`
	CreatedAt string               `json:"created_at"`
	Children  []*CategoryAdminInfo `json:"children,omitempty"`
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	ParentID *int64 `json:"parent_id"`
	Name     string `json:"name" binding:"required"`
	Icon     string `json:"icon"`
	Sort     int    `json:"sort"`
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Sort     *int   `json:"sort"`
	IsActive *bool  `json:"is_active"`
}

// CreateCategory 创建分类
func (s *ProductAdminService) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*CategoryAdminInfo, error) {
	level := 1
	if req.ParentID != nil && *req.ParentID > 0 {
		parent, err := s.categoryRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.ErrResourceNotFound.WithMessage("父分类不存在")
			}
			return nil, errors.ErrDatabaseError.WithError(err)
		}
		level = int(parent.Level) + 1
	}

	var icon *string
	if req.Icon != "" {
		icon = &req.Icon
	}

	category := &models.Category{
		ParentID: req.ParentID,
		Name:     req.Name,
		Icon:     icon,
		Sort:     req.Sort,
		Level:    int16(level),
		IsActive: true,
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toCategoryAdminInfo(category), nil
}

// UpdateCategory 更新分类
func (s *ProductAdminService) UpdateCategory(ctx context.Context, id int64, req *UpdateCategoryRequest) (*CategoryAdminInfo, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrResourceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if req.Name != "" {
		category.Name = req.Name
	}
	if req.Icon != "" {
		category.Icon = &req.Icon
	}
	if req.Sort != nil {
		category.Sort = *req.Sort
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toCategoryAdminInfo(category), nil
}

// DeleteCategory 删除分类
func (s *ProductAdminService) DeleteCategory(ctx context.Context, id int64) error {
	// 检查是否有子分类
	children, err := s.categoryRepo.ListByParentID(ctx, id)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}
	if len(children) > 0 {
		return errors.ErrOperationFailed.WithMessage("该分类下有子分类，无法删除")
	}

	// 检查是否有商品
	hasProducts, err := s.categoryRepo.HasProducts(ctx, id)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}
	if hasProducts {
		return errors.ErrOperationFailed.WithMessage("该分类下有商品，无法删除")
	}

	return s.categoryRepo.Delete(ctx, id)
}

// GetAllCategories 获取所有分类（树形结构）
func (s *ProductAdminService) GetAllCategories(ctx context.Context) ([]*CategoryAdminInfo, error) {
	categories, err := s.categoryRepo.GetCategoryTree(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.buildCategoryTree(categories, nil), nil
}

// ProductAdminInfo 商品管理信息
type ProductAdminInfo struct {
	ID            int64            `json:"id"`
	CategoryID    int64            `json:"category_id"`
	CategoryName  string           `json:"category_name,omitempty"`
	Name          string           `json:"name"`
	Subtitle      string           `json:"subtitle,omitempty"`
	Images        []string         `json:"images"`
	Price         float64          `json:"price"`
	OriginalPrice float64          `json:"original_price,omitempty"`
	Stock         int              `json:"stock"`
	Sales         int              `json:"sales"`
	Unit          string           `json:"unit"`
	IsOnSale      bool             `json:"is_on_sale"`
	IsHot         bool             `json:"is_hot"`
	IsNew         bool             `json:"is_new"`
	Sort          int              `json:"sort"`
	Skus          []*SkuAdminInfo  `json:"skus,omitempty"`
	CreatedAt     string           `json:"created_at"`
}

// SkuAdminInfo SKU 管理信息
type SkuAdminInfo struct {
	ID         int64             `json:"id"`
	SkuCode    string            `json:"sku_code"`
	Attributes map[string]string `json:"attributes"`
	Price      float64           `json:"price"`
	Stock      int               `json:"stock"`
	Image      string            `json:"image,omitempty"`
	IsActive   bool              `json:"is_active"`
}

// ProductListParams 商品列表查询参数
type ProductListParams struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	CategoryID int64  `form:"category_id"`
	Keyword    string `form:"keyword"`
	IsOnSale   *bool  `form:"is_on_sale"`
}

// CreateProductRequest 创建商品请求
type CreateProductRequest struct {
	CategoryID    int64    `json:"category_id" binding:"required"`
	Name          string   `json:"name" binding:"required"`
	Subtitle      string   `json:"subtitle"`
	Images        []string `json:"images" binding:"required"`
	Description   string   `json:"description"`
	Price         float64  `json:"price" binding:"required,gt=0"`
	OriginalPrice float64  `json:"original_price"`
	Stock         int      `json:"stock"`
	Unit          string   `json:"unit"`
	Sort          int      `json:"sort"`
	IsOnSale      bool     `json:"is_on_sale"`
	IsHot         bool     `json:"is_hot"`
	IsNew         bool     `json:"is_new"`
}

// UpdateProductRequest 更新商品请求
type UpdateProductRequest struct {
	CategoryID    *int64   `json:"category_id"`
	Name          string   `json:"name"`
	Subtitle      string   `json:"subtitle"`
	Images        []string `json:"images"`
	Description   string   `json:"description"`
	Price         *float64 `json:"price"`
	OriginalPrice *float64 `json:"original_price"`
	Stock         *int     `json:"stock"`
	Unit          string   `json:"unit"`
	Sort          *int     `json:"sort"`
	IsOnSale      *bool    `json:"is_on_sale"`
	IsHot         *bool    `json:"is_hot"`
	IsNew         *bool    `json:"is_new"`
}

// GetProducts 获取商品列表
func (s *ProductAdminService) GetProducts(ctx context.Context, params *ProductListParams) ([]*ProductAdminInfo, int64, error) {
	if params.Page == 0 {
		params.Page = 1
	}
	if params.PageSize == 0 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	repoParams := repository.ProductListParams{
		Offset:     offset,
		Limit:      params.PageSize,
		CategoryID: params.CategoryID,
		Keyword:    params.Keyword,
		IsOnSale:   params.IsOnSale,
	}

	products, total, err := s.productRepo.List(ctx, repoParams)
	if err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*ProductAdminInfo, len(products))
	for i, p := range products {
		list[i] = s.toProductAdminInfo(p)
	}

	return list, total, nil
}

// GetProductDetail 获取商品详情
func (s *ProductAdminService) GetProductDetail(ctx context.Context, id int64) (*ProductAdminInfo, error) {
	product, err := s.productRepo.GetByIDWithSkus(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrProductNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toProductAdminInfo(product), nil
}

// CreateProduct 创建商品
func (s *ProductAdminService) CreateProduct(ctx context.Context, req *CreateProductRequest) (*ProductAdminInfo, error) {
	// 检查分类是否存在
	_, err := s.categoryRepo.GetByID(ctx, req.CategoryID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrResourceNotFound.WithMessage("分类不存在")
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	imagesJSON, _ := json.Marshal(req.Images)

	var subtitle, description *string
	if req.Subtitle != "" {
		subtitle = &req.Subtitle
	}
	if req.Description != "" {
		description = &req.Description
	}
	var originalPrice *float64
	if req.OriginalPrice > 0 {
		originalPrice = &req.OriginalPrice
	}

	product := &models.Product{
		CategoryID:    req.CategoryID,
		Name:          req.Name,
		Subtitle:      subtitle,
		Images:        imagesJSON,
		Description:   description,
		Price:         req.Price,
		OriginalPrice: originalPrice,
		Stock:         req.Stock,
		Unit:          req.Unit,
		Sort:          req.Sort,
		IsOnSale:      req.IsOnSale,
		IsHot:         req.IsHot,
		IsNew:         req.IsNew,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toProductAdminInfo(product), nil
}

// UpdateProduct 更新商品
func (s *ProductAdminService) UpdateProduct(ctx context.Context, id int64, req *UpdateProductRequest) (*ProductAdminInfo, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrProductNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if req.CategoryID != nil {
		product.CategoryID = *req.CategoryID
	}
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Subtitle != "" {
		product.Subtitle = &req.Subtitle
	}
	if len(req.Images) > 0 {
		imagesJSON, _ := json.Marshal(req.Images)
		product.Images = imagesJSON
	}
	if req.Description != "" {
		product.Description = &req.Description
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.OriginalPrice != nil {
		product.OriginalPrice = req.OriginalPrice
	}
	if req.Stock != nil {
		product.Stock = *req.Stock
	}
	if req.Unit != "" {
		product.Unit = req.Unit
	}
	if req.Sort != nil {
		product.Sort = *req.Sort
	}
	if req.IsOnSale != nil {
		product.IsOnSale = *req.IsOnSale
	}
	if req.IsHot != nil {
		product.IsHot = *req.IsHot
	}
	if req.IsNew != nil {
		product.IsNew = *req.IsNew
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toProductAdminInfo(product), nil
}

// DeleteProduct 删除商品
func (s *ProductAdminService) DeleteProduct(ctx context.Context, id int64) error {
	return s.productRepo.Delete(ctx, id)
}

// UpdateProductStatus 更新商品上架状态
func (s *ProductAdminService) UpdateProductStatus(ctx context.Context, id int64, isOnSale bool) error {
	return s.productRepo.UpdateFields(ctx, id, map[string]interface{}{
		"is_on_sale": isOnSale,
	})
}

// toCategoryAdminInfo 转换为分类管理信息
func (s *ProductAdminService) toCategoryAdminInfo(c *models.Category) *CategoryAdminInfo {
	info := &CategoryAdminInfo{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Name:      c.Name,
		Sort:      c.Sort,
		Level:     int(c.Level),
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if c.Icon != nil {
		info.Icon = *c.Icon
	}
	return info
}

// buildCategoryTree 构建分类树
func (s *ProductAdminService) buildCategoryTree(categories []*models.Category, parentID *int64) []*CategoryAdminInfo {
	var result []*CategoryAdminInfo

	for _, c := range categories {
		if (parentID == nil && c.ParentID == nil) || (parentID != nil && c.ParentID != nil && *c.ParentID == *parentID) {
			info := s.toCategoryAdminInfo(c)
			info.Children = s.buildCategoryTree(categories, &c.ID)
			result = append(result, info)
		}
	}

	return result
}

// toProductAdminInfo 转换为商品管理信息
func (s *ProductAdminService) toProductAdminInfo(p *models.Product) *ProductAdminInfo {
	info := &ProductAdminInfo{
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
		Sort:       p.Sort,
		CreatedAt:  p.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if p.Subtitle != nil {
		info.Subtitle = *p.Subtitle
	}
	if p.OriginalPrice != nil {
		info.OriginalPrice = *p.OriginalPrice
	}
	if p.Category != nil {
		info.CategoryName = p.Category.Name
	}

	// 解析图片
	if p.Images != nil {
		_ = json.Unmarshal(p.Images, &info.Images)
	}

	// SKU 信息
	if len(p.Skus) > 0 {
		info.Skus = make([]*SkuAdminInfo, len(p.Skus))
		for i, sku := range p.Skus {
			skuInfo := &SkuAdminInfo{
				ID:       sku.ID,
				SkuCode:  sku.SkuCode,
				Price:    sku.Price,
				Stock:    sku.Stock,
				IsActive: sku.IsActive,
			}
			if sku.Attributes != nil {
				_ = json.Unmarshal(sku.Attributes, &skuInfo.Attributes)
			}
			if sku.Image != nil {
				skuInfo.Image = *sku.Image
			}
			info.Skus[i] = skuInfo
		}
	}

	return info
}
