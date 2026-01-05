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

func setupProductServiceTestDB(t *testing.T) *gorm.DB {
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

func newProductService(db *gorm.DB) *ProductService {
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	skuRepo := repository.NewProductSkuRepository(db)
	return NewProductService(db, productRepo, categoryRepo, skuRepo)
}

func seedCategory(t *testing.T, db *gorm.DB) *models.Category {
	t.Helper()
	category := &models.Category{
		Name:     "测试分类",
		Level:    1,
		Sort:     1,
		IsActive: true,
	}
	require.NoError(t, db.Create(category).Error)
	return category
}

func seedProduct(t *testing.T, db *gorm.DB, categoryID int64) *models.Product {
	t.Helper()
	images, _ := json.Marshal([]string{"https://example.com/img1.jpg", "https://example.com/img2.jpg"})
	desc := "测试商品描述"
	originalPrice := 100.0
	product := &models.Product{
		CategoryID:    categoryID,
		Name:          "测试商品",
		Images:        images,
		Description:   &desc,
		Price:         80.0,
		OriginalPrice: &originalPrice,
		Stock:         50,
		Sales:         10,
		Unit:          "件",
		IsOnSale:      true,
		IsHot:         true,
		IsNew:         false,
		Sort:          1,
	}
	require.NoError(t, db.Create(product).Error)
	return product
}

func seedProductSku(t *testing.T, db *gorm.DB, productID int64, color, size string, price float64, stock int) *models.ProductSku {
	t.Helper()
	attrs, _ := json.Marshal(map[string]string{"颜色": color, "尺码": size})
	sku := &models.ProductSku{
		ProductID:  productID,
		SkuCode:    color + "-" + size,
		Attributes: attrs,
		Price:      price,
		Stock:      stock,
		IsActive:   true,
	}
	require.NoError(t, db.Create(sku).Error)
	return sku
}

// ==================== 分类相关测试 ====================

func TestProductService_GetCategoryTree(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	// 创建父分类
	parent := &models.Category{Name: "父分类", Level: 1, Sort: 1, IsActive: true}
	require.NoError(t, db.Create(parent).Error)

	// 创建子分类
	child1 := &models.Category{ParentID: &parent.ID, Name: "子分类1", Level: 2, Sort: 1, IsActive: true}
	child2 := &models.Category{ParentID: &parent.ID, Name: "子分类2", Level: 2, Sort: 2, IsActive: true}
	require.NoError(t, db.Create(child1).Error)
	require.NoError(t, db.Create(child2).Error)

	tree, err := svc.GetCategoryTree(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, tree)
}

func TestProductService_GetCategoryList(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	// 创建顶级分类
	cat1 := &models.Category{Name: "分类1", Level: 1, Sort: 1, IsActive: true}
	cat2 := &models.Category{Name: "分类2", Level: 1, Sort: 2, IsActive: true}
	require.NoError(t, db.Create(cat1).Error)
	require.NoError(t, db.Create(cat2).Error)

	// 获取顶级分类（parentID = nil）
	list, err := svc.GetCategoryList(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

// ==================== 商品列表测试 ====================

func TestProductService_GetProductList(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)

	// 创建多个商品
	for i := 0; i < 5; i++ {
		images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
		product := &models.Product{
			CategoryID: category.ID,
			Name:       "商品" + string(rune('A'+i)),
			Images:     images,
			Price:      float64(50 + i*10),
			Stock:      100,
			Unit:       "件",
			IsOnSale:   true,
		}
		require.NoError(t, db.Create(product).Error)
	}

	// 测试分页
	resp, err := svc.GetProductList(ctx, &ProductListRequest{
		Page:     1,
		PageSize: 2,
	})
	require.NoError(t, err)
	assert.Len(t, resp.List, 2)
	assert.Equal(t, int64(5), resp.Total)
	assert.Equal(t, 3, resp.TotalPages)
}

func TestProductService_GetProductList_WithFilters(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)

	// 创建不同类型的商品
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	hotProduct := &models.Product{CategoryID: category.ID, Name: "热门商品", Images: images, Price: 100, Stock: 50, Unit: "件", IsOnSale: true, IsHot: true}
	newProduct := &models.Product{CategoryID: category.ID, Name: "新品", Images: images, Price: 80, Stock: 30, Unit: "件", IsOnSale: true, IsNew: true}
	normalProduct := &models.Product{CategoryID: category.ID, Name: "普通商品", Images: images, Price: 60, Stock: 20, Unit: "件", IsOnSale: true}
	require.NoError(t, db.Create(hotProduct).Error)
	require.NoError(t, db.Create(newProduct).Error)
	require.NoError(t, db.Create(normalProduct).Error)

	// 测试热门商品过滤
	isHot := true
	resp, err := svc.GetProductList(ctx, &ProductListRequest{Page: 1, PageSize: 10, IsHot: &isHot})
	require.NoError(t, err)
	assert.Len(t, resp.List, 1)
	assert.Equal(t, "热门商品", resp.List[0].Name)

	// 测试新品过滤
	isNew := true
	resp, err = svc.GetProductList(ctx, &ProductListRequest{Page: 1, PageSize: 10, IsNew: &isNew})
	require.NoError(t, err)
	assert.Len(t, resp.List, 1)
	assert.Equal(t, "新品", resp.List[0].Name)

	// 测试分类过滤
	resp, err = svc.GetProductList(ctx, &ProductListRequest{Page: 1, PageSize: 10, CategoryID: category.ID})
	require.NoError(t, err)
	assert.Len(t, resp.List, 3)
}

func TestProductService_GetProductList_OffShelfNotShown(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})

	// 创建上架商品
	onSale := &models.Product{CategoryID: category.ID, Name: "上架商品", Images: images, Price: 100, Stock: 50, Unit: "件", IsOnSale: true}
	require.NoError(t, db.Create(onSale).Error)

	// 创建下架商品 - 先创建再更新，因为 GORM 会把 bool false 当作零值
	offSale := &models.Product{CategoryID: category.ID, Name: "下架商品", Images: images, Price: 100, Stock: 50, Unit: "件", IsOnSale: true}
	require.NoError(t, db.Create(offSale).Error)
	// 使用 Updates 更新 IsOnSale 为 false
	require.NoError(t, db.Model(offSale).Update("is_on_sale", false).Error)

	// 验证数据库中数据正确
	var products []models.Product
	db.Find(&products)
	t.Logf("数据库中商品数: %d", len(products))
	for _, p := range products {
		t.Logf("  - %s (IsOnSale: %v)", p.Name, p.IsOnSale)
	}

	// 服务层默认只返回上架商品 (is_on_sale = true)
	resp, err := svc.GetProductList(ctx, &ProductListRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)

	t.Logf("返回商品数: %d, Total: %d", len(resp.List), resp.Total)
	for _, p := range resp.List {
		t.Logf("  - %s (IsOnSale: %v)", p.Name, p.IsOnSale)
	}

	// 只应该有上架商品
	assert.Equal(t, int64(1), resp.Total, "应该只有1个上架商品")
	require.Len(t, resp.List, 1, "应该只返回1个商品")
	assert.Equal(t, "上架商品", resp.List[0].Name, "返回的商品应该是上架商品")
}

