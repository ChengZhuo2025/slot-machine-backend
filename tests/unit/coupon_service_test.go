//go:build unit
// +build unit

// Package unit 优惠券服务单元测试
package unit

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
)

// setupCouponServiceTestDB 创建测试数据库
func setupCouponServiceTestDB(t *testing.T) *gorm.DB {
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
		&models.User{},
		&models.MemberLevel{},
		&models.UserWallet{},
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

// createTestCouponService 创建测试服务
func createTestCouponService(db *gorm.DB) *marketing.CouponService {
	couponRepo := repository.NewCouponRepository(db)
	userCouponRepo := repository.NewUserCouponRepository(db)
	return marketing.NewCouponService(db, couponRepo, userCouponRepo)
}

// createTestUser 创建测试用户
func createTestUser(db *gorm.DB) *models.User {
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

// createTestCoupon 创建测试优惠券
func createTestCoupon(db *gorm.DB, opts ...func(*models.Coupon)) *models.Coupon {
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

	// 保存原始状态值（GORM 会跳过零值）
	originalStatus := coupon.Status

	db.Create(coupon)

	// 如果状态是禁用(0)，需要显式更新，因为 GORM 会使用数据库默认值
	if originalStatus == models.CouponStatusDisabled {
		db.Model(coupon).Update("status", originalStatus)
	}

	return coupon
}

// TestCouponService_GetCouponList 测试获取优惠券列表
func TestCouponService_GetCouponList(t *testing.T) {
	t.Run("正常获取优惠券列表", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		// 创建多个优惠券
		createTestCoupon(db, func(c *models.Coupon) { c.Name = "优惠券1" })
		createTestCoupon(db, func(c *models.Coupon) { c.Name = "优惠券2" })
		createTestCoupon(db, func(c *models.Coupon) { c.Name = "优惠券3" })

		req := &marketing.CouponListRequest{
			Page:     1,
			PageSize: 10,
		}

		result, err := svc.GetCouponList(context.Background(), req, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Total)
		assert.Len(t, result.List, 3)
	})

	t.Run("分页获取优惠券列表", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		// 创建5个优惠券
		for i := 0; i < 5; i++ {
			createTestCoupon(db, func(c *models.Coupon) {
				c.Name = fmt.Sprintf("优惠券%d", i+1)
			})
		}

		// 获取第一页（每页2个）
		req := &marketing.CouponListRequest{
			Page:     1,
			PageSize: 2,
		}

		result, err := svc.GetCouponList(context.Background(), req, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result.Total)
		assert.Len(t, result.List, 2)

		// 获取第二页
		req.Page = 2
		result, err = svc.GetCouponList(context.Background(), req, user.ID)
		require.NoError(t, err)
		assert.Len(t, result.List, 2)
	})

	t.Run("过滤未生效优惠券", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		// 创建有效优惠券
		createTestCoupon(db, func(c *models.Coupon) { c.Name = "有效优惠券" })

		// 创建未开始优惠券
		createTestCoupon(db, func(c *models.Coupon) {
			c.Name = "未开始"
			c.StartTime = time.Now().Add(24 * time.Hour)
			c.EndTime = time.Now().Add(48 * time.Hour)
		})

		// 创建已结束优惠券
		createTestCoupon(db, func(c *models.Coupon) {
			c.Name = "已结束"
			c.StartTime = time.Now().Add(-48 * time.Hour)
			c.EndTime = time.Now().Add(-24 * time.Hour)
		})

		// 创建禁用优惠券
		createTestCoupon(db, func(c *models.Coupon) {
			c.Name = "禁用"
			c.Status = models.CouponStatusDisabled
		})

		req := &marketing.CouponListRequest{
			Page:     1,
			PageSize: 10,
		}

		result, err := svc.GetCouponList(context.Background(), req, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, "有效优惠券", result.List[0].Name)
	})

	t.Run("检查用户已领取数量和可领取状态", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		// 创建优惠券，每人限领2张
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.PerUserLimit = 2
		})

		// 用户已领取1张
		db.Create(&models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		})

		req := &marketing.CouponListRequest{
			Page:     1,
			PageSize: 10,
		}

		result, err := svc.GetCouponList(context.Background(), req, user.ID)
		require.NoError(t, err)
		assert.Len(t, result.List, 1)
		assert.Equal(t, int64(1), result.List[0].ReceivedByUser)
		assert.True(t, result.List[0].CanReceive) // 还能再领取1张
	})
}

