// Package repository 设备日志和维护记录仓储单元测试
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

func setupDeviceLogTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.DeviceLog{}, &models.Device{}, &models.DeviceMaintenance{})
	require.NoError(t, err)

	return db
}

// DeviceLogRepository 测试

func TestDeviceLogRepository_Create(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	log := &models.DeviceLog{
		DeviceID: 1,
		Type:     models.DeviceLogTypeOnline,
	}

	err := repo.Create(ctx, log)
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
}

func TestDeviceLogRepository_CreateBatch(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	logs := []*models.DeviceLog{
		{DeviceID: 1, Type: models.DeviceLogTypeOnline},
		{DeviceID: 1, Type: models.DeviceLogTypeHeartbeat},
	}

	err := repo.CreateBatch(ctx, logs)
	require.NoError(t, err)

	var count int64
	db.Model(&models.DeviceLog{}).Where("device_id = ?", 1).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestDeviceLogRepository_GetByID(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	log := &models.DeviceLog{
		DeviceID: 1,
		Type:     models.DeviceLogTypeOnline,
	}
	db.Create(log)

	found, err := repo.GetByID(ctx, log.ID)
	require.NoError(t, err)
	assert.Equal(t, log.ID, found.ID)
}

func TestDeviceLogRepository_List(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceLog{DeviceID: 1, Type: models.DeviceLogTypeOnline})
	db.Create(&models.DeviceLog{DeviceID: 1, Type: models.DeviceLogTypeHeartbeat})
	db.Create(&models.DeviceLog{DeviceID: 2, Type: models.DeviceLogTypeOffline})

	list, total, err := repo.List(ctx, 1, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))

	// 按类型过滤
	list, total, err = repo.List(ctx, 1, 0, 10, map[string]interface{}{
		"type": models.DeviceLogTypeOnline,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestDeviceLogRepository_ListByDeviceIDs(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceLog{DeviceID: 1, Type: models.DeviceLogTypeOnline})
	db.Create(&models.DeviceLog{DeviceID: 2, Type: models.DeviceLogTypeOnline})
	db.Create(&models.DeviceLog{DeviceID: 3, Type: models.DeviceLogTypeOnline})

	list, total, err := repo.ListByDeviceIDs(ctx, []int64{1, 2}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestDeviceLogRepository_GetLatestByDeviceID(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	// 创建旧日志
	db.Exec("INSERT INTO device_logs (device_id, type, created_at) VALUES (?, ?, ?)",
		1, models.DeviceLogTypeOnline, time.Now().Add(-1*time.Hour))

	// 创建新日志
	db.Create(&models.DeviceLog{
		DeviceID: 1,
		Type:     models.DeviceLogTypeHeartbeat,
	})

	latest, err := repo.GetLatestByDeviceID(ctx, 1, "")
	require.NoError(t, err)
	assert.Equal(t, models.DeviceLogTypeHeartbeat, latest.Type)

	// 按类型获取
	db.Exec("INSERT INTO device_logs (device_id, type, created_at) VALUES (?, ?, ?)",
		1, models.DeviceLogTypeOnline, time.Now())

	latest, err = repo.GetLatestByDeviceID(ctx, 1, models.DeviceLogTypeOnline)
	require.NoError(t, err)
	assert.Equal(t, models.DeviceLogTypeOnline, latest.Type)
}

func TestDeviceLogRepository_CountByDeviceID(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceLog{DeviceID: 1, Type: models.DeviceLogTypeOnline})
	db.Create(&models.DeviceLog{DeviceID: 1, Type: models.DeviceLogTypeHeartbeat})
	db.Create(&models.DeviceLog{DeviceID: 1, Type: models.DeviceLogTypeHeartbeat})

	count, err := repo.CountByDeviceID(ctx, 1, "")
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	count, err = repo.CountByDeviceID(ctx, 1, models.DeviceLogTypeHeartbeat)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestDeviceLogRepository_CountByTypeInPeriod(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	now := time.Now()
	startTime := now.Add(-2 * time.Hour)
	endTime := now.Add(1 * time.Hour)

	// 创建日志
	db.Exec("INSERT INTO device_logs (device_id, type, created_at) VALUES (?, ?, ?)",
		1, models.DeviceLogTypeOnline, now.Add(-1*time.Hour))
	db.Exec("INSERT INTO device_logs (device_id, type, created_at) VALUES (?, ?, ?)",
		1, models.DeviceLogTypeHeartbeat, now.Add(-30*time.Minute))
	db.Exec("INSERT INTO device_logs (device_id, type, created_at) VALUES (?, ?, ?)",
		1, models.DeviceLogTypeHeartbeat, now)

	countMap, err := repo.CountByTypeInPeriod(ctx, 1, startTime, endTime)
	require.NoError(t, err)
	assert.Equal(t, int64(1), countMap[models.DeviceLogTypeOnline])
	assert.Equal(t, int64(2), countMap[models.DeviceLogTypeHeartbeat])
}

func TestDeviceLogRepository_DeleteOldLogs(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceLogRepository(db)
	ctx := context.Background()

	// 创建旧日志
	db.Exec("INSERT INTO device_logs (device_id, type, created_at) VALUES (?, ?, ?)",
		1, models.DeviceLogTypeOnline, time.Now().Add(-10*24*time.Hour))

	// 创建新日志
	db.Create(&models.DeviceLog{DeviceID: 1, Type: models.DeviceLogTypeHeartbeat})

	before := time.Now().Add(-7 * 24 * time.Hour)
	affected, err := repo.DeleteOldLogs(ctx, before)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	var count int64
	db.Model(&models.DeviceLog{}).Count(&count)
	assert.Equal(t, int64(1), count) // 只剩新日志
}

// DeviceMaintenanceRepository 测试

func TestDeviceMaintenanceRepository_Create(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	maintenance := &models.DeviceMaintenance{
		DeviceID:    1,
		Type:        "repair",
		Description: "维修描述",
		OperatorID:  100,
		StartedAt:   time.Now(),
	}

	err := repo.Create(ctx, maintenance)
	require.NoError(t, err)
	assert.NotZero(t, maintenance.ID)
}

func TestDeviceMaintenanceRepository_GetByID(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	maintenance := &models.DeviceMaintenance{
		DeviceID:    1,
		Type:        "repair",
		Description: "维修描述",
		OperatorID:  100,
		StartedAt:   time.Now(),
	}
	db.Create(maintenance)

	found, err := repo.GetByID(ctx, maintenance.ID)
	require.NoError(t, err)
	assert.Equal(t, maintenance.ID, found.ID)
}

func TestDeviceMaintenanceRepository_Update(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	maintenance := &models.DeviceMaintenance{
		DeviceID:    1,
		Type:        "repair",
		Description: "原描述",
		Cost:        100.0,
		OperatorID:  100,
		StartedAt:   time.Now(),
	}
	db.Create(maintenance)

	maintenance.Description = "新描述"
	maintenance.Cost = 200.0
	err := repo.Update(ctx, maintenance)
	require.NoError(t, err)

	var found models.DeviceMaintenance
	db.First(&found, maintenance.ID)
	assert.Equal(t, "新描述", found.Description)
	assert.Equal(t, 200.0, found.Cost)
}

func TestDeviceMaintenanceRepository_UpdateStatus(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	maintenance := &models.DeviceMaintenance{
		DeviceID:    1,
		Type:        "repair",
		Description: "维修",
		OperatorID:  100,
		Status:      models.MaintenanceStatusInProgress,
		StartedAt:   time.Now(),
	}
	db.Create(maintenance)

	now := time.Now()
	err := repo.UpdateStatus(ctx, maintenance.ID, models.MaintenanceStatusCompleted, &now)
	require.NoError(t, err)

	var found models.DeviceMaintenance
	db.First(&found, maintenance.ID)
	assert.Equal(t, int8(models.MaintenanceStatusCompleted), found.Status)
	assert.NotNil(t, found.CompletedAt)
}

func TestDeviceMaintenanceRepository_List(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceMaintenance{
		DeviceID: 1, Type: "repair", Description: "维修1", OperatorID: 100, Status: models.MaintenanceStatusInProgress, StartedAt: time.Now(),
	})

	db.Create(&models.DeviceMaintenance{
		DeviceID: 2, Type: "inspect", Description: "检查1", OperatorID: 100, Status: models.MaintenanceStatusCompleted, StartedAt: time.Now(),
	})

	db.Model(&models.DeviceMaintenance{}).Create(map[string]interface{}{
		"device_id": 1, "type": "repair", "description": "维修2", "operator_id": 101,
		"status": models.MaintenanceStatusCompleted, "started_at": time.Now(),
	})

	// 获取所有
	_, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按设备过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"device_id": int64(1),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按类型过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"type": "repair",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按状态过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": int8(models.MaintenanceStatusInProgress),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按操作员过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"operator_id": int64(100),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestDeviceMaintenanceRepository_ListByDeviceID(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceMaintenance{
		DeviceID: 1, Type: "repair", Description: "维修", OperatorID: 100, StartedAt: time.Now(),
	})

	db.Create(&models.DeviceMaintenance{
		DeviceID: 1, Type: "inspect", Description: "检查", OperatorID: 100, StartedAt: time.Now(),
	})

	db.Create(&models.DeviceMaintenance{
		DeviceID: 2, Type: "repair", Description: "维修", OperatorID: 100, StartedAt: time.Now(),
	})

	list, total, err := repo.ListByDeviceID(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestDeviceMaintenanceRepository_GetInProgressByDeviceID(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceMaintenance{
		DeviceID: 1, Type: "repair", Description: "进行中", OperatorID: 100, Status: models.MaintenanceStatusInProgress, StartedAt: time.Now(),
	})

	db.Model(&models.DeviceMaintenance{}).Create(map[string]interface{}{
		"device_id": 1, "type": "inspect", "description": "已完成", "operator_id": 100,
		"status": models.MaintenanceStatusCompleted, "started_at": time.Now(),
	})

	found, err := repo.GetInProgressByDeviceID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int8(models.MaintenanceStatusInProgress), found.Status)
	assert.Equal(t, "进行中", found.Description)
}

func TestDeviceMaintenanceRepository_CountByDeviceID(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceMaintenance{
		DeviceID: 1, Type: "repair", Description: "维修1", OperatorID: 100, StartedAt: time.Now(),
	})

	db.Create(&models.DeviceMaintenance{
		DeviceID: 1, Type: "inspect", Description: "检查1", OperatorID: 100, StartedAt: time.Now(),
	})

	count, err := repo.CountByDeviceID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestDeviceMaintenanceRepository_SumCostByDeviceID(t *testing.T) {
	db := setupDeviceLogTestDB(t)
	repo := NewDeviceMaintenanceRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceMaintenance{
		DeviceID: 1, Type: "repair", Description: "维修1", Cost: 100.0, OperatorID: 100, StartedAt: time.Now(),
	})

	db.Create(&models.DeviceMaintenance{
		DeviceID: 1, Type: "repair", Description: "维修2", Cost: 200.0, OperatorID: 100, StartedAt: time.Now(),
	})

	db.Create(&models.DeviceMaintenance{
		DeviceID: 2, Type: "repair", Description: "维修3", Cost: 150.0, OperatorID: 100, StartedAt: time.Now(),
	})

	sum, err := repo.SumCostByDeviceID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 300.0, sum)
}
