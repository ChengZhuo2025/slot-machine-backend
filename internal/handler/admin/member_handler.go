// Package admin 提供管理端 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// MemberHandler 会员管理处理器
type MemberHandler struct {
	memberService *adminService.MemberAdminService
}

// NewMemberHandler 创建会员管理处理器
func NewMemberHandler(memberSvc *adminService.MemberAdminService) *MemberHandler {
	return &MemberHandler{
		memberService: memberSvc,
	}
}

// ===================== 会员等级管理 =====================

// GetMemberLevelList 获取会员等级列表
// @Summary 获取会员等级列表
// @Description 获取系统中所有会员等级
// @Tags 管理端-会员管理
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]admin.AdminMemberLevelItem}
// @Router /api/admin/member/levels [get]
func (h *MemberHandler) GetMemberLevelList(c *gin.Context) {
	levels, err := h.memberService.GetMemberLevelList(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, levels)
}

// GetMemberLevelDetail 获取会员等级详情
// @Summary 获取会员等级详情
// @Tags 管理端-会员管理
// @Produce json
// @Security Bearer
// @Param id path int true "会员等级ID"
// @Success 200 {object} response.Response{data=admin.AdminMemberLevelItem}
// @Router /api/admin/member/levels/{id} [get]
func (h *MemberHandler) GetMemberLevelDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的会员等级ID")
		return
	}

	level, err := h.memberService.GetMemberLevelDetail(c.Request.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, level)
}

// CreateMemberLevel 创建会员等级
// @Summary 创建会员等级
// @Tags 管理端-会员管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body admin.CreateMemberLevelRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/admin/member/levels [post]
func (h *MemberHandler) CreateMemberLevel(c *gin.Context) {
	var req adminService.CreateMemberLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	level, err := h.memberService.CreateMemberLevel(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, level)
}

// UpdateMemberLevel 更新会员等级
// @Summary 更新会员等级
// @Tags 管理端-会员管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "会员等级ID"
// @Param request body admin.UpdateMemberLevelRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/admin/member/levels/{id} [put]
func (h *MemberHandler) UpdateMemberLevel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的会员等级ID")
		return
	}

	var req adminService.UpdateMemberLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.memberService.UpdateMemberLevel(c.Request.Context(), id, &req); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteMemberLevel 删除会员等级
// @Summary 删除会员等级
// @Tags 管理端-会员管理
// @Produce json
// @Security Bearer
// @Param id path int true "会员等级ID"
// @Success 200 {object} response.Response
// @Router /api/admin/member/levels/{id} [delete]
func (h *MemberHandler) DeleteMemberLevel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的会员等级ID")
		return
	}

	if err := h.memberService.DeleteMemberLevel(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ===================== 会员套餐管理 =====================

// GetMemberPackageList 获取会员套餐列表
// @Summary 获取会员套餐列表
// @Tags 管理端-会员管理
// @Produce json
// @Security Bearer
// @Param status query int false "状态：0-禁用 1-启用"
// @Param member_level_id query int false "会员等级ID"
// @Param is_recommend query bool false "是否推荐"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=admin.AdminPackageListResponse}
// @Router /api/admin/member/packages [get]
func (h *MemberHandler) GetMemberPackageList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	req := &adminService.AdminPackageListRequest{
		Page:     page,
		PageSize: pageSize,
	}

	// 处理状态筛选
	if statusStr := c.Query("status"); statusStr != "" {
		status, err := strconv.ParseInt(statusStr, 10, 8)
		if err == nil {
			s := int8(status)
			req.Status = &s
		}
	}

	// 处理会员等级筛选
	if levelIDStr := c.Query("member_level_id"); levelIDStr != "" {
		levelID, err := strconv.ParseInt(levelIDStr, 10, 64)
		if err == nil {
			req.MemberLevelID = &levelID
		}
	}

	// 处理推荐筛选
	if isRecommendStr := c.Query("is_recommend"); isRecommendStr != "" {
		isRecommend := isRecommendStr == "true" || isRecommendStr == "1"
		req.IsRecommend = &isRecommend
	}

	result, err := h.memberService.GetMemberPackageList(c.Request.Context(), req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, result.List, result.Total, page, pageSize)
}

