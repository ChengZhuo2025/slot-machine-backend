// Package repository Banner仓储单元测试
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

func setupBannerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Banner{})
	require.NoError(t, err)

	return db
}

func TestBannerRepository_Create(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "测试Banner",
		Image:    "https://example.com/banner.jpg",
		Position: models.BannerPositionHome,
		Sort:     1,
		IsActive: true,
	}

	err := repo.Create(ctx, banner)
	require.NoError(t, err)
	assert.NotZero(t, banner.ID)
}

func TestBannerRepository_GetByID(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "测试Banner",
		Image:    "https://example.com/banner.jpg",
		Position: models.BannerPositionHome,
		Sort:     1,
		IsActive: true,
	}
	db.Create(banner)

	found, err := repo.GetByID(ctx, banner.ID)
	require.NoError(t, err)
	assert.Equal(t, banner.ID, found.ID)
	assert.Equal(t, "测试Banner", found.Title)
}

func TestBannerRepository_Update(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "原标题",
		Image:    "https://example.com/old.jpg",
		Position: models.BannerPositionHome,
		Sort:     1,
		IsActive: true,
	}
	db.Create(banner)

	banner.Title = "新标题"
	banner.Image = "https://example.com/new.jpg"
	banner.Sort = 2
	err := repo.Update(ctx, banner)
	require.NoError(t, err)

	var found models.Banner
	db.First(&found, banner.ID)
	assert.Equal(t, "新标题", found.Title)
	assert.Equal(t, "https://example.com/new.jpg", found.Image)
	assert.Equal(t, 2, found.Sort)
}

func TestBannerRepository_Delete(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "待删除Banner",
		Image:    "https://example.com/banner.jpg",
		Position: models.BannerPositionHome,
		Sort:     1,
		IsActive: true,
	}
	db.Create(banner)

	err := repo.Delete(ctx, banner.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Banner{}).Where("id = ?", banner.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestBannerRepository_ListByPosition(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	// 创建测试数据 - 使用 map 以避免 GORM 默认值问题
	db.Model(&models.Banner{}).Create(map[string]interface{}{
		"title":     "首页Banner1",
		"image":     "img1.jpg",
		"position":  models.BannerPositionHome,
		"sort":      1,
		"is_active": true,
	})

	db.Model(&models.Banner{}).Create(map[string]interface{}{
		"title":     "首页Banner2",
		"image":     "img2.jpg",
		"position":  models.BannerPositionHome,
		"sort":      2,
		"is_active": true,
	})

	db.Model(&models.Banner{}).Create(map[string]interface{}{
		"title":     "商城Banner1",
		"image":     "img3.jpg",
		"position":  models.BannerPositionMall,
		"sort":      1,
		"is_active": true,
	})

	db.Model(&models.Banner{}).Create(map[string]interface{}{
		"title":     "禁用Banner",
		"image":     "img4.jpg",
		"position":  models.BannerPositionHome,
		"sort":      3,
		"is_active": false,
	})

	// 获取首页活动Banner
	list, err := repo.ListByPosition(ctx, models.BannerPositionHome, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
	// 验证按 sort 排序
	assert.Equal(t, 2, list[0].Sort) // Sort DESC，所以 2 在前
	assert.Equal(t, 1, list[1].Sort)

	// 获取商城Banner
	list, err = repo.ListByPosition(ctx, models.BannerPositionMall, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
}

func TestBannerRepository_List(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	// 创建测试数据 - 使用 map 以避免 GORM 默认值问题
	db.Model(&models.Banner{}).Create(map[string]interface{}{
		"title":     "Banner1",
		"image":     "img1.jpg",
		"position":  models.BannerPositionHome,
		"sort":      1,
		"is_active": true,
	})

	db.Model(&models.Banner{}).Create(map[string]interface{}{
		"title":     "Banner2",
		"image":     "img2.jpg",
		"position":  models.BannerPositionHome,
		"sort":      2,
		"is_active": false,
	})

	db.Model(&models.Banner{}).Create(map[string]interface{}{
		"title":     "Banner3",
		"image":     "img3.jpg",
		"position":  models.BannerPositionMall,
		"sort":      1,
		"is_active": true,
	})

	// 获取所有Banner
	list, total, err := repo.List(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 3, len(list))

	// 按位置过滤
	list, total, err = repo.List(ctx, 0, 10, &BannerListFilters{
		Position: models.BannerPositionHome,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按状态过滤
	isActive := true
	list, total, err = repo.List(ctx, 0, 10, &BannerListFilters{
		IsActive: &isActive,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestBannerRepository_UpdateStatus(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "状态测试",
		Image:    "img.jpg",
		Position: models.BannerPositionHome,
		Sort:     1,
		IsActive: true,
	}
	db.Create(banner)

	// 禁用
	err := repo.UpdateStatus(ctx, banner.ID, false)
	require.NoError(t, err)

	var found models.Banner
	db.First(&found, banner.ID)
	assert.False(t, found.IsActive)

	// 启用
	err = repo.UpdateStatus(ctx, banner.ID, true)
	require.NoError(t, err)

	db.First(&found, banner.ID)
	assert.True(t, found.IsActive)
}

func TestBannerRepository_UpdateSort(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "排序测试",
		Image:    "img.jpg",
		Position: models.BannerPositionHome,
		Sort:     1,
		IsActive: true,
	}
	db.Create(banner)

	err := repo.UpdateSort(ctx, banner.ID, 10)
	require.NoError(t, err)

	var found models.Banner
	db.First(&found, banner.ID)
	assert.Equal(t, 10, found.Sort)
}

func TestBannerRepository_CountByPosition(t *testing.T) {
	db := setupBannerTestDB(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()

	banners := []*models.Banner{
		{Title: "Banner1", Image: "img1.jpg", Position: models.BannerPositionHome, Sort: 1, IsActive: true},
		{Title: "Banner2", Image: "img2.jpg", Position: models.BannerPositionHome, Sort: 2, IsActive: true},
		{Title: "Banner3", Image: "img3.jpg", Position: models.BannerPositionMall, Sort: 1, IsActive: true},
	}
	for _, b := range banners {
		db.Create(b)
	}

	counts, err := repo.CountByPosition(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts[models.BannerPositionHome])
	assert.Equal(t, int64(1), counts[models.BannerPositionMall])
}
