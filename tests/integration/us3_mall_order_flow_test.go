//go:build integration
// +build integration

package integration

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
	mallService "github.com/dumeirei/smart-locker-backend/internal/service/mall"
)

func setupUS3IntegrationDB(t *testing.T) *gorm.DB {
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
	// 允许多个连接：订单创建使用事务 + 仓储层非 tx DB 调用，单连接会导致 SQLite 死锁
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Address{},
		&models.Category{},
		&models.Product{},
		&models.ProductSku{},
		&models.CartItem{},
		&models.Order{},
		&models.OrderItem{},
		&models.Review{},
	)
	require.NoError(t, err)

	// 创建默认会员等级
	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	return db
}

func setupUS3Services(db *gorm.DB) (*mallService.ProductService, *mallService.CartService, *mallService.MallOrderService, *mallService.ReviewService) {
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	skuRepo := repository.NewProductSkuRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	reviewRepo := repository.NewReviewRepository(db)

	productSvc := mallService.NewProductService(db, productRepo, categoryRepo, skuRepo)
	cartSvc := mallService.NewCartService(db, cartRepo, productRepo, skuRepo)
	orderSvc := mallService.NewMallOrderService(db, orderRepo, cartRepo, productRepo, skuRepo, productSvc)
	reviewSvc := mallService.NewReviewService(db, reviewRepo, orderRepo)

	return productSvc, cartSvc, orderSvc, reviewSvc
}

func seedUS3IntegrationData(t *testing.T, db *gorm.DB) (*models.User, *models.Category, *models.Product, *models.ProductSku, *models.Address) {
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
	require.NoError(t, db.Create(&models.UserWallet{UserID: user.ID, Balance: 1000.0}).Error)

	// 创建地址
	address := &models.Address{
		UserID:        user.ID,
		ReceiverName:  "测试收货人",
		ReceiverPhone: "13900139000",
		Province:      "广东省",
		City:          "深圳市",
		District:      "南山区",
		Detail:        "科技园路1号",
		IsDefault:     true,
	}
	require.NoError(t, db.Create(address).Error)

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
		Sales:      0,
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

	return user, category, product, sku, address
}

// ==================== 商城订单完整流程测试 ====================

func TestUS3Integration_MallOrderFlow_DirectPurchase(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	productSvc, _, orderSvc, reviewSvc := setupUS3Services(db)
	ctx := context.Background()

	user, _, product, _, address := seedUS3IntegrationData(t, db)

	// 1. 浏览商品
	productInfo, err := productSvc.GetProductDetail(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, "测试商品", productInfo.Name)
	assert.Equal(t, 80.0, productInfo.Price)
	assert.Equal(t, 50, productInfo.Stock)

	// 2. 直接购买商品
	order, err := orderSvc.CreateOrder(ctx, user.ID, &mallService.CreateMallOrderRequest{
		Items: []mallService.OrderItemRequest{
			{ProductID: product.ID, Quantity: 2},
		},
		AddressID: address.ID,
		Remark:    "直接购买测试",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, order.OrderNo)
	assert.Equal(t, models.OrderStatusPending, order.Status)
	assert.Equal(t, 160.0, order.OriginalAmount) // 80 * 2
	assert.Equal(t, "直接购买测试", order.Remark)

	// 3. 验证库存已扣减
	updatedProduct, err := productSvc.GetProductDetail(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, 48, updatedProduct.Stock) // 50 - 2

	// 4. 模拟支付成功 - 更新订单状态
	db.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status": models.OrderStatusPaid,
	})

	// 5. 模拟发货
	db.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status": models.OrderStatusShipped,
	})

	// 6. 确认收货
	err = orderSvc.ConfirmReceive(ctx, user.ID, order.ID)
	require.NoError(t, err)

	// 7. 验证订单状态
	orderDetail, err := orderSvc.GetOrderDetail(ctx, user.ID, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusCompleted, orderDetail.Status)

	// 8. 评价商品
	review, err := reviewSvc.CreateReview(ctx, user.ID, &mallService.CreateReviewRequest{
		OrderID:     order.ID,
		ProductID:   product.ID,
		Rating:      5,
		Content:     "商品质量很好，物流很快！",
		IsAnonymous: false,
	})
	require.NoError(t, err)
	assert.Equal(t, int(5), review.Rating)
	assert.Equal(t, "商品质量很好，物流很快！", review.Content)
}

