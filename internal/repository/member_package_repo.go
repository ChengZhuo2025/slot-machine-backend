// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// MemberPackageRepository 会员套餐仓储
type MemberPackageRepository struct {
	db *gorm.DB
}

// NewMemberPackageRepository 创建会员套餐仓储
func NewMemberPackageRepository(db *gorm.DB) *MemberPackageRepository {
	return &MemberPackageRepository{db: db}
}

// GetByID 根据 ID 获取会员套餐
func (r *MemberPackageRepository) GetByID(ctx context.Context, id int64) (*models.MemberPackage, error) {
	var pkg models.MemberPackage
	err := r.db.WithContext(ctx).Preload("MemberLevel").First(&pkg, id).Error
	if err != nil {
		return nil, err
	}
	return &pkg, nil
}

// GetAll 获取所有会员套餐（按排序和ID）
func (r *MemberPackageRepository) GetAll(ctx context.Context) ([]*models.MemberPackage, error) {
	var packages []*models.MemberPackage
	err := r.db.WithContext(ctx).
		Preload("MemberLevel").
		Order("sort DESC, id ASC").
		Find(&packages).Error
	if err != nil {
		return nil, err
	}
	return packages, nil
}

// GetActive 获取所有启用的会员套餐
func (r *MemberPackageRepository) GetActive(ctx context.Context) ([]*models.MemberPackage, error) {
	var packages []*models.MemberPackage
	err := r.db.WithContext(ctx).
		Preload("MemberLevel").
		Where("status = ?", models.MemberPackageStatusActive).
		Order("sort DESC, id ASC").
		Find(&packages).Error
	if err != nil {
		return nil, err
	}
	return packages, nil
}

// GetByLevelID 根据目标等级ID获取会员套餐
func (r *MemberPackageRepository) GetByLevelID(ctx context.Context, levelID int64) ([]*models.MemberPackage, error) {
	var packages []*models.MemberPackage
	err := r.db.WithContext(ctx).
		Preload("MemberLevel").
		Where("member_level_id = ? AND status = ?", levelID, models.MemberPackageStatusActive).
		Order("sort DESC, id ASC").
		Find(&packages).Error
	if err != nil {
		return nil, err
	}
	return packages, nil
}

// Create 创建会员套餐
func (r *MemberPackageRepository) Create(ctx context.Context, pkg *models.MemberPackage) error {
	return r.db.WithContext(ctx).Create(pkg).Error
}

// Update 更新会员套餐
func (r *MemberPackageRepository) Update(ctx context.Context, pkg *models.MemberPackage) error {
	return r.db.WithContext(ctx).Save(pkg).Error
}

// UpdateStatus 更新套餐状态
func (r *MemberPackageRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.MemberPackage{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Delete 删除会员套餐
func (r *MemberPackageRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.MemberPackage{}, id).Error
}

// List 分页获取会员套餐
func (r *MemberPackageRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.MemberPackage, int64, error) {
	var packages []*models.MemberPackage
	var total int64

	query := r.db.WithContext(ctx).Model(&models.MemberPackage{})

	// 应用过滤条件
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}
	if levelID, ok := filters["member_level_id"].(int64); ok && levelID > 0 {
		query = query.Where("member_level_id = ?", levelID)
	}
	if isRecommend, ok := filters["is_recommend"].(bool); ok {
		query = query.Where("is_recommend = ?", isRecommend)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Preload("MemberLevel").
		Order("sort DESC, id ASC").
		Offset(offset).Limit(limit).
		Find(&packages).Error; err != nil {
		return nil, 0, err
	}

	return packages, total, nil
}

// GetRecommended 获取推荐的会员套餐
func (r *MemberPackageRepository) GetRecommended(ctx context.Context) ([]*models.MemberPackage, error) {
	var packages []*models.MemberPackage
	err := r.db.WithContext(ctx).
		Preload("MemberLevel").
		Where("status = ? AND is_recommend = ?", models.MemberPackageStatusActive, true).
		Order("sort DESC, id ASC").
		Find(&packages).Error
	if err != nil {
		return nil, err
	}
	return packages, nil
}

// Count 获取会员套餐总数
func (r *MemberPackageRepository) Count(ctx context.Context, status *int8) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.MemberPackage{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	err := query.Count(&count).Error
	return count, err
}
