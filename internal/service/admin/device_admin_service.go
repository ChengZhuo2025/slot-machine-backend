// Package admin 提供管理员相关服务
package admin

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	"github.com/dumeirei/smart-locker-backend/internal/service/device"
)

// DeviceAdminService 设备管理服务
type DeviceAdminService struct {
	deviceRepo            *repository.DeviceRepository
	deviceLogRepo         *repository.DeviceLogRepository
	deviceMaintenanceRepo *repository.DeviceMaintenanceRepository
	venueRepo             *repository.VenueRepository
	mqttService           *device.MQTTService
}

// NewDeviceAdminService 创建设备管理服务
func NewDeviceAdminService(
	deviceRepo *repository.DeviceRepository,
	deviceLogRepo *repository.DeviceLogRepository,
	deviceMaintenanceRepo *repository.DeviceMaintenanceRepository,
	venueRepo *repository.VenueRepository,
	mqttService *device.MQTTService,
) *DeviceAdminService {
	return &DeviceAdminService{
		deviceRepo:            deviceRepo,
		deviceLogRepo:         deviceLogRepo,
		deviceMaintenanceRepo: deviceMaintenanceRepo,
		venueRepo:             venueRepo,
		mqttService:           mqttService,
	}
}

// 预定义错误
var (
	ErrDeviceNotFound       = errors.New("设备不存在")
	ErrDeviceNoExists       = errors.New("设备编号已存在")
	ErrVenueNotFound        = errors.New("场地不存在")
	ErrDeviceInUse          = errors.New("设备正在使用中")
	ErrDeviceOffline        = errors.New("设备离线")
	ErrMaintenanceNotFound  = errors.New("维护记录不存在")
	ErrMaintenanceCompleted = errors.New("维护已完成")
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	ID              int64      `json:"id"`
	DeviceNo        string     `json:"device_no"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Model           *string    `json:"model,omitempty"`
	VenueID         int64      `json:"venue_id"`
	VenueName       string     `json:"venue_name,omitempty"`
	QRCode          string     `json:"qr_code"`
	ProductName     string     `json:"product_name"`
	ProductImage    *string    `json:"product_image,omitempty"`
	SlotCount       int        `json:"slot_count"`
	AvailableSlots  int        `json:"available_slots"`
	OnlineStatus    int8       `json:"online_status"`
	LockStatus      int8       `json:"lock_status"`
	RentalStatus    int8       `json:"rental_status"`
	FirmwareVersion *string    `json:"firmware_version,omitempty"`
	NetworkType     string     `json:"network_type"`
	SignalStrength  *int       `json:"signal_strength,omitempty"`
	BatteryLevel    *int       `json:"battery_level,omitempty"`
	Temperature     *float64   `json:"temperature,omitempty"`
	Humidity        *float64   `json:"humidity,omitempty"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	Status          int8       `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
}

// CreateDeviceRequest 创建设备请求
type CreateDeviceRequest struct {
	DeviceNo     string  `json:"device_no" binding:"required,max=64"`
	Name         string  `json:"name" binding:"required,max=100"`
	Type         string  `json:"type" binding:"required,oneof=standard mini premium"`
	Model        *string `json:"model"`
	VenueID      int64   `json:"venue_id" binding:"required"`
	ProductName  string  `json:"product_name" binding:"required,max=100"`
	ProductImage *string `json:"product_image"`
	SlotCount    int     `json:"slot_count" binding:"min=1"`
	NetworkType  string  `json:"network_type" binding:"oneof=WiFi 4G Ethernet"`
}

// CreateDevice 创建设备
func (s *DeviceAdminService) CreateDevice(ctx context.Context, req *CreateDeviceRequest, operatorID int64) (*models.Device, error) {
	// 检查设备编号是否存在
	exists, err := s.deviceRepo.ExistsByDeviceNo(ctx, req.DeviceNo)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDeviceNoExists
	}

	// 检查场地是否存在
	_, err = s.venueRepo.GetByID(ctx, req.VenueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVenueNotFound
		}
		return nil, err
	}

	slotCount := req.SlotCount
	if slotCount == 0 {
		slotCount = 1
	}

	networkType := req.NetworkType
	if networkType == "" {
		networkType = "WiFi"
	}

	device := &models.Device{
		DeviceNo:       req.DeviceNo,
		Name:           req.Name,
		Type:           req.Type,
		Model:          req.Model,
		VenueID:        req.VenueID,
		QRCode:         "", // 后续通过二维码服务生成
		ProductName:    req.ProductName,
		ProductImage:   req.ProductImage,
		SlotCount:      slotCount,
		AvailableSlots: slotCount,
		OnlineStatus:   models.DeviceOffline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    networkType,
		Status:         models.DeviceStatusActive,
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, err
	}

	// 记录日志
	s.createDeviceLog(ctx, device.ID, models.DeviceLogTypeOnline, "设备创建", &operatorID, models.DeviceLogOperatorAdmin)

	return device, nil
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	Name         string  `json:"name" binding:"required,max=100"`
	Type         string  `json:"type" binding:"required,oneof=standard mini premium"`
	Model        *string `json:"model"`
	VenueID      int64   `json:"venue_id" binding:"required"`
	ProductName  string  `json:"product_name" binding:"required,max=100"`
	ProductImage *string `json:"product_image"`
	SlotCount    int     `json:"slot_count" binding:"min=1"`
	NetworkType  string  `json:"network_type" binding:"oneof=WiFi 4G Ethernet"`
}

// UpdateDevice 更新设备
func (s *DeviceAdminService) UpdateDevice(ctx context.Context, id int64, req *UpdateDeviceRequest) error {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDeviceNotFound
		}
		return err
	}

	// 检查场地是否存在
	if req.VenueID != device.VenueID {
		_, err = s.venueRepo.GetByID(ctx, req.VenueID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrVenueNotFound
			}
			return err
		}
	}

	device.Name = req.Name
	device.Type = req.Type
	device.Model = req.Model
	device.VenueID = req.VenueID
	device.ProductName = req.ProductName
	device.ProductImage = req.ProductImage
	device.SlotCount = req.SlotCount
	device.NetworkType = req.NetworkType

	return s.deviceRepo.Update(ctx, device)
}

// UpdateDeviceStatus 更新设备状态
func (s *DeviceAdminService) UpdateDeviceStatus(ctx context.Context, id int64, status int8, operatorID int64) error {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDeviceNotFound
		}
		return err
	}

	// 如果设备正在使用中，不能禁用
	if status == models.DeviceStatusDisabled && device.RentalStatus == models.DeviceRentalInUse {
		return ErrDeviceInUse
	}

	if err := s.deviceRepo.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	// 记录日志
	content := "设备状态更新"
	switch status {
	case models.DeviceStatusDisabled:
		content = "设备禁用"
	case models.DeviceStatusActive:
		content = "设备启用"
	case models.DeviceStatusMaintenance:
		content = "设备进入维护模式"
	case models.DeviceStatusFault:
		content = "设备标记为故障"
	}
	s.createDeviceLog(ctx, id, models.DeviceLogTypeError, content, &operatorID, models.DeviceLogOperatorAdmin)

	return nil
}

// GetDevice 获取设备详情
func (s *DeviceAdminService) GetDevice(ctx context.Context, id int64) (*DeviceInfo, error) {
	device, err := s.deviceRepo.GetByIDWithVenue(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	return s.toDeviceInfo(device), nil
}

// ListDevices 获取设备列表
func (s *DeviceAdminService) ListDevices(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*DeviceInfo, int64, error) {
	devices, total, err := s.deviceRepo.List(ctx, offset, limit, filters)
	if err != nil {
		return nil, 0, err
	}

	infos := make([]*DeviceInfo, 0, len(devices))
	for _, d := range devices {
		infos = append(infos, s.toDeviceInfo(d))
	}

	return infos, total, nil
}

// RemoteUnlock 远程开锁
func (s *DeviceAdminService) RemoteUnlock(ctx context.Context, id int64, operatorID int64) error {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDeviceNotFound
		}
		return err
	}

	// 检查设备是否在线
	if device.OnlineStatus != models.DeviceOnline {
		return ErrDeviceOffline
	}

	// 发送开锁指令
	if s.mqttService != nil {
		if _, err := s.mqttService.SendUnlockCommand(ctx, device.DeviceNo, nil); err != nil {
			return err
		}
	}

	// 更新锁状态
	if err := s.deviceRepo.UpdateLockStatus(ctx, id, models.DeviceUnlocked); err != nil {
		return err
	}

	// 记录日志
	s.createDeviceLog(ctx, id, models.DeviceLogTypeUnlock, "远程开锁", &operatorID, models.DeviceLogOperatorAdmin)

	return nil
}

// RemoteLock 远程锁定
func (s *DeviceAdminService) RemoteLock(ctx context.Context, id int64, operatorID int64) error {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDeviceNotFound
		}
		return err
	}

	// 检查设备是否在线
	if device.OnlineStatus != models.DeviceOnline {
		return ErrDeviceOffline
	}

	// 发送锁定指令
	if s.mqttService != nil {
		if _, err := s.mqttService.SendLockCommand(ctx, device.DeviceNo, nil); err != nil {
			return err
		}
	}

	// 更新锁状态
	if err := s.deviceRepo.UpdateLockStatus(ctx, id, models.DeviceLocked); err != nil {
		return err
	}

	// 记录日志
	s.createDeviceLog(ctx, id, models.DeviceLogTypeLock, "远程锁定", &operatorID, models.DeviceLogOperatorAdmin)

	return nil
}

// GetDeviceLogs 获取设备日志
func (s *DeviceAdminService) GetDeviceLogs(ctx context.Context, deviceID int64, offset, limit int, filters map[string]interface{}) ([]*models.DeviceLog, int64, error) {
	return s.deviceLogRepo.List(ctx, deviceID, offset, limit, filters)
}

// CreateMaintenanceRequest 创建维护记录请求
type CreateMaintenanceRequest struct {
	DeviceID     int64       `json:"device_id" binding:"required"`
	Type         string      `json:"type" binding:"required,oneof=repair clean replace inspect"`
	Description  string      `json:"description" binding:"required"`
	BeforeImages models.JSON `json:"before_images"`
	Cost         float64     `json:"cost"`
}

// CreateMaintenance 创建维护记录
func (s *DeviceAdminService) CreateMaintenance(ctx context.Context, req *CreateMaintenanceRequest, operatorID int64) (*models.DeviceMaintenance, error) {
	// 检查设备是否存在
	device, err := s.deviceRepo.GetByID(ctx, req.DeviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	// 如果设备正在使用中，不能开始维护
	if device.RentalStatus == models.DeviceRentalInUse {
		return nil, ErrDeviceInUse
	}

	now := time.Now()
	maintenance := &models.DeviceMaintenance{
		DeviceID:     req.DeviceID,
		Type:         req.Type,
		Description:  req.Description,
		BeforeImages: req.BeforeImages,
		Cost:         req.Cost,
		OperatorID:   operatorID,
		Status:       models.MaintenanceStatusInProgress,
		StartedAt:    now,
	}

	if err := s.deviceMaintenanceRepo.Create(ctx, maintenance); err != nil {
		return nil, err
	}

	// 更新设备状态为维护中
	if err := s.deviceRepo.UpdateStatus(ctx, req.DeviceID, models.DeviceStatusMaintenance); err != nil {
		return nil, err
	}

	// 记录日志
	s.createDeviceLog(ctx, req.DeviceID, models.DeviceLogTypeError, "开始维护: "+req.Description, &operatorID, models.DeviceLogOperatorAdmin)

	return maintenance, nil
}

// CompleteMaintenanceRequest 完成维护请求
type CompleteMaintenanceRequest struct {
	AfterImages models.JSON `json:"after_images"`
	Cost        float64     `json:"cost"`
}

// CompleteMaintenance 完成维护
func (s *DeviceAdminService) CompleteMaintenance(ctx context.Context, id int64, req *CompleteMaintenanceRequest, operatorID int64) error {
	maintenance, err := s.deviceMaintenanceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMaintenanceNotFound
		}
		return err
	}

	if maintenance.Status == models.MaintenanceStatusCompleted {
		return ErrMaintenanceCompleted
	}

	now := time.Now()
	maintenance.AfterImages = req.AfterImages
	maintenance.Cost = req.Cost
	maintenance.Status = models.MaintenanceStatusCompleted
	maintenance.CompletedAt = &now

	if err := s.deviceMaintenanceRepo.Update(ctx, maintenance); err != nil {
		return err
	}

	// 恢复设备状态为正常
	if err := s.deviceRepo.UpdateStatus(ctx, maintenance.DeviceID, models.DeviceStatusActive); err != nil {
		return err
	}

	// 记录日志
	s.createDeviceLog(ctx, maintenance.DeviceID, models.DeviceLogTypeError, "维护完成", &operatorID, models.DeviceLogOperatorAdmin)

	return nil
}

// GetMaintenanceRecords 获取维护记录
func (s *DeviceAdminService) GetMaintenanceRecords(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.DeviceMaintenance, int64, error) {
	return s.deviceMaintenanceRepo.List(ctx, offset, limit, filters)
}

// GetDeviceStatistics 获取设备统计
func (s *DeviceAdminService) GetDeviceStatistics(ctx context.Context, filters map[string]interface{}) (*DeviceStatistics, error) {
	devices, _, err := s.deviceRepo.List(ctx, 0, 10000, filters)
	if err != nil {
		return nil, err
	}

	stats := &DeviceStatistics{}
	for _, d := range devices {
		stats.Total++
		if d.OnlineStatus == models.DeviceOnline {
			stats.Online++
		} else {
			stats.Offline++
		}
		if d.RentalStatus == models.DeviceRentalInUse {
			stats.InUse++
		} else {
			stats.Free++
		}
		switch d.Status {
		case models.DeviceStatusActive:
			stats.Active++
		case models.DeviceStatusMaintenance:
			stats.Maintenance++
		case models.DeviceStatusFault:
			stats.Fault++
		case models.DeviceStatusDisabled:
			stats.Disabled++
		}
	}

	return stats, nil
}

// DeviceStatistics 设备统计
type DeviceStatistics struct {
	Total       int64 `json:"total"`
	Online      int64 `json:"online"`
	Offline     int64 `json:"offline"`
	InUse       int64 `json:"in_use"`
	Free        int64 `json:"free"`
	Active      int64 `json:"active"`
	Maintenance int64 `json:"maintenance"`
	Fault       int64 `json:"fault"`
	Disabled    int64 `json:"disabled"`
}

// toDeviceInfo 转换为设备信息
func (s *DeviceAdminService) toDeviceInfo(device *models.Device) *DeviceInfo {
	info := &DeviceInfo{
		ID:              device.ID,
		DeviceNo:        device.DeviceNo,
		Name:            device.Name,
		Type:            device.Type,
		Model:           device.Model,
		VenueID:         device.VenueID,
		QRCode:          device.QRCode,
		ProductName:     device.ProductName,
		ProductImage:    device.ProductImage,
		SlotCount:       device.SlotCount,
		AvailableSlots:  device.AvailableSlots,
		OnlineStatus:    device.OnlineStatus,
		LockStatus:      device.LockStatus,
		RentalStatus:    device.RentalStatus,
		FirmwareVersion: device.FirmwareVersion,
		NetworkType:     device.NetworkType,
		SignalStrength:  device.SignalStrength,
		BatteryLevel:    device.BatteryLevel,
		Temperature:     device.Temperature,
		Humidity:        device.Humidity,
		LastHeartbeatAt: device.LastHeartbeatAt,
		Status:          device.Status,
		CreatedAt:       device.CreatedAt,
	}

	if device.Venue != nil {
		info.VenueName = device.Venue.Name
	}

	return info
}

// createDeviceLog 创建设备日志
func (s *DeviceAdminService) createDeviceLog(ctx context.Context, deviceID int64, logType, content string, operatorID *int64, operatorType string) {
	log := &models.DeviceLog{
		DeviceID:     deviceID,
		Type:         logType,
		Content:      &content,
		OperatorID:   operatorID,
		OperatorType: &operatorType,
	}
	_ = s.deviceLogRepo.Create(ctx, log)
}
