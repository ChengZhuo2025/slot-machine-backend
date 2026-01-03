// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// DeviceAlertRepository 设备告警仓储
type DeviceAlertRepository struct {
	db *gorm.DB
}

// NewDeviceAlertRepository 创建设备告警仓储
func NewDeviceAlertRepository(db *gorm.DB) *DeviceAlertRepository {
	return &DeviceAlertRepository{db: db}
}

// Create 创建告警
func (r *DeviceAlertRepository) Create(ctx context.Context, alert *models.DeviceAlert) error {
	return r.db.WithContext(ctx).Create(alert).Error
}

// GetByID 根据 ID 获取告警
func (r *DeviceAlertRepository) GetByID(ctx context.Context, id int64) (*models.DeviceAlert, error) {
	var alert models.DeviceAlert
	err := r.db.WithContext(ctx).First(&alert, id).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// Update 更新告警
func (r *DeviceAlertRepository) Update(ctx context.Context, alert *models.DeviceAlert) error {
	return r.db.WithContext(ctx).Save(alert).Error
}

// List 获取告警列表
func (r *DeviceAlertRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.DeviceAlert, int64, error) {
	var alerts []*models.DeviceAlert
	var total int64

	query := r.db.WithContext(ctx).Model(&models.DeviceAlert{})

	// 应用过滤条件
	if deviceID, ok := filters["device_id"].(int64); ok && deviceID > 0 {
		query = query.Where("device_id = ?", deviceID)
	}
	if alertType, ok := filters["type"].(string); ok && alertType != "" {
		query = query.Where("type = ?", alertType)
	}
	if level, ok := filters["level"].(string); ok && level != "" {
		query = query.Where("level = ?", level)
	}
	if isResolved, ok := filters["is_resolved"].(bool); ok {
		query = query.Where("is_resolved = ?", isResolved)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&alerts).Error; err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

// ListByDevice 获取设备的告警列表
func (r *DeviceAlertRepository) ListByDevice(ctx context.Context, deviceID int64, offset, limit int) ([]*models.DeviceAlert, int64, error) {
	var alerts []*models.DeviceAlert
	var total int64

	query := r.db.WithContext(ctx).Model(&models.DeviceAlert{}).Where("device_id = ?", deviceID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&alerts).Error; err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

// ListUnresolved 获取未解决的告警列表
func (r *DeviceAlertRepository) ListUnresolved(ctx context.Context, offset, limit int) ([]*models.DeviceAlert, int64, error) {
	var alerts []*models.DeviceAlert
	var total int64

	query := r.db.WithContext(ctx).Model(&models.DeviceAlert{}).Where("is_resolved = ?", false)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("level DESC, id DESC").Offset(offset).Limit(limit).Find(&alerts).Error; err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

// CountUnresolved 统计未解决告警数量
func (r *DeviceAlertRepository) CountUnresolved(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.DeviceAlert{}).Where("is_resolved = ?", false).Count(&count).Error
	return count, err
}

// CountByLevel 统计指定级别告警数量
func (r *DeviceAlertRepository) CountByLevel(ctx context.Context, level string, isResolved bool) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.DeviceAlert{}).
		Where("level = ?", level).
		Where("is_resolved = ?", isResolved).
		Count(&count).Error
	return count, err
}

// CountSince 统计指定时间之后的告警数量
func (r *DeviceAlertRepository) CountSince(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.DeviceAlert{}).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

// Resolve 解决告警
func (r *DeviceAlertRepository) Resolve(ctx context.Context, id int64, operatorID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.DeviceAlert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_resolved": true,
			"resolved_by": operatorID,
			"resolved_at": now,
		}).Error
}

// BatchResolve 批量解决告警
func (r *DeviceAlertRepository) BatchResolve(ctx context.Context, ids []int64, operatorID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.DeviceAlert{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"is_resolved": true,
			"resolved_by": operatorID,
			"resolved_at": now,
		}).Error
}

// Delete 删除告警
func (r *DeviceAlertRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.DeviceAlert{}, id).Error
}

// DeleteByDevice 删除设备的所有告警
func (r *DeviceAlertRepository) DeleteByDevice(ctx context.Context, deviceID int64) error {
	return r.db.WithContext(ctx).Where("device_id = ?", deviceID).Delete(&models.DeviceAlert{}).Error
}
