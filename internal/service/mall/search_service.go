// Package mall 提供商城服务
package mall

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// SearchService 商品搜索服务
type SearchService struct {
	db          *gorm.DB
	productRepo *repository.ProductRepository
}

// NewSearchService 创建搜索服务
func NewSearchService(
	db *gorm.DB,
	productRepo *repository.ProductRepository,
) *SearchService {
	return &SearchService{
		db:          db,
		productRepo: productRepo,
	}
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword    string   `form:"keyword" binding:"required"`
	CategoryID int64    `form:"category_id"`
	MinPrice   float64  `form:"min_price"`
	MaxPrice   float64  `form:"max_price"`
	SortBy     string   `form:"sort_by"` // price_asc, price_desc, sales_desc, newest
	Page       int      `form:"page" binding:"min=1"`
	PageSize   int      `form:"page_size" binding:"min=1,max=100"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Products   []*ProductInfo `json:"products"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
	Keyword    string         `json:"keyword"`
}

// SearchSuggestion 搜索建议
type SearchSuggestion struct {
	Keyword string `json:"keyword"`
	Count   int64  `json:"count"`
}

// Search 搜索商品
func (s *SearchService) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// 清理关键词
	keyword := strings.TrimSpace(req.Keyword)
	if keyword == "" {
		return nil, errors.ErrInvalidParams.WithMessage("搜索关键词不能为空")
	}

	offset := (req.Page - 1) * req.PageSize
	isOnSale := true

	params := repository.ProductListParams{
		Offset:     offset,
		Limit:      req.PageSize,
		CategoryID: req.CategoryID,
		Keyword:    keyword,
		IsOnSale:   &isOnSale,
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

	return &SearchResult{
		Products:   list,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Keyword:    keyword,
	}, nil
}

// GetHotKeywords 获取热门搜索关键词
func (s *SearchService) GetHotKeywords(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	// 实际项目中，这里可以从 Redis 或搜索引擎获取热门关键词
	// 当前返回一些默认的热门关键词
	keywords := []string{
		"情趣内衣",
		"振动棒",
		"安全套",
		"润滑剂",
		"成人用品",
	}

	if limit < len(keywords) {
		return keywords[:limit], nil
	}
	return keywords, nil
}

// GetSuggestions 获取搜索建议
func (s *SearchService) GetSuggestions(ctx context.Context, prefix string, limit int) ([]*SearchSuggestion, error) {
	if limit <= 0 {
		limit = 10
	}

	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, nil
	}

	// 从商品名称中查找匹配的关键词
	var products []*models.Product
	err := s.db.WithContext(ctx).
		Model(&models.Product{}).
		Select("name").
		Where("is_on_sale = ? AND name LIKE ?", true, prefix+"%").
		Limit(limit).
		Find(&products).Error
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	suggestions := make([]*SearchSuggestion, len(products))
	for i, p := range products {
		suggestions[i] = &SearchSuggestion{
			Keyword: p.Name,
			Count:   0, // 可以后续统计搜索次数
		}
	}

	return suggestions, nil
}

// toProductInfo 转换为商品信息
func (s *SearchService) toProductInfo(p *models.Product) *ProductInfo {
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
	if p.OriginalPrice != nil {
		info.OriginalPrice = *p.OriginalPrice
	}

	return info
}
