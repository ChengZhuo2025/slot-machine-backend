// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// OrderHandler 订单管理处理器
type OrderHandler struct {
	orderService *adminService.OrderAdminService
}

// NewOrderHandler 创建订单管理处理器
func NewOrderHandler(orderService *adminService.OrderAdminService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// List 获取订单列表
// @Summary 获取订单列表
// @Tags 管理-订单管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param order_no query string false "订单号"
// @Param user_id query int false "用户ID"
// @Param type query string false "订单类型"
// @Param status query string false "订单状态"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=response.ListData}
// @Router /api/v1/admin/orders [get]
func (h *OrderHandler) List(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	filters := &adminService.OrderListFilters{
		OrderNo: c.Query("order_no"),
		Type:    c.Query("type"),
		Status:  c.Query("status"),
	}

	if s := c.Query("user_id"); s != "" {
		if userID, err := strconv.ParseInt(s, 10, 64); err == nil {
			filters.UserID = userID
		}
	}

	startDate, endDate, ok := handler.ParseQueryDateRange(c)
	if !ok {
		return
	}
	filters.StartDate = startDate
	filters.EndDate = endDate

	orders, total, err := h.orderService.List(c.Request.Context(), p.Page, p.PageSize, filters)
	handler.MustSucceedPage(c, err, orders, total, p.Page, p.PageSize)
}

// GetByID 获取订单详情
// @Summary 获取订单详情
// @Tags 管理-订单管理
// @Produce json
// @Security Bearer
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response{data=models.Order}
// @Router /api/v1/admin/orders/{id} [get]
func (h *OrderHandler) GetByID(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "订单")
	if !ok {
		return
	}

	order, err := h.orderService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "订单不存在")
		return
	}
	response.Success(c, order)
}

// GetByOrderNo 根据订单号获取订单
// @Summary 根据订单号获取订单
// @Tags 管理-订单管理
// @Produce json
// @Security Bearer
// @Param order_no path string true "订单号"
// @Success 200 {object} response.Response{data=models.Order}
// @Router /api/v1/admin/orders/no/{order_no} [get]
func (h *OrderHandler) GetByOrderNo(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	orderNo := c.Param("order_no")
	if orderNo == "" {
		response.BadRequest(c, "订单号不能为空")
		return
	}

	order, err := h.orderService.GetByOrderNo(c.Request.Context(), orderNo)
	if err != nil {
		response.NotFound(c, "订单不存在")
		return
	}
	response.Success(c, order)
}

// CancelRequest 取消订单请求
type CancelRequest struct {
	Reason string `json:"reason" binding:"required,max=255"`
}

// Cancel 取消订单
// @Summary 取消订单
// @Tags 管理-订单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "订单ID"
// @Param request body CancelRequest true "取消原因"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/orders/{id}/cancel [post]
func (h *OrderHandler) Cancel(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "订单")
	if !ok {
		return
	}

	var req CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	handler.MustSucceed(c, h.orderService.CancelOrder(c.Request.Context(), id, req.Reason), nil)
}

// ShipRequest 发货请求
type ShipRequest struct {
	ExpressCompany string `json:"express_company" binding:"required,max=50"`
	ExpressNo      string `json:"express_no" binding:"required,max=64"`
}

// Ship 发货
// @Summary 订单发货
// @Tags 管理-订单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "订单ID"
// @Param request body ShipRequest true "发货信息"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/orders/{id}/ship [post]
func (h *OrderHandler) Ship(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "订单")
	if !ok {
		return
	}

	var req ShipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	handler.MustSucceed(c, h.orderService.ShipOrder(c.Request.Context(), id, req.ExpressCompany, req.ExpressNo), nil)
}

// ConfirmReceipt 确认收货
// @Summary 确认收货
// @Tags 管理-订单管理
// @Produce json
// @Security Bearer
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/orders/{id}/confirm-receipt [post]
func (h *OrderHandler) ConfirmReceipt(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "订单")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.orderService.ConfirmReceipt(c.Request.Context(), id), nil)
}

// RemarkRequest 备注请求
type RemarkRequest struct {
	Remark string `json:"remark" binding:"required,max=255"`
}

// AddRemark 添加备注
// @Summary 添加订单备注
// @Tags 管理-订单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "订单ID"
// @Param request body RemarkRequest true "备注内容"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/orders/{id}/remark [post]
func (h *OrderHandler) AddRemark(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "订单")
	if !ok {
		return
	}

	var req RemarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	handler.MustSucceed(c, h.orderService.AddRemark(c.Request.Context(), id, req.Remark), nil)
}

// GetStatistics 获取订单统计
// @Summary 获取订单统计
// @Tags 管理-订单管理
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.OrderStatistics}
// @Router /api/v1/admin/orders/statistics [get]
func (h *OrderHandler) GetStatistics(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	stats, err := h.orderService.GetStatistics(c.Request.Context())
	handler.MustSucceed(c, err, stats)
}

// Export 导出订单
// @Summary 导出订单
// @Tags 管理-订单管理
// @Produce json
// @Security Bearer
// @Param order_no query string false "订单号"
// @Param user_id query int false "用户ID"
// @Param type query string false "订单类型"
// @Param status query string false "订单状态"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]models.Order}
// @Router /api/v1/admin/orders/export [get]
func (h *OrderHandler) Export(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	filters := &adminService.OrderListFilters{
		OrderNo: c.Query("order_no"),
		Type:    c.Query("type"),
		Status:  c.Query("status"),
	}

	if s := c.Query("user_id"); s != "" {
		if userID, err := strconv.ParseInt(s, 10, 64); err == nil {
			filters.UserID = userID
		}
	}

	startDate, endDate, ok := handler.ParseQueryDateRange(c)
	if !ok {
		return
	}
	filters.StartDate = startDate
	filters.EndDate = endDate

	orders, err := h.orderService.ExportOrders(c.Request.Context(), filters)
	handler.MustSucceed(c, err, orders)
}
