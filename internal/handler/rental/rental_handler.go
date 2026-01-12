// Package rental 提供租借相关的 HTTP Handler
package rental

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
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
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req rentalService.CreateRentalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	rental, err := h.rentalService.CreateRental(c.Request.Context(), userID, &req)
	handler.MustSucceed(c, err, rental)
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
	userID, rentalID, ok := handler.RequireUserAndParseID(c, "租借")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.rentalService.PayRental(c.Request.Context(), userID, rentalID), nil)
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
	userID, rentalID, ok := handler.RequireUserAndParseID(c, "租借")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.rentalService.StartRental(c.Request.Context(), userID, rentalID), nil)
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
	userID, rentalID, ok := handler.RequireUserAndParseID(c, "租借")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.rentalService.ReturnRental(c.Request.Context(), userID, rentalID), nil)
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
	userID, rentalID, ok := handler.RequireUserAndParseID(c, "租借")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.rentalService.CancelRental(c.Request.Context(), userID, rentalID), nil)
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
	userID, rentalID, ok := handler.RequireUserAndParseID(c, "租借")
	if !ok {
		return
	}

	rental, err := h.rentalService.GetRental(c.Request.Context(), userID, rentalID)
	handler.MustSucceed(c, err, rental)
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
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	var status *string
	if statusStr := c.Query("status"); statusStr != "" {
		status = &statusStr
	}

	rentals, total, err := h.rentalService.ListRentals(
		c.Request.Context(),
		userID,
		p.GetOffset(),
		p.GetLimit(),
		status,
	)
	handler.MustSucceedPage(c, err, rentals, total, p.Page, p.PageSize)
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
