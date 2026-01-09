//go:build e2e
// +build e2e

// Package e2e 会员体系与权益管理 E2E 测试（业务链路）
package e2e

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
	orderService "github.com/dumeirei/smart-locker-backend/internal/service/order"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

type membershipE2ETestContext struct {
	db                *gorm.DB
	pointsSvc         *userService.PointsService
	memberLevelSvc    *userService.MemberLevelService
	memberDiscountSvc *orderService.MemberDiscountService
	pointsHook        *orderService.PointsHook
}

func setupMembershipE2ETestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.MemberLevel{},
		&models.WalletTransaction{},
		&models.Order{},
	))

	db.Create(&models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
		Benefits:  models.JSON{"free_shipping": false},
	})
	db.Create(&models.MemberLevel{
		ID:        2,
		Name:      "黄金会员",
		Level:     2,
		MinPoints: 100,
		Discount:  0.85,
		Benefits:  models.JSON{"free_shipping": true},
	})

	return db
}

func setupMembershipE2EContext(t *testing.T) *membershipE2ETestContext {
	db := setupMembershipE2ETestDB(t)
	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)

	pointsSvc := userService.NewPointsService(db, userRepo, levelRepo)
	memberLevelSvc := userService.NewMemberLevelService(db, userRepo, levelRepo)
	memberDiscountSvc := orderService.NewMemberDiscountService(db, userRepo, levelRepo)
	pointsHook := orderService.NewPointsHook(db, pointsSvc)

	return &membershipE2ETestContext{
		db:                db,
		pointsSvc:         pointsSvc,
		memberLevelSvc:    memberLevelSvc,
		memberDiscountSvc: memberDiscountSvc,
		pointsHook:        pointsHook,
	}
}

func createMembershipE2EUser(t *testing.T, db *gorm.DB) *models.User {
	phone := fmt.Sprintf("135%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "E2E会员用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func TestMembershipE2E_ConsumeToUpgradeToDiscount(t *testing.T) {
	tc := setupMembershipE2EContext(t)
	ctx := context.Background()

	user := createMembershipE2EUser(t, tc.db)

	order := &models.Order{
		OrderNo:        "O-E2E-001",
		UserID:         user.ID,
		Type:           models.OrderTypeMall,
		OriginalAmount: 150,
		DiscountAmount: 0,
		ActualAmount:   150,
		Status:         models.OrderStatusCompleted,
	}
	require.NoError(t, tc.db.Create(order).Error)

	// 订单完成触发积分与升级
	require.NoError(t, tc.pointsHook.OnOrderCompleted(ctx, order))

	// 权益确认（等级/权益）
	info, err := tc.memberLevelSvc.GetUserMemberInfo(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, info.CurrentLevel)
	assert.Equal(t, "黄金会员", info.CurrentLevel.Name)
	assert.Equal(t, true, info.CurrentLevel.Benefits["free_shipping"])

	// 折扣确认（订单创建时可计算）
	discount, err := tc.memberDiscountSvc.CalculateMemberDiscount(ctx, user.ID, 200)
	require.NoError(t, err)
	assert.True(t, discount.HasMemberDiscount)
	assert.Equal(t, 0.85, discount.DiscountRate)
	assert.Equal(t, 170.0, discount.FinalAmount)
	assert.Equal(t, 30.0, discount.DiscountAmount)
}

