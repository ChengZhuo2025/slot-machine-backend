//go:build integration
// +build integration

// Package integration 营销模块集成测试
package integration

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
	"github.com/dumeirei/smart-locker-backend/internal/service/marketing"
	"github.com/dumeirei/smart-locker-backend/internal/service/order"
)

// setupMarketingIntegrationTestDB 创建集成测试数据库
func setupMarketingIntegrationTestDB(t *testing.T) *gorm.DB {
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
		&models.Coupon{},
		&models.UserCoupon{},
		&models.Campaign{},
		&models.User{},
		&models.MemberLevel{},
		&models.UserWallet{},
		&models.Order{},
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

// MarketingTestServices 营销测试服务集合
type MarketingTestServices struct {
	DB                 *gorm.DB
	CouponService      *marketing.CouponService
	UserCouponService  *marketing.UserCouponService
	CampaignService    *marketing.CampaignService
	DiscountCalculator *order.DiscountCalculator
}

// setupMarketingTestServices 创建测试服务
func setupMarketingTestServices(db *gorm.DB) *MarketingTestServices {
	couponRepo := repository.NewCouponRepository(db)
	userCouponRepo := repository.NewUserCouponRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)

	couponSvc := marketing.NewCouponService(db, couponRepo, userCouponRepo)
	userCouponSvc := marketing.NewUserCouponService(db, couponRepo, userCouponRepo)
	campaignSvc := marketing.NewCampaignService(campaignRepo)
	discountCalc := order.NewDiscountCalculator(couponSvc, campaignSvc)

	return &MarketingTestServices{
		DB:                 db,
		CouponService:      couponSvc,
		UserCouponService:  userCouponSvc,
		CampaignService:    campaignSvc,
		DiscountCalculator: discountCalc,
	}
}

// createMarketingIntegrationTestUser 创建营销集成测试用户
func createMarketingIntegrationTestUser(db *gorm.DB) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 1000.0,
	}
	db.Create(wallet)

	return user
}

// TestMarketingIntegration_CouponReceiveAndUse 测试优惠券领取和使用完整流程
func TestMarketingIntegration_CouponReceiveAndUse(t *testing.T) {
	t.Run("完整优惠券领取使用流程", func(t *testing.T) {
		db := setupMarketingIntegrationTestDB(t)
		services := setupMarketingTestServices(db)
		user := createMarketingIntegrationTestUser(db)
		ctx := context.Background()

		// 1. 创建优惠券
		coupon := &models.Coupon{
			Name:            "满100减20",
			Type:            models.CouponTypeFixed,
			Value:           20.0,
			MinAmount:       100.0,
			TotalCount:      100,
			PerUserLimit:    2,
			ApplicableScope: models.CouponScopeAll,
			StartTime:       time.Now().Add(-time.Hour),
			EndTime:         time.Now().Add(24 * time.Hour),
			Status:          models.CouponStatusActive,
		}
		db.Create(coupon)

		// 2. 获取优惠券列表，验证可领取
		listReq := &marketing.CouponListRequest{Page: 1, PageSize: 10}
		listResult, err := services.CouponService.GetCouponList(ctx, listReq, user.ID)
		require.NoError(t, err)
		assert.Len(t, listResult.List, 1)
		assert.True(t, listResult.List[0].CanReceive)
		assert.Equal(t, int64(0), listResult.List[0].ReceivedByUser)

		// 3. 领取优惠券
		userCoupon, err := services.CouponService.ReceiveCoupon(ctx, coupon.ID, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, userCoupon)
		assert.Equal(t, int8(models.UserCouponStatusUnused), userCoupon.Status)

		// 4. 验证用户优惠券列表有记录
		userCouponReq := &marketing.UserCouponListRequest{Page: 1, PageSize: 10}
		userCouponResult, err := services.UserCouponService.GetUserCoupons(ctx, user.ID, userCouponReq)
		require.NoError(t, err)
		assert.Len(t, userCouponResult.List, 1)
		assert.Equal(t, "未使用", userCouponResult.List[0].StatusText)

		// 5. 验证优惠券已领取数量更新
		var updatedCoupon models.Coupon
		db.First(&updatedCoupon, coupon.ID)
		assert.Equal(t, 1, updatedCoupon.ReceivedCount)

		// 6. 验证可用优惠券
		availableCoupons, err := services.UserCouponService.GetAvailableCouponsForOrder(ctx, user.ID, models.CouponScopeAll, 150.0)
		require.NoError(t, err)
		assert.Len(t, availableCoupons, 1)

		// 7. 使用优惠券
		orderID := int64(12345)
		usedCoupon, discount, err := services.UserCouponService.UseCoupon(ctx, userCoupon.ID, orderID, 150.0)
		require.NoError(t, err)
		assert.NotNil(t, usedCoupon)
		assert.Equal(t, 20.0, discount)

		// 8. 验证用户优惠券状态更新
		var finalUserCoupon models.UserCoupon
		db.First(&finalUserCoupon, userCoupon.ID)
		assert.Equal(t, int8(models.UserCouponStatusUsed), finalUserCoupon.Status)
		assert.Equal(t, orderID, *finalUserCoupon.OrderID)
		assert.NotNil(t, finalUserCoupon.UsedAt)

		// 9. 验证优惠券使用数量更新
		db.First(&updatedCoupon, coupon.ID)
		assert.Equal(t, 1, updatedCoupon.UsedCount)
	})
}

