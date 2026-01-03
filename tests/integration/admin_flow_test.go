// Package integration 管理端流程集成测试
package integration

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
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// setupAdminIntegrationDB 创建管理端集成测试数据库
func setupAdminIntegrationDB(t *testing.T) *gorm.DB {
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
		&models.Venue{},
		&models.Device{},
		&models.DeviceLog{},
		&models.DeviceMaintenance{},
	)
	require.NoError(t, err)

	return db
}

// setupAdminTestEnvironment 设置管理端测试环境
func setupAdminTestEnvironment(t *testing.T, db *gorm.DB) (
	*adminService.AdminAuthService,
	*adminService.DeviceAdminService,
	*adminService.VenueAdminService,
	*adminService.MerchantAdminService,
	*jwt.Manager,
) {
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-admin-integration",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	adminRepo := repository.NewAdminRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	deviceLogRepo := repository.NewDeviceLogRepository(db)
	deviceMaintenanceRepo := repository.NewDeviceMaintenanceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	merchantRepo := repository.NewMerchantRepository(db)

	authService := adminService.NewAdminAuthService(adminRepo, jwtManager)
	deviceService := adminService.NewDeviceAdminService(deviceRepo, deviceLogRepo, deviceMaintenanceRepo, venueRepo, nil)
	venueService := adminService.NewVenueAdminService(venueRepo, merchantRepo, deviceRepo)
	merchantService := adminService.NewMerchantAdminService(merchantRepo, nil)

	return authService, deviceService, venueService, merchantService, jwtManager
}

// createIntegrationTestAdmin 创建集成测试管理员
func createIntegrationTestAdmin(t *testing.T, db *gorm.DB, username, password string, permissions []string) *models.Admin {
	// 创建角色
	role := &models.Role{
		Code:     "integration_role_" + username,
		Name:     "集成测试角色",
		IsSystem: false,
	}
	err := db.Create(role).Error
	require.NoError(t, err)

	// 创建权限并关联角色
	for _, permCode := range permissions {
		permission := &models.Permission{
			Code: permCode,
			Name: permCode,
			Type: models.PermissionTypeAPI,
		}
		db.Create(permission)

		rolePermission := &models.RolePermission{
			RoleID:       role.ID,
			PermissionID: permission.ID,
		}
		db.Create(rolePermission)
	}

	// 创建管理员
	passwordHash, err := crypto.HashPassword(password)
	require.NoError(t, err)

	admin := &models.Admin{
		Username:     username,
		PasswordHash: passwordHash,
		Name:         "集成测试管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	err = db.Create(admin).Error
	require.NoError(t, err)

	return admin
}

// TestAdminFlow_LoginAndManageDevices 测试管理员登录并管理设备的完整流程
func TestAdminFlow_LoginAndManageDevices(t *testing.T) {
	db := setupAdminIntegrationDB(t)
	authService, deviceService, venueService, merchantService, _ := setupAdminTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建管理员
	permissions := []string{"device:read", "device:write", "venue:read", "venue:write", "merchant:read", "merchant:write"}
	createIntegrationTestAdmin(t, db, "device_manager", "password123", permissions)

	// 2. 管理员登录
	loginResp, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "device_manager",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, loginResp.TokenPair.AccessToken)
	assert.Contains(t, loginResp.Permissions, "device:read")
	assert.Contains(t, loginResp.Permissions, "device:write")

	adminID := loginResp.Admin.ID

	// 3. 创建商户
	merchant, err := merchantService.CreateMerchant(ctx, &adminService.CreateMerchantRequest{
		Name:           "测试商户",
		ContactName:    "联系人",
		ContactPhone:   "13800138000",
		CommissionRate: 0.15,
		SettlementType: "monthly",
	})
	require.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, "测试商户", merchant.Name)

	// 4. 创建场地
	venue, err := venueService.CreateVenue(ctx, &adminService.CreateVenueRequest{
		MerchantID: merchant.ID,
		Name:       "测试场地",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园路1号",
	})
	require.NoError(t, err)
	assert.NotNil(t, venue)
	assert.Equal(t, "测试场地", venue.Name)

	// 5. 创建设备
	device, err := deviceService.CreateDevice(ctx, &adminService.CreateDeviceRequest{
		DeviceNo:    "DEV_INTEGRATION_001",
		Name:        "集成测试设备",
		Type:        "standard",
		VenueID:     venue.ID,
		ProductName: "测试产品",
		SlotCount:   10,
		NetworkType: "WiFi",
	}, adminID)
	require.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, "DEV_INTEGRATION_001", device.DeviceNo)
	assert.Equal(t, 10, device.SlotCount)
	assert.Equal(t, 10, device.AvailableSlots)

	// 6. 获取设备详情
	deviceInfo, err := deviceService.GetDevice(ctx, device.ID)
	require.NoError(t, err)
	assert.Equal(t, "集成测试设备", deviceInfo.Name)
	assert.Equal(t, "测试场地", deviceInfo.VenueName)

	// 7. 更新设备
	err = deviceService.UpdateDevice(ctx, device.ID, &adminService.UpdateDeviceRequest{
		Name:        "更新后的设备",
		Type:        "premium",
		VenueID:     venue.ID,
		ProductName: "更新后的产品",
		SlotCount:   20,
		NetworkType: "4G",
	})
	require.NoError(t, err)

	// 验证更新
	var updatedDevice models.Device
	db.First(&updatedDevice, device.ID)
	assert.Equal(t, "更新后的设备", updatedDevice.Name)
	assert.Equal(t, "premium", updatedDevice.Type)
	assert.Equal(t, 20, updatedDevice.SlotCount)

	// 8. 获取设备列表
	devices, total, err := deviceService.ListDevices(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, devices, 1)

	// 9. 获取设备统计
	stats, err := deviceService.GetDeviceStatistics(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Total)
	assert.Equal(t, int64(1), stats.Online)
	assert.Equal(t, int64(0), stats.Offline)
}

