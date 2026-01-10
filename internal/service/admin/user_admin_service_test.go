// Package admin 用户管理服务单元测试
package admin

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
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupUserAdminTestDB 创建用户管理测试数据库
func setupUserAdminTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.UserWallet{}, &models.MemberLevel{})
	require.NoError(t, err)

	return db
}

// setupUserAdminService 创建测试用的 UserAdminService
func setupUserAdminService(t *testing.T) (*UserAdminService, *gorm.DB) {
	db := setupUserAdminTestDB(t)
	userRepo := repository.NewUserRepository(db)
	service := NewUserAdminService(db, userRepo)
	return service, db
}

// createTestUserForAdmin 创建测试用户
func createTestUserForAdmin(t *testing.T, db *gorm.DB, phone string) *models.User {
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		Status:        models.UserStatusActive,
		MemberLevelID: 1,
		Points:        100,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID:        user.ID,
		Balance:       100.00,
		FrozenBalance: 10.00,
	}
	db.Create(wallet)

	return user
}

func TestUserAdminService_List(t *testing.T) {
	service, db := setupUserAdminService(t)
	ctx := context.Background()

	// 创建会员等级
	level := &models.MemberLevel{ID: 1, Name: "普通会员", Level: 1}
	db.Create(level)

	// 创建多个用户
	phones := []string{"13800138001", "13800138002", "13800138003", "13900139001"}
	for i, phone := range phones {
		user := &models.User{
			Phone:         &phone,
			Nickname:      "用户" + string(rune('A'+i)),
			Status:        models.UserStatusActive,
			MemberLevelID: 1,
			Points:        100 * (i + 1),
		}
		db.Create(user)
	}

	t.Run("获取全部用户列表", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 4)
	})

	t.Run("按手机号筛选", func(t *testing.T) {
		filters := &UserListFilters{Phone: "138"}
		results, total, err := service.List(ctx, 1, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
	})

	t.Run("按状态筛选", func(t *testing.T) {
		status := int8(models.UserStatusActive)
		filters := &UserListFilters{Status: &status}
		results, total, err := service.List(ctx, 1, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
	})

	t.Run("分页", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 2, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 2)

		results2, _, err := service.List(ctx, 2, 2, nil)
		require.NoError(t, err)
		assert.Len(t, results2, 2)
	})
}

func TestUserAdminService_GetByID(t *testing.T) {
	service, db := setupUserAdminService(t)
	ctx := context.Background()

	user := createTestUserForAdmin(t, db, "13800138000")

	t.Run("获取存在的用户", func(t *testing.T) {
		result, err := service.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, "测试用户", result.Nickname)
	})

	t.Run("获取不存在的用户", func(t *testing.T) {
		_, err := service.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestUserAdminService_Update(t *testing.T) {
	service, db := setupUserAdminService(t)
	ctx := context.Background()

	user := createTestUserForAdmin(t, db, "13800138000")

	t.Run("更新昵称", func(t *testing.T) {
		nickname := "新昵称"
		req := &UpdateUserRequest{Nickname: &nickname}

		updated, err := service.Update(ctx, user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "新昵称", updated.Nickname)
	})

	t.Run("更新积分", func(t *testing.T) {
		points := 500
		req := &UpdateUserRequest{Points: &points}

		updated, err := service.Update(ctx, user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, 500, updated.Points)
	})

	t.Run("更新状态", func(t *testing.T) {
		status := int8(models.UserStatusDisabled)
		req := &UpdateUserRequest{Status: &status}

		updated, err := service.Update(ctx, user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, int8(models.UserStatusDisabled), updated.Status)
	})

	t.Run("更新不存在的用户", func(t *testing.T) {
		nickname := "测试"
		req := &UpdateUserRequest{Nickname: &nickname}
		_, err := service.Update(ctx, 99999, req)
		assert.Error(t, err)
	})
}

func TestUserAdminService_UpdateStatus(t *testing.T) {
	service, db := setupUserAdminService(t)
	ctx := context.Background()

	user := createTestUserForAdmin(t, db, "13800138000")

	t.Run("更新状态", func(t *testing.T) {
		err := service.UpdateStatus(ctx, user.ID, models.UserStatusDisabled)
		require.NoError(t, err)

		// 验证状态已更新
		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, int8(models.UserStatusDisabled), updated.Status)
	})
}

func TestUserAdminService_EnableDisable(t *testing.T) {
	service, db := setupUserAdminService(t)
	ctx := context.Background()

	user := createTestUserForAdmin(t, db, "13800138000")

	t.Run("禁用用户", func(t *testing.T) {
		err := service.Disable(ctx, user.ID)
		require.NoError(t, err)

		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, int8(models.UserStatusDisabled), updated.Status)
	})

	t.Run("启用用户", func(t *testing.T) {
		err := service.Enable(ctx, user.ID)
		require.NoError(t, err)

		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, int8(models.UserStatusActive), updated.Status)
	})
}

