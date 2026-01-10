// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// ArticleRepository 文章仓储
type ArticleRepository struct {
	db *gorm.DB
}

// NewArticleRepository 创建文章仓储
func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

// Create 创建文章
func (r *ArticleRepository) Create(ctx context.Context, article *models.Article) error {
	return r.db.WithContext(ctx).Create(article).Error
}

// GetByID 根据 ID 获取文章
func (r *ArticleRepository) GetByID(ctx context.Context, id int64) (*models.Article, error) {
	var article models.Article
	err := r.db.WithContext(ctx).First(&article, id).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// Update 更新文章
func (r *ArticleRepository) Update(ctx context.Context, article *models.Article) error {
	return r.db.WithContext(ctx).Save(article).Error
}

// UpdateFields 更新指定字段
func (r *ArticleRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Article{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除文章
func (r *ArticleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Article{}, id).Error
}

// ArticleListFilters 文章列表筛选条件
type ArticleListFilters struct {
	Category    string
	IsPublished *bool
	Keyword     string
}

// List 获取文章列表
func (r *ArticleRepository) List(ctx context.Context, offset, limit int, filters *ArticleListFilters) ([]*models.Article, int64, error) {
	var articles []*models.Article
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Article{})

	if filters != nil {
		if filters.Category != "" {
			query = query.Where("category = ?", filters.Category)
		}
		if filters.IsPublished != nil {
			query = query.Where("is_published = ?", *filters.IsPublished)
		}
		if filters.Keyword != "" {
			keyword := "%" + filters.Keyword + "%"
			query = query.Where("title LIKE ? OR content LIKE ?", keyword, keyword)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("sort DESC, id DESC").Offset(offset).Limit(limit).Find(&articles).Error; err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// ListPublished 获取已发布的文章列表
func (r *ArticleRepository) ListPublished(ctx context.Context, category string, offset, limit int) ([]*models.Article, int64, error) {
	isPublished := true
	filters := &ArticleListFilters{
		Category:    category,
		IsPublished: &isPublished,
	}
	return r.List(ctx, offset, limit, filters)
}

// IncrementViewCount 增加浏览量
func (r *ArticleRepository) IncrementViewCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Article{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// Publish 发布文章
func (r *ArticleRepository) Publish(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Article{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_published":  true,
			"published_at":  now,
		}).Error
}

// Unpublish 取消发布文章
func (r *ArticleRepository) Unpublish(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&models.Article{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_published": false,
		}).Error
}

// CountByCategory 按分类统计文章数量
func (r *ArticleRepository) CountByCategory(ctx context.Context) (map[string]int64, error) {
	type Result struct {
		Category string
		Count    int64
	}

	var results []Result
	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, r := range results {
		counts[r.Category] = r.Count
	}
	return counts, nil
}

// GetByCategoryAndSort 按分类和排序获取文章
func (r *ArticleRepository) GetByCategoryAndSort(ctx context.Context, category string, limit int) ([]*models.Article, error) {
	var articles []*models.Article
	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Where("category = ? AND is_published = ?", category, true).
		Order("sort DESC, id DESC").
		Limit(limit).
		Find(&articles).Error
	return articles, err
}
