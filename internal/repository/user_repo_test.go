// Package repository 用户仓储单元测试
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// setupUserTestDB 创建用户测试数据库
func setupUserTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Distributor{},
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

func TestUserRepository_Create(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotZero(t, user.ID)

	// 验证用户已创建
	var found models.User
	db.First(&found, user.ID)
	assert.Equal(t, phone, *found.Phone)
}

func TestUserRepository_GetByID(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138001"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	t.Run("获取存在的用户", func(t *testing.T) {
		found, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, phone, *found.Phone)
	})

	t.Run("获取不存在的用户", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestUserRepository_GetByPhone(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138002"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	t.Run("根据手机号获取用户", func(t *testing.T) {
		found, err := repo.GetByPhone(ctx, phone)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("获取不存在的手机号", func(t *testing.T) {
		_, err := repo.GetByPhone(ctx, "13999999999")
		assert.Error(t, err)
	})
}

func TestUserRepository_GetByOpenID(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	openID := "oXXXXXXXXX"
	user := &models.User{
		OpenID:        &openID,
		Nickname:      "微信用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	t.Run("根据 OpenID 获取用户", func(t *testing.T) {
		found, err := repo.GetByOpenID(ctx, openID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("获取不存在的 OpenID", func(t *testing.T) {
		_, err := repo.GetByOpenID(ctx, "invalid_openid")
		assert.Error(t, err)
	})
}

func TestUserRepository_GetByInviteCode(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138003"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "分销商",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	distributor := &models.Distributor{
		UserID:     user.ID,
		InviteCode: "ABC123",
		Level:      1,
		Status:     1,
	}
	db.Create(distributor)

	t.Run("根据邀请码获取用户", func(t *testing.T) {
		found, err := repo.GetByInviteCode(ctx, "ABC123")
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("获取不存在的邀请码", func(t *testing.T) {
		_, err := repo.GetByInviteCode(ctx, "INVALID")
		assert.Error(t, err)
	})
}

func TestUserRepository_Update(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138004"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "原昵称",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	user.Nickname = "新昵称"
	err := repo.Update(ctx, user)
	require.NoError(t, err)

	var found models.User
	db.First(&found, user.ID)
	assert.Equal(t, "新昵称", found.Nickname)
}

func TestUserRepository_UpdateFields(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138005"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Points:        100,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	err := repo.UpdateFields(ctx, user.ID, map[string]interface{}{
		"nickname": "更新昵称",
		"points":   200,
	})
	require.NoError(t, err)

	var found models.User
	db.First(&found, user.ID)
	assert.Equal(t, "更新昵称", found.Nickname)
	assert.Equal(t, 200, found.Points)
}

func TestUserRepository_UpdateStatus(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138006"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	err := repo.UpdateStatus(ctx, user.ID, models.UserStatusDisabled)
	require.NoError(t, err)

	var found models.User
	db.First(&found, user.ID)
	assert.Equal(t, int8(models.UserStatusDisabled), found.Status)
}

func TestUserRepository_List(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 创建多个用户
	for i := 0; i < 5; i++ {
		phone := "1380013800" + string(rune('0'+i))
		user := &models.User{
			Phone:         &phone,
			Nickname:      "用户" + string(rune('0'+i)),
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		db.Create(user)
	}

	t.Run("获取用户列表", func(t *testing.T) {
		users, total, err := repo.List(ctx, 0, 10, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, users, 5)
	})

	t.Run("分页获取", func(t *testing.T) {
		users, total, err := repo.List(ctx, 0, 2, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, users, 2)
	})

	t.Run("按状态筛选", func(t *testing.T) {
		filters := map[string]interface{}{
			"status": int8(models.UserStatusActive),
		}
		users, total, err := repo.List(ctx, 0, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, users, 5)
	})
}

func TestUserRepository_ExistsByPhone(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138007"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	t.Run("手机号存在", func(t *testing.T) {
		exists, err := repo.ExistsByPhone(ctx, phone)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("手机号不存在", func(t *testing.T) {
		exists, err := repo.ExistsByPhone(ctx, "13999999999")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestUserRepository_GetByIDWithWallet(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138008"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 100.0,
	}
	db.Create(wallet)

	found, err := repo.GetByIDWithWallet(ctx, user.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.Wallet)
	assert.Equal(t, float64(100), found.Wallet.Balance)
}

func TestUserRepository_AddPoints(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138009"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Points:        100,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	err := repo.AddPoints(ctx, user.ID, 50)
	require.NoError(t, err)

	var found models.User
	db.First(&found, user.ID)
	assert.Equal(t, 150, found.Points)
}

func TestUserRepository_DeductPoints(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138010"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Points:        100,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	t.Run("扣减积分成功", func(t *testing.T) {
		err := repo.DeductPoints(ctx, user.ID, 30)
		require.NoError(t, err)

		var found models.User
		db.First(&found, user.ID)
		assert.Equal(t, 70, found.Points)
	})

	t.Run("积分不足扣减失败", func(t *testing.T) {
		err := repo.DeductPoints(ctx, user.ID, 100) // 超过当前积分
		assert.Error(t, err)
	})
}

func TestUserRepository_GetReferrals(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 创建推荐人
	referrerPhone := "13800138011"
	referrer := &models.User{
		Phone:         &referrerPhone,
		Nickname:      "推荐人",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(referrer)

	// 创建被推荐用户
	for i := 0; i < 3; i++ {
		phone := "1380013801" + string(rune('2'+i))
		user := &models.User{
			Phone:         &phone,
			Nickname:      "被推荐用户" + string(rune('0'+i)),
			MemberLevelID: 1,
			ReferrerID:    &referrer.ID,
			Status:        models.UserStatusActive,
		}
		db.Create(user)
	}

	referrals, total, err := repo.GetReferrals(ctx, referrer.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, referrals, 3)
}
