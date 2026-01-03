// Package rental 租借服务单元测试
package rental

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.RentalPricing{},
		&models.Order{},
		&models.Rental{},
		&models.WalletTransaction{},
	)
	require.NoError(t, err)

	// 创建默认会员等级
	level := &models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
	}
	db.Create(level)

	return db
}

// testRentalService 测试用租借服务
type testRentalService struct {
	*RentalService
	db *gorm.DB
}

// setupTestRentalService 创建测试用的 RentalService
func setupTestRentalService(t *testing.T) *testRentalService {
	db := setupTestDB(t)
	rentalRepo := repository.NewRentalRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	userRepo := repository.NewUserRepository(db)
	deviceSvc := deviceService.NewDeviceService(db, deviceRepo, venueRepo)
	walletSvc := userService.NewWalletService(db, userRepo)

	service := NewRentalService(db, rentalRepo, deviceRepo, deviceSvc, walletSvc, nil)

	return &testRentalService{
		RentalService: service,
		db:            db,
	}
}

// createTestData 创建测试数据
func createTestData(t *testing.T, db *gorm.DB) (user *models.User, device *models.Device, pricing *models.RentalPricing) {
	// 创建用户
	phone := "13800138000"
	user = &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 200.0,
	}
	err = db.Create(wallet).Error
	require.NoError(t, err)

	// 创建商户
	merchant := &models.Merchant{
		Name:           "测试商户",
		ContactName:    "联系人",
		ContactPhone:   "13900139000",
		CommissionRate:  0.2,
		SettlementType: "monthly",
		Status:         models.MerchantStatusActive,
	}
	err = db.Create(merchant).Error
	require.NoError(t, err)

	// 创建场地
	venue := &models.Venue{
		MerchantID: merchant.ID,
		Name:       "测试场地",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园路1号",
		Status:     models.VenueStatusActive,
	}
	err = db.Create(venue).Error
	require.NoError(t, err)

	// 创建设备
	device = &models.Device{
		DeviceNo:       "D20240101001",
		Name:           "测试设备",
		Type:           models.DeviceTypeStandard,
		VenueID:        venue.ID,
		QRCode:         "https://qr.example.com/D20240101001",
		ProductName:    "测试产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	err = db.Create(device).Error
	require.NoError(t, err)

	// 创建定价
	pricing = &models.RentalPricing{
		VenueID:       &venue.ID,
		DurationHours: 1,
		Price:         10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		IsActive:      true,
	}
	err = db.Create(pricing).Error
	require.NoError(t, err)

	return user, device, pricing
}

func TestRentalService_CreateRental(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("创建租借订单成功", func(t *testing.T) {
		req := &CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: pricing.ID,
		}

		rentalInfo, err := svc.CreateRental(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, rentalInfo)
		assert.NotEmpty(t, rentalInfo.OrderNo)
		assert.Equal(t, models.RentalStatusPending, rentalInfo.Status)
		assert.Equal(t, pricing.Price, rentalInfo.RentalFee)
		assert.Equal(t, pricing.Deposit, rentalInfo.Deposit)

		// 验证设备可用槽位减少
		var updatedDevice models.Device
		svc.db.First(&updatedDevice, device.ID)
		assert.Equal(t, 0, updatedDevice.AvailableSlots)
	})

	t.Run("余额不足创建失败", func(t *testing.T) {
		// 创建余额不足的用户
		poorPhone := "13800138001"
		poorUser := &models.User{
			Phone:         &poorPhone,
			Nickname:      "穷用户",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		svc.db.Create(poorUser)
		svc.db.Create(&models.UserWallet{UserID: poorUser.ID, Balance: 10.0})

		// 创建设备（避免被上一个用例预占槽位影响）
		device2 := &models.Device{
			DeviceNo:       "D20240101002",
			Name:           "测试设备2",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101002",
			ProductName:    "测试产品",
			SlotCount:      1,
			AvailableSlots: 1,
			OnlineStatus:   models.DeviceOnline,
			LockStatus:     models.DeviceLocked,
			RentalStatus:   models.DeviceRentalFree,
			NetworkType:    "WiFi",
			Status:         models.DeviceStatusActive,
		}
		svc.db.Create(device2)

		req := &CreateRentalRequest{
			DeviceID:  device2.ID,
			PricingID: pricing.ID,
		}

		_, err := svc.CreateRental(ctx, poorUser.ID, req)
		assert.Error(t, err)
	})
}

func TestRentalService_getStatusName(t *testing.T) {
	svc := setupTestRentalService(t)

	tests := []struct {
		status   string
		expected string
	}{
		{models.RentalStatusPending, "待支付"},
		{models.RentalStatusPaid, "待取货"},
		{models.RentalStatusInUse, "使用中"},
		{models.RentalStatusReturned, "已归还"},
		{models.RentalStatusCompleted, "已完成"},
		{models.RentalStatusCancelled, "已取消"},
		{"unknown", "未知"},
	}

	for _, tt := range tests {
		name := svc.getStatusName(tt.status)
		assert.Equal(t, tt.expected, name)
	}
}

func TestRentalService_PayStartReturnComplete_FullWalletFlow(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 1) 创建租借
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)

	// 2) 支付
	err = svc.PayRental(ctx, user.ID, rentalInfo.ID)
	require.NoError(t, err)

	// 验证订单状态与钱包变化
	var order models.Order
	err = svc.db.First(&order, rentalInfo.OrderID).Error
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusPaid, order.Status)
	assert.NotNil(t, order.PaidAt)

	var wallet models.UserWallet
	err = svc.db.Where("user_id = ?", user.ID).First(&wallet).Error
	require.NoError(t, err)
	assert.Equal(t, 200.0-(pricing.Price+pricing.Deposit), wallet.Balance)
	assert.Equal(t, pricing.Deposit, wallet.FrozenBalance)

	// 3) 开始租借
	err = svc.StartRental(ctx, user.ID, rentalInfo.ID)
	require.NoError(t, err)

	// 4) 归还
	err = svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
	require.NoError(t, err)

	// 5) 结算
	err = svc.CompleteRental(ctx, rentalInfo.ID)
	require.NoError(t, err)

	// 押金退还后，余额应为初始 - 租金
	err = svc.db.Where("user_id = ?", user.ID).First(&wallet).Error
	require.NoError(t, err)
	assert.Equal(t, 200.0-pricing.Price, wallet.Balance)
	assert.Equal(t, float64(0), wallet.FrozenBalance)
}