func TestUS3Integration_MallOrderFlow_CartPurchase(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	productSvc, cartSvc, orderSvc, _ := setupUS3Services(db)
	ctx := context.Background()

	user, category, product1, _, address := seedUS3IntegrationData(t, db)

	// 创建另一个商品
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	product2 := &models.Product{
		CategoryID: category.ID,
		Name:       "商品2",
		Images:     images,
		Price:      100.0,
		Stock:      30,
		Unit:       "件",
		IsOnSale:   true,
	}
	require.NoError(t, db.Create(product2).Error)

	// 1. 浏览商品
	p1, err := productSvc.GetProductDetail(ctx, product1.ID)
	require.NoError(t, err)
	assert.Equal(t, "测试商品", p1.Name)

	p2, err := productSvc.GetProductDetail(ctx, product2.ID)
	require.NoError(t, err)
	assert.Equal(t, "商品2", p2.Name)

	// 2. 添加商品到购物车
	_, err = cartSvc.AddItem(ctx, user.ID, &mallService.AddCartItemRequest{
		ProductID: product1.ID,
		Quantity:  2,
	})
	require.NoError(t, err)

	_, err = cartSvc.AddItem(ctx, user.ID, &mallService.AddCartItemRequest{
		ProductID: product2.ID,
		Quantity:  1,
	})
	require.NoError(t, err)

	// 3. 查看购物车
	cart, err := cartSvc.GetCart(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, cart.Items, 2)
	assert.Equal(t, 3, cart.TotalCount)                  // 2 + 1
	assert.Equal(t, 260.0, cart.TotalAmount)             // 80*2 + 100*1

	// 4. 从购物车创建订单
	order, err := orderSvc.CreateOrderFromCart(ctx, user.ID, &mallService.CreateFromCartRequest{
		AddressID: address.ID,
		Remark:    "购物车结算",
	})
	require.NoError(t, err)
	assert.Equal(t, 260.0, order.OriginalAmount)
	assert.Len(t, order.Items, 2)

	// 5. 验证购物车已清空选中项
	cart, err = cartSvc.GetCart(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, cart.Items)

	// 6. 验证库存已扣减
	var updatedP1, updatedP2 models.Product
	db.First(&updatedP1, product1.ID)
	db.First(&updatedP2, product2.ID)
	assert.Equal(t, 48, updatedP1.Stock) // 50 - 2
	assert.Equal(t, 29, updatedP2.Stock) // 30 - 1
}

