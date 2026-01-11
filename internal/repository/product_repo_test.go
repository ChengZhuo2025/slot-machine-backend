// Package repository 商品仓储单元测试
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

func setupProductTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Category{}, &models.Product{}, &models.ProductSku{})
	require.NoError(t, err)

	return db
}

func testImages() []byte {
	return []byte(`["image1.jpg","image2.jpg"]`)
}

func testAttributes() []byte {
	return []byte(`{"color":"red","size":"M"}`)
}

func TestProductRepository_Create(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "测试商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}

	err := repo.Create(ctx, product)
	require.NoError(t, err)
	assert.NotZero(t, product.ID)
}

func TestProductRepository_GetByID(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "测试商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}
	db.Create(product)

	found, err := repo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, product.ID, found.ID)
	assert.Equal(t, "测试商品", found.Name)
}

func TestProductRepository_GetByIDWithCategory(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	category := &models.Category{
		Name:     "分类",
		Level:    1,
		Sort:     100,
		IsActive: true,
	}
	db.Create(category)

	product := &models.Product{
		CategoryID: category.ID,
		Name:       "测试商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}
	db.Create(product)

	found, err := repo.GetByIDWithCategory(ctx, product.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.Category)
	assert.Equal(t, category.ID, found.Category.ID)
}

func TestProductRepository_GetByIDWithSkus(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "测试商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}
	db.Create(product)

	db.Create(&models.ProductSku{
		ProductID:  product.ID,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	})

	db.Model(&models.ProductSku{}).Create(map[string]interface{}{
		"product_id": product.ID,
		"sku_code":   "SKU002",
		"attributes": testAttributes(),
		"price":      89.99,
		"stock":      30,
		"is_active":  false,
	})

	found, err := repo.GetByIDWithSkus(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(found.Skus)) // 只加载活跃 SKU
}

func TestProductRepository_Update(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "原商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}
	db.Create(product)

	product.Name = "新商品"
	product.Price = 199.99
	err := repo.Update(ctx, product)
	require.NoError(t, err)

	var found models.Product
	db.First(&found, product.ID)
	assert.Equal(t, "新商品", found.Name)
	assert.Equal(t, 199.99, found.Price)
}

func TestProductRepository_UpdateFields(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "测试商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}
	db.Create(product)

	err := repo.UpdateFields(ctx, product.ID, map[string]interface{}{
		"price": 149.99,
		"stock": 200,
	})
	require.NoError(t, err)

	var found models.Product
	db.First(&found, product.ID)
	assert.Equal(t, 149.99, found.Price)
	assert.Equal(t, 200, found.Stock)
}

func TestProductRepository_Delete(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "待删除商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}
	db.Create(product)

	err := repo.Delete(ctx, product.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Product{}).Where("id = ?", product.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestProductRepository_List(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	products := []*models.Product{
		{CategoryID: 1, Name: "商品1", Images: testImages(), Price: 100, Stock: 50, IsOnSale: true},
		{CategoryID: 1, Name: "商品2", Images: testImages(), Price: 200, Stock: 30, IsOnSale: true},
	}
	for _, p := range products {
		db.Create(p)
	}

	db.Model(&models.Product{}).Create(map[string]interface{}{
		"category_id": 1,
		"name":        "下架商品",
		"images":      testImages(),
		"price":       150,
		"stock":       20,
		"is_on_sale":  false,
	})

	list, total, err := repo.List(ctx, ProductListParams{Offset: 0, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 3, len(list))

	isOnSale := true
	list, total, err = repo.List(ctx, ProductListParams{Offset: 0, Limit: 10, IsOnSale: &isOnSale})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestProductRepository_ListOnSale(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	db.Create(&models.Product{
		CategoryID: 1, Name: "上架商品", Images: testImages(), Price: 100, Stock: 50, IsOnSale: true,
	})

	db.Model(&models.Product{}).Create(map[string]interface{}{
		"category_id": 1,
		"name":        "下架商品",
		"images":      testImages(),
		"price":       150,
		"stock":       20,
		"is_on_sale":  false,
	})

	list, total, err := repo.ListOnSale(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}

func TestProductRepository_ListByCategory(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	products := []*models.Product{
		{CategoryID: 1, Name: "分类1商品", Images: testImages(), Price: 100, Stock: 50, IsOnSale: true},
		{CategoryID: 2, Name: "分类2商品", Images: testImages(), Price: 200, Stock: 30, IsOnSale: true},
	}
	for _, p := range products {
		db.Create(p)
	}

	list, total, err := repo.ListByCategory(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}

func TestProductRepository_ListHot(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	db.Create(&models.Product{
		CategoryID: 1, Name: "热门商品", Images: testImages(), Price: 100, Stock: 50,
		IsOnSale: true, IsHot: true, Sales: 1000,
	})

	db.Model(&models.Product{}).Create(map[string]interface{}{
		"category_id": 1,
		"name":        "普通商品",
		"images":      testImages(),
		"price":       150,
		"stock":       20,
		"is_on_sale":  true,
		"is_hot":      false,
	})

	list, err := repo.ListHot(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
	assert.True(t, list[0].IsHot)
}

func TestProductRepository_ListNew(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	db.Create(&models.Product{
		CategoryID: 1, Name: "新品", Images: testImages(), Price: 100, Stock: 50,
		IsOnSale: true, IsNew: true,
	})

	db.Model(&models.Product{}).Create(map[string]interface{}{
		"category_id": 1,
		"name":        "旧品",
		"images":      testImages(),
		"price":       150,
		"stock":       20,
		"is_on_sale":  true,
		"is_new":      false,
	})

	list, err := repo.ListNew(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
	assert.True(t, list[0].IsNew)
}

func TestProductRepository_IncreaseSales(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "测试商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
		Sales:      10,
	}
	db.Create(product)

	err := repo.IncreaseSales(ctx, product.ID, 5)
	require.NoError(t, err)

	var found models.Product
	db.First(&found, product.ID)
	assert.Equal(t, 15, found.Sales)
}

func TestProductRepository_DecreaseStock(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "测试商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}
	db.Create(product)

	err := repo.DecreaseStock(ctx, product.ID, 30)
	require.NoError(t, err)

	var found models.Product
	db.First(&found, product.ID)
	assert.Equal(t, 70, found.Stock)

	// 测试库存不足
	err = repo.DecreaseStock(ctx, product.ID, 100)
	assert.Error(t, err)
}

func TestProductRepository_IncreaseStock(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &models.Product{
		CategoryID: 1,
		Name:       "测试商品",
		Images:     testImages(),
		Price:      99.99,
		Stock:      100,
		IsOnSale:   true,
	}
	db.Create(product)

	err := repo.IncreaseStock(ctx, product.ID, 50)
	require.NoError(t, err)

	var found models.Product
	db.First(&found, product.ID)
	assert.Equal(t, 150, found.Stock)
}

// ProductSku 测试

func TestProductSkuRepository_Create(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	sku := &models.ProductSku{
		ProductID:  1,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	}

	err := repo.Create(ctx, sku)
	require.NoError(t, err)
	assert.NotZero(t, sku.ID)
}

func TestProductSkuRepository_CreateBatch(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	skus := []*models.ProductSku{
		{ProductID: 1, SkuCode: "SKU001", Attributes: testAttributes(), Price: 99.99, Stock: 50, IsActive: true},
		{ProductID: 1, SkuCode: "SKU002", Attributes: testAttributes(), Price: 89.99, Stock: 30, IsActive: true},
	}

	err := repo.CreateBatch(ctx, skus)
	require.NoError(t, err)

	var count int64
	db.Model(&models.ProductSku{}).Where("product_id = ?", 1).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestProductSkuRepository_GetByID(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	sku := &models.ProductSku{
		ProductID:  1,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	}
	db.Create(sku)

	found, err := repo.GetByID(ctx, sku.ID)
	require.NoError(t, err)
	assert.Equal(t, sku.ID, found.ID)
	assert.Equal(t, "SKU001", found.SkuCode)
}

func TestProductSkuRepository_GetBySkuCode(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	sku := &models.ProductSku{
		ProductID:  1,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	}
	db.Create(sku)

	found, err := repo.GetBySkuCode(ctx, "SKU001")
	require.NoError(t, err)
	assert.Equal(t, sku.ID, found.ID)
}

func TestProductSkuRepository_ListByProductID(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	db.Create(&models.ProductSku{
		ProductID: 1, SkuCode: "SKU001", Attributes: testAttributes(), Price: 99.99, Stock: 50, IsActive: true,
	})

	db.Model(&models.ProductSku{}).Create(map[string]interface{}{
		"product_id": 1,
		"sku_code":   "SKU002",
		"attributes": testAttributes(),
		"price":      89.99,
		"stock":      30,
		"is_active":  false,
	})

	list, err := repo.ListByProductID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list)) // 只返回活跃 SKU
}

func TestProductSkuRepository_Update(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	sku := &models.ProductSku{
		ProductID:  1,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	}
	db.Create(sku)

	sku.Price = 119.99
	err := repo.Update(ctx, sku)
	require.NoError(t, err)

	var found models.ProductSku
	db.First(&found, sku.ID)
	assert.Equal(t, 119.99, found.Price)
}

func TestProductSkuRepository_UpdateFields(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	sku := &models.ProductSku{
		ProductID:  1,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	}
	db.Create(sku)

	err := repo.UpdateFields(ctx, sku.ID, map[string]interface{}{
		"price": 129.99,
		"stock": 100,
	})
	require.NoError(t, err)

	var found models.ProductSku
	db.First(&found, sku.ID)
	assert.Equal(t, 129.99, found.Price)
	assert.Equal(t, 100, found.Stock)
}

func TestProductSkuRepository_Delete(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	sku := &models.ProductSku{
		ProductID:  1,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	}
	db.Create(sku)

	err := repo.Delete(ctx, sku.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.ProductSku{}).Where("id = ?", sku.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestProductSkuRepository_DeleteByProductID(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	skus := []*models.ProductSku{
		{ProductID: 1, SkuCode: "SKU001", Attributes: testAttributes(), Price: 99.99, Stock: 50, IsActive: true},
		{ProductID: 1, SkuCode: "SKU002", Attributes: testAttributes(), Price: 89.99, Stock: 30, IsActive: true},
	}
	for _, sku := range skus {
		db.Create(sku)
	}

	err := repo.DeleteByProductID(ctx, 1)
	require.NoError(t, err)

	var count int64
	db.Model(&models.ProductSku{}).Where("product_id = ?", 1).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestProductSkuRepository_DecreaseStock(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	sku := &models.ProductSku{
		ProductID:  1,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	}
	db.Create(sku)

	err := repo.DecreaseStock(ctx, sku.ID, 20)
	require.NoError(t, err)

	var found models.ProductSku
	db.First(&found, sku.ID)
	assert.Equal(t, 30, found.Stock)

	// 测试库存不足
	err = repo.DecreaseStock(ctx, sku.ID, 100)
	assert.Error(t, err)
}

func TestProductSkuRepository_IncreaseStock(t *testing.T) {
	db := setupProductTestDB(t)
	repo := NewProductSkuRepository(db)
	ctx := context.Background()

	sku := &models.ProductSku{
		ProductID:  1,
		SkuCode:    "SKU001",
		Attributes: testAttributes(),
		Price:      99.99,
		Stock:      50,
		IsActive:   true,
	}
	db.Create(sku)

	err := repo.IncreaseStock(ctx, sku.ID, 30)
	require.NoError(t, err)

	var found models.ProductSku
	db.First(&found, sku.ID)
	assert.Equal(t, 80, found.Stock)
}
