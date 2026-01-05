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

func setupReviewServiceTestDB(t *testing.T) *gorm.DB {
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
		&models.User{},
		&models.MemberLevel{},
		&models.Category{},
		&models.Product{},
		&models.Order{},
		&models.OrderItem{},
		&models.Review{},
	)
	require.NoError(t, err)

	// 创建默认会员等级
	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	return db
}

func newReviewService(db *gorm.DB) *ReviewService {
	reviewRepo := repository.NewReviewRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	return NewReviewService(db, reviewRepo, orderRepo)
}

func seedReviewTestData(t *testing.T, db *gorm.DB) (*models.User, *models.Product, *models.Order) {
	t.Helper()

	// 创建用户
	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)

	// 创建分类
	category := &models.Category{Name: "测试分类", Level: 1, Sort: 1, IsActive: true}
	require.NoError(t, db.Create(category).Error)

	// 创建商品
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	product := &models.Product{
		CategoryID: category.ID,
		Name:       "测试商品",
		Images:     images,
		Price:      80.0,
		Stock:      50,
		Unit:       "件",
		IsOnSale:   true,
	}
	require.NoError(t, db.Create(product).Error)

	// 创建已完成订单
	order := &models.Order{
		OrderNo:        "M20240101001",
		UserID:         user.ID,
		Type:           models.OrderTypeMall,
		OriginalAmount: 80.0,
		ActualAmount:   80.0,
		Status:         models.OrderStatusCompleted, // 已完成
	}
	require.NoError(t, db.Create(order).Error)

	// 创建订单项
	orderItem := &models.OrderItem{
		OrderID:     order.ID,
		ProductID:   &product.ID,
		ProductName: product.Name,
		Price:       product.Price,
		Quantity:    1,
		Subtotal:    product.Price,
	}
	require.NoError(t, db.Create(orderItem).Error)

	return user, product, order
}

// ==================== 创建评价测试 ====================

func TestReviewService_CreateReview_Success(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	review, err := svc.CreateReview(ctx, user.ID, &CreateReviewRequest{
		OrderID:     order.ID,
		ProductID:   product.ID,
		Rating:      5,
		Content:     "商品质量很好！",
		Images:      []string{"https://example.com/review1.jpg"},
		IsAnonymous: false,
	})
	require.NoError(t, err)
	assert.Equal(t, order.ID, review.OrderID)
	assert.Equal(t, product.ID, review.ProductID)
	assert.Equal(t, 5, review.Rating)
	assert.Equal(t, "商品质量很好！", review.Content)
	assert.Len(t, review.Images, 1)
	assert.False(t, review.IsAnonymous)
}

func TestReviewService_CreateReview_Anonymous(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	review, err := svc.CreateReview(ctx, user.ID, &CreateReviewRequest{
		OrderID:     order.ID,
		ProductID:   product.ID,
		Rating:      4,
		Content:     "还不错",
		IsAnonymous: true,
	})
	require.NoError(t, err)
	assert.True(t, review.IsAnonymous)
}

