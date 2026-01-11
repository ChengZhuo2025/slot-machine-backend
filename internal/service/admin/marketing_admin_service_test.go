package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupMarketingAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(&models.Coupon{}, &models.Campaign{}))
	return db
}

func TestMarketingAdminService_CreateAndListCoupon(t *testing.T) {
	db := setupMarketingAdminTestDB(t)
	svc := NewMarketingAdminService(db, repository.NewCouponRepository(db), repository.NewCampaignRepository(db))
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-time.Hour).Format("2006-01-02 15:04:05")
	end := now.Add(time.Hour).Format("2006-01-02 15:04:05")

	coupon, err := svc.CreateCoupon(ctx, &CreateCouponRequest{
		Name:            "满10减5",
		Type:            models.CouponTypeFixed,
		Value:           5,
		MinAmount:       10,
		TotalCount:      100,
		PerUserLimit:    1,
		ApplicableScope: models.CouponScopeProduct,
		ApplicableIDs:   []int64{1, 2},
		StartTime:       start,
		EndTime:         end,
	})
	require.NoError(t, err)
	require.NotNil(t, coupon)

	resp, err := svc.GetCouponList(ctx, &AdminCouponListRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, int64(1), resp.Total)
	require.Len(t, resp.List, 1)
	assert.Equal(t, coupon.ID, resp.List[0].ID)
	assert.Equal(t, "满10减5", resp.List[0].Name)
	assert.Equal(t, "固定金额", resp.List[0].TypeText)
	assert.ElementsMatch(t, []int64{1, 2}, resp.List[0].ApplicableIDs)
}

