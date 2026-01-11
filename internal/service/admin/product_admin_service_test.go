package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupProductAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.Category{},
		&models.Product{},
		&models.ProductSku{},
	))
	return db
}

func TestProductAdminService_CategoryBranches(t *testing.T) {
	db := setupProductAdminTestDB(t)
	svc := NewProductAdminService(
		db,
		repository.NewCategoryRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSkuRepository(db),
	)
	ctx := context.Background()

	t.Run("CreateCategory 父分类不存在", func(t *testing.T) {
		pid := int64(99999)
		_, err := svc.CreateCategory(ctx, &CreateCategoryRequest{ParentID: &pid, Name: "child"})
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrResourceNotFound.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "父分类不存在")
	})

	parent, err := svc.CreateCategory(ctx, &CreateCategoryRequest{Name: "parent"})
	require.NoError(t, err)
	child, err := svc.CreateCategory(ctx, &CreateCategoryRequest{ParentID: &parent.ID, Name: "child"})
	require.NoError(t, err)
	require.NotNil(t, child)

	t.Run("DeleteCategory 有子分类不允许删除", func(t *testing.T) {
		err := svc.DeleteCategory(ctx, parent.ID)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrOperationFailed.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "子分类")
	})

	t.Run("DeleteCategory 有商品不允许删除", func(t *testing.T) {
		leaf, err := svc.CreateCategory(ctx, &CreateCategoryRequest{Name: "leaf"})
		require.NoError(t, err)

		product := &models.Product{
			CategoryID: leaf.ID,
			Name:       "P1",
			Images:     []byte(`["a"]`),
			Price:      10,
			Stock:      1,
			Unit:       "件",
			IsOnSale:   true,
		}
		require.NoError(t, db.Create(product).Error)

		err = svc.DeleteCategory(ctx, leaf.ID)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrOperationFailed.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "商品")
	})
}

func TestProductAdminService_CreateProduct(t *testing.T) {
	db := setupProductAdminTestDB(t)
	svc := NewProductAdminService(
		db,
		repository.NewCategoryRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSkuRepository(db),
	)
	ctx := context.Background()

	t.Run("分类不存在返回资源不存在", func(t *testing.T) {
		_, err := svc.CreateProduct(ctx, &CreateProductRequest{
			CategoryID: 99999,
			Name:       "P",
			Images:     []string{"a"},
			Price:      10,
		})
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrResourceNotFound.Code, appErr.Code)
	})

	cat, err := svc.CreateCategory(ctx, &CreateCategoryRequest{Name: "cat"})
	require.NoError(t, err)

	info, err := svc.CreateProduct(ctx, &CreateProductRequest{
		CategoryID: cat.ID,
		Name:       "商品1",
		Images:     []string{"img1"},
		Price:      10,
		Stock:      5,
		Unit:       "件",
		IsOnSale:   true,
	})
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, cat.ID, info.CategoryID)
	assert.Equal(t, "商品1", info.Name)

	var product models.Product
	require.NoError(t, db.First(&product, info.ID).Error)
	assert.Equal(t, "商品1", product.Name)
}

func TestProductAdminService_UpdateAndListCategory(t *testing.T) {
	db := setupProductAdminTestDB(t)
	svc := NewProductAdminService(
		db,
		repository.NewCategoryRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSkuRepository(db),
	)
	ctx := context.Background()

	cat, _ := svc.CreateCategory(ctx, &CreateCategoryRequest{Name: "原分类"})

	t.Run("UpdateCategory 更新分类", func(t *testing.T) {
		_, err := svc.UpdateCategory(ctx, cat.ID, &UpdateCategoryRequest{Name: "更新分类"})
		require.NoError(t, err)

		var updated models.Category
		db.First(&updated, cat.ID)
		assert.Equal(t, "更新分类", updated.Name)
	})

	t.Run("GetAllCategories 获取所有分类", func(t *testing.T) {
		list, err := svc.GetAllCategories(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})
}

func TestProductAdminService_GetProductsAndDetail(t *testing.T) {
	db := setupProductAdminTestDB(t)
	svc := NewProductAdminService(
		db,
		repository.NewCategoryRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSkuRepository(db),
	)
	ctx := context.Background()

	cat, _ := svc.CreateCategory(ctx, &CreateCategoryRequest{Name: "商品分类"})
	product, _ := svc.CreateProduct(ctx, &CreateProductRequest{
		CategoryID: cat.ID,
		Name:       "商品列表测试",
		Images:     []string{"img1"},
		Price:      10,
		Stock:      5,
		Unit:       "件",
		IsOnSale:   true,
	})

	t.Run("GetProducts 获取商品列表", func(t *testing.T) {
		list, total, err := svc.GetProducts(ctx, &ProductListParams{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.True(t, total >= 1)
		assert.NotEmpty(t, list)
	})

	t.Run("GetProductDetail 获取商品详情", func(t *testing.T) {
		detail, err := svc.GetProductDetail(ctx, product.ID)
		require.NoError(t, err)
		assert.Equal(t, product.ID, detail.ID)
		assert.Equal(t, "商品列表测试", detail.Name)
	})

	t.Run("GetProductDetail 商品不存在", func(t *testing.T) {
		_, err := svc.GetProductDetail(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestProductAdminService_UpdateAndDeleteProduct(t *testing.T) {
	db := setupProductAdminTestDB(t)
	svc := NewProductAdminService(
		db,
		repository.NewCategoryRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSkuRepository(db),
	)
	ctx := context.Background()

	cat, _ := svc.CreateCategory(ctx, &CreateCategoryRequest{Name: "操作分类"})
	product, _ := svc.CreateProduct(ctx, &CreateProductRequest{
		CategoryID: cat.ID,
		Name:       "操作测试商品",
		Images:     []string{"img1"},
		Price:      10,
		Stock:      5,
		Unit:       "件",
		IsOnSale:   true,
	})

	t.Run("UpdateProduct 更新商品", func(t *testing.T) {
		newPrice := 20.0
		_, err := svc.UpdateProduct(ctx, product.ID, &UpdateProductRequest{
			Name:  "更新后商品",
			Price: &newPrice,
		})
		require.NoError(t, err)

		var updated models.Product
		db.First(&updated, product.ID)
		assert.Equal(t, "更新后商品", updated.Name)
		assert.Equal(t, 20.0, updated.Price)
	})

	t.Run("UpdateProductStatus 更新商品状态", func(t *testing.T) {
		err := svc.UpdateProductStatus(ctx, product.ID, false)
		require.NoError(t, err)

		var updated models.Product
		db.First(&updated, product.ID)
		assert.False(t, updated.IsOnSale)
	})

	t.Run("DeleteProduct 删除商品", func(t *testing.T) {
		err := svc.DeleteProduct(ctx, product.ID)
		require.NoError(t, err)

		var count int64
		db.Model(&models.Product{}).Where("id = ?", product.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

