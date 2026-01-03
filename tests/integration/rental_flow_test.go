//go:build integration
// +build integration

// Package integration 租借流程集成测试
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
	rentalService "github.com/dumeirei/smart-locker-backend/internal/service/rental"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

// setupRentalIntegrationDB 创建集成测试数据库
func setupRentalIntegrationDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.DeviceLog{},
		&models.RentalPricing{},
		&models.Order{},
		&models.Rental{},
		&models.Payment{},
		&models.Refund{},
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

// setupRentalTestEnvironment 设置租借测试环境
func setupRentalTestEnvironment(t *testing.T, db *gorm.DB) (*rentalService.RentalService, *models.User, *models.Device, *models.RentalPricing) {
	// 创建用户
	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	// 创建钱包，余额足够支付
	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 200.0,
	}
	db.Create(wallet)

	// 创建商户
	merchant := &models.Merchant{
		Name:           "测试商户",
		ContactName:    "联系人",
		ContactPhone:   "13900139000",
		CommissionRate: 0.2,
		SettlementType: "monthly",
		Status:         models.MerchantStatusActive,
	}
	db.Create(merchant)

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
	db.Create(venue)

	// 创建设备
	device := &models.Device{
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
	db.Create(device)

	// 创建定价
	pricing := &models.RentalPricing{
		VenueID:       &venue.ID,
		DurationHours: 1,
		Price:         10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		IsActive:      true,
	}
	db.Create(pricing)

	// 创建服务
	rentalRepo := repository.NewRentalRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	userRepo := repository.NewUserRepository(db)
	deviceSvc := deviceService.NewDeviceService(db, deviceRepo, venueRepo)
	walletSvc := userService.NewWalletService(db, userRepo)

	svc := rentalService.NewRentalService(db, rentalRepo, deviceRepo, deviceSvc, walletSvc, nil)

	return svc, user, device, pricing
}

func TestRentalFlow_CompleteProcess(t *testing.T) {
	db := setupRentalIntegrationDB(t)
	svc, user, device, pricing := setupRentalTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建租借订单
	t.Run("步骤1: 创建租借订单", func(t *testing.T) {
		req := &rentalService.CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: pricing.ID,
		}

		rentalInfo, err := svc.CreateRental(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, rentalInfo)
		assert.Equal(t, models.RentalStatusPending, rentalInfo.Status)
		
		// 检查OrderNo
		var order models.Order
		db.Where("id = ?", rentalInfo.OrderID).First(&order)
		assert.NotEmpty(t, order.OrderNo)

		// 验证设备槽位已减少
		var updatedDevice models.Device
		db.First(&updatedDevice, device.ID)
		assert.Equal(t, 0, updatedDevice.AvailableSlots)
	})

	// 获取创建的订单
	var rental models.Rental
	db.Where("user_id = ?", user.ID).First(&rental)

	// 2. 支付订单
	t.Run("步骤2: 支付订单", func(t *testing.T) {
		err := svc.PayRental(ctx, user.ID, rental.ID)
		require.NoError(t, err)

		// 验证订单状态
		db.First(&rental, rental.ID)
		assert.Equal(t, models.RentalStatusPaid, rental.Status)
		// 检查Order状态
		var order models.Order
		db.Where("id = ?", rental.OrderID).First(&order)
		assert.NotNil(t, order.PaidAt)

		// 验证钱包余额变化
		var wallet models.UserWallet
		db.Where("user_id = ?", user.ID).First(&wallet)
		assert.Equal(t, 200.0-(pricing.Price+pricing.Deposit), wallet.Balance)
		assert.Equal(t, pricing.Deposit, wallet.FrozenBalance)
	})

	// 3. 开始租借（取货）
	t.Run("步骤3: 开始租借", func(t *testing.T) {
		err := svc.StartRental(ctx, user.ID, rental.ID)
		require.NoError(t, err)

		// 验证订单状态
		db.First(&rental, rental.ID)
		assert.Equal(t, models.RentalStatusInUse, rental.Status)
		assert.NotNil(t, rental.UnlockedAt)
		assert.NotNil(t, rental.ExpectedReturnAt)

			// 验证设备状态
			var updatedDevice models.Device
			db.First(&updatedDevice, device.ID)
			assert.EqualValues(t, models.DeviceRentalInUse, updatedDevice.RentalStatus)
		})

	// 4. 归还租借
	t.Run("步骤4: 归还租借", func(t *testing.T) {
		err := svc.ReturnRental(ctx, user.ID, rental.ID)
		require.NoError(t, err)

		// 验证订单状态
		db.First(&rental, rental.ID)
		assert.Equal(t, models.RentalStatusReturned, rental.Status)
		assert.NotNil(t, rental.ReturnedAt)

			// 验证设备状态恢复
			var updatedDevice models.Device
			db.First(&updatedDevice, device.ID)
			assert.EqualValues(t, models.DeviceRentalFree, updatedDevice.RentalStatus)
			assert.Equal(t, 1, updatedDevice.AvailableSlots)
		})

	// 5. 完成租借（结算）
	t.Run("步骤5: 完成租借", func(t *testing.T) {
		err := svc.CompleteRental(ctx, rental.ID)
		require.NoError(t, err)

		// 验证订单状态
		db.First(&rental, rental.ID)
		assert.Equal(t, models.RentalStatusCompleted, rental.Status)
		// 检查Order完成时间
		var order models.Order
		db.Where("id = ?", rental.OrderID).First(&order)
		assert.NotNil(t, order.CompletedAt)

		// 验证钱包余额 - 押金已退还
		var wallet models.UserWallet
		db.Where("user_id = ?", user.ID).First(&wallet)
		assert.Equal(t, 200.0-pricing.Price, wallet.Balance)
		assert.Equal(t, float64(0), wallet.FrozenBalance)
	})
}

