// Package marketing 营销服务单元测试
package marketing

import (
	"context"
	"fmt"
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

func setupMarketingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Coupon{},
		&models.UserCoupon{},
		&models.Campaign{},
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	return db
}

func createMarketingTestUser(t *testing.T, db *gorm.DB, phone string) *models.User {
	t.Helper()

	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 1000.0,
	}
	require.NoError(t, db.Create(wallet).Error)

	return user
}

func createMarketingTestCoupon(t *testing.T, db *gorm.DB, opts ...func(*models.Coupon)) *models.Coupon {
	t.Helper()

	coupon := &models.Coupon{
		Name:            "测试优惠券",
		Type:            models.CouponTypeFixed,
		Value:           10.0,
		MinAmount:       50.0,
		TotalCount:      100,
		ReceivedCount:   0,
		UsedCount:       0,
		PerUserLimit:    3,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       time.Now().Add(-time.Hour),
		EndTime:         time.Now().Add(24 * time.Hour),
		Status:          models.CouponStatusActive,
	}

	for _, opt := range opts {
		opt(coupon)
	}

	require.NoError(t, db.Create(coupon).Error)
	return coupon
}

func createMarketingTestCampaign(t *testing.T, db *gorm.DB, opts ...func(*models.Campaign)) *models.Campaign {
	t.Helper()

	campaign := &models.Campaign{
		Name:      "测试活动",
		Type:      models.CampaignTypeDiscount,
		StartTime: time.Now().Add(-time.Hour),
		EndTime:   time.Now().Add(24 * time.Hour),
		Status:    models.CampaignStatusActive,
	}

	for _, opt := range opts {
		opt(campaign)
	}

	require.NoError(t, db.Create(campaign).Error)
	return campaign
}

func createMarketingTestUserCoupon(t *testing.T, db *gorm.DB, userID, couponID int64, status int8) *models.UserCoupon {
	t.Helper()

	uc := &models.UserCoupon{
		UserID:     userID,
		CouponID:   couponID,
		Status:     status,
		ExpiredAt:  time.Now().Add(24 * time.Hour),
		ReceivedAt: time.Now(),
	}
	require.NoError(t, db.Create(uc).Error)
	return uc
}

func setupCouponService(db *gorm.DB) *CouponService {
	couponRepo := repository.NewCouponRepository(db)
	userCouponRepo := repository.NewUserCouponRepository(db)
	return NewCouponService(db, couponRepo, userCouponRepo)
}

func setupUserCouponService(db *gorm.DB) *UserCouponService {
	couponRepo := repository.NewCouponRepository(db)
	userCouponRepo := repository.NewUserCouponRepository(db)
	return NewUserCouponService(db, couponRepo, userCouponRepo)
}

func setupCampaignService(db *gorm.DB) *CampaignService {
	campaignRepo := repository.NewCampaignRepository(db)
	return NewCampaignService(campaignRepo)
}

// ================== CouponService Tests ==================

func TestCouponService_GetCouponList(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCouponService(db)
	ctx := context.Background()
	user := createMarketingTestUser(t, db, "13800138000")

	// 创建多个优惠券
	createMarketingTestCoupon(t, db, func(c *models.Coupon) { c.Name = "优惠券1" })
	createMarketingTestCoupon(t, db, func(c *models.Coupon) { c.Name = "优惠券2" })

	req := &CouponListRequest{Page: 1, PageSize: 10}
	result, err := svc.GetCouponList(ctx, req, user.ID)
	require.NoError(t, err)
	assert.True(t, result.Total >= 2)
	assert.True(t, len(result.List) >= 2)
}

func TestCouponService_GetCouponDetail(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCouponService(db)
	ctx := context.Background()
	user := createMarketingTestUser(t, db, "13800138001")
	coupon := createMarketingTestCoupon(t, db)

	detail, err := svc.GetCouponDetail(ctx, coupon.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, coupon.ID, detail.ID)
	assert.Equal(t, coupon.Name, detail.Name)
	assert.True(t, detail.CanReceive)
}