func TestUserAdminService_AdjustPoints(t *testing.T) {
	service, db := setupUserAdminService(t)
	ctx := context.Background()

	user := createTestUserForAdmin(t, db, "13800138000")
	user.Points = 100
	db.Save(user)

	t.Run("增加积分", func(t *testing.T) {
		err := service.AdjustPoints(ctx, user.ID, 50, "管理员增加")
		require.NoError(t, err)

		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, 150, updated.Points)
	})

	t.Run("减少积分", func(t *testing.T) {
		err := service.AdjustPoints(ctx, user.ID, -30, "管理员扣除")
		require.NoError(t, err)

		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, 120, updated.Points)
	})

	t.Run("积分不能为负", func(t *testing.T) {
		err := service.AdjustPoints(ctx, user.ID, -1000, "大额扣除")
		require.NoError(t, err)

		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, 0, updated.Points) // 应该被限制为0
	})
}

func TestUserAdminService_GetStatistics(t *testing.T) {
	service, db := setupUserAdminService(t)
	ctx := context.Background()

	// 创建会员等级
	level1 := &models.MemberLevel{ID: 1, Name: "普通会员", Level: 1}
	level2 := &models.MemberLevel{ID: 2, Name: "VIP会员", Level: 2}
	db.Create(level1)
	db.Create(level2)

	// 创建多个用户
	phones := []string{"13800138001", "13800138002", "13800138003"}
	for i, phone := range phones {
		user := &models.User{
			Phone:         &phone,
			Nickname:      "用户" + string(rune('A'+i)),
			Status:        models.UserStatusActive,
			MemberLevelID: int64((i % 2) + 1),
			IsVerified:    i < 2,
		}
		db.Create(user)
	}

	// 创建一个禁用用户
	disabledPhone := "13900139000"
	disabledUser := &models.User{
		Phone:         &disabledPhone,
		Nickname:      "禁用用户",
		Status:        models.UserStatusDisabled,
		MemberLevelID: 1,
	}
	db.Create(disabledUser)

	stats, err := service.GetStatistics(ctx)
	require.NoError(t, err)

	t.Run("总用户数", func(t *testing.T) {
		assert.Equal(t, int64(4), stats.TotalUsers)
	})

	t.Run("实名用户数", func(t *testing.T) {
		assert.Equal(t, int64(2), stats.VerifiedUsers)
	})

	t.Run("禁用用户数", func(t *testing.T) {
		assert.Equal(t, int64(1), stats.DisabledUsers)
	})
}

func TestToUserListResponse(t *testing.T) {
	service, _ := setupUserAdminService(t)

	phone := "13800138000"
	avatar := "https://example.com/avatar.png"
	user := &models.User{
		ID:            1,
		Phone:         &phone,
		Nickname:      "测试用户",
		Avatar:        &avatar,
		Gender:        1,
		MemberLevelID: 1,
		MemberLevel:   &models.MemberLevel{ID: 1, Name: "普通会员"},
		Points:        100,
		IsVerified:    true,
		Status:        models.UserStatusActive,
		CreatedAt:     time.Now(),
		Wallet: &models.UserWallet{
			Balance:       100.00,
			FrozenBalance: 10.00,
		},
	}

	resp := service.toUserListResponse(user)

	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "13800138000", resp.Phone)
	assert.Equal(t, "测试用户", resp.Nickname)
	assert.Equal(t, "https://example.com/avatar.png", resp.Avatar)
	assert.Equal(t, int8(1), resp.Gender)
	assert.Equal(t, 100, resp.Points)
	assert.True(t, resp.IsVerified)
	assert.NotNil(t, resp.MemberLevel)
	assert.NotNil(t, resp.Wallet)
	assert.Equal(t, 100.00, resp.Wallet.Balance)
	assert.Equal(t, 10.00, resp.Wallet.FrozenBalance)
}
