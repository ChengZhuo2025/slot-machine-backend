// Package repository 文章仓储单元测试
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

func setupArticleTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Article{})
	require.NoError(t, err)

	return db
}

func TestArticleRepository_Create(t *testing.T) {
	db := setupArticleTestDB(t)
	repo := NewArticleRepository(db)
	ctx := context.Background()

	article := &models.Article{
		Title:       "测试文章",
		Content:     "文章内容",
		Category:    models.ArticleCategoryNotice,
		CoverImage:  stringPtr("https://example.com/image.jpg"),
		IsPublished: true,
		ViewCount:   0,
	}

	err := repo.Create(ctx, article)
	require.NoError(t, err)
	assert.NotZero(t, article.ID)
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}

func TestArticleRepository_GetByID(t *testing.T) {
	db := setupArticleTestDB(t)
	repo := NewArticleRepository(db)
	ctx := context.Background()

	article := &models.Article{
		Title:       "测试获取文章",
		Content:     "内容",
		Category:    models.ArticleCategoryHelp,
		IsPublished: true,
	}
	db.Create(article)

	found, err := repo.GetByID(ctx, article.ID)
	require.NoError(t, err)
	assert.Equal(t, article.ID, found.ID)
	assert.Equal(t, "测试获取文章", found.Title)
}

func TestArticleRepository_Update(t *testing.T) {
	db := setupArticleTestDB(t)
	repo := NewArticleRepository(db)
	ctx := context.Background()

	article := &models.Article{
		Title:       "原标题",
		Content:     "原内容",
		Category:    models.ArticleCategoryNotice,
		IsPublished: false,
	}
	db.Create(article)

	article.Title = "更新后的标题"
	article.Content = "更新后的内容"
	article.IsPublished = true
	err := repo.Update(ctx, article)
	require.NoError(t, err)

	var found models.Article
	db.First(&found, article.ID)
	assert.Equal(t, "更新后的标题", found.Title)
	assert.Equal(t, "更新后的内容", found.Content)
}

func TestArticleRepository_Delete(t *testing.T) {
	db := setupArticleTestDB(t)
	repo := NewArticleRepository(db)
	ctx := context.Background()

	article := &models.Article{
		Title:       "待删除文章",
		Content:     "内容",
		Category:    models.ArticleCategoryNotice,
		IsPublished: true,
	}
	db.Create(article)

	err := repo.Delete(ctx, article.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Article{}).Where("id = ?", article.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestArticleRepository_List(t *testing.T) {
	db := setupArticleTestDB(t)
	repo := NewArticleRepository(db)
	ctx := context.Background()

	// 创建测试数据 - 使用 map 以避免 GORM 默认值问题
	db.Model(&models.Article{}).Create(map[string]interface{}{
		"title":        "公告1",
		"content":      "内容1",
		"category":     models.ArticleCategoryNotice,
		"is_published": true,
	})

	db.Model(&models.Article{}).Create(map[string]interface{}{
		"title":        "公告2",
		"content":      "内容2",
		"category":     models.ArticleCategoryNotice,
		"is_published": false,
	})

	db.Model(&models.Article{}).Create(map[string]interface{}{
		"title":        "帮助1",
		"content":      "内容3",
		"category":     models.ArticleCategoryHelp,
		"is_published": true,
	})

	// 测试获取所有已发布文章
	isPublished := true
	list, total, err := repo.List(ctx, 0, 10, &ArticleListFilters{
		IsPublished: &isPublished,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total) // 只有2篇已发布
	assert.Equal(t, 2, len(list))

	// 测试按分类过滤
	list, total, err = repo.List(ctx, 0, 10, &ArticleListFilters{
		Category:    models.ArticleCategoryNotice,
		IsPublished: &isPublished,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestArticleRepository_IncrementViewCount(t *testing.T) {
	db := setupArticleTestDB(t)
	repo := NewArticleRepository(db)
	ctx := context.Background()

	article := &models.Article{
		Title:       "浏览量测试",
		Content:     "内容",
		Category:    models.ArticleCategoryNotice,
		IsPublished: true,
		ViewCount:   10,
	}
	db.Create(article)

	err := repo.IncrementViewCount(ctx, article.ID)
	require.NoError(t, err)

	var found models.Article
	db.First(&found, article.ID)
	assert.Equal(t, 11, found.ViewCount)

	// 再次增加
	err = repo.IncrementViewCount(ctx, article.ID)
	require.NoError(t, err)

	db.First(&found, article.ID)
	assert.Equal(t, 12, found.ViewCount)
}

func TestArticleRepository_PublishAndUnpublish(t *testing.T) {
	db := setupArticleTestDB(t)
	repo := NewArticleRepository(db)
	ctx := context.Background()

	article := &models.Article{
		Title:       "发布状态测试",
		Content:     "内容",
		Category:    models.ArticleCategoryNotice,
		IsPublished: false,
	}
	db.Create(article)

	// 发布
	err := repo.Publish(ctx, article.ID)
	require.NoError(t, err)

	var found models.Article
	db.First(&found, article.ID)
	assert.True(t, found.IsPublished)
	assert.NotNil(t, found.PublishedAt)

	// 取消发布
	err = repo.Unpublish(ctx, article.ID)
	require.NoError(t, err)

	db.First(&found, article.ID)
	assert.False(t, found.IsPublished)
}
