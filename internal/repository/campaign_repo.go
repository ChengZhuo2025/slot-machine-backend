// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// CampaignRepository 活动仓储
type CampaignRepository struct {
	db *gorm.DB
}

// NewCampaignRepository 创建活动仓储
func NewCampaignRepository(db *gorm.DB) *CampaignRepository {
	return &CampaignRepository{db: db}
}

// Create 创建活动
func (r *CampaignRepository) Create(ctx context.Context, campaign *models.Campaign) error {
	return r.db.WithContext(ctx).Create(campaign).Error
}

// GetByID 根据 ID 获取活动
func (r *CampaignRepository) GetByID(ctx context.Context, id int64) (*models.Campaign, error) {
	var campaign models.Campaign
	err := r.db.WithContext(ctx).First(&campaign, id).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// Update 更新活动
func (r *CampaignRepository) Update(ctx context.Context, campaign *models.Campaign) error {
	return r.db.WithContext(ctx).Save(campaign).Error
}

// UpdateFields 更新指定字段
func (r *CampaignRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Campaign{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除活动
func (r *CampaignRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Campaign{}, id).Error
}

// CampaignListParams 活动列表查询参数
type CampaignListParams struct {
	Offset        int
	Limit         int
	Status        *int8
	Type          string
	Keyword       string
	StartTimeFrom *time.Time
	StartTimeTo   *time.Time
	EndTimeFrom   *time.Time
	EndTimeTo     *time.Time
}

// List 获取活动列表
func (r *CampaignRepository) List(ctx context.Context, params CampaignListParams) ([]*models.Campaign, int64, error) {
	var campaigns []*models.Campaign
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Campaign{})

	// 过滤条件
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+params.Keyword+"%")
	}
	if params.StartTimeFrom != nil {
		query = query.Where("start_time >= ?", *params.StartTimeFrom)
	}
	if params.StartTimeTo != nil {
		query = query.Where("start_time <= ?", *params.StartTimeTo)
	}
	if params.EndTimeFrom != nil {
		query = query.Where("end_time >= ?", *params.EndTimeFrom)
	}
	if params.EndTimeTo != nil {
		query = query.Where("end_time <= ?", *params.EndTimeTo)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("created_at DESC").Offset(params.Offset).Limit(params.Limit).Find(&campaigns).Error; err != nil {
		return nil, 0, err
	}

	return campaigns, total, nil
}

// ListActive 获取有效活动列表（用户端）
func (r *CampaignRepository) ListActive(ctx context.Context, offset, limit int) ([]*models.Campaign, int64, error) {
	var campaigns []*models.Campaign
	var total int64
	now := time.Now()

	query := r.db.WithContext(ctx).Model(&models.Campaign{}).
		Where("status = ?", models.CampaignStatusActive).
		Where("start_time <= ?", now).
		Where("end_time >= ?", now)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&campaigns).Error; err != nil {
		return nil, 0, err
	}

	return campaigns, total, nil
}

// ListByType 根据类型获取有效活动
func (r *CampaignRepository) ListByType(ctx context.Context, campaignType string) ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("status = ?", models.CampaignStatusActive).
		Where("start_time <= ?", now).
		Where("end_time >= ?", now).
		Where("type = ?", campaignType).
		Order("created_at DESC").
		Find(&campaigns).Error

	return campaigns, err
}

// GetActiveByType 获取单个有效活动（根据类型）
func (r *CampaignRepository) GetActiveByType(ctx context.Context, campaignType string) (*models.Campaign, error) {
	var campaign models.Campaign
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("status = ?", models.CampaignStatusActive).
		Where("start_time <= ?", now).
		Where("end_time >= ?", now).
		Where("type = ?", campaignType).
		Order("created_at DESC").
		First(&campaign).Error

	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// UpdateStatus 更新活动状态
func (r *CampaignRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Campaign{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// CountByStatus 统计各状态活动数量
func (r *CampaignRepository) CountByStatus(ctx context.Context, status int8) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Campaign{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// CountActive 统计当前进行中的活动数量
func (r *CampaignRepository) CountActive(ctx context.Context) (int64, error) {
	var count int64
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&models.Campaign{}).
		Where("status = ?", models.CampaignStatusActive).
		Where("start_time <= ?", now).
		Where("end_time >= ?", now).
		Count(&count).Error
	return count, err
}
