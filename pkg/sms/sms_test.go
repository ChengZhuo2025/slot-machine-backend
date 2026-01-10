// Package sms 短信服务单元测试
package sms

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockSender_Send(t *testing.T) {
	sender := NewMockSender()
	ctx := context.Background()

	t.Run("发送短信", func(t *testing.T) {
		err := sender.Send(ctx, "13800138000", "SMS_TEMPLATE", map[string]string{
			"code": "123456",
		})
		require.NoError(t, err)

		// 验证消息已记录
		assert.Len(t, sender.SentMessages, 1)
		msg := sender.SentMessages[0]
		assert.Equal(t, "13800138000", msg.Phone)
		assert.Equal(t, "SMS_TEMPLATE", msg.TemplateCode)
		assert.Equal(t, "123456", msg.Params["code"])
		assert.NotZero(t, msg.SentAt)
	})

	t.Run("发送多条短信", func(t *testing.T) {
		sender.Clear()

		sender.Send(ctx, "13800138001", "T1", map[string]string{"key": "val1"})
		sender.Send(ctx, "13800138002", "T2", map[string]string{"key": "val2"})
		sender.Send(ctx, "13800138003", "T3", map[string]string{"key": "val3"})

		assert.Len(t, sender.SentMessages, 3)
	})
}

func TestMockSender_SendVerifyCode(t *testing.T) {
	sender := NewMockSender()
	ctx := context.Background()

	err := sender.SendVerifyCode(ctx, "13800138000", "654321")
	require.NoError(t, err)

	msg := sender.GetLastMessage()
	require.NotNil(t, msg)
	assert.Equal(t, "13800138000", msg.Phone)
	assert.Equal(t, "verify_code", msg.TemplateCode)
	assert.Equal(t, "654321", msg.Params["code"])
}

func TestMockSender_SendOrderNotify(t *testing.T) {
	sender := NewMockSender()
	ctx := context.Background()

	err := sender.SendOrderNotify(ctx, "13900139000", "ORD202501010001")
	require.NoError(t, err)

	msg := sender.GetLastMessage()
	require.NotNil(t, msg)
	assert.Equal(t, "13900139000", msg.Phone)
	assert.Equal(t, "order_notify", msg.TemplateCode)
	assert.Equal(t, "ORD202501010001", msg.Params["order_no"])
}

func TestMockSender_GetLastMessage(t *testing.T) {
	sender := NewMockSender()
	ctx := context.Background()

	t.Run("空消息列表", func(t *testing.T) {
		msg := sender.GetLastMessage()
		assert.Nil(t, msg)
	})

	t.Run("有消息时返回最后一条", func(t *testing.T) {
		sender.Send(ctx, "phone1", "T1", nil)
		sender.Send(ctx, "phone2", "T2", nil)
		sender.Send(ctx, "phone3", "T3", nil)

		msg := sender.GetLastMessage()
		require.NotNil(t, msg)
		assert.Equal(t, "phone3", msg.Phone)
		assert.Equal(t, "T3", msg.TemplateCode)
	})
}

func TestMockSender_Clear(t *testing.T) {
	sender := NewMockSender()
	ctx := context.Background()

	sender.Send(ctx, "phone1", "T1", nil)
	sender.Send(ctx, "phone2", "T2", nil)

	assert.Len(t, sender.SentMessages, 2)

	sender.Clear()

	assert.Len(t, sender.SentMessages, 0)
	assert.Nil(t, sender.GetLastMessage())
}

func TestMockClient_SendCode(t *testing.T) {
	client := NewMockClient("测试签名")
	ctx := context.Background()

	t.Run("发送登录验证码", func(t *testing.T) {
		err := client.SendCode(ctx, "13800138000", "123456", TemplateCodeLogin)
		require.NoError(t, err)
	})

	t.Run("发送注册验证码", func(t *testing.T) {
		err := client.SendCode(ctx, "13800138001", "654321", TemplateCodeRegister)
		require.NoError(t, err)
	})

	t.Run("发送绑定验证码", func(t *testing.T) {
		err := client.SendCode(ctx, "13800138002", "111111", TemplateCodeBind)
		require.NoError(t, err)
	})

	t.Run("发送重置密码验证码", func(t *testing.T) {
		err := client.SendCode(ctx, "13800138003", "222222", TemplateCodeReset)
		require.NoError(t, err)
	})
}

