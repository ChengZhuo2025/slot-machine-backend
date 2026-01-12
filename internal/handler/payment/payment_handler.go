// Package payment 提供支付相关的 HTTP Handler
package payment

import (
	"io"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	paymentService "github.com/dumeirei/smart-locker-backend/internal/service/payment"
)

// Handler 支付处理器
type Handler struct {
	paymentService *paymentService.PaymentService
}

// NewHandler 创建支付处理器
func NewHandler(paymentSvc *paymentService.PaymentService) *Handler {
	return &Handler{
		paymentService: paymentSvc,
	}
}

// CreatePayment 创建支付
// @Summary 创建支付
// @Tags 支付
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body paymentService.CreatePaymentRequest true "请求参数"
// @Success 200 {object} response.Response{data=paymentService.CreatePaymentResponse}
// @Router /api/v1/payment [post]
func (h *Handler) CreatePayment(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req paymentService.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.paymentService.CreatePayment(c.Request.Context(), userID, &req)
	handler.MustSucceed(c, err, result)
}

// QueryPayment 查询支付状态
// @Summary 查询支付状态
// @Tags 支付
// @Produce json
// @Security Bearer
// @Param payment_no path string true "支付单号"
// @Success 200 {object} response.Response{data=paymentService.PaymentInfo}
// @Router /api/v1/payment/{payment_no} [get]
func (h *Handler) QueryPayment(c *gin.Context) {
	_, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	paymentNo := c.Param("payment_no")
	if paymentNo == "" {
		response.BadRequest(c, "支付单号不能为空")
		return
	}

	result, err := h.paymentService.QueryPayment(c.Request.Context(), paymentNo)
	handler.MustSucceed(c, err, result)
}

// CreateRefund 创建退款
// @Summary 创建退款
// @Tags 支付
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body paymentService.CreateRefundRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/payment/refund [post]
func (h *Handler) CreateRefund(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req paymentService.CreateRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	handler.MustSucceed(c, h.paymentService.CreateRefund(c.Request.Context(), userID, &req), nil)
}

// WechatPayCallback 微信支付回调
// @Summary 微信支付回调
// @Tags 支付
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/payment/callback/wechat [post]
func (h *Handler) WechatPayCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(500, gin.H{
			"code":    "FAIL",
			"message": "读取请求体失败",
		})
		return
	}

	if err := h.paymentService.HandlePaymentCallback(c.Request.Context(), body); err != nil {
		c.JSON(500, gin.H{
			"code":    "FAIL",
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    "SUCCESS",
		"message": "成功",
	})
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	payment := r.Group("/payment")
	{
		payment.POST("", h.CreatePayment)
		payment.GET("/:payment_no", h.QueryPayment)
		payment.POST("/refund", h.CreateRefund)
	}
}

// RegisterCallbackRoutes 注册回调路由（无需认证）
func (h *Handler) RegisterCallbackRoutes(r *gin.RouterGroup) {
	callback := r.Group("/payment/callback")
	{
		callback.POST("/wechat", h.WechatPayCallback)
	}
}
