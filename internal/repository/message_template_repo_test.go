// Package repository 消息模板仓储单元测试
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

func setupMessageTemplateTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.MessageTemplate{})
	require.NoError(t, err)

	return db
}

func TestMessageTemplateRepository_Create(t *testing.T) {
	db := setupMessageTemplateTestDB(t)
	repo := NewMessageTemplateRepository(db)
	ctx := context.Background()

	template := &models.MessageTemplate{
		Code:      "ORDER_COMPLETE",
		Name:      "订单完成通知",
		Type:      models.MessageTemplateTypePush,
		Content:   "您的订单{{order_no}}已完成",
		Variables: models.JSON{"variables": []string{"order_no"}},
	}

	err := repo.Create(ctx, template)
	require.NoError(t, err)
	assert.NotZero(t, template.ID)
}

func TestMessageTemplateRepository_GetByID(t *testing.T) {
	db := setupMessageTemplateTestDB(t)
	repo := NewMessageTemplateRepository(db)
	ctx := context.Background()

	template := &models.MessageTemplate{
		Code:      "ORDER_COMPLETE",
		Name:      "订单完成通知",
		Type:      models.MessageTemplateTypePush,
		Content:   "您的订单{{order_no}}已完成",
		Variables: models.JSON{"variables": []string{"order_no"}},
	}
	db.Create(template)

	found, err := repo.GetByID(ctx, template.ID)
	require.NoError(t, err)
	assert.Equal(t, template.ID, found.ID)
	assert.Equal(t, "ORDER_COMPLETE", found.Code)
}

func TestMessageTemplateRepository_GetByCode(t *testing.T) {
	db := setupMessageTemplateTestDB(t)
	repo := NewMessageTemplateRepository(db)
	ctx := context.Background()

	db.Create(&models.MessageTemplate{
		Code: "ORDER_COMPLETE", Name: "订单完成通知", Type: models.MessageTemplateTypePush,
		Content: "内容", Variables: models.JSON{"variables": []string{}}, IsActive: true,
	})

	db.Model(&models.MessageTemplate{}).Create(map[string]interface{}{
		"code": "ORDER_CANCEL", "name": "订单取消通知", "type": models.MessageTemplateTypePush,
		"content": "内容", "variables": models.JSON{"variables": []string{}}, "is_active": false,
	})

	// 只返回激活的模板
	found, err := repo.GetByCode(ctx, "ORDER_COMPLETE")
	require.NoError(t, err)
	assert.Equal(t, "ORDER_COMPLETE", found.Code)

	// 禁用的模板不应返回
	_, err = repo.GetByCode(ctx, "ORDER_CANCEL")
	assert.Error(t, err)
}

func TestMessageTemplateRepository_Update(t *testing.T) {
	db := setupMessageTemplateTestDB(t)
	repo := NewMessageTemplateRepository(db)
	ctx := context.Background()

	template := &models.MessageTemplate{
		Code:      "ORDER_COMPLETE",
		Name:      "订单完成通知",
		Type:      models.MessageTemplateTypePush,
		Content:   "原内容",
		Variables: models.JSON{"variables": []string{}},
	}
	db.Create(template)

	template.Content = "新内容"
	err := repo.Update(ctx, template)
	require.NoError(t, err)

	var found models.MessageTemplate
	db.First(&found, template.ID)
	assert.Equal(t, "新内容", found.Content)
}

func TestMessageTemplateRepository_Delete(t *testing.T) {
	db := setupMessageTemplateTestDB(t)
	repo := NewMessageTemplateRepository(db)
	ctx := context.Background()

	template := &models.MessageTemplate{
		Code:      "TEST_DELETE",
		Name:      "待删除模板",
		Type:      models.MessageTemplateTypeSMS,
		Content:   "内容",
		Variables: models.JSON{"variables": []string{}},
	}
	db.Create(template)

	err := repo.Delete(ctx, template.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.MessageTemplate{}).Where("id = ?", template.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestMessageTemplateRepository_List(t *testing.T) {
	db := setupMessageTemplateTestDB(t)
	repo := NewMessageTemplateRepository(db)
	ctx := context.Background()

	db.Create(&models.MessageTemplate{
		Code: "ORDER_COMPLETE", Name: "订单完成", Type: models.MessageTemplateTypePush,
		Content: "内容", Variables: models.JSON{"variables": []string{}},
	})

	db.Create(&models.MessageTemplate{
		Code: "SMS_LOGIN", Name: "登录验证码", Type: models.MessageTemplateTypeSMS,
		Content: "内容", Variables: models.JSON{"variables": []string{}},
	})

	db.Create(&models.MessageTemplate{
		Code: "WECHAT_NOTIFY", Name: "微信通知", Type: models.MessageTemplateTypeWechat,
		Content: "内容", Variables: models.JSON{"variables": []string{}},
	})

	// 获取所有模板
	_, total, err := repo.List(ctx, 0, 10, "")
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按类型过滤
	_, total, err = repo.List(ctx, 0, 10, models.MessageTemplateTypeSMS)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestMessageTemplateRepository_GetByType(t *testing.T) {
	db := setupMessageTemplateTestDB(t)
	repo := NewMessageTemplateRepository(db)
	ctx := context.Background()

	db.Create(&models.MessageTemplate{
		Code: "SMS_LOGIN", Name: "登录验证码", Type: models.MessageTemplateTypeSMS,
		Content: "内容", Variables: models.JSON{"variables": []string{}}, IsActive: true,
	})

	db.Create(&models.MessageTemplate{
		Code: "SMS_REGISTER", Name: "注册验证码", Type: models.MessageTemplateTypeSMS,
		Content: "内容", Variables: models.JSON{"variables": []string{}}, IsActive: true,
	})

	db.Model(&models.MessageTemplate{}).Create(map[string]interface{}{
		"code": "SMS_INACTIVE", "name": "禁用短信", "type": models.MessageTemplateTypeSMS,
		"content": "内容", "variables": models.JSON{"variables": []string{}}, "is_active": false,
	})

	db.Create(&models.MessageTemplate{
		Code: "ORDER_COMPLETE", Name: "订单完成", Type: models.MessageTemplateTypePush,
		Content: "内容", Variables: models.JSON{"variables": []string{}}, IsActive: true,
	})

	templates, err := repo.GetByType(ctx, models.MessageTemplateTypeSMS)
	require.NoError(t, err)
	assert.Equal(t, 2, len(templates)) // 只返回激活的短信模板
}

func TestMessageTemplateRepository_ExistsByCode(t *testing.T) {
	db := setupMessageTemplateTestDB(t)
	repo := NewMessageTemplateRepository(db)
	ctx := context.Background()

	db.Create(&models.MessageTemplate{
		Code: "ORDER_COMPLETE", Name: "订单完成", Type: models.MessageTemplateTypePush,
		Content: "内容", Variables: models.JSON{"variables": []string{}},
	})

	exists, err := repo.ExistsByCode(ctx, "ORDER_COMPLETE")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByCode(ctx, "NOT_EXISTS")
	require.NoError(t, err)
	assert.False(t, exists)
}
