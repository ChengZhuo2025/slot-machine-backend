// Package repository 分销商仓储单元测试
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

func setupDistributorTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Distributor{})
	require.NoError(t, err)

	return db
}

func TestDistributorRepository_Create(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}

	err := repo.Create(ctx, distributor)
	require.NoError(t, err)
	assert.NotZero(t, distributor.ID)
}

func TestDistributorRepository_GetByID(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(distributor)

	found, err := repo.GetByID(ctx, distributor.ID)
	require.NoError(t, err)
	assert.Equal(t, distributor.ID, found.ID)
	assert.Equal(t, "INV001", found.InviteCode)
}

func TestDistributorRepository_GetByIDWithUser(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	phone := "13800138000"
	user := &models.User{
		Phone: &phone,
	}
	db.Create(user)

	distributor := &models.Distributor{
		UserID:     user.ID,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(distributor)

	found, err := repo.GetByIDWithUser(ctx, distributor.ID)
	require.NoError(t, err)
	assert.Equal(t, distributor.ID, found.ID)
	assert.NotNil(t, found.User)
	assert.Equal(t, user.ID, found.User.ID)
}

func TestDistributorRepository_GetByUserID(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(distributor)

	found, err := repo.GetByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, distributor.ID, found.ID)
}

func TestDistributorRepository_GetByInviteCode(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(distributor)

	found, err := repo.GetByInviteCode(ctx, "INV001")
	require.NoError(t, err)
	assert.Equal(t, distributor.ID, found.ID)
}

func TestDistributorRepository_GetByInviteCodeWithUser(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	phone := "13800138000"
	user := &models.User{
		Phone: &phone,
	}
	db.Create(user)

	distributor := &models.Distributor{
		UserID:     user.ID,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(distributor)

	found, err := repo.GetByInviteCodeWithUser(ctx, "INV001")
	require.NoError(t, err)
	assert.Equal(t, distributor.ID, found.ID)
	assert.NotNil(t, found.User)
	assert.Equal(t, user.ID, found.User.ID)
}

func TestDistributorRepository_Update(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(distributor)

	distributor.InviteCode = "INV999"
	err := repo.Update(ctx, distributor)
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, "INV999", found.InviteCode)
}

func TestDistributorRepository_UpdateFields(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(distributor)

	err := repo.UpdateFields(ctx, distributor.ID, map[string]interface{}{
		"team_count": 10,
	})
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, 10, found.TeamCount)
}

func TestDistributorRepository_UpdateStatus(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
		Status:     models.DistributorStatusPending,
	}
	db.Create(distributor)

	err := repo.UpdateStatus(ctx, distributor.ID, models.DistributorStatusApproved)
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, models.DistributorStatusApproved, found.Status)
}

func TestDistributorRepository_ExistsByUserID(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	db.Create(&models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	})

	exists, err := repo.ExistsByUserID(ctx, 1)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByUserID(ctx, 999)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDistributorRepository_ExistsByInviteCode(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	db.Create(&models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	})

	exists, err := repo.ExistsByInviteCode(ctx, "INV001")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByInviteCode(ctx, "INV999")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDistributorRepository_List(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	parent := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
		Status:     models.DistributorStatusApproved,
	}
	db.Create(parent)

	db.Create(&models.Distributor{
		UserID:     2,
		ParentID:   &parent.ID,
		InviteCode: "INV002",
		Level:      models.DistributorLevelIndirect,
		Status:     models.DistributorStatusApproved,
	})

	db.Model(&models.Distributor{}).Create(map[string]interface{}{
		"user_id":     3,
		"invite_code": "INV003",
		"level":       models.DistributorLevelDirect,
		"status":      models.DistributorStatusPending,
	})

	// 获取所有分销商
	list, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 3, len(list))

	// 按状态过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": models.DistributorStatusApproved,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按上级过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"parent_id": parent.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按层级过滤
	list, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"level": models.DistributorLevelDirect,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestDistributorRepository_ListByParentID(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	parent := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(parent)

	db.Create(&models.Distributor{
		UserID:     2,
		ParentID:   &parent.ID,
		InviteCode: "INV002",
		Level:      models.DistributorLevelIndirect,
	})

	db.Create(&models.Distributor{
		UserID:     3,
		ParentID:   &parent.ID,
		InviteCode: "INV003",
		Level:      models.DistributorLevelIndirect,
	})

	list, total, err := repo.ListByParentID(ctx, parent.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestDistributorRepository_GetDirectMembers(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	parent := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
	}
	db.Create(parent)

	db.Create(&models.Distributor{
		UserID:     2,
		ParentID:   &parent.ID,
		InviteCode: "INV002",
		Level:      models.DistributorLevelIndirect,
	})

	list, total, err := repo.GetDirectMembers(ctx, parent.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
}

func TestDistributorRepository_AddCommission(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:              1,
		InviteCode:          "INV001",
		Level:               models.DistributorLevelDirect,
		TotalCommission:     100.0,
		AvailableCommission: 100.0,
	}
	db.Create(distributor)

	err := repo.AddCommission(ctx, distributor.ID, 50.0)
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, 150.0, found.TotalCommission)
	assert.Equal(t, 150.0, found.AvailableCommission)
}

