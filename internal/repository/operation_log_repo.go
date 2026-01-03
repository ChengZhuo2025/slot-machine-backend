// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// OperationLogRepository 操作日志仓储
type OperationLogRepository struct {
	db *gorm.DB
}

// NewOperationLogRepository 创建操作日志仓储
func NewOperationLogRepository(db *gorm.DB) *OperationLogRepository {
	return &OperationLogRepository{db: db}
}

// Create 创建操作日志
func (r *OperationLogRepository) Create(ctx context.Context, log *models.OperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetByID 根据 ID 获取操作日志
func (r *OperationLogRepository) GetByID(ctx context.Context, id int64) (*models.OperationLog, error) {
	var log models.OperationLog
	err := r.db.WithContext(ctx).Preload("Admin").First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// List 获取操作日志列表
func (r *OperationLogRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.OperationLog, int64, error) {
	var logs []*models.OperationLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.OperationLog{})

	// 应用过滤条件
	if adminID, ok := filters["admin_id"].(int64); ok && adminID > 0 {
		query = query.Where("admin_id = ?", adminID)
	}
	if module, ok := filters["module"].(string); ok && module != "" {
		query = query.Where("module = ?", module)
	}
	if action, ok := filters["action"].(string); ok && action != "" {
		query = query.Where("action = ?", action)
	}
	if targetType, ok := filters["target_type"].(string); ok && targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if targetID, ok := filters["target_id"].(int64); ok && targetID > 0 {
		query = query.Where("target_id = ?", targetID)
	}
	if ip, ok := filters["ip"].(string); ok && ip != "" {
		query = query.Where("ip = ?", ip)
	}
	if startTime, ok := filters["start_time"].(time.Time); ok {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime, ok := filters["end_time"].(time.Time); ok {
		query = query.Where("created_at <= ?", endTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表（预加载管理员信息）
	if err := query.Preload("Admin").Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListByAdmin 获取管理员的操作日志
func (r *OperationLogRepository) ListByAdmin(ctx context.Context, adminID int64, offset, limit int) ([]*models.OperationLog, int64, error) {
	var logs []*models.OperationLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.OperationLog{}).Where("admin_id = ?", adminID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListByTarget 获取针对某个目标的操作日志
func (r *OperationLogRepository) ListByTarget(ctx context.Context, targetType string, targetID int64, offset, limit int) ([]*models.OperationLog, int64, error) {
	var logs []*models.OperationLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.OperationLog{}).
		Where("target_type = ?", targetType).
		Where("target_id = ?", targetID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("Admin").Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// CountByModule 统计模块操作数量
func (r *OperationLogRepository) CountByModule(ctx context.Context, module string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.OperationLog{}).
		Where("module = ?", module).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

// CountByAdmin 统计管理员操作数量
func (r *OperationLogRepository) CountByAdmin(ctx context.Context, adminID int64, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.OperationLog{}).
		Where("admin_id = ?", adminID).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

// GetModuleStats 获取模块操作统计
func (r *OperationLogRepository) GetModuleStats(ctx context.Context, since time.Time) (map[string]int64, error) {
	var results []struct {
		Module string
		Count  int64
	}

	err := r.db.WithContext(ctx).Model(&models.OperationLog{}).
		Select("module, count(*) as count").
		Where("created_at >= ?", since).
		Group("module").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, r := range results {
		stats[r.Module] = r.Count
	}
	return stats, nil
}

// GetActionStats 获取操作类型统计
func (r *OperationLogRepository) GetActionStats(ctx context.Context, since time.Time) (map[string]int64, error) {
	var results []struct {
		Action string
		Count  int64
	}

	err := r.db.WithContext(ctx).Model(&models.OperationLog{}).
		Select("action, count(*) as count").
		Where("created_at >= ?", since).
		Group("action").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, r := range results {
		stats[r.Action] = r.Count
	}
	return stats, nil
}

// DeleteBefore 删除指定时间之前的日志
func (r *OperationLogRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.OperationLog{})
	return result.RowsAffected, result.Error
}
