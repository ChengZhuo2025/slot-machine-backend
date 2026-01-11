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

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupPermissionServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.Admin{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
	))

	return db
}

func setupPermissionService(db *gorm.DB) *PermissionService {
	return NewPermissionService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		repository.NewAdminRepository(db),
	)
}

func TestPermissionService_RoleAndPermissionCRUD(t *testing.T) {
	db := setupPermissionServiceTestDB(t)
	svc := setupPermissionService(db)
	ctx := context.Background()

	menuPerm := &models.Permission{
		Code: "menu:dashboard",
		Name: "仪表盘",
		Type: models.PermissionTypeMenu,
		Sort: 1,
	}
	apiPath := "/api/test"
	apiMethod := "GET"
	apiPerm := &models.Permission{
		Code:   "api:test:get",
		Name:   "测试接口",
		Type:   models.PermissionTypeAPI,
		Path:   &apiPath,
		Method: &apiMethod,
		Sort:   1,
	}
	require.NoError(t, db.Create(menuPerm).Error)
	require.NoError(t, db.Create(apiPerm).Error)

	t.Run("CreateRole 成功并设置权限", func(t *testing.T) {
		role, err := svc.CreateRole(ctx, &CreateRoleRequest{
			Code:          "r1",
			Name:          "角色1",
			PermissionIDs: []int64{menuPerm.ID, apiPerm.ID},
		})
		require.NoError(t, err)
		require.NotNil(t, role)
		assert.Equal(t, "r1", role.Code)

		got, err := repository.NewRoleRepository(db).GetByIDWithPermissions(ctx, role.ID)
		require.NoError(t, err)
		require.Len(t, got.Permissions, 2)
	})

	t.Run("CreateRole 编码冲突", func(t *testing.T) {
		_, err := svc.CreateRole(ctx, &CreateRoleRequest{Code: "r1", Name: "角色重复"})
		require.Error(t, err)
		assert.Equal(t, ErrRoleCodeExists, err)
	})

	t.Run("UpdateRole 系统角色不修改名称但更新权限", func(t *testing.T) {
		sysRole := &models.Role{Code: "sys", Name: "系统角色", IsSystem: true}
		require.NoError(t, db.Create(sysRole).Error)
		require.NoError(t, repository.NewRoleRepository(db).SetPermissions(ctx, sysRole.ID, []int64{menuPerm.ID}))

		err := svc.UpdateRole(ctx, sysRole.ID, &UpdateRoleRequest{
			Name:          "新名称",
			PermissionIDs: []int64{apiPerm.ID},
		})
		require.NoError(t, err)

		var got models.Role
		require.NoError(t, db.First(&got, sysRole.ID).Error)
		assert.Equal(t, "系统角色", got.Name)

		gotWithPerm, err := repository.NewRoleRepository(db).GetByIDWithPermissions(ctx, sysRole.ID)
		require.NoError(t, err)
		require.Len(t, gotWithPerm.Permissions, 1)
		assert.Equal(t, "api:test:get", gotWithPerm.Permissions[0].Code)
	})

	t.Run("DeleteRole 角色下有管理员不允许删除", func(t *testing.T) {
		role := &models.Role{Code: "r_admin", Name: "有管理员"}
		require.NoError(t, db.Create(role).Error)
		admin := &models.Admin{Username: "a1", PasswordHash: "x", Name: "管理员", RoleID: role.ID, Status: models.AdminStatusActive}
		require.NoError(t, db.Create(admin).Error)

		err := svc.DeleteRole(ctx, role.ID)
		require.Error(t, err)
		assert.Equal(t, ErrRoleHasAdmins, err)
	})

	t.Run("DeleteRole 系统角色不允许删除", func(t *testing.T) {
		role := &models.Role{Code: "r_sys_del", Name: "系统", IsSystem: true}
		require.NoError(t, db.Create(role).Error)

		err := svc.DeleteRole(ctx, role.ID)
		require.Error(t, err)
		assert.Equal(t, ErrRoleIsSystem, err)
	})

	t.Run("CreatePermission 父权限不存在", func(t *testing.T) {
		pid := int64(99999)
		_, err := svc.CreatePermission(ctx, &CreatePermissionRequest{
			Code:     "p.child",
			Name:     "子权限",
			Type:     models.PermissionTypeMenu,
			ParentID: &pid,
		})
		require.Error(t, err)
		assert.Equal(t, ErrPermissionNotFound, err)
	})

	t.Run("CreatePermission 编码冲突", func(t *testing.T) {
		_, err := svc.CreatePermission(ctx, &CreatePermissionRequest{
			Code: "menu:dashboard",
			Name: "重复",
			Type: models.PermissionTypeMenu,
		})
		require.Error(t, err)
		assert.Equal(t, ErrPermissionCodeExists, err)
	})

	t.Run("DeletePermission 有子权限不允许删除", func(t *testing.T) {
		parent, err := svc.CreatePermission(ctx, &CreatePermissionRequest{
			Code: "menu:parent",
			Name: "父权限",
			Type: models.PermissionTypeMenu,
		})
		require.NoError(t, err)
		_, err = svc.CreatePermission(ctx, &CreatePermissionRequest{
			Code:     "menu:child",
			Name:     "子权限",
			Type:     models.PermissionTypeMenu,
			ParentID: &parent.ID,
		})
		require.NoError(t, err)

		err = svc.DeletePermission(ctx, parent.ID)
		require.Error(t, err)
		assert.Equal(t, ErrPermissionHasChildren, err)
	})
}

