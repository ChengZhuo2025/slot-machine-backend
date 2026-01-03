// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// DeviceLogRepository 设备日志仓储
type DeviceLogRepository struct {
	db *gorm.DB
}

// NewDeviceLogRepository 创建设备日志仓储
func NewDeviceLogRepository(db *gorm.DB) *DeviceLogRepository {
	return &DeviceLogRepository{db: db}
}

// Create 创建设备日志
func (r *DeviceLogRepository) Create(ctx context.Context, log *models.DeviceLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// CreateBatch 批量创建设备日志
func (r *DeviceLogRepository) CreateBatch(ctx context.Context, logs []*models.DeviceLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&logs).Error
}

// GetByID 根据 ID 获取日志
func (r *DeviceLogRepository) GetByID(ctx context.Context, id int64) (*models.DeviceLog, error) {
	var log models.DeviceLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// List 获取设备日志列表
func (r *DeviceLogRepository) List(ctx context.Context, deviceID int64, offset, limit int, filters map[string]interface{}) ([]*models.DeviceLog, int64, error) {
	var logs []*models.DeviceLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.DeviceLog{}).Where("device_id = ?", deviceID)

	// 应用过滤条件
	if logType, ok := filters["type"].(string); ok && logType != "" {
		query = query.Where("type = ?", logType)
	}
	if startTime, ok := filters["start_time"].(time.Time); ok {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime, ok := filters["end_time"].(time.Time); ok {
		query = query.Where("created_at <= ?", endTime)
	}
	if operatorType, ok := filters["operator_type"].(string); ok && operatorType != "" {
		query = query.Where("operator_type = ?", operatorType)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListByDeviceIDs 根据设备 ID 列表获取日志
func (r *DeviceLogRepository) ListByDeviceIDs(ctx context.Context, deviceIDs []int64, offset, limit int) ([]*models.DeviceLog, int64, error) {
	var logs []*models.DeviceLog
	var total int64

	if len(deviceIDs) == 0 {
		return logs, 0, nil
	}

	query := r.db.WithContext(ctx).Model(&models.DeviceLog{}).Where("device_id IN ?", deviceIDs)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("Device").Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetLatestByDeviceID 获取设备最新日志
func (r *DeviceLogRepository) GetLatestByDeviceID(ctx context.Context, deviceID int64, logType string) (*models.DeviceLog, error) {
	var log models.DeviceLog
	query := r.db.WithContext(ctx).Where("device_id = ?", deviceID)
	if logType != "" {
		query = query.Where("type = ?", logType)
	}
	err := query.Order("created_at DESC").First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// CountByDeviceID 统计设备日志数量
func (r *DeviceLogRepository) CountByDeviceID(ctx context.Context, deviceID int64, logType string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.DeviceLog{}).Where("device_id = ?", deviceID)
	if logType != "" {
		query = query.Where("type = ?", logType)
	}
	err := query.Count(&count).Error
	return count, err
}

// CountByTypeInPeriod 统计指定时间段内各类型日志数量
func (r *DeviceLogRepository) CountByTypeInPeriod(ctx context.Context, deviceID int64, startTime, endTime time.Time) (map[string]int64, error) {
	type Result struct {
		Type  string
		Count int64
	}
	var results []Result

	err := r.db.WithContext(ctx).Model(&models.DeviceLog{}).
		Select("type, count(*) as count").
		Where("device_id = ? AND created_at >= ? AND created_at <= ?", deviceID, startTime, endTime).
		Group("type").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	countMap := make(map[string]int64)
	for _, r := range results {
		countMap[r.Type] = r.Count
	}
	return countMap, nil
}

// DeleteOldLogs 删除旧日志
func (r *DeviceLogRepository) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&models.DeviceLog{})
	return result.RowsAffected, result.Error
}

// DeviceMaintenanceRepository 设备维护记录仓储
type DeviceMaintenanceRepository struct {
	db *gorm.DB
}

// NewDeviceMaintenanceRepository 创建设备维护记录仓储
func NewDeviceMaintenanceRepository(db *gorm.DB) *DeviceMaintenanceRepository {
	return &DeviceMaintenanceRepository{db: db}
}

// Create 创建维护记录
func (r *DeviceMaintenanceRepository) Create(ctx context.Context, maintenance *models.DeviceMaintenance) error {
	return r.db.WithContext(ctx).Create(maintenance).Error
}

// GetByID 根据 ID 获取维护记录
func (r *DeviceMaintenanceRepository) GetByID(ctx context.Context, id int64) (*models.DeviceMaintenance, error) {
	var maintenance models.DeviceMaintenance
	err := r.db.WithContext(ctx).First(&maintenance, id).Error
	if err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// GetByIDWithRelations 根据 ID 获取维护记录（包含关联）
func (r *DeviceMaintenanceRepository) GetByIDWithRelations(ctx context.Context, id int64) (*models.DeviceMaintenance, error) {
	var maintenance models.DeviceMaintenance
	err := r.db.WithContext(ctx).Preload("Device").Preload("Operator").First(&maintenance, id).Error
	if err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// Update 更新维护记录
func (r *DeviceMaintenanceRepository) Update(ctx context.Context, maintenance *models.DeviceMaintenance) error {
	return r.db.WithContext(ctx).Save(maintenance).Error
}

// UpdateStatus 更新维护状态
func (r *DeviceMaintenanceRepository) UpdateStatus(ctx context.Context, id int64, status int8, completedAt *time.Time) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if completedAt != nil {
		updates["completed_at"] = completedAt
	}
	return r.db.WithContext(ctx).Model(&models.DeviceMaintenance{}).Where("id = ?", id).Updates(updates).Error
}

// List 获取维护记录列表
func (r *DeviceMaintenanceRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.DeviceMaintenance, int64, error) {
	var maintenances []*models.DeviceMaintenance
	var total int64

	query := r.db.WithContext(ctx).Model(&models.DeviceMaintenance{})

	// 应用过滤条件
	if deviceID, ok := filters["device_id"].(int64); ok && deviceID > 0 {
		query = query.Where("device_id = ?", deviceID)
	}
	if maintenanceType, ok := filters["type"].(string); ok && maintenanceType != "" {
		query = query.Where("type = ?", maintenanceType)
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}
	if operatorID, ok := filters["operator_id"].(int64); ok && operatorID > 0 {
		query = query.Where("operator_id = ?", operatorID)
	}
	if startTime, ok := filters["start_time"].(time.Time); ok {
		query = query.Where("started_at >= ?", startTime)
	}
	if endTime, ok := filters["end_time"].(time.Time); ok {
		query = query.Where("started_at <= ?", endTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Preload("Device").Preload("Operator").Order("id DESC").Offset(offset).Limit(limit).Find(&maintenances).Error; err != nil {
		return nil, 0, err
	}

	return maintenances, total, nil
}

// ListByDeviceID 根据设备 ID 获取维护记录
func (r *DeviceMaintenanceRepository) ListByDeviceID(ctx context.Context, deviceID int64, offset, limit int) ([]*models.DeviceMaintenance, int64, error) {
	var maintenances []*models.DeviceMaintenance
	var total int64

	query := r.db.WithContext(ctx).Model(&models.DeviceMaintenance{}).Where("device_id = ?", deviceID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("Operator").Order("id DESC").Offset(offset).Limit(limit).Find(&maintenances).Error; err != nil {
		return nil, 0, err
	}

	return maintenances, total, nil
}

// GetInProgressByDeviceID 获取设备进行中的维护记录
func (r *DeviceMaintenanceRepository) GetInProgressByDeviceID(ctx context.Context, deviceID int64) (*models.DeviceMaintenance, error) {
	var maintenance models.DeviceMaintenance
	err := r.db.WithContext(ctx).Where("device_id = ? AND status = ?", deviceID, models.MaintenanceStatusInProgress).First(&maintenance).Error
	if err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// CountByDeviceID 统计设备维护记录数量
func (r *DeviceMaintenanceRepository) CountByDeviceID(ctx context.Context, deviceID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.DeviceMaintenance{}).Where("device_id = ?", deviceID).Count(&count).Error
	return count, err
}

// SumCostByDeviceID 统计设备维护成本
func (r *DeviceMaintenanceRepository) SumCostByDeviceID(ctx context.Context, deviceID int64) (float64, error) {
	var sum float64
	err := r.db.WithContext(ctx).Model(&models.DeviceMaintenance{}).
		Where("device_id = ?", deviceID).
		Select("COALESCE(SUM(cost), 0)").
		Scan(&sum).Error
	return sum, err
}