// TestMarketingIntegration_CouponRefund 测试优惠券退款恢复流程
func TestMarketingIntegration_CouponRefund(t *testing.T) {
	t.Run("优惠券使用后退款恢复", func(t *testing.T) {
		db := setupMarketingIntegrationTestDB(t)
		services := setupMarketingTestServices(db)
		user := createMarketingIntegrationTestUser(db)
		ctx := context.Background()

		// 1. 创建并领取优惠券
		coupon := &models.Coupon{
			Name:            "满50减10",
			Type:            models.CouponTypeFixed,
			Value:           10.0,
			MinAmount:       50.0,
			TotalCount:      100,
			PerUserLimit:    2,
			ApplicableScope: models.CouponScopeAll,
			StartTime:       time.Now().Add(-time.Hour),
			EndTime:         time.Now().Add(24 * time.Hour),
			Status:          models.CouponStatusActive,
		}
		db.Create(coupon)

		userCoupon, err := services.CouponService.ReceiveCoupon(ctx, coupon.ID, user.ID)
		require.NoError(t, err)

		// 2. 使用优惠券
		orderID := int64(12345)
		_, _, err = services.UserCouponService.UseCoupon(ctx, userCoupon.ID, orderID, 100.0)
		require.NoError(t, err)

		// 验证优惠券已使用
		var usedUserCoupon models.UserCoupon
		db.First(&usedUserCoupon, userCoupon.ID)
		assert.Equal(t, int8(models.UserCouponStatusUsed), usedUserCoupon.Status)

		// 3. 退款，恢复优惠券
		err = services.UserCouponService.UnuseCoupon(ctx, userCoupon.ID)
		require.NoError(t, err)

		// 4. 验证优惠券已恢复
		var restoredUserCoupon models.UserCoupon
		db.First(&restoredUserCoupon, userCoupon.ID)
		assert.Equal(t, int8(models.UserCouponStatusUnused), restoredUserCoupon.Status)
		assert.Nil(t, restoredUserCoupon.OrderID)
		assert.Nil(t, restoredUserCoupon.UsedAt)

		// 5. 验证优惠券使用数量恢复
		var updatedCoupon models.Coupon
		db.First(&updatedCoupon, coupon.ID)
		assert.Equal(t, 0, updatedCoupon.UsedCount)

		// 6. 验证优惠券可再次使用
		availableCoupons, err := services.UserCouponService.GetAvailableCouponsForOrder(ctx, user.ID, models.CouponScopeAll, 100.0)
		require.NoError(t, err)
		assert.Len(t, availableCoupons, 1)
	})
}

