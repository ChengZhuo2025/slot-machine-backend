// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// DistributorRepository 分销商仓储
type DistributorRepository struct {
	db *gorm.DB
}

// NewDistributorRepository 创建分销商仓储
func NewDistributorRepository(db *gorm.DB) *DistributorRepository {
	return &DistributorRepository{db: db}
}

// Create 创建分销商
func (r *DistributorRepository) Create(ctx context.Context, distributor *models.Distributor) error {
	return r.db.WithContext(ctx).Create(distributor).Error
}

// GetByID 根据 ID 获取分销商
func (r *DistributorRepository) GetByID(ctx context.Context, id int64) (*models.Distributor, error) {
	var distributor models.Distributor
	err := r.db.WithContext(ctx).First(&distributor, id).Error
	if err != nil {
		return nil, err
	}
	return &distributor, nil
}

// GetByIDWithUser 根据 ID 获取分销商（包含用户信息）
func (r *DistributorRepository) GetByIDWithUser(ctx context.Context, id int64) (*models.Distributor, error) {
	var distributor models.Distributor
	err := r.db.WithContext(ctx).Preload("User").First(&distributor, id).Error
	if err != nil {
		return nil, err
	}
	return &distributor, nil
}

// GetByUserID 根据用户 ID 获取分销商
func (r *DistributorRepository) GetByUserID(ctx context.Context, userID int64) (*models.Distributor, error) {
	var distributor models.Distributor
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&distributor).Error
	if err != nil {
		return nil, err
	}
	return &distributor, nil
}

// GetByInviteCode 根据邀请码获取分销商
func (r *DistributorRepository) GetByInviteCode(ctx context.Context, inviteCode string) (*models.Distributor, error) {
	var distributor models.Distributor
	err := r.db.WithContext(ctx).Where("invite_code = ?", inviteCode).First(&distributor).Error
	if err != nil {
		return nil, err
	}
	return &distributor, nil
}

// GetByInviteCodeWithUser 根据邀请码获取分销商（包含用户信息）
func (r *DistributorRepository) GetByInviteCodeWithUser(ctx context.Context, inviteCode string) (*models.Distributor, error) {
	var distributor models.Distributor
	err := r.db.WithContext(ctx).Preload("User").Where("invite_code = ?", inviteCode).First(&distributor).Error
	if err != nil {
		return nil, err
	}
	return &distributor, nil
}

// Update 更新分销商
func (r *DistributorRepository) Update(ctx context.Context, distributor *models.Distributor) error {
	return r.db.WithContext(ctx).Save(distributor).Error
}

// UpdateFields 更新指定字段
func (r *DistributorRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Distributor{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新分销商状态
func (r *DistributorRepository) UpdateStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&models.Distributor{}).Where("id = ?", id).Update("status", status).Error
}

// ExistsByUserID 检查用户是否已经是分销商
func (r *DistributorRepository) ExistsByUserID(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Distributor{}).Where("user_id = ?", userID).Count(&count).Error
	return count > 0, err
}

// ExistsByInviteCode 检查邀请码是否已存在
func (r *DistributorRepository) ExistsByInviteCode(ctx context.Context, inviteCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Distributor{}).Where("invite_code = ?", inviteCode).Count(&count).Error
	return count > 0, err
}

// List 获取分销商列表
func (r *DistributorRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Distributor, int64, error) {
	var distributors []*models.Distributor
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Distributor{})

	// 应用过滤条件
	if status, ok := filters["status"].(int); ok && status >= 0 {
		query = query.Where("status = ?", status)
	}
	if parentID, ok := filters["parent_id"].(int64); ok && parentID > 0 {
		query = query.Where("parent_id = ?", parentID)
	}
	if level, ok := filters["level"].(int); ok && level > 0 {
		query = query.Where("level = ?", level)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表（包含用户信息）
	if err := query.Preload("User").Order("id DESC").Offset(offset).Limit(limit).Find(&distributors).Error; err != nil {
		return nil, 0, err
	}

	return distributors, total, nil
}

// ListByParentID 获取下级分销商列表
func (r *DistributorRepository) ListByParentID(ctx context.Context, parentID int64, offset, limit int) ([]*models.Distributor, int64, error) {
	var distributors []*models.Distributor
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Distributor{}).Where("parent_id = ?", parentID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("User").Order("id DESC").Offset(offset).Limit(limit).Find(&distributors).Error; err != nil {
		return nil, 0, err
	}

	return distributors, total, nil
}

