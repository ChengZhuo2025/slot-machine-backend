// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"smart-locker-backend/internal/models"
)

// VenueRepository 场地仓储
type VenueRepository struct {
	db *gorm.DB
}

// NewVenueRepository 创建场地仓储
func NewVenueRepository(db *gorm.DB) *VenueRepository {
	return &VenueRepository{db: db}
}

// Create 创建场地
func (r *VenueRepository) Create(ctx context.Context, venue *models.Venue) error {
	return r.db.WithContext(ctx).Create(venue).Error
}

// GetByID 根据 ID 获取场地
func (r *VenueRepository) GetByID(ctx context.Context, id int64) (*models.Venue, error) {
	var venue models.Venue
	err := r.db.WithContext(ctx).First(&venue, id).Error
	if err != nil {
		return nil, err
	}
	return &venue, nil
}

// GetByIDWithDevices 根据 ID 获取场地（包含设备）
func (r *VenueRepository) GetByIDWithDevices(ctx context.Context, id int64) (*models.Venue, error) {
	var venue models.Venue
	err := r.db.WithContext(ctx).
		Preload("Devices", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", models.DeviceStatusActive)
		}).
		First(&venue, id).Error
	if err != nil {
		return nil, err
	}
	return &venue, nil
}

// Update 更新场地
func (r *VenueRepository) Update(ctx context.Context, venue *models.Venue) error {
	return r.db.WithContext(ctx).Save(venue).Error
}

// UpdateFields 更新指定字段
func (r *VenueRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Venue{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新场地状态
func (r *VenueRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Venue{}).Where("id = ?", id).Update("status", status).Error
}

// List 获取场地列表
func (r *VenueRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Venue, int64, error) {
	var venues []*models.Venue
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Venue{})

	// 应用过滤条件
	if merchantID, ok := filters["merchant_id"].(int64); ok && merchantID > 0 {
		query = query.Where("merchant_id = ?", merchantID)
	}
	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if venueType, ok := filters["type"].(string); ok && venueType != "" {
		query = query.Where("type = ?", venueType)
	}
	if city, ok := filters["city"].(string); ok && city != "" {
		query = query.Where("city = ?", city)
	}
	if district, ok := filters["district"].(string); ok && district != "" {
		query = query.Where("district = ?", district)
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&venues).Error; err != nil {
		return nil, 0, err
	}

	return venues, total, nil
}

// ListByMerchant 获取商户下的场地列表
func (r *VenueRepository) ListByMerchant(ctx context.Context, merchantID int64, status *int8) ([]*models.Venue, error) {
	var venues []*models.Venue
	query := r.db.WithContext(ctx).Where("merchant_id = ?", merchantID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	err := query.Order("id DESC").Find(&venues).Error
	return venues, err
}

// ListNearby 获取附近场地列表（基于经纬度）
func (r *VenueRepository) ListNearby(ctx context.Context, longitude, latitude float64, radiusKm float64, limit int) ([]*models.Venue, error) {
	var venues []*models.Venue

	// 使用 Haversine 公式计算距离
	// 6371 是地球半径（公里）
	query := r.db.WithContext(ctx).
		Select("*, (6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) AS distance", latitude, longitude, latitude).
		Where("status = ?", models.VenueStatusActive).
		Where("latitude IS NOT NULL AND longitude IS NOT NULL").
		Having("distance < ?", radiusKm).
		Order("distance ASC").
		Limit(limit)

	err := query.Find(&venues).Error
	return venues, err
}

// ListByCity 获取城市下的场地列表
func (r *VenueRepository) ListByCity(ctx context.Context, city string, offset, limit int) ([]*models.Venue, int64, error) {
	var venues []*models.Venue
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Venue{}).
		Where("city = ?", city).
		Where("status = ?", models.VenueStatusActive)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&venues).Error; err != nil {
		return nil, 0, err
	}

	return venues, total, nil
}

// GetDeviceCount 获取场地下的设备数量
func (r *VenueRepository) GetDeviceCount(ctx context.Context, venueID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Device{}).
		Where("venue_id = ?", venueID).
		Where("status = ?", models.DeviceStatusActive).
		Count(&count).Error
	return count, err
}

// GetAvailableDeviceCount 获取场地下的可用设备数量
func (r *VenueRepository) GetAvailableDeviceCount(ctx context.Context, venueID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Device{}).
		Where("venue_id = ?", venueID).
		Where("status = ?", models.DeviceStatusActive).
		Where("online_status = ?", models.DeviceOnline).
		Where("rental_status = ?", models.DeviceRentalFree).
		Where("available_slots > 0").
		Count(&count).Error
	return count, err
}

// GetCities 获取所有城市列表
func (r *VenueRepository) GetCities(ctx context.Context) ([]string, error) {
	var cities []string
	err := r.db.WithContext(ctx).Model(&models.Venue{}).
		Where("status = ?", models.VenueStatusActive).
		Distinct("city").
		Pluck("city", &cities).Error
	return cities, err
}

// ExistsByName 检查场地名称是否存在
func (r *VenueRepository) ExistsByName(ctx context.Context, merchantID int64, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Venue{}).
		Where("merchant_id = ?", merchantID).
		Where("name = ?", name).
		Count(&count).Error
	return count > 0, err
}

// Delete 删除场地（软删除）
func (r *VenueRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Venue{}, id).Error
}
