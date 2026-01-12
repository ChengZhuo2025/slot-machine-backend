// Package admin 提供管理员相关的 HTTP Handler
package admin

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// MerchantHandler 商户管理处理器
type MerchantHandler struct {
	merchantService *adminService.MerchantAdminService
}

// NewMerchantHandler 创建商户管理处理器
func NewMerchantHandler(merchantSvc *adminService.MerchantAdminService) *MerchantHandler {
	return &MerchantHandler{
		merchantService: merchantSvc,
	}
}

// Create 创建商户
// @Summary 创建商户
// @Tags 商户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateMerchantRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Merchant}
// @Router /admin/merchants [post]
func (h *MerchantHandler) Create(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	var req adminService.CreateMerchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	merchant, err := h.merchantService.CreateMerchant(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, adminService.ErrMerchantNameExists) {
			response.BadRequest(c, "商户名称已存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, merchant)
}

// Update 更新商户
// @Summary 更新商户
// @Tags 商户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "商户ID"
// @Param request body adminService.UpdateMerchantRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/merchants/{id} [put]
func (h *MerchantHandler) Update(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "商户")
	if !ok {
		return
	}

	var req adminService.UpdateMerchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	err = h.merchantService.UpdateMerchant(c.Request.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrMerchantNotFound):
			response.NotFound(c, "商户不存在")
		case errors.Is(err, adminService.ErrMerchantNameExists):
			response.BadRequest(c, "商户名称已存在")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// MerchantUpdateStatusRequest 更新状态请求
type MerchantUpdateStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// UpdateStatus 更新商户状态
// @Summary 更新商户状态
// @Tags 商户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "商户ID"
// @Param request body MerchantUpdateStatusRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/merchants/{id}/status [put]
func (h *MerchantHandler) UpdateStatus(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "商户")
	if !ok {
		return
	}

	var req MerchantUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	err = h.merchantService.UpdateMerchantStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, adminService.ErrMerchantNotFound) {
			response.NotFound(c, "商户不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除商户
// @Summary 删除商户
// @Tags 商户管理
// @Produce json
// @Security Bearer
// @Param id path int true "商户ID"
// @Success 200 {object} response.Response
// @Router /admin/merchants/{id} [delete]
func (h *MerchantHandler) Delete(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "商户")
	if !ok {
		return
	}

	err = h.merchantService.DeleteMerchant(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrMerchantNotFound):
			response.NotFound(c, "商户不存在")
		case errors.Is(err, adminService.ErrMerchantHasVenues):
			response.BadRequest(c, "商户下有场地，无法删除")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// Get 获取商户详情
// @Summary 获取商户详情
// @Tags 商户管理
// @Produce json
// @Security Bearer
// @Param id path int true "商户ID"
// @Success 200 {object} response.Response{data=adminService.MerchantInfo}
// @Router /admin/merchants/{id} [get]
func (h *MerchantHandler) Get(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "商户")
	if !ok {
		return
	}

	merchant, err := h.merchantService.GetMerchant(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, adminService.ErrMerchantNotFound) {
			response.NotFound(c, "商户不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, merchant)
}

// List 获取商户列表
// @Summary 获取商户列表
// @Tags 商户管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param name query string false "商户名称"
// @Param contact_name query string false "联系人"
// @Param contact_phone query string false "联系电话"
// @Param status query int false "状态"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /admin/merchants [get]
func (h *MerchantHandler) List(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	filters := make(map[string]interface{})
	if name := c.Query("name"); name != "" {
		filters["name"] = name
	}
	if contactName := c.Query("contact_name"); contactName != "" {
		filters["contact_name"] = contactName
	}
	if contactPhone := c.Query("contact_phone"); contactPhone != "" {
		filters["contact_phone"] = contactPhone
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			filters["status"] = int8(status)
		}
	}

	merchants, total, err := h.merchantService.ListMerchants(c.Request.Context(), p.GetOffset(), p.GetLimit(), filters)
	handler.MustSucceedPage(c, err, merchants, total, p.Page, p.PageSize)
}

// ListAll 获取所有商户（下拉选择用）
// @Summary 获取所有商户
// @Tags 商户管理
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]models.Merchant}
// @Router /admin/merchants/all [get]
func (h *MerchantHandler) ListAll(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	merchants, err := h.merchantService.ListAllMerchants(c.Request.Context())
	handler.MustSucceed(c, err, merchants)
}

// RegisterRoutes 注册路由
func (h *MerchantHandler) RegisterRoutes(r *gin.RouterGroup) {
	merchants := r.Group("/merchants")
	{
		merchants.POST("", h.Create)
		merchants.GET("", h.List)
		merchants.GET("/all", h.ListAll)
		merchants.GET("/:id", h.Get)
		merchants.PUT("/:id", h.Update)
		merchants.PUT("/:id/status", h.UpdateStatus)
		merchants.DELETE("/:id", h.Delete)
	}
}