// TestAdminFlow_DeviceMaintenance 测试设备维护完整流程
func TestAdminFlow_DeviceMaintenance(t *testing.T) {
	db := setupAdminIntegrationDB(t)
	authService, deviceService, venueService, merchantService, _ := setupAdminTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建管理员并登录
	permissions := []string{"device:read", "device:write", "device:maintenance"}
	createIntegrationTestAdmin(t, db, "maintenance_admin", "password123", permissions)

	loginResp, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "maintenance_admin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	adminID := loginResp.Admin.ID

	// 2. 创建商户和场地
	merchant, _ := merchantService.CreateMerchant(ctx, &adminService.CreateMerchantRequest{
		Name:           "维护测试商户",
		ContactName:    "联系人",
		ContactPhone:   "13900139000",
		CommissionRate: 0.15,
		SettlementType: "monthly",
	})

	venue, _ := venueService.CreateVenue(ctx, &adminService.CreateVenueRequest{
		MerchantID: merchant.ID,
		Name:       "维护测试场地",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "测试地址",
	})

	// 3. 创建设备
	device, err := deviceService.CreateDevice(ctx, &adminService.CreateDeviceRequest{
		DeviceNo:    "DEV_MAINT_001",
		Name:        "待维护设备",
		Type:        "standard",
		VenueID:     venue.ID,
		ProductName: "测试产品",
		SlotCount:   5,
		NetworkType: "WiFi",
	}, adminID)
	require.NoError(t, err)

	// 验证设备初始状态
	assert.Equal(t, int8(models.DeviceStatusActive), device.Status)

	// 4. 创建维护记录
	maintenance, err := deviceService.CreateMaintenance(ctx, &adminService.CreateMaintenanceRequest{
		DeviceID:    device.ID,
		Type:        "repair",
		Description: "更换显示屏",
		Cost:        500.0,
	}, adminID)
	require.NoError(t, err)
	assert.NotNil(t, maintenance)
	assert.Equal(t, int8(models.MaintenanceStatusInProgress), maintenance.Status)

	// 验证设备状态已变为维护中
	var deviceAfterMaintStart models.Device
	db.First(&deviceAfterMaintStart, device.ID)
	assert.Equal(t, int8(models.DeviceStatusMaintenance), deviceAfterMaintStart.Status)

	// 5. 获取维护记录列表
	maintenances, total, err := deviceService.GetMaintenanceRecords(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, maintenances, 1)

	// 6. 完成维护
	err = deviceService.CompleteMaintenance(ctx, maintenance.ID, &adminService.CompleteMaintenanceRequest{
		Cost: 550.0, // 更新最终费用
	}, adminID)
	require.NoError(t, err)

	// 验证维护记录状态
	var completedMaintenance models.DeviceMaintenance
	db.First(&completedMaintenance, maintenance.ID)
	assert.Equal(t, int8(models.MaintenanceStatusCompleted), completedMaintenance.Status)
	assert.NotNil(t, completedMaintenance.CompletedAt)
	assert.Equal(t, 550.0, completedMaintenance.Cost)

	// 验证设备状态已恢复
	var deviceAfterMaintComplete models.Device
	db.First(&deviceAfterMaintComplete, device.ID)
	assert.Equal(t, int8(models.DeviceStatusActive), deviceAfterMaintComplete.Status)
}

