// Package hotel 提供酒店预订相关服务
package hotel

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// CodeService 验证码服务
type CodeService struct{}

// NewCodeService 创建验证码服务
func NewCodeService() *CodeService {
	return &CodeService{}
}

// GenerateVerificationCode 生成核销码
// 格式：20位字母数字组合，便于酒店前台扫码核销
func (s *CodeService) GenerateVerificationCode() string {
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		// 降级使用时间戳
		return fmt.Sprintf("V%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("V%s", hex.EncodeToString(bytes)[:19])
}

// GenerateUnlockCode 生成开锁码
// 格式：6位数字，便于用户在设备上输入
func (s *CodeService) GenerateUnlockCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// 降级使用时间戳后6位
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	// 转换为6位数字
	num := int(bytes[0])<<24 | int(bytes[1])<<16 | int(bytes[2])<<8 | int(bytes[3])
	if num < 0 {
		num = -num
	}
	return fmt.Sprintf("%06d", num%1000000)
}

// GenerateQRCodeURL 生成核销二维码 URL
func (s *CodeService) GenerateQRCodeURL(bookingNo string, verificationCode string) string {
	// 二维码内容可以是一个包含预订信息的URL或JSON
	return fmt.Sprintf("/api/v1/hotel/verify/%s?code=%s", bookingNo, verificationCode)
}

// ValidateUnlockCode 验证开锁码格式
func (s *CodeService) ValidateUnlockCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ValidateVerificationCode 验证核销码格式
func (s *CodeService) ValidateVerificationCode(code string) bool {
	if len(code) < 10 || len(code) > 20 {
		return false
	}
	for _, c := range code {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == 'V') {
			return false
		}
	}
	return true
}

// IsUnlockCodeValid 检查开锁码是否在有效时段内
func (s *CodeService) IsUnlockCodeValid(checkInTime, checkOutTime time.Time) bool {
	now := time.Now()
	// 开锁码仅在入住时间到退房时间内有效
	return now.After(checkInTime) && now.Before(checkOutTime)
}
