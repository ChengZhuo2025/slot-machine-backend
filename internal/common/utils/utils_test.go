// Package utils 通用工具函数单元测试
package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== GenerateOrderNo 测试 ====================

func TestGenerateOrderNo(t *testing.T) {
	tests := []string{"O", "R", "M", ""}

	for _, prefix := range tests {
		t.Run("prefix_"+prefix, func(t *testing.T) {
			orderNo := GenerateOrderNo(prefix)
			assert.NotEmpty(t, orderNo)
			assert.True(t, strings.HasPrefix(orderNo, prefix))
			// 验证格式：前缀 + 14位时间戳 + 6位随机数 = 前缀长度 + 20
			assert.Equal(t, len(prefix)+20, len(orderNo))
		})
	}
}

func TestGenerateOrderNo_Uniqueness(t *testing.T) {
	prefix := "O"
	iterations := 100
	seen := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		orderNo := GenerateOrderNo(prefix)
		assert.False(t, seen[orderNo], "订单号应该是唯一的")
		seen[orderNo] = true
	}
}

// ==================== GenerateRandomNumber 测试 ====================

func TestGenerateRandomNumber(t *testing.T) {
	tests := []int{4, 6, 8, 10}

	for _, length := range tests {
		t.Run(string(rune(length)), func(t *testing.T) {
			number := GenerateRandomNumber(length)
			assert.Equal(t, length, len(number))
			// 验证全是数字
			for _, c := range number {
				assert.True(t, c >= '0' && c <= '9')
			}
		})
	}
}

func TestGenerateRandomNumber_Uniqueness(t *testing.T) {
	length := 6
	iterations := 100
	seen := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		number := GenerateRandomNumber(length)
		// 由于是随机的，可能会有重复，但概率很低
		seen[number] = true
	}
	// 至少应该生成一些不同的数字
	assert.Greater(t, len(seen), 50)
}

// ==================== GenerateRandomCode 测试 ====================

func TestGenerateRandomCode(t *testing.T) {
	tests := []int{4, 6, 8}

	for _, length := range tests {
		code := GenerateRandomCode(length)
		assert.Equal(t, length, len(code))
		// 验证全是数字
		for _, c := range code {
			assert.True(t, c >= '0' && c <= '9')
		}
	}
}

// ==================== GenerateInviteCode 测试 ====================

func TestGenerateInviteCode(t *testing.T) {
	tests := []int{6, 8, 10}

	for _, length := range tests {
		t.Run(string(rune(length)), func(t *testing.T) {
			code := GenerateInviteCode(length)
			assert.Equal(t, length, len(code))

			// 验证不包含易混淆字符 (0OI1)
			assert.False(t, strings.Contains(code, "0"))
			assert.False(t, strings.Contains(code, "O"))
			assert.False(t, strings.Contains(code, "I"))
			assert.False(t, strings.Contains(code, "1"))

			// 验证只包含大写字母和数字
			for _, c := range code {
				valid := (c >= 'A' && c <= 'Z') || (c >= '2' && c <= '9')
				assert.True(t, valid, "邀请码应只包含大写字母和数字（排除0OI1）")
			}
		})
	}
}

func TestGenerateInviteCode_Uniqueness(t *testing.T) {
	length := 8
	iterations := 100
	seen := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		code := GenerateInviteCode(length)
		assert.False(t, seen[code], "邀请码应该是唯一的")
		seen[code] = true
	}
}

