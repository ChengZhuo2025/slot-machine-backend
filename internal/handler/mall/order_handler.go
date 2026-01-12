// Package mall 提供商城相关的 HTTP Handler
package mall

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
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
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req mallService.CreateMallOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), userID, &req)
	handler.MustSucceed(c, err, order)
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
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req mallService.CreateFromCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	order, err := h.orderService.CreateOrderFromCart(c.Request.Context(), userID, &req)
	handler.MustSucceed(c, err, order)
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
	userID, orderID, ok := handler.RequireUserAndParseID(c, "订单")
	if !ok {
		return
	}

	order, err := h.orderService.GetOrderDetail(c.Request.Context(), userID, orderID)
	handler.MustSucceed(c, err, order)
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
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	status := c.Query("status")
	p := handler.BindPagination(c)

	orders, total, err := h.orderService.GetUserOrders(c.Request.Context(), userID, status, p.Page, p.PageSize)
	handler.MustSucceedPage(c, err, orders, total, p.Page, p.PageSize)
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
	userID, orderID, ok := handler.RequireUserAndParseID(c, "订单")
	if !ok {
		return
	}

	reason := c.Query("reason")

	handler.MustSucceed(c, h.orderService.CancelOrder(c.Request.Context(), userID, orderID, reason), nil)
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
	userID, orderID, ok := handler.RequireUserAndParseID(c, "订单")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.orderService.ConfirmReceive(c.Request.Context(), userID, orderID), nil)
}
