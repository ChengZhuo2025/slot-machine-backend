//go:build api
// +build api

package api

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

func setupUS3APIRouter(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

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

	// 默认会员等级
	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-us3-api",
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

	v1 := r.Group("/api/v1")
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

	return r, db, jwtManager
}

func seedUS3TestData(t *testing.T, db *gorm.DB) (*models.User, *models.Category, *models.Product, *models.ProductSku, *models.Address) {
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

// ==================== 商品 API 测试 ====================

func TestUS3API_GetCategories(t *testing.T) {
	router, db, _ := setupUS3APIRouter(t)

	// 创建分类
	db.Create(&models.Category{Name: "分类1", Level: 1, Sort: 1, IsActive: true})
	db.Create(&models.Category{Name: "分类2", Level: 1, Sort: 2, IsActive: true})

	req, _ := http.NewRequest("GET", "/api/v1/categories", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])
}

func TestUS3API_GetProducts(t *testing.T) {
	router, db, _ := setupUS3APIRouter(t)
	_, category, _, _, _ := seedUS3TestData(t, db)

	// 创建多个商品
	for i := 0; i < 3; i++ {
		images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
		db.Create(&models.Product{
			CategoryID: category.ID,
			Name:       "商品" + string(rune('A'+i)),
			Images:     images,
			Price:      float64(50 + i*10),
			Stock:      100,
			Unit:       "件",
			IsOnSale:   true,
		})
	}

	req, _ := http.NewRequest("GET", "/api/v1/products?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	assert.GreaterOrEqual(t, len(list), 3)
}

func TestUS3API_GetProductDetail(t *testing.T) {
	router, db, _ := setupUS3APIRouter(t)
	_, _, product, _, _ := seedUS3TestData(t, db)

	req, _ := http.NewRequest("GET", "/api/v1/products/"+strconv.FormatInt(product.ID, 10), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "测试商品", data["name"])
}

func TestUS3API_GetProductDetail_NotFound(t *testing.T) {
	router, _, _ := setupUS3APIRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/products/99999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, float64(0), resp["code"])
}

func TestUS3API_SearchProducts(t *testing.T) {
	router, db, _ := setupUS3APIRouter(t)

	// 分类
	cat1 := &models.Category{Name: "电子产品", Level: 1, Sort: 1, IsActive: true}
	cat2 := &models.Category{Name: "配件", Level: 1, Sort: 2, IsActive: true}
	require.NoError(t, db.Create(cat1).Error)
	require.NoError(t, db.Create(cat2).Error)

	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})

	// 上架且匹配
	p1 := &models.Product{
		CategoryID: cat1.ID,
		Name:       "无线蓝牙耳机",
		Images:     images,
		Price:      199.0,
		Stock:      100,
		Sales:      10,
		Unit:       "件",
		IsOnSale:   true,
	}
	require.NoError(t, db.Create(p1).Error)

	// 下架但匹配（应被过滤掉）
	p2 := &models.Product{
		CategoryID: cat2.ID,
		Name:       "耳机保护套",
		Images:     images,
		Price:      20.0,
		Stock:      100,
		Sales:      100,
		Unit:       "件",
		IsOnSale:   false,
	}
	require.NoError(t, db.Create(p2).Error)
	// gorm + default(true) 场景下，Create 可能会忽略 bool 的零值；这里强制更新为下架
	require.NoError(t, db.Model(&models.Product{}).Where("id = ?", p2.ID).UpdateColumn("is_on_sale", false).Error)

	req, _ := http.NewRequest("GET", "/api/v1/products/search?keyword=耳机&page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	products := data["products"].([]interface{})
	require.Len(t, products, 1)

	first := products[0].(map[string]interface{})
	assert.Equal(t, "无线蓝牙耳机", first["name"])
}

