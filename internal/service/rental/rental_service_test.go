// Package rental 租借服务单元测试
package rental

import (
	"context"
	"testing"
	"time"

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
		{models.RentalStatusRefunding, "退款中"},
		{models.RentalStatusRefunded, "已退款"},
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

func TestRentalService_PayRental_Errors(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("租借不存在", func(t *testing.T) {
		err := svc.PayRental(ctx, user.ID, 999999)
		assert.Error(t, err)
	})

	t.Run("无权限支付他人订单", func(t *testing.T) {
		// 创建租借
		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)

		// 另一个用户尝试支付
		err = svc.PayRental(ctx, 999999, rentalInfo.ID)
		assert.Error(t, err)

		// 完成当前租借，以便后续测试
		svc.PayRental(ctx, user.ID, rentalInfo.ID)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)
		svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
		svc.CompleteRental(ctx, rentalInfo.ID)
	})

	t.Run("状态错误不能支付", func(t *testing.T) {
		// 创建新设备避免槽位冲突
		device2 := &models.Device{
			DeviceNo:       "D20240101003",
			Name:           "测试设备3",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101003",
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

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device2.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)

		// 先支付
		err = svc.PayRental(ctx, user.ID, rentalInfo.ID)
		require.NoError(t, err)

		// 再次支付应失败
		err = svc.PayRental(ctx, user.ID, rentalInfo.ID)
		assert.Error(t, err)
	})
}

func TestRentalService_StartRental_Errors(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("租借不存在", func(t *testing.T) {
		err := svc.StartRental(ctx, user.ID, 999999)
		assert.Error(t, err)
	})

	t.Run("无权限开始他人租借", func(t *testing.T) {
		device2 := &models.Device{
			DeviceNo:       "D20240101004",
			Name:           "测试设备4",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101004",
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

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device2.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)

		err = svc.StartRental(ctx, 999999, rentalInfo.ID)
		assert.Error(t, err)

		// 完成当前租借
		svc.StartRental(ctx, user.ID, rentalInfo.ID)
		svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
		svc.CompleteRental(ctx, rentalInfo.ID)
	})

	t.Run("未支付状态不能开始", func(t *testing.T) {
		device3 := &models.Device{
			DeviceNo:       "D20240101005",
			Name:           "测试设备5",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101005",
			ProductName:    "测试产品",
			SlotCount:      1,
			AvailableSlots: 1,
			OnlineStatus:   models.DeviceOnline,
			LockStatus:     models.DeviceLocked,
			RentalStatus:   models.DeviceRentalFree,
			NetworkType:    "WiFi",
			Status:         models.DeviceStatusActive,
		}
		svc.db.Create(device3)

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device3.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)

		err = svc.StartRental(ctx, user.ID, rentalInfo.ID)
		assert.Error(t, err)

		// 取消当前租借
		svc.CancelRental(ctx, user.ID, rentalInfo.ID)
	})

	t.Run("设备已禁用不能开始", func(t *testing.T) {
		device4 := &models.Device{
			DeviceNo:       "D20240101006",
			Name:           "测试设备6",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101006",
			ProductName:    "测试产品",
			SlotCount:      1,
			AvailableSlots: 1,
			OnlineStatus:   models.DeviceOnline,
			LockStatus:     models.DeviceLocked,
			RentalStatus:   models.DeviceRentalFree,
			NetworkType:    "WiFi",
			Status:         models.DeviceStatusActive,
		}
		svc.db.Create(device4)

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device4.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)

		// 禁用设备
		svc.db.Model(&models.Device{}).Where("id = ?", device4.ID).Update("status", models.DeviceStatusDisabled)

		err = svc.StartRental(ctx, user.ID, rentalInfo.ID)
		assert.Error(t, err)

		// 恢复设备状态并完成租借
		svc.db.Model(&models.Device{}).Where("id = ?", device4.ID).Update("status", models.DeviceStatusActive)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)
		svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
		svc.CompleteRental(ctx, rentalInfo.ID)
	})

	t.Run("设备离线不能开始", func(t *testing.T) {
		device5 := &models.Device{
			DeviceNo:       "D20240101007",
			Name:           "测试设备7",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101007",
			ProductName:    "测试产品",
			SlotCount:      1,
			AvailableSlots: 1,
			OnlineStatus:   models.DeviceOnline,
			LockStatus:     models.DeviceLocked,
			RentalStatus:   models.DeviceRentalFree,
			NetworkType:    "WiFi",
			Status:         models.DeviceStatusActive,
		}
		svc.db.Create(device5)

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device5.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)

		// 设备离线
		svc.db.Model(&models.Device{}).Where("id = ?", device5.ID).Update("online_status", models.DeviceOffline)

		err = svc.StartRental(ctx, user.ID, rentalInfo.ID)
		assert.Error(t, err)
	})
}

