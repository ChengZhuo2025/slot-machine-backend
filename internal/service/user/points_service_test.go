// Package user 积分服务单元测试
package user

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

func setupPointsServiceTestDB(t *testing.T) *gorm.DB {
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
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})
	db.Create(&models.MemberLevel{ID: 2, Name: "黄金会员", Level: 2, MinPoints: 100, Discount: 0.9})

	return db
}

func newPointsServiceForTest(db *gorm.DB) (*PointsService, *repository.UserRepository, *repository.MemberLevelRepository) {
	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)
	return NewPointsService(db, userRepo, levelRepo), userRepo, levelRepo
}

func createTestUserForPoints(db *gorm.DB, points int, memberLevelID int64) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: memberLevelID,
		Points:        points,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	return user
}

func TestPointsService_GetPointsInfo(t *testing.T) {
	db := setupPointsServiceTestDB(t)
	svc, _, _ := newPointsServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForPoints(db, 50, 1)

	info, err := svc.GetPointsInfo(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 50, info.Points)
	assert.Equal(t, int64(1), info.MemberLevelID)
	assert.Equal(t, "普通会员", info.MemberLevelName)
	assert.Equal(t, 1.0, info.Discount)
	require.NotNil(t, info.NextLevelName)
	assert.Equal(t, "黄金会员", *info.NextLevelName)
	require.NotNil(t, info.NextLevelPoints)
	assert.Equal(t, 100, *info.NextLevelPoints)
	require.NotNil(t, info.PointsToNextLevel)
	assert.Equal(t, 50, *info.PointsToNextLevel)
}

func TestPointsService_AddConsumePoints_UpgradesLevelAndWritesHistory(t *testing.T) {
	db := setupPointsServiceTestDB(t)
	svc, _, _ := newPointsServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForPoints(db, 0, 1)

	err := svc.AddConsumePoints(ctx, user.ID, 120.5, "O202401010001")
	require.NoError(t, err)

	var refreshed models.User
	require.NoError(t, db.First(&refreshed, user.ID).Error)
	assert.Equal(t, 120, refreshed.Points)           // int(120.5) => 120
	assert.Equal(t, int64(2), refreshed.MemberLevelID) // 触发升级

	var txCount int64
	require.NoError(t, db.Model(&models.WalletTransaction{}).Where("user_id = ? AND type = ?", user.ID, "points_consume").Count(&txCount).Error)
	assert.Equal(t, int64(1), txCount)
}

func TestPointsService_AddPoints_InvalidParams(t *testing.T) {
	db := setupPointsServiceTestDB(t)
	svc, _, _ := newPointsServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForPoints(db, 0, 1)

	err := svc.AddPoints(ctx, user.ID, 0, PointsTypeAdmin, "x", nil)
	require.Error(t, err)
}

func TestPointsService_DeductPoints_Insufficient(t *testing.T) {
	db := setupPointsServiceTestDB(t)
	svc, _, _ := newPointsServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForPoints(db, 10, 1)

	err := svc.DeductPoints(ctx, user.ID, 20, PointsTypeExchange, "兑换", nil)
	require.Error(t, err)
}

func TestPointsService_GetPointsHistory_FilterAndOrder(t *testing.T) {
	db := setupPointsServiceTestDB(t)
	svc, _, _ := newPointsServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForPoints(db, 0, 1)
	require.NoError(t, svc.AddPoints(ctx, user.ID, 10, PointsTypeAdmin, "管理员调整", nil))
	require.NoError(t, svc.AddConsumePoints(ctx, user.ID, 5, "O2"))
	require.NoError(t, svc.DeductPoints(ctx, user.ID, 3, PointsTypeExchange, "兑换", nil))

	records, total, err := svc.GetPointsHistory(ctx, user.ID, 0, 10, "")
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, records, 3)
	assert.True(t, records[0].ID > records[1].ID) // 倒序

	filtered, filteredTotal, err := svc.GetPointsHistory(ctx, user.ID, 0, 10, PointsTypeAdmin)
	require.NoError(t, err)
	assert.Equal(t, int64(1), filteredTotal)
	require.Len(t, filtered, 1)
	assert.Equal(t, "points_admin", filtered[0].Type)
	assert.Equal(t, 10, filtered[0].Points)
}

