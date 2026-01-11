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

func TestMemberAdminService_GetMemberLevelDetail(t *testing.T) {
	db := setupMemberAdminTestDB(t)
	svc := NewMemberAdminService(
		db,
		repository.NewMemberLevelRepository(db),
		repository.NewMemberPackageRepository(db),
		repository.NewUserRepository(db),
	)
	ctx := context.Background()

	level, _ := svc.CreateMemberLevel(ctx, &CreateMemberLevelRequest{
		Name:     "VIP会员",
		Level:    2,
		Discount: 0.9,
	})

	detail, err := svc.GetMemberLevelDetail(ctx, level.ID)
	require.NoError(t, err)
	assert.Equal(t, level.ID, detail.ID)
	assert.Equal(t, "VIP会员", detail.Name)

	t.Run("会员等级不存在", func(t *testing.T) {
		_, err := svc.GetMemberLevelDetail(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestMemberAdminService_UpdateMemberLevel(t *testing.T) {
	db := setupMemberAdminTestDB(t)
	svc := NewMemberAdminService(
		db,
		repository.NewMemberLevelRepository(db),
		repository.NewMemberPackageRepository(db),
		repository.NewUserRepository(db),
	)
	ctx := context.Background()

	level, _ := svc.CreateMemberLevel(ctx, &CreateMemberLevelRequest{
		Name:     "原会员等级",
		Level:    3,
		Discount: 0.85,
	})

	newName := "更新后会员"
	newDiscount := 0.8
	err := svc.UpdateMemberLevel(ctx, level.ID, &UpdateMemberLevelRequest{
		Name:     &newName,
		Discount: &newDiscount,
	})
	require.NoError(t, err)

	var updated models.MemberLevel
	db.First(&updated, level.ID)
	assert.Equal(t, "更新后会员", updated.Name)
	assert.Equal(t, 0.8, updated.Discount)
}

func TestMemberAdminService_MemberPackageCRUD(t *testing.T) {
	db := setupMemberAdminTestDB(t)
	svc := NewMemberAdminService(
		db,
		repository.NewMemberLevelRepository(db),
		repository.NewMemberPackageRepository(db),
		repository.NewUserRepository(db),
	)
	ctx := context.Background()

	level, _ := svc.CreateMemberLevel(ctx, &CreateMemberLevelRequest{
		Name:     "套餐测试会员",
		Level:    4,
		Discount: 0.75,
	})

	originalPrice := 129.0

	t.Run("创建会员套餐", func(t *testing.T) {
		pkg, err := svc.CreateMemberPackage(ctx, &CreateMemberPackageRequest{
			Name:          "月度套餐",
			MemberLevelID: level.ID,
			Duration:      30,
			DurationUnit:  "day",
			Price:         99.0,
			OriginalPrice: &originalPrice,
		})
		require.NoError(t, err)
		assert.NotNil(t, pkg)
		assert.Equal(t, "月度套餐", pkg.Name)
	})

	t.Run("获取会员套餐列表", func(t *testing.T) {
		resp, err := svc.GetMemberPackageList(ctx, &AdminPackageListRequest{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.True(t, resp.Total >= 1)
		assert.NotEmpty(t, resp.List)
	})

	t.Run("获取会员套餐详情", func(t *testing.T) {
		pkg := &models.MemberPackage{
			Name:          "详情测试套餐",
			MemberLevelID: level.ID,
			Duration:      30,
			DurationUnit:  "day",
			Price:         99.0,
			OriginalPrice: &originalPrice,
			Status:        models.MemberPackageStatusActive,
		}
		require.NoError(t, db.Create(pkg).Error)

		detail, err := svc.GetMemberPackageDetail(ctx, pkg.ID)
		require.NoError(t, err)
		assert.Equal(t, pkg.ID, detail.ID)
		assert.Equal(t, "详情测试套餐", detail.Name)
	})

	t.Run("更新会员套餐", func(t *testing.T) {
		pkg := &models.MemberPackage{
			Name:          "更新测试套餐",
			MemberLevelID: level.ID,
			Duration:      30,
			DurationUnit:  "day",
			Price:         99.0,
			OriginalPrice: &originalPrice,
			Status:        models.MemberPackageStatusActive,
		}
		require.NoError(t, db.Create(pkg).Error)

		newName := "已更新套餐"
		newPrice := 79.0
		err := svc.UpdateMemberPackage(ctx, pkg.ID, &UpdateMemberPackageRequest{
			Name:  &newName,
			Price: &newPrice,
		})
		require.NoError(t, err)

		var updated models.MemberPackage
		db.First(&updated, pkg.ID)
		assert.Equal(t, "已更新套餐", updated.Name)
		assert.Equal(t, 79.0, updated.Price)
	})

	t.Run("更新会员套餐状态", func(t *testing.T) {
		pkg := &models.MemberPackage{
			Name:          "状态测试套餐",
			MemberLevelID: level.ID,
			Duration:      30,
			DurationUnit:  "day",
			Price:         99.0,
			OriginalPrice: &originalPrice,
			Status:        models.MemberPackageStatusActive,
		}
		require.NoError(t, db.Create(pkg).Error)

		err := svc.UpdateMemberPackageStatus(ctx, pkg.ID, models.MemberPackageStatusDisabled)
		require.NoError(t, err)

		var updated models.MemberPackage
		db.First(&updated, pkg.ID)
		assert.Equal(t, int8(models.MemberPackageStatusDisabled), updated.Status)
	})

	t.Run("删除会员套餐", func(t *testing.T) {
		pkg := &models.MemberPackage{
			Name:          "删除测试套餐",
			MemberLevelID: level.ID,
			Duration:      30,
			DurationUnit:  "day",
			Price:         99.0,
			OriginalPrice: &originalPrice,
			Status:        models.MemberPackageStatusActive,
		}
		require.NoError(t, db.Create(pkg).Error)

		err := svc.DeleteMemberPackage(ctx, pkg.ID)
		require.NoError(t, err)

		var count int64
		db.Model(&models.MemberPackage{}).Where("id = ?", pkg.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

func TestMemberAdminService_GetMemberStats(t *testing.T) {
	db := setupMemberAdminTestDB(t)
	svc := NewMemberAdminService(
		db,
		repository.NewMemberLevelRepository(db),
		repository.NewMemberPackageRepository(db),
		repository.NewUserRepository(db),
	)
	ctx := context.Background()

	level, _ := svc.CreateMemberLevel(ctx, &CreateMemberLevelRequest{
		Name:     "统计测试会员",
		Level:    5,
		Discount: 0.7,
	})

	// 创建一些用户
	for i := 0; i < 3; i++ {
		u := &models.User{
			Nickname:      fmt.Sprintf("User%d", i),
			MemberLevelID: level.ID,
			Status:        models.UserStatusActive,
		}
		db.Create(u)
	}

	stats, err := svc.GetMemberStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