func TestPermissionService_RoleOperations(t *testing.T) {
	db := setupPermissionServiceTestDB(t)
	svc := setupPermissionService(db)
	ctx := context.Background()

	perm := &models.Permission{Code: "ops:read", Name: "操作查看", Type: models.PermissionTypeMenu}
	require.NoError(t, db.Create(perm).Error)

	role := &models.Role{Code: "ops_role", Name: "操作角色"}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, repository.NewRoleRepository(db).SetPermissions(ctx, role.ID, []int64{perm.ID}))

	t.Run("GetRole 获取角色详情", func(t *testing.T) {
		got, err := svc.GetRole(ctx, role.ID)
		require.NoError(t, err)
		assert.Equal(t, "ops_role", got.Code)
		assert.Len(t, got.Permissions, 1)
	})

	t.Run("GetRole 角色不存在", func(t *testing.T) {
		_, err := svc.GetRole(ctx, 99999)
		assert.Equal(t, ErrRoleNotFound, err)
	})

	t.Run("ListRoles 获取角色列表", func(t *testing.T) {
		list, total, err := svc.ListRoles(ctx, 0, 10, nil)
		require.NoError(t, err)
		assert.True(t, total >= 1)
		assert.NotEmpty(t, list)
	})

	t.Run("ListAllRoles 获取所有角色", func(t *testing.T) {
		list, err := svc.ListAllRoles(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
	})

	t.Run("SetRolePermissions 设置角色权限", func(t *testing.T) {
		perm2 := &models.Permission{Code: "ops:write", Name: "操作写入", Type: models.PermissionTypeMenu}
		require.NoError(t, db.Create(perm2).Error)

		err := svc.SetRolePermissions(ctx, role.ID, []int64{perm.ID, perm2.ID})
		require.NoError(t, err)

		got, _ := svc.GetRole(ctx, role.ID)
		assert.Len(t, got.Permissions, 2)
	})

	t.Run("SetRolePermissions 角色不存在", func(t *testing.T) {
		err := svc.SetRolePermissions(ctx, 99999, []int64{perm.ID})
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestPermissionService_PermissionOperations(t *testing.T) {
	db := setupPermissionServiceTestDB(t)
	svc := setupPermissionService(db)
	ctx := context.Background()

	perm, err := svc.CreatePermission(ctx, &CreatePermissionRequest{
		Code: "perm:test",
		Name: "测试权限",
		Type: models.PermissionTypeMenu,
	})
	require.NoError(t, err)

	t.Run("GetPermission 获取权限详情", func(t *testing.T) {
		got, err := svc.GetPermission(ctx, perm.ID)
		require.NoError(t, err)
		assert.Equal(t, "perm:test", got.Code)
	})

	t.Run("GetPermission 权限不存在", func(t *testing.T) {
		_, err := svc.GetPermission(ctx, 99999)
		assert.Equal(t, ErrPermissionNotFound, err)
	})

	t.Run("UpdatePermission 更新权限", func(t *testing.T) {
		path := "/api/updated"
		method := "POST"
		err := svc.UpdatePermission(ctx, perm.ID, &UpdatePermissionRequest{
			Name:   "更新后权限",
			Path:   &path,
			Method: &method,
		})
		require.NoError(t, err)

		var updated models.Permission
		db.First(&updated, perm.ID)
		assert.Equal(t, "更新后权限", updated.Name)
		assert.Equal(t, "/api/updated", *updated.Path)
	})

	t.Run("UpdatePermission 权限不存在", func(t *testing.T) {
		err := svc.UpdatePermission(ctx, 99999, &UpdatePermissionRequest{Name: "x"})
		assert.Equal(t, ErrPermissionNotFound, err)
	})

	t.Run("ListPermissions 获取权限列表", func(t *testing.T) {
		list, err := svc.ListPermissions(ctx, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
	})

	t.Run("ListPermissionTree 获取权限树", func(t *testing.T) {
		list, err := svc.ListPermissionTree(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})
}

func TestPermissionService_AdminPermissions(t *testing.T) {
	db := setupPermissionServiceTestDB(t)
	svc := setupPermissionService(db)
	ctx := context.Background()

	perm := &models.Permission{Code: "admin:menu", Name: "管理菜单", Type: models.PermissionTypeMenu}
	require.NoError(t, db.Create(perm).Error)

	role := &models.Role{Code: "admin_role", Name: "管理角色"}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, repository.NewRoleRepository(db).SetPermissions(ctx, role.ID, []int64{perm.ID}))

	admin := &models.Admin{Username: "testadmin", PasswordHash: "x", Name: "测试管理员", RoleID: role.ID, Status: models.AdminStatusActive}
	require.NoError(t, db.Create(admin).Error)

	t.Run("GetAdminPermissions 获取管理员权限列表", func(t *testing.T) {
		perms, err := svc.GetAdminPermissions(ctx, admin.ID)
		require.NoError(t, err)
		assert.Contains(t, perms, "admin:menu")
	})

	t.Run("GetAdminMenus 获取管理员菜单", func(t *testing.T) {
		menus, err := svc.GetAdminMenus(ctx, admin.ID)
		require.NoError(t, err)
		assert.NotNil(t, menus)
	})

	t.Run("GetAdminMenus 超管获取所有菜单", func(t *testing.T) {
		superRole := &models.Role{Code: models.RoleCodeSuperAdmin, Name: "超管", IsSystem: true}
		require.NoError(t, db.Create(superRole).Error)
		superAdmin := &models.Admin{Username: "superadmin", PasswordHash: "x", Name: "超管", RoleID: superRole.ID, Status: models.AdminStatusActive}
		require.NoError(t, db.Create(superAdmin).Error)

		menus, err := svc.GetAdminMenus(ctx, superAdmin.ID)
		require.NoError(t, err)
		assert.NotNil(t, menus)
	})
}

func TestPermissionService_CheckPermission(t *testing.T) {
	db := setupPermissionServiceTestDB(t)
	svc := setupPermissionService(db)
	ctx := context.Background()

	perm := &models.Permission{Code: "device:read", Name: "设备查看", Type: models.PermissionTypeAPI}
	require.NoError(t, db.Create(perm).Error)

	path := "/api/v1/devices"
	method := "GET"
	apiPerm := &models.Permission{Code: "device:list", Name: "设备列表", Type: models.PermissionTypeAPI, Path: &path, Method: &method}
	require.NoError(t, db.Create(apiPerm).Error)

	roleRepo := repository.NewRoleRepository(db)

	superRole := &models.Role{Code: models.RoleCodeSuperAdmin, Name: "超管", IsSystem: true}
	require.NoError(t, db.Create(superRole).Error)
	superAdmin := &models.Admin{Username: "super", PasswordHash: "x", Name: "超管", RoleID: superRole.ID, Status: models.AdminStatusActive}
	require.NoError(t, db.Create(superAdmin).Error)

	ok, err := svc.CheckPermission(ctx, superAdmin.ID, "any:perm")
	require.NoError(t, err)
	assert.True(t, ok)

	normalRole := &models.Role{Code: "normal", Name: "普通"}
	require.NoError(t, db.Create(normalRole).Error)
	require.NoError(t, roleRepo.SetPermissions(ctx, normalRole.ID, []int64{perm.ID, apiPerm.ID}))
	admin := &models.Admin{Username: "a1", PasswordHash: "x", Name: "管理员", RoleID: normalRole.ID, Status: models.AdminStatusActive}
	require.NoError(t, db.Create(admin).Error)

	ok, err = svc.CheckPermission(ctx, admin.ID, "device:read")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = svc.CheckPermission(ctx, admin.ID, "device:write")
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = svc.CheckAPIPermission(ctx, admin.ID, "/api/v1/devices", "GET")
	require.NoError(t, err)
	assert.True(t, ok)
}