func TestDistributorRepository_FreezeCommission(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:              1,
		InviteCode:          "INV001",
		Level:               models.DistributorLevelDirect,
		AvailableCommission: 100.0,
		FrozenCommission:    0.0,
	}
	db.Create(distributor)

	err := repo.FreezeCommission(ctx, distributor.ID, 50.0)
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, 50.0, found.AvailableCommission)
	assert.Equal(t, 50.0, found.FrozenCommission)

	// 测试余额不足
	err = repo.FreezeCommission(ctx, distributor.ID, 100.0)
	assert.Error(t, err)
}

func TestDistributorRepository_UnfreezeCommission(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:              1,
		InviteCode:          "INV001",
		Level:               models.DistributorLevelDirect,
		AvailableCommission: 50.0,
		FrozenCommission:    50.0,
	}
	db.Create(distributor)

	err := repo.UnfreezeCommission(ctx, distributor.ID, 30.0)
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, 80.0, found.AvailableCommission)
	assert.Equal(t, 20.0, found.FrozenCommission)
}

func TestDistributorRepository_ConfirmWithdraw(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:              1,
		InviteCode:          "INV001",
		Level:               models.DistributorLevelDirect,
		FrozenCommission:    50.0,
		WithdrawnCommission: 0.0,
	}
	db.Create(distributor)

	err := repo.ConfirmWithdraw(ctx, distributor.ID, 30.0)
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, 20.0, found.FrozenCommission)
	assert.Equal(t, 30.0, found.WithdrawnCommission)
}

func TestDistributorRepository_IncrementTeamCount(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
		TeamCount:  0,
	}
	db.Create(distributor)

	err := repo.IncrementTeamCount(ctx, distributor.ID)
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, 1, found.TeamCount)
}

func TestDistributorRepository_IncrementDirectCount(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	distributor := &models.Distributor{
		UserID:      1,
		InviteCode:  "INV001",
		Level:       models.DistributorLevelDirect,
		DirectCount: 0,
	}
	db.Create(distributor)

	err := repo.IncrementDirectCount(ctx, distributor.ID)
	require.NoError(t, err)

	var found models.Distributor
	db.First(&found, distributor.ID)
	assert.Equal(t, 1, found.DirectCount)
}

func TestDistributorRepository_GetPendingList(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	db.Create(&models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
		Status:     models.DistributorStatusPending,
	})

	db.Create(&models.Distributor{
		UserID:     2,
		InviteCode: "INV002",
		Level:      models.DistributorLevelDirect,
		Status:     models.DistributorStatusApproved,
	})

	db.Model(&models.Distributor{}).Create(map[string]interface{}{
		"user_id":     3,
		"invite_code": "INV003",
		"level":       models.DistributorLevelDirect,
		"status":      models.DistributorStatusPending,
	})

	list, total, err := repo.GetPendingList(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestDistributorRepository_CountByStatus(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	db.Create(&models.Distributor{
		UserID:     1,
		InviteCode: "INV001",
		Level:      models.DistributorLevelDirect,
		Status:     models.DistributorStatusApproved,
	})

	db.Create(&models.Distributor{
		UserID:     2,
		InviteCode: "INV002",
		Level:      models.DistributorLevelDirect,
		Status:     models.DistributorStatusApproved,
	})

	db.Model(&models.Distributor{}).Create(map[string]interface{}{
		"user_id":     3,
		"invite_code": "INV003",
		"level":       models.DistributorLevelDirect,
		"status":      models.DistributorStatusPending,
	})

	count, err := repo.CountByStatus(ctx, models.DistributorStatusApproved)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestDistributorRepository_GetTopDistributors(t *testing.T) {
	db := setupDistributorTestDB(t)
	repo := NewDistributorRepository(db)
	ctx := context.Background()

	db.Create(&models.Distributor{
		UserID:          1,
		InviteCode:      "INV001",
		Level:           models.DistributorLevelDirect,
		Status:          models.DistributorStatusApproved,
		TotalCommission: 1000.0,
	})

	db.Create(&models.Distributor{
		UserID:          2,
		InviteCode:      "INV002",
		Level:           models.DistributorLevelDirect,
		Status:          models.DistributorStatusApproved,
		TotalCommission: 2000.0,
	})

	db.Create(&models.Distributor{
		UserID:          3,
		InviteCode:      "INV003",
		Level:           models.DistributorLevelDirect,
		Status:          models.DistributorStatusApproved,
		TotalCommission: 1500.0,
	})

	db.Model(&models.Distributor{}).Create(map[string]interface{}{
		"user_id":          4,
		"invite_code":      "INV004",
		"level":            models.DistributorLevelDirect,
		"status":           models.DistributorStatusPending,
		"total_commission": 3000.0,
	})

	list, err := repo.GetTopDistributors(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
	// 应该按佣金降序排列
	assert.Equal(t, 2000.0, list[0].TotalCommission)
	assert.Equal(t, 1500.0, list[1].TotalCommission)
}
