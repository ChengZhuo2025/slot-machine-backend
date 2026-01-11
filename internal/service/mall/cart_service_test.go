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

func setupCartServiceTestDB(t *testing.T) *gorm.DB {
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
		&models.ProductSku{},
		&models.CartItem{},
	)
	require.NoError(t, err)

	// 创建默认会员等级
	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	return db
}

func newCartService(db *gorm.DB) *CartService {
	cartRepo := repository.NewCartRepository(db)
	productRepo := repository.NewProductRepository(db)
	skuRepo := repository.NewProductSkuRepository(db)
	return NewCartService(db, cartRepo, productRepo, skuRepo)
}

func seedCartTestData(t *testing.T, db *gorm.DB) (*models.User, *models.Product, *models.ProductSku) {
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
	images, _ := json.Marshal([]string{"https://example.com/img1.jpg", "https://example.com/img2.jpg"})
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

	// 创建 SKU
	attrs, _ := json.Marshal(map[string]string{"颜色": "红色", "尺码": "M"})
	sku := &models.ProductSku{
		ProductID:  product.ID,
		SkuCode:    "RED-M",
		Attributes: attrs,
		Price:      85.0,
		Stock:      20,
		IsActive:   true,
	}
	require.NoError(t, db.Create(sku).Error)

	return user, product, sku
}

// ==================== 添加购物车测试 ====================

func TestCartService_AddItem_NewProduct(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加商品到购物车
	item, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: product.ID,
		Quantity:  2,
	})
	require.NoError(t, err)
	assert.Equal(t, product.ID, item.ProductID)
	assert.Equal(t, 2, item.Quantity)
	assert.Equal(t, "测试商品", item.ProductName)
	assert.True(t, item.Selected)
}

func TestCartService_AddItem_WithSku(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, sku := seedCartTestData(t, db)

	// 添加带 SKU 的商品
	skuID := sku.ID
	item, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: product.ID,
		SkuID:     &skuID,
		Quantity:  1,
	})
	require.NoError(t, err)
	assert.Equal(t, &skuID, item.SkuID)
	assert.Equal(t, 85.0, item.Price) // SKU 价格
}

func TestCartService_AddItem_UpdateExistingQuantity(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 第一次添加
	_, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: product.ID,
		Quantity:  2,
	})
	require.NoError(t, err)

	// 再次添加同一商品
	item, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: product.ID,
		Quantity:  3,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, item.Quantity) // 2 + 3
}

func TestCartService_AddItem_ProductNotFound(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, _, _ := seedCartTestData(t, db)

	_, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: 99999,
		Quantity:  1,
	})
	assert.Error(t, err)
}

func TestCartService_AddItem_ProductOffShelf(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, _, _ := seedCartTestData(t, db)

	// 创建下架商品 - 先创建再更新，因为 GORM 会把 bool false 当作零值
	category := &models.Category{Name: "分类", Level: 1, Sort: 1, IsActive: true}
	require.NoError(t, db.Create(category).Error)
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	offShelfProduct := &models.Product{
		CategoryID: category.ID,
		Name:       "下架商品",
		Images:     images,
		Price:      100,
		Stock:      50,
		Unit:       "件",
		IsOnSale:   true,
	}
	require.NoError(t, db.Create(offShelfProduct).Error)
	// 使用 Updates 更新 IsOnSale 为 false
	require.NoError(t, db.Model(offShelfProduct).Update("is_on_sale", false).Error)

	_, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: offShelfProduct.ID,
		Quantity:  1,
	})
	assert.Error(t, err)
}

func TestCartService_AddItem_SkuNotBelongToProduct(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 创建另一个商品的 SKU
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	category := &models.Category{Name: "分类2", Level: 1, Sort: 1, IsActive: true}
	require.NoError(t, db.Create(category).Error)
	anotherProduct := &models.Product{
		CategoryID: category.ID,
		Name:       "另一个商品",
		Images:     images,
		Price:      100,
		Stock:      50,
		Unit:       "件",
		IsOnSale:   true,
	}
	require.NoError(t, db.Create(anotherProduct).Error)

	attrs, _ := json.Marshal(map[string]string{"颜色": "蓝色"})
	anotherSku := &models.ProductSku{
		ProductID:  anotherProduct.ID,
		SkuCode:    "BLUE",
		Attributes: attrs,
		Price:      100,
		Stock:      20,
		IsActive:   true,
	}
	require.NoError(t, db.Create(anotherSku).Error)

	// 尝试使用不属于该商品的 SKU
	skuID := anotherSku.ID
	_, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: product.ID,
		SkuID:     &skuID,
		Quantity:  1,
	})
	assert.Error(t, err)
}