// TestMarketingIntegration_DiscountCalculation 测试订单优惠计算
func TestMarketingIntegration_DiscountCalculation(t *testing.T) {
	t.Run("同时使用优惠券和满减活动", func(t *testing.T) {
		db := setupMarketingIntegrationTestDB(t)
		services := setupMarketingTestServices(db)
		user := createMarketingIntegrationTestUser(db)
		ctx := context.Background()

		// 1. 创建满减活动
			campaign := &models.Campaign{
				Name:      "满减活动",
				Type:      models.CampaignTypeDiscount,
				StartTime: time.Now().Add(-time.Hour),
				EndTime:   time.Now().Add(24 * time.Hour),
				Status:    models.CampaignStatusActive,
				Rules: models.JSON{
					"rules": []marketing.DiscountRule{
						{MinAmount: 100, Discount: 10},
						{MinAmount: 200, Discount: 25},
					},
				},
			}
			db.Create(campaign)

		// 2. 创建优惠券并领取
		coupon := &models.Coupon{
			Name:            "满100减15",
			Type:            models.CouponTypeFixed,
			Value:           15.0,
			MinAmount:       100.0,
			TotalCount:      100,
			PerUserLimit:    2,
			ApplicableScope: models.CouponScopeAll,
			StartTime:       time.Now().Add(-time.Hour),
			EndTime:         time.Now().Add(24 * time.Hour),
			Status:          models.CouponStatusActive,
		}
		db.Create(coupon)

		userCoupon, err := services.CouponService.ReceiveCoupon(ctx, coupon.ID, user.ID)
		require.NoError(t, err)

		// 3. 计算订单优惠（订单金额250）
		result, err := services.DiscountCalculator.CalculateOrderDiscount(ctx, user.ID, models.CouponScopeAll, 250.0, &userCoupon.ID)
		require.NoError(t, err)

		// 验证结果
		assert.Equal(t, 250.0, result.OriginalAmount)
		assert.Equal(t, 25.0, result.CampaignDiscount) // 满200减25
		assert.Equal(t, 15.0, result.CouponDiscount)   // 优惠券减15
		assert.Equal(t, 40.0, result.TotalDiscount)    // 总优惠40
		assert.Equal(t, 210.0, result.FinalAmount)     // 最终金额210

		// 验证优惠明细
		assert.Len(t, result.DiscountDetails, 2)
	})

	t.Run("自动选择最优优惠券", func(t *testing.T) {
		db := setupMarketingIntegrationTestDB(t)
		services := setupMarketingTestServices(db)
		user := createMarketingIntegrationTestUser(db)
		ctx := context.Background()

		// 创建多个优惠券并领取
		coupons := []*models.Coupon{
			{
				Name:            "满100减10",
				Type:            models.CouponTypeFixed,
				Value:           10.0,
				MinAmount:       100.0,
				TotalCount:      100,
				PerUserLimit:    2,
				ApplicableScope: models.CouponScopeAll,
				StartTime:       time.Now().Add(-time.Hour),
				EndTime:         time.Now().Add(24 * time.Hour),
				Status:          models.CouponStatusActive,
			},
			{
				Name:            "满100减20",
				Type:            models.CouponTypeFixed,
				Value:           20.0,
				MinAmount:       100.0,
				TotalCount:      100,
				PerUserLimit:    2,
				ApplicableScope: models.CouponScopeAll,
				StartTime:       time.Now().Add(-time.Hour),
				EndTime:         time.Now().Add(24 * time.Hour),
				Status:          models.CouponStatusActive,
			},
			{
				Name:            "满200减50",
				Type:            models.CouponTypeFixed,
				Value:           50.0,
				MinAmount:       200.0,
				TotalCount:      100,
				PerUserLimit:    2,
				ApplicableScope: models.CouponScopeAll,
				StartTime:       time.Now().Add(-time.Hour),
				EndTime:         time.Now().Add(24 * time.Hour),
				Status:          models.CouponStatusActive,
			},
		}

		for _, c := range coupons {
			db.Create(c)
			services.CouponService.ReceiveCoupon(ctx, c.ID, user.ID)
		}

		// 订单金额150，应自动选择满100减20
		result, err := services.DiscountCalculator.CalculateOrderDiscount(ctx, user.ID, models.CouponScopeAll, 150.0, nil)
		require.NoError(t, err)
		assert.Equal(t, 20.0, result.CouponDiscount)
		assert.NotNil(t, result.UserCoupon)
	})
}

