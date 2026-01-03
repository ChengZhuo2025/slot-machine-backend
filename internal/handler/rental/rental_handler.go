// Package rental 提供租借相关的 HTTP Handler
package rental

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/common/utils"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	rentalService "github.com/dumeirei/smart-locker-backend/internal/service/rental"
)

// Handler 租借处理器
type Handler struct {
	rentalService *rentalService.RentalService
}

// NewHandler 创建租借处理器
func NewHandler(rentalSvc *rentalService.RentalService) *Handler {
	return &Handler{
		rentalService: rentalSvc,
	}
}

// CreateRental 创建租借订单
// @Summary 创建租借订单
// @Tags 租借
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body rentalService.CreateRentalRequest true "请求参数"
// @Success 200 {object} response.Response{data=rentalService.RentalInfo}
// @Router /api/v1/rental [post]
func (h *Handler) CreateRental(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req rentalService.CreateRentalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	rental, err := h.rentalService.CreateRental(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, rental)
}

// PayRental 支付租借订单
// @Summary 支付租借订单
// @Tags 租借
// @Produce json
// @Security Bearer
// @Param id path int true "租借ID"
// @Success 200 {object} response.Response
// @Router /api/v1/rental/{id}/pay [post]
func (h *Handler) PayRental(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	rentalID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的租借ID")
		return
	}

	if err := h.rentalService.PayRental(c.Request.Context(), userID, rentalID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// StartRental 开始租借（取货）
// @Summary 开始租借
// @Tags 租借
// @Produce json
// @Security Bearer
// @Param id path int true "租借ID"
// @Success 200 {object} response.Response
// @Router /api/v1/rental/{id}/start [post]
func (h *Handler) StartRental(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	rentalID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的租借ID")
		return
	}

	if err := h.rentalService.StartRental(c.Request.Context(), userID, rentalID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ReturnRental 归还租借
// @Summary 归还租借
// @Tags 租借
// @Produce json
// @Security Bearer
// @Param id path int true "租借ID"
// @Success 200 {object} response.Response
// @Router /api/v1/rental/{id}/return [post]
func (h *Handler) ReturnRental(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	rentalID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的租借ID")
		return
	}

	if err := h.rentalService.ReturnRental(c.Request.Context(), userID, rentalID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// CancelRental 取消租借
// @Summary 取消租借
// @Tags 租借
// @Produce json
// @Security Bearer
// @Param id path int true "租借ID"
// @Success 200 {object} response.Response
// @Router /api/v1/rental/{id}/cancel [post]
func (h *Handler) CancelRental(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	rentalID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的租借ID")
		return
	}

	if err := h.rentalService.CancelRental(c.Request.Context(), userID, rentalID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetRental 获取租借详情
// @Summary 获取租借详情
// @Tags 租借
// @Produce json
// @Security Bearer
// @Param id path int true "租借ID"
// @Success 200 {object} response.Response{data=rentalService.RentalInfo}
// @Router /api/v1/rental/{id} [get]
func (h *Handler) GetRental(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	rentalID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的租借ID")
		return
	}

	rental, err := h.rentalService.GetRental(c.Request.Context(), userID, rentalID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, rental)
}

// ListRentals 获取租借列表
// @Summary 获取租借列表
// @Tags 租借
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param status query int false "状态筛选"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/rental [get]
func (h *Handler) ListRentals(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var pagination utils.Pagination
	pagination.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pagination.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "10"))
	pagination.Normalize()

	var status *int8
	if statusStr := c.Query("status"); statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			st := int8(s)
			status = &st
		}
	}

	rentals, total, err := h.rentalService.ListRentals(
		c.Request.Context(),
		userID,
		pagination.GetOffset(),
		pagination.GetLimit(),
		status,
	)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, rentals, total, pagination.Page, pagination.PageSize)
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	rental := r.Group("/rental")
	{
		rental.POST("", h.CreateRental)
		rental.GET("", h.ListRentals)
		rental.GET("/:id", h.GetRental)
		rental.POST("/:id/pay", h.PayRental)
		rental.POST("/:id/start", h.StartRental)
		rental.POST("/:id/return", h.ReturnRental)
		rental.POST("/:id/cancel", h.CancelRental)
	}
}
