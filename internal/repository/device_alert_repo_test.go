// Package repository 设备告警仓储单元测试
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

func setupDeviceAlertTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.DeviceAlert{})
	require.NoError(t, err)

	return db
}

func TestDeviceAlertRepository_Create(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	alert := &models.DeviceAlert{
		DeviceID: 1,
		Type:     "offline",
		Level:    "high",
		Title:    "设备离线",
		Content:  "设备已离线超过10分钟",
	}

	err := repo.Create(ctx, alert)
	require.NoError(t, err)
	assert.NotZero(t, alert.ID)
}

func TestDeviceAlertRepository_GetByID(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	alert := &models.DeviceAlert{
		DeviceID: 1,
		Type:     "offline",
		Level:    "high",
		Title:    "设备离线",
		Content:  "设备已离线超过10分钟",
	}
	db.Create(alert)

	found, err := repo.GetByID(ctx, alert.ID)
	require.NoError(t, err)
	assert.Equal(t, alert.ID, found.ID)
	assert.Equal(t, "设备离线", found.Title)
}

func TestDeviceAlertRepository_Update(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	alert := &models.DeviceAlert{
		DeviceID: 1,
		Type:     "offline",
		Level:    "high",
		Title:    "设备离线",
		Content:  "设备已离线超过10分钟",
	}
	db.Create(alert)

	alert.Content = "设备已离线超过30分钟"
	err := repo.Update(ctx, alert)
	require.NoError(t, err)

	var found models.DeviceAlert
	db.First(&found, alert.ID)
	assert.Equal(t, "设备已离线超过30分钟", found.Content)
}

func TestDeviceAlertRepository_List(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "offline", Level: "high", Title: "设备1离线", Content: "内容", IsResolved: false,
	})

	db.Create(&models.DeviceAlert{
		DeviceID: 2, Type: "battery_low", Level: "medium", Title: "设备2电量低", Content: "内容", IsResolved: false,
	})

	db.Model(&models.DeviceAlert{}).Create(map[string]interface{}{
		"device_id": 1, "type": "offline", "level": "high", "title": "设备1离线2", "content": "内容", "is_resolved": true,
	})

	// 获取所有告警
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
		"type": "offline",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按级别过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"level": "high",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按解决状态过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"is_resolved": false,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestDeviceAlertRepository_ListByDevice(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "offline", Level: "high", Title: "告警1", Content: "内容",
	})

	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "battery_low", Level: "medium", Title: "告警2", Content: "内容",
	})

	db.Create(&models.DeviceAlert{
		DeviceID: 2, Type: "offline", Level: "high", Title: "告警3", Content: "内容",
	})

	list, total, err := repo.ListByDevice(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestDeviceAlertRepository_ListUnresolved(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "offline", Level: "high", Title: "未解决告警", Content: "内容", IsResolved: false,
	})

	db.Model(&models.DeviceAlert{}).Create(map[string]interface{}{
		"device_id": 1, "type": "battery_low", "level": "medium", "title": "已解决告警", "content": "内容", "is_resolved": true,
	})

	list, total, err := repo.ListUnresolved(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, len(list))
	assert.False(t, list[0].IsResolved)
}

