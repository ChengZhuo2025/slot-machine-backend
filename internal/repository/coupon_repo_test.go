// Package repository 优惠券仓储单元测试
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

// setupCouponTestDB 创建优惠券测试数据库
func setupCouponTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Coupon{},
		&models.UserCoupon{},
		&models.User{},
		&models.MemberLevel{},
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

func createCouponTestUser(t *testing.T, db *gorm.DB, phone string) *models.User {
	t.Helper()

	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createTestCouponForRepo(t *testing.T, db *gorm.DB, opts ...func(*models.Coupon)) *models.Coupon {
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

func TestCouponRepository_Create(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	coupon := &models.Coupon{
		Name:            "新建优惠券",
		Type:            models.CouponTypeFixed,
		Value:           15.0,
		MinAmount:       100.0,
		TotalCount:      50,
		PerUserLimit:    2,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(7 * 24 * time.Hour),
		Status:          models.CouponStatusActive,
	}

	err := repo.Create(ctx, coupon)
	require.NoError(t, err)
	assert.NotZero(t, coupon.ID)

	// 验证优惠券已创建
	var found models.Coupon
	db.First(&found, coupon.ID)
	assert.Equal(t, "新建优惠券", found.Name)
}

func TestCouponRepository_GetByID(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	coupon := createTestCouponForRepo(t, db)

	t.Run("获取存在的优惠券", func(t *testing.T) {
		found, err := repo.GetByID(ctx, coupon.ID)
		require.NoError(t, err)
		assert.Equal(t, coupon.ID, found.ID)
		assert.Equal(t, coupon.Name, found.Name)
	})

	t.Run("获取不存在的优惠券", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestCouponRepository_Update(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	coupon := createTestCouponForRepo(t, db)

	coupon.Name = "更新后优惠券"
	err := repo.Update(ctx, coupon)
	require.NoError(t, err)

	var found models.Coupon
	db.First(&found, coupon.ID)
	assert.Equal(t, "更新后优惠券", found.Name)
}

func TestCouponRepository_UpdateFields(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	coupon := createTestCouponForRepo(t, db)

	err := repo.UpdateFields(ctx, coupon.ID, map[string]interface{}{
		"name":   "部分更新优惠券",
		"status": models.CouponStatusDisabled,
	})
	require.NoError(t, err)

	var found models.Coupon
	db.First(&found, coupon.ID)
	assert.Equal(t, "部分更新优惠券", found.Name)
	assert.Equal(t, int8(models.CouponStatusDisabled), found.Status)
}

func TestCouponRepository_Delete(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	coupon := createTestCouponForRepo(t, db)

	err := repo.Delete(ctx, coupon.ID)
	require.NoError(t, err)

	var found models.Coupon
	result := db.First(&found, coupon.ID)
	assert.Error(t, result.Error)
}

func TestCouponRepository_List(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	// 创建多个优惠券
	createTestCouponForRepo(t, db, func(c *models.Coupon) { c.Name = "优惠券1" })
	createTestCouponForRepo(t, db, func(c *models.Coupon) { c.Name = "优惠券2" })
	createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.Name = "优惠券3"
		c.Status = models.CouponStatusDisabled
	})

	t.Run("获取所有优惠券", func(t *testing.T) {
		coupons, total, err := repo.List(ctx, CouponListParams{
			Offset: 0,
			Limit:  10,
		})
		require.NoError(t, err)
		assert.True(t, total >= 3)
		assert.True(t, len(coupons) >= 3)
	})

	t.Run("分页获取", func(t *testing.T) {
		coupons, total, err := repo.List(ctx, CouponListParams{
			Offset: 0,
			Limit:  2,
		})
		require.NoError(t, err)
		assert.True(t, total >= 3)
		assert.Len(t, coupons, 2)
	})

	t.Run("按状态筛选", func(t *testing.T) {
		status := int8(models.CouponStatusActive)
		coupons, _, err := repo.List(ctx, CouponListParams{
			Offset: 0,
			Limit:  10,
			Status: &status,
		})
		require.NoError(t, err)
		for _, c := range coupons {
			assert.Equal(t, int8(models.CouponStatusActive), c.Status)
		}
	})

	t.Run("按关键词搜索", func(t *testing.T) {
		coupons, _, err := repo.List(ctx, CouponListParams{
			Offset:  0,
			Limit:   10,
			Keyword: "优惠券1",
		})
		require.NoError(t, err)
		for _, c := range coupons {
			assert.Contains(t, c.Name, "优惠券1")
		}
	})
}

func TestCouponRepository_ListActive(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	// 创建有效优惠券
	createTestCouponForRepo(t, db, func(c *models.Coupon) { c.Name = "有效优惠券" })

	// 创建过期优惠券
	createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.Name = "过期优惠券"
		c.EndTime = time.Now().Add(-time.Hour)
	})

	// 创建已领完优惠券
	createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.Name = "已领完优惠券"
		c.TotalCount = 10
		c.ReceivedCount = 10
	})

	coupons, _, err := repo.ListActive(ctx, 0, 10)
	require.NoError(t, err)

	// 只应返回有效且未过期且未领完的优惠券
	for _, c := range coupons {
		assert.Equal(t, int8(models.CouponStatusActive), c.Status)
		assert.True(t, c.EndTime.After(time.Now()))
		assert.True(t, c.TotalCount > c.ReceivedCount)
	}
}