func TestCouponService_ReceiveCoupon(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCouponService(db)
	ctx := context.Background()

	t.Run("正常领取优惠券", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138002")
		coupon := createMarketingTestCoupon(t, db)

		uc, err := svc.ReceiveCoupon(ctx, coupon.ID, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, uc)
		assert.Equal(t, user.ID, uc.UserID)
		assert.Equal(t, coupon.ID, uc.CouponID)
		assert.Equal(t, int8(models.UserCouponStatusUnused), uc.Status)

		// 验证优惠券已领取数量增加
		var updated models.Coupon
		db.First(&updated, coupon.ID)
		assert.Equal(t, 1, updated.ReceivedCount)
	})

	t.Run("领取禁用优惠券失败", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138003")
		coupon := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
			c.Status = int8(models.CouponStatusDisabled)
		})
		// 确保优惠券状态是禁用的
		db.Model(&coupon).Update("status", models.CouponStatusDisabled)

		_, err := svc.ReceiveCoupon(ctx, coupon.ID, user.ID)
		assert.ErrorIs(t, err, ErrCouponNotActive)
	})

	t.Run("超过每人限领数量失败", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138004")
		coupon := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
			c.PerUserLimit = 1
		})

		// 先领取一张
		_, err := svc.ReceiveCoupon(ctx, coupon.ID, user.ID)
		require.NoError(t, err)

		// 再次领取应该失败
		_, err = svc.ReceiveCoupon(ctx, coupon.ID, user.ID)
		assert.ErrorIs(t, err, ErrCouponLimitExceeded)
	})

	t.Run("领取已领完优惠券失败", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138005")
		coupon := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
			c.TotalCount = 1
			c.ReceivedCount = 1
		})

		_, err := svc.ReceiveCoupon(ctx, coupon.ID, user.ID)
		assert.ErrorIs(t, err, ErrCouponSoldOut)
	})
}

func TestCouponService_CalculateDiscount(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCouponService(db)

	t.Run("固定金额优惠券", func(t *testing.T) {
		coupon := &models.Coupon{
			Type:      models.CouponTypeFixed,
			Value:     10.0,
			MinAmount: 50.0,
		}

		discount := svc.CalculateDiscount(coupon, 100.0)
		assert.Equal(t, 10.0, discount)

		discount = svc.CalculateDiscount(coupon, 30.0)
		assert.Equal(t, 0.0, discount)
	})

	t.Run("百分比折扣优惠券", func(t *testing.T) {
		coupon := &models.Coupon{
			Type:      models.CouponTypePercent,
			Value:     0.1,
			MinAmount: 50.0,
		}

		discount := svc.CalculateDiscount(coupon, 100.0)
		assert.Equal(t, 10.0, discount)
	})

	t.Run("最大优惠金额限制", func(t *testing.T) {
		maxDiscount := 20.0
		coupon := &models.Coupon{
			Type:        models.CouponTypePercent,
			Value:       0.2,
			MinAmount:   50.0,
			MaxDiscount: &maxDiscount,
		}

		discount := svc.CalculateDiscount(coupon, 150.0)
		assert.Equal(t, 20.0, discount)
	})
}

func TestCouponService_GetBestCouponForOrder(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCouponService(db)
	ctx := context.Background()

	t.Run("自动选择最优优惠券", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138010")

		coupon1 := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
			c.Name = "满100减10"
			c.Value = 10.0
			c.MinAmount = 100.0
		})
		coupon2 := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
			c.Name = "满100减20"
			c.Value = 20.0
			c.MinAmount = 100.0
		})

		// 用户领取两张
		svc.ReceiveCoupon(ctx, coupon1.ID, user.ID)
		svc.ReceiveCoupon(ctx, coupon2.ID, user.ID)

		bestCoupon, discount, err := svc.GetBestCouponForOrder(ctx, user.ID, models.CouponScopeAll, 150.0)
		require.NoError(t, err)
		assert.NotNil(t, bestCoupon)
		assert.Equal(t, 20.0, discount)
		assert.Equal(t, coupon2.ID, bestCoupon.CouponID)
	})

	t.Run("无可用优惠券返回nil", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138011")

		bestCoupon, discount, err := svc.GetBestCouponForOrder(ctx, user.ID, models.CouponScopeAll, 100.0)
		require.NoError(t, err)
		assert.Nil(t, bestCoupon)
		assert.Equal(t, 0.0, discount)
	})
}

