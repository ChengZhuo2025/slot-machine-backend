// Package content 内容服务
package content

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// TemplateService 消息模板服务
type TemplateService struct {
	templateRepo *repository.MessageTemplateRepository
}

// NewTemplateService 创建消息模板服务
func NewTemplateService(templateRepo *repository.MessageTemplateRepository) *TemplateService {
	return &TemplateService{templateRepo: templateRepo}
}

// GetByCode 根据编码获取模板
func (s *TemplateService) GetByCode(ctx context.Context, code string) (*models.MessageTemplate, error) {
	return s.templateRepo.GetByCode(ctx, code)
}

// RenderTemplate 渲染模板
func (s *TemplateService) RenderTemplate(ctx context.Context, code string, data map[string]interface{}) (string, error) {
	tpl, err := s.templateRepo.GetByCode(ctx, code)
	if err != nil {
		return "", err
	}

	return s.render(tpl.Content, data)
}

// RenderContent 渲染内容
func (s *TemplateService) RenderContent(content string, data map[string]interface{}) (string, error) {
	return s.render(content, data)
}

// render 执行模板渲染
func (s *TemplateService) render(content string, data map[string]interface{}) (string, error) {
	// 支持两种模板语法：{{.key}} 和 ${key}
	// 先处理 ${key} 格式
	for key, value := range data {
		placeholder := "${" + key + "}"
		content = strings.ReplaceAll(content, placeholder, toString(value))
	}

	// 再处理 Go 模板语法
	tmpl, err := template.New("message").Parse(content)
	if err != nil {
		return content, nil // 解析失败时返回原内容
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return content, nil
	}

	return buf.String(), nil
}

// toString 将任意类型转换为字符串
func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

// 预定义模板编码
const (
	TemplateVerifyCode     = "VERIFY_CODE"      // 验证码
	TemplateOrderPaid      = "ORDER_PAID"       // 订单支付成功
	TemplateOrderShipped   = "ORDER_SHIPPED"    // 订单发货
	TemplateRefundSuccess  = "REFUND_SUCCESS"   // 退款成功
	TemplateBookingSuccess = "BOOKING_SUCCESS"  // 预订成功
	TemplateWithdrawSuccess = "WITHDRAW_SUCCESS" // 提现成功
)

// MessageTemplateAdminService 消息模板管理服务
type MessageTemplateAdminService struct {
	templateRepo *repository.MessageTemplateRepository
}

// NewMessageTemplateAdminService 创建消息模板管理服务
func NewMessageTemplateAdminService(templateRepo *repository.MessageTemplateRepository) *MessageTemplateAdminService {
	return &MessageTemplateAdminService{templateRepo: templateRepo}
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Code      string   `json:"code" binding:"required"`
	Name      string   `json:"name" binding:"required"`
	Type      string   `json:"type" binding:"required,oneof=sms push wechat"`
	Content   string   `json:"content" binding:"required"`
	Variables []string `json:"variables"`
	IsActive  bool     `json:"is_active"`
}

// Create 创建模板
func (s *MessageTemplateAdminService) Create(ctx context.Context, req *CreateTemplateRequest) (*models.MessageTemplate, error) {
	tpl := &models.MessageTemplate{
		Code:     req.Code,
		Name:     req.Name,
		Type:     req.Type,
		Content:  req.Content,
		IsActive: req.IsActive,
	}

	if len(req.Variables) > 0 {
		tpl.Variables = models.JSON{"variables": req.Variables}
	}

	if err := s.templateRepo.Create(ctx, tpl); err != nil {
		return nil, err
	}

	return tpl, nil
}

// Update 更新模板
func (s *MessageTemplateAdminService) Update(ctx context.Context, id int64, req *CreateTemplateRequest) (*models.MessageTemplate, error) {
	tpl, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tpl.Name = req.Name
	tpl.Type = req.Type
	tpl.Content = req.Content
	tpl.IsActive = req.IsActive

	if len(req.Variables) > 0 {
		tpl.Variables = models.JSON{"variables": req.Variables}
	}

	if err := s.templateRepo.Update(ctx, tpl); err != nil {
		return nil, err
	}

	return tpl, nil
}

// Delete 删除模板
func (s *MessageTemplateAdminService) Delete(ctx context.Context, id int64) error {
	return s.templateRepo.Delete(ctx, id)
}

// List 获取模板列表
func (s *MessageTemplateAdminService) List(ctx context.Context, page, pageSize int, templateType string) ([]*models.MessageTemplate, int64, error) {
	offset := (page - 1) * pageSize
	return s.templateRepo.List(ctx, offset, pageSize, templateType)
}
