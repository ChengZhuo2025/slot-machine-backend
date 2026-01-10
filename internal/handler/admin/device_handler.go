// Package admin 提供管理员相关的 HTTP Handler
package admin

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// DeviceHandler 设备管理处理器
type DeviceHandler struct {
	deviceService *adminService.DeviceAdminService
}

// NewDeviceHandler 创建设备管理处理器
func NewDeviceHandler(deviceSvc *adminService.DeviceAdminService) *DeviceHandler {
	return &DeviceHandler{
		deviceService: deviceSvc,
	}
}

// Create 创建设备
// @Summary 创建设备
// @Tags 设备管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateDeviceRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Device}
// @Router /admin/devices [post]
func (h *DeviceHandler) Create(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req adminService.CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	device, err := h.deviceService.CreateDevice(c.Request.Context(), &req, adminID)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrDeviceNoExists):
			response.BadRequest(c, "设备编号已存在")
		case errors.Is(err, adminService.ErrVenueNotFound):
			response.BadRequest(c, "场地不存在")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, device)
}

// Update 更新设备
// @Summary 更新设备
// @Tags 设备管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Param request body adminService.UpdateDeviceRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/devices/{id} [put]
func (h *DeviceHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的设备ID")
		return
	}

	var req adminService.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	err = h.deviceService.UpdateDevice(c.Request.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrDeviceNotFound):
			response.NotFound(c, "设备不存在")
		case errors.Is(err, adminService.ErrVenueNotFound):
			response.BadRequest(c, "场地不存在")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// DeviceUpdateStatusRequest 更新设备状态请求
type DeviceUpdateStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1 2 3"`
}

// UpdateStatus 更新设备状态
// @Summary 更新设备状态
// @Tags 设备管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Param request body DeviceUpdateStatusRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/devices/{id}/status [put]
func (h *DeviceHandler) UpdateStatus(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的设备ID")
		return
	}

	var req DeviceUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	err = h.deviceService.UpdateDeviceStatus(c.Request.Context(), id, req.Status, adminID)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrDeviceNotFound):
			response.NotFound(c, "设备不存在")
		case errors.Is(err, adminService.ErrDeviceInUse):
			response.BadRequest(c, "设备正在使用中，无法禁用")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// Get 获取设备详情
// @Summary 获取设备详情
// @Tags 设备管理
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Success 200 {object} response.Response{data=adminService.DeviceInfo}
// @Router /admin/devices/{id} [get]
func (h *DeviceHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的设备ID")
		return
	}

	device, err := h.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, adminService.ErrDeviceNotFound) {
			response.NotFound(c, "设备不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, device)
}

// List 获取设备列表
// @Summary 获取设备列表
// @Tags 设备管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param venue_id query int false "场地ID"
// @Param device_no query string false "设备编号"
// @Param type query string false "设备类型"
// @Param status query int false "设备状态"
// @Param online_status query int false "在线状态"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /admin/devices [get]
func (h *DeviceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filters := make(map[string]interface{})
	if venueIDStr := c.Query("venue_id"); venueIDStr != "" {
		if venueID, err := strconv.ParseInt(venueIDStr, 10, 64); err == nil {
			filters["venue_id"] = venueID
		}
	}
	if deviceNo := c.Query("device_no"); deviceNo != "" {
		filters["device_no"] = deviceNo
	}
	if deviceType := c.Query("type"); deviceType != "" {
		filters["type"] = deviceType
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			filters["status"] = int8(status)
		}
	}
	if onlineStatusStr := c.Query("online_status"); onlineStatusStr != "" {
		if onlineStatus, err := strconv.ParseInt(onlineStatusStr, 10, 8); err == nil {
			filters["online_status"] = int8(onlineStatus)
		}
	}

	devices, total, err := h.deviceService.ListDevices(c.Request.Context(), offset, pageSize, filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithPage(c, devices, total, page, pageSize)
}

// RemoteUnlock 远程开锁
// @Summary 远程开锁
// @Tags 设备管理
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Success 200 {object} response.Response
// @Router /admin/devices/{id}/unlock [post]
func (h *DeviceHandler) RemoteUnlock(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的设备ID")
		return
	}

	err = h.deviceService.RemoteUnlock(c.Request.Context(), id, adminID)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrDeviceNotFound):
			response.NotFound(c, "设备不存在")
		case errors.Is(err, adminService.ErrDeviceOffline):
			response.BadRequest(c, "设备离线，无法开锁")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// RemoteLock 远程锁定
