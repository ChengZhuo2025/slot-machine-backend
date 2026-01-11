// Package mall 商品搜索服务单元测试
package mall

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupSearchServiceTestDB 创建搜索服务测试数据库
func setupSearchServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 使用共享内存模式避免事务隔离问题
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// 设置连接池参数避免多连接问题
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = db.AutoMigrate(
		&models.Category{},
		&models.Product{},
		&models.ProductSku{},
	)
	require.NoError(t, err)
	return db
}

// newSearchService 创建搜索服务实例
func newSearchService(db *gorm.DB) *SearchService {
	productRepo := repository.NewProductRepository(db)
	return NewSearchService(db, productRepo)
}

// seedSearchTestData 创建搜索测试数据
func seedSearchTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	// 创建分类
	category1 := &models.Category{
		ID:       1,
		Name:     "情趣用品",
		Level:    1,
		Sort:     1,
		IsActive: true,
	}
	category2 := &models.Category{
		ID:       2,
		Name:     "成人玩具",
		Level:    1,
		Sort:     2,
		IsActive: true,
	}
	require.NoError(t, db.Create(category1).Error)
	require.NoError(t, db.Create(category2).Error)

	// 创建商品
	products := []*models.Product{
		{
			CategoryID: 1,
			Name:       "情趣内衣套装",
			Price:      99.9,
			Stock:      100,
			Sales:      50,
			Unit:       "套",
			IsOnSale:   true,
			IsHot:      true,
			IsNew:      false,
			Sort:       1,
		},
		{
			CategoryID: 1,
			Name:       "振动棒",
			Price:      199.0,
			Stock:      50,
			Sales:      30,
			Unit:       "个",
			IsOnSale:   true,
			IsHot:      false,
			IsNew:      true,
			Sort:       2,
		},
		{
			CategoryID: 2,
			Name:       "安全套超薄款",
			Price:      39.9,
			Stock:      200,
			Sales:      100,
			Unit:       "盒",
			IsOnSale:   true,
			IsHot:      true,
			IsNew:      false,
			Sort:       3,
		},
		{
			CategoryID: 2,
			Name:       "润滑剂",
			Price:      59.0,
			Stock:      150,
			Sales:      80,
			Unit:       "瓶",
			IsOnSale:   true,
			IsHot:      false,
			IsNew:      false,
			Sort:       4,
		},
		{
			CategoryID: 1,
			Name:       "成人玩具清洁液",
			Price:      49.9,
			Stock:      0, // 无货
			Sales:      10,
			Unit:       "瓶",
			IsOnSale:   true,
			IsHot:      false,
			IsNew:      false,
			Sort:       5,
		},
		{
			CategoryID: 1,
			Name:       "已下架商品",
			Price:      100.0,
			Stock:      50,
			Sales:      5,
			Unit:       "个",
			IsOnSale:   false, // 已下架
			IsHot:      false,
			IsNew:      false,
			Sort:       6,
		},
	}

	for _, p := range products {
		images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
		p.Images = images
		require.NoError(t, db.Create(p).Error)
	}
}

// ==================== Search 测试 ====================

