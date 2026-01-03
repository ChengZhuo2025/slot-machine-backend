// Package helpers 提供测试辅助工具
package helpers

import (
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
func NewTestRentalPricing(deviceID int64, duration int, price, deposit float64) *models.RentalPricing {
	return &models.RentalPricing{
		DeviceID:     deviceID,
		Name:         fmt.Sprintf("%d小时租借", duration),
		Duration:     duration,
		DurationUnit: models.DurationUnitHour,
		Price:        price,
		Deposit:      deposit,
		Status:       models.RentalPricingStatusActive,
	}
}

// NewTestRental 创建测试租借订单
func NewTestRental(userID, deviceID, pricingID int64, status int8) *models.Rental {
	rentalNo := fmt.Sprintf("R%s%06d", time.Now().Format("20060102150405"), rand.Intn(1000000))
	slotNo := 1
	return &models.Rental{
		RentalNo:      rentalNo,
		UserID:        userID,
		DeviceID:      deviceID,
		PricingID:     pricingID,
		SlotNo:        &slotNo,
		Status:        status,
		UnitPrice:     10.0,
		DepositAmount: 50.0,
		RentalAmount:  10.0,
		ActualAmount:  60.0,
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