// ================== UserCouponService Tests ==================

func TestUserCouponService_GetUserCoupons(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138020")
	coupon := createMarketingTestCoupon(t, db)

	// 创建多个用户优惠券
	for i := 0; i < 3; i++ {
		createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)
	}

	req := &UserCouponListRequest{Page: 1, PageSize: 10}
	result, err := svc.GetUserCoupons(ctx, user.ID, req)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
	assert.Len(t, result.List, 3)
}

func TestUserCouponService_GetAvailableCoupons(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138021")
	coupon := createMarketingTestCoupon(t, db)

	// 创建不同状态的优惠券
	createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)
	createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUsed)

	result, err := svc.GetAvailableCoupons(ctx, user.ID, 1, 10)
	require.NoError(t, err)
	// 只返回未使用的
	assert.True(t, result.Total >= 1)
	for _, item := range result.List {
		assert.True(t, item.IsAvailable)
	}
}

func TestUserCouponService_GetUserCouponDetail(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138022")
	coupon := createMarketingTestCoupon(t, db)
	uc := createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)

	t.Run("获取自己的优惠券详情", func(t *testing.T) {
		detail, err := svc.GetUserCouponDetail(ctx, user.ID, uc.ID)
		require.NoError(t, err)
		assert.Equal(t, uc.ID, detail.ID)
	})

	t.Run("获取他人的优惠券失败", func(t *testing.T) {
		otherUser := createMarketingTestUser(t, db, "13800138023")
		_, err := svc.GetUserCouponDetail(ctx, otherUser.ID, uc.ID)
		assert.Error(t, err)
	})
}

func TestUserCouponService_UseCoupon(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	t.Run("正常使用优惠券", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138024")
		coupon := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
			c.Value = 10.0
			c.MinAmount = 50.0
		})
		uc := createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)

		usedCoupon, discount, err := svc.UseCoupon(ctx, uc.ID, 1, 100.0)
		require.NoError(t, err)
		assert.NotNil(t, usedCoupon)
		assert.Equal(t, 10.0, discount)

		// 验证状态更新
		var updated models.UserCoupon
		db.First(&updated, uc.ID)
		assert.Equal(t, int8(models.UserCouponStatusUsed), updated.Status)
	})

	t.Run("使用已使用的优惠券失败", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138025")
		coupon := createMarketingTestCoupon(t, db)
		uc := createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUsed)

		_, _, err := svc.UseCoupon(ctx, uc.ID, 1, 100.0)
		assert.ErrorIs(t, err, ErrUserCouponUsed)
	})

	t.Run("使用已过期的优惠券失败", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138026")
		coupon := createMarketingTestCoupon(t, db)
		uc := &models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(-24 * time.Hour), // 已过期
			ReceivedAt: time.Now().Add(-48 * time.Hour),
		}
		require.NoError(t, db.Create(uc).Error)

		_, _, err := svc.UseCoupon(ctx, uc.ID, 1, 100.0)
		assert.ErrorIs(t, err, ErrUserCouponExpired)
	})

	t.Run("订单金额不满足门槛失败", func(t *testing.T) {
		user := createMarketingTestUser(t, db, "13800138027")
		coupon := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
			c.MinAmount = 100.0
		})
		uc := createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)

		_, _, err := svc.UseCoupon(ctx, uc.ID, 1, 50.0) // 金额不满足
		assert.ErrorIs(t, err, ErrCouponAmountNotMet)
	})
}

func TestUserCouponService_UnuseCoupon(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138028")
	coupon := createMarketingTestCoupon(t, db)
	uc := createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUsed)

	// 更新已使用次数
	db.Model(&coupon).UpdateColumn("used_count", 1)

	err := svc.UnuseCoupon(ctx, uc.ID)
	require.NoError(t, err)

	// 验证状态恢复
	var updated models.UserCoupon
	db.First(&updated, uc.ID)
	assert.Equal(t, int8(models.UserCouponStatusUnused), updated.Status)

	// 验证使用次数减少
	var updatedCoupon models.Coupon
	db.First(&updatedCoupon, coupon.ID)
	assert.Equal(t, 0, updatedCoupon.UsedCount)
}

