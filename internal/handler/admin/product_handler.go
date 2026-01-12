// Package admin 提供管理后台的 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// ProductHandler 商品管理处理器
type ProductHandler struct {
	productAdminService *adminService.ProductAdminService
}

// NewProductHandler 创建商品管理处理器
func NewProductHandler(productAdminSvc *adminService.ProductAdminService) *ProductHandler {
	return &ProductHandler{
		productAdminService: productAdminSvc,
	}
}

// GetCategories 获取分类列表
// @Summary 获取分类列表
// @Tags 商品管理
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]adminService.CategoryAdminInfo}
// @Router /api/v1/admin/categories [get]
func (h *ProductHandler) GetCategories(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	categories, err := h.productAdminService.GetAllCategories(c.Request.Context())
	handler.MustSucceed(c, err, categories)
}

// CreateCategory 创建分类
// @Summary 创建分类
// @Tags 商品管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateCategoryRequest true "请求参数"
// @Success 200 {object} response.Response{data=adminService.CategoryAdminInfo}
// @Router /api/v1/admin/categories [post]
func (h *ProductHandler) CreateCategory(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	var req adminService.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	category, err := h.productAdminService.CreateCategory(c.Request.Context(), &req)
	handler.MustSucceed(c, err, category)
}

// UpdateCategory 更新分类
// @Summary 更新分类
// @Tags 商品管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "分类ID"
// @Param request body adminService.UpdateCategoryRequest true "请求参数"
// @Success 200 {object} response.Response{data=adminService.CategoryAdminInfo}
// @Router /api/v1/admin/categories/{id} [put]
func (h *ProductHandler) UpdateCategory(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "分类")
	if !ok {
		return
	}

	var req adminService.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	category, err := h.productAdminService.UpdateCategory(c.Request.Context(), id, &req)
	handler.MustSucceed(c, err, category)
}

// DeleteCategory 删除分类
// @Summary 删除分类
// @Tags 商品管理
// @Produce json
// @Security Bearer
// @Param id path int true "分类ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/categories/{id} [delete]
func (h *ProductHandler) DeleteCategory(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "分类")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.productAdminService.DeleteCategory(c.Request.Context(), id), nil)
}

// GetProducts 获取商品列表
// @Summary 获取商品列表
// @Tags 商品管理
// @Produce json
// @Security Bearer
// @Param category_id query int false "分类ID"
// @Param keyword query string false "关键词"
// @Param is_on_sale query bool false "是否上架"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=[]adminService.ProductAdminInfo}
// @Router /api/v1/admin/products [get]
func (h *ProductHandler) GetProducts(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	var params adminService.ProductListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if params.Page == 0 {
		params.Page = p.Page
	}
	if params.PageSize == 0 {
		params.PageSize = p.PageSize
	}

	products, total, err := h.productAdminService.GetProducts(c.Request.Context(), &params)
	handler.MustSucceedPage(c, err, products, total, params.Page, params.PageSize)
}

// GetProductDetail 获取商品详情
// @Summary 获取商品详情
// @Tags 商品管理
// @Produce json
// @Security Bearer
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response{data=adminService.ProductAdminInfo}
// @Router /api/v1/admin/products/{id} [get]
func (h *ProductHandler) GetProductDetail(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "商品")
	if !ok {
		return
	}

	product, err := h.productAdminService.GetProductDetail(c.Request.Context(), id)
	handler.MustSucceed(c, err, product)
}

// CreateProduct 创建商品
// @Summary 创建商品
// @Tags 商品管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateProductRequest true "请求参数"
// @Success 200 {object} response.Response{data=adminService.ProductAdminInfo}
// @Router /api/v1/admin/products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	var req adminService.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	product, err := h.productAdminService.CreateProduct(c.Request.Context(), &req)
	handler.MustSucceed(c, err, product)
}

// UpdateProduct 更新商品
// @Summary 更新商品
// @Tags 商品管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "商品ID"
// @Param request body adminService.UpdateProductRequest true "请求参数"
// @Success 200 {object} response.Response{data=adminService.ProductAdminInfo}
// @Router /api/v1/admin/products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "商品")
	if !ok {
		return
	}

	var req adminService.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	product, err := h.productAdminService.UpdateProduct(c.Request.Context(), id, &req)
	handler.MustSucceed(c, err, product)
}

// DeleteProduct 删除商品
// @Summary 删除商品
// @Tags 商品管理
// @Produce json
// @Security Bearer
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "商品")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.productAdminService.DeleteProduct(c.Request.Context(), id), nil)
}

// UpdateProductStatus 更新商品上架状态
// @Summary 更新商品上架状态
// @Tags 商品管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "商品ID"
// @Param is_on_sale query bool true "是否上架"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/products/{id}/status [put]
func (h *ProductHandler) UpdateProductStatus(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "商品")
	if !ok {
		return
	}

	isOnSale := c.Query("is_on_sale") == "true"

	handler.MustSucceed(c, h.productAdminService.UpdateProductStatus(c.Request.Context(), id, isOnSale), nil)
}
