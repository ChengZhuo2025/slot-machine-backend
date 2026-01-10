// Package content 轮播图服务单元测试
package content

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupBannerTestDB 创建轮播图测试数据库
func setupBannerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Banner{})
	require.NoError(t, err)

	return db
}

// setupBannerService 创建测试用的 BannerService
func setupBannerService(t *testing.T) (*BannerService, *gorm.DB) {
	db := setupBannerTestDB(t)
	bannerRepo := repository.NewBannerRepository(db)
	service := NewBannerService(bannerRepo)
	return service, db
}

// setupBannerAdminService 创建测试用的 BannerAdminService
func setupBannerAdminService(t *testing.T) (*BannerAdminService, *gorm.DB) {
	db := setupBannerTestDB(t)
	bannerRepo := repository.NewBannerRepository(db)
	service := NewBannerAdminService(bannerRepo)
	return service, db
}

func TestBannerService_ListByPosition(t *testing.T) {
	service, db := setupBannerService(t)
	ctx := context.Background()

	// 创建测试轮播图
	linkType := "page"
	linkValue := "/promo"
	banners := []*models.Banner{
		{Title: "首页Banner1", Image: "img1.png", Position: "home", Sort: 10, IsActive: true, LinkType: &linkType, LinkValue: &linkValue},
		{Title: "首页Banner2", Image: "img2.png", Position: "home", Sort: 5, IsActive: true},
		{Title: "首页Banner3", Image: "img3.png", Position: "home", Sort: 0, IsActive: true}, // 后续设为非活跃
		{Title: "商城Banner1", Image: "mall1.png", Position: "mall", Sort: 10, IsActive: true},
	}
	for _, b := range banners {
		db.Create(b)
	}
	// 手动更新一个为非活跃状态（避免GORM的零值问题）
	db.Model(&models.Banner{}).Where("title = ?", "首页Banner3").Update("is_active", false)

	t.Run("获取首页轮播图", func(t *testing.T) {
		results, err := service.ListByPosition(ctx, "home", 10)
		require.NoError(t, err)
		// 应该只有2个有效的：首页Banner1, Banner2（非活跃的被过滤）
		assert.Len(t, results, 2)
		// 按sort降序排列
		assert.Equal(t, "首页Banner1", results[0].Title)
		assert.Equal(t, "首页Banner2", results[1].Title)
	})

	t.Run("获取商城轮播图", func(t *testing.T) {
		results, err := service.ListByPosition(ctx, "mall", 10)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "商城Banner1", results[0].Title)
	})

	t.Run("限制返回数量", func(t *testing.T) {
		results, err := service.ListByPosition(ctx, "home", 1)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("默认限制", func(t *testing.T) {
		results, err := service.ListByPosition(ctx, "home", 0)
		require.NoError(t, err)
		assert.True(t, len(results) <= 10)
	})

	t.Run("不存在的位置", func(t *testing.T) {
		results, err := service.ListByPosition(ctx, "nonexistent", 10)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("包含链接信息", func(t *testing.T) {
		results, err := service.ListByPosition(ctx, "home", 10)
		require.NoError(t, err)

		// 找到带链接的banner
		var found *BannerResponse
		for _, r := range results {
			if r.Title == "首页Banner1" {
				found = r
				break
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "page", found.LinkType)
		assert.Equal(t, "/promo", found.LinkValue)
	})
}

func TestBannerService_RecordClick(t *testing.T) {
	service, db := setupBannerService(t)
	ctx := context.Background()

	banner := &models.Banner{
		Title:      "测试Banner",
		Image:      "test.png",
		Position:   "home",
		IsActive:   true,
		ClickCount: 0,
	}
	db.Create(banner)

	t.Run("记录点击", func(t *testing.T) {
		err := service.RecordClick(ctx, banner.ID)
		require.NoError(t, err)

		// 验证点击数增加
		var updated models.Banner
		db.First(&updated, banner.ID)
		assert.Equal(t, 1, updated.ClickCount)

		// 再次点击
		err = service.RecordClick(ctx, banner.ID)
		require.NoError(t, err)

		db.First(&updated, banner.ID)
		assert.Equal(t, 2, updated.ClickCount)
	})
}

func TestBannerAdminService_Create(t *testing.T) {
	service, _ := setupBannerAdminService(t)
	ctx := context.Background()

	t.Run("创建基本轮播图", func(t *testing.T) {
		req := &CreateBannerRequest{
			Title:    "新轮播图",
			Image:    "new.png",
			Position: "home",
			Sort:     10,
			IsActive: true,
		}

		banner, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.NotZero(t, banner.ID)
		assert.Equal(t, "新轮播图", banner.Title)
		assert.Equal(t, "new.png", banner.Image)
		assert.Equal(t, "home", banner.Position)
		assert.Equal(t, 10, banner.Sort)
		assert.True(t, banner.IsActive)
	})

	t.Run("创建带链接的轮播图", func(t *testing.T) {
		req := &CreateBannerRequest{
			Title:     "促销活动",
			Image:     "promo.png",
			LinkType:  "url",
			LinkValue: "https://example.com/promo",
			Position:  "home",
			IsActive:  true,
		}

		banner, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, banner.LinkType)
		assert.Equal(t, "url", *banner.LinkType)
		assert.NotNil(t, banner.LinkValue)
		assert.Equal(t, "https://example.com/promo", *banner.LinkValue)
	})

	t.Run("创建定时轮播图", func(t *testing.T) {
		startTime := time.Now().Add(1 * time.Hour)
		endTime := time.Now().Add(24 * time.Hour)
		req := &CreateBannerRequest{
			Title:     "限时活动",
			Image:     "limited.png",
			Position:  "home",
			StartTime: &startTime,
			EndTime:   &endTime,
			IsActive:  true,
		}

		banner, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, banner.StartTime)
		assert.NotNil(t, banner.EndTime)
	})
}

func TestBannerAdminService_GetByID(t *testing.T) {
	service, db := setupBannerAdminService(t)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "测试Banner",
		Image:    "test.png",
		Position: "home",
		IsActive: true,
	}
	db.Create(banner)

	t.Run("获取存在的轮播图", func(t *testing.T) {
		result, err := service.GetByID(ctx, banner.ID)
		require.NoError(t, err)
		assert.Equal(t, banner.ID, result.ID)
		assert.Equal(t, "测试Banner", result.Title)
	})

	t.Run("获取不存在的轮播图", func(t *testing.T) {
		_, err := service.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestBannerAdminService_Update(t *testing.T) {
	service, db := setupBannerAdminService(t)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "原标题",
		Image:    "old.png",
		Position: "home",
		Sort:     10,
		IsActive: true,
	}
	db.Create(banner)

	t.Run("更新标题和图片", func(t *testing.T) {
		newTitle := "新标题"
		newImage := "new.png"
		req := &UpdateBannerRequest{
			Title: &newTitle,
			Image: &newImage,
		}

		updated, err := service.Update(ctx, banner.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "新标题", updated.Title)
		assert.Equal(t, "new.png", updated.Image)
		// 其他字段保持不变
		assert.Equal(t, "home", updated.Position)
		assert.Equal(t, 10, updated.Sort)
	})

	t.Run("更新链接", func(t *testing.T) {
		linkType := "page"
		linkValue := "/product/123"
		req := &UpdateBannerRequest{
			LinkType:  &linkType,
			LinkValue: &linkValue,
		}

		updated, err := service.Update(ctx, banner.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, updated.LinkType)
		assert.Equal(t, "page", *updated.LinkType)
		assert.NotNil(t, updated.LinkValue)
		assert.Equal(t, "/product/123", *updated.LinkValue)
	})

	t.Run("更新状态", func(t *testing.T) {
		isActive := false
		req := &UpdateBannerRequest{
			IsActive: &isActive,
		}

		updated, err := service.Update(ctx, banner.ID, req)
		require.NoError(t, err)
		assert.False(t, updated.IsActive)
	})

	t.Run("更新不存在的轮播图", func(t *testing.T) {
		newTitle := "测试"
		req := &UpdateBannerRequest{Title: &newTitle}
		_, err := service.Update(ctx, 99999, req)
		assert.Error(t, err)
	})
}

func TestBannerAdminService_Delete(t *testing.T) {
	service, db := setupBannerAdminService(t)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "待删除Banner",
		Image:    "delete.png",
		Position: "home",
		IsActive: true,
	}
	db.Create(banner)

	t.Run("删除轮播图", func(t *testing.T) {
		err := service.Delete(ctx, banner.ID)
		require.NoError(t, err)

		// 验证已删除
		_, err = service.GetByID(ctx, banner.ID)
		assert.Error(t, err)
	})
}

func TestBannerAdminService_List(t *testing.T) {
	service, db := setupBannerAdminService(t)
	ctx := context.Background()

	// 创建测试数据
	banners := []*models.Banner{
		{Title: "首页Banner1", Image: "home1.png", Position: "home", Sort: 10, IsActive: true},
		{Title: "首页Banner2", Image: "home2.png", Position: "home", Sort: 5, IsActive: true},
		{Title: "商城Banner", Image: "mall.png", Position: "mall", Sort: 10, IsActive: true},
		{Title: "促销活动", Image: "promo.png", Position: "home", Sort: 0, IsActive: true},
	}
	for _, b := range banners {
		db.Create(b)
	}
	// 手动更新一个为非活跃状态（避免GORM的零值问题）
	db.Model(&models.Banner{}).Where("title = ?", "首页Banner2").Update("is_active", false)

	t.Run("获取全部列表", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "", nil, "")
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 4)
	})

	t.Run("按位置筛选", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "home", nil, "")
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		for _, r := range results {
			assert.Equal(t, "home", r.Position)
		}
	})

	t.Run("按状态筛选", func(t *testing.T) {
		isActive := true
		results, total, err := service.List(ctx, 1, 10, "", &isActive, "")
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		for _, r := range results {
			assert.True(t, r.IsActive)
		}
	})

	t.Run("按关键词搜索", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "", nil, "促销")
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "促销活动", results[0].Title)
	})

	t.Run("组合筛选", func(t *testing.T) {
		isActive := true
		_, total, err := service.List(ctx, 1, 10, "home", &isActive, "")
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
	})

	t.Run("分页", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 2, "", nil, "")
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 2)

		results2, _, err := service.List(ctx, 2, 2, "", nil, "")
		require.NoError(t, err)
		assert.Len(t, results2, 2)
	})
}