func TestRentalFlow_CancelPendingOrder(t *testing.T) {
	db := setupRentalIntegrationDB(t)
	svc, user, device, pricing := setupRentalTestEnvironment(t, db)
	ctx := context.Background()

	// 创建租借订单
	req := &rentalService.CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	}

	rentalInfo, err := svc.CreateRental(ctx, user.ID, req)
	require.NoError(t, err)

	// 验证设备槽位减少
	var updatedDevice models.Device
	db.First(&updatedDevice, device.ID)
	assert.Equal(t, 0, updatedDevice.AvailableSlots)

	// 取消订单
	err = svc.CancelRental(ctx, user.ID, rentalInfo.ID)
	require.NoError(t, err)

	// 验证订单状态
	var rental models.Rental
	db.First(&rental, rentalInfo.ID)
	assert.Equal(t, models.RentalStatusCancelled, rental.Status)

	// 验证设备槽位恢复
	db.First(&updatedDevice, device.ID)
	assert.Equal(t, 1, updatedDevice.AvailableSlots)

	// 验证钱包余额未变化
	var wallet models.UserWallet
	db.Where("user_id = ?", user.ID).First(&wallet)
	assert.Equal(t, float64(200), wallet.Balance)
}

func TestRentalFlow_PreventDuplicateActiveRental(t *testing.T) {
	db := setupRentalIntegrationDB(t)
	svc, user, device, pricing := setupRentalTestEnvironment(t, db)
	ctx := context.Background()

	// 创建第一个订单
	req := &rentalService.CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	}

	_, err := svc.CreateRental(ctx, user.ID, req)
	require.NoError(t, err)

	// 创建另一个设备和定价
	device2 := &models.Device{
		DeviceNo:       "D20240101002",
		Name:           "测试设备2",
		Type:           models.DeviceTypeStandard,
		VenueID:        1,
		QRCode:         "https://qr.example.com/D20240101002",
		ProductName:    "测试产品2",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		Status:         models.DeviceStatusActive,
	}
	db.Create(device2)

	pricing2 := &models.RentalPricing{
		VenueID:      &device2.VenueID,
		DurationHours: 1,
		Price:         10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		IsActive:      true,
	}
	db.Create(pricing2)

	// 尝试创建第二个订单（应该失败，因为有进行中的订单）
	req2 := &rentalService.CreateRentalRequest{
		DeviceID:  device2.ID,
		PricingID: pricing2.ID,
	}

	_, err = svc.CreateRental(ctx, user.ID, req2)
	assert.Error(t, err) // 应该返回错误
}

func TestRentalFlow_InsufficientBalance(t *testing.T) {
	db := setupRentalIntegrationDB(t)

	// 创建余额不足的用户
	phone := "13800138001"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "穷用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 10.0, // 余额不足
	}
	db.Create(wallet)

	// 创建设备和定价
	venue := &models.Venue{
		MerchantID: 1,
		Name:       "测试场地",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "测试地址",
		Status:     models.VenueStatusActive,
	}
	db.Create(venue)

	device := &models.Device{
		DeviceNo:       "D20240101003",
		Name:           "测试设备3",
		Type:           models.DeviceTypeStandard,
		VenueID:        venue.ID,
		QRCode:         "https://qr.example.com/D20240101003",
		ProductName:    "测试产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		Status:         models.DeviceStatusActive,
	}
	db.Create(device)

	pricing := &models.RentalPricing{
		VenueID:       &venue.ID,
		DurationHours:  1,
		Price:         10.0,
		Deposit:       50.0, // 总计60，超过余额
		OvertimeRate:  1.5,
		IsActive:      true,
	}
	db.Create(pricing)

	// 创建服务
	rentalRepo := repository.NewRentalRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	userRepo := repository.NewUserRepository(db)
	deviceSvc := deviceService.NewDeviceService(db, deviceRepo, venueRepo)
	walletSvc := userService.NewWalletService(db, userRepo)
	svc := rentalService.NewRentalService(db, rentalRepo, deviceRepo, deviceSvc, walletSvc, nil)

	ctx := context.Background()

	// 尝试创建订单
	req := &rentalService.CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	}

	_, err := svc.CreateRental(ctx, user.ID, req)
	assert.Error(t, err) // 应该返回余额不足错误
}

