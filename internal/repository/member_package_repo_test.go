// Package repository 会员套餐仓储单元测试
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

func setupMemberPackageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.MemberPackage{}, &models.MemberLevel{})
	require.NoError(t, err)

	return db
}

const (
	MemberPackageStatusActive   = 1
	MemberPackageStatusInactive = 0
)

func TestMemberPackageRepository_GetByID(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	pkg := &models.MemberPackage{
		Name:          "月度会员",
		MemberLevelID: level.ID,
		Duration:      1,
		DurationUnit:  "month",
		Price:         99.0,
		GiftPoints:    100,
		Benefits:      models.JSON{"benefits": []string{}},
	}
	db.Create(pkg)

	found, err := repo.GetByID(ctx, pkg.ID)
	require.NoError(t, err)
	assert.Equal(t, pkg.ID, found.ID)
	assert.NotNil(t, found.MemberLevel)
	assert.Equal(t, level.ID, found.MemberLevel.ID)
}

func TestMemberPackageRepository_GetAll(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	db.Create(&models.MemberPackage{
		Name: "月度会员", MemberLevelID: level.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, GiftPoints: 100, Sort: 1, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberPackage{
		Name: "季度会员", MemberLevelID: level.ID, Duration: 3, DurationUnit: "month",
		Price: 279.0, GiftPoints: 300, Sort: 2, Benefits: models.JSON{"benefits": []string{}},
	})

	packages, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(packages))
	// 应该按sort DESC排序
	assert.Equal(t, 2, packages[0].Sort)
	assert.Equal(t, 1, packages[1].Sort)
}

func TestMemberPackageRepository_GetActive(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	db.Create(&models.MemberPackage{
		Name: "月度会员", MemberLevelID: level.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, Status: MemberPackageStatusActive, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Model(&models.MemberPackage{}).Create(map[string]interface{}{
		"name": "禁用套餐", "member_level_id": level.ID, "duration": 1, "duration_unit": "month",
		"price": 99.0, "status": MemberPackageStatusInactive, "benefits": models.JSON{"benefits": []string{}},
	})

	packages, err := repo.GetActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(packages))
	assert.Equal(t, int8(MemberPackageStatusActive), packages[0].Status)
}

func TestMemberPackageRepository_GetByLevelID(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level1 := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level1)

	level2 := &models.MemberLevel{
		Name: "金牌会员", Level: 3, MinPoints: 500, Discount: 0.85, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level2)

	db.Create(&models.MemberPackage{
		Name: "银牌月度", MemberLevelID: level1.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, Status: MemberPackageStatusActive, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberPackage{
		Name: "金牌月度", MemberLevelID: level2.ID, Duration: 1, DurationUnit: "month",
		Price: 199.0, Status: MemberPackageStatusActive, Benefits: models.JSON{"benefits": []string{}},
	})

	packages, err := repo.GetByLevelID(ctx, level1.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(packages))
	assert.Equal(t, level1.ID, packages[0].MemberLevelID)
}

func TestMemberPackageRepository_Create(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	pkg := &models.MemberPackage{
		Name:          "年度会员",
		MemberLevelID: level.ID,
		Duration:      12,
		DurationUnit:  "month",
		Price:         999.0,
		GiftPoints:    1200,
		Benefits:      models.JSON{"benefits": []string{"VIP"}},
	}

	err := repo.Create(ctx, pkg)
	require.NoError(t, err)
	assert.NotZero(t, pkg.ID)
}

func TestMemberPackageRepository_Update(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	pkg := &models.MemberPackage{
		Name: "月度会员", MemberLevelID: level.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(pkg)

	pkg.Price = 89.0
	err := repo.Update(ctx, pkg)
	require.NoError(t, err)

	var found models.MemberPackage
	db.First(&found, pkg.ID)
	assert.Equal(t, 89.0, found.Price)
}

func TestMemberPackageRepository_UpdateStatus(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	pkg := &models.MemberPackage{
		Name: "月度会员", MemberLevelID: level.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, Status: MemberPackageStatusActive, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(pkg)

	err := repo.UpdateStatus(ctx, pkg.ID, MemberPackageStatusInactive)
	require.NoError(t, err)

	var found models.MemberPackage
	db.First(&found, pkg.ID)
	assert.Equal(t, int8(MemberPackageStatusInactive), found.Status)
}

func TestMemberPackageRepository_Delete(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	pkg := &models.MemberPackage{
		Name: "待删除套餐", MemberLevelID: level.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(pkg)

	err := repo.Delete(ctx, pkg.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.MemberPackage{}).Where("id = ?", pkg.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestMemberPackageRepository_List(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	db.Create(&models.MemberPackage{
		Name: "月度会员", MemberLevelID: level.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, Status: MemberPackageStatusActive, IsRecommend: true, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberPackage{
		Name: "季度会员", MemberLevelID: level.ID, Duration: 3, DurationUnit: "month",
		Price: 279.0, Status: MemberPackageStatusActive, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Model(&models.MemberPackage{}).Create(map[string]interface{}{
		"name": "禁用套餐", "member_level_id": level.ID, "duration": 1, "duration_unit": "month",
		"price": 99.0, "status": MemberPackageStatusInactive, "is_recommend": false,
		"benefits": models.JSON{"benefits": []string{}},
	})

	// 获取所有套餐
	_, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按状态过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": int8(MemberPackageStatusActive),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按等级过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"member_level_id": level.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按推荐过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"is_recommend": true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestMemberPackageRepository_GetRecommended(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	db.Create(&models.MemberPackage{
		Name: "推荐套餐", MemberLevelID: level.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, Status: MemberPackageStatusActive, IsRecommend: true, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Model(&models.MemberPackage{}).Create(map[string]interface{}{
		"name": "普通套餐", "member_level_id": level.ID, "duration": 1, "duration_unit": "month",
		"price": 99.0, "status": MemberPackageStatusActive, "is_recommend": false,
		"benefits": models.JSON{"benefits": []string{}},
	})

	packages, err := repo.GetRecommended(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(packages))
	assert.True(t, packages[0].IsRecommend)
}

func TestMemberPackageRepository_Count(t *testing.T) {
	db := setupMemberPackageTestDB(t)
	repo := NewMemberPackageRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	db.Create(&models.MemberPackage{
		Name: "月度会员", MemberLevelID: level.ID, Duration: 1, DurationUnit: "month",
		Price: 99.0, Status: MemberPackageStatusActive, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Model(&models.MemberPackage{}).Create(map[string]interface{}{
		"name": "禁用套餐", "member_level_id": level.ID, "duration": 1, "duration_unit": "month",
		"price": 99.0, "status": MemberPackageStatusInactive, "benefits": models.JSON{"benefits": []string{}},
	})

	// 统计所有套餐
	count, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 统计活跃套餐
	activeStatus := int8(MemberPackageStatusActive)
	count, err = repo.Count(ctx, &activeStatus)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
