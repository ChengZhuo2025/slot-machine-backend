// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// CategoryRepository 商品分类仓储
type CategoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository 创建分类仓储
func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// Create 创建分类
func (r *CategoryRepository) Create(ctx context.Context, category *models.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

// GetByID 根据 ID 获取分类
func (r *CategoryRepository) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	var category models.Category
	err := r.db.WithContext(ctx).First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetByIDWithChildren 根据 ID 获取分类（包含子分类）
func (r *CategoryRepository) GetByIDWithChildren(ctx context.Context, id int64) (*models.Category, error) {
	var category models.Category
	err := r.db.WithContext(ctx).Preload("Children").First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// Update 更新分类
func (r *CategoryRepository) Update(ctx context.Context, category *models.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

// UpdateFields 更新指定字段
func (r *CategoryRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Category{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除分类
func (r *CategoryRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Category{}, id).Error
}

// List 获取分类列表
func (r *CategoryRepository) List(ctx context.Context, filters map[string]interface{}) ([]*models.Category, error) {
	var categories []*models.Category

	query := r.db.WithContext(ctx).Model(&models.Category{})

	// 过滤条件
	if parentID, ok := filters["parent_id"]; ok {
		if parentID == nil {
			query = query.Where("parent_id IS NULL")
		} else {
			query = query.Where("parent_id = ?", parentID)
		}
	}
	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where("is_active = ?", isActive)
	}
	if level, ok := filters["level"].(int); ok && level > 0 {
		query = query.Where("level = ?", level)
	}

	if err := query.Order("sort DESC, id ASC").Find(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

// ListActive 获取启用的分类列表
func (r *CategoryRepository) ListActive(ctx context.Context) ([]*models.Category, error) {
	return r.List(ctx, map[string]interface{}{"is_active": true})
}

// ListRootCategories 获取顶级分类（parent_id = NULL）
func (r *CategoryRepository) ListRootCategories(ctx context.Context) ([]*models.Category, error) {
	return r.List(ctx, map[string]interface{}{"parent_id": nil, "is_active": true})
}

// ListByParentID 根据父分类ID获取子分类列表
func (r *CategoryRepository) ListByParentID(ctx context.Context, parentID int64) ([]*models.Category, error) {
	var categories []*models.Category
	err := r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("sort DESC, id ASC").
		Find(&categories).Error
	return categories, err
}

// ListWithChildren 获取分类树（包含子分类）
func (r *CategoryRepository) ListWithChildren(ctx context.Context, parentID *int64) ([]*models.Category, error) {
	var categories []*models.Category

	query := r.db.WithContext(ctx).Preload("Children", func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true).Order("sort DESC, id ASC")
	}).Where("is_active = ?", true)

	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	if err := query.Order("sort DESC, id ASC").Find(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

// GetCategoryTree 获取完整分类树
func (r *CategoryRepository) GetCategoryTree(ctx context.Context) ([]*models.Category, error) {
	// 获取所有启用的分类
	var categories []*models.Category
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("level ASC, sort DESC, id ASC").
		Find(&categories).Error; err != nil {
		return nil, err
	}

	// 构建树结构
	categoryMap := make(map[int64]*models.Category)
	for _, cat := range categories {
		categoryMap[cat.ID] = cat
	}

	var roots []*models.Category
	for _, cat := range categories {
		if cat.ParentID == nil {
			roots = append(roots, cat)
		} else if parent, ok := categoryMap[*cat.ParentID]; ok {
			parent.Children = append(parent.Children, *cat)
		}
	}

	return roots, nil
}

// HasProducts 检查分类是否有商品
func (r *CategoryRepository) HasProducts(ctx context.Context, id int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Product{}).Where("category_id = ?", id).Count(&count).Error
	return count > 0, err
}

// HasChildren 检查分类是否有子分类
func (r *CategoryRepository) HasChildren(ctx context.Context, id int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Category{}).Where("parent_id = ?", id).Count(&count).Error
	return count > 0, err
}

// GetPath 获取分类路径（从根到当前）
func (r *CategoryRepository) GetPath(ctx context.Context, id int64) ([]*models.Category, error) {
	var path []*models.Category
	currentID := id

	for {
		category, err := r.GetByID(ctx, currentID)
		if err != nil {
			return nil, err
		}
		path = append([]*models.Category{category}, path...)

		if category.ParentID == nil {
			break
		}
		currentID = *category.ParentID
	}

	return path, nil
}
