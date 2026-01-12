// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
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
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	filters := &adminService.UserListFilters{
		Phone:    c.Query("phone"),
		Nickname: c.Query("nickname"),
	}

	if s := c.Query("status"); s != "" {
		if status, err := strconv.ParseInt(s, 10, 8); err == nil {
			val := int8(status)
			filters.Status = &val
		}
	}
	if s := c.Query("member_level_id"); s != "" {
		if memberLevelID, err := strconv.ParseInt(s, 10, 64); err == nil {
			filters.MemberLevelID = memberLevelID
		}
	}

	startDate, endDate, ok := handler.ParseQueryDateRange(c)
	if !ok {
		return
	}
	filters.StartDate = startDate
	filters.EndDate = endDate

	users, total, err := h.userService.List(c.Request.Context(), p.Page, p.PageSize, filters)
	handler.MustSucceedPage(c, err, users, total, p.Page, p.PageSize)
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
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "用户")
	if !ok {
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
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "用户")
	if !ok {
		return
	}

	var req adminService.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.userService.Update(c.Request.Context(), id, &req)
	handler.MustSucceed(c, err, user)
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
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "用户")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.userService.Enable(c.Request.Context(), id), nil)
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
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "用户")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.userService.Disable(c.Request.Context(), id), nil)
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
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "用户")
	if !ok {
		return
	}

	var req AdjustPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	handler.MustSucceed(c, h.userService.AdjustPoints(c.Request.Context(), id, req.Points, req.Remark), nil)
}

// GetStatistics 获取用户统计
// @Summary 获取用户统计
// @Tags 管理-用户管理
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.UserStatistics}
// @Router /api/v1/admin/users/statistics [get]
func (h *UserHandler) GetStatistics(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	stats, err := h.userService.GetStatistics(c.Request.Context())
	handler.MustSucceed(c, err, stats)
}