func TestUS3API_SearchProducts_CategoryAndPriceRange(t *testing.T) {
	router, db, _ := setupUS3APIRouter(t)

	cat1 := &models.Category{Name: "智能家居", Level: 1, Sort: 1, IsActive: true}
	cat2 := &models.Category{Name: "穿戴设备", Level: 1, Sort: 2, IsActive: true}
	require.NoError(t, db.Create(cat1).Error)
	require.NoError(t, db.Create(cat2).Error)

	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})
	require.NoError(t, db.Create(&models.Product{
		CategoryID: cat1.ID,
		Name:       "智能插座",
		Images:     images,
		Price:      99.0,
		Stock:      100,
		Sales:      5,
		Unit:       "件",
		IsOnSale:   true,
	}).Error)
	require.NoError(t, db.Create(&models.Product{
		CategoryID: cat2.ID,
		Name:       "智能手表",
		Images:     images,
		Price:      599.0,
		Stock:      100,
		Sales:      50,
		Unit:       "件",
		IsOnSale:   true,
	}).Error)

	req, _ := http.NewRequest("GET", "/api/v1/products/search?keyword=智能&category_id="+strconv.FormatInt(cat2.ID, 10)+"&min_price=500&max_price=700&page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	products := data["products"].([]interface{})
	require.Len(t, products, 1)

	first := products[0].(map[string]interface{})
	assert.Equal(t, "智能手表", first["name"])
	assert.Equal(t, float64(599.0), first["price"])
}

func TestUS3API_SearchProducts_SortByPriceAndSales(t *testing.T) {
	router, db, _ := setupUS3APIRouter(t)

	cat := &models.Category{Name: "测试分类", Level: 1, Sort: 1, IsActive: true}
	require.NoError(t, db.Create(cat).Error)

	images, _ := json.Marshal([]string{"https://example.com/img.jpg"})

	require.NoError(t, db.Create(&models.Product{
		CategoryID: cat.ID,
		Name:       "测试商品A",
		Images:     images,
		Price:      30.0,
		Stock:      100,
		Sales:      5,
		Unit:       "件",
		IsOnSale:   true,
	}).Error)
	require.NoError(t, db.Create(&models.Product{
		CategoryID: cat.ID,
		Name:       "测试商品B",
		Images:     images,
		Price:      10.0,
		Stock:      100,
		Sales:      10,
		Unit:       "件",
		IsOnSale:   true,
	}).Error)
	require.NoError(t, db.Create(&models.Product{
		CategoryID: cat.ID,
		Name:       "测试商品C",
		Images:     images,
		Price:      20.0,
		Stock:      100,
		Sales:      50,
		Unit:       "件",
		IsOnSale:   true,
	}).Error)

	t.Run("price_asc", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/products/search?keyword=测试&sort_by=price_asc&page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		products := data["products"].([]interface{})
		require.Len(t, products, 3)
		first := products[0].(map[string]interface{})
		assert.Equal(t, float64(10.0), first["price"])
	})

	t.Run("sales_desc", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/products/search?keyword=测试&sort_by=sales_desc&page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].(map[string]interface{})
		products := data["products"].([]interface{})
		require.Len(t, products, 3)
		first := products[0].(map[string]interface{})
		assert.Equal(t, float64(50), first["sales"])
	})
}

func TestUS3API_SearchProducts_KeywordRequired(t *testing.T) {
	router, _, _ := setupUS3APIRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/products/search?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(400), resp["code"])
}

// ==================== 购物车 API 测试 ====================

func TestUS3API_Cart_AddItem(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, _ := seedUS3TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	body, _ := json.Marshal(map[string]interface{}{
		"product_id": product.ID,
		"quantity":   2,
	})
	req, _ := http.NewRequest("POST", "/api/v1/cart", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["quantity"])
}

func TestUS3API_Cart_GetCart(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, _ := seedUS3TestData(t, db)

	// 添加购物车项
	db.Create(&models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  3,
		Selected:  true,
	})

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	req, _ := http.NewRequest("GET", "/api/v1/cart", nil)
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	assert.Len(t, items, 1)
}

func TestUS3API_Cart_UpdateItem(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, _ := seedUS3TestData(t, db)

	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	body, _ := json.Marshal(map[string]interface{}{
		"quantity": 5,
	})
	req, _ := http.NewRequest("PUT", "/api/v1/cart/"+strconv.FormatInt(cartItem.ID, 10), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(5), data["quantity"])
}

func TestUS3API_Cart_RemoveItem(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, _ := seedUS3TestData(t, db)

	cartItem := &models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	}
	require.NoError(t, db.Create(cartItem).Error)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	req, _ := http.NewRequest("DELETE", "/api/v1/cart/"+strconv.FormatInt(cartItem.ID, 10), nil)
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证已删除
	var count int64
	db.Model(&models.CartItem{}).Where("id = ?", cartItem.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestUS3API_Cart_Unauthorized(t *testing.T) {
	router, _, _ := setupUS3APIRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/cart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ==================== 订单 API 测试 ====================

func TestUS3API_Order_Create(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, address := seedUS3TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	body, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": product.ID, "quantity": 2},
		},
		"address_id": address.ID,
		"remark":     "测试订单",
	})
	req, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["order_no"])
	assert.Equal(t, models.OrderStatusPending, data["status"])
}