// ==================== ValidatePhone 测试 ====================

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{"Valid 13x", "13812345678", true},
		{"Valid 15x", "15812345678", true},
		{"Valid 18x", "18812345678", true},
		{"Valid 19x", "19812345678", true},
		{"Too short", "1381234567", false},
		{"Too long", "138123456789", false},
		{"Invalid prefix", "12812345678", false},
		{"Contains letters", "1381234567a", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePhone(tt.phone)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ==================== ValidateEmail 测试 ====================

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"Valid simple", "user@example.com", true},
		{"Valid with dot", "user.name@example.com", true},
		{"Valid with plus", "user+tag@example.com", true},
		{"Valid subdomain", "user@mail.example.com", true},
		{"No @ sign", "userexample.com", false},
		{"No domain", "user@", false},
		{"No local part", "@example.com", false},
		{"No TLD", "user@example", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateEmail(tt.email)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ==================== ValidateIDCard 测试 ====================

func TestValidateIDCard(t *testing.T) {
	tests := []struct {
		name   string
		idCard string
		want   bool
	}{
		{"Valid with number", "110101199001011234", true},
		{"Valid with X", "11010119900101123X", true},
		{"Valid with x", "11010119900101123x", true},
		{"Too short", "1101011990010112", false},
		{"Too long", "11010119900101123456", false},
		{"Contains letter", "11010119900101123A", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateIDCard(tt.idCard)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ==================== FormatMoney 测试 ====================

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		cents int64
		want  string
	}{
		{100, "1.00"},
		{150, "1.50"},
		{1, "0.01"},
		{1234, "12.34"},
		{0, "0.00"},
		{-100, "-1.00"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := FormatMoney(tt.cents)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ==================== ParseMoney 测试 ====================

func TestParseMoney(t *testing.T) {
	tests := []struct {
		yuan    string
		want    int64
		wantErr bool
	}{
		{"1.00", 100, false},
		{"1.50", 150, false},
		{"0.01", 1, false},
		{"12.34", 1234, false},
		{"0", 0, false},
		{"-1.00", -100, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.yuan, func(t *testing.T) {
			result, err := ParseMoney(tt.yuan)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestFormatParseMoney_RoundTrip(t *testing.T) {
	tests := []int64{100, 150, 1234, 0, -100}

	for _, cents := range tests {
		yuan := FormatMoney(cents)
		parsed, err := ParseMoney(yuan)
		require.NoError(t, err)
		assert.Equal(t, cents, parsed)
	}
}

// ==================== 指针函数测试 ====================

func TestStringPtr(t *testing.T) {
	s := "test"
	ptr := StringPtr(s)
	assert.NotNil(t, ptr)
	assert.Equal(t, s, *ptr)
}

func TestIntPtr(t *testing.T) {
	i := 123
	ptr := IntPtr(i)
	assert.NotNil(t, ptr)
	assert.Equal(t, i, *ptr)
}

func TestInt64Ptr(t *testing.T) {
	i := int64(12345)
	ptr := Int64Ptr(i)
	assert.NotNil(t, ptr)
	assert.Equal(t, i, *ptr)
}

func TestFloat64Ptr(t *testing.T) {
	f := 123.45
	ptr := Float64Ptr(f)
	assert.NotNil(t, ptr)
	assert.Equal(t, f, *ptr)
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	ptr := TimePtr(now)
	assert.NotNil(t, ptr)
	assert.Equal(t, now, *ptr)
}

// ==================== 安全取值函数测试 ====================

func TestSafeString(t *testing.T) {
	s := "test"
	assert.Equal(t, s, SafeString(&s))
	assert.Equal(t, "", SafeString(nil))
}

func TestSafeInt(t *testing.T) {
	i := 123
	assert.Equal(t, i, SafeInt(&i))
	assert.Equal(t, 0, SafeInt(nil))
}

func TestSafeInt64(t *testing.T) {
	i := int64(12345)
	assert.Equal(t, i, SafeInt64(&i))
	assert.Equal(t, int64(0), SafeInt64(nil))
}

// ==================== 泛型函数测试 ====================

func TestContains(t *testing.T) {
	t.Run("String slice", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.True(t, Contains(slice, "a"))
		assert.True(t, Contains(slice, "b"))
		assert.False(t, Contains(slice, "d"))
	})

	t.Run("Int slice", func(t *testing.T) {
		slice := []int{1, 2, 3}
		assert.True(t, Contains(slice, 1))
		assert.False(t, Contains(slice, 4))
	})

	t.Run("Empty slice", func(t *testing.T) {
		slice := []string{}
		assert.False(t, Contains(slice, "a"))
	})
}

func TestUnique(t *testing.T) {
	t.Run("String slice", func(t *testing.T) {
		slice := []string{"a", "b", "a", "c", "b"}
		result := Unique(slice)
		assert.Len(t, result, 3)
		assert.ElementsMatch(t, []string{"a", "b", "c"}, result)
	})

	t.Run("Int slice", func(t *testing.T) {
		slice := []int{1, 2, 1, 3, 2, 4}
		result := Unique(slice)
		assert.Len(t, result, 4)
		assert.ElementsMatch(t, []int{1, 2, 3, 4}, result)
	})

	t.Run("Empty slice", func(t *testing.T) {
		slice := []string{}
		result := Unique(slice)
		assert.Empty(t, result)
	})

	t.Run("No duplicates", func(t *testing.T) {
		slice := []int{1, 2, 3}
		result := Unique(slice)
		assert.Equal(t, slice, result)
	})
}

func TestMax(t *testing.T) {
	assert.Equal(t, 5, Max(5, 3))
	assert.Equal(t, 5, Max(3, 5))
	assert.Equal(t, 5, Max(5, 5))
	assert.Equal(t, int64(100), Max(int64(100), int64(50)))
	assert.Equal(t, 10.5, Max(10.5, 8.2))
}

func TestMin(t *testing.T) {
	assert.Equal(t, 3, Min(5, 3))
	assert.Equal(t, 3, Min(3, 5))
	assert.Equal(t, 5, Min(5, 5))
	assert.Equal(t, int64(50), Min(int64(100), int64(50)))
	assert.Equal(t, 8.2, Min(10.5, 8.2))
}

// ==================== Pagination 测试 ====================

func TestPagination_GetOffset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		want     int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 10, 20},
		{1, 20, 0},
		{5, 15, 60},
	}

	for _, tt := range tests {
		p := &Pagination{Page: tt.page, PageSize: tt.pageSize}
		assert.Equal(t, tt.want, p.GetOffset())
	}
}

func TestPagination_GetLimit(t *testing.T) {
	p := &Pagination{PageSize: 20}
	assert.Equal(t, 20, p.GetLimit())
}

func TestPagination_Normalize(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedPage   int
		expectedSize   int
	}{
		{"Normal", 2, 20, 2, 20},
		{"Page too small", 0, 20, 1, 20},
		{"Page negative", -1, 20, 1, 20},
		{"PageSize too small", 1, 0, 1, 10},
		{"PageSize too large", 1, 200, 1, 100},
		{"Both invalid", 0, 0, 1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pagination{Page: tt.page, PageSize: tt.pageSize}
			p.Normalize()
			assert.Equal(t, tt.expectedPage, p.Page)
			assert.Equal(t, tt.expectedSize, p.PageSize)
		})
	}
}

func TestPagination_GetTotalPages(t *testing.T) {
	tests := []struct {
		total    int64
		pageSize int
		want     int
	}{
		{100, 10, 10},
		{95, 10, 10},  // 向上取整
		{91, 10, 10},  // 向上取整
		{0, 10, 0},
		{5, 10, 1},
		{100, 20, 5},
	}

	for _, tt := range tests {
		p := &Pagination{Total: tt.total, PageSize: tt.pageSize}
		assert.Equal(t, tt.want, p.GetTotalPages())
	}
}

// ==================== 性能测试 ====================

func BenchmarkGenerateOrderNo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateOrderNo("O")
	}
}

func BenchmarkGenerateRandomNumber(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateRandomNumber(6)
	}
}

func BenchmarkGenerateInviteCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateInviteCode(8)
	}
}

func BenchmarkValidatePhone(b *testing.B) {
	phone := "13812345678"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePhone(phone)
	}
}

func BenchmarkValidateEmail(b *testing.B) {
	email := "user@example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateEmail(email)
	}
}
