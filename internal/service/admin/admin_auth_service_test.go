// Package admin 管理员认证服务单元测试
package admin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/common/crypto"
	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupAdminAuthTestDB 创建测试数据库
func setupAdminAuthTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Admin{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
	)
	require.NoError(t, err)

	return db
}

// createTestAdmin 创建测试管理员
func createTestAdmin(t *testing.T, db *gorm.DB, username, password string) *models.Admin {
	// 创建角色
	role := &models.Role{
		Code:     "test_role",
		Name:     "测试角色",
		IsSystem: false,
	}
	err := db.Create(role).Error
	require.NoError(t, err)

	// 创建权限
	permission := &models.Permission{
		Code: "device:read",
		Name: "设备查看",
		Type: models.PermissionTypeAPI,
	}
	err = db.Create(permission).Error
	require.NoError(t, err)

	// 关联角色和权限
	rolePermission := &models.RolePermission{
		RoleID:       role.ID,
		PermissionID: permission.ID,
	}
	err = db.Create(rolePermission).Error
	require.NoError(t, err)

	// 创建管理员
	passwordHash, err := crypto.HashPassword(password)
	require.NoError(t, err)

	admin := &models.Admin{
		Username:     username,
		PasswordHash: passwordHash,
		Name:         "测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	err = db.Create(admin).Error
	require.NoError(t, err)

	return admin
}

// setupAdminAuthService 创建测试用的 AdminAuthService
func setupAdminAuthService(t *testing.T) (*AdminAuthService, *gorm.DB) {
	db := setupAdminAuthTestDB(t)
	adminRepo := repository.NewAdminRepository(db)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-admin-auth",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	service := NewAdminAuthService(adminRepo, jwtManager)
	return service, db
}

func TestAdminAuthService_Login_Success(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	createTestAdmin(t, db, "testadmin", "password123")

	// 登录
	resp, err := service.Login(ctx, &LoginRequest{
		Username: "testadmin",
		Password: "password123",
		IP:       "127.0.0.1",
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Admin)
	assert.Equal(t, "testadmin", resp.Admin.Username)
	assert.NotEmpty(t, resp.TokenPair.AccessToken)
	assert.NotEmpty(t, resp.TokenPair.RefreshToken)
}

func TestAdminAuthService_Login_AdminNotFound(t *testing.T) {
	service, _ := setupAdminAuthService(t)
	ctx := context.Background()

	// 尝试使用不存在的账号登录
	_, err := service.Login(ctx, &LoginRequest{
		Username: "nonexistent",
		Password: "password123",
		IP:       "127.0.0.1",
	})

	assert.Error(t, err)
	assert.Equal(t, ErrAdminNotFound, err)
}

func TestAdminAuthService_Login_InvalidPassword(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	createTestAdmin(t, db, "testadmin2", "correctpassword")

	// 使用错误密码登录
	_, err := service.Login(ctx, &LoginRequest{
		Username: "testadmin2",
		Password: "wrongpassword",
		IP:       "127.0.0.1",
	})

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidPassword, err)
}

func TestAdminAuthService_Login_DisabledAdmin(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	admin := createTestAdmin(t, db, "disabledadmin", "password123")

	// 禁用管理员
	err := db.Model(admin).Update("status", models.AdminStatusDisabled).Error
	require.NoError(t, err)

	// 尝试登录
	_, err = service.Login(ctx, &LoginRequest{
		Username: "disabledadmin",
		Password: "password123",
		IP:       "127.0.0.1",
	})

	assert.Error(t, err)
	assert.Equal(t, ErrAdminDisabled, err)
}

func TestAdminAuthService_Login_UpdatesLoginInfo(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	admin := createTestAdmin(t, db, "logininfoadmin", "password123")

	// 登录
	_, err := service.Login(ctx, &LoginRequest{
		Username: "logininfoadmin",
		Password: "password123",
		IP:       "192.168.1.1",
	})
	require.NoError(t, err)

	// 验证登录信息已更新
	var updatedAdmin models.Admin
	err = db.First(&updatedAdmin, admin.ID).Error
	require.NoError(t, err)

	assert.NotNil(t, updatedAdmin.LastLoginAt)
	assert.Equal(t, "192.168.1.1", *updatedAdmin.LastLoginIP)
}

func TestAdminAuthService_Login_ReturnsPermissions(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	createTestAdmin(t, db, "permadmin", "password123")

	// 登录
	resp, err := service.Login(ctx, &LoginRequest{
		Username: "permadmin",
		Password: "password123",
		IP:       "127.0.0.1",
	})

	require.NoError(t, err)
	assert.NotNil(t, resp.Permissions)
	assert.Contains(t, resp.Permissions, "device:read")
}

func TestAdminAuthService_GetAdminInfo(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	admin := createTestAdmin(t, db, "infoadmin", "password123")

	// 获取管理员信息
	info, err := service.GetAdminInfo(ctx, admin.ID)

	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, admin.ID, info.ID)
	assert.Equal(t, "infoadmin", info.Username)
	assert.Equal(t, "测试管理员", info.Name)
	assert.Equal(t, "test_role", info.RoleCode)
}

