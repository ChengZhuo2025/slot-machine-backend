// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService *adminService.UserAdminService
}

// NewUserHandler 创建用户管理处理器
func NewUserHandler(userService *adminService.UserAdminService) *UserHandler {
	return &UserHandler{userService: userService}
}

// List 获取用户列表
// @Summary 获取用户列表
// @Tags 管理-用户管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param phone query string false "手机号"
// @Param nickname query string false "昵称"
// @Param status query int false "状态"
// @Param member_level_id query int false "会员等级ID"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=response.ListData}
// @Router /api/v1/admin/users [get]
func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := &adminService.UserListFilters{
		Phone:    c.Query("phone"),
		Nickname: c.Query("nickname"),
	}

	if s := c.Query("status"); s != "" {
		status, _ := strconv.ParseInt(s, 10, 8)
		val := int8(status)
		filters.Status = &val
	}
	if s := c.Query("member_level_id"); s != "" {
		filters.MemberLevelID, _ = strconv.ParseInt(s, 10, 64)
	}
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		filters.StartDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endOfDay := t.Add(24*time.Hour - time.Second)
		filters.EndDate = &endOfDay
	}

	users, total, err := h.userService.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessList(c, users, total, page, pageSize)
}

// GetByID 获取用户详情
// @Summary 获取用户详情
// @Tags 管理-用户管理
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=models.User}
// @Router /api/v1/admin/users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	response.Success(c, user)
}

// Update 更新用户
// @Summary 更新用户信息
// @Tags 管理-用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Param request body adminService.UpdateUserRequest true "更新用户请求"
// @Success 200 {object} response.Response{data=models.User}
// @Router /api/v1/admin/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	var req adminService.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.userService.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, user)
}

// Enable 启用用户
// @Summary 启用用户
// @Tags 管理-用户管理
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/users/{id}/enable [post]
func (h *UserHandler) Enable(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := h.userService.Enable(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// Disable 禁用用户
// @Summary 禁用用户
// @Tags 管理-用户管理
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/users/{id}/disable [post]
func (h *UserHandler) Disable(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := h.userService.Disable(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// AdjustPointsRequest 调整积分请求
type AdjustPointsRequest struct {
	Points int    `json:"points" binding:"required"` // 正数增加，负数扣减
	Remark string `json:"remark"`
}

// AdjustPoints 调整积分
// @Summary 调整用户积分
// @Tags 管理-用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Param request body AdjustPointsRequest true "调整积分请求"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/users/{id}/adjust-points [post]
func (h *UserHandler) AdjustPoints(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	var req AdjustPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.userService.AdjustPoints(c.Request.Context(), id, req.Points, req.Remark); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetStatistics 获取用户统计
// @Summary 获取用户统计
// @Tags 管理-用户管理
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.UserStatistics}
// @Router /api/v1/admin/users/statistics [get]
func (h *UserHandler) GetStatistics(c *gin.Context) {
	stats, err := h.userService.GetStatistics(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
}
