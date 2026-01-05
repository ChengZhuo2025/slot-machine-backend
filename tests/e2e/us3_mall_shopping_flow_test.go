//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	mallHandler "github.com/dumeirei/smart-locker-backend/internal/handler/mall"
	userMiddleware "github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	mallService "github.com/dumeirei/smart-locker-backend/internal/service/mall"
)

type us3APIResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupUS3E2E(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	// 允许多个连接：订单创建使用事务 + 仓储层非 tx DB 调用，单连接会导致 SQLite 死锁
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)

	require.NoError(t, db.AutoMigrate(
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
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-us3-e2e",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	// 创建 repositories
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	skuRepo := repository.NewProductSkuRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	reviewRepo := repository.NewReviewRepository(db)

	// 创建 services
	productSvc := mallService.NewProductService(db, productRepo, categoryRepo, skuRepo)
	searchSvc := mallService.NewSearchService(db, productRepo)
	cartSvc := mallService.NewCartService(db, cartRepo, productRepo, skuRepo)
	orderSvc := mallService.NewMallOrderService(db, orderRepo, cartRepo, productRepo, skuRepo, productSvc)
	reviewSvc := mallService.NewReviewService(db, reviewRepo, orderRepo)

	// 创建 handlers
	productH := mallHandler.NewProductHandler(productSvc, searchSvc)
	cartH := mallHandler.NewCartHandler(cartSvc)
	orderH := mallHandler.NewOrderHandler(orderSvc)
	reviewH := mallHandler.NewReviewHandler(reviewSvc)

	v1 := engine.Group("/api/v1")
	{
		// 公开接口
		v1.GET("/categories", productH.GetCategories)
		v1.GET("/products", productH.GetProducts)
		v1.GET("/products/search", productH.SearchProducts)
		v1.GET("/products/:id", productH.GetProductDetail)
		v1.GET("/products/:id/reviews", reviewH.GetProductReviews)

		// 需要认证的接口
		user := v1.Group("")
		user.Use(userMiddleware.UserAuth(jwtManager))
		{
			// 购物车
			user.GET("/cart", cartH.GetCart)
			user.POST("/cart", cartH.AddItem)
			user.PUT("/cart/:id", cartH.UpdateItem)
			user.DELETE("/cart/:id", cartH.RemoveItem)
			user.DELETE("/cart", cartH.ClearCart)
			user.PUT("/cart/select-all", cartH.SelectAll)
			user.GET("/cart/count", cartH.GetCartCount)

			// 订单
			user.POST("/orders", orderH.CreateOrder)
			user.POST("/orders/from-cart", orderH.CreateOrderFromCart)
			user.GET("/orders", orderH.GetOrders)
			user.GET("/orders/:id", orderH.GetOrderDetail)
			user.POST("/orders/:id/cancel", orderH.CancelOrder)
			user.POST("/orders/:id/confirm", orderH.ConfirmReceive)

			// 评价
			user.POST("/reviews", reviewH.CreateReview)
			user.GET("/user/reviews", reviewH.GetUserReviews)
			user.DELETE("/reviews/:id", reviewH.DeleteReview)
		}
	}

	return engine, db, jwtManager
}

func seedUS3E2EData(t *testing.T, db *gorm.DB) (*models.User, []*models.Product, *models.Address) {
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
	require.NoError(t, db.Create(&models.UserWallet{UserID: user.ID, Balance: 5000.0}).Error)

	// 创建地址
	address := &models.Address{
		UserID:        user.ID,
		ReceiverName:  "张三",
		ReceiverPhone: "13900139000",
		Province:      "广东省",
		City:          "深圳市",
		District:      "南山区",
		Detail:        "科技园路1号A座501",
		IsDefault:     true,
	}
	require.NoError(t, db.Create(address).Error)

	// 创建分类
	category1 := &models.Category{Name: "电子产品", Level: 1, Sort: 1, IsActive: true}
	category2 := &models.Category{Name: "家居用品", Level: 1, Sort: 2, IsActive: true}
	require.NoError(t, db.Create(category1).Error)
	require.NoError(t, db.Create(category2).Error)

	// 创建商品
	var products []*models.Product
	productData := []struct {
		CategoryID int64
		Name       string
		Price      float64
		Stock      int
		IsHot      bool
		IsNew      bool
	}{
		{category1.ID, "无线蓝牙耳机", 199.0, 100, true, true},
		{category1.ID, "智能手表", 599.0, 50, true, false},
		{category2.ID, "台灯", 89.0, 200, false, true},
		{category2.ID, "收纳箱", 39.0, 500, false, false},
	}

	for _, pd := range productData {
		images, _ := json.Marshal([]string{"https://example.com/img1.jpg", "https://example.com/img2.jpg"})
		product := &models.Product{
			CategoryID: pd.CategoryID,
			Name:       pd.Name,
			Images:     images,
			Price:      pd.Price,
			Stock:      pd.Stock,
			Unit:       "件",
			IsOnSale:   true,
			IsHot:      pd.IsHot,
			IsNew:      pd.IsNew,
		}
		require.NoError(t, db.Create(product).Error)
		products = append(products, product)
	}

	// 为部分商品创建 SKU
	attrs1, _ := json.Marshal(map[string]string{"颜色": "黑色"})
	attrs2, _ := json.Marshal(map[string]string{"颜色": "白色"})
	db.Create(&models.ProductSku{ProductID: products[0].ID, SkuCode: "BT-BLACK", Attributes: attrs1, Price: 199.0, Stock: 50, IsActive: true})
	db.Create(&models.ProductSku{ProductID: products[0].ID, SkuCode: "BT-WHITE", Attributes: attrs2, Price: 199.0, Stock: 50, IsActive: true})

	return user, products, address
}

// ==================== 完整购物流程 E2E 测试 ====================

func TestUS3_E2E_FullShoppingFlow_DirectPurchase(t *testing.T) {
	router, db, jwtManager := setupUS3E2E(t)
	user, products, address := seedUS3E2EData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 1. 浏览分类
	catReq, _ := http.NewRequest("GET", "/api/v1/categories", nil)
	catW := httptest.NewRecorder()
	router.ServeHTTP(catW, catReq)
	require.Equal(t, http.StatusOK, catW.Code)

	// 2. 浏览商品列表
	listReq, _ := http.NewRequest("GET", "/api/v1/products?page=1&page_size=10", nil)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)

	var listResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(listW.Body.Bytes())).Decode(&listResp))
	require.Equal(t, 0, listResp.Code)

	// 3. 查看商品详情
	detailReq, _ := http.NewRequest("GET", "/api/v1/products/"+strconv.FormatInt(products[0].ID, 10), nil)
	detailW := httptest.NewRecorder()
	router.ServeHTTP(detailW, detailReq)
	require.Equal(t, http.StatusOK, detailW.Code)

	// 4. 直接购买 - 创建订单
	orderBody, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": products[0].ID, "quantity": 1},
			{"product_id": products[2].ID, "quantity": 2},
		},
		"address_id": address.ID,
		"remark":     "请尽快发货",
	})
	orderReq, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(orderBody))
	orderReq.Header.Set("Content-Type", "application/json")
	orderReq.Header.Set("Authorization", authz)
	orderW := httptest.NewRecorder()
	router.ServeHTTP(orderW, orderReq)
	require.Equal(t, http.StatusOK, orderW.Code)

	var orderResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(orderW.Body.Bytes())).Decode(&orderResp))
	require.Equal(t, 0, orderResp.Code)

	var orderData struct {
		ID             int64   `json:"id"`
		OrderNo        string  `json:"order_no"`
		Status         string  `json:"status"`
		OriginalAmount float64 `json:"original_amount"`
	}
	require.NoError(t, json.Unmarshal(orderResp.Data, &orderData))
	assert.NotEmpty(t, orderData.OrderNo)
	assert.Equal(t, models.OrderStatusPending, orderData.Status)
	assert.Equal(t, 377.0, orderData.OriginalAmount) // 199 + 89*2

	// 5. 验证库存扣减
	var product0, product2 models.Product
	db.First(&product0, products[0].ID)
	db.First(&product2, products[2].ID)
	assert.Equal(t, 99, product0.Stock)  // 100 - 1
	assert.Equal(t, 198, product2.Stock) // 200 - 2

	// 6. 模拟支付、发货
	db.Model(&models.Order{}).Where("id = ?", orderData.ID).Update("status", models.OrderStatusShipped)

	// 7. 确认收货
	confirmReq, _ := http.NewRequest("POST", "/api/v1/orders/"+strconv.FormatInt(orderData.ID, 10)+"/confirm", nil)
	confirmReq.Header.Set("Authorization", authz)
	confirmW := httptest.NewRecorder()
	router.ServeHTTP(confirmW, confirmReq)
	require.Equal(t, http.StatusOK, confirmW.Code)

	// 8. 验证订单状态
	getOrderReq, _ := http.NewRequest("GET", "/api/v1/orders/"+strconv.FormatInt(orderData.ID, 10), nil)
	getOrderReq.Header.Set("Authorization", authz)
	getOrderW := httptest.NewRecorder()
	router.ServeHTTP(getOrderW, getOrderReq)
	require.Equal(t, http.StatusOK, getOrderW.Code)

	var getOrderResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(getOrderW.Body.Bytes())).Decode(&getOrderResp))
	var finalOrder struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(getOrderResp.Data, &finalOrder))
	assert.Equal(t, models.OrderStatusCompleted, finalOrder.Status)

	// 9. 评价商品
	reviewBody, _ := json.Marshal(map[string]interface{}{
		"order_id":   orderData.ID,
		"product_id": products[0].ID,
		"rating":     5,
		"content":    "耳机音质很好，超值！",
	})
	reviewReq, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBuffer(reviewBody))
	reviewReq.Header.Set("Content-Type", "application/json")
	reviewReq.Header.Set("Authorization", authz)
	reviewW := httptest.NewRecorder()
	router.ServeHTTP(reviewW, reviewReq)
	require.Equal(t, http.StatusOK, reviewW.Code)

	// 10. 查看商品评价
	getReviewsReq, _ := http.NewRequest("GET", "/api/v1/products/"+strconv.FormatInt(products[0].ID, 10)+"/reviews", nil)
	getReviewsW := httptest.NewRecorder()
	router.ServeHTTP(getReviewsW, getReviewsReq)
	require.Equal(t, http.StatusOK, getReviewsW.Code)
}