func TestAdminAuthService_GetAdminInfo_NotFound(t *testing.T) {
	service, _ := setupAdminAuthService(t)
	ctx := context.Background()

	// 获取不存在的管理员
	_, err := service.GetAdminInfo(ctx, 99999)

	assert.Error(t, err)
	assert.Equal(t, ErrAdminNotFound, err)
}

func TestAdminAuthService_GetAdminWithPermissions(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	admin := createTestAdmin(t, db, "withpermadmin", "password123")

	// 获取管理员信息（包含权限）
	resp, err := service.GetAdminWithPermissions(ctx, admin.ID)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Admin)
	assert.NotNil(t, resp.Permissions)
	assert.Contains(t, resp.Permissions, "device:read")
}

func TestAdminAuthService_ChangePassword_Success(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	admin := createTestAdmin(t, db, "changepwadmin", "oldpassword")

	// 修改密码
	err := service.ChangePassword(ctx, admin.ID, &ChangePasswordRequest{
		OldPassword: "oldpassword",
		NewPassword: "newpassword123",
	})
	require.NoError(t, err)

	// 验证新密码可以登录
	_, err = service.Login(ctx, &LoginRequest{
		Username: "changepwadmin",
		Password: "newpassword123",
		IP:       "127.0.0.1",
	})
	assert.NoError(t, err)

	// 验证旧密码不能登录
	_, err = service.Login(ctx, &LoginRequest{
		Username: "changepwadmin",
		Password: "oldpassword",
		IP:       "127.0.0.1",
	})
	assert.Error(t, err)
}

func TestAdminAuthService_ChangePassword_InvalidOldPassword(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	admin := createTestAdmin(t, db, "invalidoldpw", "correctpassword")

	// 使用错误的原密码
	err := service.ChangePassword(ctx, admin.ID, &ChangePasswordRequest{
		OldPassword: "wrongpassword",
		NewPassword: "newpassword123",
	})

	assert.Error(t, err)
	assert.Equal(t, ErrOldPasswordInvalid, err)
}

func TestAdminAuthService_ChangePassword_AdminNotFound(t *testing.T) {
	service, _ := setupAdminAuthService(t)
	ctx := context.Background()

	// 修改不存在管理员的密码
	err := service.ChangePassword(ctx, 99999, &ChangePasswordRequest{
		OldPassword: "oldpassword",
		NewPassword: "newpassword123",
	})

	assert.Error(t, err)
	assert.Equal(t, ErrAdminNotFound, err)
}

