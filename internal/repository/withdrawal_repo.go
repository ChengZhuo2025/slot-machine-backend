// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// WithdrawalRepository 提现仓储
type WithdrawalRepository struct {
	db *gorm.DB
}

// NewWithdrawalRepository 创建提现仓储
func NewWithdrawalRepository(db *gorm.DB) *WithdrawalRepository {
	return &WithdrawalRepository{db: db}
}

// Create 创建提现记录
func (r *WithdrawalRepository) Create(ctx context.Context, withdrawal *models.Withdrawal) error {
	return r.db.WithContext(ctx).Create(withdrawal).Error
}

// GetByID 根据 ID 获取提现记录
func (r *WithdrawalRepository) GetByID(ctx context.Context, id int64) (*models.Withdrawal, error) {
	var withdrawal models.Withdrawal
	err := r.db.WithContext(ctx).First(&withdrawal, id).Error
	if err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

// GetByIDWithRelations 根据 ID 获取提现记录（包含关联）
func (r *WithdrawalRepository) GetByIDWithRelations(ctx context.Context, id int64) (*models.Withdrawal, error) {
	var withdrawal models.Withdrawal
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Operator").
		First(&withdrawal, id).Error
	if err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

// GetByWithdrawalNo 根据提现单号获取记录
func (r *WithdrawalRepository) GetByWithdrawalNo(ctx context.Context, withdrawalNo string) (*models.Withdrawal, error) {
	var withdrawal models.Withdrawal
	err := r.db.WithContext(ctx).Where("withdrawal_no = ?", withdrawalNo).First(&withdrawal).Error
	if err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

// GetByUserID 根据用户 ID 获取提现记录列表
func (r *WithdrawalRepository) GetByUserID(ctx context.Context, userID int64, offset, limit int) ([]*models.Withdrawal, int64, error) {
	var withdrawals []*models.Withdrawal
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Withdrawal{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&withdrawals).Error; err != nil {
		return nil, 0, err
	}

	return withdrawals, total, nil
}

// Update 更新提现记录
func (r *WithdrawalRepository) Update(ctx context.Context, withdrawal *models.Withdrawal) error {
	return r.db.WithContext(ctx).Save(withdrawal).Error
}

// UpdateStatus 更新提现状态
func (r *WithdrawalRepository) UpdateStatus(ctx context.Context, id int64, status string, operatorID *int64) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if operatorID != nil {
		updates["operator_id"] = *operatorID
		updates["processed_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&models.Withdrawal{}).Where("id = ?", id).Updates(updates).Error
}

// Approve 审核通过
func (r *WithdrawalRepository) Approve(ctx context.Context, id int64, operatorID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("id = ? AND status = ?", id, models.WithdrawalStatusPending).
		Updates(map[string]interface{}{
			"status":       models.WithdrawalStatusApproved,
			"operator_id":  operatorID,
			"processed_at": now,
		}).Error
}

// Reject 审核拒绝
func (r *WithdrawalRepository) Reject(ctx context.Context, id int64, operatorID int64, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("id = ? AND status = ?", id, models.WithdrawalStatusPending).
		Updates(map[string]interface{}{
			"status":        models.WithdrawalStatusRejected,
			"operator_id":   operatorID,
			"processed_at":  now,
			"reject_reason": reason,
		}).Error
}

// MarkProcessing 标记为处理中
func (r *WithdrawalRepository) MarkProcessing(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("id = ? AND status = ?", id, models.WithdrawalStatusApproved).
		Update("status", models.WithdrawalStatusProcessing).Error
}

// MarkSuccess 标记为成功
func (r *WithdrawalRepository) MarkSuccess(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("id = ? AND status = ?", id, models.WithdrawalStatusProcessing).
		Update("status", models.WithdrawalStatusSuccess).Error
}

// List 获取提现记录列表
func (r *WithdrawalRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Withdrawal, int64, error) {
	var withdrawals []*models.Withdrawal
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Withdrawal{})

	// 应用过滤条件
	if userID, ok := filters["user_id"].(int64); ok && userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if withdrawType, ok := filters["type"].(string); ok && withdrawType != "" {
		query = query.Where("type = ?", withdrawType)
	}
	if withdrawTo, ok := filters["withdraw_to"].(string); ok && withdrawTo != "" {
		query = query.Where("withdraw_to = ?", withdrawTo)
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
		Preload("User").
		Preload("Operator").
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&withdrawals).Error; err != nil {
		return nil, 0, err
	}

	return withdrawals, total, nil
}

// GetPendingList 获取待审核列表
func (r *WithdrawalRepository) GetPendingList(ctx context.Context, offset, limit int) ([]*models.Withdrawal, int64, error) {
	var withdrawals []*models.Withdrawal
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Withdrawal{}).Where("status = ?", models.WithdrawalStatusPending)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("User").
		Order("id ASC"). // 按时间先后顺序
		Offset(offset).
		Limit(limit).
		Find(&withdrawals).Error; err != nil {
		return nil, 0, err
	}

	return withdrawals, total, nil
}

// GetApprovedList 获取已审批待打款列表
func (r *WithdrawalRepository) GetApprovedList(ctx context.Context, offset, limit int) ([]*models.Withdrawal, int64, error) {
	var withdrawals []*models.Withdrawal
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Withdrawal{}).Where("status = ?", models.WithdrawalStatusApproved)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("User").
		Order("id ASC").
		Offset(offset).
		Limit(limit).
		Find(&withdrawals).Error; err != nil {
		return nil, 0, err
	}

	return withdrawals, total, nil
}

// SumByUserID 统计用户的提现总额
func (r *WithdrawalRepository) SumByUserID(ctx context.Context, userID int64, status *string) (float64, error) {
	var sum float64
	query := r.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Select("COALESCE(SUM(actual_amount), 0)").
		Where("user_id = ?", userID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Scan(&sum).Error
	return sum, err
}

// CountByStatus 按状态统计提现数量
func (r *WithdrawalRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Withdrawal{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// CountPendingByUserID 统计用户待处理的提现数量
func (r *WithdrawalRepository) CountPendingByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("user_id = ? AND status IN ?", userID, []string{
			models.WithdrawalStatusPending,
			models.WithdrawalStatusApproved,
			models.WithdrawalStatusProcessing,
		}).
		Count(&count).Error
	return count, err
}

// GetStatsByUserID 获取用户提现统计
func (r *WithdrawalRepository) GetStatsByUserID(ctx context.Context, userID int64) (map[string]interface{}, error) {
	type Stats struct {
		TotalAmount     float64 `gorm:"column:total_amount"`
		SuccessAmount   float64 `gorm:"column:success_amount"`
		PendingAmount   float64 `gorm:"column:pending_amount"`
		TotalFee        float64 `gorm:"column:total_fee"`
		TotalCount      int64   `gorm:"column:total_count"`
		SuccessCount    int64   `gorm:"column:success_count"`
	}

	var stats Stats
	err := r.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Select(`
			COALESCE(SUM(amount), 0) as total_amount,
			COALESCE(SUM(CASE WHEN status = 'success' THEN actual_amount ELSE 0 END), 0) as success_amount,
			COALESCE(SUM(CASE WHEN status IN ('pending', 'approved', 'processing') THEN amount ELSE 0 END), 0) as pending_amount,
			COALESCE(SUM(CASE WHEN status = 'success' THEN fee ELSE 0 END), 0) as total_fee,
			COUNT(*) as total_count,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as success_count
		`).
		Where("user_id = ?", userID).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_amount":   stats.TotalAmount,
		"success_amount": stats.SuccessAmount,
		"pending_amount": stats.PendingAmount,
		"total_fee":      stats.TotalFee,
		"total_count":    stats.TotalCount,
		"success_count":  stats.SuccessCount,
	}, nil
}

// ExistsWithdrawalNo 检查提现单号是否存在
func (r *WithdrawalRepository) ExistsWithdrawalNo(ctx context.Context, withdrawalNo string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Withdrawal{}).Where("withdrawal_no = ?", withdrawalNo).Count(&count).Error
	return count > 0, err
}