// TestAdminFlow_DeviceStatusManagement 测试设备状态管理流程
func TestAdminFlow_DeviceStatusManagement(t *testing.T) {
	db := setupAdminIntegrationDB(t)
	authService, deviceService, venueService, merchantService, _ := setupAdminTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建管理员并登录
	permissions := []string{"device:read", "device:write", "device:status"}
	createIntegrationTestAdmin(t, db, "status_admin", "password123", permissions)

	loginResp, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "status_admin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	adminID := loginResp.Admin.ID

	// 2. 创建商户、场地和设备
	merchant, _ := merchantService.CreateMerchant(ctx, &adminService.CreateMerchantRequest{
		Name:           "状态测试商户",
		ContactName:    "联系人",
		ContactPhone:   "13900139001",
		CommissionRate: 0.15,
		SettlementType: "monthly",
	})

	venue, _ := venueService.CreateVenue(ctx, &adminService.CreateVenueRequest{
		MerchantID: merchant.ID,
		Name:       "状态测试场地",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "测试地址",
	})

	device, err := deviceService.CreateDevice(ctx, &adminService.CreateDeviceRequest{
		DeviceNo:    "DEV_STATUS_001",
		Name:        "状态测试设备",
		Type:        "standard",
		VenueID:     venue.ID,
		ProductName: "测试产品",
		SlotCount:   5,
		NetworkType: "WiFi",
	}, adminID)
	require.NoError(t, err)

	// 3. 更新设备状态为维护中
	err = deviceService.UpdateDeviceStatus(ctx, device.ID, models.DeviceStatusMaintenance, adminID)
	require.NoError(t, err)

	var deviceMaint models.Device
	db.First(&deviceMaint, device.ID)
	assert.Equal(t, int8(models.DeviceStatusMaintenance), deviceMaint.Status)

	// 4. 更新设备状态为正常
	err = deviceService.UpdateDeviceStatus(ctx, device.ID, models.DeviceStatusActive, adminID)
	require.NoError(t, err)

	var deviceActive models.Device
	db.First(&deviceActive, device.ID)
	assert.Equal(t, int8(models.DeviceStatusActive), deviceActive.Status)

	// 5. 尝试禁用正在使用的设备（应该失败）
	db.Model(&models.Device{}).Where("id = ?", device.ID).Update("rental_status", models.DeviceRentalInUse)

	err = deviceService.UpdateDeviceStatus(ctx, device.ID, models.DeviceStatusDisabled, adminID)
	assert.Error(t, err)
	assert.Equal(t, adminService.ErrDeviceInUse, err)
}

// TestAdminFlow_PasswordChange 测试管理员密码修改流程
func TestAdminFlow_PasswordChange(t *testing.T) {
	db := setupAdminIntegrationDB(t)
	authService, _, _, _, _ := setupAdminTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建管理员
	permissions := []string{"admin:profile"}
	createIntegrationTestAdmin(t, db, "password_admin", "oldpassword", permissions)

	// 2. 使用原密码登录
	loginResp, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "password_admin",
		Password: "oldpassword",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	adminID := loginResp.Admin.ID

	// 3. 修改密码
	err = authService.ChangePassword(ctx, adminID, &adminService.ChangePasswordRequest{
		OldPassword: "oldpassword",
		NewPassword: "newpassword123",
	})
	require.NoError(t, err)

	// 4. 使用新密码登录
	newLoginResp, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "password_admin",
		Password: "newpassword123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, newLoginResp.TokenPair.AccessToken)

	// 5. 使用旧密码登录（应该失败）
	_, err = authService.Login(ctx, &adminService.LoginRequest{
		Username: "password_admin",
		Password: "oldpassword",
		IP:       "127.0.0.1",
	})
	assert.Error(t, err)
	assert.Equal(t, adminService.ErrInvalidPassword, err)
}