func TestRentalService_ReturnRental_Errors(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("租借不存在", func(t *testing.T) {
		err := svc.ReturnRental(ctx, user.ID, 999999)
		assert.Error(t, err)
	})

	t.Run("无权限归还他人租借", func(t *testing.T) {
		device2 := &models.Device{
			DeviceNo:       "D20240101008",
			Name:           "测试设备8",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101008",
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

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device2.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)

		err = svc.ReturnRental(ctx, 999999, rentalInfo.ID)
		assert.Error(t, err)

		// 完成当前租借
		svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
		svc.CompleteRental(ctx, rentalInfo.ID)
	})

	t.Run("非使用中状态不能归还", func(t *testing.T) {
		device3 := &models.Device{
			DeviceNo:       "D20240101009",
			Name:           "测试设备9",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101009",
			ProductName:    "测试产品",
			SlotCount:      1,
			AvailableSlots: 1,
			OnlineStatus:   models.DeviceOnline,
			LockStatus:     models.DeviceLocked,
			RentalStatus:   models.DeviceRentalFree,
			NetworkType:    "WiFi",
			Status:         models.DeviceStatusActive,
		}
		svc.db.Create(device3)

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device3.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)

		// 未开始就尝试归还
		err = svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
		assert.Error(t, err)
	})
}

func TestRentalService_CompleteRental_Errors(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("租借不存在", func(t *testing.T) {
		err := svc.CompleteRental(ctx, 999999)
		assert.Error(t, err)
	})

	t.Run("非已归还状态不能完成", func(t *testing.T) {
		device2 := &models.Device{
			DeviceNo:       "D20240101010",
			Name:           "测试设备10",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101010",
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

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device2.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)

		// 未归还就尝试完成
		err = svc.CompleteRental(ctx, rentalInfo.ID)
		assert.Error(t, err)
	})
}

func TestRentalService_CreateRental_MoreErrors(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("定价不存在", func(t *testing.T) {
		_, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: 999999,
		})
		assert.Error(t, err)
	})

	t.Run("已有进行中租借", func(t *testing.T) {
		// 先创建一个租借
		_, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)

		// 再创建另一个设备
		device2 := &models.Device{
			DeviceNo:       "D20240101030",
			Name:           "测试设备30",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101030",
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

		// 尝试创建第二个租借
		_, err = svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device2.ID,
			PricingID: pricing.ID,
		})
		assert.Error(t, err)
	})

	t.Run("定价已停用", func(t *testing.T) {
		// 创建另一个用户
		phone2 := "13800138099"
		user2 := &models.User{
			Phone:         &phone2,
			Nickname:      "测试用户2",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		svc.db.Create(user2)
		svc.db.Create(&models.UserWallet{UserID: user2.ID, Balance: 200.0})

		device3 := &models.Device{
			DeviceNo:       "D20240101031",
			Name:           "测试设备31",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101031",
			ProductName:    "测试产品",
			SlotCount:      1,
			AvailableSlots: 1,
			OnlineStatus:   models.DeviceOnline,
			LockStatus:     models.DeviceLocked,
			RentalStatus:   models.DeviceRentalFree,
			NetworkType:    "WiFi",
			Status:         models.DeviceStatusActive,
		}
		svc.db.Create(device3)

		// 创建停用的定价（先创建再更新，避免 default:true 覆盖）
		disabledPricing := &models.RentalPricing{
			VenueID:       &device.VenueID,
			DurationHours: 2,
			Price:         20.0,
			Deposit:       100.0,
			OvertimeRate:  2.0,
			IsActive:      true,
		}
		svc.db.Create(disabledPricing)
		svc.db.Model(disabledPricing).Update("is_active", false)

		_, err := svc.CreateRental(ctx, user2.ID, &CreateRentalRequest{
			DeviceID:  device3.ID,
			PricingID: disabledPricing.ID,
		})
		assert.Error(t, err)
	})
}

