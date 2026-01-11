// Package repository 活动仓储单元测试
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

func setupCampaignTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Campaign{})
	require.NoError(t, err)

	return db
}

func TestCampaignRepository_Create(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	campaign := &models.Campaign{
		Name:      "测试活动",
		Type:      models.CampaignTypeDiscount,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(24 * time.Hour),
		Status:    models.CampaignStatusActive,
	}

	err := repo.Create(ctx, campaign)
	require.NoError(t, err)
	assert.NotZero(t, campaign.ID)
}

func TestCampaignRepository_GetByID(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	campaign := &models.Campaign{
		Name:      "测试活动",
		Type:      models.CampaignTypeDiscount,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(24 * time.Hour),
		Status:    models.CampaignStatusActive,
	}
	db.Create(campaign)

	found, err := repo.GetByID(ctx, campaign.ID)
	require.NoError(t, err)
	assert.Equal(t, campaign.ID, found.ID)
	assert.Equal(t, "测试活动", found.Name)
}

func TestCampaignRepository_Update(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	campaign := &models.Campaign{
		Name:      "原活动",
		Type:      models.CampaignTypeDiscount,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(24 * time.Hour),
		Status:    models.CampaignStatusActive,
	}
	db.Create(campaign)

	campaign.Name = "新活动"
	err := repo.Update(ctx, campaign)
	require.NoError(t, err)

	var found models.Campaign
	db.First(&found, campaign.ID)
	assert.Equal(t, "新活动", found.Name)
}

func TestCampaignRepository_UpdateFields(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	campaign := &models.Campaign{
		Name:      "测试活动",
		Type:      models.CampaignTypeDiscount,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(24 * time.Hour),
		Status:    models.CampaignStatusActive,
	}
	db.Create(campaign)

	err := repo.UpdateFields(ctx, campaign.ID, map[string]interface{}{
		"name": "更新后的活动",
	})
	require.NoError(t, err)

	var found models.Campaign
	db.First(&found, campaign.ID)
	assert.Equal(t, "更新后的活动", found.Name)
}

func TestCampaignRepository_Delete(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	campaign := &models.Campaign{
		Name:      "待删除活动",
		Type:      models.CampaignTypeDiscount,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(24 * time.Hour),
		Status:    models.CampaignStatusActive,
	}
	db.Create(campaign)

	err := repo.Delete(ctx, campaign.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Campaign{}).Where("id = ?", campaign.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCampaignRepository_List(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	now := time.Now()
	db.Create(&models.Campaign{
		Name: "活动1", Type: models.CampaignTypeDiscount, StartTime: now, EndTime: now.Add(24 * time.Hour), Status: models.CampaignStatusActive,
	})

	db.Model(&models.Campaign{}).Create(map[string]interface{}{
		"name": "活动2", "type": models.CampaignTypeGift,
		"start_time": now, "end_time": now.Add(24 * time.Hour),
		"status": models.CampaignStatusDisabled,
	})

	list, total, err := repo.List(ctx, CampaignListParams{Offset: 0, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))

	status := int8(models.CampaignStatusActive)
	list, total, err = repo.List(ctx, CampaignListParams{Offset: 0, Limit: 10, Status: &status})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}

func TestCampaignRepository_ListActive(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	now := time.Now()
	// 活跃活动 - 正在进行中
	db.Create(&models.Campaign{
		Name: "活跃活动", Type: models.CampaignTypeDiscount,
		StartTime: now.Add(-1 * time.Hour), EndTime: now.Add(1 * time.Hour),
		Status: models.CampaignStatusActive,
	})

	// 未开始
	db.Create(&models.Campaign{
		Name: "未开始活动", Type: models.CampaignTypeDiscount,
		StartTime: now.Add(1 * time.Hour), EndTime: now.Add(3 * time.Hour),
		Status: models.CampaignStatusActive,
	})

	list, total, err := repo.ListActive(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}

func TestCampaignRepository_ListByType(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	now := time.Now()
	campaigns := []*models.Campaign{
		{Name: "打折活动", Type: models.CampaignTypeDiscount, StartTime: now.Add(-1 * time.Hour), EndTime: now.Add(1 * time.Hour), Status: models.CampaignStatusActive},
		{Name: "赠品活动", Type: models.CampaignTypeGift, StartTime: now.Add(-1 * time.Hour), EndTime: now.Add(1 * time.Hour), Status: models.CampaignStatusActive},
	}
	for _, c := range campaigns {
		db.Create(c)
	}

	list, err := repo.ListByType(ctx, models.CampaignTypeDiscount)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))
}

func TestCampaignRepository_UpdateStatus(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	campaign := &models.Campaign{
		Name: "测试活动", Type: models.CampaignTypeDiscount,
		StartTime: time.Now(), EndTime: time.Now().Add(24 * time.Hour),
		Status: models.CampaignStatusDisabled,
	}
	db.Create(campaign)

	err := repo.UpdateStatus(ctx, campaign.ID, models.CampaignStatusActive)
	require.NoError(t, err)

	var found models.Campaign
	db.First(&found, campaign.ID)
	assert.Equal(t, int8(models.CampaignStatusActive), found.Status)
}

func TestCampaignRepository_CountByStatus(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	now := time.Now()
	db.Create(&models.Campaign{
		Name: "活动1", Type: models.CampaignTypeDiscount, StartTime: now, EndTime: now.Add(24 * time.Hour), Status: models.CampaignStatusActive,
	})
	db.Create(&models.Campaign{
		Name: "活动2", Type: models.CampaignTypeGift, StartTime: now, EndTime: now.Add(24 * time.Hour), Status: models.CampaignStatusActive,
	})

	db.Model(&models.Campaign{}).Create(map[string]interface{}{
		"name": "活动3", "type": models.CampaignTypeDiscount,
		"start_time": now, "end_time": now.Add(24 * time.Hour),
		"status": models.CampaignStatusDisabled,
	})

	count, err := repo.CountByStatus(ctx, models.CampaignStatusActive)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestCampaignRepository_CountActive(t *testing.T) {
	db := setupCampaignTestDB(t)
	repo := NewCampaignRepository(db)
	ctx := context.Background()

	now := time.Now()
	// 进行中
	db.Create(&models.Campaign{
		Name: "进行中", Type: models.CampaignTypeDiscount,
		StartTime: now.Add(-1 * time.Hour), EndTime: now.Add(1 * time.Hour),
		Status: models.CampaignStatusActive,
	})

	// 未开始
	db.Create(&models.Campaign{
		Name: "未开始", Type: models.CampaignTypeDiscount,
		StartTime: now.Add(1 * time.Hour), EndTime: now.Add(3 * time.Hour),
		Status: models.CampaignStatusActive,
	})

	count, err := repo.CountActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
