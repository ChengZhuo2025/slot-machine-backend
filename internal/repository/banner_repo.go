// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// BannerRepository 轮播图仓储
type BannerRepository struct {
	db *gorm.DB
}

// NewBannerRepository 创建轮播图仓储
func NewBannerRepository(db *gorm.DB) *BannerRepository {
	return &BannerRepository{db: db}
}

// Create 创建轮播图
func (r *BannerRepository) Create(ctx context.Context, banner *models.Banner) error {
	return r.db.WithContext(ctx).Create(banner).Error
}

// GetByID 根据 ID 获取轮播图
func (r *BannerRepository) GetByID(ctx context.Context, id int64) (*models.Banner, error) {
	var banner models.Banner
	err := r.db.WithContext(ctx).First(&banner, id).Error
	if err != nil {
		return nil, err
	}
	return &banner, nil
}

// Update 更新轮播图
func (r *BannerRepository) Update(ctx context.Context, banner *models.Banner) error {
	return r.db.WithContext(ctx).Save(banner).Error
}

// Delete 删除轮播图
func (r *BannerRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Banner{}, id).Error
}

// BannerListFilters 轮播图列表筛选条件
type BannerListFilters struct {
	Position string
	IsActive *bool
	Keyword  string
}

// List 获取轮播图列表（管理端）
func (r *BannerRepository) List(ctx context.Context, offset, limit int, filters *BannerListFilters) ([]*models.Banner, int64, error) {
	var banners []*models.Banner
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Banner{})

	if filters != nil {
		if filters.Position != "" {
			query = query.Where("position = ?", filters.Position)
		}
		if filters.IsActive != nil {
			query = query.Where("is_active = ?", *filters.IsActive)
		}
		if filters.Keyword != "" {
			keyword := "%" + filters.Keyword + "%"
			query = query.Where("title LIKE ?", keyword)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("sort DESC, id DESC").Offset(offset).Limit(limit).Find(&banners).Error; err != nil {
		return nil, 0, err
	}

	return banners, total, nil
}

// ListByPosition 获取指定位置的有效轮播图（用户端）
func (r *BannerRepository) ListByPosition(ctx context.Context, position string, limit int) ([]*models.Banner, error) {
	var banners []*models.Banner
	now := time.Now()

	query := r.db.WithContext(ctx).Model(&models.Banner{}).
		Where("position = ? AND is_active = ?", position, true).
		Where("(start_time IS NULL OR start_time <= ?)", now).
		Where("(end_time IS NULL OR end_time >= ?)", now).
		Order("sort DESC, id DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&banners).Error
	return banners, err
}

// UpdateStatus 更新轮播图状态
func (r *BannerRepository) UpdateStatus(ctx context.Context, id int64, isActive bool) error {
	return r.db.WithContext(ctx).Model(&models.Banner{}).
		Where("id = ?", id).
		Update("is_active", isActive).Error
}

// UpdateSort 更新排序
func (r *BannerRepository) UpdateSort(ctx context.Context, id int64, sort int) error {
	return r.db.WithContext(ctx).Model(&models.Banner{}).
		Where("id = ?", id).
		Update("sort", sort).Error
}

// IncrementClickCount 增加点击次数
func (r *BannerRepository) IncrementClickCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Banner{}).
		Where("id = ?", id).
		UpdateColumn("click_count", gorm.Expr("click_count + 1")).Error
}

// CountByPosition 按位置统计轮播图数量
func (r *BannerRepository) CountByPosition(ctx context.Context) (map[string]int64, error) {
	type Result struct {
		Position string
		Count    int64
	}

	var results []Result
	err := r.db.WithContext(ctx).Model(&models.Banner{}).
		Select("position, COUNT(*) as count").
		Group("position").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, r := range results {
		counts[r.Position] = r.Count
	}
	return counts, nil
}

// GetActiveCount 获取启用的轮播图数量
func (r *BannerRepository) GetActiveCount(ctx context.Context, position string) (int64, error) {
	var count int64
	now := time.Now()

	query := r.db.WithContext(ctx).Model(&models.Banner{}).
		Where("is_active = ?", true).
		Where("(start_time IS NULL OR start_time <= ?)", now).
		Where("(end_time IS NULL OR end_time >= ?)", now)

	if position != "" {
		query = query.Where("position = ?", position)
	}

	err := query.Count(&count).Error
	return count, err
}