func TestUS3API_Order_CreateFromCart(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, address := seedUS3TestData(t, db)

	// 添加购物车项
	db.Create(&models.CartItem{
		UserID:    user.ID,
		ProductID: product.ID,
		Quantity:  2,
		Selected:  true,
	})

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	body, _ := json.Marshal(map[string]interface{}{
		"address_id": address.ID,
	})
	req, _ := http.NewRequest("POST", "/api/v1/orders/from-cart", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	// 验证购物车已清空
	var count int64
	db.Model(&models.CartItem{}).Where("user_id = ? AND selected = ?", user.ID, true).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestUS3API_Order_GetList(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, address := seedUS3TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 创建订单
	body, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": product.ID, "quantity": 1},
		},
		"address_id": address.ID,
	})
	for i := 0; i < 3; i++ {
		createReq, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", authz)
		createW := httptest.NewRecorder()
		router.ServeHTTP(createW, createReq)
		require.Equal(t, http.StatusOK, createW.Code)
	}

	// 获取订单列表
	req, _ := http.NewRequest("GET", "/api/v1/orders", nil)
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	assert.Len(t, list, 3)
}

func TestUS3API_Order_GetDetail(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, address := seedUS3TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 创建订单
	body, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": product.ID, "quantity": 2},
		},
		"address_id": address.ID,
	})
	createReq, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", authz)
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusOK, createW.Code)

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &createResp))
	orderData := createResp["data"].(map[string]interface{})
	orderID := int64(orderData["id"].(float64))

	// 获取订单详情
	req, _ := http.NewRequest("GET", "/api/v1/orders/"+strconv.FormatInt(orderID, 10), nil)
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, orderData["order_no"], data["order_no"])
}

func TestUS3API_Order_Cancel(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, address := seedUS3TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 创建订单
	body, _ := json.Marshal(map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": product.ID, "quantity": 2},
		},
		"address_id": address.ID,
	})
	createReq, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", authz)
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusOK, createW.Code)

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &createResp))
	orderData := createResp["data"].(map[string]interface{})
	orderID := int64(orderData["id"].(float64))

	// 取消订单
	req, _ := http.NewRequest("POST", "/api/v1/orders/"+strconv.FormatInt(orderID, 10)+"/cancel?reason=不想要了", nil)
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证订单状态
	var order models.Order
	require.NoError(t, db.First(&order, orderID).Error)
	assert.Equal(t, models.OrderStatusCancelled, order.Status)
}

// ==================== 评价 API 测试 ====================

func TestUS3API_Review_Create(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, _ := seedUS3TestData(t, db)

	// 创建已完成订单
	order := &models.Order{
		OrderNo:        "M20240101001",
		UserID:         user.ID,
		Type:           models.OrderTypeMall,
		OriginalAmount: 80.0,
		ActualAmount:   80.0,
		Status:         models.OrderStatusCompleted,
	}
	require.NoError(t, db.Create(order).Error)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	body, _ := json.Marshal(map[string]interface{}{
		"order_id":     order.ID,
		"product_id":   product.ID,
		"rating":       5,
		"content":      "非常满意！",
		"is_anonymous": false,
	})
	req, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(5), data["rating"])
}