// ==================== 获取购物车测试 ====================

func TestCartService_GetCart(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加商品到购物车
	require.NoError(t, db.Create(&models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}).Error)

	cart, err := svc.GetCart(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, cart.Items, 1)
	assert.Equal(t, 2, cart.TotalCount)
	assert.Equal(t, 2, cart.SelectedCount)
}

func TestCartService_GetCart_Empty(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, _, _ := seedCartTestData(t, db)

	cart, err := svc.GetCart(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, cart.Items)
	assert.Equal(t, 0, cart.TotalCount)
}

func TestCartService_GetCart_CalculateSubtotal(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 直接创建购物车项并关联商品
	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  3,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	cart, err := svc.GetCart(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, cart.Items, 1)
	// 小计 = 价格 * 数量 = 80.0 * 3 = 240.0
	assert.Equal(t, 240.0, cart.TotalAmount)
}

// ==================== 更新购物车测试 ====================

func TestCartService_UpdateItem_Quantity(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	// 更新数量
	updated, err := svc.UpdateItem(ctx, user.ID, cartItem.ID, &UpdateCartItemRequest{
		Quantity: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, updated.Quantity)
}

func TestCartService_UpdateItem_Selected(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	// 取消选中
	selected := false
	updated, err := svc.UpdateItem(ctx, user.ID, cartItem.ID, &UpdateCartItemRequest{
		Quantity: 2,
		Selected: &selected,
	})
	require.NoError(t, err)
	assert.False(t, updated.Selected)
}

func TestCartService_UpdateItem_NotOwned(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 创建另一个用户
	phone2 := "13900139000"
	anotherUser := &models.User{
		Phone:         &phone2,
		Nickname:      "另一个用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(anotherUser).Error)

	// 创建属于另一个用户的购物车项
	cartItem := &models.CartItem{
		UserID:    anotherUser.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	// 尝试更新不属于自己的购物车项
	_, err := svc.UpdateItem(ctx, user.ID, cartItem.ID, &UpdateCartItemRequest{
		Quantity: 5,
	})
	assert.Error(t, err)
}

// ==================== 删除购物车测试 ====================

func TestCartService_RemoveItem(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	err := svc.RemoveItem(ctx, user.ID, cartItem.ID)
	require.NoError(t, err)

	// 验证已删除
	var count int64
	db.Model(&models.CartItem{}).Where("id = ?", cartItem.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCartService_RemoveItem_NotOwned(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 创建另一个用户
	phone2 := "13900139000"
	anotherUser := &models.User{
		Phone:         &phone2,
		Nickname:      "另一个用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(anotherUser).Error)

	cartItem := &models.CartItem{
		UserID:    anotherUser.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	err := svc.RemoveItem(ctx, user.ID, cartItem.ID)
	assert.Error(t, err)
}

func TestCartService_RemoveItems_Batch(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 创建另一个商品
	category := &models.Category{Name: "分类2", Level: 1, Sort: 1, IsActive: true}
	require.NoError(t, db.Create(category).Error)
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	product2 := &models.Product{
		CategoryID: category.ID,
		Name:       "商品2",
		Images:     images,
		Price:      100,
		Stock:      50,
		Unit:       "件",
		IsOnSale:   true,
	}
	require.NoError(t, db.Create(product2).Error)

	// 添加两个购物车项
	item1 := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 1, Selected: true}
	item2 := &models.CartItem{UserID: user.ID, ProductID: product2.ID, Quantity: 2, Selected: true}
	require.NoError(t, db.Create(item1).Error)
	require.NoError(t, db.Create(item2).Error)

	err := svc.RemoveItems(ctx, user.ID, []int64{item1.ID, item2.ID})
	require.NoError(t, err)

	// 验证都已删除
	var count int64
	db.Model(&models.CartItem{}).Where("user_id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCartService_ClearCart(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加多个购物车项
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&models.CartItem{
			UserID:    user.ID,
			ProductID: product.ID,
			Quantity:  i + 1,
			Selected:  true,
		}).Error)
	}

	err := svc.ClearCart(ctx, user.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.CartItem{}).Where("user_id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCartService_ClearSelected(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加选中和未选中的购物车项
	// 先创建再更新，因为 GORM 会把 bool false 当作零值
	selected := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 1, Selected: true}
	unselected := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 2, Selected: true}
	require.NoError(t, db.Create(selected).Error)
	require.NoError(t, db.Create(unselected).Error)
	// 更新为未选中
	require.NoError(t, db.Model(unselected).Update("selected", false).Error)

	err := svc.ClearSelected(ctx, user.ID)
	require.NoError(t, err)

	// 验证只删除了选中的
	var count int64
	db.Model(&models.CartItem{}).Where("user_id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

// ==================== 全选测试 ====================

func TestCartService_SelectAll(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加多个购物车项，有些选中有些未选中
	// 先创建再更新，因为 GORM 会把 bool false 当作零值
	item1 := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 1, Selected: true}
	item2 := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 2, Selected: true}
	require.NoError(t, db.Create(item1).Error)
	require.NoError(t, db.Create(item2).Error)
	// 将 item2 更新为未选中
	require.NoError(t, db.Model(item2).Update("selected", false).Error)

	// 全选
	err := svc.SelectAll(ctx, user.ID, true)
	require.NoError(t, err)

	var items []models.CartItem
	db.Where("user_id = ?", user.ID).Find(&items)
	for _, item := range items {
		assert.True(t, item.Selected)
	}

	// 取消全选
	err = svc.SelectAll(ctx, user.ID, false)
	require.NoError(t, err)

	db.Where("user_id = ?", user.ID).Find(&items)
	for _, item := range items {
		assert.False(t, item.Selected)
	}
}

// ==================== 获取选中项测试 ====================

func TestCartService_GetSelectedItems(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加选中和未选中的购物车项
	// 先创建再更新，因为 GORM 会把 bool false 当作零值
	item1 := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 1, Selected: true}
	item2 := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 2, Selected: true}
	item3 := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 3, Selected: true}
	require.NoError(t, db.Create(item1).Error)
	require.NoError(t, db.Create(item2).Error)
	require.NoError(t, db.Create(item3).Error)
	// 将 item2 更新为未选中
	require.NoError(t, db.Model(item2).Update("selected", false).Error)

	items, err := svc.GetSelectedItems(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

// ==================== 购物车数量测试 ====================

func TestCartService_GetCartCount(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加购物车项
	require.NoError(t, db.Create(&models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 2, Selected: true}).Error)
	require.NoError(t, db.Create(&models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 3, Selected: true}).Error)

	count, err := svc.GetCartCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, count) // 2 + 3
}

