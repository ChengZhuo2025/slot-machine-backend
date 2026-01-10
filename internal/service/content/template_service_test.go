// Package content 消息模板服务单元测试
package content

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

// setupTemplateTestDB 创建模板测试数据库
func setupTemplateTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.MessageTemplate{})
	require.NoError(t, err)

	return db
}

// setupTemplateService 创建测试用的 TemplateService
func setupTemplateService(t *testing.T) (*TemplateService, *gorm.DB) {
	db := setupTemplateTestDB(t)
	templateRepo := repository.NewMessageTemplateRepository(db)
	service := NewTemplateService(templateRepo)
	return service, db
}

// setupMessageTemplateAdminService 创建测试用的 MessageTemplateAdminService
func setupMessageTemplateAdminService(t *testing.T) (*MessageTemplateAdminService, *gorm.DB) {
	db := setupTemplateTestDB(t)
	templateRepo := repository.NewMessageTemplateRepository(db)
	service := NewMessageTemplateAdminService(templateRepo)
	return service, db
}

func TestTemplateService_GetByCode(t *testing.T) {
	service, db := setupTemplateService(t)
	ctx := context.Background()

	// 创建测试模板
	template := &models.MessageTemplate{
		Code:     "VERIFY_CODE",
		Name:     "验证码",
		Type:     "sms",
		Content:  "您的验证码是${code}，有效期${minutes}分钟。",
		IsActive: true,
	}
	db.Create(template)

	t.Run("获取活跃模板", func(t *testing.T) {
		result, err := service.GetByCode(ctx, "VERIFY_CODE")
		require.NoError(t, err)
		assert.Equal(t, "VERIFY_CODE", result.Code)
		assert.Equal(t, "验证码", result.Name)
	})

	t.Run("获取不存在的模板", func(t *testing.T) {
		_, err := service.GetByCode(ctx, "NONEXISTENT")
		assert.Error(t, err)
	})
}

func TestTemplateService_RenderTemplate(t *testing.T) {
	service, db := setupTemplateService(t)
	ctx := context.Background()

	// 创建使用 ${key} 语法的模板
	template := &models.MessageTemplate{
		Code:     "ORDER_PAID",
		Name:     "订单支付成功",
		Type:     "sms",
		Content:  "您的订单${orderNo}已支付成功，金额${amount}元。",
		IsActive: true,
	}
	db.Create(template)

	// 创建使用 Go 模板语法的模板
	goTemplate := &models.MessageTemplate{
		Code:     "WELCOME",
		Name:     "欢迎消息",
		Type:     "push",
		Content:  "欢迎{{.name}}加入，您的等级是{{.level}}。",
		IsActive: true,
	}
	db.Create(goTemplate)

	t.Run("渲染 ${} 语法模板", func(t *testing.T) {
		data := map[string]interface{}{
			"orderNo": "ORD123456",
			"amount":  99.99,
		}

		result, err := service.RenderTemplate(ctx, "ORDER_PAID", data)
		require.NoError(t, err)
		assert.Equal(t, "您的订单ORD123456已支付成功，金额99.99元。", result)
	})

	t.Run("渲染 Go 模板语法", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "张三",
			"level": "VIP",
		}

		result, err := service.RenderTemplate(ctx, "WELCOME", data)
		require.NoError(t, err)
		assert.Equal(t, "欢迎张三加入，您的等级是VIP。", result)
	})

	t.Run("渲染不存在的模板", func(t *testing.T) {
		data := map[string]interface{}{}
		_, err := service.RenderTemplate(ctx, "NONEXISTENT", data)
		assert.Error(t, err)
	})
}

func TestTemplateService_RenderContent(t *testing.T) {
	service, _ := setupTemplateService(t)

	t.Run("渲染 ${} 语法", func(t *testing.T) {
		content := "尊敬的${name}，您好！您的订单${orderNo}已发货。"
		data := map[string]interface{}{
			"name":    "张三",
			"orderNo": "ORD123456",
		}

		result, err := service.RenderContent(content, data)
		require.NoError(t, err)
		assert.Equal(t, "尊敬的张三，您好！您的订单ORD123456已发货。", result)
	})

	t.Run("渲染 Go 模板语法", func(t *testing.T) {
		content := "您的验证码是{{.code}}，有效期{{.minutes}}分钟。"
		data := map[string]interface{}{
			"code":    "123456",
			"minutes": 5,
		}

		result, err := service.RenderContent(content, data)
		require.NoError(t, err)
		assert.Equal(t, "您的验证码是123456，有效期5分钟。", result)
	})

	t.Run("混合语法", func(t *testing.T) {
		content := "您好${name}，验证码{{.code}}，有效期{{.minutes}}分钟。"
		data := map[string]interface{}{
			"name":    "用户",
			"code":    "666666",
			"minutes": 10,
		}

		result, err := service.RenderContent(content, data)
		require.NoError(t, err)
		assert.Equal(t, "您好用户，验证码666666，有效期10分钟。", result)
	})

	t.Run("缺少变量不报错", func(t *testing.T) {
		content := "您好${name}，欢迎使用！"
		data := map[string]interface{}{}

		result, err := service.RenderContent(content, data)
		require.NoError(t, err)
		assert.Equal(t, "您好${name}，欢迎使用！", result) // 未匹配的变量保留原样
	})

	t.Run("数值类型转换", func(t *testing.T) {
		content := "金额：${amount}元，数量：${count}个"
		data := map[string]interface{}{
			"amount": 199.99,
			"count":  3,
		}

		result, err := service.RenderContent(content, data)
		require.NoError(t, err)
		assert.Equal(t, "金额：199.99元，数量：3个", result)
	})
}

