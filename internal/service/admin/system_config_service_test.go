// Package admin 系统配置服务单元测试
package admin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupSystemConfigTestDB 创建系统配置测试数据库
func setupSystemConfigTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.SystemConfig{})
	require.NoError(t, err)

	return db
}

// setupSystemConfigService 创建测试用的 SystemConfigService
func setupSystemConfigService(t *testing.T) (*SystemConfigService, *gorm.DB) {
	db := setupSystemConfigTestDB(t)
	configRepo := repository.NewSystemConfigRepository(db)
	service := NewSystemConfigService(configRepo)
	return service, db
}

func TestSystemConfigService_Create(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	t.Run("创建字符串配置", func(t *testing.T) {
		req := &CreateConfigRequest{
			Group:       "app",
			Key:         "name",
			Value:       "Smart Locker",
			Type:        models.ConfigTypeString,
			Description: "应用名称",
			IsPublic:    true,
		}

		config, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.NotZero(t, config.ID)
		assert.Equal(t, "app", config.Group)
		assert.Equal(t, "name", config.Key)
		assert.Equal(t, "Smart Locker", config.Value)
		assert.Equal(t, models.ConfigTypeString, config.Type)
		assert.True(t, config.IsPublic)
	})

	t.Run("创建数字配置", func(t *testing.T) {
		req := &CreateConfigRequest{
			Group: "payment",
			Key:   "min_amount",
			Value: "100",
			Type:  models.ConfigTypeNumber,
		}

		config, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, models.ConfigTypeNumber, config.Type)
	})

	t.Run("创建布尔配置", func(t *testing.T) {
		req := &CreateConfigRequest{
			Group: "feature",
			Key:   "enable_wechat",
			Value: "true",
			Type:  models.ConfigTypeBoolean,
		}

		config, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, models.ConfigTypeBoolean, config.Type)
	})

	t.Run("创建JSON配置", func(t *testing.T) {
		req := &CreateConfigRequest{
			Group: "sms",
			Key:   "templates",
			Value: `{"verify":"SMS001","order":"SMS002"}`,
			Type:  models.ConfigTypeJSON,
		}

		config, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, models.ConfigTypeJSON, config.Type)
	})

	t.Run("默认类型为字符串", func(t *testing.T) {
		req := &CreateConfigRequest{
			Group: "test",
			Key:   "default_type",
			Value: "value",
		}

		config, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, models.ConfigTypeString, config.Type)
	})

	t.Run("重复配置报错", func(t *testing.T) {
		// 先验证 app:name 已存在
		var count int64
		db.Model(&models.SystemConfig{}).Where("\"group\" = ? AND \"key\" = ?", "app", "name").Count(&count)
		assert.Equal(t, int64(1), count)

		// 尝试创建重复配置
		req := &CreateConfigRequest{
			Group: "app",
			Key:   "name",
			Value: "Duplicate",
		}

		_, err := service.Create(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, ErrConfigAlreadyExists, err)
	})
}