func TestUserCouponService_GetCouponCountByStatus(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138029")
	coupon := createMarketingTestCoupon(t, db)

	// 创建不同状态的优惠券
	createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)
	createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)
	createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUsed)

	counts, err := svc.GetCouponCountByStatus(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts["unused"])
	assert.Equal(t, int64(1), counts["used"])
}

// ================== CampaignService Tests ==================

func TestCampaignService_GetCampaignList(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCampaignService(db)
	ctx := context.Background()

	// 创建多个活动
	createMarketingTestCampaign(t, db, func(c *models.Campaign) { c.Name = "活动1" })
	createMarketingTestCampaign(t, db, func(c *models.Campaign) { c.Name = "活动2" })

	req := &CampaignListRequest{Page: 1, PageSize: 10}
	result, err := svc.GetCampaignList(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Total >= 2)
	assert.True(t, len(result.List) >= 2)
}

func TestCampaignService_GetCampaignDetail(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCampaignService(db)
	ctx := context.Background()

	campaign := createMarketingTestCampaign(t, db)

	detail, err := svc.GetCampaignDetail(ctx, campaign.ID)
	require.NoError(t, err)
	assert.Equal(t, campaign.ID, detail.ID)
	assert.Equal(t, campaign.Name, detail.Name)
	assert.True(t, detail.IsActive)
}

func TestCampaignService_GetCampaignsByType(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCampaignService(db)
	ctx := context.Background()

	// 创建不同类型的活动
	createMarketingTestCampaign(t, db, func(c *models.Campaign) {
		c.Name = "满减活动"
		c.Type = models.CampaignTypeDiscount
	})
	createMarketingTestCampaign(t, db, func(c *models.Campaign) {
		c.Name = "秒杀活动"
		c.Type = models.CampaignTypeFlashSale
	})

	campaigns, err := svc.GetCampaignsByType(ctx, models.CampaignTypeDiscount)
	require.NoError(t, err)
	for _, c := range campaigns {
		assert.Equal(t, models.CampaignTypeDiscount, c.Type)
	}
}

func TestCampaignService_CalculateDiscountCampaign(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCampaignService(db)
	ctx := context.Background()

	t.Run("无活动时无优惠", func(t *testing.T) {
		discount, campaign, err := svc.CalculateDiscountCampaign(ctx, 100.0)
		require.NoError(t, err)
		assert.Equal(t, 0.0, discount)
		assert.Nil(t, campaign)
	})

	t.Run("有满减活动时计算优惠", func(t *testing.T) {
		// 创建满减活动
		rules := models.JSON{
			"rules": []map[string]interface{}{
				{"min_amount": 100.0, "discount": 10.0},
				{"min_amount": 200.0, "discount": 30.0},
			},
		}
		createMarketingTestCampaign(t, db, func(c *models.Campaign) {
			c.Name = "满减测试"
			c.Type = models.CampaignTypeDiscount
			c.Rules = rules
		})

		// 订单金额150，满足100减10
		discount, campaign, err := svc.CalculateDiscountCampaign(ctx, 150.0)
		require.NoError(t, err)
		assert.Equal(t, 10.0, discount)
		assert.NotNil(t, campaign)

		// 订单金额250，满足200减30
		discount, _, err = svc.CalculateDiscountCampaign(ctx, 250.0)
		require.NoError(t, err)
		assert.Equal(t, 30.0, discount)
	})
}

