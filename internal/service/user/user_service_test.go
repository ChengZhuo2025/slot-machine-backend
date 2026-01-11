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

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupUserServiceTestDB(t *testing.T) *gorm.DB {
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

	require.NoError(t, db.AutoMigrate(&models.User{}, &models.MemberLevel{}))

	require.NoError(t, db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0}).Error)
	require.NoError(t, db.Create(&models.MemberLevel{ID: 2, Name: "VIP", Level: 2, MinPoints: 100, Discount: 0.9}).Error)

	return db
}

func setupUserService(db *gorm.DB) *UserService {
	userRepo := repository.NewUserRepository(db)
	return NewUserService(db, userRepo)
}

func createUserServiceTestUser(t *testing.T, db *gorm.DB, opts ...func(*models.User)) *models.User {
	t.Helper()

	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	u := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	for _, opt := range opts {
		opt(u)
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

func TestUserService_GetProfile(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := setupUserService(db)
	ctx := context.Background()

	t.Run("用户不存在", func(t *testing.T) {
		_, err := svc.GetProfile(ctx, 99999)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrUserNotFound.Code, appErr.Code)
	})

	t.Run("包含会员等级信息", func(t *testing.T) {
		u := createUserServiceTestUser(t, db, func(u *models.User) {
			u.MemberLevelID = 2
			u.Points = 123
			u.IsVerified = true
			u.Gender = models.GenderMale
		})

		profile, err := svc.GetProfile(ctx, u.ID)
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, u.ID, profile.ID)
		assert.Equal(t, u.Nickname, profile.Nickname)
		assert.Equal(t, 123, profile.Points)
		assert.True(t, profile.IsVerified)
		require.NotNil(t, profile.MemberLevel)
		assert.Equal(t, int64(2), profile.MemberLevel.ID)
		assert.Equal(t, "VIP", profile.MemberLevel.Name)
		assert.Equal(t, 2, profile.MemberLevel.Level)
		assert.Equal(t, 0.9, profile.MemberLevel.Discount)
	})

	t.Run("会员等级不存在时 MemberLevel 为空", func(t *testing.T) {
		u := createUserServiceTestUser(t, db, func(u *models.User) { u.MemberLevelID = 999 })
		profile, err := svc.GetProfile(ctx, u.ID)
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Nil(t, profile.MemberLevel)
		assert.Equal(t, int64(999), profile.MemberLevelID)
	})
}

func TestUserService_UpdateProfile(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := setupUserService(db)
	ctx := context.Background()

	u := createUserServiceTestUser(t, db, func(u *models.User) { u.Nickname = "old" })

	t.Run("无更新字段直接返回", func(t *testing.T) {
		require.NoError(t, svc.UpdateProfile(ctx, u.ID, &UpdateProfileRequest{}))

		var got models.User
		require.NoError(t, db.First(&got, u.ID).Error)
		assert.Equal(t, "old", got.Nickname)
	})

	t.Run("更新部分字段", func(t *testing.T) {
		nickname := "new"
		avatar := "https://img.example/avatar.png"
		gender := int8(models.GenderFemale)
		birthday := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)

		req := &UpdateProfileRequest{
			Nickname: &nickname,
			Avatar:   &avatar,
			Gender:   &gender,
			Birthday: &birthday,
		}
		require.NoError(t, svc.UpdateProfile(ctx, u.ID, req))

		var got models.User
		require.NoError(t, db.First(&got, u.ID).Error)
		assert.Equal(t, "new", got.Nickname)
		require.NotNil(t, got.Avatar)
		assert.Equal(t, avatar, *got.Avatar)
		assert.Equal(t, int8(models.GenderFemale), got.Gender)
		require.NotNil(t, got.Birthday)
		assert.Equal(t, birthday, *got.Birthday)
	})
}

func TestUserService_GetMemberLevels(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := setupUserService(db)
	ctx := context.Background()

	levels, err := svc.GetMemberLevels(ctx)
	require.NoError(t, err)
	require.Len(t, levels, 2)
	assert.Equal(t, 1, levels[0].Level)
	assert.Equal(t, 2, levels[1].Level)
}

func TestUserService_RealNameVerify(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := setupUserService(db)
	ctx := context.Background()

	t.Run("用户不存在", func(t *testing.T) {
		err := svc.RealNameVerify(ctx, 99999, &RealNameVerifyRequest{RealName: "张三", IDCard: "123"})
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrUserNotFound.Code, appErr.Code)
	})

	t.Run("已认证不允许重复认证", func(t *testing.T) {
		u := createUserServiceTestUser(t, db, func(u *models.User) { u.IsVerified = true })
		err := svc.RealNameVerify(ctx, u.ID, &RealNameVerifyRequest{RealName: "张三", IDCard: "123"})
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrRealNameVerified.Code, appErr.Code)
	})

	t.Run("认证成功更新实名信息", func(t *testing.T) {
		u := createUserServiceTestUser(t, db)
		err := svc.RealNameVerify(ctx, u.ID, &RealNameVerifyRequest{RealName: "张三", IDCard: "123"})
		require.NoError(t, err)

		var got models.User
		require.NoError(t, db.First(&got, u.ID).Error)
		assert.True(t, got.IsVerified)
		require.NotNil(t, got.RealNameEncrypted)
		require.NotNil(t, got.IDCardEncrypted)
		assert.Equal(t, "张三", *got.RealNameEncrypted)
		assert.Equal(t, "123", *got.IDCardEncrypted)
	})
}

func TestUserService_GetPoints(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := setupUserService(db)
	ctx := context.Background()

	t.Run("用户不存在", func(t *testing.T) {
		_, err := svc.GetPoints(ctx, 99999)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrUserNotFound.Code, appErr.Code)
	})

	t.Run("获取积分成功", func(t *testing.T) {
		u := createUserServiceTestUser(t, db, func(u *models.User) { u.Points = 88 })
		points, err := svc.GetPoints(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, 88, points)
	})
}
