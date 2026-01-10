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
