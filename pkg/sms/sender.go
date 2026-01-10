// Package sms 短信服务
package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
)

// Sender 短信发送器接口
type Sender interface {
	Send(ctx context.Context, phone, templateCode string, params map[string]string) error
	SendVerifyCode(ctx context.Context, phone, code string) error
	SendOrderNotify(ctx context.Context, phone, orderNo string) error
}

// AliyunSender 阿里云短信发送器
type AliyunSender struct {
	client     *dysmsapi.Client
	signName   string
	templates  map[string]string
}

// AliyunConfig 阿里云短信配置
type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	RegionID        string // 默认 cn-hangzhou
}

// DefaultTemplates 默认模板编码
var DefaultTemplates = map[string]string{
	"verify_code":   "SMS_xxxxxx", // 验证码模板
	"order_notify":  "SMS_xxxxxx", // 订单通知模板
	"payment_success": "SMS_xxxxxx", // 支付成功模板
	"refund_notify": "SMS_xxxxxx", // 退款通知模板
}

// NewAliyunSender 创建阿里云短信发送器
func NewAliyunSender(config *AliyunConfig) (*AliyunSender, error) {
	if config.RegionID == "" {
		config.RegionID = "cn-hangzhou"
	}

	cfg := &openapi.Config{
		AccessKeyId:     tea.String(config.AccessKeyID),
		AccessKeySecret: tea.String(config.AccessKeySecret),
		RegionId:        tea.String(config.RegionID),
	}

	client, err := dysmsapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建阿里云短信客户端失败: %v", err)
	}

	return &AliyunSender{
		client:    client,
		signName:  config.SignName,
		templates: DefaultTemplates,
	}, nil
}

// SetTemplates 设置模板编码
func (s *AliyunSender) SetTemplates(templates map[string]string) {
	for k, v := range templates {
		s.templates[k] = v
	}
}

// Send 发送短信
func (s *AliyunSender) Send(ctx context.Context, phone, templateCode string, params map[string]string) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("序列化参数失败: %v", err)
	}

	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(s.signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(string(paramsJSON)),
	}

	resp, err := s.client.SendSms(req)
	if err != nil {
		return fmt.Errorf("发送短信失败: %v", err)
	}

	if resp.Body == nil || *resp.Body.Code != "OK" {
		msg := "未知错误"
		if resp.Body != nil && resp.Body.Message != nil {
			msg = *resp.Body.Message
		}
		return fmt.Errorf("发送短信失败: %s", msg)
	}

	return nil
}

// SendVerifyCode 发送验证码
func (s *AliyunSender) SendVerifyCode(ctx context.Context, phone, code string) error {
	templateCode, ok := s.templates["verify_code"]
	if !ok {
		return fmt.Errorf("验证码模板未配置")
	}

	return s.Send(ctx, phone, templateCode, map[string]string{
		"code": code,
	})
}

// SendOrderNotify 发送订单通知
func (s *AliyunSender) SendOrderNotify(ctx context.Context, phone, orderNo string) error {
	templateCode, ok := s.templates["order_notify"]
	if !ok {
		return fmt.Errorf("订单通知模板未配置")
	}

	return s.Send(ctx, phone, templateCode, map[string]string{
		"order_no": orderNo,
	})
}

// SendPaymentSuccess 发送支付成功通知
func (s *AliyunSender) SendPaymentSuccess(ctx context.Context, phone, orderNo, amount string) error {
	templateCode, ok := s.templates["payment_success"]
	if !ok {
		return fmt.Errorf("支付成功模板未配置")
	}

	return s.Send(ctx, phone, templateCode, map[string]string{
		"order_no": orderNo,
		"amount":   amount,
	})
}

// SendRefundNotify 发送退款通知
func (s *AliyunSender) SendRefundNotify(ctx context.Context, phone, orderNo, amount string) error {
	templateCode, ok := s.templates["refund_notify"]
	if !ok {
		return fmt.Errorf("退款通知模板未配置")
	}

	return s.Send(ctx, phone, templateCode, map[string]string{
		"order_no": orderNo,
		"amount":   amount,
	})
}

// MockSender 模拟短信发送器（用于开发/测试）
type MockSender struct {
	SentMessages []MockMessage
}

// MockMessage 模拟消息
type MockMessage struct {
	Phone        string
	TemplateCode string
	Params       map[string]string
	SentAt       time.Time
}

// NewMockSender 创建模拟发送器
func NewMockSender() *MockSender {
	return &MockSender{
		SentMessages: make([]MockMessage, 0),
	}
}

// Send 模拟发送
func (s *MockSender) Send(ctx context.Context, phone, templateCode string, params map[string]string) error {
	s.SentMessages = append(s.SentMessages, MockMessage{
		Phone:        phone,
		TemplateCode: templateCode,
		Params:       params,
		SentAt:       time.Now(),
	})
	return nil
}

// SendVerifyCode 模拟发送验证码
func (s *MockSender) SendVerifyCode(ctx context.Context, phone, code string) error {
	return s.Send(ctx, phone, "verify_code", map[string]string{"code": code})
}

// SendOrderNotify 模拟发送订单通知
func (s *MockSender) SendOrderNotify(ctx context.Context, phone, orderNo string) error {
	return s.Send(ctx, phone, "order_notify", map[string]string{"order_no": orderNo})
}

// GetLastMessage 获取最后发送的消息
func (s *MockSender) GetLastMessage() *MockMessage {
	if len(s.SentMessages) == 0 {
		return nil
	}
	return &s.SentMessages[len(s.SentMessages)-1]
}

// Clear 清空消息记录
func (s *MockSender) Clear() {
	s.SentMessages = make([]MockMessage, 0)
}
