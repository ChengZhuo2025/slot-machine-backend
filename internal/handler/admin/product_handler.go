// Package admin 提供管理后台的 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
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
	categories, err := h.productAdminService.GetAllCategories(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, categories)
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
	var req adminService.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	category, err := h.productAdminService.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, category)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的分类ID")
		return
	}

	var req adminService.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	category, err := h.productAdminService.UpdateCategory(c.Request.Context(), id, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, category)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的分类ID")
		return
	}

	if err := h.productAdminService.DeleteCategory(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
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
	var params adminService.ProductListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	products, total, err := h.productAdminService.GetProducts(c.Request.Context(), &params)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      products,
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
	})
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的商品ID")
		return
	}

	product, err := h.productAdminService.GetProductDetail(c.Request.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, product)
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
	var req adminService.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	product, err := h.productAdminService.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, product)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的商品ID")
		return
	}

	var req adminService.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	product, err := h.productAdminService.UpdateProduct(c.Request.Context(), id, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, product)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的商品ID")
		return
	}

	if err := h.productAdminService.DeleteProduct(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的商品ID")
		return
	}

	isOnSale := c.Query("is_on_sale") == "true"

	if err := h.productAdminService.UpdateProductStatus(c.Request.Context(), id, isOnSale); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