// GetMemberPackageDetail 获取会员套餐详情
// @Summary 获取会员套餐详情
// @Tags 管理端-会员管理
// @Produce json
// @Security Bearer
// @Param id path int true "套餐ID"
// @Success 200 {object} response.Response{data=admin.AdminMemberPackageItem}
// @Router /api/admin/member/packages/{id} [get]
func (h *MemberHandler) GetMemberPackageDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的套餐ID")
		return
	}

	pkg, err := h.memberService.GetMemberPackageDetail(c.Request.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, pkg)
}

// CreateMemberPackage 创建会员套餐
// @Summary 创建会员套餐
// @Tags 管理端-会员管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body admin.CreateMemberPackageRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/admin/member/packages [post]
func (h *MemberHandler) CreateMemberPackage(c *gin.Context) {
	var req adminService.CreateMemberPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	pkg, err := h.memberService.CreateMemberPackage(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, pkg)
}

// UpdateMemberPackage 更新会员套餐
// @Summary 更新会员套餐
// @Tags 管理端-会员管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "套餐ID"
// @Param request body admin.UpdateMemberPackageRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/admin/member/packages/{id} [put]
func (h *MemberHandler) UpdateMemberPackage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的套餐ID")
		return
	}

	var req adminService.UpdateMemberPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.memberService.UpdateMemberPackage(c.Request.Context(), id, &req); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// UpdateMemberPackageStatus 更新套餐状态
// @Summary 更新套餐状态
// @Tags 管理端-会员管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "套餐ID"
// @Param request body object{status=int} true "状态"
// @Success 200 {object} response.Response
// @Router /api/admin/member/packages/{id}/status [put]
func (h *MemberHandler) UpdateMemberPackageStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的套餐ID")
		return
	}

	var req struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.memberService.UpdateMemberPackageStatus(c.Request.Context(), id, req.Status); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteMemberPackage 删除会员套餐
// @Summary 删除会员套餐
// @Tags 管理端-会员管理
// @Produce json
// @Security Bearer
// @Param id path int true "套餐ID"
// @Success 200 {object} response.Response
// @Router /api/admin/member/packages/{id} [delete]
func (h *MemberHandler) DeleteMemberPackage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的套餐ID")
		return
	}

	if err := h.memberService.DeleteMemberPackage(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ===================== 会员统计 =====================

// GetMemberStats 获取会员统计
// @Summary 获取会员统计
// @Description 获取会员等级分布、套餐销售等统计信息
// @Tags 管理端-会员管理
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=admin.MemberStats}
// @Router /api/admin/member/stats [get]
func (h *MemberHandler) GetMemberStats(c *gin.Context) {
	stats, err := h.memberService.GetMemberStats(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
}

// RegisterRoutes 注册会员管理路由
func (h *MemberHandler) RegisterRoutes(r *gin.RouterGroup) {
	member := r.Group("/member")
	{
		// 统计
		member.GET("/stats", h.GetMemberStats)

		// 会员等级
		member.GET("/levels", h.GetMemberLevelList)
		member.GET("/levels/:id", h.GetMemberLevelDetail)
		member.POST("/levels", h.CreateMemberLevel)
		member.PUT("/levels/:id", h.UpdateMemberLevel)
		member.DELETE("/levels/:id", h.DeleteMemberLevel)

		// 会员套餐
		member.GET("/packages", h.GetMemberPackageList)
		member.GET("/packages/:id", h.GetMemberPackageDetail)
		member.POST("/packages", h.CreateMemberPackage)
		member.PUT("/packages/:id", h.UpdateMemberPackage)
		member.PUT("/packages/:id/status", h.UpdateMemberPackageStatus)
		member.DELETE("/packages/:id", h.DeleteMemberPackage)
	}
}