func TestBannerAdminService_UpdateStatus(t *testing.T) {
	service, db := setupBannerAdminService(t)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "测试Banner",
		Image:    "test.png",
		Position: "home",
		IsActive: true,
	}
	db.Create(banner)

	t.Run("禁用轮播图", func(t *testing.T) {
		err := service.UpdateStatus(ctx, banner.ID, false)
		require.NoError(t, err)

		result, _ := service.GetByID(ctx, banner.ID)
		assert.False(t, result.IsActive)
	})

	t.Run("启用轮播图", func(t *testing.T) {
		err := service.UpdateStatus(ctx, banner.ID, true)
		require.NoError(t, err)

		result, _ := service.GetByID(ctx, banner.ID)
		assert.True(t, result.IsActive)
	})
}

func TestBannerAdminService_UpdateSort(t *testing.T) {
	service, db := setupBannerAdminService(t)
	ctx := context.Background()

	banner := &models.Banner{
		Title:    "测试Banner",
		Image:    "test.png",
		Position: "home",
		Sort:     10,
		IsActive: true,
	}
	db.Create(banner)

	t.Run("更新排序", func(t *testing.T) {
		err := service.UpdateSort(ctx, banner.ID, 100)
		require.NoError(t, err)

		result, _ := service.GetByID(ctx, banner.ID)
		assert.Equal(t, 100, result.Sort)
	})
}

