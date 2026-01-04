// Package mall 提供商城相关的 HTTP Handler
package mall

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	mallService "github.com/dumeirei/smart-locker-backend/internal/service/mall"
)

// OrderHandler 商城订单处理器
type OrderHandler struct {
	orderService *mallService.MallOrderService
}

// NewOrderHandler 创建商城订单处理器
func NewOrderHandler(orderSvc *mallService.MallOrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderSvc,
	}
}

// CreateOrder 创建订单
// @Summary 创建商城订单
// @Tags 商城订单
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body mall.CreateMallOrderRequest true "请求参数"
// @Success 200 {object} response.Response{data=mall.MallOrderInfo}
// @Router /api/v1/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req mallService.CreateMallOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, order)
}

// CreateOrderFromCart 从购物车创建订单
// @Summary 从购物车创建订单
// @Tags 商城订单
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body mall.CreateFromCartRequest true "请求参数"
// @Success 200 {object} response.Response{data=mall.MallOrderInfo}
// @Router /api/v1/orders/from-cart [post]
func (h *OrderHandler) CreateOrderFromCart(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req mallService.CreateFromCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	order, err := h.orderService.CreateOrderFromCart(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, order)
}

// GetOrderDetail 获取订单详情
// @Summary 获取订单详情
// @Tags 商城订单
// @Produce json
// @Security Bearer
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response{data=mall.MallOrderInfo}
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandler) GetOrderDetail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单ID")
		return
	}

	order, err := h.orderService.GetOrderDetail(c.Request.Context(), userID, orderID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, order)
}

// GetOrders 获取订单列表
// @Summary 获取订单列表
// @Tags 商城订单
// @Produce json
// @Security Bearer
// @Param status query string false "订单状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=[]mall.MallOrderInfo}
// @Router /api/v1/orders [get]
func (h *OrderHandler) GetOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	orders, total, err := h.orderService.GetUserOrders(c.Request.Context(), userID, status, page, pageSize)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":       orders,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// CancelOrder 取消订单
// @Summary 取消订单
// @Tags 商城订单
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "订单ID"
// @Param reason query string false "取消原因"
// @Success 200 {object} response.Response
// @Router /api/v1/orders/{id}/cancel [post]
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单ID")
		return
	}

	reason := c.Query("reason")

	if err := h.orderService.CancelOrder(c.Request.Context(), userID, orderID, reason); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ConfirmReceive 确认收货
// @Summary 确认收货
// @Tags 商城订单
// @Produce json
// @Security Bearer
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response
// @Router /api/v1/orders/{id}/confirm [post]
func (h *OrderHandler) ConfirmReceive(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单ID")
		return
	}

	if err := h.orderService.ConfirmReceive(c.Request.Context(), userID, orderID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
