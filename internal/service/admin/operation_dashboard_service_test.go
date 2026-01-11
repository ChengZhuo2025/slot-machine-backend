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
)

func setupOperationDashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.MemberLevel{},
		&models.Order{},
		&models.Coupon{},
		&models.UserCoupon{},
		&models.Campaign{},
		&models.Distributor{},
		&models.Commission{},
		&models.Article{},
		&models.Banner{},
	))
	return db
}

func TestOperationDashboardService_GetOperationOverview(t *testing.T) {
	db := setupOperationDashboardTestDB(t)
	svc := NewOperationDashboardService(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1}).Error)
	require.NoError(t, db.Create(&models.MemberLevel{ID: 2, Name: "VIP", Level: 2, MinPoints: 100, Discount: 0.9}).Error)

	u1 := &models.User{Nickname: "U1", MemberLevelID: 1, Status: models.UserStatusActive}
	u2 := &models.User{Nickname: "U2", MemberLevelID: 2, Status: models.UserStatusActive}
	require.NoError(t, db.Create(u1).Error)
	require.NoError(t, db.Create(u2).Error)

	memberOrder := &models.Order{
		OrderNo:         "O_MEMBER",
		UserID:          u2.ID,
		Type:            "member",
		OriginalAmount:  100,
		DiscountAmount:  0,
		ActualAmount:    100,
		DepositAmount:   0,
		Status:          models.OrderStatusCompleted,
	}
	require.NoError(t, db.Create(memberOrder).Error)

	now := time.Now()
	coupon := &models.Coupon{
		Name:            "券",
		Type:            models.CouponTypeFixed,
		Value:           10,
		MinAmount:       0,
		TotalCount:      10,
		ReceivedCount:   1,
		UsedCount:       1,
		PerUserLimit:    1,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       now.Add(-time.Hour),
		EndTime:         now.Add(time.Hour),
		Status:          models.CouponStatusActive,
	}
	require.NoError(t, db.Create(coupon).Error)

	usedAt := now
	userCoupon := &models.UserCoupon{
		UserID:     u1.ID,
		CouponID:   coupon.ID,
		Status:     models.UserCouponStatusUsed,
		ExpiredAt:  now.Add(time.Hour),
		UsedAt:     &usedAt,
		ReceivedAt: now.Add(-time.Hour),
	}
	require.NoError(t, db.Create(userCoupon).Error)

	campaign := &models.Campaign{
		Name:      "活动",
		Type:      models.CampaignTypeDiscount,
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
		Status:    models.CampaignStatusActive,
	}
	require.NoError(t, db.Create(campaign).Error)

	approved := &models.Distributor{UserID: u1.ID, Level: models.DistributorLevelDirect, InviteCode: "I1", Status: models.DistributorStatusApproved}
	pending := &models.Distributor{UserID: u2.ID, Level: models.DistributorLevelDirect, InviteCode: "I2", Status: models.DistributorStatusPending}
	require.NoError(t, db.Create(approved).Error)
	require.NoError(t, db.Create(pending).Error)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	commission := &models.Commission{
		DistributorID: approved.ID,
		OrderID:       memberOrder.ID,
		FromUserID:    u2.ID,
		Type:          models.CommissionTypeDirect,
		OrderAmount:   100,
		Rate:          0.1,
		Amount:        5,
		Status:        models.CommissionStatusSettled,
		SettledAt:     &monthStart,
	}
	require.NoError(t, db.Create(commission).Error)

	publishedAt := now
	article := &models.Article{
		Category:    models.ArticleCategoryNotice,
		Title:       "公告",
		Content:     "内容",
		IsPublished: true,
		PublishedAt: &publishedAt,
	}
	require.NoError(t, db.Create(article).Error)

	banner := &models.Banner{
		Title:    "轮播",
		Image:    "img",
		Position: models.BannerPositionHome,
		IsActive: true,
	}
	require.NoError(t, db.Create(banner).Error)

	overview, err := svc.GetOperationOverview(ctx)
	require.NoError(t, err)
	require.NotNil(t, overview)

	assert.Equal(t, int64(2), overview.TotalUsers)
	assert.Equal(t, int64(1), overview.TotalMembers)
	assert.Equal(t, int64(1), overview.ActiveCoupons)
	assert.Equal(t, int64(1), overview.UsedCoupons)
	assert.Equal(t, int64(1), overview.TodayUsedCoupons)
	assert.Equal(t, int64(1), overview.ActiveCampaigns)
	assert.Equal(t, int64(1), overview.TotalDistributors)
	assert.Equal(t, int64(1), overview.PendingDistributors)
	assert.Equal(t, float64(5), overview.MonthCommission)
	assert.Equal(t, int64(1), overview.TotalArticles)
	assert.Equal(t, int64(1), overview.PublishedArticles)
	assert.Equal(t, int64(1), overview.TotalBanners)
	assert.Equal(t, int64(1), overview.ActiveBanners)
}

