// Package helpers 提供测试辅助工具
package helpers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomString 生成随机字符串
func RandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// RandomPhone 生成随机手机号
func RandomPhone() string {
	return fmt.Sprintf("138%08d", rand.Intn(100000000))
}

// RandomInt 生成随机整数
func RandomInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

// RandomFloat 生成随机浮点数
func RandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// NewTestUser 创建测试用户
func NewTestUser() *models.User {
	phone := RandomPhone()
	return &models.User{
		Phone:         &phone,
		Nickname:      "测试用户" + RandomString(4),
		Gender:        int8(RandomInt(0, 2)),
		MemberLevelID: 1,
		Points:        RandomInt(0, 1000),
		IsVerified:    false,
		Status:        models.UserStatusActive,
	}
}

// NewTestUserWithPhone 创建指定手机号的测试用户
func NewTestUserWithPhone(phone string) *models.User {
	return &models.User{
		Phone:         &phone,
		Nickname:      "测试用户" + phone[len(phone)-4:],
		Gender:        0,
		MemberLevelID: 1,
		Points:        0,
		IsVerified:    false,
		Status:        models.UserStatusActive,
	}
}

// NewTestUserWallet 创建测试用户钱包
func NewTestUserWallet(userID int64, balance float64) *models.UserWallet {
	return &models.UserWallet{
		UserID:         userID,
		Balance:        balance,
		FrozenBalance:  0,
		TotalRecharged: balance,
		TotalConsumed:  0,
		TotalWithdrawn: 0,
		Version:        0,
	}
}

// NewTestMemberLevel 创建测试会员等级
func NewTestMemberLevel(level int) *models.MemberLevel {
	return &models.MemberLevel{
		Name:      fmt.Sprintf("等级%d", level),
		Level:     level,
		MinPoints: level * 1000,
		Discount:  1.0 - float64(level)*0.05,
	}
}

// NewTestMerchant 创建测试商户
func NewTestMerchant() *models.Merchant {
	return &models.Merchant{
		Name:           "测试商户" + RandomString(4),
		ContactName:    "联系人" + RandomString(2),
		ContactPhone:   RandomPhone(),
		CommissionRate: 0.2,
		SettlementType: "monthly",
		Status:         models.MerchantStatusActive,
	}
}

// NewTestVenue 创建测试场地
func NewTestVenue(merchantID int64) *models.Venue {
	contactName := "场地联系人"
	contactPhone := RandomPhone()
	return &models.Venue{
		MerchantID:   merchantID,
		Name:         "测试场地" + RandomString(4),
		Type:         "mall",
		Province:     "广东省",
		City:         "深圳市",
		District:     "南山区",
		Address:      "科技园路" + RandomString(2) + "号",
		ContactName:  &contactName,
		ContactPhone: &contactPhone,
		Status:       models.VenueStatusActive,
	}
}

