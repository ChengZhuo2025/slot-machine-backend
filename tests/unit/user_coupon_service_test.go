//go:build unit
// +build unit

// Package unit 用户优惠券服务单元测试
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

// setupUserCouponServiceTestDB 创建测试数据库
func setupUserCouponServiceTestDB(t *testing.T) *gorm.DB {
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

// createUserCouponTestService 创建测试服务
func createUserCouponTestService(db *gorm.DB) *marketing.UserCouponService {
	couponRepo := repository.NewCouponRepository(db)
	userCouponRepo := repository.NewUserCouponRepository(db)
	return marketing.NewUserCouponService(db, couponRepo, userCouponRepo)
}

// createUCTestUser 创建测试用户
func createUCTestUser(db *gorm.DB) *models.User {
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

// createUCTestCoupon 创建测试优惠券
func createUCTestCoupon(db *gorm.DB, opts ...func(*models.Coupon)) *models.Coupon {
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

	db.Create(coupon)
	return coupon
}

// createUCTestUserCoupon 创建测试用户优惠券
func createUCTestUserCoupon(db *gorm.DB, userID, couponID int64, opts ...func(*models.UserCoupon)) *models.UserCoupon {
	userCoupon := &models.UserCoupon{
		UserID:     userID,
		CouponID:   couponID,
		Status:     models.UserCouponStatusUnused,
		ExpiredAt:  time.Now().Add(24 * time.Hour),
		ReceivedAt: time.Now(),
	}

	for _, opt := range opts {
		opt(userCoupon)
	}

	db.Create(userCoupon)
	return userCoupon
}

// TestUserCouponService_GetUserCoupons 测试获取用户优惠券列表
func TestUserCouponService_GetUserCoupons(t *testing.T) {
	t.Run("正常获取用户优惠券列表", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		// 创建多个优惠券并领取
		coupon1 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "优惠券1" })
		coupon2 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "优惠券2" })
		coupon3 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "优惠券3" })

		createUCTestUserCoupon(db, user.ID, coupon1.ID)
		createUCTestUserCoupon(db, user.ID, coupon2.ID)
		createUCTestUserCoupon(db, user.ID, coupon3.ID)

		req := &marketing.UserCouponListRequest{
			Page:     1,
			PageSize: 10,
		}

		result, err := svc.GetUserCoupons(context.Background(), user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Total)
		assert.Len(t, result.List, 3)
	})

	t.Run("分页获取用户优惠券列表", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		// 创建5个优惠券并领取
		for i := 0; i < 5; i++ {
			coupon := createUCTestCoupon(db, func(c *models.Coupon) {
				c.Name = fmt.Sprintf("优惠券%d", i+1)
			})
			createUCTestUserCoupon(db, user.ID, coupon.ID)
		}

		// 获取第一页（每页2个）
		req := &marketing.UserCouponListRequest{
			Page:     1,
			PageSize: 2,
		}

		result, err := svc.GetUserCoupons(context.Background(), user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result.Total)
		assert.Len(t, result.List, 2)

		// 获取第二页
		req.Page = 2
		result, err = svc.GetUserCoupons(context.Background(), user.ID, req)
		require.NoError(t, err)
		assert.Len(t, result.List, 2)
	})

	t.Run("按状态筛选用户优惠券", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		// 创建不同状态的用户优惠券
		coupon1 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "未使用" })
		coupon2 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "已使用" })
		coupon3 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "已过期" })

		createUCTestUserCoupon(db, user.ID, coupon1.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUnused
		})
		createUCTestUserCoupon(db, user.ID, coupon2.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUsed
			now := time.Now()
			uc.UsedAt = &now
		})
		createUCTestUserCoupon(db, user.ID, coupon3.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusExpired
		})

		// 筛选未使用
		unusedStatus := int8(models.UserCouponStatusUnused)
		req := &marketing.UserCouponListRequest{
			Page:     1,
			PageSize: 10,
			Status:   &unusedStatus,
		}

		result, err := svc.GetUserCoupons(context.Background(), user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, int8(models.UserCouponStatusUnused), result.List[0].Status)

		// 筛选已使用
		usedStatus := int8(models.UserCouponStatusUsed)
		req.Status = &usedStatus

		result, err = svc.GetUserCoupons(context.Background(), user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, int8(models.UserCouponStatusUsed), result.List[0].Status)
	})
}

