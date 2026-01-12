// Package mall 提供商城相关的 HTTP Handler
package mall

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	mallService "github.com/dumeirei/smart-locker-backend/internal/service/mall"
)

// ProductHandler 商品处理器
type ProductHandler struct {
	productService *mallService.ProductService
	searchService  *mallService.SearchService
}

// NewProductHandler 创建商品处理器
func NewProductHandler(productSvc *mallService.ProductService, searchSvc *mallService.SearchService) *ProductHandler {
	return &ProductHandler{
		productService: productSvc,
		searchService:  searchSvc,
	}
}

// GetCategories 获取分类列表
// @Summary 获取分类列表
// @Tags 商品
// @Produce json
// @Success 200 {object} response.Response{data=[]mall.CategoryInfo}
// @Router /api/v1/categories [get]
func (h *ProductHandler) GetCategories(c *gin.Context) {
	categories, err := h.productService.GetCategoryTree(c.Request.Context())
	handler.MustSucceed(c, err, categories)
}

// GetProducts 获取商品列表
// @Summary 获取商品列表
// @Tags 商品
// @Produce json
// @Param category_id query int false "分类ID"
// @Param sort_by query string false "排序方式：price_asc, price_desc, sales_desc, newest"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=mall.ProductListResponse}
// @Router /api/v1/products [get]
func (h *ProductHandler) GetProducts(c *gin.Context) {
	var req mallService.ProductListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.productService.GetProductList(c.Request.Context(), &req)
	handler.MustSucceed(c, err, result)
}

// GetProductDetail 获取商品详情
// @Summary 获取商品详情
// @Tags 商品
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response{data=mall.ProductInfo}
// @Router /api/v1/products/{id} [get]
func (h *ProductHandler) GetProductDetail(c *gin.Context) {
	productID, ok := handler.ParseID(c, "商品")
	if !ok {
		return
	}

	product, err := h.productService.GetProductDetail(c.Request.Context(), productID)
	handler.MustSucceed(c, err, product)
}

// SearchProducts 搜索商品
// @Summary 搜索商品
// @Tags 商品
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Param category_id query int false "分类ID"
// @Param min_price query number false "最低价格"
// @Param max_price query number false "最高价格"
// @Param sort_by query string false "排序方式"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=mall.SearchResult}
// @Router /api/v1/products/search [get]
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	var req mallService.SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.searchService.Search(c.Request.Context(), &req)
	handler.MustSucceed(c, err, result)
}

// GetHotKeywords 获取热门搜索关键词
// @Summary 获取热门搜索关键词
// @Tags 商品
// @Produce json
// @Success 200 {object} response.Response{data=[]string}
// @Router /api/v1/search/hot-keywords [get]
func (h *ProductHandler) GetHotKeywords(c *gin.Context) {
	keywords, err := h.searchService.GetHotKeywords(c.Request.Context(), 10)
	handler.MustSucceed(c, err, keywords)
}

// GetSearchSuggestions 获取搜索建议
// @Summary 获取搜索建议
// @Tags 商品
// @Produce json
// @Param prefix query string true "搜索前缀"
// @Success 200 {object} response.Response{data=[]mall.SearchSuggestion}
// @Router /api/v1/search/suggestions [get]
func (h *ProductHandler) GetSearchSuggestions(c *gin.Context) {
	prefix := c.Query("prefix")
	if prefix == "" {
		response.BadRequest(c, "请输入搜索关键词")
		return
	}

	suggestions, err := h.searchService.GetSuggestions(c.Request.Context(), prefix, 10)
	handler.MustSucceed(c, err, suggestions)
}