func TestMessageTemplateAdminService_Create(t *testing.T) {
	service, _ := setupMessageTemplateAdminService(t)
	ctx := context.Background()

	t.Run("创建短信模板", func(t *testing.T) {
		req := &CreateTemplateRequest{
			Code:      "SMS_VERIFY",
			Name:      "短信验证码",
			Type:      "sms",
			Content:   "验证码${code}，有效期${minutes}分钟。",
			Variables: []string{"code", "minutes"},
			IsActive:  true,
		}

		template, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.NotZero(t, template.ID)
		assert.Equal(t, "SMS_VERIFY", template.Code)
		assert.Equal(t, "sms", template.Type)
		assert.True(t, template.IsActive)
	})

	t.Run("创建推送模板", func(t *testing.T) {
		req := &CreateTemplateRequest{
			Code:     "PUSH_ORDER",
			Name:     "订单通知",
			Type:     "push",
			Content:  "您的订单{{.orderNo}}已支付成功",
			IsActive: true,
		}

		template, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "push", template.Type)
	})

	t.Run("创建微信模板", func(t *testing.T) {
		req := &CreateTemplateRequest{
			Code:      "WECHAT_BOOKING",
			Name:      "预订成功通知",
			Type:      "wechat",
			Content:   "预订成功！房间号：{{.roomNo}}，入住时间：{{.checkInTime}}",
			Variables: []string{"roomNo", "checkInTime"},
			IsActive:  true, // 创建时设为true，后续测试更新逻辑
		}

		template, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "wechat", template.Type)
		assert.True(t, template.IsActive)
	})
}

func TestMessageTemplateAdminService_Update(t *testing.T) {
	service, db := setupMessageTemplateAdminService(t)
	ctx := context.Background()

	// 创建测试模板
	template := &models.MessageTemplate{
		Code:     "UPDATE_TEST",
		Name:     "原名称",
		Type:     "sms",
		Content:  "原内容",
		IsActive: true,
	}
	db.Create(template)

	t.Run("更新模板", func(t *testing.T) {
		req := &CreateTemplateRequest{
			Code:      "UPDATE_TEST", // Code不变
			Name:      "新名称",
			Type:      "push",
			Content:   "新内容${var}",
			Variables: []string{"var"},
			IsActive:  false,
		}

		updated, err := service.Update(ctx, template.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "新名称", updated.Name)
		assert.Equal(t, "push", updated.Type)
		assert.Equal(t, "新内容${var}", updated.Content)
		assert.False(t, updated.IsActive)
	})

	t.Run("更新不存在的模板", func(t *testing.T) {
		req := &CreateTemplateRequest{
			Code:    "TEST",
			Name:    "测试",
			Type:    "sms",
			Content: "内容",
		}
		_, err := service.Update(ctx, 99999, req)
		assert.Error(t, err)
	})
}

func TestMessageTemplateAdminService_Delete(t *testing.T) {
	service, db := setupMessageTemplateAdminService(t)
	ctx := context.Background()

	template := &models.MessageTemplate{
		Code:     "DELETE_TEST",
		Name:     "待删除模板",
		Type:     "sms",
		Content:  "测试内容",
		IsActive: true,
	}
	db.Create(template)

	t.Run("删除模板", func(t *testing.T) {
		err := service.Delete(ctx, template.ID)
		require.NoError(t, err)

		// 验证已删除
		var count int64
		db.Model(&models.MessageTemplate{}).Where("id = ?", template.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

func TestMessageTemplateAdminService_List(t *testing.T) {
	service, db := setupMessageTemplateAdminService(t)
	ctx := context.Background()

	// 创建测试数据
	templates := []*models.MessageTemplate{
		{Code: "SMS_1", Name: "短信1", Type: "sms", Content: "内容1", IsActive: true},
		{Code: "SMS_2", Name: "短信2", Type: "sms", Content: "内容2", IsActive: true},
		{Code: "PUSH_1", Name: "推送1", Type: "push", Content: "内容3", IsActive: true},
		{Code: "WECHAT_1", Name: "微信1", Type: "wechat", Content: "内容4", IsActive: true},
	}
	for _, tpl := range templates {
		db.Create(tpl)
	}
	// 手动更新一个为非活跃状态
	db.Model(&models.MessageTemplate{}).Where("code = ?", "WECHAT_1").Update("is_active", false)

	t.Run("获取全部模板", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "")
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 4)
	})

	t.Run("按类型筛选", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "sms")
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		for _, r := range results {
			assert.Equal(t, "sms", r.Type)
		}
	})

	t.Run("分页", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 2, "")
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 2)

		results2, _, err := service.List(ctx, 2, 2, "")
		require.NoError(t, err)
		assert.Len(t, results2, 2)
	})
}

func TestToString(t *testing.T) {
	t.Run("字符串", func(t *testing.T) {
		assert.Equal(t, "hello", toString("hello"))
	})

	t.Run("整数", func(t *testing.T) {
		assert.Equal(t, "42", toString(42))
	})

	t.Run("浮点数", func(t *testing.T) {
		assert.Equal(t, "3.14", toString(3.14))
	})

	t.Run("布尔值", func(t *testing.T) {
		assert.Equal(t, "true", toString(true))
		assert.Equal(t, "false", toString(false))
	})

	t.Run("nil", func(t *testing.T) {
		assert.Equal(t, "<nil>", toString(nil))
	})
}