func TestReviewService_CreateReview_WithoutContent(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	review, err := svc.CreateReview(ctx, user.ID, &CreateReviewRequest{
		OrderID:   order.ID,
		ProductID: product.ID,
		Rating:    5,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, review.Rating)
	assert.Empty(t, review.Content)
}

func TestReviewService_CreateReview_OrderNotFound(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, _ := seedReviewTestData(t, db)

	_, err := svc.CreateReview(ctx, user.ID, &CreateReviewRequest{
		OrderID:   99999,
		ProductID: product.ID,
		Rating:    5,
	})
	assert.Error(t, err)
}

func TestReviewService_CreateReview_OrderNotOwned(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	_, product, order := seedReviewTestData(t, db)

	// 创建另一个用户
	phone2 := "13900139000"
	anotherUser := &models.User{
		Phone:         &phone2,
		Nickname:      "另一个用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(anotherUser).Error)

	// 尝试评价不属于自己的订单
	_, err := svc.CreateReview(ctx, anotherUser.ID, &CreateReviewRequest{
		OrderID:   order.ID,
		ProductID: product.ID,
		Rating:    5,
	})
	assert.Error(t, err)
}

func TestReviewService_CreateReview_OrderNotCompleted(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	// 将订单状态改为待支付
	db.Model(&models.Order{}).Where("id = ?", order.ID).Update("status", models.OrderStatusPending)

	_, err := svc.CreateReview(ctx, user.ID, &CreateReviewRequest{
		OrderID:   order.ID,
		ProductID: product.ID,
		Rating:    5,
	})
	assert.Error(t, err)
}

func TestReviewService_CreateReview_AlreadyReviewed(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	// 第一次评价
	_, err := svc.CreateReview(ctx, user.ID, &CreateReviewRequest{
		OrderID:   order.ID,
		ProductID: product.ID,
		Rating:    5,
	})
	require.NoError(t, err)

	// 尝试重复评价
	_, err = svc.CreateReview(ctx, user.ID, &CreateReviewRequest{
		OrderID:   order.ID,
		ProductID: product.ID,
		Rating:    4,
	})
	assert.Error(t, err)
}

// ==================== 获取商品评价列表测试 ====================

func TestReviewService_GetProductReviews(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	// 创建多个评价
	for i := 0; i < 5; i++ {
		content := "评价内容" + string(rune('A'+i))
		review := &models.Review{
			OrderID:   order.ID + int64(i), // 不同订单
			ProductID: product.ID,
			UserID:    user.ID,
			Rating:    int16(3 + i%3),
			Content:   &content,
			Status:    int16(models.ReviewStatusVisible),
		}
		// 需要先创建对应的订单
		newOrder := &models.Order{
			OrderNo:        "M2024010100" + string(rune('1'+i)),
			UserID:         user.ID,
			Type:           models.OrderTypeMall,
			OriginalAmount: 100,
			ActualAmount:   100,
			Status:         models.OrderStatusCompleted,
		}
		db.Create(newOrder)
		review.OrderID = newOrder.ID
		db.Create(review)
	}

	resp, err := svc.GetProductReviews(ctx, product.ID, 1, 3)
	require.NoError(t, err)
	assert.Len(t, resp.List, 3)
	assert.Equal(t, int64(5), resp.Total)
	assert.Equal(t, 2, resp.TotalPages)
}

func TestReviewService_GetProductReviews_Pagination(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, _ := seedReviewTestData(t, db)

	// 创建 10 个评价
	for i := 0; i < 10; i++ {
		newOrder := &models.Order{
			OrderNo:        "M2024010100" + string(rune('0'+i)),
			UserID:         user.ID,
			Type:           models.OrderTypeMall,
			OriginalAmount: 100,
			ActualAmount:   100,
			Status:         models.OrderStatusCompleted,
		}
		db.Create(newOrder)

		content := "评价" + string(rune('0'+i))
		review := &models.Review{
			OrderID:   newOrder.ID,
			ProductID: product.ID,
			UserID:    user.ID,
			Rating:    int16(3 + i%3),
			Content:   &content,
			Status:    int16(models.ReviewStatusVisible),
		}
		db.Create(review)
	}

	// 第一页
	resp1, err := svc.GetProductReviews(ctx, product.ID, 1, 4)
	require.NoError(t, err)
	assert.Len(t, resp1.List, 4)
	assert.Equal(t, 1, resp1.Page)

	// 第二页
	resp2, err := svc.GetProductReviews(ctx, product.ID, 2, 4)
	require.NoError(t, err)
	assert.Len(t, resp2.List, 4)
	assert.Equal(t, 2, resp2.Page)

	// 第三页
	resp3, err := svc.GetProductReviews(ctx, product.ID, 3, 4)
	require.NoError(t, err)
	assert.Len(t, resp3.List, 2)
	assert.Equal(t, 3, resp3.Page)
}

// ==================== 获取用户评价列表测试 ====================

func TestReviewService_GetUserReviews(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, _ := seedReviewTestData(t, db)

	// 创建用户的多个评价
	for i := 0; i < 3; i++ {
		newOrder := &models.Order{
			OrderNo:        "M2024010100" + string(rune('0'+i)),
			UserID:         user.ID,
			Type:           models.OrderTypeMall,
			OriginalAmount: 100,
			ActualAmount:   100,
			Status:         models.OrderStatusCompleted,
		}
		db.Create(newOrder)

		content := "用户评价" + string(rune('0'+i))
		review := &models.Review{
			OrderID:   newOrder.ID,
			ProductID: product.ID,
			UserID:    user.ID,
			Rating:    int16(4 + i%2),
			Content:   &content,
			Status:    int16(models.ReviewStatusVisible),
		}
		db.Create(review)
	}

	resp, err := svc.GetUserReviews(ctx, user.ID, 1, 10)
	require.NoError(t, err)
	assert.Len(t, resp.List, 3)
	assert.Equal(t, int64(3), resp.Total)
}

// ==================== 评价统计测试 ====================

func TestReviewService_GetProductReviewStats(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, _ := seedReviewTestData(t, db)

	// 创建不同评分的评价
	ratings := []int16{5, 5, 4, 4, 4, 3, 2}
	for i, rating := range ratings {
		newOrder := &models.Order{
			OrderNo:        "M2024010100" + string(rune('0'+i)),
			UserID:         user.ID,
			Type:           models.OrderTypeMall,
			OriginalAmount: 100,
			ActualAmount:   100,
			Status:         models.OrderStatusCompleted,
		}
		db.Create(newOrder)

		review := &models.Review{
			OrderID:   newOrder.ID,
			ProductID: product.ID,
			UserID:    user.ID,
			Rating:    rating,
			Status:    int16(models.ReviewStatusVisible),
		}
		db.Create(review)
	}

	stats, err := svc.GetProductReviewStats(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(7), stats.TotalCount)
	// 平均评分 = (5+5+4+4+4+3+2)/7 = 27/7 ≈ 3.86
	assert.InDelta(t, 3.86, stats.AverageRating, 0.1)
	assert.Equal(t, int64(2), stats.Distribution[5])
	assert.Equal(t, int64(3), stats.Distribution[4])
	assert.Equal(t, int64(1), stats.Distribution[3])
	assert.Equal(t, int64(1), stats.Distribution[2])
}

// ==================== 获取评价详情测试 ====================

func TestReviewService_GetReviewByID(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	content := "测试评价内容"
	review := &models.Review{
		OrderID:   order.ID,
		ProductID: product.ID,
		UserID:    user.ID,
		Rating:    5,
		Content:   &content,
		Status:    int16(models.ReviewStatusVisible),
	}
	require.NoError(t, db.Create(review).Error)

	info, err := svc.GetReviewByID(ctx, review.ID)
	require.NoError(t, err)
	assert.Equal(t, review.ID, info.ID)
	assert.Equal(t, 5, info.Rating)
	assert.Equal(t, "测试评价内容", info.Content)
}

func TestReviewService_GetReviewByID_NotFound(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	_, err := svc.GetReviewByID(ctx, 99999)
	assert.Error(t, err)
}

// ==================== 删除评价测试 ====================

func TestReviewService_DeleteReview_Success(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	review := &models.Review{
		OrderID:   order.ID,
		ProductID: product.ID,
		UserID:    user.ID,
		Rating:    5,
		Status:    int16(models.ReviewStatusVisible),
	}
	require.NoError(t, db.Create(review).Error)

	err := svc.DeleteReview(ctx, user.ID, review.ID)
	require.NoError(t, err)

	// 验证已删除
	var count int64
	db.Model(&models.Review{}).Where("id = ?", review.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestReviewService_DeleteReview_NotOwned(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	// 创建另一个用户
	phone2 := "13900139000"
	anotherUser := &models.User{
		Phone:         &phone2,
		Nickname:      "另一个用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(anotherUser).Error)

	review := &models.Review{
		OrderID:   order.ID,
		ProductID: product.ID,
		UserID:    user.ID,
		Rating:    5,
		Status:    int16(models.ReviewStatusVisible),
	}
	require.NoError(t, db.Create(review).Error)

	// 尝试删除不属于自己的评价
	err := svc.DeleteReview(ctx, anotherUser.ID, review.ID)
	assert.Error(t, err)
}

// ==================== 商家回复测试 ====================

func TestReviewService_ReplyReview(t *testing.T) {
	db := setupReviewServiceTestDB(t)
	svc := newReviewService(db)
	ctx := context.Background()

	user, product, order := seedReviewTestData(t, db)

	review := &models.Review{
		OrderID:   order.ID,
		ProductID: product.ID,
		UserID:    user.ID,
		Rating:    5,
		Status:    int16(models.ReviewStatusVisible),
	}
	require.NoError(t, db.Create(review).Error)

	err := svc.ReplyReview(ctx, review.ID, "感谢您的好评！")
	require.NoError(t, err)

	// 验证回复
	var updated models.Review
	require.NoError(t, db.First(&updated, review.ID).Error)
	assert.Equal(t, "感谢您的好评！", *updated.Reply)
	assert.NotNil(t, updated.RepliedAt)
}