// TestUserCouponService_GetAvailableCoupons 测试获取可用优惠券列表
func TestUserCouponService_GetAvailableCoupons(t *testing.T) {
	t.Run("正常获取可用优惠券列表", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		// 创建可用优惠券
		coupon1 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "可用1" })
		coupon2 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "可用2" })

		createUCTestUserCoupon(db, user.ID, coupon1.ID)
		createUCTestUserCoupon(db, user.ID, coupon2.ID)

		// 创建已使用优惠券
		coupon3 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "已使用" })
		createUCTestUserCoupon(db, user.ID, coupon3.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUsed
		})

		// 创建已过期优惠券
		coupon4 := createUCTestCoupon(db, func(c *models.Coupon) { c.Name = "已过期" })
		createUCTestUserCoupon(db, user.ID, coupon4.ID, func(uc *models.UserCoupon) {
			uc.ExpiredAt = time.Now().Add(-time.Hour) // 已过期
		})

		result, err := svc.GetAvailableCoupons(context.Background(), user.ID, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.List, 2)
	})
}

// TestUserCouponService_GetAvailableCouponsForOrder 测试获取订单可用优惠券
func TestUserCouponService_GetAvailableCouponsForOrder(t *testing.T) {
	t.Run("正常获取订单可用优惠券", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		// 创建满足条件的优惠券
		coupon1 := createUCTestCoupon(db, func(c *models.Coupon) {
			c.Name = "满50减10"
			c.MinAmount = 50.0
			c.Value = 10.0
		})
		createUCTestUserCoupon(db, user.ID, coupon1.ID)

		// 创建不满足金额的优惠券
		coupon2 := createUCTestCoupon(db, func(c *models.Coupon) {
			c.Name = "满200减30"
			c.MinAmount = 200.0
			c.Value = 30.0
		})
		createUCTestUserCoupon(db, user.ID, coupon2.ID)

		// 订单金额100
		result, err := svc.GetAvailableCouponsForOrder(context.Background(), user.ID, models.CouponScopeAll, 100.0)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "满50减10", result[0].CouponName)
	})

	t.Run("按适用范围筛选", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		// 创建全场通用优惠券
		coupon1 := createUCTestCoupon(db, func(c *models.Coupon) {
			c.Name = "全场通用"
			c.ApplicableScope = models.CouponScopeAll
			c.MinAmount = 0
		})
		createUCTestUserCoupon(db, user.ID, coupon1.ID)

		// 创建商城专用优惠券
		coupon2 := createUCTestCoupon(db, func(c *models.Coupon) {
			c.Name = "商城专用"
			c.ApplicableScope = "mall"
			c.MinAmount = 0
		})
		createUCTestUserCoupon(db, user.ID, coupon2.ID)

		// 创建租借专用优惠券
		coupon3 := createUCTestCoupon(db, func(c *models.Coupon) {
			c.Name = "租借专用"
			c.ApplicableScope = "rental"
			c.MinAmount = 0
		})
		createUCTestUserCoupon(db, user.ID, coupon3.ID)

		// 商城订单应该能使用全场通用和商城专用
		result, err := svc.GetAvailableCouponsForOrder(context.Background(), user.ID, "mall", 100.0)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

// TestUserCouponService_GetUserCouponDetail 测试获取用户优惠券详情
func TestUserCouponService_GetUserCouponDetail(t *testing.T) {
	t.Run("正常获取用户优惠券详情", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		coupon := createUCTestCoupon(db)
		userCoupon := createUCTestUserCoupon(db, user.ID, coupon.ID)

		detail, err := svc.GetUserCouponDetail(context.Background(), user.ID, userCoupon.ID)
		require.NoError(t, err)
		assert.Equal(t, userCoupon.ID, detail.ID)
		assert.Equal(t, coupon.ID, detail.CouponID)
		assert.Equal(t, coupon.Name, detail.CouponName)
	})

	t.Run("获取不存在的用户优惠券", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		_, err := svc.GetUserCouponDetail(context.Background(), user.ID, 99999)
		assert.Error(t, err)
	})

	t.Run("获取其他用户的优惠券返回错误", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user1 := createUCTestUser(db)
		user2 := createUCTestUser(db)

		coupon := createUCTestCoupon(db)
		userCoupon := createUCTestUserCoupon(db, user1.ID, coupon.ID)

		// user2 尝试获取 user1 的优惠券
		_, err := svc.GetUserCouponDetail(context.Background(), user2.ID, userCoupon.ID)
		assert.ErrorIs(t, err, marketing.ErrUserCouponNotFound)
	})
}

