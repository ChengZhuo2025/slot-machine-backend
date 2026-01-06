// Package admin 提供管理端 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// MarketingHandler 营销管理处理器
type MarketingHandler struct {
	marketingService *adminService.MarketingAdminService
}

// NewMarketingHandler 创建营销管理处理器
func NewMarketingHandler(marketingSvc *adminService.MarketingAdminService) *MarketingHandler {
	return &MarketingHandler{
		marketingService: marketingSvc,
	}
}

// GetCouponList 获取优惠券列表
// @Summary 获取优惠券列表
// @Tags 管理端-营销管理
// @Produce json
// @Security Bearer
// @Param status query int false "状态：0-禁用 1-启用"
// @Param type query string false "类型：fixed/percent"
// @Param applicable_type query string false "适用范围：all/category/product"
// @Param keyword query string false "关键词"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=admin.AdminCouponListResponse}
// @Router /api/v1/admin/marketing/coupons [get]
func (h *MarketingHandler) GetCouponList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	req := &adminService.AdminCouponListRequest{
		Page:           page,
		PageSize:       pageSize,
		Type:           c.Query("type"),
		ApplicableType: c.Query("applicable_type"),
		Keyword:        c.Query("keyword"),
	}

	// 处理状态筛选
	if statusStr := c.Query("status"); statusStr != "" {
		status, err := strconv.ParseInt(statusStr, 10, 8)
		if err == nil {
			s := int8(status)
			req.Status = &s
		}
	}

	result, err := h.marketingService.GetCouponList(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, result.List, result.Total, page, pageSize)
}

// GetCouponDetail 获取优惠券详情
// @Summary 获取优惠券详情
// @Tags 管理端-营销管理
// @Produce json
// @Security Bearer
// @Param id path int true "优惠券ID"
// @Success 200 {object} response.Response{data=admin.AdminCouponItem}
// @Router /api/v1/admin/marketing/coupons/{id} [get]
func (h *MarketingHandler) GetCouponDetail(c *gin.Context) {
	couponID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的优惠券ID")
		return
	}

	coupon, err := h.marketingService.GetCouponDetail(c.Request.Context(), couponID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, coupon)
}

// CreateCoupon 创建优惠券
// @Summary 创建优惠券
// @Tags 管理端-营销管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body admin.CreateCouponRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Coupon}
// @Router /api/v1/admin/marketing/coupons [post]
func (h *MarketingHandler) CreateCoupon(c *gin.Context) {
	var req adminService.CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	coupon, err := h.marketingService.CreateCoupon(c.Request.Context(), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "创建成功", coupon)
}

// UpdateCoupon 更新优惠券
// @Summary 更新优惠券
// @Tags 管理端-营销管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "优惠券ID"
// @Param request body admin.UpdateCouponRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/marketing/coupons/{id} [put]
func (h *MarketingHandler) UpdateCoupon(c *gin.Context) {
	couponID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的优惠券ID")
		return
	}

	var req adminService.UpdateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.marketingService.UpdateCoupon(c.Request.Context(), couponID, &req); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// UpdateCouponStatusRequest 更新优惠券状态请求
type UpdateCouponStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// UpdateCouponStatus 更新优惠券状态
// @Summary 更新优惠券状态
// @Tags 管理端-营销管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "优惠券ID"
// @Param request body UpdateCouponStatusRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/marketing/coupons/{id}/status [put]
func (h *MarketingHandler) UpdateCouponStatus(c *gin.Context) {
	couponID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的优惠券ID")
		return
	}

	var req UpdateCouponStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.marketingService.UpdateCouponStatus(c.Request.Context(), couponID, req.Status); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "状态更新成功", nil)
}

// DeleteCoupon 删除优惠券
// @Summary 删除优惠券
// @Tags 管理端-营销管理
// @Produce json
// @Security Bearer
// @Param id path int true "优惠券ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/marketing/coupons/{id} [delete]
func (h *MarketingHandler) DeleteCoupon(c *gin.Context) {
	couponID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的优惠券ID")
		return
	}

	if err := h.marketingService.DeleteCoupon(c.Request.Context(), couponID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetCampaignList 获取活动列表
// @Summary 获取活动列表
// @Tags 管理端-营销管理
// @Produce json
// @Security Bearer
// @Param status query int false "状态：0-禁用 1-启用"
// @Param type query string false "类型：discount/gift/flashsale/groupbuy"
// @Param keyword query string false "关键词"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=admin.AdminCampaignListResponse}
// @Router /api/v1/admin/marketing/campaigns [get]
func (h *MarketingHandler) GetCampaignList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	req := &adminService.AdminCampaignListRequest{
		Page:     page,
		PageSize: pageSize,
		Type:     c.Query("type"),
		Keyword:  c.Query("keyword"),
	}

	// 处理状态筛选
	if statusStr := c.Query("status"); statusStr != "" {
		status, err := strconv.ParseInt(statusStr, 10, 8)
		if err == nil {
			s := int8(status)
			req.Status = &s
		}
	}

	result, err := h.marketingService.GetCampaignList(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, result.List, result.Total, page, pageSize)
}

// GetCampaignDetail 获取活动详情
// @Summary 获取活动详情
// @Tags 管理端-营销管理
// @Produce json
// @Security Bearer
// @Param id path int true "活动ID"
// @Success 200 {object} response.Response{data=admin.AdminCampaignItem}
// @Router /api/v1/admin/marketing/campaigns/{id} [get]
func (h *MarketingHandler) GetCampaignDetail(c *gin.Context) {
	campaignID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动ID")
		return
	}

	campaign, err := h.marketingService.GetCampaignDetail(c.Request.Context(), campaignID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, campaign)
}

// CreateCampaign 创建活动
// @Summary 创建活动
// @Tags 管理端-营销管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body admin.CreateCampaignRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Campaign}
// @Router /api/v1/admin/marketing/campaigns [post]
func (h *MarketingHandler) CreateCampaign(c *gin.Context) {
	var req adminService.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	campaign, err := h.marketingService.CreateCampaign(c.Request.Context(), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "创建成功", campaign)
}

// UpdateCampaign 更新活动
// @Summary 更新活动
// @Tags 管理端-营销管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "活动ID"
// @Param request body admin.UpdateCampaignRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/marketing/campaigns/{id} [put]
func (h *MarketingHandler) UpdateCampaign(c *gin.Context) {
	campaignID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动ID")
		return
	}

	var req adminService.UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.marketingService.UpdateCampaign(c.Request.Context(), campaignID, &req); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// UpdateCampaignStatusRequest 更新活动状态请求
type UpdateCampaignStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// UpdateCampaignStatus 更新活动状态
// @Summary 更新活动状态
// @Tags 管理端-营销管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "活动ID"
// @Param request body UpdateCampaignStatusRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/marketing/campaigns/{id}/status [put]
func (h *MarketingHandler) UpdateCampaignStatus(c *gin.Context) {
	campaignID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动ID")
		return
	}

	var req UpdateCampaignStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.marketingService.UpdateCampaignStatus(c.Request.Context(), campaignID, req.Status); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "状态更新成功", nil)
}

// DeleteCampaign 删除活动
// @Summary 删除活动
// @Tags 管理端-营销管理
// @Produce json
// @Security Bearer
// @Param id path int true "活动ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/marketing/campaigns/{id} [delete]
func (h *MarketingHandler) DeleteCampaign(c *gin.Context) {
	campaignID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动ID")
		return
	}

	if err := h.marketingService.DeleteCampaign(c.Request.Context(), campaignID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}
