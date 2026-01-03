// Package device 提供设备服务
package device

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// DeviceService 设备服务
type DeviceService struct {
	db         *gorm.DB
	deviceRepo *repository.DeviceRepository
	venueRepo  *repository.VenueRepository
}

// NewDeviceService 创建设备服务
func NewDeviceService(
	db *gorm.DB,
	deviceRepo *repository.DeviceRepository,
	venueRepo *repository.VenueRepository,
) *DeviceService {
	return &DeviceService{
		db:         db,
		deviceRepo: deviceRepo,
		venueRepo:  venueRepo,
	}
}

// DeviceInfo 设备信息（用户端）
type DeviceInfo struct {
	ID             int64        `json:"id"`
	DeviceNo       string       `json:"device_no"`
	Name           string       `json:"name"`
	Type           string       `json:"type"`
	ProductName    string       `json:"product_name"`
	ProductImage   *string      `json:"product_image,omitempty"`
	SlotCount      int          `json:"slot_count"`
	AvailableSlots int          `json:"available_slots"`
	OnlineStatus   int8         `json:"online_status"`
	RentalStatus   int8         `json:"rental_status"`
	Venue          *VenueInfo   `json:"venue,omitempty"`
	Pricings       []PricingInfo `json:"pricings,omitempty"`
}

// VenueInfo 场地信息
type VenueInfo struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Province string   `json:"province"`
	City     string   `json:"city"`
	District string   `json:"district"`
	Address  string   `json:"address"`
	Longitude *float64 `json:"longitude,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
}

// PricingInfo 定价信息
type PricingInfo struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Duration      int      `json:"duration"`
	DurationUnit  string   `json:"duration_unit"`
	Price         float64  `json:"price"`
	OriginalPrice *float64 `json:"original_price,omitempty"`
	Deposit       float64  `json:"deposit"`
	IsDefault     bool     `json:"is_default"`
}

// GetDeviceByQRCode 根据二维码获取设备信息
func (s *DeviceService) GetDeviceByQRCode(ctx context.Context, qrCode string) (*DeviceInfo, error) {
	device, err := s.deviceRepo.GetByQRCodeWithVenue(ctx, qrCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrDeviceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if device.Status != models.DeviceStatusActive {
		return nil, errors.ErrDeviceDisabled
	}

	// 获取定价信息
	pricings, err := s.deviceRepo.GetPricingsByDevice(ctx, device.ID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toDeviceInfo(device, pricings), nil
}

// GetDeviceByNo 根据设备编号获取设备信息
func (s *DeviceService) GetDeviceByNo(ctx context.Context, deviceNo string) (*DeviceInfo, error) {
	device, err := s.deviceRepo.GetByDeviceNoWithVenue(ctx, deviceNo)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrDeviceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if device.Status != models.DeviceStatusActive {
		return nil, errors.ErrDeviceDisabled
	}

	pricings, err := s.deviceRepo.GetPricingsByDevice(ctx, device.ID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toDeviceInfo(device, pricings), nil
}

// GetDeviceByID 根据 ID 获取设备信息
func (s *DeviceService) GetDeviceByID(ctx context.Context, deviceID int64) (*DeviceInfo, error) {
	device, err := s.deviceRepo.GetByIDWithVenue(ctx, deviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrDeviceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if device.Status != models.DeviceStatusActive {
		return nil, errors.ErrDeviceDisabled
	}

	pricings, err := s.deviceRepo.GetPricingsByDevice(ctx, device.ID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toDeviceInfo(device, pricings), nil
}

// CheckDeviceAvailable 检查设备是否可租借
func (s *DeviceService) CheckDeviceAvailable(ctx context.Context, deviceID int64) error {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrDeviceNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if device.Status != models.DeviceStatusActive {
		return errors.ErrDeviceDisabled
	}

	if device.OnlineStatus != models.DeviceOnline {
		return errors.ErrDeviceOffline
	}

	if device.AvailableSlots <= 0 {
		return errors.ErrDeviceNoSlot
	}

	return nil
}

// GetPricing 获取定价信息
func (s *DeviceService) GetPricing(ctx context.Context, pricingID int64) (*PricingInfo, error) {
	pricing, err := s.deviceRepo.GetPricingByID(ctx, pricingID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrPricingNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if pricing.Status != models.RentalPricingStatusActive {
		return nil, errors.ErrPricingNotFound
	}

	return &PricingInfo{
		ID:            pricing.ID,
		Name:          pricing.Name,
		Duration:      pricing.Duration,
		DurationUnit:  pricing.DurationUnit,
		Price:         pricing.Price,
		OriginalPrice: pricing.OriginalPrice,
		Deposit:       pricing.Deposit,
		IsDefault:     pricing.IsDefault,
	}, nil
}

// GetDevicePricings 获取设备的定价列表
func (s *DeviceService) GetDevicePricings(ctx context.Context, deviceID int64) ([]PricingInfo, error) {
	pricings, err := s.deviceRepo.GetPricingsByDevice(ctx, deviceID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]PricingInfo, len(pricings))
	for i, p := range pricings {
		result[i] = PricingInfo{
			ID:            p.ID,
			Name:          p.Name,
			Duration:      p.Duration,
			DurationUnit:  p.DurationUnit,
			Price:         p.Price,
			OriginalPrice: p.OriginalPrice,
			Deposit:       p.Deposit,
			IsDefault:     p.IsDefault,
		}
	}

	return result, nil
}

// ListVenueDevices 获取场地下的设备列表
func (s *DeviceService) ListVenueDevices(ctx context.Context, venueID int64) ([]*DeviceInfo, error) {
	// 验证场地存在
	venue, err := s.venueRepo.GetByID(ctx, venueID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrVenueNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if venue.Status != models.VenueStatusActive {
		return nil, errors.ErrVenueDisabled
	}

	// 只获取正常状态的设备
	status := int8(models.DeviceStatusActive)
	devices, err := s.deviceRepo.ListByVenue(ctx, venueID, &status)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*DeviceInfo, len(devices))
	for i, d := range devices {
		result[i] = &DeviceInfo{
			ID:             d.ID,
			DeviceNo:       d.DeviceNo,
			Name:           d.Name,
			Type:           d.Type,
			ProductName:    d.ProductName,
			ProductImage:   d.ProductImage,
			SlotCount:      d.SlotCount,
			AvailableSlots: d.AvailableSlots,
			OnlineStatus:   d.OnlineStatus,
			RentalStatus:   d.RentalStatus,
		}
	}

	return result, nil
}

// UpdateDeviceHeartbeat 更新设备心跳
func (s *DeviceService) UpdateDeviceHeartbeat(ctx context.Context, deviceNo string, data *HeartbeatData) error {
	device, err := s.deviceRepo.GetByDeviceNo(ctx, deviceNo)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrDeviceNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	now := time.Now()
	fields := map[string]interface{}{
		"last_heartbeat_at": now,
		"online_status":     models.DeviceOnline,
	}

	if data.SignalStrength != nil {
		fields["signal_strength"] = *data.SignalStrength
	}
	if data.BatteryLevel != nil {
		fields["battery_level"] = *data.BatteryLevel
	}
	if data.Temperature != nil {
		fields["temperature"] = *data.Temperature
	}
	if data.Humidity != nil {
		fields["humidity"] = *data.Humidity
	}
	if data.FirmwareVersion != nil {
		fields["firmware_version"] = *data.FirmwareVersion
	}

	// 如果之前是离线状态，记录上线时间
	if device.OnlineStatus == models.DeviceOffline {
		fields["last_online_at"] = now
		// 记录上线日志
		_ = s.deviceRepo.CreateLog(ctx, &models.DeviceLog{
			DeviceID:     device.ID,
			Type:         models.DeviceLogTypeOnline,
			OperatorType: stringPtr(models.DeviceLogOperatorSystem),
		})
	}

	return s.deviceRepo.UpdateHeartbeat(ctx, device.ID, fields)
}

// HeartbeatData 心跳数据
type HeartbeatData struct {
	SignalStrength  *int     `json:"signal_strength,omitempty"`
	BatteryLevel    *int     `json:"battery_level,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	Humidity        *float64 `json:"humidity,omitempty"`
	FirmwareVersion *string  `json:"firmware_version,omitempty"`
	LockStatus      *int8    `json:"lock_status,omitempty"`
}

