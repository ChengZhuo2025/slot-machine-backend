// Package order 会员折扣服务单元测试
package order

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

func setupMemberDiscountTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.MemberLevel{},
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})
	db.Create(&models.MemberLevel{ID: 2, Name: "黄金会员", Level: 2, MinPoints: 100, Discount: 0.9})
	db.Create(&models.MemberLevel{ID: 3, Name: "钻石会员", Level: 3, MinPoints: 200, Discount: 0.95})

	return db
}

func createMemberDiscountTestUser(db *gorm.DB, memberLevelID int64) *models.User {
	phone := fmt.Sprintf("136%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "折扣用户",
		MemberLevelID: memberLevelID,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	return user
}

func TestMemberDiscountService_CalculateMemberDiscount(t *testing.T) {
	db := setupMemberDiscountTestDB(t)
	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)
	svc := NewMemberDiscountService(db, userRepo, levelRepo)
	ctx := context.Background()

	t.Run("金额<=0返回原值", func(t *testing.T) {
		r, err := svc.CalculateMemberDiscount(ctx, 1, 0)
		require.NoError(t, err)
		assert.Equal(t, 0.0, r.FinalAmount)
		assert.False(t, r.HasMemberDiscount)
	})

	t.Run("无会员折扣不打折", func(t *testing.T) {
		user := createMemberDiscountTestUser(db, 1)
		r, err := svc.CalculateMemberDiscount(ctx, user.ID, 100)
		require.NoError(t, err)
		assert.Equal(t, 100.0, r.FinalAmount)
		assert.False(t, r.HasMemberDiscount)
	})

	t.Run("有会员折扣向下保留两位小数", func(t *testing.T) {
		user := createMemberDiscountTestUser(db, 2) // 0.9
		r, err := svc.CalculateMemberDiscount(ctx, user.ID, 99.99)
		require.NoError(t, err)
		assert.True(t, r.HasMemberDiscount)
		assert.Equal(t, 89.99, r.FinalAmount)
		assert.Equal(t, 9.99, r.DiscountAmount)
	})
}

func TestMemberDiscountService_GetDiscountDescription(t *testing.T) {
	db := setupMemberDiscountTestDB(t)
	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)
	svc := NewMemberDiscountService(db, userRepo, levelRepo)

	assert.Equal(t, "", svc.GetDiscountDescription(1.0, "普通会员"))
	assert.Equal(t, "黄金会员享9折", svc.GetDiscountDescription(0.9, "黄金会员"))
	assert.Equal(t, "钻石会员享9.5折", svc.GetDiscountDescription(0.95, "钻石会员"))
}

func setupEnhancedMemberDiscountTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Coupon{},
		&models.UserCoupon{},
		&models.Campaign{},
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})
	db.Create(&models.MemberLevel{ID: 2, Name: "黄金会员", Level: 2, MinPoints: 100, Discount: 0.9})

	return db
}

func createEnhancedMemberDiscountTestUser(t *testing.T, db *gorm.DB, memberLevelID int64) *models.User {
	t.Helper()

	phone := fmt.Sprintf("136%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "折扣用户",
		MemberLevelID: memberLevelID,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&models.UserWallet{UserID: user.ID, Balance: 1000.0}).Error)
	return user
}

func TestMemberDiscountService_CalculateWithMemberDiscount(t *testing.T) {
	db := setupEnhancedMemberDiscountTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)
	svc := NewMemberDiscountService(db, userRepo, levelRepo)

	discountCalc := setupDiscountCalculator(db)

	t.Run("无 DiscountCalculator 时仅计算会员折扣", func(t *testing.T) {
		user := createEnhancedMemberDiscountTestUser(t, db, 2) // 0.9
		result, err := svc.CalculateWithMemberDiscount(ctx, user.ID, 200.0, nil, models.OrderTypeMall, nil)
		require.NoError(t, err)

		assert.Equal(t, 200.0, result.OriginalAmount)
		assert.Equal(t, 180.0, result.FinalAmount)
		assert.Equal(t, 20.0, result.MemberDiscount)
		assert.Equal(t, 20.0, result.TotalDiscount)
		require.NotNil(t, result.MemberLevel)
		assert.Equal(t, int64(2), result.MemberLevel.ID)
		assert.Equal(t, "黄金会员", result.MemberLevel.Name)
		assert.Equal(t, 0.9, result.MemberLevel.Discount)
		require.Len(t, result.DiscountDetails, 1)
		assert.Equal(t, "member", result.DiscountDetails[0].Type)
	})

	t.Run("会员折扣后再计算活动与优惠券优惠", func(t *testing.T) {
		user := createEnhancedMemberDiscountTestUser(t, db, 2) // 0.9

		rules := models.JSON{
			"rules": []map[string]interface{}{
				{"min_amount": 100.0, "discount": 10.0},
				{"min_amount": 180.0, "discount": 20.0},
			},
		}
		createTestCampaign(t, db, func(c *models.Campaign) {
			c.Name = "满100减10，满180减20"
			c.Type = models.CampaignTypeDiscount
			c.Rules = rules
		})

		coupon := createTestCouponForDiscount(t, db, func(c *models.Coupon) {
			c.Name = "满150减5"
			c.Value = 5.0
			c.MinAmount = 150.0
		})
		userCoupon := &models.UserCoupon{
			UserID:     user.ID,
			CouponID:   coupon.ID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  time.Now().Add(24 * time.Hour),
			ReceivedAt: time.Now(),
		}
		require.NoError(t, db.Create(userCoupon).Error)

		result, err := svc.CalculateWithMemberDiscount(ctx, user.ID, 200.0, discountCalc, models.OrderTypeMall, &userCoupon.ID)
		require.NoError(t, err)

		// 原始 200 -> 会员折扣 0.9 后 180 -> 活动 20 -> 优惠券 5 => 155
		assert.Equal(t, 200.0, result.OriginalAmount)
		assert.Equal(t, 155.0, result.FinalAmount)
		assert.Equal(t, 20.0, result.MemberDiscount)
		assert.Equal(t, 20.0, result.CampaignDiscount)
		assert.Equal(t, 5.0, result.CouponDiscount)
		assert.Equal(t, 45.0, result.TotalDiscount)

		require.Len(t, result.DiscountDetails, 3)
		assert.Equal(t, "member", result.DiscountDetails[0].Type)
		assert.Equal(t, "campaign", result.DiscountDetails[1].Type)
		assert.Equal(t, "coupon", result.DiscountDetails[2].Type)
	})
}
