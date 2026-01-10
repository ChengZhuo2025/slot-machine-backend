// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// MessageTemplateRepository 消息模板仓储
type MessageTemplateRepository struct {
	db *gorm.DB
}

// NewMessageTemplateRepository 创建消息模板仓储
func NewMessageTemplateRepository(db *gorm.DB) *MessageTemplateRepository {
	return &MessageTemplateRepository{db: db}
}

// Create 创建模板
func (r *MessageTemplateRepository) Create(ctx context.Context, template *models.MessageTemplate) error {
	return r.db.WithContext(ctx).Create(template).Error
}

// GetByID 根据 ID 获取模板
func (r *MessageTemplateRepository) GetByID(ctx context.Context, id int64) (*models.MessageTemplate, error) {
	var template models.MessageTemplate
	err := r.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetByCode 根据编码获取模板
func (r *MessageTemplateRepository) GetByCode(ctx context.Context, code string) (*models.MessageTemplate, error) {
	var template models.MessageTemplate
	err := r.db.WithContext(ctx).Where("code = ? AND is_active = ?", code, true).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// Update 更新模板
func (r *MessageTemplateRepository) Update(ctx context.Context, template *models.MessageTemplate) error {
	return r.db.WithContext(ctx).Save(template).Error
}

// Delete 删除模板
func (r *MessageTemplateRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.MessageTemplate{}, id).Error
}

// List 获取模板列表
func (r *MessageTemplateRepository) List(ctx context.Context, offset, limit int, templateType string) ([]*models.MessageTemplate, int64, error) {
	var templates []*models.MessageTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&models.MessageTemplate{})

	if templateType != "" {
		query = query.Where("type = ?", templateType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// GetByType 获取指定类型的所有模板
func (r *MessageTemplateRepository) GetByType(ctx context.Context, templateType string) ([]*models.MessageTemplate, error) {
	var templates []*models.MessageTemplate
	err := r.db.WithContext(ctx).
		Where("type = ? AND is_active = ?", templateType, true).
		Find(&templates).Error
	return templates, err
}

// ExistsByCode 检查模板编码是否存在
func (r *MessageTemplateRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.MessageTemplate{}).
		Where("code = ?", code).
		Count(&count).Error
	return count > 0, err
}
