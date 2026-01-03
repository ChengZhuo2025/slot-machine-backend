// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// DeviceRepository 设备仓储
type DeviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository 创建设备仓储
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create 创建设备
func (r *DeviceRepository) Create(ctx context.Context, device *models.Device) error {
	return r.db.WithContext(ctx).Create(device).Error
}

// GetByID 根据 ID 获取设备
func (r *DeviceRepository) GetByID(ctx context.Context, id int64) (*models.Device, error) {
	var device models.Device
	err := r.db.WithContext(ctx).First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByIDWithVenue 根据 ID 获取设备（包含场地信息）
func (r *DeviceRepository) GetByIDWithVenue(ctx context.Context, id int64) (*models.Device, error) {
	var device models.Device
	err := r.db.WithContext(ctx).Preload("Venue").First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByDeviceNo 根据设备编号获取设备
func (r *DeviceRepository) GetByDeviceNo(ctx context.Context, deviceNo string) (*models.Device, error) {
	var device models.Device
	err := r.db.WithContext(ctx).Where("device_no = ?", deviceNo).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByDeviceNoWithVenue 根据设备编号获取设备（包含场地信息）
func (r *DeviceRepository) GetByDeviceNoWithVenue(ctx context.Context, deviceNo string) (*models.Device, error) {
	var device models.Device
	err := r.db.WithContext(ctx).Preload("Venue").Where("device_no = ?", deviceNo).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByQRCode 根据二维码获取设备
func (r *DeviceRepository) GetByQRCode(ctx context.Context, qrCode string) (*models.Device, error) {
	var device models.Device
	err := r.db.WithContext(ctx).Where("qr_code = ?", qrCode).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByQRCodeWithVenue 根据二维码获取设备（包含场地信息）
func (r *DeviceRepository) GetByQRCodeWithVenue(ctx context.Context, qrCode string) (*models.Device, error) {
	var device models.Device
	err := r.db.WithContext(ctx).Preload("Venue").Where("qr_code = ?", qrCode).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// Update 更新设备
func (r *DeviceRepository) Update(ctx context.Context, device *models.Device) error {
	return r.db.WithContext(ctx).Save(device).Error
}

// UpdateFields 更新指定字段
func (r *DeviceRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新设备状态
func (r *DeviceRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateOnlineStatus 更新在线状态
func (r *DeviceRepository) UpdateOnlineStatus(ctx context.Context, id int64, onlineStatus int8) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", id).Update("online_status", onlineStatus).Error
}

// UpdateRentalStatus 更新租借状态
func (r *DeviceRepository) UpdateRentalStatus(ctx context.Context, id int64, rentalStatus int8, currentRentalID *int64) error {
	fields := map[string]interface{}{
		"rental_status":     rentalStatus,
		"current_rental_id": currentRentalID,
	}
	return r.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateLockStatus 更新锁状态
func (r *DeviceRepository) UpdateLockStatus(ctx context.Context, id int64, lockStatus int8) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", id).Update("lock_status", lockStatus).Error
}

// UpdateHeartbeat 更新心跳信息
func (r *DeviceRepository) UpdateHeartbeat(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", id).Updates(fields).Error
}

// List 获取设备列表
func (r *DeviceRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error) {
	var devices []*models.Device
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Device{})

	// 应用过滤条件
	if venueID, ok := filters["venue_id"].(int64); ok && venueID > 0 {
		query = query.Where("venue_id = ?", venueID)
	}
	if deviceNo, ok := filters["device_no"].(string); ok && deviceNo != "" {
		query = query.Where("device_no LIKE ?", "%"+deviceNo+"%")
	}
	if deviceType, ok := filters["type"].(string); ok && deviceType != "" {
		query = query.Where("type = ?", deviceType)
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}
	if onlineStatus, ok := filters["online_status"].(int8); ok {
		query = query.Where("online_status = ?", onlineStatus)
	}
	if rentalStatus, ok := filters["rental_status"].(int8); ok {
		query = query.Where("rental_status = ?", rentalStatus)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&devices).Error; err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// ListByVenue 获取场地下的设备列表
func (r *DeviceRepository) ListByVenue(ctx context.Context, venueID int64, status *int8) ([]*models.Device, error) {
	var devices []*models.Device
	query := r.db.WithContext(ctx).Where("venue_id = ?", venueID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	err := query.Order("id ASC").Find(&devices).Error
	return devices, err
}

// ListAvailable 获取可租借设备列表
func (r *DeviceRepository) ListAvailable(ctx context.Context, venueID int64) ([]*models.Device, error) {
	var devices []*models.Device
	query := r.db.WithContext(ctx).
		Where("venue_id = ?", venueID).
		Where("status = ?", models.DeviceStatusActive).
		Where("online_status = ?", models.DeviceOnline).
		Where("rental_status = ?", models.DeviceRentalFree).
		Where("available_slots > 0")
	err := query.Order("id ASC").Find(&devices).Error
	return devices, err
}

// ExistsByDeviceNo 检查设备编号是否存在
func (r *DeviceRepository) ExistsByDeviceNo(ctx context.Context, deviceNo string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Device{}).Where("device_no = ?", deviceNo).Count(&count).Error
	return count > 0, err
}

// CreateLog 创建设备日志
func (r *DeviceRepository) CreateLog(ctx context.Context, log *models.DeviceLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// ListLogs 获取设备日志列表
func (r *DeviceRepository) ListLogs(ctx context.Context, deviceID int64, offset, limit int, logType string) ([]*models.DeviceLog, int64, error) {
	var logs []*models.DeviceLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.DeviceLog{}).Where("device_id = ?", deviceID)

	if logType != "" {
		query = query.Where("type = ?", logType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetPricingsByDevice 获取设备的定价列表
func (r *DeviceRepository) GetPricingsByDevice(ctx context.Context, deviceID int64) ([]*models.RentalPricing, error) {
	// 先获取设备信息得到venue_id
	device, err := r.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	var pricings []*models.RentalPricing
	err = r.db.WithContext(ctx).
		Where("venue_id = ?", device.VenueID).
		Where("is_active = ?", true).
		Order("duration_hours ASC, id ASC").
		Find(&pricings).Error
	return pricings, err
}

// GetPricingByID 根据 ID 获取定价
func (r *DeviceRepository) GetPricingByID(ctx context.Context, id int64) (*models.RentalPricing, error) {
	var pricing models.RentalPricing
	err := r.db.WithContext(ctx).First(&pricing, id).Error
	if err != nil {
		return nil, err
	}
	return &pricing, nil
}

// GetDefaultPricing 获取默认定价
func (r *DeviceRepository) GetDefaultPricing(ctx context.Context, deviceID int64) (*models.RentalPricing, error) {
	// 先获取设备信息得到venue_id
	device, err := r.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	var pricing models.RentalPricing
	err = r.db.WithContext(ctx).
		Where("venue_id = ?", device.VenueID).
		Where("is_active = ?", true).
		Order("duration_hours ASC").
		First(&pricing).Error
	if err != nil {
		return nil, err
	}
	return &pricing, nil
}

// GetForUpdate 获取设备（加锁）
func (r *DeviceRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, id int64) (*models.Device, error) {
	var device models.Device
	err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// IncrementAvailableSlots 增加可用槽位
func (r *DeviceRepository) IncrementAvailableSlots(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("id = ?", id).
		UpdateColumn("available_slots", gorm.Expr("available_slots + 1")).
		Error
}

// DecrementAvailableSlots 减少可用槽位
func (r *DeviceRepository) DecrementAvailableSlots(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Model(&models.Device{}).
		Where("id = ? AND available_slots > 0", id).
		UpdateColumn("available_slots", gorm.Expr("available_slots - 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateQRCode 更新设备二维码
func (r *DeviceRepository) UpdateQRCode(ctx context.Context, id int64, qrCode string) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", id).Update("qr_code", qrCode).Error
}