func TestRentalService_ReturnRental_OvertimeFee(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("无超时费用", func(t *testing.T) {
		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)

		err = svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
		require.NoError(t, err)

		// 验证无超时费
		var rental models.Rental
		svc.db.First(&rental, rentalInfo.ID)
		assert.Equal(t, float64(0), rental.OvertimeFee)
		assert.Equal(t, models.RentalStatusReturned, rental.Status)
	})
}

func TestRentalService_GetRental_DBError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 关闭数据库连接模拟错误
	sqlDB, _ := svc.db.DB()
	sqlDB.Close()

	_, err := svc.GetRental(ctx, 1, 1)
	assert.Error(t, err)
}

func TestRentalService_ListRentals_DBError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 关闭数据库连接模拟错误
	sqlDB, _ := svc.db.DB()
	sqlDB.Close()

	_, _, err := svc.ListRentals(ctx, 1, 1, 10, nil)
	assert.Error(t, err)
}

func TestRentalService_CreateRental_DBError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 关闭数据库连接模拟错误
	sqlDB, _ := svc.db.DB()
	sqlDB.Close()

	_, err := svc.CreateRental(ctx, 1, &CreateRentalRequest{
		DeviceID:  1,
		PricingID: 1,
	})
	assert.Error(t, err)
}

func TestRentalService_PayRental_DBError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 关闭数据库连接模拟错误
	sqlDB, _ := svc.db.DB()
	sqlDB.Close()

	err := svc.PayRental(ctx, 1, 1)
	assert.Error(t, err)
}

func TestRentalService_StartRental_DBError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 关闭数据库连接模拟错误
	sqlDB, _ := svc.db.DB()
	sqlDB.Close()

	err := svc.StartRental(ctx, 1, 1)
	assert.Error(t, err)
}

func TestRentalService_ReturnRental_DBError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 关闭数据库连接模拟错误
	sqlDB, _ := svc.db.DB()
	sqlDB.Close()

	err := svc.ReturnRental(ctx, 1, 1)
	assert.Error(t, err)
}

func TestRentalService_CompleteRental_DBError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 关闭数据库连接模拟错误
	sqlDB, _ := svc.db.DB()
	sqlDB.Close()

	err := svc.CompleteRental(ctx, 1)
	assert.Error(t, err)
}

func TestRentalService_CancelRental_DBError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 关闭数据库连接模拟错误
	sqlDB, _ := svc.db.DB()
	sqlDB.Close()

	err := svc.CancelRental(ctx, 1, 1)
	assert.Error(t, err)
}

func TestRentalService_CreateRental_DeviceCheckFail(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, _, pricing := createTestData(t, svc.db)

	t.Run("设备不存在", func(t *testing.T) {
		_, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  999999, // 不存在的设备
			PricingID: pricing.ID,
		})
		assert.Error(t, err)
	})
}

func TestRentalService_CreateRental_NoSlot(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 将设备槽位设为0
	svc.db.Model(&models.Device{}).Where("id = ?", device.ID).Update("available_slots", 0)

	_, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	assert.Error(t, err)
}

func TestRentalService_CancelRental_StatusError(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 创建租借并支付
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)

	// 支付后尝试取消
	svc.PayRental(ctx, user.ID, rentalInfo.ID)
	err = svc.CancelRental(ctx, user.ID, rentalInfo.ID)
	assert.Error(t, err)
}

func TestRentalService_CancelRental_NotFound(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, _, _ := createTestData(t, svc.db)

	err := svc.CancelRental(ctx, user.ID, 999999)
	assert.Error(t, err)
}

func TestRentalService_CompleteRental_OrderNotFound(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 创建完整流程的租借
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)

	svc.PayRental(ctx, user.ID, rentalInfo.ID)
	svc.StartRental(ctx, user.ID, rentalInfo.ID)
	svc.ReturnRental(ctx, user.ID, rentalInfo.ID)

	// 删除订单
	svc.db.Delete(&models.Order{}, "id = ?", rentalInfo.OrderID)

	// 尝试完成
	err = svc.CompleteRental(ctx, rentalInfo.ID)
	assert.Error(t, err)
}

