// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	contentService "github.com/dumeirei/smart-locker-backend/internal/service/content"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

// BannerHandler 轮播图管理处理器
type BannerHandler struct {
	bannerService *contentService.BannerAdminService
}

// NewBannerHandler 创建轮播图管理处理器
func NewBannerHandler(bannerService *contentService.BannerAdminService) *BannerHandler {
	return &BannerHandler{bannerService: bannerService}
}

// Create 创建轮播图
// @Summary 创建轮播图
// @Tags 管理-轮播图
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body contentService.CreateBannerRequest true "轮播图信息"
// @Success 200 {object} response.Response{data=models.Banner}
// @Router /api/v1/admin/banners [post]
func (h *BannerHandler) Create(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var req contentService.CreateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	banner, err := h.bannerService.Create(c.Request.Context(), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, banner)
}

// GetByID 获取轮播图详情
// @Summary 获取轮播图详情
// @Tags 管理-轮播图
// @Produce json
// @Security Bearer
// @Param id path int true "轮播图ID"
// @Success 200 {object} response.Response{data=models.Banner}
// @Router /api/v1/admin/banners/{id} [get]
func (h *BannerHandler) GetByID(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "轮播图")
	if !ok {
		return
	}

	banner, err := h.bannerService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "轮播图不存在")
		return
	}

	response.Success(c, banner)
}

// Update 更新轮播图
// @Summary 更新轮播图
// @Tags 管理-轮播图
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "轮播图ID"
// @Param request body contentService.UpdateBannerRequest true "轮播图信息"
// @Success 200 {object} response.Response{data=models.Banner}
// @Router /api/v1/admin/banners/{id} [put]
func (h *BannerHandler) Update(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "轮播图")
	if !ok {
		return
	}

	var req contentService.UpdateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	banner, err := h.bannerService.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, banner)
}

// Delete 删除轮播图
// @Summary 删除轮播图
// @Tags 管理-轮播图
// @Produce json
// @Security Bearer
// @Param id path int true "轮播图ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/banners/{id} [delete]
func (h *BannerHandler) Delete(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "轮播图")
	if !ok {
		return
	}

	if err := h.bannerService.Delete(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// List 获取轮播图列表
// @Summary 获取轮播图列表
// @Tags 管理-轮播图
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param position query string false "位置"
// @Param is_active query bool false "是否启用"
// @Param keyword query string false "关键词"
// @Success 200 {object} response.Response{data=response.ListData}
// @Router /api/v1/admin/banners [get]
func (h *BannerHandler) List(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)
	position := c.Query("position")
	keyword := c.Query("keyword")

	var isActive *bool
	if p := c.Query("is_active"); p != "" {
		val := p == "true" || p == "1"
		isActive = &val
	}

	banners, total, err := h.bannerService.List(c.Request.Context(), p.Page, p.PageSize, position, isActive, keyword)
	handler.MustSucceedPage(c, err, banners, total, p.Page, p.PageSize)
}

// UpdateStatus 更新状态
// @Summary 更新轮播图状态
// @Tags 管理-轮播图
// @Produce json
// @Security Bearer
// @Param id path int true "轮播图ID"
// @Param is_active query bool true "是否启用"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/banners/{id}/status [put]
func (h *BannerHandler) UpdateStatus(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "轮播图")
	if !ok {
		return
	}

	isActive := c.Query("is_active") == "true" || c.Query("is_active") == "1"

	if err := h.bannerService.UpdateStatus(c.Request.Context(), id, isActive); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// UpdateSortRequest 更新排序请求
type UpdateSortRequest struct {
	Sort int `json:"sort"`
}

// UpdateSort 更新排序
// @Summary 更新轮播图排序
// @Tags 管理-轮播图
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "轮播图ID"
// @Param request body UpdateSortRequest true "排序值"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/banners/{id}/sort [put]
func (h *BannerHandler) UpdateSort(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "轮播图")
	if !ok {
		return
	}

	var req UpdateSortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.bannerService.UpdateSort(c.Request.Context(), id, req.Sort); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetStatistics 获取统计
// @Summary 获取轮播图统计
// @Tags 管理-轮播图
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=contentService.BannerStatistics}
// @Router /api/v1/admin/banners/statistics [get]
func (h *BannerHandler) GetStatistics(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	stats, err := h.bannerService.GetStatistics(c.Request.Context())
	handler.MustSucceed(c, err, stats)
}

// FeedbackHandler 反馈管理处理器
type FeedbackHandler struct {
	feedbackService *userService.FeedbackAdminService
}

// NewFeedbackHandler 创建反馈管理处理器
func NewFeedbackHandler(feedbackService *userService.FeedbackAdminService) *FeedbackHandler {
	return &FeedbackHandler{feedbackService: feedbackService}
}

// List 获取反馈列表
// @Summary 获取反馈列表
// @Tags 管理-反馈
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param type query string false "反馈类型"
// @Param status query int false "状态"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=response.ListData}
// @Router /api/v1/admin/feedbacks [get]
func (h *FeedbackHandler) List(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)
	feedbackType := c.Query("type")

	var status *int8
	if s := c.Query("status"); s != "" {
		if val, err := strconv.ParseInt(s, 10, 8); err == nil {
			st := int8(val)
			status = &st
		}
	}

	startDate, endDate, ok := handler.ParseQueryDateRange(c)
	if !ok {
		return
	}

	feedbacks, total, err := h.feedbackService.List(c.Request.Context(), p.Page, p.PageSize, feedbackType, status, startDate, endDate)
	handler.MustSucceedPage(c, err, feedbacks, total, p.Page, p.PageSize)
}

// GetByID 获取反馈详情
// @Summary 获取反馈详情
// @Tags 管理-反馈
// @Produce json
// @Security Bearer
// @Param id path int true "反馈ID"
// @Success 200 {object} response.Response{data=models.UserFeedback}
// @Router /api/v1/admin/feedbacks/{id} [get]
func (h *FeedbackHandler) GetByID(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "反馈")
	if !ok {
		return
	}

	feedback, err := h.feedbackService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "反馈不存在")
		return
	}

	response.Success(c, feedback)
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status int8 `json:"status" binding:"required,oneof=0 1 2"`
}

// UpdateStatus 更新状态
// @Summary 更新反馈状态
// @Tags 管理-反馈
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "反馈ID"
// @Param request body UpdateStatusRequest true "状态"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/feedbacks/{id}/status [put]
func (h *FeedbackHandler) UpdateStatus(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "反馈")
	if !ok {
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.feedbackService.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ReplyRequest 回复请求
type ReplyRequest struct {
	Reply string `json:"reply" binding:"required,max=2000"`
}

// Reply 回复反馈
// @Summary 回复反馈
// @Tags 管理-反馈
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "反馈ID"
// @Param request body ReplyRequest true "回复内容"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/feedbacks/{id}/reply [post]
func (h *FeedbackHandler) Reply(c *gin.Context) {
	adminID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "反馈")
	if !ok {
		return
	}

	var req ReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.feedbackService.Reply(c.Request.Context(), id, req.Reply, adminID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetStatistics 获取反馈统计
// @Summary 获取反馈统计
// @Tags 管理-反馈
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=userService.FeedbackStatistics}
// @Router /api/v1/admin/feedbacks/statistics [get]
func (h *FeedbackHandler) GetStatistics(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	stats, err := h.feedbackService.GetStatistics(c.Request.Context())
	handler.MustSucceed(c, err, stats)
}
