// Package auth 提供认证服务
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"smart-locker-backend/internal/common/utils"
	"smart-locker-backend/pkg/sms"
)

// CodeService 验证码服务
type CodeService struct {
	redis     *redis.Client
	smsSender sms.Sender
	codeLen   int
	expireIn  time.Duration
}

// CodeType 验证码类型
type CodeType string

const (
	CodeTypeLogin    CodeType = "login"
	CodeTypeRegister CodeType = "register"
	CodeTypeBind     CodeType = "bind"
	CodeTypeReset    CodeType = "reset"
)

// CodeServiceConfig 验证码服务配置
type CodeServiceConfig struct {
	CodeLength int
	ExpireIn   time.Duration
}

// DefaultCodeServiceConfig 默认配置
func DefaultCodeServiceConfig() *CodeServiceConfig {
	return &CodeServiceConfig{
		CodeLength: 6,
		ExpireIn:   5 * time.Minute,
	}
}

// NewCodeService 创建验证码服务
func NewCodeService(redis *redis.Client, smsSender sms.Sender, cfg *CodeServiceConfig) *CodeService {
	if cfg == nil {
		cfg = DefaultCodeServiceConfig()
	}
	return &CodeService{
		redis:     redis,
		smsSender: smsSender,
		codeLen:   cfg.CodeLength,
		expireIn:  cfg.ExpireIn,
	}
}

// codeKey 生成验证码 Redis 键
func (s *CodeService) codeKey(phone string, codeType CodeType) string {
	return fmt.Sprintf("sms:code:%s:%s", codeType, phone)
}

// sendLimitKey 生成发送频率限制键
func (s *CodeService) sendLimitKey(phone string) string {
	return fmt.Sprintf("sms:limit:%s", phone)
}

// dayLimitKey 生成每日发送限制键
func (s *CodeService) dayLimitKey(phone string) string {
	return fmt.Sprintf("sms:day:%s", phone)
}

// SendCode 发送验证码
func (s *CodeService) SendCode(ctx context.Context, phone string, codeType CodeType) error {
	// 检查发送频率（1分钟内只能发送1条）
	limitKey := s.sendLimitKey(phone)
	exists, err := s.redis.Exists(ctx, limitKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check send limit: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("短信发送过于频繁，请稍后再试")
	}

	// 检查每日发送限制（每天最多10条）
	dayKey := s.dayLimitKey(phone)
	dayCount, err := s.redis.Incr(ctx, dayKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check day limit: %w", err)
	}
	if dayCount == 1 {
		// 设置到当天结束的过期时间
		now := time.Now()
		endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		s.redis.ExpireAt(ctx, dayKey, endOfDay)
	}
	if dayCount > 10 {
		return fmt.Errorf("今日短信发送次数已达上限")
	}

	// 生成验证码
	code := utils.GenerateRandomCode(s.codeLen)

	// 存储验证码
	codeKey := s.codeKey(phone, codeType)
	if err := s.redis.Set(ctx, codeKey, code, s.expireIn).Err(); err != nil {
		return fmt.Errorf("failed to store code: %w", err)
	}

	// 设置发送频率限制
	s.redis.Set(ctx, limitKey, "1", time.Minute)

	// 发送短信
	templateCode := s.getTemplateCode(codeType)
	if err := s.smsSender.SendCode(ctx, phone, code, templateCode); err != nil {
		// 发送失败，删除存储的验证码
		s.redis.Del(ctx, codeKey)
		return fmt.Errorf("failed to send sms: %w", err)
	}

	return nil
}

// VerifyCode 验证验证码
func (s *CodeService) VerifyCode(ctx context.Context, phone string, code string, codeType CodeType) (bool, error) {
	codeKey := s.codeKey(phone, codeType)

	storedCode, err := s.redis.Get(ctx, codeKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get code: %w", err)
	}

	if storedCode != code {
		return false, nil
	}

	// 验证成功后删除验证码（一次性使用）
	s.redis.Del(ctx, codeKey)

	return true, nil
}

// getTemplateCode 获取短信模板编码
func (s *CodeService) getTemplateCode(codeType CodeType) sms.TemplateCode {
	switch codeType {
	case CodeTypeLogin:
		return sms.TemplateCodeLogin
	case CodeTypeRegister:
		return sms.TemplateCodeRegister
	case CodeTypeBind:
		return sms.TemplateCodeBind
	case CodeTypeReset:
		return sms.TemplateCodeReset
	default:
		return sms.TemplateCodeLogin
	}
}

// GetCodeExpireIn 获取验证码有效期
func (s *CodeService) GetCodeExpireIn() time.Duration {
	return s.expireIn
}