func TestRentalService_ReturnRental_WithOvertimeFee(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("超时费不超过押金", func(t *testing.T) {
		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)

		// 手动设置过期时间为过去（模拟超时2小时）
		pastTime := time.Now().Add(-2 * time.Hour)
		svc.db.Model(&models.Rental{}).Where("id = ?", rentalInfo.ID).Update("expected_return_at", pastTime)

		err = svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
		require.NoError(t, err)

		// 验证超时费计算 (超时3小时 * 1.5 = 4.5)
		var rental models.Rental
		svc.db.First(&rental, rentalInfo.ID)
		assert.Greater(t, rental.OvertimeFee, float64(0))
		assert.LessOrEqual(t, rental.OvertimeFee, rental.Deposit)
	})

	t.Run("超时费超过押金时限制为押金", func(t *testing.T) {
		device2 := &models.Device{
			DeviceNo:       "D20240101040",
			Name:           "测试设备40",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101040",
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

		// 创建高超时费定价
		highOvertimePricing := &models.RentalPricing{
			VenueID:       &device.VenueID,
			DurationHours: 1,
			Price:         10.0,
			Deposit:       20.0,    // 低押金
			OvertimeRate:  100.0,   // 高超时费率
			IsActive:      true,
		}
		svc.db.Create(highOvertimePricing)

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device2.ID,
			PricingID: highOvertimePricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)

		// 手动设置过期时间为过去（模拟超时10小时）
		pastTime := time.Now().Add(-10 * time.Hour)
		svc.db.Model(&models.Rental{}).Where("id = ?", rentalInfo.ID).Update("expected_return_at", pastTime)

		err = svc.ReturnRental(ctx, user.ID, rentalInfo.ID)
		require.NoError(t, err)

		// 验证超时费不超过押金
		var rental models.Rental
		svc.db.First(&rental, rentalInfo.ID)
		assert.Equal(t, rental.Deposit, rental.OvertimeFee) // 超时费被限制为押金
	})
}

func TestRentalService_CompleteRental_WithOvertimeFee(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("有超时费时押金扣除", func(t *testing.T) {
		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)

		// 手动设置超时费
		svc.db.Model(&models.Rental{}).Where("id = ?", rentalInfo.ID).Updates(map[string]interface{}{
			"status":       models.RentalStatusReturned,
			"overtime_fee": 10.0,
			"returned_at":  time.Now(),
		})

		// 记录当前钱包状态
		var walletBefore models.UserWallet
		svc.db.Where("user_id = ?", user.ID).First(&walletBefore)

		err = svc.CompleteRental(ctx, rentalInfo.ID)
		require.NoError(t, err)

		// 验证钱包变化
		var walletAfter models.UserWallet
		svc.db.Where("user_id = ?", user.ID).First(&walletAfter)

		// 冻结余额应该减少（押金部分退还）
		assert.Less(t, walletAfter.FrozenBalance, walletBefore.FrozenBalance)
	})

	t.Run("无超时费时全额退还押金", func(t *testing.T) {
		device2 := &models.Device{
			DeviceNo:       "D20240101041",
			Name:           "测试设备41",
			Type:           models.DeviceTypeStandard,
			VenueID:        device.VenueID,
			QRCode:         "https://qr.example.com/D20240101041",
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

		rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
			DeviceID:  device2.ID,
			PricingID: pricing.ID,
		})
		require.NoError(t, err)
		svc.PayRental(ctx, user.ID, rentalInfo.ID)
		svc.StartRental(ctx, user.ID, rentalInfo.ID)
		svc.ReturnRental(ctx, user.ID, rentalInfo.ID)

		err = svc.CompleteRental(ctx, rentalInfo.ID)
		require.NoError(t, err)

		// 验证租借状态
		var rental models.Rental
		svc.db.First(&rental, rentalInfo.ID)
		assert.Equal(t, models.RentalStatusCompleted, rental.Status)
	})
}