// TestCouponService_GetCouponDetail 测试获取优惠券详情
func TestCouponService_GetCouponDetail(t *testing.T) {
	t.Run("正常获取优惠券详情", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db)

		detail, err := svc.GetCouponDetail(context.Background(), coupon.ID, user.ID)
		require.NoError(t, err)
		assert.Equal(t, coupon.ID, detail.ID)
		assert.Equal(t, coupon.Name, detail.Name)
		assert.Equal(t, coupon.Type, detail.Type)
		assert.Equal(t, coupon.Value, detail.Value)
		assert.True(t, detail.CanReceive)
	})

	t.Run("获取不存在的优惠券", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		_, err := svc.GetCouponDetail(context.Background(), 99999, user.ID)
		assert.Error(t, err)
	})

	t.Run("检查禁用优惠券不可领取", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.Status = models.CouponStatusDisabled
		})

		detail, err := svc.GetCouponDetail(context.Background(), coupon.ID, user.ID)
		require.NoError(t, err)
		assert.False(t, detail.CanReceive)
	})

	t.Run("检查未开始优惠券不可领取", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.StartTime = time.Now().Add(24 * time.Hour)
			c.EndTime = time.Now().Add(48 * time.Hour)
		})

		detail, err := svc.GetCouponDetail(context.Background(), coupon.ID, user.ID)
		require.NoError(t, err)
		assert.False(t, detail.CanReceive)
	})

	t.Run("检查已结束优惠券不可领取", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.StartTime = time.Now().Add(-48 * time.Hour)
			c.EndTime = time.Now().Add(-24 * time.Hour)
		})

		detail, err := svc.GetCouponDetail(context.Background(), coupon.ID, user.ID)
		require.NoError(t, err)
		assert.False(t, detail.CanReceive)
	})
}

// TestCouponService_ReceiveCoupon 测试领取优惠券
func TestCouponService_ReceiveCoupon(t *testing.T) {
	t.Run("正常领取优惠券", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db)

		userCoupon, err := svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, userCoupon)
		assert.Equal(t, user.ID, userCoupon.UserID)
		assert.Equal(t, coupon.ID, userCoupon.CouponID)
		assert.Equal(t, int8(models.UserCouponStatusUnused), userCoupon.Status)

		// 验证优惠券已领取数量增加
		var updatedCoupon models.Coupon
		db.First(&updatedCoupon, coupon.ID)
		assert.Equal(t, 1, updatedCoupon.ReceivedCount)
	})

	t.Run("领取禁用优惠券失败", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.Status = models.CouponStatusDisabled
		})

		_, err := svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		assert.ErrorIs(t, err, marketing.ErrCouponNotActive)
	})

	t.Run("领取未开始优惠券失败", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.StartTime = time.Now().Add(24 * time.Hour)
			c.EndTime = time.Now().Add(48 * time.Hour)
		})

		_, err := svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		assert.ErrorIs(t, err, marketing.ErrCouponNotStarted)
	})

	t.Run("领取已结束优惠券失败", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.StartTime = time.Now().Add(-48 * time.Hour)
			c.EndTime = time.Now().Add(-24 * time.Hour)
		})

		_, err := svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		assert.ErrorIs(t, err, marketing.ErrCouponExpired)
	})

	t.Run("领取已领完优惠券失败", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.TotalCount = 10
			c.ReceivedCount = 10 // 已领完
		})

		_, err := svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		assert.ErrorIs(t, err, marketing.ErrCouponSoldOut)
	})

	t.Run("超过每人限领数量失败", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.PerUserLimit = 1
		})

		// 先领取一张
		_, err := svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		require.NoError(t, err)

		// 再次领取应该失败
		_, err = svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		assert.ErrorIs(t, err, marketing.ErrCouponLimitExceeded)
	})

	t.Run("使用ValidDays计算过期时间", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		validDays := 7
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.ValidDays = &validDays
			c.EndTime = time.Now().Add(30 * 24 * time.Hour) // 优惠券活动30天后结束
		})

		userCoupon, err := svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		require.NoError(t, err)

		// 过期时间应该是领取后7天
		expectedExpireAt := time.Now().AddDate(0, 0, 7)
		// 允许1秒误差
		assert.WithinDuration(t, expectedExpireAt, userCoupon.ExpiredAt, time.Second)
	})

	t.Run("ValidDays过期时间不超过优惠券结束时间", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		validDays := 30 // 30天有效期
		endTime := time.Now().Add(7 * 24 * time.Hour) // 但优惠券7天后就结束
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.ValidDays = &validDays
			c.EndTime = endTime
		})

		userCoupon, err := svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)
		require.NoError(t, err)

		// 过期时间应该是优惠券结束时间，而不是30天后
		assert.WithinDuration(t, endTime, userCoupon.ExpiredAt, time.Second)
	})
}