// TestUserCouponService_UseCoupon 测试使用优惠券
func TestUserCouponService_UseCoupon(t *testing.T) {
	t.Run("正常使用优惠券", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		coupon := createUCTestCoupon(db, func(c *models.Coupon) {
			c.Value = 10.0
			c.MinAmount = 50.0
		})
		userCoupon := createUCTestUserCoupon(db, user.ID, coupon.ID)

		orderID := int64(12345)
		orderAmount := 100.0

		usedCoupon, discount, err := svc.UseCoupon(context.Background(), userCoupon.ID, orderID, orderAmount)
		require.NoError(t, err)
		assert.NotNil(t, usedCoupon)
		assert.Equal(t, 10.0, discount)

		// 验证状态已更新
		var updatedUserCoupon models.UserCoupon
		db.First(&updatedUserCoupon, userCoupon.ID)
		assert.Equal(t, int8(models.UserCouponStatusUsed), updatedUserCoupon.Status)
		assert.NotNil(t, updatedUserCoupon.UsedAt)
		assert.Equal(t, orderID, *updatedUserCoupon.OrderID)

		// 验证优惠券使用数量增加
		var updatedCoupon models.Coupon
		db.First(&updatedCoupon, coupon.ID)
		assert.Equal(t, 1, updatedCoupon.UsedCount)
	})

	t.Run("使用已使用的优惠券失败", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		coupon := createUCTestCoupon(db)
		userCoupon := createUCTestUserCoupon(db, user.ID, coupon.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUsed
		})

		_, _, err := svc.UseCoupon(context.Background(), userCoupon.ID, 12345, 100.0)
		assert.ErrorIs(t, err, marketing.ErrUserCouponUsed)
	})

	t.Run("使用已过期的优惠券失败", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		coupon := createUCTestCoupon(db)
		userCoupon := createUCTestUserCoupon(db, user.ID, coupon.ID, func(uc *models.UserCoupon) {
			uc.ExpiredAt = time.Now().Add(-time.Hour) // 已过期
		})

		_, _, err := svc.UseCoupon(context.Background(), userCoupon.ID, 12345, 100.0)
		assert.ErrorIs(t, err, marketing.ErrUserCouponExpired)
	})

	t.Run("订单金额不满足门槛失败", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		coupon := createUCTestCoupon(db, func(c *models.Coupon) {
			c.MinAmount = 100.0 // 门槛100
		})
		userCoupon := createUCTestUserCoupon(db, user.ID, coupon.ID)

		// 订单金额50，不满足门槛
		_, _, err := svc.UseCoupon(context.Background(), userCoupon.ID, 12345, 50.0)
		assert.ErrorIs(t, err, marketing.ErrCouponAmountNotMet)
	})
}

