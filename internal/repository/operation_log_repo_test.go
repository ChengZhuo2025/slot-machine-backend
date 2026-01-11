// Package repository 操作日志仓储单元测试
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupOperationLogTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.OperationLog{}, &models.Admin{})
	require.NoError(t, err)

	return db
}

func TestOperationLogRepository_Create(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	log := &models.OperationLog{
		AdminID: 1,
		Module:  "user",
		Action:  "create",
		IP:      "192.168.1.1",
	}

	err := repo.Create(ctx, log)
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
}

func TestOperationLogRepository_GetByID(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	admin := &models.Admin{
		Username:     "admin",
		PasswordHash: "hash",
		Name:         "管理员",
		RoleID:       1,
	}
	db.Create(admin)

	log := &models.OperationLog{
		AdminID: admin.ID,
		Module:  "user",
		Action:  "create",
		IP:      "192.168.1.1",
	}
	db.Create(log)

	found, err := repo.GetByID(ctx, log.ID)
	require.NoError(t, err)
	assert.Equal(t, log.ID, found.ID)
	assert.NotNil(t, found.Admin)
	assert.Equal(t, admin.ID, found.Admin.ID)
}

func TestOperationLogRepository_List(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	db.Create(&models.OperationLog{
		AdminID: 1, Module: "user", Action: "create", IP: "192.168.1.1",
	})

	db.Create(&models.OperationLog{
		AdminID: 1, Module: "product", Action: "update", IP: "192.168.1.2",
	})

	db.Create(&models.OperationLog{
		AdminID: 2, Module: "user", Action: "delete", IP: "192.168.1.1",
	})

	// 获取所有日志
	_, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按管理员过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"admin_id": int64(1),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按模块过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"module": "user",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按操作过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"action": "create",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按IP过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"ip": "192.168.1.1",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestOperationLogRepository_ListByAdmin(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	db.Create(&models.OperationLog{
		AdminID: 1, Module: "user", Action: "create", IP: "192.168.1.1",
	})

	db.Create(&models.OperationLog{
		AdminID: 1, Module: "product", Action: "update", IP: "192.168.1.1",
	})

	db.Create(&models.OperationLog{
		AdminID: 2, Module: "user", Action: "delete", IP: "192.168.1.1",
	})

	_, total, err := repo.ListByAdmin(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestOperationLogRepository_ListByTarget(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	targetType := "user"
	targetID := int64(123)

	db.Create(&models.OperationLog{
		AdminID: 1, Module: "user", Action: "update", TargetType: &targetType, TargetID: &targetID, IP: "192.168.1.1",
	})

	otherTargetID := int64(456)
	db.Create(&models.OperationLog{
		AdminID: 1, Module: "user", Action: "delete", TargetType: &targetType, TargetID: &otherTargetID, IP: "192.168.1.1",
	})

	_, total, err := repo.ListByTarget(ctx, "user", 123, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestOperationLogRepository_CountByModule(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	since := time.Now().Add(-1 * time.Hour)

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "create", "192.168.1.1", time.Now().Add(-30*time.Minute))

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "update", "192.168.1.1", time.Now())

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "delete", "192.168.1.1", time.Now().Add(-2*time.Hour))

	count, err := repo.CountByModule(ctx, "user", since)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count) // 只统计1小时内的
}

func TestOperationLogRepository_CountByAdmin(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	since := time.Now().Add(-1 * time.Hour)

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "create", "192.168.1.1", time.Now().Add(-30*time.Minute))

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "product", "update", "192.168.1.1", time.Now())

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		2, "user", "delete", "192.168.1.1", time.Now())

	count, err := repo.CountByAdmin(ctx, 1, since)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestOperationLogRepository_GetModuleStats(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	since := time.Now().Add(-1 * time.Hour)

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "create", "192.168.1.1", time.Now())

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "update", "192.168.1.1", time.Now())

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "product", "create", "192.168.1.1", time.Now())

	stats, err := repo.GetModuleStats(ctx, since)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats["user"])
	assert.Equal(t, int64(1), stats["product"])
}

func TestOperationLogRepository_GetActionStats(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	since := time.Now().Add(-1 * time.Hour)

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "create", "192.168.1.1", time.Now())

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "product", "create", "192.168.1.1", time.Now())

	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "update", "192.168.1.1", time.Now())

	stats, err := repo.GetActionStats(ctx, since)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats["create"])
	assert.Equal(t, int64(1), stats["update"])
}

func TestOperationLogRepository_DeleteBefore(t *testing.T) {
	db := setupOperationLogTestDB(t)
	repo := NewOperationLogRepository(db)
	ctx := context.Background()

	// 创建旧日志
	db.Exec("INSERT INTO operation_logs (admin_id, module, action, ip, created_at) VALUES (?, ?, ?, ?, ?)",
		1, "user", "create", "192.168.1.1", time.Now().Add(-10*24*time.Hour))

	// 创建新日志
	db.Create(&models.OperationLog{
		AdminID: 1, Module: "user", Action: "update", IP: "192.168.1.1",
	})

	before := time.Now().Add(-7 * 24 * time.Hour)
	affected, err := repo.DeleteBefore(ctx, before)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	var count int64
	db.Model(&models.OperationLog{}).Count(&count)
	assert.Equal(t, int64(1), count) // 只剩新日志
}
