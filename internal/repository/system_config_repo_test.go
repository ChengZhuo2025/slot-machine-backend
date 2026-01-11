// Package repository 系统配置仓储单元测试
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

func setupSystemConfigTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.SystemConfig{})
	require.NoError(t, err)

	return db
}

func TestSystemConfigRepository_Create(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	config := &models.SystemConfig{
		Group:    "general",
		Key:      "site_name",
		Value:    "智能储物柜",
		Type:     models.ConfigTypeString,
		IsPublic: true,
	}

	err := repo.Create(ctx, config)
	require.NoError(t, err)
	assert.NotZero(t, config.ID)
}

func TestSystemConfigRepository_GetByID(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	config := &models.SystemConfig{
		Group:    "general",
		Key:      "site_name",
		Value:    "智能储物柜",
		Type:     models.ConfigTypeString,
		IsPublic: true,
	}
	db.Create(config)

	found, err := repo.GetByID(ctx, config.ID)
	require.NoError(t, err)
	assert.Equal(t, config.ID, found.ID)
	assert.Equal(t, "general", found.Group)
}

func TestSystemConfigRepository_GetByGroupAndKey(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	db.Create(&models.SystemConfig{
		Group:    "payment",
		Key:      "alipay_appid",
		Value:    "test_appid",
		Type:     models.ConfigTypeString,
		IsPublic: true,
	})

	found, err := repo.GetByGroupAndKey(ctx, "payment", "alipay_appid")
	require.NoError(t, err)
	assert.Equal(t, "payment", found.Group)
	assert.Equal(t, "alipay_appid", found.Key)
	assert.Equal(t, "test_appid", found.Value)
}

func TestSystemConfigRepository_GetByGroup(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	db.Create(&models.SystemConfig{
		Group: "payment", Key: "alipay_appid", Value: "app1", Type: models.ConfigTypeString, IsPublic: true,
	})

	db.Create(&models.SystemConfig{
		Group: "payment", Key: "wechat_appid", Value: "app2", Type: models.ConfigTypeString, IsPublic: true,
	})

	db.Create(&models.SystemConfig{
		Group: "sms", Key: "api_key", Value: "key1", Type: models.ConfigTypeString, IsPublic: true,
	})

	configs, err := repo.GetByGroup(ctx, "payment")
	require.NoError(t, err)
	assert.Equal(t, 2, len(configs))
}

func TestSystemConfigRepository_Update(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	config := &models.SystemConfig{
		Group:    "general",
		Key:      "site_name",
		Value:    "旧名称",
		Type:     models.ConfigTypeString,
		IsPublic: true,
	}
	db.Create(config)

	config.Value = "新名称"
	err := repo.Update(ctx, config)
	require.NoError(t, err)

	var found models.SystemConfig
	db.First(&found, config.ID)
	assert.Equal(t, "新名称", found.Value)
}

func TestSystemConfigRepository_UpdateValue(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	config := &models.SystemConfig{
		Group:    "general",
		Key:      "site_name",
		Value:    "旧值",
		Type:     models.ConfigTypeString,
		IsPublic: true,
	}
	db.Create(config)

	err := repo.UpdateValue(ctx, config.ID, "新值")
	require.NoError(t, err)

	var found models.SystemConfig
	db.First(&found, config.ID)
	assert.Equal(t, "新值", found.Value)
}

func TestSystemConfigRepository_Delete(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	config := &models.SystemConfig{
		Group:    "general",
		Key:      "test_key",
		Value:    "test_value",
		Type:     models.ConfigTypeString,
		IsPublic: true,
	}
	db.Create(config)

	err := repo.Delete(ctx, config.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.SystemConfig{}).Where("id = ?", config.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestSystemConfigRepository_List(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	db.Create(&models.SystemConfig{
		Group: "payment", Key: "alipay_key", Value: "v1", Type: models.ConfigTypeString, IsPublic: true,
	})

	db.Model(&models.SystemConfig{}).Create(map[string]interface{}{
		"group": "payment", "key": "wechat_secret", "value": "v2", "type": models.ConfigTypeString, "is_public": false,
	})

	db.Create(&models.SystemConfig{
		Group: "sms", Key: "api_key", Value: "v3", Type: models.ConfigTypeString, IsPublic: true,
	})

	// 获取所有配置
	_, total, err := repo.List(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按分组过滤
	filters := &SystemConfigListFilters{Group: "payment"}
	_, total, err = repo.List(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按关键字过滤
	filters = &SystemConfigListFilters{Keyword: "wechat"}
	_, total, err = repo.List(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按公开状态过滤
	isPublic := true
	filters = &SystemConfigListFilters{IsPublic: &isPublic}
	_, total, err = repo.List(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	isPublic = false
	filters = &SystemConfigListFilters{IsPublic: &isPublic}
	_, total, err = repo.List(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestSystemConfigRepository_GetPublicConfigs(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	db.Create(&models.SystemConfig{
		Group: "general", Key: "site_name", Value: "智能储物柜", Type: models.ConfigTypeString, IsPublic: true,
	})

	db.Model(&models.SystemConfig{}).Create(map[string]interface{}{
		"group": "payment", "key": "secret_key", "value": "secret", "type": models.ConfigTypeString, "is_public": false,
	})

	configs, err := repo.GetPublicConfigs(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(configs))
	assert.Equal(t, "site_name", configs[0].Key)
}

func TestSystemConfigRepository_GetAllGroups(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	db.Create(&models.SystemConfig{
		Group: "payment", Key: "key1", Value: "v1", Type: models.ConfigTypeString, IsPublic: true,
	})

	db.Create(&models.SystemConfig{
		Group: "sms", Key: "key2", Value: "v2", Type: models.ConfigTypeString, IsPublic: true,
	})

	db.Create(&models.SystemConfig{
		Group: "payment", Key: "key3", Value: "v3", Type: models.ConfigTypeString, IsPublic: true,
	})

	groups, err := repo.GetAllGroups(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(groups))
	assert.Contains(t, groups, "payment")
	assert.Contains(t, groups, "sms")
}

func TestSystemConfigRepository_BatchUpsert(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 先创建一条记录
	db.Create(&models.SystemConfig{
		Group: "payment", Key: "alipay_appid", Value: "old_value", Type: models.ConfigTypeString, IsPublic: true,
	})

	// 批量更新和创建
	configs := []*models.SystemConfig{
		{
			Group: "payment", Key: "alipay_appid", Value: "new_value", Type: models.ConfigTypeString, IsPublic: true,
		},
		{
			Group: "payment", Key: "wechat_appid", Value: "wx123", Type: models.ConfigTypeString, IsPublic: true,
		},
	}

	err := repo.BatchUpsert(ctx, configs)
	require.NoError(t, err)

	// 验证更新
	found, err := repo.GetByGroupAndKey(ctx, "payment", "alipay_appid")
	require.NoError(t, err)
	assert.Equal(t, "new_value", found.Value)

	// 验证创建
	found, err = repo.GetByGroupAndKey(ctx, "payment", "wechat_appid")
	require.NoError(t, err)
	assert.Equal(t, "wx123", found.Value)
}

func TestSystemConfigRepository_ExistsByGroupAndKey(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	db.Create(&models.SystemConfig{
		Group: "payment", Key: "alipay_appid", Value: "app1", Type: models.ConfigTypeString, IsPublic: true,
	})

	exists, err := repo.ExistsByGroupAndKey(ctx, "payment", "alipay_appid")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByGroupAndKey(ctx, "payment", "not_exist")
	require.NoError(t, err)
	assert.False(t, exists)
}