// TestUserCouponService_UnuseCoupon 测试取消使用优惠券
func TestUserCouponService_UnuseCoupon(t *testing.T) {
	t.Run("正常取消使用优惠券", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		coupon := createUCTestCoupon(db)
		// 初始使用数量为1
		db.Model(&coupon).Update("used_count", 1)

		orderID := int64(12345)
		usedAt := time.Now()
		userCoupon := createUCTestUserCoupon(db, user.ID, coupon.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUsed
			uc.OrderID = &orderID
			uc.UsedAt = &usedAt
		})

		err := svc.UnuseCoupon(context.Background(), userCoupon.ID)
		require.NoError(t, err)

		// 验证状态已恢复
		var updatedUserCoupon models.UserCoupon
		db.First(&updatedUserCoupon, userCoupon.ID)
		assert.Equal(t, int8(models.UserCouponStatusUnused), updatedUserCoupon.Status)
		assert.Nil(t, updatedUserCoupon.OrderID)
		assert.Nil(t, updatedUserCoupon.UsedAt)

		// 验证优惠券使用数量减少
		var updatedCoupon models.Coupon
		db.First(&updatedCoupon, coupon.ID)
		assert.Equal(t, 0, updatedCoupon.UsedCount)
	})

	t.Run("取消未使用的优惠券无操作", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		coupon := createUCTestCoupon(db)
		userCoupon := createUCTestUserCoupon(db, user.ID, coupon.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUnused // 未使用
		})

		err := svc.UnuseCoupon(context.Background(), userCoupon.ID)
		require.NoError(t, err) // 不应报错

		// 验证状态未变
		var updatedUserCoupon models.UserCoupon
		db.First(&updatedUserCoupon, userCoupon.ID)
		assert.Equal(t, int8(models.UserCouponStatusUnused), updatedUserCoupon.Status)
	})
}

// TestUserCouponService_ExpireUserCoupons 测试批量过期处理
func TestUserCouponService_ExpireUserCoupons(t *testing.T) {
	t.Run("正常批量标记过期优惠券", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		// 创建已过期但未标记的优惠券
		coupon1 := createUCTestCoupon(db)
		coupon2 := createUCTestCoupon(db)
		coupon3 := createUCTestCoupon(db)

		createUCTestUserCoupon(db, user.ID, coupon1.ID, func(uc *models.UserCoupon) {
			uc.ExpiredAt = time.Now().Add(-time.Hour) // 已过期
			uc.Status = models.UserCouponStatusUnused
		})
		createUCTestUserCoupon(db, user.ID, coupon2.ID, func(uc *models.UserCoupon) {
			uc.ExpiredAt = time.Now().Add(-time.Hour) // 已过期
			uc.Status = models.UserCouponStatusUnused
		})
		createUCTestUserCoupon(db, user.ID, coupon3.ID, func(uc *models.UserCoupon) {
			uc.ExpiredAt = time.Now().Add(time.Hour) // 未过期
			uc.Status = models.UserCouponStatusUnused
		})

		affected, err := svc.ExpireUserCoupons(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(2), affected)

		// 验证已过期的被标记
		var expiredCoupons []models.UserCoupon
		db.Where("status = ?", models.UserCouponStatusExpired).Find(&expiredCoupons)
		assert.Len(t, expiredCoupons, 2)
	})
}

// TestUserCouponService_GetCouponCountByStatus 测试获取各状态优惠券数量
func TestUserCouponService_GetCouponCountByStatus(t *testing.T) {
	t.Run("正常获取各状态数量", func(t *testing.T) {
		db := setupUserCouponServiceTestDB(t)
		svc := createUserCouponTestService(db)
		user := createUCTestUser(db)

		// 创建不同状态的用户优惠券
		coupon1 := createUCTestCoupon(db)
		coupon2 := createUCTestCoupon(db)
		coupon3 := createUCTestCoupon(db)
		coupon4 := createUCTestCoupon(db)
		coupon5 := createUCTestCoupon(db)

		// 2个未使用
		createUCTestUserCoupon(db, user.ID, coupon1.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUnused
		})
		createUCTestUserCoupon(db, user.ID, coupon2.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUnused
		})

		// 2个已使用
		createUCTestUserCoupon(db, user.ID, coupon3.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUsed
		})
		createUCTestUserCoupon(db, user.ID, coupon4.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusUsed
		})

		// 1个已过期
		createUCTestUserCoupon(db, user.ID, coupon5.ID, func(uc *models.UserCoupon) {
			uc.Status = models.UserCouponStatusExpired
		})

		counts, err := svc.GetCouponCountByStatus(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), counts["unused"])
		assert.Equal(t, int64(2), counts["used"])
		assert.Equal(t, int64(1), counts["expired"])
	})
}