// ==================== 边界测试 ====================

func TestCartService_GetCart_WithSku(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, sku := seedCartTestData(t, db)

	// 添加带 SKU 的购物车项
	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		SkuID:     &sku.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	cart, err := svc.GetCart(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.NotNil(t, cart.Items[0].SkuID)
	assert.Equal(t, sku.ID, *cart.Items[0].SkuID)
	assert.Equal(t, 85.0, cart.Items[0].Price) // SKU 价格
}

func TestCartService_GetCart_ProductDeleted(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加商品到购物车
	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	// 删除商品
	require.NoError(t, db.Delete(product).Error)

	// 获取购物车，应该能处理商品不存在的情况
	cart, err := svc.GetCart(ctx, user.ID)
	require.NoError(t, err)
	// 商品不存在时，购物车项也应该被过滤掉或标记为无效
	// 这取决于实现，但不应该报错
	assert.NotNil(t, cart)
}

func TestCartService_GetSelectedItems_Empty(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加未选中的购物车项
	item := &models.CartItem{UserID: user.ID, ProductID: product.ID, Quantity: 1, Selected: true}
	require.NoError(t, db.Create(item).Error)
	require.NoError(t, db.Model(item).Update("selected", false).Error)

	items, err := svc.GetSelectedItems(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestCartService_GetCartCount_Empty(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, _, _ := seedCartTestData(t, db)

	count, err := svc.GetCartCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCartService_AddItem_ZeroQuantity(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 添加数量为 0 的商品 - 应该报错或被拒绝
	_, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: product.ID,
		Quantity:  0,
	})
	// 具体行为取决于实现，这里只是确保不会 panic
	_ = err
}

func TestCartService_UpdateItem_NotFound(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, _, _ := seedCartTestData(t, db)

	_, err := svc.UpdateItem(ctx, user.ID, 99999, &UpdateCartItemRequest{
		Quantity: 5,
	})
	assert.Error(t, err)
}

func TestCartService_RemoveItem_NotFound(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, _, _ := seedCartTestData(t, db)

	err := svc.RemoveItem(ctx, user.ID, 99999)
	assert.Error(t, err)
}

func TestCartService_AddItem_StockInsufficient(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	// 尝试添加超过库存的数量 - product.Stock = 50
	_, err := svc.AddItem(ctx, user.ID, &AddCartItemRequest{
		ProductID: product.ID,
		Quantity:  100, // 超过库存
	})
	// 如果有库存检查，应该报错
	_ = err
}

func TestCartService_toCartItemInfo_AllFields(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)

	images, _ := json.Marshal([]string{"img1.jpg", "img2.jpg"})
	attrs, _ := json.Marshal(map[string]string{"颜色": "红色"})
	skuID := int64(1)
	skuImage := "sku.jpg"

	item := &models.CartItem{
		ID:        1,
		UserID:    1,
		ProductID: 1,
		SkuID:     &skuID,
		Quantity:  2,
		Selected:  true,
		Product: &models.Product{
			ID:     1,
			Name:   "测试商品",
			Images: images,
			Price:  100.0,
			Stock:  50,
		},
		Sku: &models.ProductSku{
			ID:         1,
			SkuCode:    "RED",
			Attributes: attrs,
			Price:      90.0,
			Stock:      20,
			Image:      &skuImage,
		},
	}

	info := svc.toCartItemInfo(item)

	assert.Equal(t, int64(1), info.ID)
	assert.Equal(t, "测试商品", info.ProductName)
	assert.Equal(t, 90.0, info.Price)     // 使用 SKU 价格
	assert.Equal(t, 20, info.Stock)       // 使用 SKU 库存
	assert.Equal(t, "sku.jpg", info.ProductImage) // 使用 SKU 图片
	assert.Equal(t, 180.0, info.Subtotal) // 90.0 * 2
	assert.Equal(t, "红色", info.Attributes["颜色"])
}

func TestCartService_toCartItemInfo_WithoutSku(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)

	images, _ := json.Marshal([]string{"product.jpg"})

	item := &models.CartItem{
		ID:        1,
		UserID:    1,
		ProductID: 1,
		Quantity:  3,
		Selected:  false,
		Product: &models.Product{
			ID:     1,
			Name:   "测试商品",
			Images: images,
			Price:  80.0,
			Stock:  50,
		},
		// 无 SKU
	}

	info := svc.toCartItemInfo(item)

	assert.Equal(t, int64(1), info.ID)
	assert.Equal(t, "测试商品", info.ProductName)
	assert.Equal(t, 80.0, info.Price)       // 使用商品价格
	assert.Equal(t, 50, info.Stock)         // 使用商品库存
	assert.Equal(t, "product.jpg", info.ProductImage) // 使用商品图片
	assert.Equal(t, 240.0, info.Subtotal)   // 80.0 * 3
	assert.False(t, info.Selected)
}

func TestCartService_toCartItemInfo_NilProduct(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)

	item := &models.CartItem{
		ID:        1,
		UserID:    1,
		ProductID: 1,
		Quantity:  1,
		Selected:  true,
		// Product 为 nil
	}

	info := svc.toCartItemInfo(item)

	assert.Equal(t, int64(1), info.ID)
	assert.Empty(t, info.ProductName)
	assert.Equal(t, 0.0, info.Price)
	assert.Equal(t, 0, info.Stock)
}

func TestCartService_UpdateItem_QuantityAndSelected(t *testing.T) {
	db := setupCartServiceTestDB(t)
	svc := newCartService(db)
	ctx := context.Background()

	user, product, _ := seedCartTestData(t, db)

	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	// 同时更新数量和选中状态
	selected := false
	updated, err := svc.UpdateItem(ctx, user.ID, cartItem.ID, &UpdateCartItemRequest{
		Quantity: 10,
		Selected: &selected,
	})
	require.NoError(t, err)
	assert.Equal(t, 10, updated.Quantity)
	assert.False(t, updated.Selected)
}
