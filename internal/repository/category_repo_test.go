// Package repository 分类仓储单元测试
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupCategoryTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Category{}, &models.Product{})
	require.NoError(t, err)

	return db
}

func TestCategoryRepository_Create(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	category := &models.Category{
		Name:     "测试分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}

	err := repo.Create(ctx, category)
	require.NoError(t, err)
	assert.NotZero(t, category.ID)
}

func TestCategoryRepository_GetByID(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	category := &models.Category{
		Name:     "测试分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(category)

	found, err := repo.GetByID(ctx, category.ID)
	require.NoError(t, err)
	assert.Equal(t, category.ID, found.ID)
	assert.Equal(t, "测试分类", found.Name)
}

func TestCategoryRepository_GetByIDWithChildren(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	parent := &models.Category{
		Name:     "父分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(parent)

	child := &models.Category{
		Name:     "子分类",
		ParentID: &parent.ID,
		Level:    2,
		Sort:     50,
		IsActive: true,
	}
	db.Create(child)

	found, err := repo.GetByIDWithChildren(ctx, parent.ID)
	require.NoError(t, err)
	assert.Equal(t, parent.ID, found.ID)
	assert.Equal(t, 1, len(found.Children))
}

func TestCategoryRepository_Update(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	category := &models.Category{
		Name:     "原分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(category)

	category.Name = "新分类"
	err := repo.Update(ctx, category)
	require.NoError(t, err)

	var found models.Category
	db.First(&found, category.ID)
	assert.Equal(t, "新分类", found.Name)
}

func TestCategoryRepository_UpdateFields(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	category := &models.Category{
		Name:     "测试分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(category)

	err := repo.UpdateFields(ctx, category.ID, map[string]interface{}{
		"sort": 200,
	})
	require.NoError(t, err)

	var found models.Category
	db.First(&found, category.ID)
	assert.Equal(t, 200, found.Sort)
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	category := &models.Category{
		Name:     "待删除分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(category)

	err := repo.Delete(ctx, category.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Category{}).Where("id = ?", category.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCategoryRepository_List(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	categories := []*models.Category{
		{Name: "分类1", Level: 1, Sort: 100, IsActive: true},
		{Name: "分类2", Level: 1, Sort: 90, IsActive: true},
	}
	for _, c := range categories {
		db.Create(c)
	}

	db.Model(&models.Category{}).Create(map[string]interface{}{
		"name":      "禁用分类",
		"level":     1,
		"sort":      80,
		"is_active": false,
	})

	list, err := repo.List(ctx, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, 3, len(list))

	list, err = repo.List(ctx, map[string]interface{}{"is_active": true})
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
}

func TestCategoryRepository_ListActive(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	db.Create(&models.Category{
		Name: "活跃分类", Level: 1, Sort: 100, IsActive: true,
	})

	db.Model(&models.Category{}).Create(map[string]interface{}{
		"name": "禁用分类", "level": 1, "sort": 90, "is_active": false,
	})

	list, err := repo.ListActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
}

func TestCategoryRepository_ListRootCategories(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	root := &models.Category{
		Name:     "根分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(root)

	child := &models.Category{
		Name:     "子分类",
		ParentID: &root.ID,
		Level:    2,
		Sort:     50,
		IsActive: true,
	}
	db.Create(child)

	list, err := repo.ListRootCategories(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
	assert.Equal(t, "根分类", list[0].Name)
}

func TestCategoryRepository_ListByParentID(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	parent := &models.Category{
		Name:     "父分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(parent)

	children := []*models.Category{
		{Name: "子分类1", ParentID: &parent.ID, Level: 2, Sort: 50, IsActive: true},
		{Name: "子分类2", ParentID: &parent.ID, Level: 2, Sort: 40, IsActive: true},
	}
	for _, c := range children {
		db.Create(c)
	}

	list, err := repo.ListByParentID(ctx, parent.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
}

func TestCategoryRepository_ListWithChildren(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	parent := &models.Category{
		Name:     "父分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(parent)

	child := &models.Category{
		Name:     "子分类",
		ParentID: &parent.ID,
		Level:    2,
		Sort:     50,
		IsActive: true,
	}
	db.Create(child)

	list, err := repo.ListWithChildren(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
	assert.Equal(t, 1, len(list[0].Children))
}

func TestCategoryRepository_GetCategoryTree(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	parent := &models.Category{
		Name:     "父分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(parent)

	child := &models.Category{
		Name:     "子分类",
		ParentID: &parent.ID,
		Level:    2,
		Sort:     50,
		IsActive: true,
	}
	db.Create(child)

	tree, err := repo.GetCategoryTree(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(tree))
	assert.Equal(t, 1, len(tree[0].Children))
}

func TestCategoryRepository_HasProducts(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	category := &models.Category{
		Name:     "分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(category)

	has, err := repo.HasProducts(ctx, category.ID)
	require.NoError(t, err)
	assert.False(t, has)

	db.Create(&models.Product{
		CategoryID: category.ID,
		Name:       "商品",
		Images:     []byte(`["test.jpg"]`),
		Price:      100,
	})

	has, err = repo.HasProducts(ctx, category.ID)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestCategoryRepository_HasChildren(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	parent := &models.Category{
		Name:     "父分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(parent)

	has, err := repo.HasChildren(ctx, parent.ID)
	require.NoError(t, err)
	assert.False(t, has)

	child := &models.Category{
		Name:     "子分类",
		ParentID: &parent.ID,
		Level:    2,
		Sort:     50,
		IsActive: true,
	}
	db.Create(child)

	has, err = repo.HasChildren(ctx, parent.ID)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestCategoryRepository_GetPath(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	root := &models.Category{
		Name:     "根分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(root)

	child := &models.Category{
		Name:     "子分类",
		ParentID: &root.ID,
		Level:    2,
		Sort:     50,
		IsActive: true,
	}
	db.Create(child)

	grandchild := &models.Category{
		Name:     "孙分类",
		ParentID: &child.ID,
		Level:    3,
		Sort:     25,
		IsActive: true,
	}
	db.Create(grandchild)

	path, err := repo.GetPath(ctx, grandchild.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(path))
	assert.Equal(t, "根分类", path[0].Name)
	assert.Equal(t, "子分类", path[1].Name)
	assert.Equal(t, "孙分类", path[2].Name)
}