// TestMarketingIntegration_CouponExpiration 测试优惠券过期处理
func TestMarketingIntegration_CouponExpiration(t *testing.T) {
	t.Run("批量标记过期优惠券", func(t *testing.T) {
		db := setupMarketingIntegrationTestDB(t)
		services := setupMarketingTestServices(db)
		user := createMarketingIntegrationTestUser(db)
		ctx := context.Background()

		// 创建优惠券
		coupon := &models.Coupon{
			Name:            "测试优惠券",
			Type:            models.CouponTypeFixed,
			Value:           10.0,
			MinAmount:       50.0,
			TotalCount:      100,
			PerUserLimit:    5,
			ApplicableScope: models.CouponScopeAll,
			StartTime:       time.Now().Add(-48 * time.Hour),
			EndTime:         time.Now().Add(24 * time.Hour),
			Status:          models.CouponStatusActive,
		}
		db.Create(coupon)

		// 创建用户优惠券
		// 1. 已过期未标记
		uc1 := &models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(-time.Hour),
			ReceivedAt: time.Now().Add(-24 * time.Hour),
		}
		db.Create(uc1)

		// 2. 已过期已使用（不应被标记为过期）
		uc2 := &models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUsed,
			ExpiredAt:  time.Now().Add(-time.Hour),
			ReceivedAt: time.Now().Add(-24 * time.Hour),
		}
		db.Create(uc2)

		// 3. 未过期
		uc3 := &models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(time.Hour),
			ReceivedAt: time.Now(),
		}
		db.Create(uc3)

		// 执行过期处理
		affected, err := services.UserCouponService.ExpireUserCoupons(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected) // 只有1个未使用的过期券

		// 验证结果
		var expiredCoupon models.UserCoupon
		db.First(&expiredCoupon, uc1.ID)
		assert.Equal(t, int8(models.UserCouponStatusExpired), expiredCoupon.Status)

		var usedCoupon models.UserCoupon
		db.First(&usedCoupon, uc2.ID)
		assert.Equal(t, int8(models.UserCouponStatusUsed), usedCoupon.Status) // 保持已使用状态

		var validCoupon models.UserCoupon
		db.First(&validCoupon, uc3.ID)
		assert.Equal(t, int8(models.UserCouponStatusUnused), validCoupon.Status) // 保持未使用状态
	})
}

