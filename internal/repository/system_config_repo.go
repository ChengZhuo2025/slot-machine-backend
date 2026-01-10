// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// SystemConfigRepository 系统配置仓储
type SystemConfigRepository struct {
	db *gorm.DB
}

// NewSystemConfigRepository 创建系统配置仓储
func NewSystemConfigRepository(db *gorm.DB) *SystemConfigRepository {
	return &SystemConfigRepository{db: db}
}

// Create 创建配置
func (r *SystemConfigRepository) Create(ctx context.Context, config *models.SystemConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

// GetByID 根据 ID 获取配置
func (r *SystemConfigRepository) GetByID(ctx context.Context, id int64) (*models.SystemConfig, error) {
	var config models.SystemConfig
	err := r.db.WithContext(ctx).First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByGroupAndKey 根据分组和键获取配置
func (r *SystemConfigRepository) GetByGroupAndKey(ctx context.Context, group, key string) (*models.SystemConfig, error) {
	var config models.SystemConfig
	err := r.db.WithContext(ctx).
		Where("\"group\" = ? AND \"key\" = ?", group, key).
		First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByGroup 获取分组下的所有配置
func (r *SystemConfigRepository) GetByGroup(ctx context.Context, group string) ([]*models.SystemConfig, error) {
	var configs []*models.SystemConfig
	err := r.db.WithContext(ctx).
		Where("\"group\" = ?", group).
		Order("id ASC").
		Find(&configs).Error
	return configs, err
}

// Update 更新配置
func (r *SystemConfigRepository) Update(ctx context.Context, config *models.SystemConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

// UpdateValue 更新配置值
func (r *SystemConfigRepository) UpdateValue(ctx context.Context, id int64, value string) error {
	return r.db.WithContext(ctx).Model(&models.SystemConfig{}).
		Where("id = ?", id).
		Update("value", value).Error
}

// Delete 删除配置
func (r *SystemConfigRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.SystemConfig{}, id).Error
}

// SystemConfigListFilters 配置列表筛选条件
type SystemConfigListFilters struct {
	Group    string
	Keyword  string
	IsPublic *bool
}

// List 获取配置列表
func (r *SystemConfigRepository) List(ctx context.Context, offset, limit int, filters *SystemConfigListFilters) ([]*models.SystemConfig, int64, error) {
	var configs []*models.SystemConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.SystemConfig{})

	if filters != nil {
		if filters.Group != "" {
			query = query.Where("\"group\" = ?", filters.Group)
		}
		if filters.Keyword != "" {
			keyword := "%" + filters.Keyword + "%"
			query = query.Where("\"key\" LIKE ? OR description LIKE ?", keyword, keyword)
		}
		if filters.IsPublic != nil {
			query = query.Where("is_public = ?", *filters.IsPublic)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("\"group\" ASC, id ASC").Offset(offset).Limit(limit).Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// GetPublicConfigs 获取所有公开配置
func (r *SystemConfigRepository) GetPublicConfigs(ctx context.Context) ([]*models.SystemConfig, error) {
	var configs []*models.SystemConfig
	err := r.db.WithContext(ctx).
		Where("is_public = ?", true).
		Order("\"group\" ASC, id ASC").
		Find(&configs).Error
	return configs, err
}

// GetAllGroups 获取所有配置分组
func (r *SystemConfigRepository) GetAllGroups(ctx context.Context) ([]string, error) {
	var groups []string
	err := r.db.WithContext(ctx).Model(&models.SystemConfig{}).
		Distinct("\"group\"").
		Pluck("\"group\"", &groups).Error
	return groups, err
}

// BatchUpsert 批量创建或更新配置
func (r *SystemConfigRepository) BatchUpsert(ctx context.Context, configs []*models.SystemConfig) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, config := range configs {
			var existing models.SystemConfig
			err := tx.Where("\"group\" = ? AND \"key\" = ?", config.Group, config.Key).First(&existing).Error
			if err == nil {
				// 更新
				config.ID = existing.ID
				if err := tx.Save(config).Error; err != nil {
					return err
				}
			} else if err == gorm.ErrRecordNotFound {
				// 创建
				if err := tx.Create(config).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		return nil
	})
}

// ExistsByGroupAndKey 检查配置是否存在
func (r *SystemConfigRepository) ExistsByGroupAndKey(ctx context.Context, group, key string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.SystemConfig{}).
		Where("\"group\" = ? AND \"key\" = ?", group, key).
		Count(&count).Error
	return count > 0, err
}