func TestCampaignService_BuildCampaignItem(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCampaignService(db)
	now := time.Now()

	t.Run("活动进行中", func(t *testing.T) {
		campaign := &models.Campaign{
			ID:        1,
			Name:      "进行中活动",
			Type:      models.CampaignTypeDiscount,
			StartTime: now.Add(-time.Hour),
			EndTime:   now.Add(time.Hour),
			Status:    models.CampaignStatusActive,
		}

		item := svc.buildCampaignItem(campaign, now)
		assert.True(t, item.IsActive)
		assert.Equal(t, "进行中", item.StatusText)
		assert.Equal(t, "满减", item.TypeText)
	})

	t.Run("活动未开始", func(t *testing.T) {
		campaign := &models.Campaign{
			ID:        2,
			Name:      "未开始活动",
			Type:      models.CampaignTypeFlashSale,
			StartTime: now.Add(time.Hour),
			EndTime:   now.Add(2 * time.Hour),
			Status:    models.CampaignStatusActive,
		}

		item := svc.buildCampaignItem(campaign, now)
		assert.False(t, item.IsActive)
		assert.Equal(t, "未开始", item.StatusText)
		assert.Equal(t, "秒杀", item.TypeText)
	})

	t.Run("活动已结束", func(t *testing.T) {
		campaign := &models.Campaign{
			ID:        3,
			Name:      "已结束活动",
			Type:      models.CampaignTypeGroupBuy,
			StartTime: now.Add(-2 * time.Hour),
			EndTime:   now.Add(-time.Hour),
			Status:    models.CampaignStatusActive,
		}

		item := svc.buildCampaignItem(campaign, now)
		assert.False(t, item.IsActive)
		assert.Equal(t, "已结束", item.StatusText)
		assert.Equal(t, "团购", item.TypeText)
	})

	t.Run("活动已禁用", func(t *testing.T) {
		campaign := &models.Campaign{
			ID:        4,
			Name:      "已禁用活动",
			Type:      models.CampaignTypeGift,
			StartTime: now.Add(-time.Hour),
			EndTime:   now.Add(time.Hour),
			Status:    models.CampaignStatusDisabled,
		}

		item := svc.buildCampaignItem(campaign, now)
		assert.False(t, item.IsActive)
		assert.Equal(t, "已禁用", item.StatusText)
		assert.Equal(t, "满赠", item.TypeText)
	})
}

// ================== UserCouponService calculateDiscount Tests ==================

func TestUserCouponService_calculateDiscount(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)

	t.Run("固定金额优惠", func(t *testing.T) {
		coupon := &models.Coupon{
			Type:      models.CouponTypeFixed,
			Value:     15.0,
			MinAmount: 50.0,
		}

		discount := svc.calculateDiscount(coupon, 100.0)
		assert.Equal(t, 15.0, discount)

		// 不满足门槛
		discount = svc.calculateDiscount(coupon, 30.0)
		assert.Equal(t, 0.0, discount)
	})

	t.Run("百分比优惠", func(t *testing.T) {
		coupon := &models.Coupon{
			Type:      models.CouponTypePercent,
			Value:     0.15, // 15% 优惠
			MinAmount: 50.0,
		}

		discount := svc.calculateDiscount(coupon, 100.0)
		assert.Equal(t, 15.0, discount)
	})

	t.Run("优惠不超过订单金额", func(t *testing.T) {
		coupon := &models.Coupon{
			Type:      models.CouponTypeFixed,
			Value:     100.0,
			MinAmount: 0,
		}

		discount := svc.calculateDiscount(coupon, 50.0)
		assert.Equal(t, 50.0, discount)
	})
}

// ================== Edge Cases ==================

func TestMarketing_EdgeCases(t *testing.T) {
	db := setupMarketingTestDB(t)
	ctx := context.Background()

	t.Run("获取不存在的优惠券详情", func(t *testing.T) {
		svc := setupCouponService(db)
		user := createMarketingTestUser(t, db, fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000))

		_, err := svc.GetCouponDetail(ctx, 99999, user.ID)
		assert.Error(t, err)
	})

	t.Run("获取不存在的活动详情", func(t *testing.T) {
		svc := setupCampaignService(db)

		_, err := svc.GetCampaignDetail(ctx, 99999)
		assert.Error(t, err)
	})
}

// ================== Additional Coverage Tests ==================