// SetDeviceOffline 设置设备离线
func (s *DeviceService) SetDeviceOffline(ctx context.Context, deviceID int64) error {
	now := time.Now()
	fields := map[string]interface{}{
		"online_status":   models.DeviceOffline,
		"last_offline_at": now,
	}

	if err := s.deviceRepo.UpdateFields(ctx, deviceID, fields); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	// 记录离线日志
	_ = s.deviceRepo.CreateLog(ctx, &models.DeviceLog{
		DeviceID:     deviceID,
		Type:         models.DeviceLogTypeOffline,
		OperatorType: stringPtr(models.DeviceLogOperatorSystem),
	})

	return nil
}

// toDeviceInfo 转换为设备信息
func (s *DeviceService) toDeviceInfo(device *models.Device, pricings []*models.RentalPricing) *DeviceInfo {
	info := &DeviceInfo{
		ID:             device.ID,
		DeviceNo:       device.DeviceNo,
		Name:           device.Name,
		Type:           device.Type,
		ProductName:    device.ProductName,
		ProductImage:   device.ProductImage,
		SlotCount:      device.SlotCount,
		AvailableSlots: device.AvailableSlots,
		OnlineStatus:   device.OnlineStatus,
		RentalStatus:   device.RentalStatus,
	}

	if device.Venue != nil {
		info.Venue = &VenueInfo{
			ID:        device.Venue.ID,
			Name:      device.Venue.Name,
			Type:      device.Venue.Type,
			Province:  device.Venue.Province,
			City:      device.Venue.City,
			District:  device.Venue.District,
			Address:   device.Venue.Address,
			Longitude: device.Venue.Longitude,
			Latitude:  device.Venue.Latitude,
		}
	}

	if len(pricings) > 0 {
		info.Pricings = make([]PricingInfo, len(pricings))
		for i, p := range pricings {
			info.Pricings[i] = PricingInfo{
				ID:            p.ID,
				Name:          p.Name,
				Duration:      p.Duration,
				DurationUnit:  p.DurationUnit,
				Price:         p.Price,
				OriginalPrice: p.OriginalPrice,
				Deposit:       p.Deposit,
				IsDefault:     p.IsDefault,
			}
		}
	}

	return info
}

func stringPtr(s string) *string {
	return &s
}
