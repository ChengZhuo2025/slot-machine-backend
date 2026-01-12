// Package order 提供订单相关的 HTTP Handler
package order

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	orderService "github.com/dumeirei/smart-locker-backend/internal/service/order"
)

// RefundHandler 退款处理器
type RefundHandler struct {
	refundService *orderService.RefundService
}

// NewRefundHandler 创建退款处理器
func NewRefundHandler(refundSvc *orderService.RefundService) *RefundHandler {
	return &RefundHandler{
		refundService: refundSvc,
	}
}

// CreateRefund 创建退款申请
// @Summary 创建退款申请
// @Tags 退款
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body orderService.CreateRefundRequest true "请求参数"
// @Success 200 {object} response.Response{data=orderService.RefundInfo}
// @Router /api/v1/refunds [post]
func (h *RefundHandler) CreateRefund(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req orderService.CreateRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	refund, err := h.refundService.CreateRefund(c.Request.Context(), userID, &req)
	handler.MustSucceed(c, err, refund)
}

// GetRefundDetail 获取退款详情
// @Summary 获取退款详情
// @Tags 退款
// @Produce json
// @Security Bearer
// @Param id path int true "退款ID"
// @Success 200 {object} response.Response{data=orderService.RefundInfo}
// @Router /api/v1/refunds/{id} [get]
func (h *RefundHandler) GetRefundDetail(c *gin.Context) {
	userID, refundID, ok := handler.RequireUserAndParseID(c, "退款")
	if !ok {
		return
	}

	refund, err := h.refundService.GetRefundDetail(c.Request.Context(), userID, refundID)
	handler.MustSucceed(c, err, refund)
}

// GetRefunds 获取退款列表
// @Summary 获取退款列表
// @Tags 退款
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=orderService.RefundListResponse}
// @Router /api/v1/refunds [get]
func (h *RefundHandler) GetRefunds(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	result, err := h.refundService.GetUserRefunds(c.Request.Context(), userID, p.Page, p.PageSize)
	handler.MustSucceed(c, err, result)
}

// CancelRefund 取消退款申请
// @Summary 取消退款申请
// @Tags 退款
// @Produce json
// @Security Bearer
// @Param id path int true "退款ID"
// @Success 200 {object} response.Response
// @Router /api/v1/refunds/{id}/cancel [post]
func (h *RefundHandler) CancelRefund(c *gin.Context) {
	userID, refundID, ok := handler.RequireUserAndParseID(c, "退款")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.refundService.CancelRefund(c.Request.Context(), userID, refundID), nil)
}
