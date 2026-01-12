// Package auth 提供认证相关的 HTTP Handler
package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	authService "github.com/dumeirei/smart-locker-backend/internal/service/auth"
)

// Handler 认证处理器
type Handler struct {
	authService   *authService.AuthService
	wechatService *authService.WechatService
	codeService   *authService.CodeService
}

// NewHandler 创建认证处理器
func NewHandler(
	authSvc *authService.AuthService,
	wechatSvc *authService.WechatService,
	codeSvc *authService.CodeService,
) *Handler {
	return &Handler{
		authService:   authSvc,
		wechatService: wechatSvc,
		codeService:   codeSvc,
	}
}

// SendSmsCode 发送短信验证码
// @Summary 发送短信验证码
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body authService.SendSmsCodeRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /auth/sms/send [post]
func (h *Handler) SendSmsCode(c *gin.Context) {
	var req authService.SendSmsCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if handler.HandleError(c, h.authService.SendSmsCode(c.Request.Context(), &req)) {
		return
	}

	response.Success(c, gin.H{
		"expire_in": int(h.codeService.GetCodeExpireIn().Seconds()),
	})
}

// SmsLogin 短信验证码登录
// @Summary 短信验证码登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body authService.SmsLoginRequest true "请求参数"
// @Success 200 {object} response.Response{data=authService.LoginResponse}
// @Router /auth/login/sms [post]
func (h *Handler) SmsLogin(c *gin.Context) {
	var req authService.SmsLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.authService.SmsLogin(c.Request.Context(), &req)
	handler.MustSucceed(c, err, result)
}

// WechatLogin 微信小程序登录
// @Summary 微信小程序登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body authService.WechatLoginRequest true "请求参数"
// @Success 200 {object} response.Response{data=authService.LoginResponse}
// @Router /auth/login/wechat [post]
func (h *Handler) WechatLogin(c *gin.Context) {
	var req authService.WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.wechatService.WechatLogin(c.Request.Context(), &req)
	handler.MustSucceed(c, err, result)
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken 刷新 Token
// @Summary 刷新 Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	handler.MustSucceed(c, err, tokenPair)
}

// BindPhoneRequest 绑定手机号请求
type BindPhoneRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// BindPhone 绑定手机号
// @Summary 绑定手机号
// @Tags 认证
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body BindPhoneRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /auth/bind-phone [post]
func (h *Handler) BindPhone(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req BindPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	handler.MustSucceed(c, h.wechatService.BindPhone(c.Request.Context(), userID, req.Phone, req.Code, h.codeService), nil)
}

// GetCurrentUser 获取当前用户信息
// @Summary 获取当前用户信息
// @Tags 认证
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=authService.UserInfo}
// @Router /auth/me [get]
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	response.Success(c, &authService.UserInfo{
		ID:            user.ID,
		Phone:         user.Phone,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Gender:        user.Gender,
		MemberLevelID: user.MemberLevelID,
		Points:        user.Points,
		IsVerified:    user.IsVerified,
	})
}

// Logout 退出登录
// @Summary 退出登录
// @Tags 认证
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	// TODO: 如果需要，可以将 token 加入黑名单
	response.Success(c, nil)
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		// 公开接口
		auth.POST("/sms/send", h.SendSmsCode)
		auth.POST("/login/sms", h.SmsLogin)
		auth.POST("/login/wechat", h.WechatLogin)
		auth.POST("/refresh", h.RefreshToken)
	}
}

// RegisterProtectedRoutes 注册需要认证的路由
func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.GET("/me", h.GetCurrentUser)
		auth.POST("/bind-phone", h.BindPhone)
		auth.POST("/logout", h.Logout)
	}
}