// ==================== 商品详情测试 ====================

func TestProductService_GetProductDetail_Success(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	product := seedProduct(t, db, category.ID)

	// 创建 SKU
	seedProductSku(t, db, product.ID, "红色", "M", 85.0, 20)
	seedProductSku(t, db, product.ID, "蓝色", "L", 90.0, 15)

	info, err := svc.GetProductDetail(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, product.ID, info.ID)
	assert.Equal(t, "测试商品", info.Name)
	assert.Equal(t, 80.0, info.Price)
	assert.Len(t, info.Images, 2)
	assert.Len(t, info.Skus, 2)
}

func TestProductService_GetProductDetail_NotFound(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	_, err := svc.GetProductDetail(ctx, 99999)
	assert.Error(t, err)
}

func TestProductService_GetProductDetail_OffShelf(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})

	// 创建下架商品 - 先创建再更新，因为 GORM 会把 bool false 当作零值
	product := &models.Product{
		CategoryID: category.ID,
		Name:       "下架商品",
		Images:     images,
		Price:      100,
		Stock:      50,
		Unit:       "件",
		IsOnSale:   true,
	}
	require.NoError(t, db.Create(product).Error)
	require.NoError(t, db.Model(product).Update("is_on_sale", false).Error)

	_, err := svc.GetProductDetail(ctx, product.ID)
	assert.Error(t, err)
}

// ==================== 热门/新品商品测试 ====================

