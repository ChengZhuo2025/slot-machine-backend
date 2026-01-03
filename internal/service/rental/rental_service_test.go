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
		CommissionRate: 0.2,
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
		DeviceID:     device.ID,
		Name:         "1小时租借",
		Duration:     1,
		DurationUnit: models.DurationUnitHour,
		Price:        10.0,
		Deposit:      50.0,
		IsDefault:    true,
		Status:       models.RentalPricingStatusActive,
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
		assert.NotEmpty(t, rentalInfo.RentalNo)
		assert.Equal(t, models.RentalStatusPending, rentalInfo.Status)
		assert.Equal(t, pricing.Price, rentalInfo.UnitPrice)
		assert.Equal(t, pricing.Deposit, rentalInfo.DepositAmount)

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

		// 创建新设备
		newDevice := &models.Device{
			DeviceNo:       "D20240101002",
			Name:           "测试设备2",
			Type:           models.DeviceTypeStandard,
			VenueID:        1,
			QRCode:         "https://qr.example.com/D20240101002",
			ProductName:    "测试产品",
			SlotCount:      1,
			AvailableSlots: 1,
			OnlineStatus:   models.DeviceOnline,
			Status:         models.DeviceStatusActive,
		}
		svc.db.Create(newDevice)

		newPricing := &models.RentalPricing{
			DeviceID:     newDevice.ID,
			Name:         "1小时租借",
			Duration:     1,
			DurationUnit: models.DurationUnitHour,
			Price:        10.0,
			Deposit:      50.0,
			Status:       models.RentalPricingStatusActive,
		}
		svc.db.Create(newPricing)

		req := &CreateRentalRequest{
			DeviceID:  newDevice.ID,
			PricingID: newPricing.ID,
		}

		_, err := svc.CreateRental(ctx, poorUser.ID, req)
		assert.Error(t, err)
	})

	t.Run("设备不可用创建失败", func(t *testing.T) {
		// 创建禁用设备
		disabledDevice := &models.Device{
			DeviceNo:       "D20240101003",
			Name:           "禁用设备",
			Type:           models.DeviceTypeStandard,
			VenueID:        1,
			QRCode:         "https://qr.example.com/D20240101003",
			ProductName:    "测试产品",
			SlotCount:      1,
			AvailableSlots: 1,
			Status:         models.DeviceStatusDisabled,
		}
		svc.db.Create(disabledDevice)

		disabledPricing := &models.RentalPricing{
			DeviceID:     disabledDevice.ID,
			Name:         "1小时租借",
			Duration:     1,
			DurationUnit: models.DurationUnitHour,
			Price:        10.0,
			Deposit:      50.0,
			Status:       models.RentalPricingStatusActive,
		}
		svc.db.Create(disabledPricing)

		req := &CreateRentalRequest{
			DeviceID:  disabledDevice.ID,
			PricingID: disabledPricing.ID,
		}

		_, err := svc.CreateRental(ctx, user.ID, req)
		assert.Error(t, err)
	})
}