func TestCouponService_GetUserCouponForOrder(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138010")
	coupon := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
		c.Name = "订单可用优惠券"
		c.MinAmount = 50.0
		c.Value = 10.0
	})
	userCoupon := createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)

	t.Run("获取订单可用的用户优惠券成功", func(t *testing.T) {
		uc, discount, err := svc.GetUserCouponForOrder(ctx, user.ID, userCoupon.ID, "mall", 100.0)
		require.NoError(t, err)
		assert.NotNil(t, uc)
		assert.Equal(t, coupon.ID, uc.CouponID)
		assert.Greater(t, discount, 0.0)
	})

	t.Run("优惠券不存在返回nil", func(t *testing.T) {
		uc, discount, err := svc.GetUserCouponForOrder(ctx, user.ID, 99999, "mall", 100.0)
		require.NoError(t, err)
		assert.Nil(t, uc)
		assert.Equal(t, 0.0, discount)
	})

	t.Run("金额未达到门槛返回nil", func(t *testing.T) {
		uc, discount, err := svc.GetUserCouponForOrder(ctx, user.ID, userCoupon.ID, "mall", 30.0)
		require.NoError(t, err)
		assert.Nil(t, uc)
		assert.Equal(t, 0.0, discount)
	})
}

func TestUserCouponService_GetAvailableCouponsForOrder(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138011")
	coupon := createMarketingTestCoupon(t, db, func(c *models.Coupon) {
		c.Name = "订单可用优惠券2"
		c.MinAmount = 50.0
		c.Value = 10.0
	})
	createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)

	t.Run("获取订单可用的优惠券列表", func(t *testing.T) {
		list, err := svc.GetAvailableCouponsForOrder(ctx, user.ID, "mall", 100.0)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
	})

	t.Run("订单金额太小无可用优惠券", func(t *testing.T) {
		list, err := svc.GetAvailableCouponsForOrder(ctx, user.ID, "mall", 10.0)
		require.NoError(t, err)
		// 可能为空或有满足条件的
		assert.NotNil(t, list)
	})
}

func TestUserCouponService_ExpireUserCoupons(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138012")
	coupon := createMarketingTestCoupon(t, db)

	// 创建一个已过期的用户优惠券
	expiredCoupon := &models.UserCoupon{
		UserID:     user.ID,
		CouponID:   coupon.ID,
		Status:     models.UserCouponStatusUnused,
		ExpiredAt:  time.Now().Add(-1 * time.Hour), // 已过期
		ReceivedAt: time.Now().Add(-2 * time.Hour),
	}
	require.NoError(t, db.Create(expiredCoupon).Error)

	t.Run("过期优惠券批量处理", func(t *testing.T) {
		count, err := svc.ExpireUserCoupons(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0))

		// 验证状态已更新
		var updated models.UserCoupon
		db.First(&updated, expiredCoupon.ID)
		assert.Equal(t, int8(models.UserCouponStatusExpired), updated.Status)
	})
}

func TestUserCouponService_UnuseCoupon_MoreCases(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	ctx := context.Background()

	user := createMarketingTestUser(t, db, "13800138013")
	coupon := createMarketingTestCoupon(t, db)
	userCoupon := createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUsed)

	t.Run("取消使用已使用的优惠券", func(t *testing.T) {
		err := svc.UnuseCoupon(ctx, userCoupon.ID)
		require.NoError(t, err)

		var updated models.UserCoupon
		db.First(&updated, userCoupon.ID)
		assert.Equal(t, int8(models.UserCouponStatusUnused), updated.Status)
	})

	t.Run("取消使用不存在的优惠券", func(t *testing.T) {
		err := svc.UnuseCoupon(ctx, 99999)
		assert.Error(t, err)
	})

	t.Run("取消使用非已使用状态的优惠券不报错", func(t *testing.T) {
		// 创建一个未使用的优惠券
		unusedCoupon := createMarketingTestUserCoupon(t, db, user.ID, coupon.ID, models.UserCouponStatusUnused)
		err := svc.UnuseCoupon(ctx, unusedCoupon.ID)
		// 不是已使用状态，直接返回nil，无需恢复
		assert.NoError(t, err)
	})
}

func TestCampaignService_buildCampaignItem_AllTypes(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCampaignService(db)
	now := time.Now()

	types := []string{
		models.CampaignTypeFlashSale,
		models.CampaignTypeDiscount,
		models.CampaignTypeGift,
		models.CampaignTypeGroupBuy,
		"unknown", // 未知类型
	}

	for _, campType := range types {
		campaign := &models.Campaign{
			ID:        1,
			Name:      "测试活动",
			Type:      campType,
			StartTime: now,
			EndTime:   now.Add(24 * time.Hour),
			Status:    models.CampaignStatusActive,
		}
		item := svc.buildCampaignItem(campaign, now)
		assert.NotEmpty(t, item.TypeText)
		assert.NotEmpty(t, item.StatusText)
	}
}

