// Package repository 租借仓储单元测试
package repository

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
)

// setupRentalTestDB 创建租借测试数据库
func setupRentalTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.MemberLevel{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.RentalPricing{},
		&models.Rental{},
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

// createRentalTestData 创建租借测试基础数据
func createRentalTestData(t *testing.T, db *gorm.DB) (user *models.User, device *models.Device, pricing *models.RentalPricing) {
	// 创建用户
	phone := "13800138000"
	user = &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

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
		Status:         models.DeviceStatusActive,
	}
	db.Create(device)

	// 创建定价
	pricing = &models.RentalPricing{
		DeviceID:     device.ID,
		Name:         "1小时租借",
		Duration:     1,
		DurationUnit: models.DurationUnitHour,
		Price:        10.0,
		Deposit:      50.0,
		Status:       models.RentalPricingStatusActive,
	}
	db.Create(pricing)

	return user, device, pricing
}

func TestRentalRepository_Create(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101001",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPending,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
	}

	err := repo.Create(ctx, rental)
	require.NoError(t, err)
	assert.NotZero(t, rental.ID)
}

func TestRentalRepository_GetByID(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101002",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPending,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
	}
	db.Create(rental)

	t.Run("获取存在的租借", func(t *testing.T) {
		found, err := repo.GetByID(ctx, rental.ID)
		require.NoError(t, err)
		assert.Equal(t, rental.RentalNo, found.RentalNo)
	})

	t.Run("获取不存在的租借", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestRentalRepository_GetByRentalNo(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101003",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPending,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
	}
	db.Create(rental)

	t.Run("根据订单号获取租借", func(t *testing.T) {
		found, err := repo.GetByRentalNo(ctx, rental.RentalNo)
		require.NoError(t, err)
		assert.Equal(t, rental.ID, found.ID)
	})

	t.Run("获取不存在的订单号", func(t *testing.T) {
		_, err := repo.GetByRentalNo(ctx, "R99999999999")
		assert.Error(t, err)
	})
}

func TestRentalRepository_Update(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

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
	db.Create(rental)

	rental.Status = models.RentalStatusPaid
	err := repo.Update(ctx, rental)
	require.NoError(t, err)

	var found models.Rental
	db.First(&found, rental.ID)
	assert.Equal(t, int8(models.RentalStatusPaid), found.Status)
}

func TestRentalRepository_UpdateFields(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101005",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPending,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
	}
	db.Create(rental)

	now := time.Now()
	err := repo.UpdateFields(ctx, rental.ID, map[string]interface{}{
		"status":  models.RentalStatusPaid,
		"paid_at": now,
	})
	require.NoError(t, err)

	var found models.Rental
	db.First(&found, rental.ID)
	assert.Equal(t, int8(models.RentalStatusPaid), found.Status)
	assert.NotNil(t, found.PaidAt)
}

func TestRentalRepository_UpdateStatus(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

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
	}
	db.Create(rental)

	err := repo.UpdateStatus(ctx, rental.ID, models.RentalStatusCancelled)
	require.NoError(t, err)

	var found models.Rental
	db.First(&found, rental.ID)
	assert.Equal(t, int8(models.RentalStatusCancelled), found.Status)
}

func TestRentalRepository_ListByUser(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	// 创建多个租借
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
		db.Create(rental)
	}

	t.Run("获取用户租借列表", func(t *testing.T) {
		rentals, total, err := repo.ListByUser(ctx, user.ID, 0, 10, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 5)
	})

	t.Run("按状态筛选", func(t *testing.T) {
		status := int8(models.RentalStatusPending)
		rentals, total, err := repo.ListByUser(ctx, user.ID, 0, 10, &status)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 5)
	})

	t.Run("分页获取", func(t *testing.T) {
		rentals, total, err := repo.ListByUser(ctx, user.ID, 0, 2, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 2)
	})
}

