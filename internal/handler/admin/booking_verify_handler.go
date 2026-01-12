// Package admin 提供管理员相关的 HTTP Handler
package admin

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	hotelService "github.com/dumeirei/smart-locker-backend/internal/service/hotel"
)

// BookingVerifyHandler 预订核销处理器
type BookingVerifyHandler struct {
	bookingService *hotelService.BookingService
}

// NewBookingVerifyHandler 创建预订核销处理器
func NewBookingVerifyHandler(bookingSvc *hotelService.BookingService) *BookingVerifyHandler {
	return &BookingVerifyHandler{
		bookingService: bookingSvc,
	}
}

// VerifyByCodeRequest 通过核销码核销请求
type VerifyByCodeRequest struct {
	VerificationCode string `json:"verification_code" binding:"required"`
}

// VerifyByCode 通过核销码核销
// @Summary 通过核销码核销预订
// @Description 酒店前台扫码核销，验证顾客的预订
// @Tags 预订核销
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body VerifyByCodeRequest true "请求参数"
// @Success 200 {object} response.Response{data=hotelService.BookingInfo}
// @Router /admin/bookings/verify [post]
func (h *BookingVerifyHandler) VerifyByCode(c *gin.Context) {
	adminID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	var req VerifyByCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请提供核销码")
		return
	}

	booking, err := h.bookingService.VerifyBooking(c.Request.Context(), req.VerificationCode, adminID)
	handler.MustSucceed(c, err, booking)
}

// VerifyByQRCode 通过二维码核销
// @Summary 通过二维码核销预订
// @Description 酒店前台扫描二维码核销，验证顾客的预订
// @Tags 预订核销
// @Produce json
// @Security Bearer
// @Param booking_no path string true "预订号"
// @Param code query string true "核销码"
// @Success 200 {object} response.Response{data=hotelService.BookingInfo}
// @Router /admin/hotel/verify/{booking_no} [get]
func (h *BookingVerifyHandler) VerifyByQRCode(c *gin.Context) {
	adminID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	bookingNo := c.Param("booking_no")
	verificationCode := c.Query("code")

	if bookingNo == "" || verificationCode == "" {
		response.BadRequest(c, "参数不完整")
		return
	}

	booking, err := h.bookingService.VerifyBooking(c.Request.Context(), verificationCode, adminID)
	if handler.HandleError(c, err) {
		return
	}

	// 验证预订号是否匹配
	if booking.BookingNo != bookingNo {
		response.BadRequest(c, "核销码与预订号不匹配")
		return
	}

	response.Success(c, booking)
}

// CompleteBookingRequest 完成预订请求
type CompleteBookingRequest struct {
	BookingID int64 `json:"booking_id" binding:"required"`
}

// CompleteBooking 完成预订
// @Summary 完成预订
// @Description 手动完成预订（退房）
// @Tags 预订核销
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "预订ID"
// @Success 200 {object} response.Response
// @Router /admin/bookings/{id}/complete [post]
func (h *BookingVerifyHandler) CompleteBooking(c *gin.Context) {
	_, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	bookingID, ok := handler.ParseID(c, "预订")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.bookingService.CompleteBooking(c.Request.Context(), bookingID), nil)
}

// RegisterRoutes 注册路由
func (h *BookingVerifyHandler) RegisterRoutes(r *gin.RouterGroup) {
	// 核销相关接口
	r.POST("/bookings/verify", h.VerifyByCode)
	r.GET("/hotel/verify/:booking_no", h.VerifyByQRCode)
	r.POST("/bookings/:id/complete", h.CompleteBooking)
}
