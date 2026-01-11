// Package repository 结算仓储单元测试
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

func setupSettlementTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Settlement{}, &models.Admin{})
	require.NoError(t, err)

	return db
}

const (
	SettlementTypeMerchant    = "merchant"
	SettlementTypeDistributor = "distributor"
)

func TestSettlementRepository_Create(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	settlement := &models.Settlement{
		SettlementNo: "S202601001",
		Type:         SettlementTypeMerchant,
		TargetID:     1,
		PeriodStart:  time.Now().AddDate(0, 0, -7),
		PeriodEnd:    time.Now(),
		TotalAmount:  1000.0,
		Fee:          10.0,
		ActualAmount: 990.0,
		OrderCount:   10,
		Status:       models.SettlementStatusPending,
	}

	err := repo.Create(ctx, settlement)
	require.NoError(t, err)
	assert.NotZero(t, settlement.ID)
}

func TestSettlementRepository_GetByID(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	settlement := &models.Settlement{
		SettlementNo: "S202601001",
		Type:         SettlementTypeMerchant,
		TargetID:     1,
		PeriodStart:  time.Now().AddDate(0, 0, -7),
		PeriodEnd:    time.Now(),
		TotalAmount:  1000.0,
		Fee:          10.0,
		ActualAmount: 990.0,
		OrderCount:   10,
		Status:       models.SettlementStatusPending,
	}
	db.Create(settlement)

	found, err := repo.GetByID(ctx, settlement.ID)
	require.NoError(t, err)
	assert.Equal(t, settlement.ID, found.ID)
}

func TestSettlementRepository_GetBySettlementNo(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	db.Create(&models.Settlement{
		SettlementNo: "S202601001",
		Type:         SettlementTypeMerchant,
		TargetID:     1,
		PeriodStart:  time.Now().AddDate(0, 0, -7),
		PeriodEnd:    time.Now(),
		TotalAmount:  1000.0,
		Fee:          10.0,
		ActualAmount: 990.0,
		OrderCount:   10,
		Status:       models.SettlementStatusPending,
	})

	found, err := repo.GetBySettlementNo(ctx, "S202601001")
	require.NoError(t, err)
	assert.Equal(t, "S202601001", found.SettlementNo)
}

func TestSettlementRepository_Update(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	settlement := &models.Settlement{
		SettlementNo: "S202601001",
		Type:         SettlementTypeMerchant,
		TargetID:     1,
		PeriodStart:  time.Now().AddDate(0, 0, -7),
		PeriodEnd:    time.Now(),
		TotalAmount:  1000.0,
		Fee:          10.0,
		ActualAmount: 990.0,
		OrderCount:   10,
		Status:       models.SettlementStatusPending,
	}
	db.Create(settlement)

	settlement.Status = models.SettlementStatusCompleted
	err := repo.Update(ctx, settlement)
	require.NoError(t, err)

	var found models.Settlement
	db.First(&found, settlement.ID)
	assert.Equal(t, models.SettlementStatusCompleted, found.Status)
}

func TestSettlementRepository_UpdateStatus(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	settlement := &models.Settlement{
		SettlementNo: "S202601001",
		Type:         SettlementTypeMerchant,
		TargetID:     1,
		PeriodStart:  time.Now().AddDate(0, 0, -7),
		PeriodEnd:    time.Now(),
		TotalAmount:  1000.0,
		Fee:          10.0,
		ActualAmount: 990.0,
		OrderCount:   10,
		Status:       models.SettlementStatusPending,
	}
	db.Create(settlement)

	operatorID := int64(100)
	err := repo.UpdateStatus(ctx, settlement.ID, models.SettlementStatusCompleted, &operatorID)
	require.NoError(t, err)

	var found models.Settlement
	db.First(&found, settlement.ID)
	assert.Equal(t, models.SettlementStatusCompleted, found.Status)
	assert.NotNil(t, found.OperatorID)
	assert.Equal(t, int64(100), *found.OperatorID)
	assert.NotNil(t, found.SettledAt)
}

