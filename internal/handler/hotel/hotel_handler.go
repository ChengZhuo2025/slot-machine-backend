// Package hotel 提供酒店相关的 HTTP Handler
package hotel

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	hotelService "github.com/dumeirei/smart-locker-backend/internal/service/hotel"
)

// Handler 酒店处理器
type Handler struct {
	hotelService *hotelService.HotelService
}

// NewHandler 创建酒店处理器
func NewHandler(hotelSvc *hotelService.HotelService) *Handler {
	return &Handler{
		hotelService: hotelSvc,
	}
}

// GetHotelList 获取酒店列表
// @Summary 获取酒店列表
// @Tags 酒店
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param city query string false "城市"
// @Param district query string false "区县"
// @Param star_rating query int false "星级"
// @Param keyword query string false "关键词"
// @Param longitude query number false "经度"
// @Param latitude query number false "纬度"
// @Param radius_km query number false "搜索半径(公里)"
// @Success 200 {object} response.Response{data=[]hotelService.HotelInfo}
// @Router /api/v1/hotels [get]
func (h *Handler) GetHotelList(c *gin.Context) {
	var req hotelService.HotelListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	hotels, total, err := h.hotelService.GetHotelList(c.Request.Context(), &req)
	handler.MustSucceedPage(c, err, hotels, total, req.Page, req.PageSize)
}

// GetHotelDetail 获取酒店详情
// @Summary 获取酒店详情
// @Tags 酒店
// @Produce json
// @Param id path int true "酒店ID"
// @Success 200 {object} response.Response{data=hotelService.HotelInfo}
// @Router /api/v1/hotels/{id} [get]
func (h *Handler) GetHotelDetail(c *gin.Context) {
	hotelID, ok := handler.ParseID(c, "酒店")
	if !ok {
		return
	}

	hotel, err := h.hotelService.GetHotelDetail(c.Request.Context(), hotelID)
	handler.MustSucceed(c, err, hotel)
}

// GetRoomList 获取房间列表
// @Summary 获取房间列表
// @Tags 酒店
// @Produce json
// @Param id path int true "酒店ID"
// @Success 200 {object} response.Response{data=[]hotelService.RoomInfo}
// @Router /api/v1/hotels/{id}/rooms [get]
func (h *Handler) GetRoomList(c *gin.Context) {
	hotelID, ok := handler.ParseID(c, "酒店")
	if !ok {
		return
	}

	rooms, err := h.hotelService.GetRoomList(c.Request.Context(), hotelID)
	handler.MustSucceed(c, err, rooms)
}

// GetRoomDetail 获取房间详情
// @Summary 获取房间详情
// @Tags 酒店
// @Produce json
// @Param id path int true "房间ID"
// @Success 200 {object} response.Response{data=hotelService.RoomInfo}
// @Router /api/v1/rooms/{id} [get]
func (h *Handler) GetRoomDetail(c *gin.Context) {
	roomID, ok := handler.ParseID(c, "房间")
	if !ok {
		return
	}

	room, err := h.hotelService.GetRoomDetail(c.Request.Context(), roomID)
	handler.MustSucceed(c, err, room)
}

// CheckRoomAvailability 检查房间可用性
// @Summary 检查房间可用性
// @Tags 酒店
// @Produce json
// @Param id path int true "房间ID"
// @Param check_in query string true "入住时间"
// @Param check_out query string true "退房时间"
// @Success 200 {object} response.Response{data=bool}
// @Router /api/v1/rooms/{id}/availability [get]
func (h *Handler) CheckRoomAvailability(c *gin.Context) {
	roomID, ok := handler.ParseID(c, "房间")
	if !ok {
		return
	}

	var req struct {
		CheckIn  string `form:"check_in" binding:"required"`
		CheckOut string `form:"check_out" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "请提供入住和退房时间")
		return
	}

	checkIn, err := handler.ParseDateTime(req.CheckIn)
	if err != nil {
		response.BadRequest(c, "入住时间格式错误")
		return
	}

	checkOut, err := handler.ParseDateTime(req.CheckOut)
	if err != nil {
		response.BadRequest(c, "退房时间格式错误")
		return
	}

	available, err := h.hotelService.CheckRoomAvailability(c.Request.Context(), roomID, checkIn, checkOut)
	handler.MustSucceed(c, err, gin.H{"available": available})
}

// GetCities 获取城市列表
// @Summary 获取城市列表
// @Tags 酒店
// @Produce json
// @Success 200 {object} response.Response{data=[]string}
// @Router /api/v1/hotels/cities [get]
func (h *Handler) GetCities(c *gin.Context) {
	cities, err := h.hotelService.GetCities(c.Request.Context())
	handler.MustSucceed(c, err, cities)
}

// GetRoomTimeSlots 获取房间时段价格
// @Summary 获取房间时段价格
// @Tags 酒店
// @Produce json
// @Param id path int true "房间ID"
// @Success 200 {object} response.Response{data=[]hotelService.TimeSlotInfo}
// @Router /api/v1/rooms/{id}/time-slots [get]
func (h *Handler) GetRoomTimeSlots(c *gin.Context) {
	roomID, ok := handler.ParseID(c, "房间")
	if !ok {
		return
	}

	slots, err := h.hotelService.GetTimeSlotsByRoom(c.Request.Context(), roomID)
	handler.MustSucceed(c, err, slots)
}
