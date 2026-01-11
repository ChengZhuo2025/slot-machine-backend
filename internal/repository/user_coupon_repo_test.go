// Package repository 用户优惠券仓储单元测试
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

func setupUserCouponTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.UserCoupon{}, &models.Coupon{}, &models.User{}, &models.Order{})
	require.NoError(t, err)

	return db
}

func TestUserCouponRepository_Create(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	userCoupon := &models.UserCoupon{
		UserID:    1,
		CouponID:  1,
		Status:    models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	}

	err := repo.Create(ctx, userCoupon)
	require.NoError(t, err)
	assert.NotZero(t, userCoupon.ID)
}

func TestUserCouponRepository_GetByID(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	userCoupon := &models.UserCoupon{
		UserID:    1,
		CouponID:  1,
		Status:    models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	}
	db.Create(userCoupon)

	found, err := repo.GetByID(ctx, userCoupon.ID)
	require.NoError(t, err)
	assert.Equal(t, userCoupon.ID, found.ID)
}

func TestUserCouponRepository_GetByIDWithCoupon(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	coupon := &models.Coupon{
		Name:            "测试优惠券",
		Type:            "discount",
		Value:           10.0,
		MinAmount:       100.0,
		TotalCount:      100,
		ApplicableScope: models.CouponScopeAll,
		StartTime:       time.Now(),
		EndTime:         time.Now().AddDate(0, 1, 0),
		ApplicableIDs:   models.JSON{},
	}
	db.Create(coupon)

	userCoupon := &models.UserCoupon{
		UserID:    1,
		CouponID:  coupon.ID,
		Status:    models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	}
	db.Create(userCoupon)

	found, err := repo.GetByIDWithCoupon(ctx, userCoupon.ID)
	require.NoError(t, err)
	assert.Equal(t, userCoupon.ID, found.ID)
	assert.NotNil(t, found.Coupon)
	assert.Equal(t, coupon.ID, found.Coupon.ID)
}

func TestUserCouponRepository_GetByUserIDAndCouponID(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	found, err := repo.GetByUserIDAndCouponID(ctx, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), found.UserID)
	assert.Equal(t, int64(1), found.CouponID)
}

func TestUserCouponRepository_Update(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	userCoupon := &models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	}
	db.Create(userCoupon)

	userCoupon.Status = models.UserCouponStatusUsed
	err := repo.Update(ctx, userCoupon)
	require.NoError(t, err)

	var found models.UserCoupon
	db.First(&found, userCoupon.ID)
	assert.Equal(t, int8(models.UserCouponStatusUsed), found.Status)
}

func TestUserCouponRepository_UpdateFields(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	userCoupon := &models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	}
	db.Create(userCoupon)

	orderID := int64(100)
	err := repo.UpdateFields(ctx, userCoupon.ID, map[string]interface{}{
		"order_id": orderID,
	})
	require.NoError(t, err)

	var found models.UserCoupon
	db.First(&found, userCoupon.ID)
	assert.NotNil(t, found.OrderID)
	assert.Equal(t, int64(100), *found.OrderID)
}