func TestRentalService_CompleteRental_NegativeOvertimeFee(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 创建租借并手动设置负超时费（测试边界条件）
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)
	svc.PayRental(ctx, user.ID, rentalInfo.ID)
	svc.StartRental(ctx, user.ID, rentalInfo.ID)

	// 手动设置负超时费（异常数据）
	svc.db.Model(&models.Rental{}).Where("id = ?", rentalInfo.ID).Updates(map[string]interface{}{
		"status":       models.RentalStatusReturned,
		"overtime_fee": -10.0,
		"returned_at":  time.Now(),
	})

	err = svc.CompleteRental(ctx, rentalInfo.ID)
	require.NoError(t, err)

	// 验证完成成功
	var rental models.Rental
	svc.db.First(&rental, rentalInfo.ID)
	assert.Equal(t, models.RentalStatusCompleted, rental.Status)
}

func TestRentalService_CompleteRental_OvertimeExceedsDeposit(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)
	svc.PayRental(ctx, user.ID, rentalInfo.ID)
	svc.StartRental(ctx, user.ID, rentalInfo.ID)

	// 手动设置超时费大于押金（异常数据，应该被处理）
	svc.db.Model(&models.Rental{}).Where("id = ?", rentalInfo.ID).Updates(map[string]interface{}{
		"status":       models.RentalStatusReturned,
		"overtime_fee": pricing.Deposit + 100, // 超过押金
		"returned_at":  time.Now(),
	})

	err = svc.CompleteRental(ctx, rentalInfo.ID)
	require.NoError(t, err)

	var rental models.Rental
	svc.db.First(&rental, rentalInfo.ID)
	assert.Equal(t, models.RentalStatusCompleted, rental.Status)
}

func TestRentalService_toRentalInfo_NoOrder(t *testing.T) {
	svc := setupTestRentalService(t)

	// 测试没有OrderID的情况
	rental := &models.Rental{
		ID:            1,
		OrderID:       0, // 没有关联订单
		UserID:        1,
		DeviceID:      1,
		DurationHours: 1,
		RentalFee:     10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		Status:        models.RentalStatusPending,
		CreatedAt:     time.Now(),
	}

	info := svc.toRentalInfo(rental, nil, nil)
	assert.NotNil(t, info)
	assert.Equal(t, int64(1), info.ID)
	assert.Empty(t, info.OrderNo)
	assert.Nil(t, info.Device)
}

func TestRentalService_toRentalInfo_WithDevice(t *testing.T) {
	svc := setupTestRentalService(t)

	rental := &models.Rental{
		ID:            1,
		OrderID:       0,
		UserID:        1,
		DeviceID:      1,
		DurationHours: 1,
		RentalFee:     10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		Status:        models.RentalStatusInUse,
		CreatedAt:     time.Now(),
	}

	productImage := "/images/test.png"
	device := &models.Device{
		ID:           1,
		DeviceNo:     "D123",
		Name:         "测试设备",
		Type:         models.DeviceTypeStandard,
		ProductName:  "测试产品",
		ProductImage: &productImage,
	}

	info := svc.toRentalInfo(rental, device, nil)
	assert.NotNil(t, info)
	require.NotNil(t, info.Device)
	assert.Equal(t, device.ID, info.Device.ID)
	assert.Equal(t, device.DeviceNo, info.Device.DeviceNo)
	assert.Equal(t, device.Name, info.Device.Name)
	assert.Equal(t, device.ProductName, info.Device.ProductName)
	assert.Equal(t, *device.ProductImage, *info.Device.ProductImage)
}

func TestRentalService_PayRental_OrderNotFound(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 创建租借
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)

	// 删除订单
	svc.db.Delete(&models.Order{}, "id = ?", rentalInfo.OrderID)

	// 尝试支付
	err = svc.PayRental(ctx, user.ID, rentalInfo.ID)
	assert.Error(t, err)
}

func TestRentalService_StartRental_DeviceNotFound(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 创建租借
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)
	svc.PayRental(ctx, user.ID, rentalInfo.ID)

	// 删除设备
	svc.db.Delete(&models.Device{}, "id = ?", device.ID)

	// 尝试开始
	err = svc.StartRental(ctx, user.ID, rentalInfo.ID)
	assert.Error(t, err)
}