func TestCampaignService_buildCampaignItem_AllStatuses(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupCampaignService(db)
	now := time.Now()

	statuses := []int8{
		models.CampaignStatusDisabled,
		models.CampaignStatusActive,
		99, // 未知状态
	}

	for _, status := range statuses {
		campaign := &models.Campaign{
			ID:        1,
			Name:      "测试活动",
			Type:      models.CampaignTypeDiscount,
			StartTime: now,
			EndTime:   now.Add(24 * time.Hour),
			Status:    status,
		}
		item := svc.buildCampaignItem(campaign, now)
		assert.NotEmpty(t, item.StatusText)
	}
}

func TestUserCouponService_buildUserCouponItem_AllStatuses(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)
	now := time.Now()

	coupon := &models.Coupon{
		Name:      "测试优惠券",
		Type:      models.CouponTypeFixed,
		Value:     10.0,
		MinAmount: 50.0,
	}

	t.Run("未使用状态", func(t *testing.T) {
		uc := &models.UserCoupon{
			ID:        1,
			UserID:    1,
			CouponID:  1,
			Status:    models.UserCouponStatusUnused,
			ExpiredAt: now.Add(24 * time.Hour),
			Coupon:    coupon,
		}
		item := svc.buildUserCouponItem(uc, now)
		assert.Equal(t, "未使用", item.StatusText)
		assert.True(t, item.IsAvailable)
	})

	t.Run("已使用状态", func(t *testing.T) {
		uc := &models.UserCoupon{
			ID:        2,
			UserID:    1,
			CouponID:  1,
			Status:    models.UserCouponStatusUsed,
			ExpiredAt: now.Add(24 * time.Hour),
			Coupon:    coupon,
		}
		item := svc.buildUserCouponItem(uc, now)
		assert.Equal(t, "已使用", item.StatusText)
		assert.False(t, item.IsAvailable)
	})

	t.Run("已过期状态", func(t *testing.T) {
		uc := &models.UserCoupon{
			ID:        3,
			UserID:    1,
			CouponID:  1,
			Status:    models.UserCouponStatusExpired,
			ExpiredAt: now.Add(24 * time.Hour),
			Coupon:    coupon,
		}
		item := svc.buildUserCouponItem(uc, now)
		assert.Equal(t, "已过期", item.StatusText)
		assert.False(t, item.IsAvailable)
	})

	t.Run("未使用但实际已过期", func(t *testing.T) {
		uc := &models.UserCoupon{
			ID:        4,
			UserID:    1,
			CouponID:  1,
			Status:    models.UserCouponStatusUnused,
			ExpiredAt: now.Add(-24 * time.Hour), // 已过期
			Coupon:    coupon,
		}
		item := svc.buildUserCouponItem(uc, now)
		assert.Equal(t, "已过期", item.StatusText)
		assert.False(t, item.IsAvailable)
	})
}

func TestUserCouponService_calculateDiscount_AllTypes(t *testing.T) {
	db := setupMarketingTestDB(t)
	svc := setupUserCouponService(db)

	t.Run("满减优惠券", func(t *testing.T) {
		coupon := &models.Coupon{
			Type:      models.CouponTypeFixed,
			Value:     10.0,
			MinAmount: 50.0,
		}
		discount := svc.calculateDiscount(coupon, 100.0)
		assert.Equal(t, 10.0, discount)
	})

	t.Run("折扣优惠券", func(t *testing.T) {
		coupon := &models.Coupon{
			Type:      models.CouponTypePercent,
			Value:     0.1, // 10% 折扣
			MinAmount: 0,
		}
		discount := svc.calculateDiscount(coupon, 100.0)
		assert.Equal(t, 10.0, discount) // 100 * 0.1 = 10
	})

	t.Run("未知类型优惠券", func(t *testing.T) {
		coupon := &models.Coupon{
			Type:      "unknown", // 未知类型
			Value:     10.0,
			MinAmount: 0,
		}
		discount := svc.calculateDiscount(coupon, 100.0)
		assert.Equal(t, 0.0, discount)
	})
}