// NewTestDevice 创建测试设备
func NewTestDevice(venueID int64) *models.Device {
	deviceNo := fmt.Sprintf("D%s%04d", time.Now().Format("20060102"), rand.Intn(10000))
	return &models.Device{
		DeviceNo:       deviceNo,
		Name:           "测试设备" + RandomString(4),
		Type:           models.DeviceTypeStandard,
		VenueID:        venueID,
		QRCode:         fmt.Sprintf("https://qr.example.com/%s", deviceNo),
		ProductName:    "测试产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
}

// NewTestRentalPricing 创建测试租借定价
func NewTestRentalPricing(venueID *int64, durationHours int, price, deposit float64) *models.RentalPricing {
	return &models.RentalPricing{
		VenueID:       venueID,
		DurationHours: durationHours,
		Price:         price,
		Deposit:       deposit,
		OvertimeRate:  1.5,
		IsActive:      true,
	}
}

// NewTestRental 创建测试租借订单
func NewTestRental(orderID, userID, deviceID int64, durationHours int, status string) *models.Rental {
	return &models.Rental{
		OrderID:       orderID,
		UserID:        userID,
		DeviceID:      deviceID,
		DurationHours: durationHours,
		RentalFee:     10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		OvertimeFee:   0.0,
		Status:        status,
	}
}

// NewTestPayment 创建测试支付记录
func NewTestPayment(userID, orderID int64, orderNo string, amount float64, status int8) *models.Payment {
	paymentNo := fmt.Sprintf("P%s%06d", time.Now().Format("20060102150405"), rand.Intn(1000000))
	return &models.Payment{
		PaymentNo:      paymentNo,
		OrderID:        orderID,
		OrderNo:        orderNo,
		UserID:         userID,
		Amount:         amount,
		PaymentMethod:  models.PaymentMethodWechat,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         status,
	}
}

// ==================== 商城模块测试辅助函数 - US3 ====================

// NewTestCategory 创建测试分类
func NewTestCategory(parentID *int64, level int) *models.Category {
	return &models.Category{
		ParentID: parentID,
		Name:     "测试分类" + RandomString(4),
		Level:    int16(level),
		Sort:     rand.Intn(100),
		IsActive: true,
	}
}

// NewTestProduct 创建测试商品
func NewTestProduct(categoryID int64) *models.Product {
	images, _ := json.Marshal([]string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"})
	originalPrice := RandomFloat(50, 200)
	description := "测试商品描述"
	return &models.Product{
		CategoryID:    categoryID,
		Name:          "测试商品" + RandomString(4),
		Images:        images,
		Description:   &description,
		Price:         originalPrice * 0.8,
		OriginalPrice: &originalPrice,
		Stock:         RandomInt(10, 100),
		Sales:         RandomInt(0, 50),
		Unit:          "件",
		IsOnSale:      true,
		IsHot:         rand.Intn(2) == 1,
		IsNew:         rand.Intn(2) == 1,
		Sort:          rand.Intn(100),
	}
}

// NewTestProductWithStock 创建指定库存的测试商品
func NewTestProductWithStock(categoryID int64, stock int) *models.Product {
	product := NewTestProduct(categoryID)
	product.Stock = stock
	return product
}

// NewTestProductSku 创建测试 SKU
func NewTestProductSku(productID int64, color, size string) *models.ProductSku {
	attrs, _ := json.Marshal(map[string]string{"颜色": color, "尺码": size})
	return &models.ProductSku{
		ProductID:  productID,
		SkuCode:    fmt.Sprintf("SKU%d%s%s", productID, color, size),
		Attributes: attrs,
		Price:      RandomFloat(30, 150),
		Stock:      RandomInt(5, 50),
		IsActive:   true,
	}
}

// NewTestCartItem 创建测试购物车项
func NewTestCartItem(userID, productID int64, skuID *int64, quantity int) *models.CartItem {
	return &models.CartItem{
		UserID:    userID,
		ProductID: productID,
		SkuID:     skuID,
		Quantity:  quantity,
		Selected:  true,
	}
}

// NewTestMallOrder 创建测试商城订单
func NewTestMallOrder(userID int64, amount float64) *models.Order {
	orderNo := fmt.Sprintf("M%s%06d", time.Now().Format("20060102150405"), rand.Intn(1000000))
	return &models.Order{
		OrderNo:        orderNo,
		UserID:         userID,
		Type:           models.OrderTypeMall,
		OriginalAmount: amount,
		DiscountAmount: 0,
		ActualAmount:   amount,
		DepositAmount:  0,
		Status:         models.OrderStatusPending,
	}
}

// NewTestOrderItem 创建测试订单项
func NewTestOrderItem(orderID int64, productID int64, productName string, price float64, quantity int) *models.OrderItem {
	image := "https://example.com/product.jpg"
	return &models.OrderItem{
		OrderID:      orderID,
		ProductID:    &productID,
		ProductName:  productName,
		ProductImage: &image,
		Price:        price,
		Quantity:     quantity,
		Subtotal:     price * float64(quantity),
	}
}

// NewTestReview 创建测试评价
func NewTestReview(orderID, productID, userID int64, rating int16) *models.Review {
	content := "测试评价内容" + RandomString(10)
	images, _ := json.Marshal([]string{"https://example.com/review1.jpg"})
	return &models.Review{
		OrderID:     orderID,
		ProductID:   productID,
		UserID:      userID,
		Rating:      rating,
		Content:     &content,
		Images:      images,
		IsAnonymous: false,
		Status:      models.ReviewStatusVisible,
	}
}

// NewTestAddress 创建测试地址
func NewTestAddress(userID int64) *models.Address {
	return &models.Address{
		UserID:        userID,
		ReceiverName:  "测试收货人",
		ReceiverPhone: RandomPhone(),
		Province:      "广东省",
		City:          "深圳市",
		District:      "南山区",
		Detail:        "科技园路" + RandomString(2) + "号",
		IsDefault:     true,
	}
}
