// Package repository 佣金仓储单元测试
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

func setupCommissionTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Commission{}, &models.Distributor{}, &models.User{})
	require.NoError(t, err)

	return db
}

func TestCommissionRepository_Create(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	commission := &models.Commission{
		DistributorID: 1,
		OrderID:       1,
		FromUserID:    1,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100.0,
		Rate:          0.1,
		Amount:        10.0,
	}

	err := repo.Create(ctx, commission)
	require.NoError(t, err)
	assert.NotZero(t, commission.ID)
}

func TestCommissionRepository_CreateBatch(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	commissions := []*models.Commission{
		{DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect, OrderAmount: 100.0, Rate: 0.1, Amount: 10.0},
		{DistributorID: 2, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeIndirect, OrderAmount: 100.0, Rate: 0.05, Amount: 5.0},
	}

	err := repo.CreateBatch(ctx, commissions)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Commission{}).Where("order_id = ?", 1).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestCommissionRepository_GetByID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	commission := &models.Commission{
		DistributorID: 1,
		OrderID:       1,
		FromUserID:    1,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100.0,
		Rate:          0.1,
		Amount:        10.0,
	}
	db.Create(commission)

	found, err := repo.GetByID(ctx, commission.ID)
	require.NoError(t, err)
	assert.Equal(t, commission.ID, found.ID)
	assert.Equal(t, 10.0, found.Amount)
}

func TestCommissionRepository_GetByOrderID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect, OrderAmount: 100.0, Rate: 0.1, Amount: 10.0,
	})

	db.Create(&models.Commission{
		DistributorID: 2, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeIndirect, OrderAmount: 100.0, Rate: 0.05, Amount: 5.0,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeDirect, OrderAmount: 200.0, Rate: 0.1, Amount: 20.0,
	})

	list, err := repo.GetByOrderID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
}

func TestCommissionRepository_GetByDistributorID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect, OrderAmount: 100.0, Rate: 0.1, Amount: 10.0,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeDirect, OrderAmount: 200.0, Rate: 0.1, Amount: 20.0,
	})

	db.Create(&models.Commission{
		DistributorID: 2, OrderID: 3, FromUserID: 1, Type: models.CommissionTypeDirect, OrderAmount: 150.0, Rate: 0.1, Amount: 15.0,
	})

	list, total, err := repo.GetByDistributorID(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestCommissionRepository_Update(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	commission := &models.Commission{
		DistributorID: 1,
		OrderID:       1,
		FromUserID:    1,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100.0,
		Rate:          0.1,
		Amount:        10.0,
	}
	db.Create(commission)

	commission.Amount = 15.0
	err := repo.Update(ctx, commission)
	require.NoError(t, err)

	var found models.Commission
	db.First(&found, commission.ID)
	assert.Equal(t, 15.0, found.Amount)
}

func TestCommissionRepository_UpdateStatus(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	commission := &models.Commission{
		DistributorID: 1,
		OrderID:       1,
		FromUserID:    1,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100.0,
		Rate:          0.1,
		Amount:        10.0,
		Status:        models.CommissionStatusPending,
	}
	db.Create(commission)

	err := repo.UpdateStatus(ctx, commission.ID, models.CommissionStatusSettled)
	require.NoError(t, err)

	var found models.Commission
	db.First(&found, commission.ID)
	assert.Equal(t, models.CommissionStatusSettled, found.Status)
}

func TestCommissionRepository_Settle(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	commission := &models.Commission{
		DistributorID: 1,
		OrderID:       1,
		FromUserID:    1,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100.0,
		Rate:          0.1,
		Amount:        10.0,
		Status:        models.CommissionStatusPending,
	}
	db.Create(commission)

	err := repo.Settle(ctx, commission.ID)
	require.NoError(t, err)

	var found models.Commission
	db.First(&found, commission.ID)
	assert.Equal(t, models.CommissionStatusSettled, found.Status)
	assert.NotNil(t, found.SettledAt)
}

func TestCommissionRepository_CancelByOrderID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusPending,
	})

	db.Create(&models.Commission{
		DistributorID: 2, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeIndirect,
		OrderAmount: 100.0, Rate: 0.05, Amount: 5.0, Status: models.CommissionStatusPending,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusSettled,
	})

	err := repo.CancelByOrderID(ctx, 1)
	require.NoError(t, err)

	var cancelled int64
	db.Model(&models.Commission{}).Where("order_id = ? AND status = ?", 1, models.CommissionStatusCancelled).Count(&cancelled)
	assert.Equal(t, int64(2), cancelled) // 只取消待结算的
}

