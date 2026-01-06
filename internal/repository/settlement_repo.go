// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// SettlementRepository 结算仓储
type SettlementRepository struct {
	db *gorm.DB
}

// NewSettlementRepository 创建结算仓储
func NewSettlementRepository(db *gorm.DB) *SettlementRepository {
	return &SettlementRepository{db: db}
}

// Create 创建结算记录
func (r *SettlementRepository) Create(ctx context.Context, settlement *models.Settlement) error {
	return r.db.WithContext(ctx).Create(settlement).Error
}

// GetByID 根据 ID 获取结算记录
func (r *SettlementRepository) GetByID(ctx context.Context, id int64) (*models.Settlement, error) {
	var settlement models.Settlement
	err := r.db.WithContext(ctx).First(&settlement, id).Error
	if err != nil {
		return nil, err
	}
	return &settlement, nil
}

// GetBySettlementNo 根据结算单号获取结算记录
func (r *SettlementRepository) GetBySettlementNo(ctx context.Context, settlementNo string) (*models.Settlement, error) {
	var settlement models.Settlement
	err := r.db.WithContext(ctx).Where("settlement_no = ?", settlementNo).First(&settlement).Error
	if err != nil {
		return nil, err
	}
	return &settlement, nil
}

// Update 更新结算记录
func (r *SettlementRepository) Update(ctx context.Context, settlement *models.Settlement) error {
	return r.db.WithContext(ctx).Save(settlement).Error
}

// UpdateStatus 更新结算状态
func (r *SettlementRepository) UpdateStatus(ctx context.Context, id int64, status string, operatorID *int64) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if operatorID != nil {
		updates["operator_id"] = *operatorID
	}
	if status == models.SettlementStatusCompleted {
		now := time.Now()
		updates["settled_at"] = &now
	}
	return r.db.WithContext(ctx).Model(&models.Settlement{}).Where("id = ?", id).Updates(updates).Error
}

// SettlementFilter 结算查询过滤条件
type SettlementFilter struct {
	Type        string
	TargetID    *int64
	Status      string
	PeriodStart *time.Time
	PeriodEnd   *time.Time
}

// List 获取结算列表
func (r *SettlementRepository) List(ctx context.Context, filter *SettlementFilter, offset, limit int) ([]*models.Settlement, int64, error) {
	var settlements []*models.Settlement
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Settlement{})

	if filter != nil {
		if filter.Type != "" {
			query = query.Where("type = ?", filter.Type)
		}
		if filter.TargetID != nil {
			query = query.Where("target_id = ?", *filter.TargetID)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.PeriodStart != nil {
			query = query.Where("period_start >= ?", *filter.PeriodStart)
		}
		if filter.PeriodEnd != nil {
			query = query.Where("period_end <= ?", *filter.PeriodEnd)
		}
	}

	// 获取总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取数据
	err = query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&settlements).Error
	if err != nil {
		return nil, 0, err
	}

	return settlements, total, nil
}

// ListByTarget 获取指定目标的结算列表
func (r *SettlementRepository) ListByTarget(ctx context.Context, settlementType string, targetID int64, offset, limit int) ([]*models.Settlement, int64, error) {
	filter := &SettlementFilter{
		Type:     settlementType,
		TargetID: &targetID,
	}
	return r.List(ctx, filter, offset, limit)
}

// GetPendingSettlements 获取待结算记录
func (r *SettlementRepository) GetPendingSettlements(ctx context.Context, settlementType string) ([]*models.Settlement, error) {
	var settlements []*models.Settlement
	query := r.db.WithContext(ctx).Where("status = ?", models.SettlementStatusPending)
	if settlementType != "" {
		query = query.Where("type = ?", settlementType)
	}
	err := query.Order("created_at ASC").Find(&settlements).Error
	return settlements, err
}

// GetSummary 获取结算汇总统计
func (r *SettlementRepository) GetSummary(ctx context.Context, settlementType string, startDate, endDate *time.Time) (*models.SettlementSummary, error) {
	var summary models.SettlementSummary

	query := r.db.WithContext(ctx).Model(&models.Settlement{})
	if settlementType != "" {
		query = query.Where("type = ?", settlementType)
	}
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	// 获取总数和金额汇总
	err := query.Select(
		"COUNT(*) as total_settlements",
		"COALESCE(SUM(total_amount), 0) as total_amount",
		"COALESCE(SUM(fee), 0) as total_fee",
		"COALESCE(SUM(actual_amount), 0) as total_actual",
	).Scan(&summary).Error
	if err != nil {
		return nil, err
	}

	// 获取待结算数量
	var pendingCount int64
	err = r.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("status = ?", models.SettlementStatusPending).
		Count(&pendingCount).Error
	if err != nil {
		return nil, err
	}
	summary.PendingCount = int(pendingCount)

	// 获取已完成数量
	var completedCount int64
	err = r.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("status = ?", models.SettlementStatusCompleted).
		Count(&completedCount).Error
	if err != nil {
		return nil, err
	}
	summary.CompletedCount = int(completedCount)

	return &summary, nil
}

// CountPending 统计待结算数量
func (r *SettlementRepository) CountPending(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("status = ?", models.SettlementStatusPending).
		Count(&count).Error
	return count, err
}

// ExistsForPeriod 检查指定周期是否已存在结算记录
func (r *SettlementRepository) ExistsForPeriod(ctx context.Context, settlementType string, targetID int64, periodStart, periodEnd time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ?", settlementType).
		Where("target_id = ?", targetID).
		Where("period_start = ?", periodStart).
		Where("period_end = ?", periodEnd).
		Count(&count).Error
	return count > 0, err
}

// BatchCreate 批量创建结算记录
func (r *SettlementRepository) BatchCreate(ctx context.Context, settlements []*models.Settlement) error {
	if len(settlements) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(settlements, 100).Error
}

// GetMerchantSettlements 获取商户结算汇总
func (r *SettlementRepository) GetMerchantSettlements(ctx context.Context, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	query := r.db.WithContext(ctx).Model(&models.Settlement{}).
		Select(
			"target_id",
			"SUM(total_amount) as total_amount",
			"SUM(fee) as total_fee",
			"SUM(actual_amount) as actual_amount",
			"SUM(order_count) as order_count",
		).
		Where("type = ?", models.SettlementTypeMerchant).
		Group("target_id")

	if startDate != nil {
		query = query.Where("period_start >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("period_end <= ?", *endDate)
	}

	var results []map[string]interface{}
	err := query.Find(&results).Error
	return results, err
}

// GetDistributorSettlements 获取分销商结算汇总
func (r *SettlementRepository) GetDistributorSettlements(ctx context.Context, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	query := r.db.WithContext(ctx).Model(&models.Settlement{}).
		Select(
			"target_id",
			"SUM(total_amount) as total_amount",
			"SUM(fee) as total_fee",
			"SUM(actual_amount) as actual_amount",
			"SUM(order_count) as order_count",
		).
		Where("type = ?", models.SettlementTypeDistributor).
		Group("target_id")

	if startDate != nil {
		query = query.Where("period_start >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("period_end <= ?", *endDate)
	}

	var results []map[string]interface{}
	err := query.Find(&results).Error
	return results, err
}
