// Package admin 提供管理员相关的 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// HotelHandler 酒店管理处理器
type HotelHandler struct {
	hotelService *adminService.HotelAdminService
}

// NewHotelHandler 创建酒店管理处理器
func NewHotelHandler(hotelSvc *adminService.HotelAdminService) *HotelHandler {
	return &HotelHandler{
		hotelService: hotelSvc,
	}
}

// CreateHotel 创建酒店
// @Summary 创建酒店
// @Tags 酒店管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateHotelRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Hotel}
// @Router /admin/hotels [post]
func (h *HotelHandler) CreateHotel(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var req adminService.CreateHotelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	hotel, err := h.hotelService.CreateHotel(c.Request.Context(), &req)
	handler.MustSucceed(c, err, hotel)
}

// UpdateHotel 更新酒店
// @Summary 更新酒店
// @Tags 酒店管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "酒店ID"
// @Param request body adminService.UpdateHotelRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Hotel}
// @Router /admin/hotels/{id} [put]
func (h *HotelHandler) UpdateHotel(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "酒店")
	if !ok {
		return
	}

	var req adminService.UpdateHotelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	hotel, err := h.hotelService.UpdateHotel(c.Request.Context(), id, &req)
	handler.MustSucceed(c, err, hotel)
}

// GetHotel 获取酒店详情
// @Summary 获取酒店详情
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param id path int true "酒店ID"
// @Success 200 {object} response.Response{data=models.Hotel}
// @Router /admin/hotels/{id} [get]
func (h *HotelHandler) GetHotel(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "酒店")
	if !ok {
		return
	}

	hotel, err := h.hotelService.GetHotelByID(c.Request.Context(), id)
	handler.MustSucceed(c, err, hotel)
}

// ListHotels 获取酒店列表
// @Summary 获取酒店列表
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param name query string false "酒店名称"
// @Param city query string false "城市"
// @Param status query int false "状态"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /admin/hotels [get]
func (h *HotelHandler) ListHotels(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	filters := make(map[string]interface{})
	if name := c.Query("name"); name != "" {
		filters["name"] = name
	}
	if city := c.Query("city"); city != "" {
		filters["city"] = city
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			filters["status"] = int8(status)
		}
	}

	hotels, total, err := h.hotelService.GetHotelList(c.Request.Context(), p.Page, p.PageSize, filters)
	handler.MustSucceedPage(c, err, hotels, total, p.Page, p.PageSize)
}

// DeleteHotel 删除酒店
// @Summary 删除酒店
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param id path int true "酒店ID"
// @Success 200 {object} response.Response
// @Router /admin/hotels/{id} [delete]
func (h *HotelHandler) DeleteHotel(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "酒店")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.hotelService.DeleteHotel(c.Request.Context(), id), nil)
}

// HotelUpdateStatusRequest 更新酒店状态请求
type HotelUpdateStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// UpdateHotelStatus 更新酒店状态
// @Summary 更新酒店状态
// @Tags 酒店管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "酒店ID"
// @Param request body HotelUpdateStatusRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/hotels/{id}/status [put]
func (h *HotelHandler) UpdateHotelStatus(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "酒店")
	if !ok {
		return
	}

	var req HotelUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	handler.MustSucceed(c, h.hotelService.UpdateHotelStatus(c.Request.Context(), id, req.Status), nil)
}

// CreateRoom 创建房间
// @Summary 创建房间
// @Tags 酒店管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateRoomRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Room}
// @Router /admin/rooms [post]
func (h *HotelHandler) CreateRoom(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var req adminService.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	room, err := h.hotelService.CreateRoom(c.Request.Context(), &req)
	handler.MustSucceed(c, err, room)
}

// UpdateRoom 更新房间
// @Summary 更新房间
// @Tags 酒店管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "房间ID"
// @Param request body adminService.UpdateRoomRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Room}
// @Router /admin/rooms/{id} [put]
func (h *HotelHandler) UpdateRoom(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "房间")
	if !ok {
		return
	}

	var req adminService.UpdateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	room, err := h.hotelService.UpdateRoom(c.Request.Context(), id, &req)
	handler.MustSucceed(c, err, room)
}

// GetRoom 获取房间详情
// @Summary 获取房间详情
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param id path int true "房间ID"
// @Success 200 {object} response.Response{data=models.Room}
// @Router /admin/rooms/{id} [get]
func (h *HotelHandler) GetRoom(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "房间")
	if !ok {
		return
	}

	room, err := h.hotelService.GetRoomByID(c.Request.Context(), id)
	handler.MustSucceed(c, err, room)
}

// ListRooms 获取房间列表
// @Summary 获取房间列表
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param hotel_id query int false "酒店ID"
// @Param room_type query string false "房间类型"
// @Param status query int false "状态"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /admin/rooms [get]
func (h *HotelHandler) ListRooms(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	filters := make(map[string]interface{})
	if hotelIDStr := c.Query("hotel_id"); hotelIDStr != "" {
		if hotelID, err := strconv.ParseInt(hotelIDStr, 10, 64); err == nil {
			filters["hotel_id"] = hotelID
		}
	}
	if roomType := c.Query("room_type"); roomType != "" {
		filters["room_type"] = roomType
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			filters["status"] = int8(status)
		}
	}

	rooms, total, err := h.hotelService.GetRoomList(c.Request.Context(), p.Page, p.PageSize, filters)
	handler.MustSucceedPage(c, err, rooms, total, p.Page, p.PageSize)
}