func TestCouponRepository_ListAvailableForUser(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	user := createCouponTestUser(t, db, "13800138000")

	// 创建可领取优惠券
	availableCoupon := createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.Name = "可领取优惠券"
		c.PerUserLimit = 3
	})

	// 创建已领满的优惠券
	fullCoupon := createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.Name = "已领满优惠券"
		c.PerUserLimit = 1
	})

	// 用户已领取一张 fullCoupon
	userCoupon := &models.UserCoupon{
		UserID:     user.ID,
		CouponID:   fullCoupon.ID,
		Status:     models.UserCouponStatusUnused,
		ExpiredAt:  time.Now().Add(24 * time.Hour),
		ReceivedAt: time.Now(),
	}
	db.Create(userCoupon)

	coupons, _, err := repo.ListAvailableForUser(ctx, user.ID, 0, 10)
	require.NoError(t, err)

	// 应包含可领取的，不包含已领满的
	foundAvailable := false
	foundFull := false
	for _, c := range coupons {
		if c.ID == availableCoupon.ID {
			foundAvailable = true
		}
		if c.ID == fullCoupon.ID {
			foundFull = true
		}
	}
	assert.True(t, foundAvailable)
	assert.False(t, foundFull)
}

func TestCouponRepository_IncrementIssuedCount(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	t.Run("正常增加已发放数量", func(t *testing.T) {
		coupon := createTestCouponForRepo(t, db, func(c *models.Coupon) {
			c.TotalCount = 10
			c.ReceivedCount = 0
		})

		err := repo.IncrementIssuedCount(ctx, coupon.ID)
		require.NoError(t, err)

		var found models.Coupon
		db.First(&found, coupon.ID)
		assert.Equal(t, 1, found.ReceivedCount)
	})

	t.Run("已领完时增加失败", func(t *testing.T) {
		coupon := createTestCouponForRepo(t, db, func(c *models.Coupon) {
			c.TotalCount = 10
			c.ReceivedCount = 10
		})

		err := repo.IncrementIssuedCount(ctx, coupon.ID)
		assert.Error(t, err)
	})
}

func TestCouponRepository_IncrementUsedCount(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	coupon := createTestCouponForRepo(t, db)

	err := repo.IncrementUsedCount(ctx, coupon.ID)
	require.NoError(t, err)

	var found models.Coupon
	db.First(&found, coupon.ID)
	assert.Equal(t, 1, found.UsedCount)
}

func TestCouponRepository_DecrementIssuedCount(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	coupon := createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.ReceivedCount = 5
	})

	err := repo.DecrementIssuedCount(ctx, coupon.ID)
	require.NoError(t, err)

	var found models.Coupon
	db.First(&found, coupon.ID)
	assert.Equal(t, 4, found.ReceivedCount)
}

func TestCouponRepository_DecrementUsedCount(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	coupon := createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.UsedCount = 3
	})

	err := repo.DecrementUsedCount(ctx, coupon.ID)
	require.NoError(t, err)

	var found models.Coupon
	db.First(&found, coupon.ID)
	assert.Equal(t, 2, found.UsedCount)
}

func TestCouponRepository_GetUserReceivedCount(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	user := createCouponTestUser(t, db, "13800138001")
	coupon := createTestCouponForRepo(t, db)

	// 用户领取3张
	for i := 0; i < 3; i++ {
		userCoupon := &models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		}
		db.Create(userCoupon)
	}

	count, err := repo.GetUserReceivedCount(ctx, user.ID, coupon.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestCouponRepository_ListByApplicableType(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	// 创建不同适用类型的优惠券
	createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.Name = "分类优惠券"
		c.ApplicableScope = models.CouponScopeCategory
	})
	createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.Name = "商品优惠券"
		c.ApplicableScope = models.CouponScopeProduct
	})
	createTestCouponForRepo(t, db, func(c *models.Coupon) {
		c.Name = "通用优惠券"
		c.ApplicableScope = models.CouponScopeAll
	})

	t.Run("获取分类优惠券", func(t *testing.T) {
		coupons, err := repo.ListByApplicableType(ctx, models.CouponScopeCategory)
		require.NoError(t, err)
		// 应返回分类专用和通用优惠券
		assert.True(t, len(coupons) >= 2)
	})

	t.Run("获取商品优惠券", func(t *testing.T) {
		coupons, err := repo.ListByApplicableType(ctx, models.CouponScopeProduct)
		require.NoError(t, err)
		// 应返回商品专用和通用优惠券
		assert.True(t, len(coupons) >= 2)
	})
}

func TestCouponRepository_EdgeCases(t *testing.T) {
	db := setupCouponTestDB(t)
	repo := NewCouponRepository(db)
	ctx := context.Background()

	t.Run("空优惠券列表", func(t *testing.T) {
		// 搜索不存在的关键词
		coupons, total, err := repo.List(ctx, CouponListParams{
			Offset:  0,
			Limit:   10,
			Keyword: "不存在的优惠券名称xyz",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, coupons)
	})

	t.Run("用户未领取任何优惠券", func(t *testing.T) {
		user := createCouponTestUser(t, db, "13800138002")
		coupon := createTestCouponForRepo(t, db)

		count, err := repo.GetUserReceivedCount(ctx, user.ID, coupon.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("减少已为0的发放数量", func(t *testing.T) {
		coupon := createTestCouponForRepo(t, db, func(c *models.Coupon) {
			c.ReceivedCount = 0
		})

		// 不应报错，但数量不变
		err := repo.DecrementIssuedCount(ctx, coupon.ID)
		require.NoError(t, err)

		var found models.Coupon
		db.First(&found, coupon.ID)
		assert.Equal(t, 0, found.ReceivedCount)
	})
}
