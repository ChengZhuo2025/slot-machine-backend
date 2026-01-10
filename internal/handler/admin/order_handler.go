// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := &adminService.OrderListFilters{
		OrderNo: c.Query("order_no"),
		Type:    c.Query("type"),
		Status:  c.Query("status"),
	}

	if s := c.Query("user_id"); s != "" {
		filters.UserID, _ = strconv.ParseInt(s, 10, 64)
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

	orders, total, err := h.orderService.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessList(c, orders, total, page, pageSize)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单ID")
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单ID")
		return
	}

	var req CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.orderService.CancelOrder(c.Request.Context(), id, req.Reason); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单ID")
		return
	}

	var req ShipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.orderService.ShipOrder(c.Request.Context(), id, req.ExpressCompany, req.ExpressNo); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单ID")
		return
	}

	if err := h.orderService.ConfirmReceipt(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的订单ID")
		return
	}

	var req RemarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.orderService.AddRemark(c.Request.Context(), id, req.Remark); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetStatistics 获取订单统计
// @Summary 获取订单统计
// @Tags 管理-订单管理
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.OrderStatistics}
// @Router /api/v1/admin/orders/statistics [get]
func (h *OrderHandler) GetStatistics(c *gin.Context) {
	stats, err := h.orderService.GetStatistics(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
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
	filters := &adminService.OrderListFilters{
		OrderNo: c.Query("order_no"),
		Type:    c.Query("type"),
		Status:  c.Query("status"),
	}

	if s := c.Query("user_id"); s != "" {
		filters.UserID, _ = strconv.ParseInt(s, 10, 64)
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

	orders, err := h.orderService.ExportOrders(c.Request.Context(), filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, orders)
}