func TestRentalRepository_HasActiveRental(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	t.Run("无进行中租借", func(t *testing.T) {
		has, err := repo.HasActiveRental(ctx, user.ID)
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("有进行中租借", func(t *testing.T) {
		slotNo := 1
		rental := &models.Rental{
			RentalNo:      "R20240101012",
			UserID:        user.ID,
			DeviceID:      device.ID,
			PricingID:     pricing.ID,
			SlotNo:        &slotNo,
			Status:        models.RentalStatusInUse,
			UnitPrice:     pricing.Price,
			DepositAmount: pricing.Deposit,
		}
		db.Create(rental)

		has, err := repo.HasActiveRental(ctx, user.ID)
		require.NoError(t, err)
		assert.True(t, has)
	})
}

func TestRentalRepository_GetActiveByUser(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101013",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPaid,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
	}
	db.Create(rental)

	found, err := repo.GetActiveByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, rental.ID, found.ID)
}

func TestRentalRepository_GetCurrentByDevice(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101014",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusInUse,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
	}
	db.Create(rental)

	found, err := repo.GetCurrentByDevice(ctx, device.ID)
	require.NoError(t, err)
	assert.Equal(t, rental.ID, found.ID)
}

func TestRentalRepository_GetExpiredPending(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	// 创建过期的待支付订单（通过直接修改数据库记录）
	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101015",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPending,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
	}
	db.Create(rental)

	// 手动更新 created_at 为过去时间
	db.Model(rental).Update("created_at", time.Now().Add(-1*time.Hour))

	rentals, err := repo.GetExpiredPending(ctx, time.Now().Add(-30*time.Minute), 10)
	require.NoError(t, err)
	assert.Len(t, rentals, 1)
}

func TestRentalRepository_GetOverdue(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	// 创建超时的租借
	slotNo := 1
	now := time.Now()
	startTime := now.Add(-2 * time.Hour)
	endTime := now.Add(-1 * time.Hour) // 已过期
	rental := &models.Rental{
		RentalNo:      "R20240101016",
		UserID:        user.ID,
		DeviceID:      device.ID,
		PricingID:     pricing.ID,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusInUse,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
		StartTime:     &startTime,
		EndTime:       &endTime,
	}
	db.Create(rental)

	rentals, err := repo.GetOverdue(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, rentals, 1)
}

func TestRentalRepository_CountByStatus(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	// 创建不同状态的租借
	statuses := []int8{
		models.RentalStatusPending,
		models.RentalStatusPaid,
		models.RentalStatusInUse,
		models.RentalStatusCompleted,
	}

	slotNo := 1
	for i, status := range statuses {
		rental := &models.Rental{
			RentalNo:      "R2024010101" + string(rune('7'+i)),
			UserID:        user.ID,
			DeviceID:      device.ID,
			PricingID:     pricing.ID,
			SlotNo:        &slotNo,
			Status:        status,
			UnitPrice:     pricing.Price,
			DepositAmount: pricing.Deposit,
		}
		db.Create(rental)
	}

	counts, err := repo.CountByStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), counts[models.RentalStatusPending])
	assert.Equal(t, int64(1), counts[models.RentalStatusPaid])
	assert.Equal(t, int64(1), counts[models.RentalStatusInUse])
	assert.Equal(t, int64(1), counts[models.RentalStatusCompleted])
}

func TestRentalRepository_List(t *testing.T) {
	db := setupRentalTestDB(t)
	repo := NewRentalRepository(db)
	ctx := context.Background()

	user, device, pricing := createRentalTestData(t, db)

	// 创建多个租借
	slotNo := 1
	for i := 0; i < 5; i++ {
		rental := &models.Rental{
			RentalNo:      "R2024010102" + string(rune('0'+i)),
			UserID:        user.ID,
			DeviceID:      device.ID,
			PricingID:     pricing.ID,
			SlotNo:        &slotNo,
			Status:        models.RentalStatusPending,
			UnitPrice:     pricing.Price,
			DepositAmount: pricing.Deposit,
		}
		db.Create(rental)
	}

	t.Run("获取所有租借列表", func(t *testing.T) {
		rentals, total, err := repo.List(ctx, 0, 10, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 5)
	})

	t.Run("按用户筛选", func(t *testing.T) {
		filters := map[string]interface{}{
			"user_id": user.ID,
		}
		rentals, total, err := repo.List(ctx, 0, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 5)
	})

	t.Run("按设备筛选", func(t *testing.T) {
		filters := map[string]interface{}{
			"device_id": device.ID,
		}
		rentals, total, err := repo.List(ctx, 0, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, rentals, 5)
	})
}