// GetTeamMembers 获取团队成员（直推和间推）
func (r *DistributorRepository) GetTeamMembers(ctx context.Context, distributorID int64, offset, limit int) ([]*models.Distributor, int64, error) {
	var distributors []*models.Distributor
	var total int64

	// 获取直推成员
	query := r.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("parent_id = ? OR parent_id IN (SELECT id FROM distributors WHERE parent_id = ?)", distributorID, distributorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("User").Order("id DESC").Offset(offset).Limit(limit).Find(&distributors).Error; err != nil {
		return nil, 0, err
	}

	return distributors, total, nil
}

// GetDirectMembers 获取直推成员
func (r *DistributorRepository) GetDirectMembers(ctx context.Context, distributorID int64, offset, limit int) ([]*models.Distributor, int64, error) {
	var distributors []*models.Distributor
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Distributor{}).Where("parent_id = ?", distributorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("User").Order("id DESC").Offset(offset).Limit(limit).Find(&distributors).Error; err != nil {
		return nil, 0, err
	}

	return distributors, total, nil
}

// AddCommission 增加可用佣金
func (r *DistributorRepository) AddCommission(ctx context.Context, distributorID int64, amount float64) error {
	return r.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("id = ?", distributorID).
		Updates(map[string]interface{}{
			"total_commission":     gorm.Expr("total_commission + ?", amount),
			"available_commission": gorm.Expr("available_commission + ?", amount),
		}).Error
}

// FreezeCommission 冻结佣金（提现时）
func (r *DistributorRepository) FreezeCommission(ctx context.Context, distributorID int64, amount float64) error {
	result := r.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("id = ? AND available_commission >= ?", distributorID, amount).
		Updates(map[string]interface{}{
			"available_commission": gorm.Expr("available_commission - ?", amount),
			"frozen_commission":    gorm.Expr("frozen_commission + ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UnfreezeCommission 解冻佣金（提现失败时）
func (r *DistributorRepository) UnfreezeCommission(ctx context.Context, distributorID int64, amount float64) error {
	result := r.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("id = ? AND frozen_commission >= ?", distributorID, amount).
		Updates(map[string]interface{}{
			"available_commission": gorm.Expr("available_commission + ?", amount),
			"frozen_commission":    gorm.Expr("frozen_commission - ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ConfirmWithdraw 确认提现（提现成功时）
func (r *DistributorRepository) ConfirmWithdraw(ctx context.Context, distributorID int64, amount float64) error {
	result := r.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("id = ? AND frozen_commission >= ?", distributorID, amount).
		Updates(map[string]interface{}{
			"frozen_commission":    gorm.Expr("frozen_commission - ?", amount),
			"withdrawn_commission": gorm.Expr("withdrawn_commission + ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IncrementTeamCount 增加团队人数
func (r *DistributorRepository) IncrementTeamCount(ctx context.Context, distributorID int64) error {
	return r.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("id = ?", distributorID).
		UpdateColumn("team_count", gorm.Expr("team_count + 1")).
		Error
}

// IncrementDirectCount 增加直推人数
func (r *DistributorRepository) IncrementDirectCount(ctx context.Context, distributorID int64) error {
	return r.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("id = ?", distributorID).
		UpdateColumn("direct_count", gorm.Expr("direct_count + 1")).
		Error
}

// GetPendingList 获取待审核列表
func (r *DistributorRepository) GetPendingList(ctx context.Context, offset, limit int) ([]*models.Distributor, int64, error) {
	var distributors []*models.Distributor
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Distributor{}).Where("status = ?", models.DistributorStatusPending)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("User").Order("id DESC").Offset(offset).Limit(limit).Find(&distributors).Error; err != nil {
		return nil, 0, err
	}

	return distributors, total, nil
}

// CountByStatus 按状态统计分销商数量
func (r *DistributorRepository) CountByStatus(ctx context.Context, status int) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Distributor{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// GetTopDistributors 获取佣金排行榜
func (r *DistributorRepository) GetTopDistributors(ctx context.Context, limit int) ([]*models.Distributor, error) {
	var distributors []*models.Distributor
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("status = ?", models.DistributorStatusApproved).
		Order("total_commission DESC").
		Limit(limit).
		Find(&distributors).Error
	return distributors, err
}