func TestUS3API_Review_GetProductReviews(t *testing.T) {
	router, db, _ := setupUS3APIRouter(t)
	_, _, product, _, _ := seedUS3TestData(t, db)

	// 创建评价
	for i := 0; i < 3; i++ {
		order := &models.Order{
			OrderNo:        "M2024010100" + strconv.Itoa(i),
			UserID:         1,
			Type:           models.OrderTypeMall,
			OriginalAmount: 80.0,
			ActualAmount:   80.0,
			Status:         models.OrderStatusCompleted,
		}
		db.Create(order)

		content := "评价内容" + strconv.Itoa(i)
		review := &models.Review{
			OrderID:   order.ID,
			ProductID: product.ID,
			UserID:    1,
			Rating:    int16(4 + i%2),
			Content:   &content,
			Status:    int16(models.ReviewStatusVisible),
		}
		db.Create(review)
	}

	req, _ := http.NewRequest("GET", "/api/v1/products/"+strconv.FormatInt(product.ID, 10)+"/reviews?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	assert.Len(t, list, 3)
}

func TestUS3API_Review_Delete(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, _ := seedUS3TestData(t, db)

	// 创建订单和评价
	order := &models.Order{
		OrderNo:        "M20240101001",
		UserID:         user.ID,
		Type:           models.OrderTypeMall,
		OriginalAmount: 80.0,
		ActualAmount:   80.0,
		Status:         models.OrderStatusCompleted,
	}
	require.NoError(t, db.Create(order).Error)

	content := "测试评价"
	review := &models.Review{
		OrderID:   order.ID,
		ProductID: product.ID,
		UserID:    user.ID,
		Rating:    5,
		Content:   &content,
		Status:    int16(models.ReviewStatusVisible),
	}
	require.NoError(t, db.Create(review).Error)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	req, _ := http.NewRequest("DELETE", "/api/v1/reviews/"+strconv.FormatInt(review.ID, 10), nil)
	req.Header.Set("Authorization", authz)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证已删除
	var count int64
	db.Model(&models.Review{}).Where("id = ?", review.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// ==================== 完整购物流程测试 ====================

func TestUS3API_FullShoppingFlow(t *testing.T) {
	router, db, jwtManager := setupUS3APIRouter(t)
	user, _, product, _, address := seedUS3TestData(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 1. 浏览商品
	browseReq, _ := http.NewRequest("GET", "/api/v1/products/"+strconv.FormatInt(product.ID, 10), nil)
	browseW := httptest.NewRecorder()
	router.ServeHTTP(browseW, browseReq)
	require.Equal(t, http.StatusOK, browseW.Code)

	// 2. 添加到购物车
	addCartBody, _ := json.Marshal(map[string]interface{}{
		"product_id": product.ID,
		"quantity":   2,
	})
	addCartReq, _ := http.NewRequest("POST", "/api/v1/cart", bytes.NewBuffer(addCartBody))
	addCartReq.Header.Set("Content-Type", "application/json")
	addCartReq.Header.Set("Authorization", authz)
	addCartW := httptest.NewRecorder()
	router.ServeHTTP(addCartW, addCartReq)
	require.Equal(t, http.StatusOK, addCartW.Code)

	// 3. 查看购物车
	getCartReq, _ := http.NewRequest("GET", "/api/v1/cart", nil)
	getCartReq.Header.Set("Authorization", authz)
	getCartW := httptest.NewRecorder()
	router.ServeHTTP(getCartW, getCartReq)
	require.Equal(t, http.StatusOK, getCartW.Code)

	// 4. 从购物车创建订单
	createOrderBody, _ := json.Marshal(map[string]interface{}{
		"address_id": address.ID,
		"remark":     "请尽快发货",
	})
	createOrderReq, _ := http.NewRequest("POST", "/api/v1/orders/from-cart", bytes.NewBuffer(createOrderBody))
	createOrderReq.Header.Set("Content-Type", "application/json")
	createOrderReq.Header.Set("Authorization", authz)
	createOrderW := httptest.NewRecorder()
	router.ServeHTTP(createOrderW, createOrderReq)
	require.Equal(t, http.StatusOK, createOrderW.Code)

	var createOrderResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createOrderW.Body.Bytes(), &createOrderResp))
	orderData := createOrderResp["data"].(map[string]interface{})
	orderID := int64(orderData["id"].(float64))

	// 5. 模拟支付、发货、收货
	db.Model(&models.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"status": models.OrderStatusCompleted,
	})

	// 6. 评价商品
	reviewBody, _ := json.Marshal(map[string]interface{}{
		"order_id":     orderID,
		"product_id":   product.ID,
		"rating":       5,
		"content":      "商品质量很好，发货速度快！",
		"is_anonymous": false,
	})
	reviewReq, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBuffer(reviewBody))
	reviewReq.Header.Set("Content-Type", "application/json")
	reviewReq.Header.Set("Authorization", authz)
	reviewW := httptest.NewRecorder()
	router.ServeHTTP(reviewW, reviewReq)
	require.Equal(t, http.StatusOK, reviewW.Code)

	// 验证评价已创建
	var reviewCount int64
	db.Model(&models.Review{}).Where("order_id = ? AND product_id = ?", orderID, product.ID).Count(&reviewCount)
	assert.Equal(t, int64(1), reviewCount)
}