// TestAdminFlow_TokenRefresh 测试令牌刷新流程
func TestAdminFlow_TokenRefresh(t *testing.T) {
	db := setupAdminIntegrationDB(t)
	authService, _, _, _, _ := setupAdminTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建管理员并登录
	permissions := []string{"admin:profile"}
	createIntegrationTestAdmin(t, db, "token_admin", "password123", permissions)

	loginResp, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "token_admin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	originalAccessToken := loginResp.TokenPair.AccessToken
	refreshToken := loginResp.TokenPair.RefreshToken

	// 2. 刷新令牌
	newTokenPair, err := authService.RefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newTokenPair.AccessToken)
	assert.NotEmpty(t, newTokenPair.RefreshToken)
	assert.NotEqual(t, originalAccessToken, newTokenPair.AccessToken)

	// 3. 验证新令牌有效
	claims, err := authService.ValidateAdminToken(ctx, newTokenPair.AccessToken)
	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, jwt.UserTypeAdmin, claims.UserType)
}

// TestAdminFlow_MerchantAndVenueManagement 测试商户和场地管理流程
func TestAdminFlow_MerchantAndVenueManagement(t *testing.T) {
	db := setupAdminIntegrationDB(t)
	authService, _, venueService, merchantService, _ := setupAdminTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建管理员并登录
	permissions := []string{"merchant:read", "merchant:write", "venue:read", "venue:write"}
	createIntegrationTestAdmin(t, db, "merchant_admin", "password123", permissions)

	loginResp, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "merchant_admin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	assert.Contains(t, loginResp.Permissions, "merchant:read")

	// 2. 创建商户
	merchant, err := merchantService.CreateMerchant(ctx, &adminService.CreateMerchantRequest{
		Name:           "新商户",
		ContactName:    "张三",
		ContactPhone:   "13800138001",
		CommissionRate: 0.20,
		SettlementType: "weekly",
	})
	require.NoError(t, err)
	assert.Equal(t, "新商户", merchant.Name)
	assert.Equal(t, 0.20, merchant.CommissionRate)

	// 3. 获取商户详情
	merchantInfo, err := merchantService.GetMerchant(ctx, merchant.ID)
	require.NoError(t, err)
	assert.Equal(t, "新商户", merchantInfo.Name)

	// 4. 更新商户
	err = merchantService.UpdateMerchant(ctx, merchant.ID, &adminService.UpdateMerchantRequest{
		Name:           "更新后商户",
		ContactName:    "李四",
		ContactPhone:   "13800138002",
		CommissionRate: 0.25,
		SettlementType: "monthly",
	})
	require.NoError(t, err)

	// 验证更新
	var updatedMerchant models.Merchant
	db.First(&updatedMerchant, merchant.ID)
	assert.Equal(t, "更新后商户", updatedMerchant.Name)
	assert.Equal(t, "李四", updatedMerchant.ContactName)

	// 5. 创建场地
	venue, err := venueService.CreateVenue(ctx, &adminService.CreateVenueRequest{
		MerchantID: merchant.ID,
		Name:       "新场地",
		Type:       "hotel",
		Province:   "北京市",
		City:       "北京市",
		District:   "朝阳区",
		Address:    "建国路1号",
	})
	require.NoError(t, err)
	assert.Equal(t, "新场地", venue.Name)
	assert.Equal(t, "hotel", venue.Type)

	// 6. 获取场地列表
	venues, total, err := venueService.ListVenues(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, venues, 1)

	// 7. 按商户筛选场地
	filters := map[string]interface{}{"merchant_id": merchant.ID}
	venues, total, err = venueService.ListVenues(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 8. 更新场地状态
	err = venueService.UpdateVenueStatus(ctx, venue.ID, models.VenueStatusDisabled)
	require.NoError(t, err)

	var updatedVenue models.Venue
	db.First(&updatedVenue, venue.ID)
	assert.Equal(t, int8(models.VenueStatusDisabled), updatedVenue.Status)

	// 9. 获取商户列表
	merchants, total, err := merchantService.ListMerchants(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, merchants, 1)
}

// TestAdminFlow_DisabledAdminCannotLogin 测试禁用的管理员无法登录
func TestAdminFlow_DisabledAdminCannotLogin(t *testing.T) {
	db := setupAdminIntegrationDB(t)
	authService, _, _, _, _ := setupAdminTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建管理员
	permissions := []string{"admin:profile"}
	admin := createIntegrationTestAdmin(t, db, "disabled_admin", "password123", permissions)

	// 2. 正常登录
	_, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "disabled_admin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)

	// 3. 禁用管理员
	db.Model(admin).Update("status", models.AdminStatusDisabled)

	// 4. 尝试登录（应该失败）
	_, err = authService.Login(ctx, &adminService.LoginRequest{
		Username: "disabled_admin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	assert.Error(t, err)
	assert.Equal(t, adminService.ErrAdminDisabled, err)
}

// TestAdminFlow_MultipleDevicesWithFiltering 测试多设备筛选
func TestAdminFlow_MultipleDevicesWithFiltering(t *testing.T) {
	db := setupAdminIntegrationDB(t)
	authService, deviceService, venueService, merchantService, _ := setupAdminTestEnvironment(t, db)
	ctx := context.Background()

	// 1. 创建管理员并登录
	permissions := []string{"device:read", "device:write"}
	createIntegrationTestAdmin(t, db, "filter_admin", "password123", permissions)

	loginResp, err := authService.Login(ctx, &adminService.LoginRequest{
		Username: "filter_admin",
		Password: "password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)
	adminID := loginResp.Admin.ID

	// 2. 创建商户和场地
	merchant, _ := merchantService.CreateMerchant(ctx, &adminService.CreateMerchantRequest{
		Name:           "筛选测试商户",
		ContactName:    "联系人",
		ContactPhone:   "13900139002",
		CommissionRate: 0.15,
		SettlementType: "monthly",
	})

	venue1, _ := venueService.CreateVenue(ctx, &adminService.CreateVenueRequest{
		MerchantID: merchant.ID,
		Name:       "场地1",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "地址1",
	})

	venue2, _ := venueService.CreateVenue(ctx, &adminService.CreateVenueRequest{
		MerchantID: merchant.ID,
		Name:       "场地2",
		Type:       "hotel",
		Province:   "广东省",
		City:       "广州市",
		District:   "天河区",
		Address:    "地址2",
	})

	// 3. 创建多个设备
	device1, _ := deviceService.CreateDevice(ctx, &adminService.CreateDeviceRequest{
		DeviceNo:    "DEV_FILTER_001",
		Name:        "设备1",
		Type:        "standard",
		VenueID:     venue1.ID,
		ProductName: "产品A",
		SlotCount:   5,
		NetworkType: "WiFi",
	}, adminID)

	device2, _ := deviceService.CreateDevice(ctx, &adminService.CreateDeviceRequest{
		DeviceNo:    "DEV_FILTER_002",
		Name:        "设备2",
		Type:        "premium",
		VenueID:     venue1.ID,
		ProductName: "产品B",
		SlotCount:   10,
		NetworkType: "4G",
	}, adminID)

	device3, _ := deviceService.CreateDevice(ctx, &adminService.CreateDeviceRequest{
		DeviceNo:    "DEV_FILTER_003",
		Name:        "设备3",
		Type:        "standard",
		VenueID:     venue2.ID,
		ProductName: "产品A",
		SlotCount:   5,
		NetworkType: "WiFi",
	}, adminID)

	// 设置设备2离线
	db.Model(device2).Update("online_status", models.DeviceOffline)

	// 4. 按场地筛选
	filters := map[string]interface{}{"venue_id": venue1.ID}
	devices, total, err := deviceService.ListDevices(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, devices, 2)

	// 5. 按在线状态筛选
	filters = map[string]interface{}{"online_status": models.DeviceOnline}
	devices, total, err = deviceService.ListDevices(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total) // device1 和 device3 在线

	// 6. 按设备类型筛选
	filters = map[string]interface{}{"type": "standard"}
	devices, total, err = deviceService.ListDevices(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total) // device1 和 device3 是 standard

	// 7. 获取统计
	stats, err := deviceService.GetDeviceStatistics(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Total)
	assert.Equal(t, int64(2), stats.Online)
	assert.Equal(t, int64(1), stats.Offline)

	_ = device1
	_ = device3
}