func TestRentalService_PayRental(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("支付订单成功", func(t *testing.T) {
		// 先创建订单
		req := &CreateRentalRequest{
			DeviceID:  device.ID,
			PricingID: pricing.ID,
		}
		rentalInfo, err := svc.CreateRental(ctx, user.ID, req)
		require.NoError(t, err)

		// 支付订单
		err = svc.PayRental(ctx, user.ID, rentalInfo.ID)
		require.NoError(t, err)

		// 验证订单状态
		var rental models.Rental
		svc.db.First(&rental, rentalInfo.ID)
		assert.Equal(t, models.RentalStatusPaid, rental.Status)
		assert.NotNil(t, rental.PaidAt)

		// 验证钱包扣款
		var wallet models.UserWallet
		svc.db.Where("user_id = ?", user.ID).First(&wallet)
		assert.Equal(t, 200.0-pricing.Price, wallet.Balance)
		assert.Equal(t, pricing.Deposit, wallet.FrozenBalance)
	})

	t.Run("非订单所有者支付失败", func(t *testing.T) {
		// 创建另一个用户
		anotherPhone := "13800138002"
		anotherUser := &models.User{
			Phone:         &anotherPhone,
			Nickname:      "另一个用户",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		svc.db.Create(anotherUser)

		// 获取一个待支付订单
		var rental models.Rental
		svc.db.Where("status = ?", models.RentalStatusPending).First(&rental)

		if rental.ID > 0 {
			err := svc.PayRental(ctx, anotherUser.ID, rental.ID)
			assert.Error(t, err)
		}
	})
}

func TestRentalService_StartRental(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	// 创建测试数据并手动创建已支付订单
	user, device, pricing := createTestData(t, svc.db)

	// 手动创建已支付状态的租借订单
	now := time.Now()
	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101001",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPaid,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
		RentalAmount:  pricing.Price,
		ActualAmount:  pricing.Price + pricing.Deposit,
		PaidAt:        &now,
	}
	svc.db.Create(rental)

	t.Run("开始租借成功", func(t *testing.T) {
		err := svc.StartRental(ctx, user.ID, rental.ID)
		require.NoError(t, err)

		// 验证订单状态
		var updatedRental models.Rental
		svc.db.First(&updatedRental, rental.ID)
		assert.Equal(t, models.RentalStatusInUse, updatedRental.Status)
		assert.NotNil(t, updatedRental.StartTime)
		assert.NotNil(t, updatedRental.EndTime)

		// 验证设备状态
		var updatedDevice models.Device
		svc.db.First(&updatedDevice, device.ID)
		assert.Equal(t, models.DeviceRentalInUse, updatedDevice.RentalStatus)
	})

	t.Run("非已支付状态开始失败", func(t *testing.T) {
		// 创建待支付订单
		pendingRental := &models.Rental{
			RentalNo:      "R20240101002",
			UserID:        user.ID,
			DeviceID:      device.ID,
			PricingID:     pricing.ID,
			SlotNo:        &slotNo,
			Status:        models.RentalStatusPending,
			UnitPrice:     pricing.Price,
			DepositAmount: pricing.Deposit,
		}
		svc.db.Create(pendingRental)

		err := svc.StartRental(ctx, user.ID, pendingRental.ID)
		assert.Error(t, err)
	})
}

func TestRentalService_ReturnRental(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 创建使用中的租借订单
	now := time.Now()
	startTime := now.Add(-30 * time.Minute)
	endTime := now.Add(30 * time.Minute)
	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101003",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusInUse,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
		RentalAmount:  pricing.Price,
		ActualAmount:  pricing.Price + pricing.Deposit,
		PaidAt:        &startTime,
		StartTime:     &startTime,
		EndTime:       &endTime,
	}
	svc.db.Create(rental)

	// 更新设备状态
	svc.db.Model(&device).Updates(map[string]interface{}{
		"rental_status":     models.DeviceRentalInUse,
		"current_rental_id": rental.ID,
		"available_slots":   0,
	})

	t.Run("归还租借成功", func(t *testing.T) {
		err := svc.ReturnRental(ctx, user.ID, rental.ID)
		require.NoError(t, err)

		// 验证订单状态
		var updatedRental models.Rental
		svc.db.First(&updatedRental, rental.ID)
		assert.Equal(t, models.RentalStatusReturned, updatedRental.Status)
		assert.NotNil(t, updatedRental.ReturnedAt)
		assert.NotNil(t, updatedRental.Duration)

		// 验证设备状态恢复
		var updatedDevice models.Device
		svc.db.First(&updatedDevice, device.ID)
		assert.Equal(t, models.DeviceRentalFree, updatedDevice.RentalStatus)
		assert.Equal(t, 1, updatedDevice.AvailableSlots)
	})
}

