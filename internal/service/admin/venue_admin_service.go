// Package admin 提供管理员相关服务
package admin

import (
	"context"
	"errors"

	"gorm.io/gorm"

	commonErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// VenueAdminService 场地管理服务
type VenueAdminService struct {
	venueRepo    *repository.VenueRepository
	merchantRepo *repository.MerchantRepository
	deviceRepo   *repository.DeviceRepository
}

// NewVenueAdminService 创建场地管理服务
func NewVenueAdminService(
	venueRepo *repository.VenueRepository,
	merchantRepo *repository.MerchantRepository,
	deviceRepo *repository.DeviceRepository,
) *VenueAdminService {
	return &VenueAdminService{
		venueRepo:    venueRepo,
		merchantRepo: merchantRepo,
		deviceRepo:   deviceRepo,
	}
}

// 预定义错误（使用 common/errors 包的 AppError）
var (
	venueNotFoundErr   = commonErrors.ErrVenueNotFound
	venueHasDevicesErr = commonErrors.ErrVenueHasDevices
)

// VenueInfo 场地信息
type VenueInfo struct {
	ID           int64   `json:"id"`
	MerchantID   int64   `json:"merchant_id"`
	MerchantName string  `json:"merchant_name,omitempty"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Province     string  `json:"province"`
	City         string  `json:"city"`
	District     string  `json:"district"`
	Address      string  `json:"address"`
	Longitude    float64 `json:"longitude,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	DeviceCount  int64   `json:"device_count"`
	Status       int8    `json:"status"`
}

// CreateVenueRequest 创建场地请求
type CreateVenueRequest struct {
	MerchantID   int64    `json:"merchant_id" binding:"required"`
	Name         string   `json:"name" binding:"required,max=100"`
	Type         string   `json:"type" binding:"required,oneof=mall hotel community office other"`
	Province     string   `json:"province" binding:"required,max=50"`
	City         string   `json:"city" binding:"required,max=50"`
	District     string   `json:"district" binding:"required,max=50"`
	Address      string   `json:"address" binding:"required,max=255"`
	Longitude    *float64 `json:"longitude"`
	Latitude     *float64 `json:"latitude"`
	ContactName  *string  `json:"contact_name"`
	ContactPhone *string  `json:"contact_phone"`
}

// CreateVenue 创建场地
func (s *VenueAdminService) CreateVenue(ctx context.Context, req *CreateVenueRequest) (*models.Venue, error) {
	// 检查商户是否存在
	_, err := s.merchantRepo.GetByID(ctx, req.MerchantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, commonErrors.ErrMerchantNotFound
		}
		return nil, err
	}

	venue := &models.Venue{
		MerchantID:   req.MerchantID,
		Name:         req.Name,
		Type:         req.Type,
		Province:     req.Province,
		City:         req.City,
		District:     req.District,
		Address:      req.Address,
		Longitude:    req.Longitude,
		Latitude:     req.Latitude,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Status:       models.VenueStatusActive,
	}

	if err := s.venueRepo.Create(ctx, venue); err != nil {
		return nil, err
	}

	return venue, nil
}

// UpdateVenueRequest 更新场地请求
type UpdateVenueRequest struct {
	MerchantID   int64    `json:"merchant_id" binding:"required"`
	Name         string   `json:"name" binding:"required,max=100"`
	Type         string   `json:"type" binding:"required,oneof=mall hotel community office other"`
	Province     string   `json:"province" binding:"required,max=50"`
	City         string   `json:"city" binding:"required,max=50"`
	District     string   `json:"district" binding:"required,max=50"`
	Address      string   `json:"address" binding:"required,max=255"`
	Longitude    *float64 `json:"longitude"`
	Latitude     *float64 `json:"latitude"`
	ContactName  *string  `json:"contact_name"`
	ContactPhone *string  `json:"contact_phone"`
}

// UpdateVenue 更新场地
func (s *VenueAdminService) UpdateVenue(ctx context.Context, id int64, req *UpdateVenueRequest) error {
	venue, err := s.venueRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return venueNotFoundErr
		}
		return err
	}

	// 检查商户是否存在
	if req.MerchantID != venue.MerchantID {
		_, err = s.merchantRepo.GetByID(ctx, req.MerchantID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return commonErrors.ErrMerchantNotFound
			}
			return err
		}
	}

	venue.MerchantID = req.MerchantID
	venue.Name = req.Name
	venue.Type = req.Type
	venue.Province = req.Province
	venue.City = req.City
	venue.District = req.District
	venue.Address = req.Address
	venue.Longitude = req.Longitude
	venue.Latitude = req.Latitude
	venue.ContactName = req.ContactName
	venue.ContactPhone = req.ContactPhone

	return s.venueRepo.Update(ctx, venue)
}

// UpdateVenueStatus 更新场地状态
func (s *VenueAdminService) UpdateVenueStatus(ctx context.Context, id int64, status int8) error {
	_, err := s.venueRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return venueNotFoundErr
		}
		return err
	}

	return s.venueRepo.UpdateStatus(ctx, id, status)
}

// DeleteVenue 删除场地
func (s *VenueAdminService) DeleteVenue(ctx context.Context, id int64) error {
	// 检查场地是否存在
	_, err := s.venueRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return venueNotFoundErr
		}
		return err
	}

	// 检查场地下是否有设备
	count, err := s.venueRepo.CountDevices(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return venueHasDevicesErr
	}

	return s.venueRepo.Delete(ctx, id)
}

// GetVenue 获取场地详情
func (s *VenueAdminService) GetVenue(ctx context.Context, id int64) (*VenueInfo, error) {
	venue, err := s.venueRepo.GetByIDWithMerchant(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, venueNotFoundErr
		}
		return nil, err
	}

	// 获取设备数量
	deviceCount, _ := s.venueRepo.CountDevices(ctx, id)

	return s.toVenueInfo(venue, deviceCount), nil
}

// ListVenues 获取场地列表
func (s *VenueAdminService) ListVenues(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*VenueInfo, int64, error) {
	venues, total, err := s.venueRepo.List(ctx, offset, limit, filters)
	if err != nil {
		return nil, 0, err
	}

	infos := make([]*VenueInfo, 0, len(venues))
	for _, v := range venues {
		deviceCount, _ := s.venueRepo.CountDevices(ctx, v.ID)
		infos = append(infos, s.toVenueInfo(v, deviceCount))
	}

	return infos, total, nil
}

// ListVenuesByMerchant 获取商户下的场地列表
func (s *VenueAdminService) ListVenuesByMerchant(ctx context.Context, merchantID int64) ([]*models.Venue, error) {
	return s.venueRepo.ListByMerchantSimple(ctx, merchantID)
}

// toVenueInfo 转换为场地信息
func (s *VenueAdminService) toVenueInfo(venue *models.Venue, deviceCount int64) *VenueInfo {
	info := &VenueInfo{
		ID:           venue.ID,
		MerchantID:   venue.MerchantID,
		Name:         venue.Name,
		Type:         venue.Type,
		Province:     venue.Province,
		City:         venue.City,
		District:     venue.District,
		Address:      venue.Address,
		ContactName:  venue.ContactName,
		ContactPhone: venue.ContactPhone,
		DeviceCount:  deviceCount,
		Status:       venue.Status,
	}

	if venue.Longitude != nil {
		info.Longitude = *venue.Longitude
	}
	if venue.Latitude != nil {
		info.Latitude = *venue.Latitude
	}

	if venue.Merchant != nil {
		info.MerchantName = venue.Merchant.Name
	}

	return info
}