func TestRentalService_CreateRental_ZeroTotalAmount(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, _ := createTestData(t, svc.db)

	// 创建免费定价（价格和押金都是0）
	freePricing := &models.RentalPricing{
		VenueID:       &device.VenueID,
		DurationHours: 1,
		Price:         0,
		Deposit:       0,
		OvertimeRate:  0,
		IsActive:      true,
	}
	svc.db.Create(freePricing)

	// 创建新设备
	device2 := &models.Device{
		DeviceNo:       "D20240101050",
		Name:           "免费测试设备",
		Type:           models.DeviceTypeStandard,
		VenueID:        device.VenueID,
		QRCode:         "https://qr.example.com/D20240101050",
		ProductName:    "免费产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	svc.db.Create(device2)

	// 免费租借应该成功（不检查余额）
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device2.ID,
		PricingID: freePricing.ID,
	})
	require.NoError(t, err)
	assert.NotNil(t, rentalInfo)
	assert.Equal(t, float64(0), rentalInfo.RentalFee)
	assert.Equal(t, float64(0), rentalInfo.Deposit)
}

func TestRentalService_PayRental_ZeroAmounts(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, _ := createTestData(t, svc.db)

	// 创建免费定价
	freePricing := &models.RentalPricing{
		VenueID:       &device.VenueID,
		DurationHours: 1,
		Price:         0,
		Deposit:       0,
		OvertimeRate:  0,
		IsActive:      true,
	}
	svc.db.Create(freePricing)

	// 创建新设备
	device2 := &models.Device{
		DeviceNo:       "D20240101051",
		Name:           "免费测试设备2",
		Type:           models.DeviceTypeStandard,
		VenueID:        device.VenueID,
		QRCode:         "https://qr.example.com/D20240101051",
		ProductName:    "免费产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	svc.db.Create(device2)

	// 创建免费租借
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device2.ID,
		PricingID: freePricing.ID,
	})
	require.NoError(t, err)

	// 支付免费租借
	err = svc.PayRental(ctx, user.ID, rentalInfo.ID)
	require.NoError(t, err)

	// 验证订单状态
	var order models.Order
	svc.db.First(&order, rentalInfo.OrderID)
	assert.Equal(t, models.OrderStatusPaid, order.Status)
}

func TestRentalService_CompleteRental_ZeroDeposit(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, _ := createTestData(t, svc.db)

	// 创建无押金定价
	noDepositPricing := &models.RentalPricing{
		VenueID:       &device.VenueID,
		DurationHours: 1,
		Price:         10.0,
		Deposit:       0, // 无押金
		OvertimeRate:  0,
		IsActive:      true,
	}
	svc.db.Create(noDepositPricing)

	// 创建新设备
	device2 := &models.Device{
		DeviceNo:       "D20240101052",
		Name:           "无押金测试设备",
		Type:           models.DeviceTypeStandard,
		VenueID:        device.VenueID,
		QRCode:         "https://qr.example.com/D20240101052",
		ProductName:    "无押金产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	svc.db.Create(device2)

	// 创建无押金租借
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device2.ID,
		PricingID: noDepositPricing.ID,
	})
	require.NoError(t, err)

	// 完整流程
	svc.PayRental(ctx, user.ID, rentalInfo.ID)
	svc.StartRental(ctx, user.ID, rentalInfo.ID)
	svc.ReturnRental(ctx, user.ID, rentalInfo.ID)

	// 完成无押金租借
	err = svc.CompleteRental(ctx, rentalInfo.ID)
	require.NoError(t, err)

	// 验证租借状态
	var rental models.Rental
	svc.db.First(&rental, rentalInfo.ID)
	assert.Equal(t, models.RentalStatusCompleted, rental.Status)
}

func TestRentalService_GetRental_NotFound(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, _, _ := createTestData(t, svc.db)

	_, err := svc.GetRental(ctx, user.ID, 999999)
	assert.Error(t, err)
}

func TestRentalService_toRentalInfo_WithOrderID(t *testing.T) {
	svc := setupTestRentalService(t)

	user, device, pricing := createTestData(t, svc.db)

	// 创建租借，获取关联的Order
	ctx := context.Background()
	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)

	// 获取租借详情验证OrderNo
	info, err := svc.GetRental(ctx, user.ID, rentalInfo.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, info.OrderNo)
	assert.Equal(t, rentalInfo.OrderID, info.OrderID)
}