func TestSearchService_Search_Success(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)
	ctx := context.Background()

	seedSearchTestData(t, db)

	t.Run("搜索情趣内衣", func(t *testing.T) {
		req := &SearchRequest{
			Keyword:  "情趣内衣",
			Page:     1,
			PageSize: 10,
		}

		result, err := service.Search(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.Total, int64(0))
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PageSize)
		assert.Equal(t, "情趣内衣", result.Keyword)
		assert.NotEmpty(t, result.Products)

		// 验证搜索结果包含关键词
		found := false
		for _, p := range result.Products {
			if strings.Contains(p.Name, "情趣内衣") {
				found = true
				break
			}
		}
		assert.True(t, found, "搜索结果应包含关键词")
	})

	t.Run("搜索安全套", func(t *testing.T) {
		req := &SearchRequest{
			Keyword:  "安全套",
			Page:     1,
			PageSize: 10,
		}

		result, err := service.Search(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.Total, int64(0))

		// 验证搜索结果
		found := false
		for _, p := range result.Products {
			if strings.Contains(p.Name, "安全套") {
				found = true
				assert.True(t, p.IsOnSale, "搜索结果应该是在售商品")
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("分类过滤", func(t *testing.T) {
		req := &SearchRequest{
			Keyword:    "套",
			CategoryID: 1, // 情趣用品分类
			Page:       1,
			PageSize:   10,
		}

		result, err := service.Search(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// 验证所有结果都属于指定分类
		for _, p := range result.Products {
			assert.Equal(t, int64(1), p.CategoryID)
		}
	})

	t.Run("价格范围过滤", func(t *testing.T) {
		req := &SearchRequest{
			Keyword:  "套",
			MinPrice: 50.0,
			MaxPrice: 150.0,
			Page:     1,
			PageSize: 10,
		}

		result, err := service.Search(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// 验证所有结果都在价格范围内
		for _, p := range result.Products {
			assert.GreaterOrEqual(t, p.Price, 50.0)
			assert.LessOrEqual(t, p.Price, 150.0)
		}
	})

	t.Run("按价格升序排序", func(t *testing.T) {
		req := &SearchRequest{
			Keyword:  "套",
			SortBy:   "price_asc",
			Page:     1,
			PageSize: 10,
		}

		result, err := service.Search(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// 验证价格升序
		if len(result.Products) > 1 {
			for i := 1; i < len(result.Products); i++ {
				assert.LessOrEqual(t, result.Products[i-1].Price, result.Products[i].Price)
			}
		}
	})

	t.Run("分页测试", func(t *testing.T) {
		req := &SearchRequest{
			Keyword:  "套",
			Page:     1,
			PageSize: 2,
		}

		result, err := service.Search(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.LessOrEqual(t, len(result.Products), 2)
		assert.Greater(t, result.TotalPages, 0)
	})

	t.Run("默认分页参数", func(t *testing.T) {
		req := &SearchRequest{
			Keyword: "套",
		}

		result, err := service.Search(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})
}

func TestSearchService_Search_EmptyKeyword(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)
	ctx := context.Background()

	req := &SearchRequest{
		Keyword:  "",
		Page:     1,
		PageSize: 10,
	}

	result, err := service.Search(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "搜索关键词不能为空")
}

func TestSearchService_Search_WhitespaceKeyword(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)
	ctx := context.Background()

	req := &SearchRequest{
		Keyword:  "   ",
		Page:     1,
		PageSize: 10,
	}

	result, err := service.Search(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "搜索关键词不能为空")
}

func TestSearchService_Search_NoResults(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)
	ctx := context.Background()

	seedSearchTestData(t, db)

	req := &SearchRequest{
		Keyword:  "不存在的商品",
		Page:     1,
		PageSize: 10,
	}

	result, err := service.Search(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.Products)
}

func TestSearchService_Search_OnlyOnSaleProducts(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)
	ctx := context.Background()

	seedSearchTestData(t, db)

	req := &SearchRequest{
		Keyword:  "商品",
		Page:     1,
		PageSize: 100,
	}

	result, err := service.Search(ctx, req)
	require.NoError(t, err)

	// 验证所有搜索结果都是在售商品
	for _, p := range result.Products {
		assert.True(t, p.IsOnSale, "搜索结果应该只包含在售商品")
	}
}

// ==================== GetHotKeywords 测试 ====================

func TestSearchService_GetHotKeywords_Success(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)
	ctx := context.Background()

	t.Run("获取热门关键词", func(t *testing.T) {
		keywords, err := service.GetHotKeywords(ctx, 5)
		require.NoError(t, err)
		assert.NotEmpty(t, keywords)
		assert.LessOrEqual(t, len(keywords), 5)
	})

	t.Run("默认限制10个", func(t *testing.T) {
		keywords, err := service.GetHotKeywords(ctx, 0)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(keywords), 10)
	})

	t.Run("限制超过总数", func(t *testing.T) {
		keywords, err := service.GetHotKeywords(ctx, 100)
		require.NoError(t, err)
		assert.NotEmpty(t, keywords)
	})
}

// ==================== GetSuggestions 测试 ====================

func TestSearchService_GetSuggestions_Success(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)
	ctx := context.Background()

	seedSearchTestData(t, db)

	t.Run("获取搜索建议", func(t *testing.T) {
		suggestions, err := service.GetSuggestions(ctx, "情趣", 10)
		require.NoError(t, err)
		assert.NotNil(t, suggestions)

		// 验证建议包含前缀
		for _, s := range suggestions {
			assert.True(t, strings.HasPrefix(s.Keyword, "情趣"), "建议关键词应该以前缀开头")
		}
	})

	t.Run("空前缀返回空", func(t *testing.T) {
		suggestions, err := service.GetSuggestions(ctx, "", 10)
		require.NoError(t, err)
		assert.Nil(t, suggestions)
	})

	t.Run("空白前缀返回空", func(t *testing.T) {
		suggestions, err := service.GetSuggestions(ctx, "   ", 10)
		require.NoError(t, err)
		assert.Nil(t, suggestions)
	})

	t.Run("限制建议数量", func(t *testing.T) {
		suggestions, err := service.GetSuggestions(ctx, "成", 2)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(suggestions), 2)
	})

	t.Run("默认限制10个", func(t *testing.T) {
		suggestions, err := service.GetSuggestions(ctx, "套", 0)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(suggestions), 10)
	})

	t.Run("无匹配建议", func(t *testing.T) {
		suggestions, err := service.GetSuggestions(ctx, "不存在", 10)
		require.NoError(t, err)
		assert.Empty(t, suggestions)
	})
}