func TestSettlementRepository_List(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	db.Create(&models.Settlement{
		SettlementNo: "S001", Type: SettlementTypeMerchant, TargetID: 1,
		PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
		TotalAmount: 1000.0, Fee: 10.0, ActualAmount: 990.0, OrderCount: 10,
		Status: models.SettlementStatusPending,
	})

	db.Create(&models.Settlement{
		SettlementNo: "S002", Type: SettlementTypeDistributor, TargetID: 2,
		PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
		TotalAmount: 500.0, Fee: 5.0, ActualAmount: 495.0, OrderCount: 5,
		Status: models.SettlementStatusCompleted,
	})

	// 获取所有结算
	_, total, err := repo.List(ctx, nil, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按类型过滤
	filter := &SettlementFilter{Type: SettlementTypeMerchant}
	_, total, err = repo.List(ctx, filter, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按状态过滤
	filter = &SettlementFilter{Status: models.SettlementStatusPending}
	_, total, err = repo.List(ctx, filter, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestSettlementRepository_ListByTarget(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	db.Create(&models.Settlement{
		SettlementNo: "S001", Type: SettlementTypeMerchant, TargetID: 1,
		PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
		TotalAmount: 1000.0, Fee: 10.0, ActualAmount: 990.0, OrderCount: 10,
		Status: models.SettlementStatusPending,
	})

	db.Create(&models.Settlement{
		SettlementNo: "S002", Type: SettlementTypeMerchant, TargetID: 2,
		PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
		TotalAmount: 500.0, Fee: 5.0, ActualAmount: 495.0, OrderCount: 5,
		Status: models.SettlementStatusPending,
	})

	_, total, err := repo.ListByTarget(ctx, SettlementTypeMerchant, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestSettlementRepository_GetPendingSettlements(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	db.Create(&models.Settlement{
		SettlementNo: "S001", Type: SettlementTypeMerchant, TargetID: 1,
		PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
		TotalAmount: 1000.0, Fee: 10.0, ActualAmount: 990.0, OrderCount: 10,
		Status: models.SettlementStatusPending,
	})

	db.Create(&models.Settlement{
		SettlementNo: "S002", Type: SettlementTypeMerchant, TargetID: 2,
		PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
		TotalAmount: 500.0, Fee: 5.0, ActualAmount: 495.0, OrderCount: 5,
		Status: models.SettlementStatusCompleted,
	})

	settlements, err := repo.GetPendingSettlements(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, 1, len(settlements))
}

func TestSettlementRepository_CountPending(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	db.Create(&models.Settlement{
		SettlementNo: "S001", Type: SettlementTypeMerchant, TargetID: 1,
		PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
		TotalAmount: 1000.0, Fee: 10.0, ActualAmount: 990.0, OrderCount: 10,
		Status: models.SettlementStatusPending,
	})

	db.Create(&models.Settlement{
		SettlementNo: "S002", Type: SettlementTypeDistributor, TargetID: 2,
		PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
		TotalAmount: 500.0, Fee: 5.0, ActualAmount: 495.0, OrderCount: 5,
		Status: models.SettlementStatusCompleted,
	})

	count, err := repo.CountPending(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestSettlementRepository_ExistsForPeriod(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	periodStart := time.Now().AddDate(0, 0, -7)
	periodEnd := time.Now()

	db.Create(&models.Settlement{
		SettlementNo: "S001", Type: SettlementTypeMerchant, TargetID: 1,
		PeriodStart: periodStart, PeriodEnd: periodEnd,
		TotalAmount: 1000.0, Fee: 10.0, ActualAmount: 990.0, OrderCount: 10,
		Status: models.SettlementStatusPending,
	})

	exists, err := repo.ExistsForPeriod(ctx, SettlementTypeMerchant, 1, periodStart, periodEnd)
	require.NoError(t, err)
	assert.True(t, exists)

	otherStart := time.Now().AddDate(0, 0, -14)
	otherEnd := time.Now().AddDate(0, 0, -8)
	exists, err = repo.ExistsForPeriod(ctx, SettlementTypeMerchant, 1, otherStart, otherEnd)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSettlementRepository_BatchCreate(t *testing.T) {
	db := setupSettlementTestDB(t)
	repo := NewSettlementRepository(db)
	ctx := context.Background()

	settlements := []*models.Settlement{
		{
			SettlementNo: "S001", Type: SettlementTypeMerchant, TargetID: 1,
			PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
			TotalAmount: 1000.0, Fee: 10.0, ActualAmount: 990.0, OrderCount: 10,
			Status: models.SettlementStatusPending,
		},
		{
			SettlementNo: "S002", Type: SettlementTypeMerchant, TargetID: 2,
			PeriodStart: time.Now().AddDate(0, 0, -7), PeriodEnd: time.Now(),
			TotalAmount: 500.0, Fee: 5.0, ActualAmount: 495.0, OrderCount: 5,
			Status: models.SettlementStatusPending,
		},
	}

	err := repo.BatchCreate(ctx, settlements)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Settlement{}).Count(&count)
	assert.Equal(t, int64(2), count)
}