func TestUS3Integration_MallOrderFlow_WithSku(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	productSvc, cartSvc, orderSvc, _ := setupUS3Services(db)
	ctx := context.Background()

	user, _, product, sku, address := seedUS3IntegrationData(t, db)

	// 1. 获取商品详情（含 SKU）
	productInfo, err := productSvc.GetProductDetail(ctx, product.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, productInfo.Skus)

	// 2. 获取 SKU 信息
	skuInfo, err := productSvc.GetSkuByID(ctx, sku.ID)
	require.NoError(t, err)
	assert.Equal(t, 85.0, skuInfo.Price)
	assert.Equal(t, 20, skuInfo.Stock)

	// 3. 添加带 SKU 的商品到购物车
	skuID := sku.ID
	_, err = cartSvc.AddItem(ctx, user.ID, &mallService.AddCartItemRequest{
		ProductID: product.ID,
		SkuID:     &skuID,
		Quantity:  3,
	})
	require.NoError(t, err)

	// 4. 从购物车创建订单
	order, err := orderSvc.CreateOrderFromCart(ctx, user.ID, &mallService.CreateFromCartRequest{
		AddressID: address.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 255.0, order.OriginalAmount) // 85 * 3 (SKU 价格)

	// 5. 验证 SKU 库存已扣减
	var updatedSku models.ProductSku
	db.First(&updatedSku, sku.ID)
	assert.Equal(t, 17, updatedSku.Stock) // 20 - 3

	// 6. 验证商品总库存也已扣减
	var updatedProduct models.Product
	db.First(&updatedProduct, product.ID)
	assert.Equal(t, 47, updatedProduct.Stock) // 50 - 3
}

func TestUS3Integration_MallOrderFlow_CancelAndRestoreStock(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	_, _, orderSvc, _ := setupUS3Services(db)
	ctx := context.Background()

	user, _, product, _, address := seedUS3IntegrationData(t, db)

	// 记录原始库存
	var originalProduct models.Product
	db.First(&originalProduct, product.ID)
	originalStock := originalProduct.Stock

	// 1. 创建订单
	order, err := orderSvc.CreateOrder(ctx, user.ID, &mallService.CreateMallOrderRequest{
		Items: []mallService.OrderItemRequest{
			{ProductID: product.ID, Quantity: 5},
		},
		AddressID: address.ID,
	})
	require.NoError(t, err)

	// 2. 验证库存已扣减
	var afterPurchase models.Product
	db.First(&afterPurchase, product.ID)
	assert.Equal(t, originalStock-5, afterPurchase.Stock)

	// 3. 取消订单
	err = orderSvc.CancelOrder(ctx, user.ID, order.ID, "不想要了")
	require.NoError(t, err)

	// 4. 验证库存已恢复
	var afterCancel models.Product
	db.First(&afterCancel, product.ID)
	assert.Equal(t, originalStock, afterCancel.Stock)

	// 5. 验证订单状态
	orderDetail, err := orderSvc.GetOrderDetail(ctx, user.ID, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusCancelled, orderDetail.Status)
}

func TestUS3Integration_MallOrderFlow_MultipleOrdersFromSameProduct(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	_, _, orderSvc, _ := setupUS3Services(db)
	ctx := context.Background()

	user, _, product, _, address := seedUS3IntegrationData(t, db)

	// 记录原始库存
	var originalProduct models.Product
	db.First(&originalProduct, product.ID)
	originalStock := originalProduct.Stock

	// 创建多个订单
	for i := 0; i < 3; i++ {
		_, err := orderSvc.CreateOrder(ctx, user.ID, &mallService.CreateMallOrderRequest{
			Items: []mallService.OrderItemRequest{
				{ProductID: product.ID, Quantity: 5},
			},
			AddressID: address.ID,
		})
		require.NoError(t, err)
	}

	// 验证库存扣减正确
	var afterOrders models.Product
	db.First(&afterOrders, product.ID)
	assert.Equal(t, originalStock-15, afterOrders.Stock) // 50 - 5*3 = 35
}

func TestUS3Integration_MallOrderFlow_InsufficientStock(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	_, _, orderSvc, _ := setupUS3Services(db)
	ctx := context.Background()

	user, _, product, _, address := seedUS3IntegrationData(t, db)

	// 尝试购买超过库存数量
	_, err := orderSvc.CreateOrder(ctx, user.ID, &mallService.CreateMallOrderRequest{
		Items: []mallService.OrderItemRequest{
			{ProductID: product.ID, Quantity: 100}, // 库存只有 50
		},
		AddressID: address.ID,
	})
	assert.Error(t, err)

	// 验证库存未变化
	var product2 models.Product
	db.First(&product2, product.ID)
	assert.Equal(t, 50, product2.Stock)
}

// ==================== 评价流程测试 ====================

func TestUS3Integration_ReviewFlow(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	_, _, orderSvc, reviewSvc := setupUS3Services(db)
	ctx := context.Background()

	user, _, product, _, address := seedUS3IntegrationData(t, db)

	// 1. 创建订单并完成
	order, err := orderSvc.CreateOrder(ctx, user.ID, &mallService.CreateMallOrderRequest{
		Items: []mallService.OrderItemRequest{
			{ProductID: product.ID, Quantity: 1},
		},
		AddressID: address.ID,
	})
	require.NoError(t, err)

	// 模拟订单完成
	db.Model(&models.Order{}).Where("id = ?", order.ID).Update("status", models.OrderStatusCompleted)

	// 2. 评价商品
	review, err := reviewSvc.CreateReview(ctx, user.ID, &mallService.CreateReviewRequest{
		OrderID:     order.ID,
		ProductID:   product.ID,
		Rating:      5,
		Content:     "非常满意！",
		Images:      []string{"https://example.com/review1.jpg"},
		IsAnonymous: false,
	})
	require.NoError(t, err)
	assert.Equal(t, int(5), review.Rating)

	// 3. 获取商品评价
	reviews, err := reviewSvc.GetProductReviews(ctx, product.ID, 1, 10)
	require.NoError(t, err)
	assert.Len(t, reviews.List, 1)

	// 4. 获取评价统计
	stats, err := reviewSvc.GetProductReviewStats(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalCount)
	assert.Equal(t, float64(5), stats.AverageRating)

	// 5. 尝试重复评价 - 应该失败
	_, err = reviewSvc.CreateReview(ctx, user.ID, &mallService.CreateReviewRequest{
		OrderID:   order.ID,
		ProductID: product.ID,
		Rating:    4,
	})
	assert.Error(t, err)
}

func TestUS3Integration_ReviewFlow_OrderNotCompleted(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	_, _, orderSvc, reviewSvc := setupUS3Services(db)
	ctx := context.Background()

	user, _, product, _, address := seedUS3IntegrationData(t, db)

	// 创建订单但不完成
	order, err := orderSvc.CreateOrder(ctx, user.ID, &mallService.CreateMallOrderRequest{
		Items: []mallService.OrderItemRequest{
			{ProductID: product.ID, Quantity: 1},
		},
		AddressID: address.ID,
	})
	require.NoError(t, err)

	// 尝试评价未完成订单 - 应该失败
	_, err = reviewSvc.CreateReview(ctx, user.ID, &mallService.CreateReviewRequest{
		OrderID:   order.ID,
		ProductID: product.ID,
		Rating:    5,
	})
	assert.Error(t, err)
}

// ==================== 购物车与订单关联测试 ====================

func TestUS3Integration_CartToOrderConsistency(t *testing.T) {
	db := setupUS3IntegrationDB(t)
	productSvc, cartSvc, orderSvc, _ := setupUS3Services(db)
	ctx := context.Background()

	user, category, product1, _, address := seedUS3IntegrationData(t, db)

	// 创建更多商品
	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	product2 := &models.Product{CategoryID: category.ID, Name: "商品2", Images: images, Price: 50.0, Stock: 100, Unit: "件", IsOnSale: true}
	product3 := &models.Product{CategoryID: category.ID, Name: "商品3", Images: images, Price: 30.0, Stock: 100, Unit: "件", IsOnSale: true}
	db.Create(product2)
	db.Create(product3)

	// 添加多个商品到购物车
	cartSvc.AddItem(ctx, user.ID, &mallService.AddCartItemRequest{ProductID: product1.ID, Quantity: 2})
	cartSvc.AddItem(ctx, user.ID, &mallService.AddCartItemRequest{ProductID: product2.ID, Quantity: 3})
	cartSvc.AddItem(ctx, user.ID, &mallService.AddCartItemRequest{ProductID: product3.ID, Quantity: 1})

	// 取消选中商品3
	var cartItems []models.CartItem
	db.Where("user_id = ?", user.ID).Find(&cartItems)
	for _, item := range cartItems {
		if item.ProductID == product3.ID {
			db.Model(&item).Update("selected", false)
		}
	}

	// 从购物车创建订单
	order, err := orderSvc.CreateOrderFromCart(ctx, user.ID, &mallService.CreateFromCartRequest{
		AddressID: address.ID,
	})
	require.NoError(t, err)

	// 验证订单金额只包含选中商品
	expectedAmount := 80.0*2 + 50.0*3 // 商品1和商品2
	assert.Equal(t, expectedAmount, order.OriginalAmount)

	// 验证订单项数量
	assert.Len(t, order.Items, 2)

	// 验证未选中商品还在购物车中
	cart, err := cartSvc.GetCart(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, cart.Items, 1)
	assert.Equal(t, product3.ID, cart.Items[0].ProductID)

	// 验证库存扣减正确
	p1, _ := productSvc.GetProductDetail(ctx, product1.ID)
	p2, _ := productSvc.GetProductDetail(ctx, product2.ID)
	p3, _ := productSvc.GetProductDetail(ctx, product3.ID)
	assert.Equal(t, 48, p1.Stock)  // 50 - 2
	assert.Equal(t, 97, p2.Stock)  // 100 - 3
	assert.Equal(t, 100, p3.Stock) // 未购买，库存不变
}