func TestRentalFlow_OverdueRental(t *testing.T) {
	db := setupRentalIntegrationDB(t)
	ctx := context.Background()

	// 创建用户
	phone := "13800138002"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 200.0,
	}
	db.Create(wallet)

	// 创建设备
	venue := &models.Venue{
		MerchantID: 1,
		Name:       "测试场地",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "测试地址",
		Status:     models.VenueStatusActive,
	}
	db.Create(venue)

	device := &models.Device{
		DeviceNo:       "D20240101004",
		Name:           "测试设备4",
		Type:           models.DeviceTypeStandard,
		VenueID:        venue.ID,
		QRCode:         "https://qr.example.com/D20240101004",
		ProductName:    "测试产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		Status:         models.DeviceStatusActive,
	}
	db.Create(device)

	pricing := &models.RentalPricing{
		VenueID:       &venue.ID,
		DurationHours:  1,
		Price:         10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		IsActive:      true,
	}
	db.Create(pricing)

	// 直接创建一个已归还但超时的订单
	now := time.Now()
	unlockedAt := now.Add(-3 * time.Hour)
	expectedReturnAt := now.Add(-2 * time.Hour)     // 应该在2小时前归还
	returnedAt := now.Add(-30 * time.Minute) // 实际30分钟前归还，超时1.5小时

	// 先创建Order
	order := &models.Order{
		OrderNo:        "R20240101001",
		UserID:         user.ID,
		Type:           models.OrderTypeRental,
		OriginalAmount:  pricing.Price + pricing.Deposit,
		DiscountAmount: 0.0,
		ActualAmount:   pricing.Price + pricing.Deposit,
		DepositAmount:  pricing.Deposit,
		Status:         models.OrderStatusCompleted,
		PaidAt:        &unlockedAt,
		CompletedAt:   &returnedAt,
	}
	db.Create(order)

	rental := &models.Rental{
		OrderID:          order.ID,
		UserID:           user.ID,
		DeviceID:         device.ID,
		DurationHours:     1,
		RentalFee:        pricing.Price,
		Deposit:          pricing.Deposit,
		OvertimeRate:      1.5,
		OvertimeFee:       20.0, // 超时费用
		Status:           models.RentalStatusReturned,
		UnlockedAt:       &unlockedAt,
		ExpectedReturnAt: &expectedReturnAt,
		ReturnedAt:       &returnedAt,
	}
	db.Create(rental)

	// 设置钱包冻结余额
	// 模拟已支付：租金 + 押金已从余额扣除，押金被冻结
	db.Model(&wallet).Updates(map[string]interface{}{
		"balance":        200.0 - (pricing.Price + pricing.Deposit),
		"frozen_balance": pricing.Deposit,
	})

	// 创建服务
	rentalRepo := repository.NewRentalRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	userRepo := repository.NewUserRepository(db)
	deviceSvc := deviceService.NewDeviceService(db, deviceRepo, venueRepo)
	walletSvc := userService.NewWalletService(db, userRepo)
	svc := rentalService.NewRentalService(db, rentalRepo, deviceRepo, deviceSvc, walletSvc, nil)

	// 完成租借（结算）
	err := svc.CompleteRental(ctx, rental.ID)
	require.NoError(t, err)

	// 验证订单状态
	db.First(&rental, rental.ID)
	assert.Equal(t, models.RentalStatusCompleted, rental.Status)

	// 超时费用应从押金扣除
	// 超时约1.5小时，按小时计算应该扣除 2 小时的费用 = 20元
	// 押金50元，扣除20元后应退还30元
	var updatedWallet models.UserWallet
	db.Where("user_id = ?", user.ID).First(&updatedWallet)
	assert.Equal(t, 200.0-(pricing.Price+20.0), updatedWallet.Balance) // 初始200 - 租金10 - 超时20
	assert.Equal(t, float64(0), updatedWallet.FrozenBalance)
}
