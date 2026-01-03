// Package repository 管理员仓储单元测试
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

// setupAdminRepoTestDB 创建测试数据库
func setupAdminRepoTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Admin{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.Merchant{},
	)
	require.NoError(t, err)

	return db
}

// createTestRole 创建测试角色
func createTestRole(t *testing.T, db *gorm.DB, code, name string) *models.Role {
	role := &models.Role{
		Code:     code,
		Name:     name,
		IsSystem: false,
	}
	err := db.Create(role).Error
	require.NoError(t, err)
	return role
}

func TestAdminRepository_Create(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role", "测试角色")

	admin := &models.Admin{
		Username:     "testadmin",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}

	err := repo.Create(ctx, admin)

	require.NoError(t, err)
	assert.NotZero(t, admin.ID)
}

func TestAdminRepository_GetByID(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role", "测试角色")

	admin := &models.Admin{
		Username:     "getbyid_admin",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	result, err := repo.GetByID(ctx, admin.ID)

	require.NoError(t, err)
	assert.Equal(t, admin.ID, result.ID)
	assert.Equal(t, "getbyid_admin", result.Username)
}

func TestAdminRepository_GetByID_NotFound(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)

	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestAdminRepository_GetByIDWithRole(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "role_with_admin", "角色")

	admin := &models.Admin{
		Username:     "withrole_admin",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	result, err := repo.GetByIDWithRole(ctx, admin.ID)

	require.NoError(t, err)
	assert.NotNil(t, result.Role)
	assert.Equal(t, "role_with_admin", result.Role.Code)
}

func TestAdminRepository_GetByIDWithRoleAndPermissions(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "role_with_perms", "角色")

	// 创建权限
	permission := &models.Permission{
		Code: "device:read",
		Name: "设备查看",
		Type: models.PermissionTypeAPI,
	}
	db.Create(permission)

	// 关联角色和权限
	rolePermission := &models.RolePermission{
		RoleID:       role.ID,
		PermissionID: permission.ID,
	}
	db.Create(rolePermission)

	admin := &models.Admin{
		Username:     "withperms_admin",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	result, err := repo.GetByIDWithRoleAndPermissions(ctx, admin.ID)

	require.NoError(t, err)
	assert.NotNil(t, result.Role)
	assert.Len(t, result.Role.Permissions, 1)
	assert.Equal(t, "device:read", result.Role.Permissions[0].Code)
}

func TestAdminRepository_GetByUsername(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role2", "测试角色")

	admin := &models.Admin{
		Username:     "unique_username",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	result, err := repo.GetByUsername(ctx, "unique_username")

	require.NoError(t, err)
	assert.Equal(t, admin.ID, result.ID)
	assert.Equal(t, "unique_username", result.Username)
}

func TestAdminRepository_GetByUsername_NotFound(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	_, err := repo.GetByUsername(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestAdminRepository_Update(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role3", "测试角色")

	admin := &models.Admin{
		Username:     "update_admin",
		PasswordHash: "hashedpassword",
		Name:         "原名称",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	admin.Name = "新名称"
	err := repo.Update(ctx, admin)

	require.NoError(t, err)

	var updated models.Admin
	db.First(&updated, admin.ID)
	assert.Equal(t, "新名称", updated.Name)
}

func TestAdminRepository_UpdateFields(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role4", "测试角色")

	admin := &models.Admin{
		Username:     "updatefields_admin",
		PasswordHash: "hashedpassword",
		Name:         "原名称",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	phone := "13800138000"
	err := repo.UpdateFields(ctx, admin.ID, map[string]interface{}{
		"name":  "更新名称",
		"phone": phone,
	})

	require.NoError(t, err)

	var updated models.Admin
	db.First(&updated, admin.ID)
	assert.Equal(t, "更新名称", updated.Name)
	assert.Equal(t, &phone, updated.Phone)
}

func TestAdminRepository_UpdateStatus(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role5", "测试角色")

	admin := &models.Admin{
		Username:     "status_admin",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	err := repo.UpdateStatus(ctx, admin.ID, models.AdminStatusDisabled)

	require.NoError(t, err)

	var updated models.Admin
	db.First(&updated, admin.ID)
	assert.Equal(t, models.AdminStatusDisabled, updated.Status)
}

func TestAdminRepository_UpdatePassword(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role6", "测试角色")

	admin := &models.Admin{
		Username:     "password_admin",
		PasswordHash: "oldhash",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	err := repo.UpdatePassword(ctx, admin.ID, "newhash")

	require.NoError(t, err)

	var updated models.Admin
	db.First(&updated, admin.ID)
	assert.Equal(t, "newhash", updated.PasswordHash)
}

func TestAdminRepository_UpdateLoginInfo(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role7", "测试角色")

	admin := &models.Admin{
		Username:     "logininfo_admin",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	err := repo.UpdateLoginInfo(ctx, admin.ID, "192.168.1.1")

	require.NoError(t, err)

	var updated models.Admin
	db.First(&updated, admin.ID)
	assert.NotNil(t, updated.LastLoginAt)
	assert.Equal(t, "192.168.1.1", *updated.LastLoginIP)
}

func TestAdminRepository_Delete(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role8", "测试角色")

	admin := &models.Admin{
		Username:     "delete_admin",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	err := repo.Delete(ctx, admin.ID)

	require.NoError(t, err)

	// 验证已软删除
	var count int64
	db.Model(&models.Admin{}).Where("id = ?", admin.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestAdminRepository_List(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role9", "测试角色")

	// 创建多个管理员
	for i := 0; i < 5; i++ {
		admin := &models.Admin{
			Username:     "list_admin_" + string(rune('A'+i)),
			PasswordHash: "hashedpassword",
			Name:         "管理员" + string(rune('A'+i)),
			RoleID:       role.ID,
			Status:       models.AdminStatusActive,
		}
		db.Create(admin)
	}

	admins, total, err := repo.List(ctx, 0, 10, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, admins, 5)
}

func TestAdminRepository_List_WithFilters(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role1 := createTestRole(t, db, "role_a", "角色A")
	role2 := createTestRole(t, db, "role_b", "角色B")

	// 创建不同角色的管理员
	admin1 := &models.Admin{
		Username:     "filter_admin_1",
		PasswordHash: "hashedpassword",
		Name:         "管理员1",
		RoleID:       role1.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin1)

	admin2 := &models.Admin{
		Username:     "filter_admin_2",
		PasswordHash: "hashedpassword",
		Name:         "管理员2",
		RoleID:       role2.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin2)

	// 按角色过滤
	filters := map[string]interface{}{
		"role_id": role1.ID,
	}
	admins, total, err := repo.List(ctx, 0, 10, filters)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, admins, 1)
	assert.Equal(t, "filter_admin_1", admins[0].Username)
}

func TestAdminRepository_List_WithPagination(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role10", "测试角色")

	// 创建多个管理员
	for i := 0; i < 10; i++ {
		admin := &models.Admin{
			Username:     "page_admin_" + string(rune('A'+i)),
			PasswordHash: "hashedpassword",
			Name:         "管理员" + string(rune('A'+i)),
			RoleID:       role.ID,
			Status:       models.AdminStatusActive,
		}
		db.Create(admin)
	}

	// 获取第二页
	admins, total, err := repo.List(ctx, 5, 5, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(10), total)
	assert.Len(t, admins, 5)
}

func TestAdminRepository_ExistsByUsername(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role11", "测试角色")

	admin := &models.Admin{
		Username:     "exists_admin",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	// 存在的用户名
	exists, err := repo.ExistsByUsername(ctx, "exists_admin")
	require.NoError(t, err)
	assert.True(t, exists)

	// 不存在的用户名
	exists, err = repo.ExistsByUsername(ctx, "nonexistent_admin")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestAdminRepository_ExistsByUsernameExcludeID(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role12", "测试角色")

	admin1 := &models.Admin{
		Username:     "exclude_admin_1",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员1",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin1)

	admin2 := &models.Admin{
		Username:     "exclude_admin_2",
		PasswordHash: "hashedpassword",
		Name:         "测试管理员2",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin2)

	// 排除自己的 ID，检查是否存在相同用户名
	exists, err := repo.ExistsByUsernameExcludeID(ctx, "exclude_admin_1", admin1.ID)
	require.NoError(t, err)
	assert.False(t, exists) // 排除自己后不应存在

	// 检查其他用户的用户名
	exists, err = repo.ExistsByUsernameExcludeID(ctx, "exclude_admin_2", admin1.ID)
	require.NoError(t, err)
	assert.True(t, exists) // 其他用户的用户名应该存在
}

func TestAdminRepository_GetByMerchantID(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role := createTestRole(t, db, "test_role13", "测试角色")

	// 创建商户
	merchant := &models.Merchant{
		Name:   "测试商户",
		Status: models.MerchantStatusActive,
	}
	db.Create(merchant)

	merchantID := merchant.ID

	admin1 := &models.Admin{
		Username:     "merchant_admin_1",
		PasswordHash: "hashedpassword",
		Name:         "商户管理员1",
		RoleID:       role.ID,
		MerchantID:   &merchantID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin1)

	admin2 := &models.Admin{
		Username:     "merchant_admin_2",
		PasswordHash: "hashedpassword",
		Name:         "商户管理员2",
		RoleID:       role.ID,
		MerchantID:   &merchantID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin2)

	admins, err := repo.GetByMerchantID(ctx, merchantID)

	require.NoError(t, err)
	assert.Len(t, admins, 2)
}

func TestAdminRepository_CountByRoleID(t *testing.T) {
	db := setupAdminRepoTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	role1 := createTestRole(t, db, "count_role_1", "角色1")
	role2 := createTestRole(t, db, "count_role_2", "角色2")

	// 创建 3 个使用 role1 的管理员
	for i := 0; i < 3; i++ {
		admin := &models.Admin{
			Username:     "count_admin_" + string(rune('A'+i)),
			PasswordHash: "hashedpassword",
			Name:         "管理员",
			RoleID:       role1.ID,
			Status:       models.AdminStatusActive,
		}
		db.Create(admin)
	}

	// 创建 1 个使用 role2 的管理员
	admin := &models.Admin{
		Username:     "count_admin_D",
		PasswordHash: "hashedpassword",
		Name:         "管理员",
		RoleID:       role2.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	count1, err := repo.CountByRoleID(ctx, role1.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count1)

	count2, err := repo.CountByRoleID(ctx, role2.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count2)
}