func TestBannerAdminService_GetStatistics(t *testing.T) {
	service, db := setupBannerAdminService(t)
	ctx := context.Background()

	// 创建测试数据
	banners := []*models.Banner{
		{Title: "首页Banner1", Image: "home1.png", Position: "home", IsActive: true},
		{Title: "首页Banner2", Image: "home2.png", Position: "home", IsActive: true},
		{Title: "首页Banner3", Image: "home3.png", Position: "home", IsActive: true},
		{Title: "商城Banner", Image: "mall.png", Position: "mall", IsActive: true},
		{Title: "我的Banner", Image: "my.png", Position: "my", IsActive: true},
	}
	for _, b := range banners {
		db.Create(b)
	}
	// 手动更新一个为非活跃状态
	db.Model(&models.Banner{}).Where("title = ?", "首页Banner3").Update("is_active", false)

	stats, err := service.GetStatistics(ctx)
	require.NoError(t, err)

	t.Run("活跃数量", func(t *testing.T) {
		assert.Equal(t, int64(4), stats.ActiveCount)
	})

	t.Run("按位置统计", func(t *testing.T) {
		assert.Equal(t, int64(3), stats.PositionCounts["home"])
		assert.Equal(t, int64(1), stats.PositionCounts["mall"])
		assert.Equal(t, int64(1), stats.PositionCounts["my"])
	})
}