// @Summary 远程锁定
// @Tags 设备管理
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Success 200 {object} response.Response
// @Router /admin/devices/{id}/lock [post]
func (h *DeviceHandler) RemoteLock(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的设备ID")
		return
	}

	err = h.deviceService.RemoteLock(c.Request.Context(), id, adminID)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrDeviceNotFound):
			response.NotFound(c, "设备不存在")
		case errors.Is(err, adminService.ErrDeviceOffline):
			response.BadRequest(c, "设备离线，无法锁定")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// GetLogs 获取设备日志
// @Summary 获取设备日志
// @Tags 设备管理
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param type query string false "日志类型"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /admin/devices/{id}/logs [get]
func (h *DeviceHandler) GetLogs(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的设备ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filters := make(map[string]interface{})
	if logType := c.Query("type"); logType != "" {
		filters["type"] = logType
	}

	logs, total, err := h.deviceService.GetDeviceLogs(c.Request.Context(), deviceID, offset, pageSize, filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithPage(c, logs, total, page, pageSize)
}

// CreateMaintenance 创建维护记录
// @Summary 创建维护记录
// @Tags 设备管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateMaintenanceRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.DeviceMaintenance}
// @Router /admin/devices/maintenance [post]
func (h *DeviceHandler) CreateMaintenance(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req adminService.CreateMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	maintenance, err := h.deviceService.CreateMaintenance(c.Request.Context(), &req, adminID)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrDeviceNotFound):
			response.NotFound(c, "设备不存在")
		case errors.Is(err, adminService.ErrDeviceInUse):
			response.BadRequest(c, "设备正在使用中，无法开始维护")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, maintenance)
}

// CompleteMaintenance 完成维护
// @Summary 完成维护
// @Tags 设备管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "维护记录ID"
// @Param request body adminService.CompleteMaintenanceRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /admin/devices/maintenance/{id}/complete [post]
func (h *DeviceHandler) CompleteMaintenance(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的维护记录ID")
		return
	}

	var req adminService.CompleteMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	err = h.deviceService.CompleteMaintenance(c.Request.Context(), id, &req, adminID)
	if err != nil {
		switch {
		case errors.Is(err, adminService.ErrMaintenanceNotFound):
			response.NotFound(c, "维护记录不存在")
		case errors.Is(err, adminService.ErrMaintenanceCompleted):
			response.BadRequest(c, "维护已完成")
		default:
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// ListMaintenance 获取维护记录列表
// @Summary 获取维护记录列表
// @Tags 设备管理
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param device_id query int false "设备ID"
// @Param type query string false "维护类型"
// @Param status query int false "状态"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /admin/devices/maintenance [get]
func (h *DeviceHandler) ListMaintenance(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filters := make(map[string]interface{})
	if deviceIDStr := c.Query("device_id"); deviceIDStr != "" {
		if deviceID, err := strconv.ParseInt(deviceIDStr, 10, 64); err == nil {
			filters["device_id"] = deviceID
		}
	}
	if maintenanceType := c.Query("type"); maintenanceType != "" {
		filters["type"] = maintenanceType
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			filters["status"] = int8(status)
		}
	}

	maintenances, total, err := h.deviceService.GetMaintenanceRecords(c.Request.Context(), offset, pageSize, filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithPage(c, maintenances, total, page, pageSize)
}

// GetStatistics 获取设备统计
// @Summary 获取设备统计
// @Tags 设备管理
// @Produce json
// @Security Bearer
// @Param venue_id query int false "场地ID"
// @Success 200 {object} response.Response{data=adminService.DeviceStatistics}
// @Router /admin/devices/statistics [get]
func (h *DeviceHandler) GetStatistics(c *gin.Context) {
	filters := make(map[string]interface{})
	if venueIDStr := c.Query("venue_id"); venueIDStr != "" {
		if venueID, err := strconv.ParseInt(venueIDStr, 10, 64); err == nil {
			filters["venue_id"] = venueID
		}
	}

	stats, err := h.deviceService.GetDeviceStatistics(c.Request.Context(), filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
}

// RegisterRoutes 注册路由
func (h *DeviceHandler) RegisterRoutes(r *gin.RouterGroup) {
	devices := r.Group("/devices")
	{
		devices.POST("", h.Create)
		devices.GET("", h.List)
		devices.GET("/statistics", h.GetStatistics)
		devices.GET("/:id", h.Get)
		devices.PUT("/:id", h.Update)
		devices.PUT("/:id/status", h.UpdateStatus)
		devices.POST("/:id/unlock", h.RemoteUnlock)
		devices.POST("/:id/lock", h.RemoteLock)
		devices.GET("/:id/logs", h.GetLogs)

		// 维护记录
		devices.POST("/maintenance", h.CreateMaintenance)
		devices.GET("/maintenance", h.ListMaintenance)
		devices.POST("/maintenance/:id/complete", h.CompleteMaintenance)
	}
}