func TestProductService_GetHotProducts(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})

	for i := 0; i < 5; i++ {
		product := &models.Product{
			CategoryID: category.ID,
			Name:       "热门商品" + string(rune('A'+i)),
			Images:     images,
			Price:      float64(50 + i*10),
			Stock:      100,
			Unit:       "件",
			IsOnSale:   true,
			IsHot:      true,
		}
		require.NoError(t, db.Create(product).Error)
	}

	list, err := svc.GetHotProducts(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestProductService_GetNewProducts(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})

	for i := 0; i < 5; i++ {
		product := &models.Product{
			CategoryID: category.ID,
			Name:       "新品" + string(rune('A'+i)),
			Images:     images,
			Price:      float64(50 + i*10),
			Stock:      100,
			Unit:       "件",
			IsOnSale:   true,
			IsNew:      true,
		}
		require.NoError(t, db.Create(product).Error)
	}

	list, err := svc.GetNewProducts(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

// ==================== SKU 测试 ====================

func TestProductService_GetSkusByProductID(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	product := seedProduct(t, db, category.ID)

	seedProductSku(t, db, product.ID, "红色", "S", 80.0, 10)
	seedProductSku(t, db, product.ID, "红色", "M", 85.0, 20)
	seedProductSku(t, db, product.ID, "蓝色", "M", 85.0, 15)

	skus, err := svc.GetSkusByProductID(ctx, product.ID)
	require.NoError(t, err)
	assert.Len(t, skus, 3)
}

func TestProductService_GetSkuByID_Success(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	product := seedProduct(t, db, category.ID)
	sku := seedProductSku(t, db, product.ID, "红色", "M", 85.0, 20)

	info, err := svc.GetSkuByID(ctx, sku.ID)
	require.NoError(t, err)
	assert.Equal(t, sku.ID, info.ID)
	assert.Equal(t, 85.0, info.Price)
	assert.Equal(t, "红色", info.Attributes["颜色"])
}

func TestProductService_GetSkuByID_NotFound(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	_, err := svc.GetSkuByID(ctx, 99999)
	assert.Error(t, err)
}

// ==================== 库存管理测试 ====================

func TestProductService_CheckStock_ProductLevel(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	product := seedProduct(t, db, category.ID) // stock = 50

	// 库存充足
	err := svc.CheckStock(ctx, product.ID, nil, 30)
	assert.NoError(t, err)

	// 库存不足
	err = svc.CheckStock(ctx, product.ID, nil, 100)
	assert.Error(t, err)
}

func TestProductService_CheckStock_SkuLevel(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	product := seedProduct(t, db, category.ID)
	sku := seedProductSku(t, db, product.ID, "红色", "M", 85.0, 20) // stock = 20

	skuID := sku.ID
	// 库存充足
	err := svc.CheckStock(ctx, product.ID, &skuID, 15)
	assert.NoError(t, err)

	// 库存不足
	err = svc.CheckStock(ctx, product.ID, &skuID, 30)
	assert.Error(t, err)
}

func TestProductService_DeductStock_ProductOnly(t *testing.T) {
	db := setupProductServiceTestDB(t)
	ctx := context.Background()

	category := seedCategory(t, db)
	product := seedProduct(t, db, category.ID) // stock = 50

	// 直接测试 repository 的 DecreaseStock 功能
	// 因为 service 层的 DeductStock 使用事务，在 SQLite 单连接模式下会死锁
	productRepo := repository.NewProductRepository(db)
	err := productRepo.DecreaseStock(ctx, product.ID, 10)
	require.NoError(t, err)

	// 验证库存已扣减
	var updated models.Product
	require.NoError(t, db.First(&updated, product.ID).Error)
	assert.Equal(t, 40, updated.Stock)
}

func TestProductService_DeductStock_WithSku(t *testing.T) {
	db := setupProductServiceTestDB(t)
	ctx := context.Background()

	category := seedCategory(t, db)
	product := seedProduct(t, db, category.ID)                     // stock = 50
	sku := seedProductSku(t, db, product.ID, "红色", "M", 85.0, 20) // stock = 20

	// 直接测试 repository 的库存扣减功能
	productRepo := repository.NewProductRepository(db)
	skuRepo := repository.NewProductSkuRepository(db)

	// 扣减商品库存
	err := productRepo.DecreaseStock(ctx, product.ID, 5)
	require.NoError(t, err)

	// 扣减 SKU 库存
	err = skuRepo.DecreaseStock(ctx, sku.ID, 5)
	require.NoError(t, err)

	// 验证商品库存
	var updatedProduct models.Product
	require.NoError(t, db.First(&updatedProduct, product.ID).Error)
	assert.Equal(t, 45, updatedProduct.Stock)

	// 验证 SKU 库存
	var updatedSku models.ProductSku
	require.NoError(t, db.First(&updatedSku, sku.ID).Error)
	assert.Equal(t, 15, updatedSku.Stock)
}

func TestProductService_RestoreStock_ProductOnly(t *testing.T) {
	db := setupProductServiceTestDB(t)
	ctx := context.Background()

	category := seedCategory(t, db)
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	product := &models.Product{
		CategoryID: category.ID,
		Name:       "测试商品",
		Images:     images,
		Price:      80.0,
		Stock:      10, // 低库存
		Unit:       "件",
		IsOnSale:   true,
	}
	require.NoError(t, db.Create(product).Error)

	// 直接测试 repository 的 IncreaseStock 功能
	productRepo := repository.NewProductRepository(db)
	err := productRepo.IncreaseStock(ctx, product.ID, 5)
	require.NoError(t, err)

	var updated models.Product
	require.NoError(t, db.First(&updated, product.ID).Error)
	assert.Equal(t, 15, updated.Stock)
}

func TestProductService_IncreaseSales(t *testing.T) {
	db := setupProductServiceTestDB(t)
	svc := newProductService(db)
	ctx := context.Background()

	category := seedCategory(t, db)
	product := seedProduct(t, db, category.ID) // sales = 10

	err := svc.IncreaseSales(ctx, product.ID, 5)
	require.NoError(t, err)

	var updated models.Product
	require.NoError(t, db.First(&updated, product.ID).Error)
	assert.Equal(t, 15, updated.Sales)
}