// TestCouponService_CalculateDiscount 测试计算优惠金额
func TestCouponService_CalculateDiscount(t *testing.T) {
	t.Run("固定金额优惠券", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)

		coupon := &models.Coupon{
			Type:      models.CouponTypeFixed,
			Value:     10.0,
			MinAmount: 50.0,
		}

		// 订单金额满足门槛
		discount := svc.CalculateDiscount(coupon, 100.0)
		assert.Equal(t, 10.0, discount)

		// 订单金额不满足门槛
		discount = svc.CalculateDiscount(coupon, 30.0)
		assert.Equal(t, 0.0, discount)
	})

	t.Run("百分比折扣优惠券", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)

		coupon := &models.Coupon{
			Type:      models.CouponTypePercent,
			Value:     0.1, // 10% 折扣
			MinAmount: 50.0,
		}

		// 订单金额满足门槛
		discount := svc.CalculateDiscount(coupon, 100.0)
		assert.Equal(t, 10.0, discount)

		// 订单金额不满足门槛
		discount = svc.CalculateDiscount(coupon, 30.0)
		assert.Equal(t, 0.0, discount)
	})

	t.Run("最大优惠金额限制", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)

		maxDiscount := 20.0
		coupon := &models.Coupon{
			Type:        models.CouponTypePercent,
			Value:       0.2, // 20% 折扣
			MinAmount:   50.0,
			MaxDiscount: &maxDiscount,
		}

		// 计算优惠为30，但最大只能20
		discount := svc.CalculateDiscount(coupon, 150.0)
		assert.Equal(t, 20.0, discount)
	})

	t.Run("优惠金额不超过订单金额", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)

		coupon := &models.Coupon{
			Type:      models.CouponTypeFixed,
			Value:     100.0, // 减100
			MinAmount: 0,     // 无门槛
		}

		// 订单金额50，优惠最多50
		discount := svc.CalculateDiscount(coupon, 50.0)
		assert.Equal(t, 50.0, discount)
	})
}

// TestCouponService_GetBestCouponForOrder 测试获取订单最优优惠券
func TestCouponService_GetBestCouponForOrder(t *testing.T) {
	t.Run("自动选择最优优惠券", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		// 创建多个优惠券
		coupon1 := createTestCoupon(db, func(c *models.Coupon) {
			c.Name = "满100减10"
			c.Type = models.CouponTypeFixed
			c.Value = 10.0
			c.MinAmount = 100.0
		})
		coupon2 := createTestCoupon(db, func(c *models.Coupon) {
			c.Name = "满50减5"
			c.Type = models.CouponTypeFixed
			c.Value = 5.0
			c.MinAmount = 50.0
		})
		coupon3 := createTestCoupon(db, func(c *models.Coupon) {
			c.Name = "满100减20"
			c.Type = models.CouponTypeFixed
			c.Value = 20.0
			c.MinAmount = 100.0
		})

		// 用户领取所有优惠券
		svc.ReceiveCoupon(context.Background(), coupon1.ID, user.ID)
		svc.ReceiveCoupon(context.Background(), coupon2.ID, user.ID)
		svc.ReceiveCoupon(context.Background(), coupon3.ID, user.ID)

		// 订单金额150，应该选择减20的
		bestCoupon, discount, err := svc.GetBestCouponForOrder(context.Background(), user.ID, models.CouponScopeAll, 150.0)
		require.NoError(t, err)
		assert.NotNil(t, bestCoupon)
		assert.Equal(t, 20.0, discount)
		assert.Equal(t, coupon3.ID, bestCoupon.CouponID)
	})

	t.Run("无可用优惠券返回nil", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		bestCoupon, discount, err := svc.GetBestCouponForOrder(context.Background(), user.ID, models.CouponScopeAll, 100.0)
		require.NoError(t, err)
		assert.Nil(t, bestCoupon)
		assert.Equal(t, 0.0, discount)
	})

	t.Run("订单金额不满足任何优惠券门槛", func(t *testing.T) {
		db := setupCouponServiceTestDB(t)
		svc := createTestCouponService(db)
		user := createTestUser(db)

		// 创建优惠券，门槛100
		coupon := createTestCoupon(db, func(c *models.Coupon) {
			c.MinAmount = 100.0
		})

		// 用户领取
		svc.ReceiveCoupon(context.Background(), coupon.ID, user.ID)

		// 订单金额50，不满足门槛
		bestCoupon, discount, err := svc.GetBestCouponForOrder(context.Background(), user.ID, models.CouponScopeAll, 50.0)
		require.NoError(t, err)
		assert.Nil(t, bestCoupon)
		assert.Equal(t, 0.0, discount)
	})
}
