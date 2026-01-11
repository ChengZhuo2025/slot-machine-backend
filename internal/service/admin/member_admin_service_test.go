package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupMemberAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(&models.User{}, &models.MemberLevel{}, &models.MemberPackage{}))
	return db
}

func TestMemberAdminService_MemberLevelBranches(t *testing.T) {
	db := setupMemberAdminTestDB(t)
	svc := NewMemberAdminService(
		db,
		repository.NewMemberLevelRepository(db),
		repository.NewMemberPackageRepository(db),
		repository.NewUserRepository(db),
	)
	ctx := context.Background()

	level1, err := svc.CreateMemberLevel(ctx, &CreateMemberLevelRequest{
		Name:     "普通会员",
		Level:    1,
		Discount: 1.0,
	})
	require.NoError(t, err)
	require.NotNil(t, level1)

	t.Run("CreateMemberLevel 等级序号冲突", func(t *testing.T) {
		_, err := svc.CreateMemberLevel(ctx, &CreateMemberLevelRequest{
			Name:     "重复",
			Level:    1,
			Discount: 1.0,
		})
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrOperationFailed.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "等级序号已存在")
	})

	u := &models.User{Nickname: "U1", MemberLevelID: level1.ID, Status: models.UserStatusActive}
	require.NoError(t, db.Create(u).Error)

	t.Run("GetMemberLevelList 返回用户数量", func(t *testing.T) {
		list, err := svc.GetMemberLevelList(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, int64(1), list[0].UserCount)
	})

	t.Run("DeleteMemberLevel 有用户不允许删除", func(t *testing.T) {
		err := svc.DeleteMemberLevel(ctx, level1.ID)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrOperationFailed.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "有用户")
	})
}

