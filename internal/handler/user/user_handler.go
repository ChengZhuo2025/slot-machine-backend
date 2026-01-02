// Package user 提供用户相关的 HTTP Handler
package user

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"smart-locker-backend/internal/common/errors"
	"smart-locker-backend/internal/common/response"
	"smart-locker-backend/internal/common/utils"
	"smart-locker-backend/internal/middleware"
	userService "smart-locker-backend/internal/service/user"
)

// Handler 用户处理器
type Handler struct {
	userService   *userService.UserService
	walletService *userService.WalletService
}

// NewHandler 创建用户处理器
func NewHandler(
	userSvc *userService.UserService,
	walletSvc *userService.WalletService,
) *Handler {
	return &Handler{
		userService:   userSvc,
		walletService: walletSvc,
	}
}

// GetProfile 获取用户信息
// @Summary 获取用户信息
// @Tags 用户
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=userService.UserProfile}
// @Router /api/v1/user/profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	profile, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, profile)
}

// UpdateProfile 更新用户信息
// @Summary 更新用户信息
// @Tags 用户
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body userService.UpdateProfileRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/user/profile [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req userService.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.userService.UpdateProfile(c.Request.Context(), userID, &req); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetWallet 获取钱包信息
// @Summary 获取钱包信息
// @Tags 用户-钱包
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=userService.WalletInfo}
// @Router /api/v1/user/wallet [get]
func (h *Handler) GetWallet(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	wallet, err := h.walletService.GetWallet(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, wallet)
}

// GetTransactions 获取交易记录
// @Summary 获取交易记录
// @Tags 用户-钱包
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param type query string false "交易类型"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/user/wallet/transactions [get]
func (h *Handler) GetTransactions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var pagination utils.Pagination
	pagination.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pagination.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "10"))
	pagination.Normalize()

	txType := c.Query("type")

	transactions, total, err := h.walletService.GetTransactions(
		c.Request.Context(),
		userID,
		pagination.GetOffset(),
		pagination.GetLimit(),
		txType,
	)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, transactions, total, pagination.Page, pagination.PageSize)
}

// GetMemberLevels 获取会员等级列表
// @Summary 获取会员等级列表
// @Tags 用户
// @Produce json
// @Success 200 {object} response.Response{data=[]userService.MemberLevelInfo}
// @Router /api/v1/user/member-levels [get]
func (h *Handler) GetMemberLevels(c *gin.Context) {
	levels, err := h.userService.GetMemberLevels(c.Request.Context())
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

// RealNameVerify 实名认证
// @Summary 实名认证
// @Tags 用户
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body userService.RealNameVerifyRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/user/real-name-verify [post]
func (h *Handler) RealNameVerify(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req userService.RealNameVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.userService.RealNameVerify(c.Request.Context(), userID, &req); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetPoints 获取用户积分
// @Summary 获取用户积分
// @Tags 用户
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/user/points [get]
func (h *Handler) GetPoints(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	points, err := h.userService.GetPoints(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"points": points})
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	user := r.Group("/user")
	{
		user.GET("/profile", h.GetProfile)
		user.PUT("/profile", h.UpdateProfile)
		user.GET("/wallet", h.GetWallet)
		user.GET("/wallet/transactions", h.GetTransactions)
		user.GET("/member-levels", h.GetMemberLevels)
		user.POST("/real-name-verify", h.RealNameVerify)
		user.GET("/points", h.GetPoints)
	}
}
