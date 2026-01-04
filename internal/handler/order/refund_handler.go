// Package order 提供订单相关的 HTTP Handler
package order

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
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
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req orderService.CreateRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	refund, err := h.refundService.CreateRefund(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, refund)
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
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	refundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的退款ID")
		return
	}

	refund, err := h.refundService.GetRefundDetail(c.Request.Context(), userID, refundID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, refund)
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
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := h.refundService.GetUserRefunds(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
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
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	refundID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的退款ID")
		return
	}

	if err := h.refundService.CancelRefund(c.Request.Context(), userID, refundID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