// ==================== toProductInfo 测试 ====================

func TestSearchService_toProductInfo(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)

	subtitle := "测试副标题"
	originalPrice := 100.0
	product := &models.Product{
		ID:            1,
		CategoryID:    1,
		Name:          "测试商品",
		Price:         80.0,
		Subtitle:      &subtitle,
		OriginalPrice: &originalPrice,
		Stock:         50,
		Sales:         10,
		Unit:          "件",
		IsOnSale:      true,
		IsHot:         true,
		IsNew:         false,
	}

	info := service.toProductInfo(product)

	assert.Equal(t, product.ID, info.ID)
	assert.Equal(t, product.CategoryID, info.CategoryID)
	assert.Equal(t, product.Name, info.Name)
	assert.Equal(t, product.Price, info.Price)
	assert.Equal(t, subtitle, info.Subtitle)
	assert.Equal(t, originalPrice, info.OriginalPrice)
	assert.Equal(t, product.Stock, info.Stock)
	assert.Equal(t, product.Sales, info.Sales)
	assert.Equal(t, product.Unit, info.Unit)
	assert.Equal(t, product.IsOnSale, info.IsOnSale)
	assert.Equal(t, product.IsHot, info.IsHot)
	assert.Equal(t, product.IsNew, info.IsNew)
}

func TestSearchService_toProductInfo_NilFields(t *testing.T) {
	db := setupSearchServiceTestDB(t)
	service := newSearchService(db)

	product := &models.Product{
		ID:            1,
		CategoryID:    1,
		Name:          "测试商品",
		Price:         80.0,
		Subtitle:      nil,
		OriginalPrice: nil,
		Stock:         50,
		Sales:         10,
		Unit:          "件",
		IsOnSale:      true,
		IsHot:         false,
		IsNew:         false,
	}

	info := service.toProductInfo(product)

	assert.Equal(t, product.ID, info.ID)
	assert.Equal(t, "", info.Subtitle)
	assert.Equal(t, 0.0, info.OriginalPrice)
}