func TestOperationDashboardService_GetUserGrowthTrend_Bounds(t *testing.T) {
	db := setupOperationDashboardTestDB(t)
	svc := NewOperationDashboardService(db)
	ctx := context.Background()

	trends, err := svc.GetUserGrowthTrend(ctx, 0)
	require.NoError(t, err)
	require.Len(t, trends, 7)

	trends, err = svc.GetUserGrowthTrend(ctx, 999)
	require.NoError(t, err)
	require.Len(t, trends, 30)
}

func TestOperationDashboardService_GetCouponUsageStats(t *testing.T) {
	db := setupOperationDashboardTestDB(t)
	svc := NewOperationDashboardService(db)
	ctx := context.Background()

	now := time.Now()
	coupon := &models.Coupon{
		Name:            "测试券",
		Type:            models.CouponTypeFixed,
		Value:           10,
		MinAmount:       0,
		TotalCount:      100,
		ReceivedCount:   10,
		UsedCount:       5,
		PerUserLimit:    1,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       now.Add(-time.Hour),
		EndTime:         now.Add(time.Hour),
		Status:          models.CouponStatusActive,
	}
	require.NoError(t, db.Create(coupon).Error)

	stats, err := svc.GetCouponUsageStats(ctx, 10)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestOperationDashboardService_GetMemberLevelDistribution(t *testing.T) {
	db := setupOperationDashboardTestDB(t)
	svc := NewOperationDashboardService(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1}).Error)
	require.NoError(t, db.Create(&models.MemberLevel{ID: 2, Name: "VIP", Level: 2, MinPoints: 100, Discount: 0.9}).Error)

	u1 := &models.User{Nickname: "U1", MemberLevelID: 1, Status: models.UserStatusActive}
	u2 := &models.User{Nickname: "U2", MemberLevelID: 2, Status: models.UserStatusActive}
	require.NoError(t, db.Create(u1).Error)
	require.NoError(t, db.Create(u2).Error)

	dist, err := svc.GetMemberLevelDistribution(ctx)
	require.NoError(t, err)
	assert.NotNil(t, dist)
}

func TestOperationDashboardService_GetDistributorRank(t *testing.T) {
	db := setupOperationDashboardTestDB(t)
	svc := NewOperationDashboardService(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1}).Error)

	u1 := &models.User{Nickname: "U1", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, db.Create(u1).Error)

	distributor := &models.Distributor{
		UserID:          u1.ID,
		Level:           models.DistributorLevelDirect,
		InviteCode:      "RANK01",
		Status:          models.DistributorStatusApproved,
		TotalCommission: 100.0,
		TeamCount:       5,
	}
	require.NoError(t, db.Create(distributor).Error)

	rank, err := svc.GetDistributorRank(ctx, 10)
	require.NoError(t, err)
	assert.NotNil(t, rank)
}

func TestOperationDashboardService_GetActiveCampaigns(t *testing.T) {
	db := setupOperationDashboardTestDB(t)
	svc := NewOperationDashboardService(db)
	ctx := context.Background()

	now := time.Now()
	campaign := &models.Campaign{
		Name:      "活动测试",
		Type:      models.CampaignTypeDiscount,
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
		Status:    models.CampaignStatusActive,
	}
	require.NoError(t, db.Create(campaign).Error)

	campaigns, err := svc.GetActiveCampaigns(ctx, 10)
	require.NoError(t, err)
	assert.NotNil(t, campaigns)
}

func TestOperationDashboardService_GetUserFeedbackStats(t *testing.T) {
	db := setupOperationDashboardTestDB(t)
	svc := NewOperationDashboardService(db)
	ctx := context.Background()

	stats, err := svc.GetUserFeedbackStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