func TestMockClient_SendNotification(t *testing.T) {
	client := NewMockClient("测试签名")
	ctx := context.Background()

	t.Run("发送通知短信", func(t *testing.T) {
		params := map[string]string{
			"order_no": "ORD123456",
			"amount":   "99.00",
		}
		err := client.SendNotification(ctx, "13800138000", "SMS_ORDER_PAID", params)
		require.NoError(t, err)
	})

	t.Run("发送空参数通知", func(t *testing.T) {
		err := client.SendNotification(ctx, "13800138001", "SMS_SIMPLE", nil)
		require.NoError(t, err)
	})
}

func TestTemplateCode_Constants(t *testing.T) {
	// 验证模板编码常量
	assert.Equal(t, TemplateCode("SMS_LOGIN"), TemplateCodeLogin)
	assert.Equal(t, TemplateCode("SMS_REGISTER"), TemplateCodeRegister)
	assert.Equal(t, TemplateCode("SMS_BIND"), TemplateCodeBind)
	assert.Equal(t, TemplateCode("SMS_RESET"), TemplateCodeReset)
}

func TestDefaultTemplates(t *testing.T) {
	// 验证默认模板存在
	assert.Contains(t, DefaultTemplates, "verify_code")
	assert.Contains(t, DefaultTemplates, "order_notify")
	assert.Contains(t, DefaultTemplates, "payment_success")
	assert.Contains(t, DefaultTemplates, "refund_notify")
}

func TestAliyunSender_SetTemplates(t *testing.T) {
	// 由于 AliyunSender 需要阿里云凭证，这里只测试 SetTemplates 逻辑
	// 使用结构体直接测试
	sender := &AliyunSender{
		templates: make(map[string]string),
	}

	// 初始化默认模板
	for k, v := range DefaultTemplates {
		sender.templates[k] = v
	}

	t.Run("设置新模板", func(t *testing.T) {
		sender.SetTemplates(map[string]string{
			"new_template": "SMS_NEW",
		})
		assert.Equal(t, "SMS_NEW", sender.templates["new_template"])
	})

	t.Run("覆盖已有模板", func(t *testing.T) {
		sender.SetTemplates(map[string]string{
			"verify_code": "SMS_CUSTOM_VERIFY",
		})
		assert.Equal(t, "SMS_CUSTOM_VERIFY", sender.templates["verify_code"])
	})
}

// TestSenderInterfaceImpl 验证接口实现
func TestSenderInterfaceImpl(t *testing.T) {
	// 验证 MockSender 实现了 sender.go 中的 Sender 接口（Send, SendVerifyCode, SendOrderNotify）
	// 注意：sender.go 和 aliyun.go 有两个不同的 Sender 接口定义
	// MockSender 实现的是 sender.go 中的 Sender 接口
	sender := NewMockSender()

	// 验证方法存在且可调用
	ctx := context.Background()
	_ = sender.Send(ctx, "phone", "code", nil)
	_ = sender.SendVerifyCode(ctx, "phone", "code")
	_ = sender.SendOrderNotify(ctx, "phone", "order")
}

// TestMockClientInterface 验证 MockClient 实现
func TestMockClientInterface(t *testing.T) {
	// 验证 MockClient 的方法可调用
	client := NewMockClient("测试签名")
	ctx := context.Background()

	_ = client.SendCode(ctx, "13800138000", "123456", TemplateCodeLogin)
	_ = client.SendNotification(ctx, "13800138000", "SMS_TEST", map[string]string{"key": "value"})
}