func TestCommissionRepository_List(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusPending,
	})

	db.Create(&models.Commission{
		DistributorID: 2, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeIndirect,
		OrderAmount: 200.0, Rate: 0.05, Amount: 10.0, Status: models.CommissionStatusSettled,
	})

	db.Model(&models.Commission{}).Create(map[string]interface{}{
		"distributor_id": 1,
		"order_id":       3,
		"from_user_id":   1,
		"type":           models.CommissionTypeDirect,
		"order_amount":   150.0,
		"rate":           0.1,
		"amount":         15.0,
		"status":         models.CommissionStatusCancelled,
	})

	// 获取所有佣金
	_, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按分销商过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"distributor_id": int64(1),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按状态过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": models.CommissionStatusPending,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按类型过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"type": models.CommissionTypeDirect,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestCommissionRepository_GetPendingByDistributorID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusPending,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 200.0, Rate: 0.1, Amount: 20.0, Status: models.CommissionStatusSettled,
	})

	db.Model(&models.Commission{}).Create(map[string]interface{}{
		"distributor_id": 1,
		"order_id":       3,
		"from_user_id":   1,
		"type":           models.CommissionTypeDirect,
		"order_amount":   150.0,
		"rate":           0.1,
		"amount":         15.0,
		"status":         models.CommissionStatusPending,
	})

	list, err := repo.GetPendingByDistributorID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list)) // 只返回待结算的
}

func TestCommissionRepository_GetSettledByDistributorID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusSettled,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 200.0, Rate: 0.1, Amount: 20.0, Status: models.CommissionStatusPending,
	})

	list, total, err := repo.GetSettledByDistributorID(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total) // 只返回已结算的
	assert.Equal(t, 1, len(list))
}

func TestCommissionRepository_SumByDistributorID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusPending,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 200.0, Rate: 0.1, Amount: 20.0, Status: models.CommissionStatusSettled,
	})

	db.Create(&models.Commission{
		DistributorID: 2, OrderID: 3, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 150.0, Rate: 0.1, Amount: 15.0, Status: models.CommissionStatusSettled,
	})

	// 统计所有状态
	sum, err := repo.SumByDistributorID(ctx, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, 30.0, sum)

	// 统计待结算
	pendingStatus := models.CommissionStatusPending
	sum, err = repo.SumByDistributorID(ctx, 1, &pendingStatus)
	require.NoError(t, err)
	assert.Equal(t, 10.0, sum)

	// 统计已结算
	settledStatus := models.CommissionStatusSettled
	sum, err = repo.SumByDistributorID(ctx, 1, &settledStatus)
	require.NoError(t, err)
	assert.Equal(t, 20.0, sum)
}

func TestCommissionRepository_SumByDistributorIDAndType(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusSettled,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeIndirect,
		OrderAmount: 200.0, Rate: 0.05, Amount: 10.0, Status: models.CommissionStatusSettled,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 3, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 150.0, Rate: 0.1, Amount: 15.0, Status: models.CommissionStatusPending,
	})

	sum, err := repo.SumByDistributorIDAndType(ctx, 1, models.CommissionTypeDirect)
	require.NoError(t, err)
	assert.Equal(t, 10.0, sum) // 只统计已结算的直推佣金
}

func TestCommissionRepository_CountByDistributorID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusPending,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 200.0, Rate: 0.1, Amount: 20.0, Status: models.CommissionStatusSettled,
	})

	// 统计所有状态
	count, err := repo.CountByDistributorID(ctx, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 统计待结算
	pendingStatus := models.CommissionStatusPending
	count, err = repo.CountByDistributorID(ctx, 1, &pendingStatus)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestCommissionRepository_GetStatsByDistributorID(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 1, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 100.0, Rate: 0.1, Amount: 10.0, Status: models.CommissionStatusPending,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 200.0, Rate: 0.1, Amount: 20.0, Status: models.CommissionStatusSettled,
	})

	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 3, FromUserID: 1, Type: models.CommissionTypeIndirect,
		OrderAmount: 150.0, Rate: 0.05, Amount: 7.5, Status: models.CommissionStatusSettled,
	})

	stats, err := repo.GetStatsByDistributorID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 37.5, stats["total_amount"])
	assert.Equal(t, 10.0, stats["pending_amount"])
	assert.Equal(t, 27.5, stats["settled_amount"])
	assert.Equal(t, int64(3), stats["total_count"])
	assert.Equal(t, 30.0, stats["direct_amount"])
	assert.Equal(t, 7.5, stats["indirect_amount"])
}

func TestCommissionRepository_SettlePendingByTime(t *testing.T) {
	db := setupCommissionTestDB(t)
	repo := NewCommissionRepository(db)
	ctx := context.Background()

	// 创建旧的待结算佣金
	db.Exec("INSERT INTO commissions (distributor_id, order_id, from_user_id, type, order_amount, rate, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		1, 1, 1, models.CommissionTypeDirect, 100.0, 0.1, 10.0, models.CommissionStatusPending, time.Now().Add(-2*time.Hour))

	// 创建新的待结算佣金
	db.Create(&models.Commission{
		DistributorID: 1, OrderID: 2, FromUserID: 1, Type: models.CommissionTypeDirect,
		OrderAmount: 200.0, Rate: 0.1, Amount: 20.0, Status: models.CommissionStatusPending,
	})

	beforeTime := time.Now().Add(-1 * time.Hour)
	affected, err := repo.SettlePendingByTime(ctx, beforeTime)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected) // 只结算1小时前的

	var settled int64
	db.Model(&models.Commission{}).Where("status = ?", models.CommissionStatusSettled).Count(&settled)
	assert.Equal(t, int64(1), settled)
}
