// Package repository 评价仓储单元测试
package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupReviewTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Review{}, &models.User{}, &models.Product{}, &models.Order{})
	require.NoError(t, err)

	return db
}

func TestReviewRepository_Create(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{"img1.jpg"})
	review := &models.Review{
		OrderID:   1,
		ProductID: 1,
		UserID:    1,
		Rating:    5,
		Images:    images,
	}

	err := repo.Create(ctx, review)
	require.NoError(t, err)
	assert.NotZero(t, review.ID)
}

func TestReviewRepository_GetByID(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})
	review := &models.Review{
		OrderID:   1,
		ProductID: 1,
		UserID:    1,
		Rating:    5,
		Images:    images,
	}
	db.Create(review)

	found, err := repo.GetByID(ctx, review.ID)
	require.NoError(t, err)
	assert.Equal(t, review.ID, found.ID)
}

func TestReviewRepository_GetByIDWithUser(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	phone := "13800138000"
	user := &models.User{
		Phone: &phone,
	}
	db.Create(user)

	images, _ := json.Marshal([]string{})
	review := &models.Review{
		OrderID:   1,
		ProductID: 1,
		UserID:    user.ID,
		Rating:    5,
		Images:    images,
	}
	db.Create(review)

	found, err := repo.GetByIDWithUser(ctx, review.ID)
	require.NoError(t, err)
	assert.Equal(t, review.ID, found.ID)
	assert.NotNil(t, found.User)
	assert.Equal(t, user.ID, found.User.ID)
}

func TestReviewRepository_GetByOrderAndProduct(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})
	db.Create(&models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images,
	})

	found, err := repo.GetByOrderAndProduct(ctx, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), found.OrderID)
	assert.Equal(t, int64(1), found.ProductID)
}

func TestReviewRepository_Update(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})
	review := &models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 4, Images: images,
	}
	db.Create(review)

	review.Rating = 5
	err := repo.Update(ctx, review)
	require.NoError(t, err)

	var found models.Review
	db.First(&found, review.ID)
	assert.Equal(t, int16(5), found.Rating)
}

func TestReviewRepository_UpdateFields(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})
	review := &models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images,
	}
	db.Create(review)

	reply := "感谢您的评价"
	err := repo.UpdateFields(ctx, review.ID, map[string]interface{}{
		"reply": reply,
	})
	require.NoError(t, err)

	var found models.Review
	db.First(&found, review.ID)
	assert.NotNil(t, found.Reply)
	assert.Equal(t, reply, *found.Reply)
}

func TestReviewRepository_Delete(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})
	review := &models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images,
	}
	db.Create(review)

	err := repo.Delete(ctx, review.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Review{}).Where("id = ?", review.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestReviewRepository_List(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})

	db.Create(&models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Create(&models.Review{
		OrderID: 2, ProductID: 1, UserID: 2, Rating: 4, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Model(&models.Review{}).Create(map[string]interface{}{
		"order_id": 3, "product_id": 2, "user_id": 1, "rating": 3, "images": images, "status": models.ReviewStatusHidden,
	})

	// 获取所有评价
	params := ReviewListParams{Offset: 0, Limit: 10}
	_, total, err := repo.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按商品过滤
	params = ReviewListParams{Offset: 0, Limit: 10, ProductID: 1}
	_, total, err = repo.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按用户过滤
	params = ReviewListParams{Offset: 0, Limit: 10, UserID: 1}
	_, total, err = repo.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按评分过滤
	rating := int16(5)
	params = ReviewListParams{Offset: 0, Limit: 10, Rating: &rating}
	_, total, err = repo.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按状态过滤
	status := int16(models.ReviewStatusVisible)
	params = ReviewListParams{Offset: 0, Limit: 10, Status: &status}
	_, total, err = repo.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestReviewRepository_ListByProductID(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})

	db.Create(&models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Create(&models.Review{
		OrderID: 2, ProductID: 1, UserID: 2, Rating: 4, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Model(&models.Review{}).Create(map[string]interface{}{
		"order_id": 3, "product_id": 1, "user_id": 3, "rating": 3, "images": images, "status": models.ReviewStatusHidden,
	})

	_, total, err := repo.ListByProductID(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total) // 只返回可见的评价
}

func TestReviewRepository_ListByUserID(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})

	db.Create(&models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images,
	})

	db.Create(&models.Review{
		OrderID: 2, ProductID: 2, UserID: 1, Rating: 4, Images: images,
	})

	db.Create(&models.Review{
		OrderID: 3, ProductID: 1, UserID: 2, Rating: 3, Images: images,
	})

	_, total, err := repo.ListByUserID(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestReviewRepository_CountByProductID(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})

	db.Create(&models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Create(&models.Review{
		OrderID: 2, ProductID: 1, UserID: 2, Rating: 4, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Model(&models.Review{}).Create(map[string]interface{}{
		"order_id": 3, "product_id": 1, "user_id": 3, "rating": 3, "images": images, "status": models.ReviewStatusHidden,
	})

	count, err := repo.CountByProductID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count) // 只统计可见的
}

func TestReviewRepository_GetAverageRatingByProductID(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})

	db.Create(&models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Create(&models.Review{
		OrderID: 2, ProductID: 1, UserID: 2, Rating: 3, Images: images, Status: models.ReviewStatusVisible,
	})

	avg, err := repo.GetAverageRatingByProductID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 4.0, avg) // (5+3)/2 = 4
}

func TestReviewRepository_GetRatingDistribution(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})

	db.Create(&models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Create(&models.Review{
		OrderID: 2, ProductID: 1, UserID: 2, Rating: 5, Images: images, Status: models.ReviewStatusVisible,
	})

	db.Create(&models.Review{
		OrderID: 3, ProductID: 1, UserID: 3, Rating: 4, Images: images, Status: models.ReviewStatusVisible,
	})

	distribution, err := repo.GetRatingDistribution(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), distribution[5])
	assert.Equal(t, int64(1), distribution[4])
}

func TestReviewRepository_ExistsByOrderAndProduct(t *testing.T) {
	db := setupReviewTestDB(t)
	repo := NewReviewRepository(db)
	ctx := context.Background()

	images, _ := json.Marshal([]string{})
	db.Create(&models.Review{
		OrderID: 1, ProductID: 1, UserID: 1, Rating: 5, Images: images,
	})

	exists, err := repo.ExistsByOrderAndProduct(ctx, 1, 1)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByOrderAndProduct(ctx, 1, 2)
	require.NoError(t, err)
	assert.False(t, exists)
}
