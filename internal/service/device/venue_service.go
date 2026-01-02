// Package device 提供设备服务
package device

import (
	"context"
	"math"

	"gorm.io/gorm"

	"smart-locker-backend/internal/common/errors"
	"smart-locker-backend/internal/models"
	"smart-locker-backend/internal/repository"
)

// VenueService 场地服务
type VenueService struct {
	db         *gorm.DB
	venueRepo  *repository.VenueRepository
	deviceRepo *repository.DeviceRepository
}

// NewVenueService 创建场地服务
func NewVenueService(
	db *gorm.DB,
	venueRepo *repository.VenueRepository,
	deviceRepo *repository.DeviceRepository,
) *VenueService {
	return &VenueService{
		db:         db,
		venueRepo:  venueRepo,
		deviceRepo: deviceRepo,
	}
}

// VenueDetail 场地详情
type VenueDetail struct {
	ID                   int64    `json:"id"`
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	Province             string   `json:"province"`
	City                 string   `json:"city"`
	District             string   `json:"district"`
	Address              string   `json:"address"`
	Longitude            *float64 `json:"longitude,omitempty"`
	Latitude             *float64 `json:"latitude,omitempty"`
	DeviceCount          int64    `json:"device_count"`
	AvailableDeviceCount int64    `json:"available_device_count"`
	Distance             *float64 `json:"distance,omitempty"` // 距离（公里）
}

// VenueListItem 场地列表项
type VenueListItem struct {
	ID                   int64    `json:"id"`
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	City                 string   `json:"city"`
	District             string   `json:"district"`
	Address              string   `json:"address"`
	Longitude            *float64 `json:"longitude,omitempty"`
	Latitude             *float64 `json:"latitude,omitempty"`
	AvailableDeviceCount int64    `json:"available_device_count"`
	Distance             *float64 `json:"distance,omitempty"`
}

// GetVenueByID 根据 ID 获取场地详情
func (s *VenueService) GetVenueByID(ctx context.Context, venueID int64) (*VenueDetail, error) {
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

	// 获取设备数量
	deviceCount, err := s.venueRepo.GetDeviceCount(ctx, venueID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 获取可用设备数量
	availableCount, err := s.venueRepo.GetAvailableDeviceCount(ctx, venueID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return &VenueDetail{
		ID:                   venue.ID,
		Name:                 venue.Name,
		Type:                 venue.Type,
		Province:             venue.Province,
		City:                 venue.City,
		District:             venue.District,
		Address:              venue.Address,
		Longitude:            venue.Longitude,
		Latitude:             venue.Latitude,
		DeviceCount:          deviceCount,
		AvailableDeviceCount: availableCount,
	}, nil
}

// ListNearbyVenues 获取附近场地列表
func (s *VenueService) ListNearbyVenues(ctx context.Context, longitude, latitude float64, radiusKm float64, limit int) ([]*VenueListItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if radiusKm <= 0 {
		radiusKm = 5.0 // 默认 5 公里
	}

	venues, err := s.venueRepo.ListNearby(ctx, longitude, latitude, radiusKm, limit)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*VenueListItem, 0, len(venues))
	for _, v := range venues {
		// 获取可用设备数量
		availableCount, _ := s.venueRepo.GetAvailableDeviceCount(ctx, v.ID)

		item := &VenueListItem{
			ID:                   v.ID,
			Name:                 v.Name,
			Type:                 v.Type,
			City:                 v.City,
			District:             v.District,
			Address:              v.Address,
			Longitude:            v.Longitude,
			Latitude:             v.Latitude,
			AvailableDeviceCount: availableCount,
		}

		// 计算距离
		if v.Longitude != nil && v.Latitude != nil {
			distance := calculateDistance(latitude, longitude, *v.Latitude, *v.Longitude)
			item.Distance = &distance
		}

		result = append(result, item)
	}

	return result, nil
}

// ListVenuesByCity 获取城市下的场地列表
func (s *VenueService) ListVenuesByCity(ctx context.Context, city string, offset, limit int) ([]*VenueListItem, int64, error) {
	venues, total, err := s.venueRepo.ListByCity(ctx, city, offset, limit)
	if err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*VenueListItem, len(venues))
	for i, v := range venues {
		availableCount, _ := s.venueRepo.GetAvailableDeviceCount(ctx, v.ID)

		result[i] = &VenueListItem{
			ID:                   v.ID,
			Name:                 v.Name,
			Type:                 v.Type,
			City:                 v.City,
			District:             v.District,
			Address:              v.Address,
			Longitude:            v.Longitude,
			Latitude:             v.Latitude,
			AvailableDeviceCount: availableCount,
		}
	}

	return result, total, nil
}

// GetCities 获取所有城市列表
func (s *VenueService) GetCities(ctx context.Context) ([]string, error) {
	cities, err := s.venueRepo.GetCities(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	return cities, nil
}

// SearchVenues 搜索场地
func (s *VenueService) SearchVenues(ctx context.Context, keyword string, city string, offset, limit int) ([]*VenueListItem, int64, error) {
	filters := map[string]interface{}{
		"status": int8(models.VenueStatusActive),
	}

	if keyword != "" {
		filters["name"] = keyword
	}
	if city != "" {
		filters["city"] = city
	}

	venues, total, err := s.venueRepo.List(ctx, offset, limit, filters)
	if err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*VenueListItem, len(venues))
	for i, v := range venues {
		availableCount, _ := s.venueRepo.GetAvailableDeviceCount(ctx, v.ID)

		result[i] = &VenueListItem{
			ID:                   v.ID,
			Name:                 v.Name,
			Type:                 v.Type,
			City:                 v.City,
			District:             v.District,
			Address:              v.Address,
			Longitude:            v.Longitude,
			Latitude:             v.Latitude,
			AvailableDeviceCount: availableCount,
		}
	}

	return result, total, nil
}

// calculateDistance 计算两点之间的距离（Haversine 公式）
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // 地球半径（公里）

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