func TestAdminAuthService_RefreshToken(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	createTestAdmin(t, db, "refreshadmin", "password123")

	// 登录获取 token
	loginResp, err := service.Login(ctx, &LoginRequest{
		Username: "refreshadmin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)

	// 刷新 token
	newTokenPair, err := service.RefreshToken(ctx, loginResp.TokenPair.RefreshToken)

	require.NoError(t, err)
	assert.NotNil(t, newTokenPair)
	assert.NotEmpty(t, newTokenPair.AccessToken)
	assert.NotEmpty(t, newTokenPair.RefreshToken)
}

func TestAdminAuthService_RefreshToken_InvalidToken(t *testing.T) {
	service, _ := setupAdminAuthService(t)
	ctx := context.Background()

	// 使用无效的 refresh token
	_, err := service.RefreshToken(ctx, "invalid-token")

	assert.Error(t, err)
}

func TestAdminAuthService_ValidateAdminToken(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	createTestAdmin(t, db, "validateadmin", "password123")

	// 登录获取 token
	loginResp, err := service.Login(ctx, &LoginRequest{
		Username: "validateadmin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)

	// 验证 token
	claims, err := service.ValidateAdminToken(ctx, loginResp.TokenPair.AccessToken)

	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, jwt.UserTypeAdmin, claims.UserType)
}

func TestAdminAuthService_ValidateAdminToken_InvalidToken(t *testing.T) {
	service, _ := setupAdminAuthService(t)
	ctx := context.Background()

	// 验证无效的 token
	_, err := service.ValidateAdminToken(ctx, "invalid-token")

	assert.Error(t, err)
}

func TestAdminAuthService_ValidateAdminToken_DisabledAdmin(t *testing.T) {
	service, db := setupAdminAuthService(t)
	ctx := context.Background()

	// 创建测试管理员
	admin := createTestAdmin(t, db, "disabledvalidate", "password123")

	// 登录获取 token
	loginResp, err := service.Login(ctx, &LoginRequest{
		Username: "disabledvalidate",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)

	// 禁用管理员
	err = db.Model(admin).Update("status", models.AdminStatusDisabled).Error
	require.NoError(t, err)

	// 验证 token（应该失败因为管理员已被禁用）
	_, err = service.ValidateAdminToken(ctx, loginResp.TokenPair.AccessToken)

	assert.Error(t, err)
	assert.Equal(t, ErrAdminDisabled, err)
}

func TestAdminAuthService_toAdminInfo(t *testing.T) {
	service, _ := setupAdminAuthService(t)

	phone := "13800138000"
	email := "test@example.com"
	merchantID := int64(100)

	admin := &models.Admin{
		ID:         1,
		Username:   "testuser",
		Name:       "测试用户",
		Phone:      &phone,
		Email:      &email,
		RoleID:     2,
		MerchantID: &merchantID,
		Role: &models.Role{
			Code: "manager",
			Name: "管理员",
		},
	}

	info := service.toAdminInfo(admin)

	assert.Equal(t, int64(1), info.ID)
	assert.Equal(t, "testuser", info.Username)
	assert.Equal(t, "测试用户", info.Name)
	assert.Equal(t, &phone, info.Phone)
	assert.Equal(t, &email, info.Email)
	assert.Equal(t, int64(2), info.RoleID)
	assert.Equal(t, "manager", info.RoleCode)
	assert.Equal(t, "管理员", info.RoleName)
	assert.Equal(t, &merchantID, info.MerchantID)
}

func TestAdminAuthService_extractPermissions(t *testing.T) {
	service, _ := setupAdminAuthService(t)

	admin := &models.Admin{
		Role: &models.Role{
			Permissions: []models.Permission{
				{Code: "device:read"},
				{Code: "device:write"},
				{Code: "user:read"},
			},
		},
	}

	permissions := service.extractPermissions(admin)

	assert.Len(t, permissions, 3)
	assert.Contains(t, permissions, "device:read")
	assert.Contains(t, permissions, "device:write")
	assert.Contains(t, permissions, "user:read")
}

func TestAdminAuthService_extractPermissions_NoRole(t *testing.T) {
	service, _ := setupAdminAuthService(t)

	admin := &models.Admin{}

	permissions := service.extractPermissions(admin)

	assert.Nil(t, permissions)
}

func TestAdminAuthService_extractPermissions_NoPermissions(t *testing.T) {
	service, _ := setupAdminAuthService(t)

	admin := &models.Admin{
		Role: &models.Role{
			Permissions: []models.Permission{},
		},
	}

	permissions := service.extractPermissions(admin)

	assert.Nil(t, permissions)
}
