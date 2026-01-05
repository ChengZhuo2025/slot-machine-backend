// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// CommissionRepository 佣金仓储
type CommissionRepository struct {
	db *gorm.DB
}

// NewCommissionRepository 创建佣金仓储
func NewCommissionRepository(db *gorm.DB) *CommissionRepository {
	return &CommissionRepository{db: db}
}

// Create 创建佣金记录
func (r *CommissionRepository) Create(ctx context.Context, commission *models.Commission) error {
	return r.db.WithContext(ctx).Create(commission).Error
}

// CreateBatch 批量创建佣金记录
func (r *CommissionRepository) CreateBatch(ctx context.Context, commissions []*models.Commission) error {
	if len(commissions) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&commissions).Error
}

// GetByID 根据 ID 获取佣金记录
func (r *CommissionRepository) GetByID(ctx context.Context, id int64) (*models.Commission, error) {
	var commission models.Commission
	err := r.db.WithContext(ctx).First(&commission, id).Error
	if err != nil {
		return nil, err
	}
	return &commission, nil
}

// GetByIDWithRelations 根据 ID 获取佣金记录（包含关联）
func (r *CommissionRepository) GetByIDWithRelations(ctx context.Context, id int64) (*models.Commission, error) {
	var commission models.Commission
	err := r.db.WithContext(ctx).
		Preload("Distributor").
		Preload("Distributor.User").
		Preload("Order").
		Preload("FromUser").
		First(&commission, id).Error
	if err != nil {
		return nil, err
	}
	return &commission, nil
}

// GetByOrderID 根据订单 ID 获取佣金记录列表
func (r *CommissionRepository) GetByOrderID(ctx context.Context, orderID int64) ([]*models.Commission, error) {
	var commissions []*models.Commission
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Find(&commissions).Error
	if err != nil {
		return nil, err
	}
	return commissions, nil
}

// GetByDistributorID 根据分销商 ID 获取佣金记录列表
func (r *CommissionRepository) GetByDistributorID(ctx context.Context, distributorID int64, offset, limit int) ([]*models.Commission, int64, error) {
	var commissions []*models.Commission
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Commission{}).Where("distributor_id = ?", distributorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("Order").
		Preload("FromUser").
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&commissions).Error; err != nil {
		return nil, 0, err
	}

	return commissions, total, nil
}

// Update 更新佣金记录
func (r *CommissionRepository) Update(ctx context.Context, commission *models.Commission) error {
	return r.db.WithContext(ctx).Save(commission).Error
}

// UpdateStatus 更新佣金状态
func (r *CommissionRepository) UpdateStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&models.Commission{}).Where("id = ?", id).Update("status", status).Error
}

// Settle 结算佣金
func (r *CommissionRepository) Settle(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Commission{}).
		Where("id = ? AND status = ?", id, models.CommissionStatusPending).
		Updates(map[string]interface{}{
			"status":     models.CommissionStatusSettled,
			"settled_at": now,
		}).Error
}

// CancelByOrderID 根据订单 ID 取消佣金记录（退款时）
func (r *CommissionRepository) CancelByOrderID(ctx context.Context, orderID int64) error {
	return r.db.WithContext(ctx).Model(&models.Commission{}).
		Where("order_id = ? AND status = ?", orderID, models.CommissionStatusPending).
		Update("status", models.CommissionStatusCancelled).Error
}

// List 获取佣金记录列表
func (r *CommissionRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Commission, int64, error) {
	var commissions []*models.Commission
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Commission{})

	// 应用过滤条件
	if distributorID, ok := filters["distributor_id"].(int64); ok && distributorID > 0 {
		query = query.Where("distributor_id = ?", distributorID)
	}
	if status, ok := filters["status"].(int); ok && status >= 0 {
		query = query.Where("status = ?", status)
	}
	if commissionType, ok := filters["type"].(string); ok && commissionType != "" {
		query = query.Where("type = ?", commissionType)
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

	// 查询列表
	if err := query.
		Preload("Distributor").
		Preload("Distributor.User").
		Preload("Order").
		Preload("FromUser").
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&commissions).Error; err != nil {
		return nil, 0, err
	}

	return commissions, total, nil
}

