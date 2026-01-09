// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// MemberLevelRepository 会员等级仓储
type MemberLevelRepository struct {
	db *gorm.DB
}

// NewMemberLevelRepository 创建会员等级仓储
func NewMemberLevelRepository(db *gorm.DB) *MemberLevelRepository {
	return &MemberLevelRepository{db: db}
}

// GetByID 根据 ID 获取会员等级
func (r *MemberLevelRepository) GetByID(ctx context.Context, id int64) (*models.MemberLevel, error) {
	var level models.MemberLevel
	err := r.db.WithContext(ctx).First(&level, id).Error
	if err != nil {
		return nil, err
	}
	return &level, nil
}

// GetByLevel 根据等级序号获取会员等级
func (r *MemberLevelRepository) GetByLevel(ctx context.Context, level int) (*models.MemberLevel, error) {
	var memberLevel models.MemberLevel
	err := r.db.WithContext(ctx).Where("level = ?", level).First(&memberLevel).Error
	if err != nil {
		return nil, err
	}
	return &memberLevel, nil
}

// GetAll 获取所有会员等级（按等级序号升序）
func (r *MemberLevelRepository) GetAll(ctx context.Context) ([]*models.MemberLevel, error) {
	var levels []*models.MemberLevel
	err := r.db.WithContext(ctx).Order("level ASC").Find(&levels).Error
	if err != nil {
		return nil, err
	}
	return levels, nil
}

// GetByMinPoints 根据积分获取对应的会员等级（最高匹配）
func (r *MemberLevelRepository) GetByMinPoints(ctx context.Context, points int) (*models.MemberLevel, error) {
	var level models.MemberLevel
	err := r.db.WithContext(ctx).
		Where("min_points <= ?", points).
		Order("min_points DESC").
		First(&level).Error
	if err != nil {
		return nil, err
	}
	return &level, nil
}

// Create 创建会员等级
func (r *MemberLevelRepository) Create(ctx context.Context, level *models.MemberLevel) error {
	return r.db.WithContext(ctx).Create(level).Error
}

// Update 更新会员等级
func (r *MemberLevelRepository) Update(ctx context.Context, level *models.MemberLevel) error {
	return r.db.WithContext(ctx).Save(level).Error
}

// Delete 删除会员等级
func (r *MemberLevelRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.MemberLevel{}, id).Error
}

// GetNextLevel 获取下一个会员等级（根据当前等级序号）
func (r *MemberLevelRepository) GetNextLevel(ctx context.Context, currentLevel int) (*models.MemberLevel, error) {
	var level models.MemberLevel
	err := r.db.WithContext(ctx).
		Where("level > ?", currentLevel).
		Order("level ASC").
		First(&level).Error
	if err != nil {
		return nil, err
	}
	return &level, nil
}

// GetDefaultLevel 获取默认会员等级（等级1）
func (r *MemberLevelRepository) GetDefaultLevel(ctx context.Context) (*models.MemberLevel, error) {
	var level models.MemberLevel
	err := r.db.WithContext(ctx).Where("level = 1").First(&level).Error
	if err != nil {
		return nil, err
	}
	return &level, nil
}

// Count 获取会员等级总数
func (r *MemberLevelRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.MemberLevel{}).Count(&count).Error
	return count, err
}
