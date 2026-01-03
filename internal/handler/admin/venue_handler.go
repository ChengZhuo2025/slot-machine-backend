// Package admin 提供管理员相关的 HTTP Handler
package admin

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// VenueHandler 场地管理处理器
type VenueHandler struct {
	venueService *adminService.VenueAdminService
}

// NewVenueHandler 创建场地管理处理器
func NewVenueHandler(venueSvc *adminService.VenueAdminService) *VenueHandler {
	return &VenueHandler{
		venueService: venueSvc,
	}
}

// Create 创建场地
// @Summary 创建场地
// @Tags 场地管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateVenueRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Venue}
// @Router /admin/venues [post]
func (h *VenueHandler) Create(c *gin.Context) {
	var req adminService.CreateVenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	venue, err := h.venueService.CreateVenue(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, adminService.ErrMerchantNotFound) {
			response.BadRequest(c, "商户不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, venue)
}

// Update 更新场地
// @Summary 更新场地
// @Tags 场地管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "场地ID"
// @Param request body adminService.UpdateVenueRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/venues/{id} [put]
func (h *VenueHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的场地ID")
		return
	}

	var req adminService.UpdateVenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	err = h.venueService.UpdateVenue(c.Request.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrVenueNotFound):
			response.NotFound(c, "场地不存在")
		case errors.Is(err, adminService.ErrMerchantNotFound):
			response.BadRequest(c, "商户不存在")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// UpdateStatusRequest 更新状态请求
type VenueUpdateStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// UpdateStatus 更新场地状态
// @Summary 更新场地状态
// @Tags 场地管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "场地ID"
// @Param request body VenueUpdateStatusRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/venues/{id}/status [put]
func (h *VenueHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的场地ID")
		return
	}

	var req VenueUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	err = h.venueService.UpdateVenueStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, adminService.ErrVenueNotFound) {
			response.NotFound(c, "场地不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除场地
// @Summary 删除场地
// @Tags 场地管理
// @Produce json
// @Security Bearer
// @Param id path int true "场地ID"
// @Success 200 {object} response.Response
// @Router /admin/venues/{id} [delete]
func (h *VenueHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的场地ID")
		return
	}

	err = h.venueService.DeleteVenue(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrVenueNotFound):
			response.NotFound(c, "场地不存在")
		case errors.Is(err, adminService.ErrVenueHasDevices):
			response.BadRequest(c, "场地下有设备，无法删除")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// Get 获取场地详情
// @Summary 获取场地详情
// @Tags 场地管理
// @Produce json
// @Security Bearer
// @Param id path int true "场地ID"
// @Success 200 {object} response.Response{data=adminService.VenueInfo}
// @Router /admin/venues/{id} [get]
func (h *VenueHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的场地ID")
		return
	}

	venue, err := h.venueService.GetVenue(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, adminService.ErrVenueNotFound) {
			response.NotFound(c, "场地不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, venue)
}

// List 获取场地列表
// @Summary 获取场地列表
// @Tags 场地管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param merchant_id query int false "商户ID"
// @Param name query string false "场地名称"
// @Param type query string false "场地类型"
// @Param city query string false "城市"
// @Param status query int false "状态"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /admin/venues [get]
func (h *VenueHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filters := make(map[string]interface{})
	if merchantIDStr := c.Query("merchant_id"); merchantIDStr != "" {
		if merchantID, err := strconv.ParseInt(merchantIDStr, 10, 64); err == nil {
			filters["merchant_id"] = merchantID
		}
	}
	if name := c.Query("name"); name != "" {
		filters["name"] = name
	}
	if venueType := c.Query("type"); venueType != "" {
		filters["type"] = venueType
	}
	if city := c.Query("city"); city != "" {
		filters["city"] = city
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			filters["status"] = int8(status)
		}
	}

	venues, total, err := h.venueService.ListVenues(c.Request.Context(), offset, pageSize, filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithPage(c, venues, total, page, pageSize)
}

// ListByMerchant 获取商户下的场地列表
// @Summary 获取商户下的场地列表
// @Tags 场地管理
// @Produce json
// @Security Bearer
// @Param merchant_id path int true "商户ID"
// @Success 200 {object} response.Response{data=[]models.Venue}
// @Router /admin/merchants/{merchant_id}/venues [get]
func (h *VenueHandler) ListByMerchant(c *gin.Context) {
	merchantIDStr := c.Param("merchant_id")
	merchantID, err := strconv.ParseInt(merchantIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的商户ID")
		return
	}

	venues, err := h.venueService.ListVenuesByMerchant(c.Request.Context(), merchantID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, venues)
}

// RegisterRoutes 注册路由
func (h *VenueHandler) RegisterRoutes(r *gin.RouterGroup) {
	venues := r.Group("/venues")
	{
		venues.POST("", h.Create)
		venues.GET("", h.List)
		venues.GET("/:id", h.Get)
		venues.PUT("/:id", h.Update)
		venues.PUT("/:id/status", h.UpdateStatus)
		venues.DELETE("/:id", h.Delete)
	}
}
