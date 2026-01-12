// Package hotel 提供酒店预订相关的 HTTP Handler
package hotel

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	hotelService "github.com/dumeirei/smart-locker-backend/internal/service/hotel"
)

// BookingHandler 预订处理器
type BookingHandler struct {
	bookingService *hotelService.BookingService
}

// NewBookingHandler 创建预订处理器
func NewBookingHandler(bookingSvc *hotelService.BookingService) *BookingHandler {
	return &BookingHandler{
		bookingService: bookingSvc,
	}
}

// CreateBookingRequest 创建预订请求
type CreateBookingRequest struct {
	RoomID        int64  `json:"room_id" binding:"required"`
	DurationHours int    `json:"duration_hours" binding:"required,min=1"`
	CheckInTime   string `json:"check_in_time" binding:"required"`
}

// CreateBooking 创建预订
// @Summary 创建预订
// @Tags 预订
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateBookingRequest true "请求参数"
// @Success 200 {object} response.Response{data=hotelService.BookingInfo}
// @Router /api/v1/bookings [post]
func (h *BookingHandler) CreateBooking(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 解析入住时间
	checkInTime, err := handler.ParseDateTime(req.CheckInTime)
	if err != nil {
		response.BadRequest(c, "入住时间格式错误")
		return
	}

	serviceReq := &hotelService.CreateBookingRequest{
		RoomID:        req.RoomID,
		DurationHours: req.DurationHours,
		CheckInTime:   checkInTime,
	}

	booking, err := h.bookingService.CreateBooking(c.Request.Context(), userID, serviceReq)
	handler.MustSucceed(c, err, booking)
}

// GetBookingDetail 获取预订详情
// @Summary 获取预订详情
// @Tags 预订
// @Produce json
// @Security Bearer
// @Param id path int true "预订ID"
// @Success 200 {object} response.Response{data=hotelService.BookingInfo}
// @Router /api/v1/bookings/{id} [get]
func (h *BookingHandler) GetBookingDetail(c *gin.Context) {
	userID, bookingID, ok := handler.RequireUserAndParseID(c, "预订")
	if !ok {
		return
	}

	booking, err := h.bookingService.GetBookingByID(c.Request.Context(), bookingID, userID)
	handler.MustSucceed(c, err, booking)
}

// GetBookingByNo 根据预订号获取预订
// @Summary 根据预订号获取预订
// @Tags 预订
// @Produce json
// @Security Bearer
// @Param booking_no path string true "预订号"
// @Success 200 {object} response.Response{data=hotelService.BookingInfo}
// @Router /api/v1/bookings/no/{booking_no} [get]
func (h *BookingHandler) GetBookingByNo(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	bookingNo := c.Param("booking_no")
	if bookingNo == "" {
		response.BadRequest(c, "预订号不能为空")
		return
	}

	booking, err := h.bookingService.GetBookingByNo(c.Request.Context(), bookingNo, userID)
	handler.MustSucceed(c, err, booking)
}

// GetMyBookings 获取我的预订列表
// @Summary 获取我的预订列表
// @Tags 预订
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param status query string false "状态"
// @Success 200 {object} response.Response{data=[]hotelService.BookingInfo}
// @Router /api/v1/bookings [get]
func (h *BookingHandler) GetMyBookings(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)
	status := c.Query("status")

	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	bookings, total, err := h.bookingService.GetUserBookings(c.Request.Context(), userID, p.Page, p.PageSize, statusPtr)
	handler.MustSucceedPage(c, err, bookings, total, p.Page, p.PageSize)
}

// CancelBooking 取消预订
// @Summary 取消预订
// @Tags 预订
// @Produce json
// @Security Bearer
// @Param id path int true "预订ID"
// @Success 200 {object} response.Response
// @Router /api/v1/bookings/{id}/cancel [post]
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	userID, bookingID, ok := handler.RequireUserAndParseID(c, "预订")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.bookingService.CancelBooking(c.Request.Context(), bookingID, userID), nil)
}

// UnlockByCode 使用开锁码开锁
// @Summary 使用开锁码开锁
// @Tags 预订
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body UnlockRequest true "请求参数"
// @Success 200 {object} response.Response{data=hotelService.BookingInfo}
// @Router /api/v1/bookings/unlock [post]
func (h *BookingHandler) UnlockByCode(c *gin.Context) {
	_, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req UnlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	booking, err := h.bookingService.UnlockByCode(c.Request.Context(), req.DeviceID, req.UnlockCode)
	handler.MustSucceed(c, err, booking)
}

// UnlockRequest 开锁请求
type UnlockRequest struct {
	DeviceID   int64  `json:"device_id" binding:"required"`
	UnlockCode string `json:"unlock_code" binding:"required"`
}
