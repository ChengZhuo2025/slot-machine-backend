// Package device 提供设备相关的 HTTP Handler
package device

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
)

// Handler 设备处理器
type Handler struct {
	deviceService *deviceService.DeviceService
	venueService  *deviceService.VenueService
}

// NewHandler 创建设备处理器
func NewHandler(
	deviceSvc *deviceService.DeviceService,
	venueSvc *deviceService.VenueService,
) *Handler {
	return &Handler{
		deviceService: deviceSvc,
		venueService:  venueSvc,
	}
}

// GetDeviceByQRCode 根据二维码获取设备信息
// @Summary 扫码获取设备信息
// @Tags 设备
// @Produce json
// @Param qr_code query string true "二维码内容"
// @Success 200 {object} response.Response{data=deviceService.DeviceInfo}
// @Router /api/v1/device/scan [get]
func (h *Handler) GetDeviceByQRCode(c *gin.Context) {
	qrCode := c.Query("qr_code")
	if qrCode == "" {
		response.BadRequest(c, "二维码内容不能为空")
		return
	}

	device, err := h.deviceService.GetDeviceByQRCode(c.Request.Context(), qrCode)
	handler.MustSucceed(c, err, device)
}

// GetDeviceByID 根据 ID 获取设备信息
// @Summary 获取设备详情
// @Tags 设备
// @Produce json
// @Param id path int true "设备ID"
// @Success 200 {object} response.Response{data=deviceService.DeviceInfo}
// @Router /api/v1/device/{id} [get]
func (h *Handler) GetDeviceByID(c *gin.Context) {
	deviceID, ok := handler.ParseID(c, "设备")
	if !ok {
		return
	}

	device, err := h.deviceService.GetDeviceByID(c.Request.Context(), deviceID)
	handler.MustSucceed(c, err, device)
}

// GetDevicePricings 获取设备定价列表
// @Summary 获取设备定价列表
// @Tags 设备
// @Produce json
// @Param id path int true "设备ID"
// @Success 200 {object} response.Response{data=[]deviceService.PricingInfo}
// @Router /api/v1/device/{id}/pricings [get]
func (h *Handler) GetDevicePricings(c *gin.Context) {
	deviceID, ok := handler.ParseID(c, "设备")
	if !ok {
		return
	}

	pricings, err := h.deviceService.GetDevicePricings(c.Request.Context(), deviceID)
	handler.MustSucceed(c, err, pricings)
}

// GetVenueByID 获取场地详情
// @Summary 获取场地详情
// @Tags 场地
// @Produce json
// @Param id path int true "场地ID"
// @Success 200 {object} response.Response{data=deviceService.VenueDetail}
// @Router /api/v1/venue/{id} [get]
func (h *Handler) GetVenueByID(c *gin.Context) {
	venueID, ok := handler.ParseID(c, "场地")
	if !ok {
		return
	}

	venue, err := h.venueService.GetVenueByID(c.Request.Context(), venueID)
	handler.MustSucceed(c, err, venue)
}

// GetVenueDevices 获取场地下的设备列表
// @Summary 获取场地设备列表
// @Tags 场地
// @Produce json
// @Param id path int true "场地ID"
// @Success 200 {object} response.Response{data=[]deviceService.DeviceInfo}
// @Router /api/v1/venue/{id}/devices [get]
func (h *Handler) GetVenueDevices(c *gin.Context) {
	venueID, ok := handler.ParseID(c, "场地")
	if !ok {
		return
	}

	devices, err := h.deviceService.ListVenueDevices(c.Request.Context(), venueID)
	handler.MustSucceed(c, err, devices)
}

// ListNearbyVenues 获取附近场地列表
// @Summary 获取附近场地
// @Tags 场地
// @Produce json
// @Param longitude query number true "经度"
// @Param latitude query number true "纬度"
// @Param radius query number false "搜索半径(公里)" default(5)
// @Param limit query int false "返回数量" default(20)
// @Success 200 {object} response.Response{data=[]deviceService.VenueListItem}
// @Router /api/v1/venue/nearby [get]
func (h *Handler) ListNearbyVenues(c *gin.Context) {
	longitude, err := strconv.ParseFloat(c.Query("longitude"), 64)
	if err != nil {
		response.BadRequest(c, "无效的经度")
		return
	}

	latitude, err := strconv.ParseFloat(c.Query("latitude"), 64)
	if err != nil {
		response.BadRequest(c, "无效的纬度")
		return
	}

	radiusKm := 5.0
	if radiusStr := c.Query("radius"); radiusStr != "" {
		if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 {
			radiusKm = r
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	venues, err := h.venueService.ListNearbyVenues(c.Request.Context(), longitude, latitude, radiusKm, limit)
	handler.MustSucceed(c, err, venues)
}

// ListVenuesByCity 获取城市场地列表
// @Summary 获取城市场地列表
// @Tags 场地
// @Produce json
// @Param city query string true "城市"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/venue/city [get]
func (h *Handler) ListVenuesByCity(c *gin.Context) {
	city := c.Query("city")
	if city == "" {
		response.BadRequest(c, "城市不能为空")
		return
	}

	p := handler.BindPagination(c)

	venues, total, err := h.venueService.ListVenuesByCity(c.Request.Context(), city, p.GetOffset(), p.GetLimit())
	handler.MustSucceedPage(c, err, venues, total, p.Page, p.PageSize)
}

// SearchVenues 搜索场地
// @Summary 搜索场地
// @Tags 场地
// @Produce json
// @Param keyword query string false "搜索关键词"
// @Param city query string false "城市"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/venue/search [get]
func (h *Handler) SearchVenues(c *gin.Context) {
	keyword := c.Query("keyword")
	city := c.Query("city")

	p := handler.BindPagination(c)

	venues, total, err := h.venueService.SearchVenues(c.Request.Context(), keyword, city, p.GetOffset(), p.GetLimit())
	handler.MustSucceedPage(c, err, venues, total, p.Page, p.PageSize)
}

// GetCities 获取城市列表
// @Summary 获取城市列表
// @Tags 场地
// @Produce json
// @Success 200 {object} response.Response{data=[]string}
// @Router /api/v1/venue/cities [get]
func (h *Handler) GetCities(c *gin.Context) {
	cities, err := h.venueService.GetCities(c.Request.Context())
	handler.MustSucceed(c, err, cities)
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// 设备相关
	device := r.Group("/device")
	{
		device.GET("/scan", h.GetDeviceByQRCode)
		device.GET("/:id", h.GetDeviceByID)
		device.GET("/:id/pricings", h.GetDevicePricings)
	}

	// 场地相关
	venue := r.Group("/venue")
	{
		venue.GET("/nearby", h.ListNearbyVenues)
		venue.GET("/city", h.ListVenuesByCity)
		venue.GET("/cities", h.GetCities)
		venue.GET("/search", h.SearchVenues)
		venue.GET("/:id", h.GetVenueByID)
		venue.GET("/:id/devices", h.GetVenueDevices)
	}
}