func TestRentalService_CancelRental(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	t.Run("取消待支付订单成功", func(t *testing.T) {
		slotNo := 1
		rental := &models.Rental{
			RentalNo:      "R20240101004",
			UserID:        user.ID,
			DeviceID:      device.ID,
			PricingID:     pricing.ID,
			SlotNo:        &slotNo,
			Status:        models.RentalStatusPending,
			UnitPrice:     pricing.Price,
			DepositAmount: pricing.Deposit,
		}
		svc.db.Create(rental)

		// 减少设备槽位
		svc.db.Model(&device).Update("available_slots", 0)

		err := svc.CancelRental(ctx, user.ID, rental.ID)
		require.NoError(t, err)

		// 验证订单状态
		var updatedRental models.Rental
		svc.db.First(&updatedRental, rental.ID)
		assert.Equal(t, models.RentalStatusCancelled, updatedRental.Status)

		// 验证设备槽位恢复
		var updatedDevice models.Device
		svc.db.First(&updatedDevice, device.ID)
		assert.Equal(t, 1, updatedDevice.AvailableSlots)
	})

	t.Run("非待支付状态取消失败", func(t *testing.T) {
		slotNo := 1
		paidRental := &models.Rental{
			RentalNo:      "R20240101005",
			UserID:        user.ID,
			DeviceID:      device.ID,
			PricingID:     pricing.ID,
			SlotNo:        &slotNo,
			Status:        models.RentalStatusPaid,
			UnitPrice:     pricing.Price,
			DepositAmount: pricing.Deposit,
		}
		svc.db.Create(paidRental)

		err := svc.CancelRental(ctx, user.ID, paidRental.ID)
		assert.Error(t, err)
	})
}

func TestRentalService_GetRental(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101006",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPending,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
		RentalAmount:  pricing.Price,
		ActualAmount:  pricing.Price + pricing.Deposit,
	}
	svc.db.Create(rental)

	t.Run("获取租借详情成功", func(t *testing.T) {
		info, err := svc.GetRental(ctx, user.ID, rental.ID)
		require.NoError(t, err)
		assert.Equal(t, rental.ID, info.ID)
		assert.Equal(t, rental.RentalNo, info.RentalNo)
		assert.Equal(t, rental.Status, info.Status)
	})

	t.Run("获取不存在的租借失败", func(t *testing.T) {
		_, err := svc.GetRental(ctx, user.ID, 99999)
		assert.Error(t, err)
	})

	t.Run("获取非自己的租借失败", func(t *testing.T) {
		anotherPhone := "13800138003"
		anotherUser := &models.User{
			Phone:         &anotherPhone,
			Nickname:      "另一个用户",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		svc.db.Create(anotherUser)

		_, err := svc.GetRental(ctx, anotherUser.ID, rental.ID)
		assert.Error(t, err)
	})
}

func TestRentalService_ListRentals(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()

	user, device, pricing := createTestData(t, svc.db)

	// 创建多个租借订单
	slotNo := 1
	for i := 0; i < 5; i++ {
		rental := &models.Rental{
			RentalNo:      "R2024010100" + string(rune('7'+i)),
			UserID:        user.ID,
			DeviceID:      device.ID,
			PricingID:     pricing.ID,
			SlotNo:        &slotNo,
			Status:        models.RentalStatusPending,
			UnitPrice:     pricing.Price,
			DepositAmount: pricing.Deposit,
		}
		svc.db.Create(rental)
	}

	t.Run("获取租借列表成功", func(t *testing.T) {
		rentals, total, err := svc.ListRentals(ctx, user.ID, 0, 10, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 5)
	})

	t.Run("按状态筛选", func(t *testing.T) {
		status := int8(models.RentalStatusPending)
		rentals, total, err := svc.ListRentals(ctx, user.ID, 0, 10, &status)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 5)
	})

	t.Run("分页获取", func(t *testing.T) {
		rentals, total, err := svc.ListRentals(ctx, user.ID, 0, 2, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 2)
	})
}

func TestRentalService_getStatusName(t *testing.T) {
	svc := setupTestRentalService(t)

	tests := []struct {
		status   int8
		expected string
	}{
		{models.RentalStatusPending, "待支付"},
		{models.RentalStatusPaid, "待取货"},
		{models.RentalStatusInUse, "使用中"},
		{models.RentalStatusReturned, "已归还"},
		{models.RentalStatusCompleted, "已完成"},
		{models.RentalStatusCancelled, "已取消"},
		{models.RentalStatusOverdue, "超时未还"},
		{99, "未知"},
	}

	for _, tt := range tests {
		name := svc.getStatusName(tt.status)
		assert.Equal(t, tt.expected, name)
	}
}
