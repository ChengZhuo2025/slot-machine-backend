//go:build integration
// +build integration

// Package integration 会员体系集成测试（消费→积分→升级→权益生效）
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
	orderService "github.com/dumeirei/smart-locker-backend/internal/service/order"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

func setupMembershipIntegrationDB(t *testing.T) *gorm.DB {
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
		Discount:  0.8,
		Benefits:  models.JSON{"free_shipping": true},
	})

	return db
}

func TestMembershipFlow_ConsumePoints_UpgradeAndBenefits(t *testing.T) {
	db := setupMembershipIntegrationDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)

	pointsSvc := userService.NewPointsService(db, userRepo, levelRepo)
	memberLevelSvc := userService.NewMemberLevelService(db, userRepo, levelRepo)
	memberDiscountSvc := orderService.NewMemberDiscountService(db, userRepo, levelRepo)
	pointsHook := orderService.NewPointsHook(db, pointsSvc)

	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "集成测试用户",
		MemberLevelID: 1,
		Points:        0,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)

	order := &models.Order{
		OrderNo:        "O-INTEGRATION-001",
		UserID:         user.ID,
		Type:           models.OrderTypeRental,
		OriginalAmount: 120,
		DiscountAmount: 0,
		ActualAmount:   120,
		Status:         models.OrderStatusCompleted,
		CompletedAt:    func() *time.Time { now := time.Now(); return &now }(),
	}
	require.NoError(t, db.Create(order).Error)

	// 1) 订单完成触发积分累积 + 自动升级
	require.NoError(t, pointsHook.OnOrderCompleted(ctx, order))

	var refreshed models.User
	require.NoError(t, db.First(&refreshed, user.ID).Error)
	assert.Equal(t, 120, refreshed.Points)
	assert.Equal(t, int64(2), refreshed.MemberLevelID)

	// 2) 权益生效（会员信息与权益）
	info, err := memberLevelSvc.GetUserMemberInfo(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, info.CurrentLevel)
	assert.Equal(t, "黄金会员", info.CurrentLevel.Name)
	require.NotNil(t, info.CurrentLevel.Benefits)
	assert.Equal(t, true, info.CurrentLevel.Benefits["free_shipping"])

	// 3) 折扣生效（下单时可用的会员折扣）
	discount, err := memberDiscountSvc.CalculateMemberDiscount(ctx, user.ID, 100)
	require.NoError(t, err)
	assert.True(t, discount.HasMemberDiscount)
	assert.Equal(t, 0.8, discount.DiscountRate)
	assert.Equal(t, 80.0, discount.FinalAmount)
	assert.Equal(t, 20.0, discount.DiscountAmount)
}

