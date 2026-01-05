// Package hotel CodeService 单元测试
package hotel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCodeService_GenerateVerificationCode(t *testing.T) {
	svc := NewCodeService()

	t.Run("生成核销码格式正确", func(t *testing.T) {
		code := svc.GenerateVerificationCode()
		assert.NotEmpty(t, code)
		assert.True(t, len(code) >= 10 && len(code) <= 20, "核销码长度应在10-20之间")
		assert.Equal(t, byte('V'), code[0], "核销码应以V开头")
	})

	t.Run("生成的核销码唯一", func(t *testing.T) {
		codes := make(map[string]bool)
		for i := 0; i < 100; i++ {
			code := svc.GenerateVerificationCode()
			assert.False(t, codes[code], "核销码应该唯一")
			codes[code] = true
		}
	})
}

func TestCodeService_GenerateUnlockCode(t *testing.T) {
	svc := NewCodeService()

	t.Run("生成开锁码格式正确", func(t *testing.T) {
		code := svc.GenerateUnlockCode()
		assert.NotEmpty(t, code)
		assert.Len(t, code, 6, "开锁码应为6位")
		// 验证是纯数字
		for _, c := range code {
			assert.True(t, c >= '0' && c <= '9', "开锁码应该只包含数字")
		}
	})

	t.Run("生成的开锁码多样性", func(t *testing.T) {
		codes := make(map[string]int)
		for i := 0; i < 100; i++ {
			code := svc.GenerateUnlockCode()
			codes[code]++
		}
		// 100次生成应该有较多不同的值（允许少量重复）
		assert.Greater(t, len(codes), 50, "开锁码应该有足够的多样性")
	})
}

func TestCodeService_GenerateQRCodeURL(t *testing.T) {
	svc := NewCodeService()

	t.Run("生成二维码URL格式正确", func(t *testing.T) {
		bookingNo := "B202401010001"
		verificationCode := "V1234567890123456789"

		url := svc.GenerateQRCodeURL(bookingNo, verificationCode)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, bookingNo)
		assert.Contains(t, url, verificationCode)
		assert.Contains(t, url, "/api/v1/hotel/verify/")
	})
}

func TestCodeService_ValidateUnlockCode(t *testing.T) {
	svc := NewCodeService()

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"有效的6位数字", "123456", true},
		{"有效的6位数字（全0）", "000000", true},
		{"有效的6位数字（全9）", "999999", true},
		{"太短", "12345", false},
		{"太长", "1234567", false},
		{"包含字母", "12345a", false},
		{"包含空格", "12345 ", false},
		{"空字符串", "", false},
		{"包含特殊字符", "12345!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.ValidateUnlockCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCodeService_ValidateVerificationCode(t *testing.T) {
	svc := NewCodeService()

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"有效的核销码（20位）", "V1234567890abcdef123", true},
		{"有效的核销码（10位）", "V123456789", true},
		{"有效的核销码（含大写字母）", "V123456789ABCDEF", true},
		{"太短（9位）", "V12345678", false},
		{"太长（21位）", "V12345678901234567890", false},
		{"包含无效字符", "V123456789@#$%", false},
		{"空字符串", "", false},
		{"不以V开头但格式合规", "1234567890", true}, // ValidateVerificationCode 不强制 V 开头
		{"包含非十六进制字符", "V123456789ghijk", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.ValidateVerificationCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCodeService_IsUnlockCodeValid(t *testing.T) {
	svc := NewCodeService()

	t.Run("当前时间在有效时段内", func(t *testing.T) {
		checkInTime := time.Now().Add(-1 * time.Hour)
		checkOutTime := time.Now().Add(1 * time.Hour)

		result := svc.IsUnlockCodeValid(checkInTime, checkOutTime)
		assert.True(t, result)
	})

	t.Run("当前时间在入住时间之前", func(t *testing.T) {
		checkInTime := time.Now().Add(1 * time.Hour)
		checkOutTime := time.Now().Add(3 * time.Hour)

		result := svc.IsUnlockCodeValid(checkInTime, checkOutTime)
		assert.False(t, result)
	})

	t.Run("当前时间在退房时间之后", func(t *testing.T) {
		checkInTime := time.Now().Add(-3 * time.Hour)
		checkOutTime := time.Now().Add(-1 * time.Hour)

		result := svc.IsUnlockCodeValid(checkInTime, checkOutTime)
		assert.False(t, result)
	})

	t.Run("入住时间正好是现在", func(t *testing.T) {
		// 边界情况：入住时间稍微在未来（确保 now.After(checkInTime) 返回 false）
		checkInTime := time.Now().Add(1 * time.Millisecond)
		checkOutTime := time.Now().Add(2 * time.Hour)

		result := svc.IsUnlockCodeValid(checkInTime, checkOutTime)
		assert.False(t, result)
	})

	t.Run("退房时间正好是现在", func(t *testing.T) {
		// 边界情况：退房时间是现在，应该返回false（需要before，不是equal）
		checkInTime := time.Now().Add(-2 * time.Hour)
		checkOutTime := time.Now()

		result := svc.IsUnlockCodeValid(checkInTime, checkOutTime)
		assert.False(t, result)
	})
}

func TestCodeService_Integration(t *testing.T) {
	svc := NewCodeService()

	t.Run("生成的核销码应该能通过验证", func(t *testing.T) {
		code := svc.GenerateVerificationCode()
		assert.True(t, svc.ValidateVerificationCode(code))
	})

	t.Run("生成的开锁码应该能通过验证", func(t *testing.T) {
		code := svc.GenerateUnlockCode()
		assert.True(t, svc.ValidateUnlockCode(code))
	})
}