func TestUserCouponRepository_Delete(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	userCoupon := &models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	}
	db.Create(userCoupon)

	err := repo.Delete(ctx, userCoupon.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.UserCoupon{}).Where("id = ?", userCoupon.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestUserCouponRepository_List(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 2, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	db.Model(&models.UserCoupon{}).Create(map[string]interface{}{
		"user_id": 2, "coupon_id": 1, "status": models.UserCouponStatusUnused,
		"expired_at": time.Now().AddDate(0, 0, 7),
	})

	// 获取所有用户优惠券
	_, total, err := repo.List(ctx, UserCouponListParams{Offset: 0, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按用户过滤
	_, total, err = repo.List(ctx, UserCouponListParams{UserID: 1, Offset: 0, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按优惠券过滤
	_, total, err = repo.List(ctx, UserCouponListParams{CouponID: 1, Offset: 0, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按状态过滤
	status := int8(models.UserCouponStatusUnused)
	_, total, err = repo.List(ctx, UserCouponListParams{Status: &status, Offset: 0, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestUserCouponRepository_ListByUserID(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 2, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	_, total, err := repo.ListByUserID(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestUserCouponRepository_ListByUserIDAndStatus(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 2, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	_, total, err := repo.ListByUserIDAndStatus(ctx, 1, models.UserCouponStatusUnused, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestUserCouponRepository_ListAvailableByUserID(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	// 可用（未使用且未过期）
	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	// 已使用
	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 2, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	// 已过期（时间过期）
	db.Exec("INSERT INTO user_coupons (user_id, coupon_id, status, expired_at) VALUES (?, ?, ?, ?)",
		1, 3, models.UserCouponStatusUnused, time.Now().Add(-24*time.Hour))

	_, total, err := repo.ListAvailableByUserID(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total) // 只有1张可用
}

func TestUserCouponRepository_ListAvailableForOrder(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	// 创建优惠券
	coupon1 := &models.Coupon{
		Name: "全场通用", Type: "discount", Value: 10.0, MinAmount: 100.0, TotalCount: 100,
		ApplicableScope: models.CouponScopeAll, StartTime: time.Now(), EndTime: time.Now().AddDate(0, 1, 0),
		ApplicableIDs: models.JSON{},
	}
	db.Create(coupon1)

	coupon2 := &models.Coupon{
		Name: "分类优惠", Type: "discount", Value: 20.0, MinAmount: 200.0, TotalCount: 100,
		ApplicableScope: models.CouponScopeCategory, StartTime: time.Now(), EndTime: time.Now().AddDate(0, 1, 0),
		ApplicableIDs: models.JSON{},
	}
	db.Create(coupon2)

	// 创建用户优惠券
	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: coupon1.ID, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: coupon2.ID, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	// 订单金额150，能使用全场通用但不能使用分类优惠（最低200）
	coupons, err := repo.ListAvailableForOrder(ctx, 1, models.CouponScopeAll, 150.0)
	require.NoError(t, err)
	assert.Equal(t, 1, len(coupons))
}

func TestUserCouponRepository_MarkAsUsed(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	userCoupon := &models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	}
	db.Create(userCoupon)

	err := repo.MarkAsUsed(ctx, userCoupon.ID, 100)
	require.NoError(t, err)

	var found models.UserCoupon
	db.First(&found, userCoupon.ID)
	assert.Equal(t, int8(models.UserCouponStatusUsed), found.Status)
	assert.NotNil(t, found.OrderID)
	assert.Equal(t, int64(100), *found.OrderID)
	assert.NotNil(t, found.UsedAt)
}

func TestUserCouponRepository_MarkAsUnused(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	now := time.Now()
	orderID := int64(100)
	userCoupon := &models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7), OrderID: &orderID, UsedAt: &now,
	}
	db.Create(userCoupon)

	err := repo.MarkAsUnused(ctx, userCoupon.ID)
	require.NoError(t, err)

	var found models.UserCoupon
	db.First(&found, userCoupon.ID)
	assert.Equal(t, int8(models.UserCouponStatusUnused), found.Status)
	assert.Nil(t, found.OrderID)
	assert.Nil(t, found.UsedAt)
}

func TestUserCouponRepository_MarkAsExpired(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	userCoupon := &models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	}
	db.Create(userCoupon)

	err := repo.MarkAsExpired(ctx, userCoupon.ID)
	require.NoError(t, err)

	var found models.UserCoupon
	db.First(&found, userCoupon.ID)
	assert.Equal(t, int8(models.UserCouponStatusExpired), found.Status)
}

func TestUserCouponRepository_BatchMarkAsExpired(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	// 创建未过期的
	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	// 创建已过期的（2张）
	db.Exec("INSERT INTO user_coupons (user_id, coupon_id, status, expired_at) VALUES (?, ?, ?, ?)",
		1, 2, models.UserCouponStatusUnused, time.Now().Add(-24*time.Hour))
	db.Exec("INSERT INTO user_coupons (user_id, coupon_id, status, expired_at) VALUES (?, ?, ?, ?)",
		1, 3, models.UserCouponStatusUnused, time.Now().Add(-48*time.Hour))

	affected, err := repo.BatchMarkAsExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)
}

func TestUserCouponRepository_CountByUserIDAndCouponID(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	count, err := repo.CountByUserIDAndCouponID(ctx, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestUserCouponRepository_CountByUserID(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 2, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	count, err := repo.CountByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestUserCouponRepository_CountAvailableByUserID(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	// 可用
	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUnused,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	// 已使用
	db.Create(&models.UserCoupon{
		UserID: 1, CouponID: 2, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
	})

	// 已过期
	db.Exec("INSERT INTO user_coupons (user_id, coupon_id, status, expired_at) VALUES (?, ?, ?, ?)",
		1, 3, models.UserCouponStatusUnused, time.Now().Add(-24*time.Hour))

	count, err := repo.CountAvailableByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestUserCouponRepository_GetByOrderID(t *testing.T) {
	db := setupUserCouponTestDB(t)
	repo := NewUserCouponRepository(db)
	ctx := context.Background()

	orderID := int64(100)
	userCoupon := &models.UserCoupon{
		UserID: 1, CouponID: 1, Status: models.UserCouponStatusUsed,
		ExpiredAt: time.Now().AddDate(0, 0, 7), OrderID: &orderID,
	}
	db.Create(userCoupon)

	found, err := repo.GetByOrderID(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, userCoupon.ID, found.ID)
	assert.NotNil(t, found.OrderID)
	assert.Equal(t, int64(100), *found.OrderID)
}
