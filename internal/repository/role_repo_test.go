// Package repository 角色权限仓储单元测试
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

func setupRoleTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Role{}, &models.Permission{}, &models.RolePermission{})
	require.NoError(t, err)

	return db
}

// RoleRepository 测试

func TestRoleRepository_Create(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{
		Code: "admin",
		Name: "管理员",
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)
	assert.NotZero(t, role.ID)
}

func TestRoleRepository_GetByID(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{
		Code: "admin",
		Name: "管理员",
	}
	db.Create(role)

	found, err := repo.GetByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, role.ID, found.ID)
	assert.Equal(t, "admin", found.Code)
}

func TestRoleRepository_GetByIDWithPermissions(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{
		Code: "admin",
		Name: "管理员",
	}
	db.Create(role)

	perm := &models.Permission{
		Code: "user.create",
		Name: "创建用户",
		Type: models.PermissionTypeAPI,
	}
	db.Create(perm)

	db.Create(&models.RolePermission{
		RoleID:       role.ID,
		PermissionID: perm.ID,
	})

	found, err := repo.GetByIDWithPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, role.ID, found.ID)
	assert.Equal(t, 1, len(found.Permissions))
}

func TestRoleRepository_GetByCode(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	db.Create(&models.Role{
		Code: "admin",
		Name: "管理员",
	})

	found, err := repo.GetByCode(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", found.Code)
}

func TestRoleRepository_Update(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{
		Code: "admin",
		Name: "管理员",
	}
	db.Create(role)

	role.Name = "超级管理员"
	err := repo.Update(ctx, role)
	require.NoError(t, err)

	var found models.Role
	db.First(&found, role.ID)
	assert.Equal(t, "超级管理员", found.Name)
}

func TestRoleRepository_UpdateFields(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{
		Code: "admin",
		Name: "管理员",
	}
	db.Create(role)

	desc := "系统管理员"
	err := repo.UpdateFields(ctx, role.ID, map[string]interface{}{
		"description": desc,
	})
	require.NoError(t, err)

	var found models.Role
	db.First(&found, role.ID)
	assert.NotNil(t, found.Description)
	assert.Equal(t, desc, *found.Description)
}

func TestRoleRepository_Delete(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{
		Code: "test_role",
		Name: "测试角色",
	}
	db.Create(role)

	err := repo.Delete(ctx, role.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Role{}).Where("id = ?", role.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestRoleRepository_List(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	db.Create(&models.Role{Code: "admin", Name: "管理员", IsSystem: true})
	db.Create(&models.Role{Code: "user", Name: "普通用户"})

	db.Model(&models.Role{}).Create(map[string]interface{}{
		"code": "operator", "name": "操作员", "is_system": false,
	})

	// 获取所有角色
	_, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按系统角色过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"is_system": true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestRoleRepository_ListAll(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	db.Create(&models.Role{Code: "admin", Name: "管理员"})
	db.Create(&models.Role{Code: "user", Name: "普通用户"})

	roles, err := repo.ListAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(roles))
}

func TestRoleRepository_ExistsByCode(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	db.Create(&models.Role{Code: "admin", Name: "管理员"})

	exists, err := repo.ExistsByCode(ctx, "admin")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByCode(ctx, "not_exists")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRoleRepository_SetPermissions(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{Code: "admin", Name: "管理员"}
	db.Create(role)

	perm1 := &models.Permission{Code: "user.create", Name: "创建用户", Type: models.PermissionTypeAPI}
	perm2 := &models.Permission{Code: "user.delete", Name: "删除用户", Type: models.PermissionTypeAPI}
	db.Create(perm1)
	db.Create(perm2)

	err := repo.SetPermissions(ctx, role.ID, []int64{perm1.ID, perm2.ID})
	require.NoError(t, err)

	var count int64
	db.Model(&models.RolePermission{}).Where("role_id = ?", role.ID).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestRoleRepository_GetPermissionIDs(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{Code: "admin", Name: "管理员"}
	db.Create(role)

	perm1 := &models.Permission{Code: "user.create", Name: "创建用户", Type: models.PermissionTypeAPI}
	perm2 := &models.Permission{Code: "user.delete", Name: "删除用户", Type: models.PermissionTypeAPI}
	db.Create(perm1)
	db.Create(perm2)

	db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm1.ID})
	db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm2.ID})

	ids, err := repo.GetPermissionIDs(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(ids))
}

// PermissionRepository 测试

func TestPermissionRepository_Create(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	perm := &models.Permission{
		Code: "user.create",
		Name: "创建用户",
		Type: models.PermissionTypeAPI,
	}

	err := repo.Create(ctx, perm)
	require.NoError(t, err)
	assert.NotZero(t, perm.ID)
}

func TestPermissionRepository_GetByID(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	perm := &models.Permission{
		Code: "user.create",
		Name: "创建用户",
		Type: models.PermissionTypeAPI,
	}
	db.Create(perm)

	found, err := repo.GetByID(ctx, perm.ID)
	require.NoError(t, err)
	assert.Equal(t, perm.ID, found.ID)
}

func TestPermissionRepository_GetByCode(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Permission{
		Code: "user.create",
		Name: "创建用户",
		Type: models.PermissionTypeAPI,
	})

	found, err := repo.GetByCode(ctx, "user.create")
	require.NoError(t, err)
	assert.Equal(t, "user.create", found.Code)
}