func TestMarketingAdminService_GetCouponDetail(t *testing.T) {
	db := setupMarketingAdminTestDB(t)
	svc := NewMarketingAdminService(db, repository.NewCouponRepository(db), repository.NewCampaignRepository(db))
	ctx := context.Background()

	now := time.Now()
	coupon := &models.Coupon{
		Name:            "测试优惠券",
		Type:            models.CouponTypeFixed,
		Value:           10,
		MinAmount:       20,
		TotalCount:      100,
		ReceivedCount:   5,
		UsedCount:       2,
		PerUserLimit:    1,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       now.Add(-time.Hour),
		EndTime:         now.Add(time.Hour),
		Status:          models.CouponStatusActive,
	}
	require.NoError(t, db.Create(coupon).Error)

	detail, err := svc.GetCouponDetail(ctx, coupon.ID)
	require.NoError(t, err)
	assert.Equal(t, coupon.ID, detail.ID)
	assert.Equal(t, "测试优惠券", detail.Name)

	t.Run("优惠券不存在", func(t *testing.T) {
		_, err := svc.GetCouponDetail(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestMarketingAdminService_UpdateCoupon(t *testing.T) {
	db := setupMarketingAdminTestDB(t)
	svc := NewMarketingAdminService(db, repository.NewCouponRepository(db), repository.NewCampaignRepository(db))
	ctx := context.Background()

	now := time.Now()
	coupon := &models.Coupon{
		Name:            "原优惠券",
		Type:            models.CouponTypeFixed,
		Value:           10,
		MinAmount:       20,
		TotalCount:      100,
		PerUserLimit:    1,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       now.Add(-time.Hour),
		EndTime:         now.Add(time.Hour),
		Status:          models.CouponStatusActive,
	}
	require.NoError(t, db.Create(coupon).Error)

	newName := "更新后优惠券"
	newValue := 20.0
	err := svc.UpdateCoupon(ctx, coupon.ID, &UpdateCouponRequest{
		Name:  &newName,
		Value: &newValue,
	})
	require.NoError(t, err)

	var updated models.Coupon
	db.First(&updated, coupon.ID)
	assert.Equal(t, "更新后优惠券", updated.Name)
	assert.Equal(t, 20.0, updated.Value)
}

func TestMarketingAdminService_UpdateCouponStatus(t *testing.T) {
	db := setupMarketingAdminTestDB(t)
	svc := NewMarketingAdminService(db, repository.NewCouponRepository(db), repository.NewCampaignRepository(db))
	ctx := context.Background()

	now := time.Now()
	coupon := &models.Coupon{
		Name:            "状态测试",
		Type:            models.CouponTypeFixed,
		Value:           10,
		MinAmount:       20,
		TotalCount:      100,
		PerUserLimit:    1,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       now.Add(-time.Hour),
		EndTime:         now.Add(time.Hour),
		Status:          models.CouponStatusActive,
	}
	require.NoError(t, db.Create(coupon).Error)

	err := svc.UpdateCouponStatus(ctx, coupon.ID, models.CouponStatusDisabled)
	require.NoError(t, err)

	var updated models.Coupon
	db.First(&updated, coupon.ID)
	assert.Equal(t, int8(models.CouponStatusDisabled), updated.Status)
}

func TestMarketingAdminService_DeleteCoupon(t *testing.T) {
	db := setupMarketingAdminTestDB(t)
	svc := NewMarketingAdminService(db, repository.NewCouponRepository(db), repository.NewCampaignRepository(db))
	ctx := context.Background()

	now := time.Now()
	coupon := &models.Coupon{
		Name:            "删除测试",
		Type:            models.CouponTypeFixed,
		Value:           10,
		MinAmount:       20,
		TotalCount:      100,
		PerUserLimit:    1,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       now.Add(-time.Hour),
		EndTime:         now.Add(time.Hour),
		Status:          models.CouponStatusActive,
	}
	require.NoError(t, db.Create(coupon).Error)

	err := svc.DeleteCoupon(ctx, coupon.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Coupon{}).Where("id = ?", coupon.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestMarketingAdminService_CampaignCRUD(t *testing.T) {
	db := setupMarketingAdminTestDB(t)
	svc := NewMarketingAdminService(db, repository.NewCouponRepository(db), repository.NewCampaignRepository(db))
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-time.Hour).Format("2006-01-02 15:04:05")
	end := now.Add(time.Hour).Format("2006-01-02 15:04:05")
	desc := "测试活动描述"

	t.Run("创建活动", func(t *testing.T) {
		campaign, err := svc.CreateCampaign(ctx, &CreateCampaignRequest{
			Name:        "测试活动",
			Type:        "discount",
			Description: &desc,
			StartTime:   start,
			EndTime:     end,
		})
		require.NoError(t, err)
		assert.NotNil(t, campaign)
		assert.Equal(t, "测试活动", campaign.Name)
	})

	t.Run("获取活动列表", func(t *testing.T) {
		resp, err := svc.GetCampaignList(ctx, &AdminCampaignListRequest{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.True(t, resp.Total >= 1)
		assert.NotEmpty(t, resp.List)
	})

	t.Run("获取活动详情", func(t *testing.T) {
		campaign := &models.Campaign{
			Name:        "详情测试活动",
			Type:        "discount",
			Description: &desc,
			StartTime:   now.Add(-time.Hour),
			EndTime:     now.Add(time.Hour),
			Status:      models.CampaignStatusActive,
		}
		require.NoError(t, db.Create(campaign).Error)

		detail, err := svc.GetCampaignDetail(ctx, campaign.ID)
		require.NoError(t, err)
		assert.Equal(t, campaign.ID, detail.ID)
		assert.Equal(t, "详情测试活动", detail.Name)
	})

	t.Run("更新活动", func(t *testing.T) {
		campaign := &models.Campaign{
			Name:        "更新测试活动",
			Type:        "discount",
			Description: &desc,
			StartTime:   now.Add(-time.Hour),
			EndTime:     now.Add(time.Hour),
			Status:      models.CampaignStatusActive,
		}
		require.NoError(t, db.Create(campaign).Error)

		newName := "已更新活动"
		err := svc.UpdateCampaign(ctx, campaign.ID, &UpdateCampaignRequest{
			Name: &newName,
		})
		require.NoError(t, err)

		var updated models.Campaign
		db.First(&updated, campaign.ID)
		assert.Equal(t, "已更新活动", updated.Name)
	})

	t.Run("更新活动状态", func(t *testing.T) {
		campaign := &models.Campaign{
			Name:        "状态测试活动",
			Type:        "discount",
			Description: &desc,
			StartTime:   now.Add(-time.Hour),
			EndTime:     now.Add(time.Hour),
			Status:      models.CampaignStatusActive,
		}
		require.NoError(t, db.Create(campaign).Error)

		err := svc.UpdateCampaignStatus(ctx, campaign.ID, models.CampaignStatusDisabled)
		require.NoError(t, err)

		var updated models.Campaign
		db.First(&updated, campaign.ID)
		assert.Equal(t, int8(models.CampaignStatusDisabled), updated.Status)
	})

	t.Run("删除活动", func(t *testing.T) {
		campaign := &models.Campaign{
			Name:        "删除测试活动",
			Type:        "discount",
			Description: &desc,
			StartTime:   now.Add(-time.Hour),
			EndTime:     now.Add(time.Hour),
			Status:      models.CampaignStatusActive,
		}
		require.NoError(t, db.Create(campaign).Error)

		err := svc.DeleteCampaign(ctx, campaign.ID)
		require.NoError(t, err)

		var count int64
		db.Model(&models.Campaign{}).Where("id = ?", campaign.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

