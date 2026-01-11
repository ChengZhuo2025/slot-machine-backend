// Package user 会员等级服务单元测试
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

func setupMemberLevelServiceTestDB(t *testing.T) *gorm.DB {
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

	db.Create(&models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
		Benefits:  models.JSON{"free_shipping": true},
	})
	db.Create(&models.MemberLevel{ID: 2, Name: "黄金会员", Level: 2, MinPoints: 100, Discount: 0.9})

	return db
}

func newMemberLevelServiceForTest(db *gorm.DB) (*MemberLevelService, *repository.UserRepository, *repository.MemberLevelRepository) {
	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)
	return NewMemberLevelService(db, userRepo, levelRepo), userRepo, levelRepo
}

func createTestUserForMember(db *gorm.DB, points int, memberLevelID int64) *models.User {
	phone := fmt.Sprintf("139%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "会员用户",
		MemberLevelID: memberLevelID,
		Points:        points,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	return user
}

func TestMemberLevelService_GetAllLevels(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	levels, err := svc.GetAllLevels(ctx)
	require.NoError(t, err)
	require.Len(t, levels, 2)
	assert.Equal(t, "普通会员", levels[0].Name)
	assert.Equal(t, 1, levels[0].Level)
	assert.Equal(t, "黄金会员", levels[1].Name)
	assert.Equal(t, 2, levels[1].Level)
}

func TestMemberLevelService_GetUserMemberInfo_Progress(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForMember(db, 50, 1)

	info, err := svc.GetUserMemberInfo(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, info.CurrentLevel)
	assert.Equal(t, "普通会员", info.CurrentLevel.Name)
	require.NotNil(t, info.NextLevel)
	assert.Equal(t, "黄金会员", info.NextLevel.Name)
	require.NotNil(t, info.PointsToNextLevel)
	assert.Equal(t, 50, *info.PointsToNextLevel)
	require.NotNil(t, info.ProgressPercent)
	assert.InDelta(t, 50.0, *info.ProgressPercent, 0.001)
}

func TestMemberLevelService_CheckAndUpgradeLevel(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForMember(db, 120, 1)

	upgraded, newLevel, err := svc.CheckAndUpgradeLevel(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, upgraded)
	require.NotNil(t, newLevel)
	assert.Equal(t, int64(2), newLevel.ID)

	var refreshed models.User
	require.NoError(t, db.First(&refreshed, user.ID).Error)
	assert.Equal(t, int64(2), refreshed.MemberLevelID)
}

func TestMemberLevelService_GetDiscount_UserNotFound(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	discount, err := svc.GetDiscount(ctx, 999999)
	require.NoError(t, err)
	assert.Equal(t, 1.0, discount)
}

func TestMemberLevelService_GetLevelByID(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	t.Run("获取存在的会员等级", func(t *testing.T) {
		level, err := svc.GetLevelByID(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, level)
		assert.Equal(t, "普通会员", level.Name)
		assert.Equal(t, 1, level.Level)
	})

	t.Run("获取不存在的会员等级", func(t *testing.T) {
		level, err := svc.GetLevelByID(ctx, 999)
		assert.Error(t, err)
		assert.Nil(t, level)
	})
}

func TestMemberLevelService_CheckAndUpgradeLevel_NoUpgrade(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	// 积分不够升级
	user := createTestUserForMember(db, 50, 1)

	upgraded, newLevel, err := svc.CheckAndUpgradeLevel(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, upgraded)
	assert.Nil(t, newLevel)

	var refreshed models.User
	require.NoError(t, db.First(&refreshed, user.ID).Error)
	assert.Equal(t, int64(1), refreshed.MemberLevelID)
}

func TestMemberLevelService_CheckAndUpgradeLevel_AlreadyHighest(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	// 已经是最高等级
	user := createTestUserForMember(db, 1000, 2)

	upgraded, newLevel, err := svc.CheckAndUpgradeLevel(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, upgraded)
	assert.Nil(t, newLevel)
}

func TestMemberLevelService_CheckAndUpgradeLevel_UserNotFound(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	upgraded, newLevel, err := svc.CheckAndUpgradeLevel(ctx, 999999)
	assert.Error(t, err)
	assert.False(t, upgraded)
	assert.Nil(t, newLevel)
}

func TestMemberLevelService_CheckAndUpgradeLevelTx(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	t.Run("事务中升级成功", func(t *testing.T) {
		user := createTestUserForMember(db, 150, 1)

		err := db.Transaction(func(tx *gorm.DB) error {
			upgraded, err := svc.CheckAndUpgradeLevelTx(ctx, tx, user.ID)
			if err != nil {
				return err
			}
			assert.True(t, upgraded)
			return nil
		})
		require.NoError(t, err)

		var refreshed models.User
		require.NoError(t, db.First(&refreshed, user.ID).Error)
		assert.Equal(t, int64(2), refreshed.MemberLevelID)
	})

	t.Run("事务中不需要升级", func(t *testing.T) {
		user := createTestUserForMember(db, 50, 1)

		err := db.Transaction(func(tx *gorm.DB) error {
			upgraded, err := svc.CheckAndUpgradeLevelTx(ctx, tx, user.ID)
			if err != nil {
				return err
			}
			assert.False(t, upgraded)
			return nil
		})
		require.NoError(t, err)

		var refreshed models.User
		require.NoError(t, db.First(&refreshed, user.ID).Error)
		assert.Equal(t, int64(1), refreshed.MemberLevelID)
	})

	t.Run("事务中用户不存在", func(t *testing.T) {
		err := db.Transaction(func(tx *gorm.DB) error {
			_, err := svc.CheckAndUpgradeLevelTx(ctx, tx, 999999)
			return err
		})
		assert.Error(t, err)
	})
}

func TestMemberLevelService_GetDiscount_Success(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	// 创建黄金会员用户
	user := createTestUserForMember(db, 200, 2)

	discount, err := svc.GetDiscount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0.9, discount)
}

func TestMemberLevelService_GetUserMemberInfo_HighestLevel(t *testing.T) {
	db := setupMemberLevelServiceTestDB(t)
	svc, _, _ := newMemberLevelServiceForTest(db)
	ctx := context.Background()

	// 创建已是最高等级的用户
	user := createTestUserForMember(db, 500, 2)

	info, err := svc.GetUserMemberInfo(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, info.CurrentLevel)
	assert.Equal(t, "黄金会员", info.CurrentLevel.Name)
	// 已是最高等级，无下一级
	assert.Nil(t, info.NextLevel)
	assert.Nil(t, info.PointsToNextLevel)
	assert.Nil(t, info.ProgressPercent)
}