func TestSystemConfigService_GetByID(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	// 创建测试配置
	config := &models.SystemConfig{
		Group:    "test",
		Key:      "config1",
		Value:    "value1",
		Type:     models.ConfigTypeString,
		IsPublic: false,
	}
	db.Create(config)

	t.Run("获取存在的配置", func(t *testing.T) {
		result, err := service.GetByID(ctx, config.ID)
		require.NoError(t, err)
		assert.Equal(t, config.ID, result.ID)
		assert.Equal(t, "value1", result.Value)
	})

	t.Run("获取不存在的配置", func(t *testing.T) {
		_, err := service.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestSystemConfigService_GetByGroupAndKey(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	config := &models.SystemConfig{
		Group: "payment",
		Key:   "rate",
		Value: "0.006",
		Type:  models.ConfigTypeNumber,
	}
	db.Create(config)

	t.Run("获取存在的配置", func(t *testing.T) {
		result, err := service.GetByGroupAndKey(ctx, "payment", "rate")
		require.NoError(t, err)
		assert.Equal(t, "0.006", result.Value)
	})

	t.Run("获取不存在的配置", func(t *testing.T) {
		_, err := service.GetByGroupAndKey(ctx, "nonexistent", "key")
		assert.Error(t, err)
	})
}

func TestSystemConfigService_GetByGroup(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	// 创建多个同组配置
	configs := []*models.SystemConfig{
		{Group: "rental", Key: "price_per_hour", Value: "5", Type: models.ConfigTypeNumber},
		{Group: "rental", Key: "min_hours", Value: "1", Type: models.ConfigTypeNumber},
		{Group: "rental", Key: "max_hours", Value: "24", Type: models.ConfigTypeNumber},
		{Group: "other", Key: "key1", Value: "val", Type: models.ConfigTypeString},
	}
	for _, c := range configs {
		db.Create(c)
	}

	t.Run("获取分组配置", func(t *testing.T) {
		results, err := service.GetByGroup(ctx, "rental")
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("获取空分组", func(t *testing.T) {
		results, err := service.GetByGroup(ctx, "empty")
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestSystemConfigService_Update(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	config := &models.SystemConfig{
		Group:    "update_test",
		Key:      "key1",
		Value:    "original",
		Type:     models.ConfigTypeString,
		IsPublic: false,
	}
	db.Create(config)

	t.Run("更新配置值", func(t *testing.T) {
		newType := models.ConfigTypeNumber
		newDesc := "更新后的描述"
		isPublic := true

		req := &UpdateConfigRequest{
			Value:       "100",
			Type:        &newType,
			Description: &newDesc,
			IsPublic:    &isPublic,
		}

		updated, err := service.Update(ctx, config.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "100", updated.Value)
		assert.Equal(t, models.ConfigTypeNumber, updated.Type)
		assert.Equal(t, "更新后的描述", *updated.Description)
		assert.True(t, updated.IsPublic)
	})

	t.Run("部分更新", func(t *testing.T) {
		req := &UpdateConfigRequest{
			Value: "200",
		}

		updated, err := service.Update(ctx, config.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "200", updated.Value)
		// 其他字段保持不变
		assert.Equal(t, models.ConfigTypeNumber, updated.Type)
		assert.True(t, updated.IsPublic)
	})

	t.Run("更新不存在的配置", func(t *testing.T) {
		req := &UpdateConfigRequest{Value: "test"}
		_, err := service.Update(ctx, 99999, req)
		assert.Error(t, err)
	})
}

func TestSystemConfigService_Delete(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	config := &models.SystemConfig{
		Group: "delete_test",
		Key:   "key1",
		Value: "value",
		Type:  models.ConfigTypeString,
	}
	db.Create(config)

	t.Run("删除配置", func(t *testing.T) {
		err := service.Delete(ctx, config.ID)
		require.NoError(t, err)

		// 验证已删除
		var count int64
		db.Model(&models.SystemConfig{}).Where("id = ?", config.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

func TestSystemConfigService_List(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	// 创建测试数据
	configs := []*models.SystemConfig{
		{Group: "app", Key: "name", Value: "Test App", Type: models.ConfigTypeString, IsPublic: true},
		{Group: "app", Key: "version", Value: "1.0.0", Type: models.ConfigTypeString, IsPublic: true},
		{Group: "payment", Key: "rate", Value: "0.006", Type: models.ConfigTypeNumber, IsPublic: false},
		{Group: "payment", Key: "timeout", Value: "30", Type: models.ConfigTypeNumber, IsPublic: false},
	}
	for _, c := range configs {
		db.Create(c)
	}

	t.Run("获取全部列表", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 4)
	})

	t.Run("按分组筛选", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "app", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, results, 2)
	})

	t.Run("按公开状态筛选", func(t *testing.T) {
		isPublic := true
		results, total, err := service.List(ctx, 1, 10, "", "", &isPublic)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		for _, r := range results {
			assert.True(t, r.IsPublic)
		}
	})

	t.Run("分页", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 2, "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 2)

		// 第二页
		results2, _, err := service.List(ctx, 2, 2, "", "", nil)
		require.NoError(t, err)
		assert.Len(t, results2, 2)
	})
}

func TestSystemConfigService_GetPublicConfigs(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	configs := []*models.SystemConfig{
		{Group: "app", Key: "name", Value: "App", Type: models.ConfigTypeString, IsPublic: true},
		{Group: "app", Key: "logo", Value: "logo.png", Type: models.ConfigTypeString, IsPublic: true},
		{Group: "api", Key: "secret", Value: "xxx", Type: models.ConfigTypeString, IsPublic: false},
	}
	for _, c := range configs {
		db.Create(c)
	}

	result, err := service.GetPublicConfigs(ctx)
	require.NoError(t, err)

	// 只有公开配置
	assert.Contains(t, result, "app")
	assert.Len(t, result["app"], 2)
	assert.NotContains(t, result, "api")
}

func TestSystemConfigService_GetTypedValues(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	configs := []*models.SystemConfig{
		{Group: "test", Key: "string_val", Value: "hello", Type: models.ConfigTypeString},
		{Group: "test", Key: "int_val", Value: "42", Type: models.ConfigTypeNumber},
		{Group: "test", Key: "float_val", Value: "3.14", Type: models.ConfigTypeNumber},
		{Group: "test", Key: "bool_val", Value: "true", Type: models.ConfigTypeBoolean},
		{Group: "test", Key: "json_val", Value: `{"key":"value"}`, Type: models.ConfigTypeJSON},
	}
	for _, c := range configs {
		db.Create(c)
	}

	t.Run("GetString", func(t *testing.T) {
		val := service.GetString(ctx, "test", "string_val", "default")
		assert.Equal(t, "hello", val)

		// 不存在时返回默认值
		val = service.GetString(ctx, "test", "nonexistent", "default")
		assert.Equal(t, "default", val)
	})

	t.Run("GetInt", func(t *testing.T) {
		val := service.GetInt(ctx, "test", "int_val", 0)
		assert.Equal(t, 42, val)

		// 不存在时返回默认值
		val = service.GetInt(ctx, "test", "nonexistent", 100)
		assert.Equal(t, 100, val)
	})

	t.Run("GetFloat", func(t *testing.T) {
		val := service.GetFloat(ctx, "test", "float_val", 0.0)
		assert.Equal(t, 3.14, val)

		// 不存在时返回默认值
		val = service.GetFloat(ctx, "test", "nonexistent", 1.23)
		assert.Equal(t, 1.23, val)
	})

	t.Run("GetBool", func(t *testing.T) {
		val := service.GetBool(ctx, "test", "bool_val", false)
		assert.True(t, val)

		// 不存在时返回默认值
		val = service.GetBool(ctx, "test", "nonexistent", true)
		assert.True(t, val)
	})

	t.Run("GetJSON", func(t *testing.T) {
		var result map[string]string
		err := service.GetJSON(ctx, "test", "json_val", &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])

		// 不存在时返回错误
		err = service.GetJSON(ctx, "test", "nonexistent", &result)
		assert.Error(t, err)
	})
}

func TestSystemConfigService_BatchUpdate(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	// 创建初始配置
	config := &models.SystemConfig{
		Group: "batch",
		Key:   "existing",
		Value: "old",
		Type:  models.ConfigTypeString,
	}
	db.Create(config)

	req := &BatchUpdateRequest{
		Configs: []struct {
			Group string `json:"group" binding:"required"`
			Key   string `json:"key" binding:"required"`
			Value string `json:"value" binding:"required"`
		}{
			{Group: "batch", Key: "existing", Value: "updated"},
			{Group: "batch", Key: "new_key", Value: "new_value"},
		},
	}

	err := service.BatchUpdate(ctx, req)
	require.NoError(t, err)

	// 验证更新
	updated, _ := service.GetByGroupAndKey(ctx, "batch", "existing")
	assert.Equal(t, "updated", updated.Value)

	// 验证新建
	created, _ := service.GetByGroupAndKey(ctx, "batch", "new_key")
	assert.Equal(t, "new_value", created.Value)
}

func TestSystemConfigService_GetAllGroups(t *testing.T) {
	service, db := setupSystemConfigService(t)
	ctx := context.Background()

	configs := []*models.SystemConfig{
		{Group: "app", Key: "k1", Value: "v1", Type: models.ConfigTypeString},
		{Group: "payment", Key: "k2", Value: "v2", Type: models.ConfigTypeString},
		{Group: "sms", Key: "k3", Value: "v3", Type: models.ConfigTypeString},
	}
	for _, c := range configs {
		db.Create(c)
	}

	groups, err := service.GetAllGroups(ctx)
	require.NoError(t, err)
	assert.Len(t, groups, 3)
	assert.Contains(t, groups, "app")
	assert.Contains(t, groups, "payment")
	assert.Contains(t, groups, "sms")
}
