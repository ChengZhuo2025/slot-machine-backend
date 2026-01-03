// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// MerchantRepository 商户仓储
type MerchantRepository struct {
	db *gorm.DB
}

// NewMerchantRepository 创建商户仓储
func NewMerchantRepository(db *gorm.DB) *MerchantRepository {
	return &MerchantRepository{db: db}
}

// Create 创建商户
func (r *MerchantRepository) Create(ctx context.Context, merchant *models.Merchant) error {
	return r.db.WithContext(ctx).Create(merchant).Error
}

// GetByID 根据 ID 获取商户
func (r *MerchantRepository) GetByID(ctx context.Context, id int64) (*models.Merchant, error) {
	var merchant models.Merchant
	err := r.db.WithContext(ctx).First(&merchant, id).Error
	if err != nil {
		return nil, err
	}
	return &merchant, nil
}

// Update 更新商户
func (r *MerchantRepository) Update(ctx context.Context, merchant *models.Merchant) error {
	return r.db.WithContext(ctx).Save(merchant).Error
}

// UpdateFields 更新指定字段
func (r *MerchantRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Merchant{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新商户状态
func (r *MerchantRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Merchant{}).Where("id = ?", id).Update("status", status).Error
}

// Delete 删除商户（软删除）
func (r *MerchantRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Merchant{}, id).Error
}

// List 获取商户列表
func (r *MerchantRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Merchant, int64, error) {
	var merchants []*models.Merchant
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Merchant{})

	// 应用过滤条件
	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if contactName, ok := filters["contact_name"].(string); ok && contactName != "" {
		query = query.Where("contact_name LIKE ?", "%"+contactName+"%")
	}
	if contactPhone, ok := filters["contact_phone"].(string); ok && contactPhone != "" {
		query = query.Where("contact_phone LIKE ?", "%"+contactPhone+"%")
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&merchants).Error; err != nil {
		return nil, 0, err
	}

	return merchants, total, nil
}

// ListAll 获取所有商户
func (r *MerchantRepository) ListAll(ctx context.Context) ([]*models.Merchant, error) {
	var merchants []*models.Merchant
	err := r.db.WithContext(ctx).Where("status = ?", models.MerchantStatusActive).Order("id DESC").Find(&merchants).Error
	return merchants, err
}

// ExistsByName 检查商户名称是否存在
func (r *MerchantRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Merchant{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

// ExistsByNameExcludeID 检查商户名称是否存在（排除指定 ID）
func (r *MerchantRepository) ExistsByNameExcludeID(ctx context.Context, name string, excludeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Merchant{}).Where("name = ? AND id != ?", name, excludeID).Count(&count).Error
	return count > 0, err
}

// CountVenues 统计商户下的场地数量
func (r *MerchantRepository) CountVenues(ctx context.Context, merchantID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Venue{}).Where("merchant_id = ?", merchantID).Count(&count).Error
	return count, err
}

// CountDevices 统计商户下的设备数量
func (r *MerchantRepository) CountDevices(ctx context.Context, merchantID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Device{}).
		Joins("JOIN venues ON venues.id = devices.venue_id").
		Where("venues.merchant_id = ?", merchantID).
		Count(&count).Error
	return count, err
}

// GetStatistics 获取商户统计信息
func (r *MerchantRepository) GetStatistics(ctx context.Context, merchantID int64) (*MerchantStats, error) {
	stats := &MerchantStats{}

	// 统计场地数量
	if err := r.db.WithContext(ctx).Model(&models.Venue{}).Where("merchant_id = ?", merchantID).Count(&stats.VenueCount).Error; err != nil {
		return nil, err
	}

	// 统计设备数量
	if err := r.db.WithContext(ctx).Model(&models.Device{}).
		Joins("JOIN venues ON venues.id = devices.venue_id").
		Where("venues.merchant_id = ?", merchantID).
		Count(&stats.DeviceCount).Error; err != nil {
		return nil, err
	}

	// 统计在线设备
	if err := r.db.WithContext(ctx).Model(&models.Device{}).
		Joins("JOIN venues ON venues.id = devices.venue_id").
		Where("venues.merchant_id = ?", merchantID).
		Where("devices.online_status = ?", models.DeviceOnline).
		Count(&stats.OnlineDeviceCount).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// MerchantStats 商户统计
type MerchantStats struct {
	VenueCount        int64 `json:"venue_count"`
	DeviceCount       int64 `json:"device_count"`
	OnlineDeviceCount int64 `json:"online_device_count"`
}