func TestDeviceAlertRepository_CountUnresolved(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "offline", Level: "high", Title: "未解决1", Content: "内容", IsResolved: false,
	})

	db.Create(&models.DeviceAlert{
		DeviceID: 2, Type: "battery_low", Level: "medium", Title: "未解决2", Content: "内容", IsResolved: false,
	})

	db.Model(&models.DeviceAlert{}).Create(map[string]interface{}{
		"device_id": 1, "type": "offline", "level": "high", "title": "已解决", "content": "内容", "is_resolved": true,
	})

	count, err := repo.CountUnresolved(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestDeviceAlertRepository_CountByLevel(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "offline", Level: "high", Title: "高级别未解决", Content: "内容", IsResolved: false,
	})

	db.Create(&models.DeviceAlert{
		DeviceID: 2, Type: "battery_low", Level: "medium", Title: "中级别未解决", Content: "内容", IsResolved: false,
	})

	db.Model(&models.DeviceAlert{}).Create(map[string]interface{}{
		"device_id": 1, "type": "offline", "level": "high", "title": "高级别已解决", "content": "内容", "is_resolved": true,
	})

	count, err := repo.CountByLevel(ctx, "high", false)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = repo.CountByLevel(ctx, "high", true)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestDeviceAlertRepository_CountSince(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	// 创建旧告警
	db.Exec("INSERT INTO device_alerts (device_id, type, level, title, content, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		1, "offline", "high", "旧告警", "内容", time.Now().Add(-2*time.Hour))

	// 创建新告警
	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "battery_low", Level: "medium", Title: "新告警", Content: "内容",
	})

	since := time.Now().Add(-1 * time.Hour)
	count, err := repo.CountSince(ctx, since)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count) // 只统计1小时内的
}

func TestDeviceAlertRepository_Resolve(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	alert := &models.DeviceAlert{
		DeviceID:   1,
		Type:       "offline",
		Level:      "high",
		Title:      "告警",
		Content:    "内容",
		IsResolved: false,
	}
	db.Create(alert)

	err := repo.Resolve(ctx, alert.ID, 100)
	require.NoError(t, err)

	var found models.DeviceAlert
	db.First(&found, alert.ID)
	assert.True(t, found.IsResolved)
	assert.NotNil(t, found.ResolvedBy)
	assert.Equal(t, int64(100), *found.ResolvedBy)
	assert.NotNil(t, found.ResolvedAt)
}

func TestDeviceAlertRepository_BatchResolve(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	alert1 := &models.DeviceAlert{
		DeviceID: 1, Type: "offline", Level: "high", Title: "告警1", Content: "内容", IsResolved: false,
	}
	db.Create(alert1)

	alert2 := &models.DeviceAlert{
		DeviceID: 2, Type: "battery_low", Level: "medium", Title: "告警2", Content: "内容", IsResolved: false,
	}
	db.Create(alert2)

	alert3 := &models.DeviceAlert{
		DeviceID: 3, Type: "offline", Level: "high", Title: "告警3", Content: "内容", IsResolved: false,
	}
	db.Create(alert3)

	err := repo.BatchResolve(ctx, []int64{alert1.ID, alert2.ID}, 100)
	require.NoError(t, err)

	var resolved int64
	db.Model(&models.DeviceAlert{}).Where("is_resolved = ?", true).Count(&resolved)
	assert.Equal(t, int64(2), resolved)
}

func TestDeviceAlertRepository_Delete(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	alert := &models.DeviceAlert{
		DeviceID: 1, Type: "offline", Level: "high", Title: "告警", Content: "内容",
	}
	db.Create(alert)

	err := repo.Delete(ctx, alert.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.DeviceAlert{}).Where("id = ?", alert.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDeviceAlertRepository_DeleteByDevice(t *testing.T) {
	db := setupDeviceAlertTestDB(t)
	repo := NewDeviceAlertRepository(db)
	ctx := context.Background()

	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "offline", Level: "high", Title: "告警1", Content: "内容",
	})

	db.Create(&models.DeviceAlert{
		DeviceID: 1, Type: "battery_low", Level: "medium", Title: "告警2", Content: "内容",
	})

	db.Create(&models.DeviceAlert{
		DeviceID: 2, Type: "offline", Level: "high", Title: "告警3", Content: "内容",
	})

	err := repo.DeleteByDevice(ctx, 1)
	require.NoError(t, err)

	var count int64
	db.Model(&models.DeviceAlert{}).Where("device_id = ?", 1).Count(&count)
	assert.Equal(t, int64(0), count)

	db.Model(&models.DeviceAlert{}).Count(&count)
	assert.Equal(t, int64(1), count) // 还剩下设备2的告警
}