// TestMarketingIntegration_CouponLimit 测试优惠券领取限制
func TestMarketingIntegration_CouponLimit(t *testing.T) {
	t.Run("用户领取数量限制", func(t *testing.T) {
		db := setupMarketingIntegrationTestDB(t)
		services := setupMarketingTestServices(db)
		user := createMarketingIntegrationTestUser(db)
		ctx := context.Background()

		// 创建限领2张的优惠券
		coupon := &models.Coupon{
			Name:            "限领2张",
			Type:            models.CouponTypeFixed,
			Value:           10.0,
			MinAmount:       50.0,
			TotalCount:      100,
			PerUserLimit:    2,
			ApplicableScope: models.CouponScopeAll,
			StartTime:       time.Now().Add(-time.Hour),
			EndTime:         time.Now().Add(24 * time.Hour),
			Status:          models.CouponStatusActive,
		}
		db.Create(coupon)

		// 领取第1张
		_, err := services.CouponService.ReceiveCoupon(ctx, coupon.ID, user.ID)
		require.NoError(t, err)

		// 领取第2张
		_, err = services.CouponService.ReceiveCoupon(ctx, coupon.ID, user.ID)
		require.NoError(t, err)

		// 领取第3张应失败
		_, err = services.CouponService.ReceiveCoupon(ctx, coupon.ID, user.ID)
		assert.ErrorIs(t, err, marketing.ErrCouponLimitExceeded)

		// 验证用户优惠券数量
		req := &marketing.UserCouponListRequest{Page: 1, PageSize: 10}
		result, _ := services.UserCouponService.GetUserCoupons(ctx, user.ID, req)
		assert.Equal(t, int64(2), result.Total)
	})

	t.Run("优惠券总数限制", func(t *testing.T) {
		db := setupMarketingIntegrationTestDB(t)
		services := setupMarketingTestServices(db)
		ctx := context.Background()

		// 创建限量2张的优惠券
		coupon := &models.Coupon{
			Name:            "限量2张",
			Type:            models.CouponTypeFixed,
			Value:           10.0,
			MinAmount:       50.0,
			TotalCount:      2,
			PerUserLimit:    5,
			ApplicableScope: models.CouponScopeAll,
			StartTime:       time.Now().Add(-time.Hour),
			EndTime:         time.Now().Add(24 * time.Hour),
			Status:          models.CouponStatusActive,
		}
		db.Create(coupon)

		// 用户1领取
		user1 := createMarketingIntegrationTestUser(db)
		_, err := services.CouponService.ReceiveCoupon(ctx, coupon.ID, user1.ID)
		require.NoError(t, err)

		// 用户2领取
		user2 := createMarketingIntegrationTestUser(db)
		_, err = services.CouponService.ReceiveCoupon(ctx, coupon.ID, user2.ID)
		require.NoError(t, err)

		// 用户3领取应失败
		user3 := createMarketingIntegrationTestUser(db)
		_, err = services.CouponService.ReceiveCoupon(ctx, coupon.ID, user3.ID)
		assert.ErrorIs(t, err, marketing.ErrCouponSoldOut)
	})
}

// TestMarketingIntegration_CouponStatistics 测试优惠券统计
func TestMarketingIntegration_CouponStatistics(t *testing.T) {
	t.Run("用户优惠券统计", func(t *testing.T) {
		db := setupMarketingIntegrationTestDB(t)
		services := setupMarketingTestServices(db)
		user := createMarketingIntegrationTestUser(db)
		ctx := context.Background()

		// 创建优惠券
		coupon := &models.Coupon{
			Name:            "测试优惠券",
			Type:            models.CouponTypeFixed,
			Value:           10.0,
			MinAmount:       50.0,
			TotalCount:      100,
			PerUserLimit:    10,
			ApplicableScope: models.CouponScopeAll,
			StartTime:       time.Now().Add(-time.Hour),
			EndTime:         time.Now().Add(24 * time.Hour),
			Status:          models.CouponStatusActive,
		}
		db.Create(coupon)

		// 领取5张
		var userCoupons []*models.UserCoupon
		for i := 0; i < 5; i++ {
			uc, _ := services.CouponService.ReceiveCoupon(ctx, coupon.ID, user.ID)
			userCoupons = append(userCoupons, uc)
		}

		// 使用2张
		services.UserCouponService.UseCoupon(ctx, userCoupons[0].ID, 1001, 100.0)
		services.UserCouponService.UseCoupon(ctx, userCoupons[1].ID, 1002, 100.0)

		// 过期1张（手动设置）
		db.Model(&userCoupons[2]).Updates(map[string]interface{}{
			"status":     models.UserCouponStatusExpired,
			"expired_at": time.Now().Add(-time.Hour),
		})

		// 获取统计
		counts, err := services.UserCouponService.GetCouponCountByStatus(ctx, user.ID)
		require.NoError(t, err)

		assert.Equal(t, int64(2), counts["unused"])
		assert.Equal(t, int64(2), counts["used"])
		assert.Equal(t, int64(1), counts["expired"])
	})
}