func TestPermissionRepository_Update(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	perm := &models.Permission{
		Code: "user.create",
		Name: "创建用户",
		Type: models.PermissionTypeAPI,
	}
	db.Create(perm)

	perm.Name = "新建用户"
	err := repo.Update(ctx, perm)
	require.NoError(t, err)

	var found models.Permission
	db.First(&found, perm.ID)
	assert.Equal(t, "新建用户", found.Name)
}

func TestPermissionRepository_Delete(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	perm := &models.Permission{
		Code: "test.perm",
		Name: "测试权限",
		Type: models.PermissionTypeAPI,
	}
	db.Create(perm)

	err := repo.Delete(ctx, perm.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Permission{}).Where("id = ?", perm.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestPermissionRepository_List(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Permission{Code: "user.create", Name: "创建用户", Type: models.PermissionTypeAPI})
	db.Create(&models.Permission{Code: "menu.user", Name: "用户菜单", Type: models.PermissionTypeMenu})

	// 获取所有权限
	perms, err := repo.List(ctx, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(perms))

	// 按类型过滤
	perms, err = repo.List(ctx, map[string]interface{}{
		"type": models.PermissionTypeAPI,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, len(perms))
}

func TestPermissionRepository_ListAll(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Permission{Code: "user.create", Name: "创建用户", Type: models.PermissionTypeAPI})
	db.Create(&models.Permission{Code: "user.delete", Name: "删除用户", Type: models.PermissionTypeAPI})

	perms, err := repo.ListAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(perms))
}

func TestPermissionRepository_ListByType(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Permission{Code: "user.create", Name: "创建用户", Type: models.PermissionTypeAPI})
	db.Create(&models.Permission{Code: "menu.user", Name: "用户菜单", Type: models.PermissionTypeMenu})

	perms, err := repo.ListByType(ctx, models.PermissionTypeAPI)
	require.NoError(t, err)
	assert.Equal(t, 1, len(perms))
}

func TestPermissionRepository_ExistsByCode(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	db.Create(&models.Permission{
		Code: "user.create",
		Name: "创建用户",
		Type: models.PermissionTypeAPI,
	})

	exists, err := repo.ExistsByCode(ctx, "user.create")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByCode(ctx, "not_exists")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPermissionRepository_GetByIDs(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	perm1 := &models.Permission{Code: "user.create", Name: "创建用户", Type: models.PermissionTypeAPI}
	perm2 := &models.Permission{Code: "user.delete", Name: "删除用户", Type: models.PermissionTypeAPI}
	db.Create(perm1)
	db.Create(perm2)

	perms, err := repo.GetByIDs(ctx, []int64{perm1.ID, perm2.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, len(perms))
}

func TestPermissionRepository_HasChildren(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	parent := &models.Permission{Code: "user", Name: "用户管理", Type: models.PermissionTypeMenu}
	db.Create(parent)

	child := &models.Permission{Code: "user.create", Name: "创建用户", Type: models.PermissionTypeAPI, ParentID: &parent.ID}
	db.Create(child)

	hasChildren, err := repo.HasChildren(ctx, parent.ID)
	require.NoError(t, err)
	assert.True(t, hasChildren)

	hasChildren, err = repo.HasChildren(ctx, child.ID)
	require.NoError(t, err)
	assert.False(t, hasChildren)
}
