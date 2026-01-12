// Package admin 提供管理员相关的 HTTP Handler
package admin

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// AuthHandler 管理员认证处理器
type AuthHandler struct {
	adminAuthService *adminService.AdminAuthService
}

// NewAuthHandler 创建管理员认证处理器
func NewAuthHandler(adminAuthSvc *adminService.AdminAuthService) *AuthHandler {
	return &AuthHandler{
		adminAuthService: adminAuthSvc,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 管理员登录
// @Summary 管理员登录
// @Tags 管理员认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "请求参数"
// @Success 200 {object} response.Response{data=adminService.LoginResponse}
// @Router /admin/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	loginReq := &adminService.LoginRequest{
		Username: req.Username,
		Password: req.Password,
		IP:       c.ClientIP(),
	}

	result, err := h.adminAuthService.Login(c.Request.Context(), loginReq)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrAdminNotFound):
			response.Unauthorized(c, "用户名或密码错误")
		case errors.Is(err, adminService.ErrInvalidPassword):
			response.Unauthorized(c, "用户名或密码错误")
		case errors.Is(err, adminService.ErrAdminDisabled):
			response.Forbidden(c, "账号已被禁用")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, result)
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken 刷新 Token
// @Summary 刷新管理员 Token
// @Tags 管理员认证
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	tokenPair, err := h.adminAuthService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "刷新令牌失败")
		return
	}

	response.Success(c, tokenPair)
}

// GetCurrentAdmin 获取当前管理员信息
// @Summary 获取当前管理员信息
// @Tags 管理员认证
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.LoginResponse}
// @Router /admin/auth/me [get]
func (h *AuthHandler) GetCurrentAdmin(c *gin.Context) {
	adminID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	result, err := h.adminAuthService.GetAdminWithPermissions(c.Request.Context(), adminID)
	if err != nil {
		if errors.Is(err, adminService.ErrAdminNotFound) {
			response.Unauthorized(c, "管理员不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
}

// ChangePassword 修改密码
// @Summary 修改管理员密码
// @Tags 管理员认证
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body ChangePasswordRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	adminID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	changeReq := &adminService.ChangePasswordRequest{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	err := h.adminAuthService.ChangePassword(c.Request.Context(), adminID, changeReq)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrAdminNotFound):
			response.NotFound(c, "管理员不存在")
		case errors.Is(err, adminService.ErrOldPasswordInvalid):
			response.BadRequest(c, "原密码错误")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// Logout 退出登录
// @Summary 管理员退出登录
// @Tags 管理员认证
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /admin/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// 当前设计：依赖 JWT 自然过期机制，无需 token 黑名单
	// 如需立即吊销 token，可使用 Redis 实现黑名单机制
	response.Success(c, nil)
}

// RegisterRoutes 注册公开路由
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
	}
}

// RegisterProtectedRoutes 注册需要认证的路由
func (h *AuthHandler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.GET("/me", h.GetCurrentAdmin)
		auth.PUT("/password", h.ChangePassword)
		auth.POST("/logout", h.Logout)
	}
}
