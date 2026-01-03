// Package sms 提供短信服务
package sms

import (
	"context"
	"encoding/json"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
)

// Config 短信配置
type Config struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	Endpoint        string
}

// Client 短信客户端
type Client struct {
	client   *dysmsapi.Client
	signName string
}

// TemplateCode 短信模板编码
type TemplateCode string

const (
	TemplateCodeLogin    TemplateCode = "SMS_LOGIN"    // 登录验证码
	TemplateCodeRegister TemplateCode = "SMS_REGISTER" // 注册验证码
	TemplateCodeBind     TemplateCode = "SMS_BIND"     // 绑定验证码
	TemplateCodeReset    TemplateCode = "SMS_RESET"    // 重置密码
)

// NewClient 创建短信客户端
func NewClient(cfg *Config) (*Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(cfg.AccessKeyID),
		AccessKeySecret: tea.String(cfg.AccessKeySecret),
	}

	if cfg.Endpoint != "" {
		config.Endpoint = tea.String(cfg.Endpoint)
	} else {
		config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
	}

	client, err := dysmsapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sms client: %w", err)
	}

	return &Client{
		client:   client,
		signName: cfg.SignName,
	}, nil
}

// SendCode 发送验证码
func (c *Client) SendCode(ctx context.Context, phone string, code string, templateCode TemplateCode) error {
	templateParam, _ := json.Marshal(map[string]string{
		"code": code,
	})

	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(c.signName),
		TemplateCode:  tea.String(string(templateCode)),
		TemplateParam: tea.String(string(templateParam)),
	}

	response, err := c.client.SendSms(request)
	if err != nil {
		return fmt.Errorf("failed to send sms: %w", err)
	}

	if *response.Body.Code != "OK" {
		return fmt.Errorf("sms send failed: %s - %s", *response.Body.Code, *response.Body.Message)
	}

	return nil
}

// SendNotification 发送通知短信
func (c *Client) SendNotification(ctx context.Context, phone string, templateCode TemplateCode, params map[string]string) error {
	templateParam, _ := json.Marshal(params)

	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(c.signName),
		TemplateCode:  tea.String(string(templateCode)),
		TemplateParam: tea.String(string(templateParam)),
	}

	response, err := c.client.SendSms(request)
	if err != nil {
		return fmt.Errorf("failed to send sms: %w", err)
	}

	if *response.Body.Code != "OK" {
		return fmt.Errorf("sms send failed: %s - %s", *response.Body.Code, *response.Body.Message)
	}

	return nil
}

// MockClient 模拟短信客户端（用于开发测试）
type MockClient struct {
	signName string
}

// NewMockClient 创建模拟客户端
func NewMockClient(signName string) *MockClient {
	return &MockClient{signName: signName}
}

// SendCode 模拟发送验证码（仅打印日志）
func (c *MockClient) SendCode(ctx context.Context, phone string, code string, templateCode TemplateCode) error {
	fmt.Printf("[MockSMS] Send code to %s: %s (template: %s)\n", phone, code, templateCode)
	return nil
}

// SendNotification 模拟发送通知
func (c *MockClient) SendNotification(ctx context.Context, phone string, templateCode TemplateCode, params map[string]string) error {
	fmt.Printf("[MockSMS] Send notification to %s (template: %s, params: %v)\n", phone, templateCode, params)
	return nil
}

// Sender 短信发送接口
type Sender interface {
	SendCode(ctx context.Context, phone string, code string, templateCode TemplateCode) error
	SendNotification(ctx context.Context, phone string, templateCode TemplateCode, params map[string]string) error
}