// GetPendingByDistributorID 获取待结算的佣金记录
func (r *CommissionRepository) GetPendingByDistributorID(ctx context.Context, distributorID int64) ([]*models.Commission, error) {
	var commissions []*models.Commission
	err := r.db.WithContext(ctx).
		Where("distributor_id = ? AND status = ?", distributorID, models.CommissionStatusPending).
		Order("id DESC").
		Find(&commissions).Error
	return commissions, err
}

// GetSettledByDistributorID 获取已结算的佣金记录
func (r *CommissionRepository) GetSettledByDistributorID(ctx context.Context, distributorID int64, offset, limit int) ([]*models.Commission, int64, error) {
	var commissions []*models.Commission
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ? AND status = ?", distributorID, models.CommissionStatusSettled)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("Order").
		Preload("FromUser").
		Order("settled_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&commissions).Error; err != nil {
		return nil, 0, err
	}

	return commissions, total, nil
}

// SumByDistributorID 统计分销商的佣金总额
func (r *CommissionRepository) SumByDistributorID(ctx context.Context, distributorID int64, status *int) (float64, error) {
	var sum float64
	query := r.db.WithContext(ctx).Model(&models.Commission{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("distributor_id = ?", distributorID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Scan(&sum).Error
	return sum, err
}

// SumByDistributorIDAndType 按类型统计分销商的佣金
func (r *CommissionRepository) SumByDistributorIDAndType(ctx context.Context, distributorID int64, commissionType string) (float64, error) {
	var sum float64
	err := r.db.WithContext(ctx).Model(&models.Commission{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("distributor_id = ? AND type = ? AND status = ?", distributorID, commissionType, models.CommissionStatusSettled).
		Scan(&sum).Error
	return sum, err
}

// CountByDistributorID 统计分销商的佣金记录数
func (r *CommissionRepository) CountByDistributorID(ctx context.Context, distributorID int64, status *int) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ?", distributorID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Count(&count).Error
	return count, err
}

// GetStatsByDistributorID 获取分销商佣金统计
func (r *CommissionRepository) GetStatsByDistributorID(ctx context.Context, distributorID int64) (map[string]interface{}, error) {
	type Stats struct {
		TotalAmount   float64 `gorm:"column:total_amount"`
		PendingAmount float64 `gorm:"column:pending_amount"`
		SettledAmount float64 `gorm:"column:settled_amount"`
		TotalCount    int64   `gorm:"column:total_count"`
		DirectAmount  float64 `gorm:"column:direct_amount"`
		IndirectAmount float64 `gorm:"column:indirect_amount"`
	}

	var stats Stats
	err := r.db.WithContext(ctx).Model(&models.Commission{}).
		Select(`
			COALESCE(SUM(amount), 0) as total_amount,
			COALESCE(SUM(CASE WHEN status = 0 THEN amount ELSE 0 END), 0) as pending_amount,
			COALESCE(SUM(CASE WHEN status = 1 THEN amount ELSE 0 END), 0) as settled_amount,
			COUNT(*) as total_count,
			COALESCE(SUM(CASE WHEN type = 'direct' THEN amount ELSE 0 END), 0) as direct_amount,
			COALESCE(SUM(CASE WHEN type = 'indirect' THEN amount ELSE 0 END), 0) as indirect_amount
		`).
		Where("distributor_id = ?", distributorID).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_amount":    stats.TotalAmount,
		"pending_amount":  stats.PendingAmount,
		"settled_amount":  stats.SettledAmount,
		"total_count":     stats.TotalCount,
		"direct_amount":   stats.DirectAmount,
		"indirect_amount": stats.IndirectAmount,
	}, nil
}

// SettlePendingByTime 结算指定时间之前的待结算佣金
func (r *CommissionRepository) SettlePendingByTime(ctx context.Context, beforeTime time.Time) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Commission{}).
		Where("status = ? AND created_at < ?", models.CommissionStatusPending, beforeTime).
		Updates(map[string]interface{}{
			"status":     models.CommissionStatusSettled,
			"settled_at": now,
		})
	return result.RowsAffected, result.Error
}
