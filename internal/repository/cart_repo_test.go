// Package repository 购物车仓储单元测试
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

func setupCartTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.CartItem{}, &models.Product{}, &models.ProductSku{})
	require.NoError(t, err)

	return db
}

func TestCartRepository_Create(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	item := &models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  2,
		Selected:  true,
	}

	err := repo.Create(ctx, item)
	require.NoError(t, err)
	assert.NotZero(t, item.ID)
}

func TestCartRepository_GetByID(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	item := &models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  2,
		Selected:  true,
	}
	db.Create(item)

	found, err := repo.GetByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, item.ID, found.ID)
	assert.Equal(t, 2, found.Quantity)
}

func TestCartRepository_GetByUserIDAndProductSku(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	skuID := int64(100)
	item := &models.CartItem{
		UserID:    1,
		ProductID: 1,
		SkuID:     &skuID,
		Quantity:  2,
		Selected:  true,
	}
	db.Create(item)

	// 测试带 SKU ID
	found, err := repo.GetByUserIDAndProductSku(ctx, 1, 1, &skuID)
	require.NoError(t, err)
	assert.Equal(t, item.ID, found.ID)

	// 测试不带 SKU ID
	item2 := &models.CartItem{
		UserID:    1,
		ProductID: 2,
		Quantity:  1,
		Selected:  true,
	}
	db.Create(item2)

	found, err = repo.GetByUserIDAndProductSku(ctx, 1, 2, nil)
	require.NoError(t, err)
	assert.Equal(t, item2.ID, found.ID)
}

func TestCartRepository_Update(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	item := &models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  2,
		Selected:  true,
	}
	db.Create(item)

	item.Quantity = 5
	err := repo.Update(ctx, item)
	require.NoError(t, err)

	var found models.CartItem
	db.First(&found, item.ID)
	assert.Equal(t, 5, found.Quantity)
}

func TestCartRepository_UpdateQuantity(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	item := &models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  2,
		Selected:  true,
	}
	db.Create(item)

	err := repo.UpdateQuantity(ctx, item.ID, 10)
	require.NoError(t, err)

	var found models.CartItem
	db.First(&found, item.ID)
	assert.Equal(t, 10, found.Quantity)
}

func TestCartRepository_UpdateSelected(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	item := &models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  2,
		Selected:  true,
	}
	db.Create(item)

	err := repo.UpdateSelected(ctx, item.ID, false)
	require.NoError(t, err)

	var found models.CartItem
	db.First(&found, item.ID)
	assert.False(t, found.Selected)
}

func TestCartRepository_UpdateAllSelected(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	items := []*models.CartItem{
		{UserID: 1, ProductID: 1, Quantity: 1, Selected: true},
		{UserID: 1, ProductID: 2, Quantity: 1, Selected: true},
	}
	for _, item := range items {
		db.Create(item)
	}

	err := repo.UpdateAllSelected(ctx, 1, false)
	require.NoError(t, err)

	var found []models.CartItem
	db.Where("user_id = ?", 1).Find(&found)
	for _, item := range found {
		assert.False(t, item.Selected)
	}
}

func TestCartRepository_Delete(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	item := &models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  2,
		Selected:  true,
	}
	db.Create(item)

	err := repo.Delete(ctx, item.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.CartItem{}).Where("id = ?", item.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCartRepository_DeleteByIDs(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	items := []*models.CartItem{
		{UserID: 1, ProductID: 1, Quantity: 1, Selected: true},
		{UserID: 1, ProductID: 2, Quantity: 1, Selected: true},
		{UserID: 1, ProductID: 3, Quantity: 1, Selected: true},
	}
	for _, item := range items {
		db.Create(item)
	}

	ids := []int64{items[0].ID, items[1].ID}
	err := repo.DeleteByIDs(ctx, 1, ids)
	require.NoError(t, err)

	var count int64
	db.Model(&models.CartItem{}).Where("user_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestCartRepository_DeleteByUserID(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	items := []*models.CartItem{
		{UserID: 1, ProductID: 1, Quantity: 1, Selected: true},
		{UserID: 1, ProductID: 2, Quantity: 1, Selected: true},
	}
	for _, item := range items {
		db.Create(item)
	}

	err := repo.DeleteByUserID(ctx, 1)
	require.NoError(t, err)

	var count int64
	db.Model(&models.CartItem{}).Where("user_id = ?", 1).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCartRepository_DeleteSelected(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	db.Create(&models.CartItem{
		UserID: 1, ProductID: 1, Quantity: 1, Selected: true,
	})

	db.Model(&models.CartItem{}).Create(map[string]interface{}{
		"user_id":    1,
		"product_id": 2,
		"quantity":   1,
		"selected":   false,
	})

	err := repo.DeleteSelected(ctx, 1)
	require.NoError(t, err)

	var count int64
	db.Model(&models.CartItem{}).Where("user_id = ?", 1).Count(&count)
	assert.Equal(t, int64(1), count) // 只剩下未选中的
}

func TestCartRepository_ListByUserID(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	items := []*models.CartItem{
		{UserID: 1, ProductID: 1, Quantity: 1, Selected: true},
		{UserID: 1, ProductID: 2, Quantity: 1, Selected: true},
		{UserID: 2, ProductID: 1, Quantity: 1, Selected: true},
	}
	for _, item := range items {
		db.Create(item)
	}

	list, err := repo.ListByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
}

func TestCartRepository_ListSelectedByUserID(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	db.Create(&models.CartItem{
		UserID: 1, ProductID: 1, Quantity: 1, Selected: true,
	})

	db.Model(&models.CartItem{}).Create(map[string]interface{}{
		"user_id":    1,
		"product_id": 2,
		"quantity":   1,
		"selected":   false,
	})

	list, err := repo.ListSelectedByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
	assert.True(t, list[0].Selected)
}

func TestCartRepository_CountByUserID(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	items := []*models.CartItem{
		{UserID: 1, ProductID: 1, Quantity: 1, Selected: true},
		{UserID: 1, ProductID: 2, Quantity: 1, Selected: true},
	}
	for _, item := range items {
		db.Create(item)
	}

	count, err := repo.CountByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestCartRepository_SumQuantityByUserID(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	items := []*models.CartItem{
		{UserID: 1, ProductID: 1, Quantity: 3, Selected: true},
		{UserID: 1, ProductID: 2, Quantity: 5, Selected: true},
	}
	for _, item := range items {
		db.Create(item)
	}

	sum, err := repo.SumQuantityByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 8, sum)
}

func TestCartRepository_IncrementQuantity(t *testing.T) {
	db := setupCartTestDB(t)
	repo := NewCartRepository(db)
	ctx := context.Background()

	item := &models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  5,
		Selected:  true,
	}
	db.Create(item)

	err := repo.IncrementQuantity(ctx, item.ID, 3)
	require.NoError(t, err)

	var found models.CartItem
	db.First(&found, item.ID)
	assert.Equal(t, 8, found.Quantity)
}
