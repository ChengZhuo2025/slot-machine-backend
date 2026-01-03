//go:build integration
// +build integration

package integration

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
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

func setupUS2PermissionIntegrationDB(t *testing.T) *gorm.DB {
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

	require.NoError(t, db.AutoMigrate(
		&models.Admin{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
	))
	return db
}

func TestUS2PermissionFlow_DeleteRoleBlockedWhenHasAdmins(t *testing.T) {
	db := setupUS2PermissionIntegrationDB(t)
	ctx := context.Background()

	adminRepo := repository.NewAdminRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	svc := adminService.NewPermissionService(roleRepo, permissionRepo, adminRepo)

	perm, err := svc.CreatePermission(ctx, &adminService.CreatePermissionRequest{
		Code: "device:read",
		Name: "设备查看",
		Type: models.PermissionTypeAPI,
	})
	require.NoError(t, err)
	require.NotNil(t, perm)

	role, err := svc.CreateRole(ctx, &adminService.CreateRoleRequest{
		Code:          "role_test_1",
		Name:          "测试角色",
		PermissionIDs: []int64{perm.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, role)

	// Create admin binds role
	admin := &models.Admin{
		Username:     "perm_admin",
		PasswordHash: "hash",
		Name:         "管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	require.NoError(t, db.Create(admin).Error)

	// Should be blocked
	err = svc.DeleteRole(ctx, role.ID)
	assert.ErrorIs(t, err, adminService.ErrRoleHasAdmins)

	// Remove admin then delete role
	require.NoError(t, db.Delete(&models.Admin{}, admin.ID).Error)
	require.NoError(t, svc.DeleteRole(ctx, role.ID))
}

func TestUS2PermissionFlow_DeleteSystemRoleBlocked(t *testing.T) {
	db := setupUS2PermissionIntegrationDB(t)
	ctx := context.Background()

	adminRepo := repository.NewAdminRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	svc := adminService.NewPermissionService(roleRepo, permissionRepo, adminRepo)

	role := &models.Role{Code: "sys_role_1", Name: "系统角色", IsSystem: true}
	require.NoError(t, db.Create(role).Error)

	err := svc.DeleteRole(ctx, role.ID)
	assert.ErrorIs(t, err, adminService.ErrRoleIsSystem)
}

