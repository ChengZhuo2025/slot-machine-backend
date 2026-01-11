// Package repository 会员等级仓储单元测试
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

func setupMemberLevelTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.MemberLevel{})
	require.NoError(t, err)

	return db
}

func TestMemberLevelRepository_GetByID(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
		Benefits:  models.JSON{"benefits": []string{"基础服务"}},
	}
	db.Create(level)

	found, err := repo.GetByID(ctx, level.ID)
	require.NoError(t, err)
	assert.Equal(t, level.ID, found.ID)
	assert.Equal(t, "普通会员", found.Name)
}

func TestMemberLevelRepository_GetByLevel(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	db.Create(&models.MemberLevel{
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
		Benefits:  models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name:      "银牌会员",
		Level:     2,
		MinPoints: 100,
		Discount:  0.95,
		Benefits:  models.JSON{"benefits": []string{}},
	})

	found, err := repo.GetByLevel(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, found.Level)
	assert.Equal(t, "银牌会员", found.Name)
}

func TestMemberLevelRepository_GetAll(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	db.Create(&models.MemberLevel{
		Name: "金牌会员", Level: 3, MinPoints: 500, Discount: 0.85, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	})

	levels, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, len(levels))
	// 应该按等级升序排列
	assert.Equal(t, 1, levels[0].Level)
	assert.Equal(t, 2, levels[1].Level)
	assert.Equal(t, 3, levels[2].Level)
}

func TestMemberLevelRepository_GetByMinPoints(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	db.Create(&models.MemberLevel{
		Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "金牌会员", Level: 3, MinPoints: 500, Discount: 0.85, Benefits: models.JSON{"benefits": []string{}},
	})

	// 积分150应匹配银牌会员(min_points=100)
	level, err := repo.GetByMinPoints(ctx, 150)
	require.NoError(t, err)
	assert.Equal(t, 2, level.Level)
	assert.Equal(t, "银牌会员", level.Name)

	// 积分600应匹配金牌会员(min_points=500)
	level, err = repo.GetByMinPoints(ctx, 600)
	require.NoError(t, err)
	assert.Equal(t, 3, level.Level)
	assert.Equal(t, "金牌会员", level.Name)
}

func TestMemberLevelRepository_Create(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name:      "钻石会员",
		Level:     4,
		MinPoints: 1000,
		Discount:  0.75,
		Benefits:  models.JSON{"benefits": []string{"VIP服务"}},
	}

	err := repo.Create(ctx, level)
	require.NoError(t, err)
	assert.NotZero(t, level.ID)
}

func TestMemberLevelRepository_Update(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
		Benefits:  models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	level.Discount = 0.98
	err := repo.Update(ctx, level)
	require.NoError(t, err)

	var found models.MemberLevel
	db.First(&found, level.ID)
	assert.Equal(t, 0.98, found.Discount)
}

func TestMemberLevelRepository_Delete(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	level := &models.MemberLevel{
		Name:      "测试等级",
		Level:     99,
		MinPoints: 9999,
		Discount:  0.5,
		Benefits:  models.JSON{"benefits": []string{}},
	}
	db.Create(level)

	err := repo.Delete(ctx, level.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.MemberLevel{}).Where("id = ?", level.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestMemberLevelRepository_GetNextLevel(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	db.Create(&models.MemberLevel{
		Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "金牌会员", Level: 3, MinPoints: 500, Discount: 0.85, Benefits: models.JSON{"benefits": []string{}},
	})

	// 普通会员的下一级应该是银牌会员
	nextLevel, err := repo.GetNextLevel(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, nextLevel.Level)
	assert.Equal(t, "银牌会员", nextLevel.Name)
}

func TestMemberLevelRepository_GetDefaultLevel(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	db.Create(&models.MemberLevel{
		Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	})

	defaultLevel, err := repo.GetDefaultLevel(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, defaultLevel.Level)
	assert.Equal(t, "普通会员", defaultLevel.Name)
}

func TestMemberLevelRepository_Count(t *testing.T) {
	db := setupMemberLevelTestDB(t)
	repo := NewMemberLevelRepository(db)
	ctx := context.Background()

	db.Create(&models.MemberLevel{
		Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "银牌会员", Level: 2, MinPoints: 100, Discount: 0.95, Benefits: models.JSON{"benefits": []string{}},
	})

	db.Create(&models.MemberLevel{
		Name: "金牌会员", Level: 3, MinPoints: 500, Discount: 0.85, Benefits: models.JSON{"benefits": []string{}},
	})

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}