// DeleteRoom 删除房间
// @Summary 删除房间
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param id path int true "房间ID"
// @Success 200 {object} response.Response
// @Router /admin/rooms/{id} [delete]
func (h *HotelHandler) DeleteRoom(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "房间")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.hotelService.DeleteRoom(c.Request.Context(), id), nil)
}

// CreateTimeSlot 创建时段价格
// @Summary 创建时段价格
// @Tags 酒店管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateTimeSlotRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.RoomTimeSlot}
// @Router /admin/time-slots [post]
func (h *HotelHandler) CreateTimeSlot(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var req adminService.CreateTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	slot, err := h.hotelService.CreateTimeSlot(c.Request.Context(), &req)
	handler.MustSucceed(c, err, slot)
}

// UpdateTimeSlotRequest 更新时段请求
type UpdateTimeSlotRequest struct {
	DurationHours *int     `json:"duration_hours"`
	Price         *float64 `json:"price"`
	StartTime     *string  `json:"start_time"`
	EndTime       *string  `json:"end_time"`
	Sort          *int     `json:"sort"`
	IsActive      *bool    `json:"is_active"`
}

// UpdateTimeSlot 更新时段价格
// @Summary 更新时段价格
// @Tags 酒店管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "时段ID"
// @Param request body UpdateTimeSlotRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/time-slots/{id} [put]
func (h *HotelHandler) UpdateTimeSlot(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "时段")
	if !ok {
		return
	}

	var req UpdateTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	fields := make(map[string]interface{})
	if req.DurationHours != nil {
		fields["duration_hours"] = *req.DurationHours
	}
	if req.Price != nil {
		fields["price"] = *req.Price
	}
	if req.StartTime != nil {
		fields["start_time"] = *req.StartTime
	}
	if req.EndTime != nil {
		fields["end_time"] = *req.EndTime
	}
	if req.Sort != nil {
		fields["sort"] = *req.Sort
	}
	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	handler.MustSucceed(c, h.hotelService.UpdateTimeSlot(c.Request.Context(), id, fields), nil)
}

// DeleteTimeSlot 删除时段
// @Summary 删除时段
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param id path int true "时段ID"
// @Success 200 {object} response.Response
// @Router /admin/time-slots/{id} [delete]
func (h *HotelHandler) DeleteTimeSlot(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "时段")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.hotelService.DeleteTimeSlot(c.Request.Context(), id), nil)
}

// ListBookings 获取预订列表
// @Summary 获取预订列表
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param hotel_id query int false "酒店ID"
// @Param room_id query int false "房间ID"
// @Param user_id query int false "用户ID"
// @Param status query string false "状态"
// @Param booking_no query string false "预订号"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /admin/bookings [get]
func (h *HotelHandler) ListBookings(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	filters := make(map[string]interface{})
	if hotelIDStr := c.Query("hotel_id"); hotelIDStr != "" {
		if hotelID, err := strconv.ParseInt(hotelIDStr, 10, 64); err == nil {
			filters["hotel_id"] = hotelID
		}
	}
	if roomIDStr := c.Query("room_id"); roomIDStr != "" {
		if roomID, err := strconv.ParseInt(roomIDStr, 10, 64); err == nil {
			filters["room_id"] = roomID
		}
	}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			filters["user_id"] = userID
		}
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if bookingNo := c.Query("booking_no"); bookingNo != "" {
		filters["booking_no"] = bookingNo
	}

	bookings, total, err := h.hotelService.GetBookingList(c.Request.Context(), p.Page, p.PageSize, filters)
	handler.MustSucceedPage(c, err, bookings, total, p.Page, p.PageSize)
}

// GetBooking 获取预订详情
// @Summary 获取预订详情
// @Tags 酒店管理
// @Produce json
// @Security Bearer
// @Param id path int true "预订ID"
// @Success 200 {object} response.Response{data=models.Booking}
// @Router /admin/bookings/{id} [get]
func (h *HotelHandler) GetBooking(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "预订")
	if !ok {
		return
	}

	booking, err := h.hotelService.GetBookingByID(c.Request.Context(), id)
	handler.MustSucceed(c, err, booking)
}

// RegisterRoutes 注册路由
func (h *HotelHandler) RegisterRoutes(r *gin.RouterGroup) {
	// 酒店管理
	hotels := r.Group("/hotels")
	{
		hotels.POST("", h.CreateHotel)
		hotels.GET("", h.ListHotels)
		hotels.GET("/:id", h.GetHotel)
		hotels.PUT("/:id", h.UpdateHotel)
		hotels.PUT("/:id/status", h.UpdateHotelStatus)
		hotels.DELETE("/:id", h.DeleteHotel)
	}

	// 房间管理
	rooms := r.Group("/rooms")
	{
		rooms.POST("", h.CreateRoom)
		rooms.GET("", h.ListRooms)
		rooms.GET("/:id", h.GetRoom)
		rooms.PUT("/:id", h.UpdateRoom)
		rooms.DELETE("/:id", h.DeleteRoom)
	}

	// 时段价格管理
	timeSlots := r.Group("/time-slots")
	{
		timeSlots.POST("", h.CreateTimeSlot)
		timeSlots.PUT("/:id", h.UpdateTimeSlot)
		timeSlots.DELETE("/:id", h.DeleteTimeSlot)
	}

	// 预订管理
	bookings := r.Group("/bookings")
	{
		bookings.GET("", h.ListBookings)
		bookings.GET("/:id", h.GetBooking)
	}
}