func TestUS3_E2E_FullShoppingFlow_CartPurchase(t *testing.T) {
	router, db, jwtManager := setupUS3E2E(t)
	user, products, address := seedUS3E2EData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 1. 添加商品到购物车
	for i := 0; i < 3; i++ {
		addBody, _ := json.Marshal(map[string]interface{}{
			"product_id": products[i].ID,
			"quantity":   i + 1,
		})
		addReq, _ := http.NewRequest("POST", "/api/v1/cart", bytes.NewBuffer(addBody))
		addReq.Header.Set("Content-Type", "application/json")
		addReq.Header.Set("Authorization", authz)
		addW := httptest.NewRecorder()
		router.ServeHTTP(addW, addReq)
		require.Equal(t, http.StatusOK, addW.Code)
	}

	// 2. 查看购物车
	getCartReq, _ := http.NewRequest("GET", "/api/v1/cart", nil)
	getCartReq.Header.Set("Authorization", authz)
	getCartW := httptest.NewRecorder()
	router.ServeHTTP(getCartW, getCartReq)
	require.Equal(t, http.StatusOK, getCartW.Code)

	var cartResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(getCartW.Body.Bytes())).Decode(&cartResp))
	var cartData struct {
		Items       []interface{} `json:"items"`
		TotalCount  int           `json:"total_count"`
		TotalAmount float64       `json:"total_amount"`
	}
	require.NoError(t, json.Unmarshal(cartResp.Data, &cartData))
	assert.Len(t, cartData.Items, 3)
	assert.Equal(t, 6, cartData.TotalCount) // 1 + 2 + 3

	// 3. 修改购物车商品数量
	var cartItems []models.CartItem
	db.Where("user_id = ?", user.ID).Find(&cartItems)
	updateBody, _ := json.Marshal(map[string]interface{}{"quantity": 5})
	updateReq, _ := http.NewRequest("PUT", "/api/v1/cart/"+strconv.FormatInt(cartItems[0].ID, 10), bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", authz)
	updateW := httptest.NewRecorder()
	router.ServeHTTP(updateW, updateReq)
	require.Equal(t, http.StatusOK, updateW.Code)

	// 4. 取消选中一个商品
	unselectBody, _ := json.Marshal(map[string]interface{}{"quantity": 3, "selected": false})
	unselectReq, _ := http.NewRequest("PUT", "/api/v1/cart/"+strconv.FormatInt(cartItems[2].ID, 10), bytes.NewBuffer(unselectBody))
	unselectReq.Header.Set("Content-Type", "application/json")
	unselectReq.Header.Set("Authorization", authz)
	unselectW := httptest.NewRecorder()
	router.ServeHTTP(unselectW, unselectReq)
	require.Equal(t, http.StatusOK, unselectW.Code)

	// 5. 从购物车创建订单（只包含选中商品）
	orderBody, _ := json.Marshal(map[string]interface{}{
		"address_id": address.ID,
		"remark":     "购物车结算",
	})
	orderReq, _ := http.NewRequest("POST", "/api/v1/orders/from-cart", bytes.NewBuffer(orderBody))
	orderReq.Header.Set("Content-Type", "application/json")
	orderReq.Header.Set("Authorization", authz)
	orderW := httptest.NewRecorder()
	router.ServeHTTP(orderW, orderReq)
	require.Equal(t, http.StatusOK, orderW.Code)

	var orderResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(orderW.Body.Bytes())).Decode(&orderResp))
	var orderData struct {
		ID             int64   `json:"id"`
		OriginalAmount float64 `json:"original_amount"`
		Items          []struct {
			ProductName string  `json:"product_name"`
			Quantity    int     `json:"quantity"`
			Subtotal    float64 `json:"subtotal"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(orderResp.Data, &orderData))

	// 验证只有选中的商品被购买
	assert.Len(t, orderData.Items, 2) // 只有2个选中的商品

	// 6. 验证未选中商品还在购物车中
	getCartReq2, _ := http.NewRequest("GET", "/api/v1/cart", nil)
	getCartReq2.Header.Set("Authorization", authz)
	getCartW2 := httptest.NewRecorder()
	router.ServeHTTP(getCartW2, getCartReq2)
	require.Equal(t, http.StatusOK, getCartW2.Code)

	var cartResp2 us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(getCartW2.Body.Bytes())).Decode(&cartResp2))
	var cartData2 struct {
		Items []interface{} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(cartResp2.Data, &cartData2))
	assert.Len(t, cartData2.Items, 1) // 只剩未选中的商品
}

func TestUS3_E2E_OrderCancelAndStockRestore(t *testing.T) {
	router, db, jwtManager := setupUS3E2E(t)
	user, products, address := seedUS3E2EData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 记录原始库存
	var originalProduct models.Product
	db.First(&originalProduct, products[0].ID)
	originalStock := originalProduct.Stock

	// 1. 创建订单
	orderBody, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": products[0].ID, "quantity": 10},
		},
		"address_id": address.ID,
	})
	orderReq, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(orderBody))
	orderReq.Header.Set("Content-Type", "application/json")
	orderReq.Header.Set("Authorization", authz)
	orderW := httptest.NewRecorder()
	router.ServeHTTP(orderW, orderReq)
	require.Equal(t, http.StatusOK, orderW.Code)

	var orderResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(orderW.Body.Bytes())).Decode(&orderResp))
	var orderData struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(orderResp.Data, &orderData))

	// 2. 验证库存已扣减
	var afterOrder models.Product
	db.First(&afterOrder, products[0].ID)
	assert.Equal(t, originalStock-10, afterOrder.Stock)

	// 3. 取消订单
	cancelReq, _ := http.NewRequest("POST", "/api/v1/orders/"+strconv.FormatInt(orderData.ID, 10)+"/cancel?reason=不想要了", nil)
	cancelReq.Header.Set("Authorization", authz)
	cancelW := httptest.NewRecorder()
	router.ServeHTTP(cancelW, cancelReq)
	require.Equal(t, http.StatusOK, cancelW.Code)

	// 4. 验证库存已恢复
	var afterCancel models.Product
	db.First(&afterCancel, products[0].ID)
	assert.Equal(t, originalStock, afterCancel.Stock)

	// 5. 验证订单状态
	getOrderReq, _ := http.NewRequest("GET", "/api/v1/orders/"+strconv.FormatInt(orderData.ID, 10), nil)
	getOrderReq.Header.Set("Authorization", authz)
	getOrderW := httptest.NewRecorder()
	router.ServeHTTP(getOrderW, getOrderReq)
	require.Equal(t, http.StatusOK, getOrderW.Code)

	var getOrderResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(getOrderW.Body.Bytes())).Decode(&getOrderResp))
	var finalOrder struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(getOrderResp.Data, &finalOrder))
	assert.Equal(t, models.OrderStatusCancelled, finalOrder.Status)
}

func TestUS3_E2E_ReviewAfterOrderComplete(t *testing.T) {
	router, db, jwtManager := setupUS3E2E(t)
	user, products, address := seedUS3E2EData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 1. 创建订单
	orderBody, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": products[0].ID, "quantity": 1},
			{"product_id": products[1].ID, "quantity": 1},
		},
		"address_id": address.ID,
	})
	orderReq, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(orderBody))
	orderReq.Header.Set("Content-Type", "application/json")
	orderReq.Header.Set("Authorization", authz)
	orderW := httptest.NewRecorder()
	router.ServeHTTP(orderW, orderReq)
	require.Equal(t, http.StatusOK, orderW.Code)

	var orderResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(orderW.Body.Bytes())).Decode(&orderResp))
	var orderData struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(orderResp.Data, &orderData))

	// 2. 尝试在订单未完成时评价 - 应该失败
	reviewBody1, _ := json.Marshal(map[string]interface{}{
		"order_id":   orderData.ID,
		"product_id": products[0].ID,
		"rating":     5,
		"content":    "很好",
	})
	reviewReq1, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBuffer(reviewBody1))
	reviewReq1.Header.Set("Content-Type", "application/json")
	reviewReq1.Header.Set("Authorization", authz)
	reviewW1 := httptest.NewRecorder()
	router.ServeHTTP(reviewW1, reviewReq1)

	var reviewResp1 us3APIResp
	json.NewDecoder(bytes.NewReader(reviewW1.Body.Bytes())).Decode(&reviewResp1)
	assert.NotEqual(t, 0, reviewResp1.Code) // 应该返回错误

	// 3. 模拟订单完成
	db.Model(&models.Order{}).Where("id = ?", orderData.ID).Update("status", models.OrderStatusCompleted)

	// 4. 评价商品1
	reviewBody2, _ := json.Marshal(map[string]interface{}{
		"order_id":   orderData.ID,
		"product_id": products[0].ID,
		"rating":     5,
		"content":    "无线蓝牙耳机音质很好！",
		"images":     []string{"https://example.com/review1.jpg"},
	})
	reviewReq2, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBuffer(reviewBody2))
	reviewReq2.Header.Set("Content-Type", "application/json")
	reviewReq2.Header.Set("Authorization", authz)
	reviewW2 := httptest.NewRecorder()
	router.ServeHTTP(reviewW2, reviewReq2)
	require.Equal(t, http.StatusOK, reviewW2.Code)

	// 5. 评价商品2
	reviewBody3, _ := json.Marshal(map[string]interface{}{
		"order_id":     orderData.ID,
		"product_id":   products[1].ID,
		"rating":       4,
		"content":      "智能手表功能齐全，就是续航一般",
		"is_anonymous": true,
	})
	reviewReq3, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBuffer(reviewBody3))
	reviewReq3.Header.Set("Content-Type", "application/json")
	reviewReq3.Header.Set("Authorization", authz)
	reviewW3 := httptest.NewRecorder()
	router.ServeHTTP(reviewW3, reviewReq3)
	require.Equal(t, http.StatusOK, reviewW3.Code)

	// 6. 尝试重复评价 - 应该失败
	reviewBody4, _ := json.Marshal(map[string]interface{}{
		"order_id":   orderData.ID,
		"product_id": products[0].ID,
		"rating":     3,
		"content":    "再评一次",
	})
	reviewReq4, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBuffer(reviewBody4))
	reviewReq4.Header.Set("Content-Type", "application/json")
	reviewReq4.Header.Set("Authorization", authz)
	reviewW4 := httptest.NewRecorder()
	router.ServeHTTP(reviewW4, reviewReq4)

	var reviewResp4 us3APIResp
	json.NewDecoder(bytes.NewReader(reviewW4.Body.Bytes())).Decode(&reviewResp4)
	assert.NotEqual(t, 0, reviewResp4.Code) // 应该返回错误

	// 7. 查看商品评价
	getReviewsReq, _ := http.NewRequest("GET", "/api/v1/products/"+strconv.FormatInt(products[0].ID, 10)+"/reviews", nil)
	getReviewsW := httptest.NewRecorder()
	router.ServeHTTP(getReviewsW, getReviewsReq)
	require.Equal(t, http.StatusOK, getReviewsW.Code)

	var reviewsResp us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(getReviewsW.Body.Bytes())).Decode(&reviewsResp))
	var reviewsData struct {
		List  []interface{} `json:"list"`
		Total int64         `json:"total"`
	}
	require.NoError(t, json.Unmarshal(reviewsResp.Data, &reviewsData))
	assert.Len(t, reviewsData.List, 1)
	assert.Equal(t, int64(1), reviewsData.Total)
}

func TestUS3_E2E_MultipleUsersOrdering(t *testing.T) {
	router, db, jwtManager := setupUS3E2E(t)
	user1, products, address1 := seedUS3E2EData(t, db)

	// 创建第二个用户
	phone2 := "13900139001"
	user2 := &models.User{
		Phone:         &phone2,
		Nickname:      "用户2",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user2).Error)
	require.NoError(t, db.Create(&models.UserWallet{UserID: user2.ID, Balance: 5000.0}).Error)
	address2 := &models.Address{
		UserID:        user2.ID,
		ReceiverName:  "李四",
		ReceiverPhone: "13800138001",
		Province:      "北京市",
		City:          "北京市",
		District:      "朝阳区",
		Detail:        "建国路1号",
		IsDefault:     true,
	}
	require.NoError(t, db.Create(address2).Error)

	token1, _ := jwtManager.GenerateTokenPair(user1.ID, jwt.UserTypeUser, "")
	token2, _ := jwtManager.GenerateTokenPair(user2.ID, jwt.UserTypeUser, "")
	authz1 := "Bearer " + token1.AccessToken
	authz2 := "Bearer " + token2.AccessToken

	// 记录原始库存
	var originalProduct models.Product
	db.First(&originalProduct, products[0].ID)
	originalStock := originalProduct.Stock

	// 用户1购买商品
	orderBody1, _ := json.Marshal(map[string]interface{}{
		"items":      []map[string]interface{}{{"product_id": products[0].ID, "quantity": 30}},
		"address_id": address1.ID,
	})
	orderReq1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(orderBody1))
	orderReq1.Header.Set("Content-Type", "application/json")
	orderReq1.Header.Set("Authorization", authz1)
	orderW1 := httptest.NewRecorder()
	router.ServeHTTP(orderW1, orderReq1)
	require.Equal(t, http.StatusOK, orderW1.Code)

	// 用户2购买同一商品
	orderBody2, _ := json.Marshal(map[string]interface{}{
		"items":      []map[string]interface{}{{"product_id": products[0].ID, "quantity": 20}},
		"address_id": address2.ID,
	})
	orderReq2, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(orderBody2))
	orderReq2.Header.Set("Content-Type", "application/json")
	orderReq2.Header.Set("Authorization", authz2)
	orderW2 := httptest.NewRecorder()
	router.ServeHTTP(orderW2, orderReq2)
	require.Equal(t, http.StatusOK, orderW2.Code)

	// 验证库存扣减正确
	var afterOrders models.Product
	db.First(&afterOrders, products[0].ID)
	assert.Equal(t, originalStock-50, afterOrders.Stock) // 100 - 30 - 20 = 50

	// 用户1查看自己的订单
	getOrdersReq1, _ := http.NewRequest("GET", "/api/v1/orders", nil)
	getOrdersReq1.Header.Set("Authorization", authz1)
	getOrdersW1 := httptest.NewRecorder()
	router.ServeHTTP(getOrdersW1, getOrdersReq1)
	require.Equal(t, http.StatusOK, getOrdersW1.Code)

	var ordersResp1 us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(getOrdersW1.Body.Bytes())).Decode(&ordersResp1))
	var ordersData1 struct {
		List []interface{} `json:"list"`
	}
	require.NoError(t, json.Unmarshal(ordersResp1.Data, &ordersData1))
	assert.Len(t, ordersData1.List, 1) // 用户1只能看到自己的1个订单

	// 用户2查看自己的订单
	getOrdersReq2, _ := http.NewRequest("GET", "/api/v1/orders", nil)
	getOrdersReq2.Header.Set("Authorization", authz2)
	getOrdersW2 := httptest.NewRecorder()
	router.ServeHTTP(getOrdersW2, getOrdersReq2)
	require.Equal(t, http.StatusOK, getOrdersW2.Code)

	var ordersResp2 us3APIResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(getOrdersW2.Body.Bytes())).Decode(&ordersResp2))
	var ordersData2 struct {
		List []interface{} `json:"list"`
	}
	require.NoError(t, json.Unmarshal(ordersResp2.Data, &ordersData2))
	assert.Len(t, ordersData2.List, 1) // 用户2只能看到自己的1个订单
}
