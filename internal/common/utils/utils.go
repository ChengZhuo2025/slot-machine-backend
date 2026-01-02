// Package utils 提供通用工具函数
package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GenerateOrderNo 生成订单号
// 格式: 前缀 + 年月日时分秒 + 6位随机数
func GenerateOrderNo(prefix string) string {
	now := time.Now()
	timestamp := now.Format("20060102150405")
	random := GenerateRandomNumber(6)
	return fmt.Sprintf("%s%s%s", prefix, timestamp, random)
}

// GenerateRandomNumber 生成指定长度的随机数字字符串
func GenerateRandomNumber(length int) string {
	var result strings.Builder
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		result.WriteString(strconv.Itoa(int(n.Int64())))
	}
	return result.String()
}

// GenerateRandomCode 生成随机验证码
func GenerateRandomCode(length int) string {
	return GenerateRandomNumber(length)
}

// GenerateInviteCode 生成邀请码
func GenerateInviteCode(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除易混淆字符 0OI1
	var result strings.Builder
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result.WriteByte(charset[n.Int64()])
	}
	return result.String()
}

// ValidatePhone 验证手机号
func ValidatePhone(phone string) bool {
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// ValidateEmail 验证邮箱
func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ValidateIDCard 验证身份证号
func ValidateIDCard(idCard string) bool {
	if len(idCard) != 18 {
		return false
	}
	// 简化验证，实际应包含校验码验证
	pattern := `^\d{17}[\dXx]$`
	matched, _ := regexp.MatchString(pattern, idCard)
	return matched
}

// FormatMoney 格式化金额（分转元）
func FormatMoney(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

// ParseMoney 解析金额（元转分）
func ParseMoney(yuan string) (int64, error) {
	f, err := strconv.ParseFloat(yuan, 64)
	if err != nil {
		return 0, err
	}
	return int64(f * 100), nil
}

// StringPtr 返回字符串指针
func StringPtr(s string) *string {
	return &s
}

// IntPtr 返回整数指针
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr 返回 int64 指针
func Int64Ptr(i int64) *int64 {
	return &i
}

// Float64Ptr 返回 float64 指针
func Float64Ptr(f float64) *float64 {
	return &f
}

// TimePtr 返回时间指针
func TimePtr(t time.Time) *time.Time {
	return &t
}

// SafeString 安全获取字符串指针的值
func SafeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SafeInt 安全获取整数指针的值
func SafeInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// SafeInt64 安全获取 int64 指针的值
func SafeInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// Contains 判断切片是否包含元素
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Unique 切片去重
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Max 返回两个数中的较大值
func Max[T int | int64 | float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Min 返回两个数中的较小值
func Min[T int | int64 | float64](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Pagination 分页参数
type Pagination struct {
	Page     int   `json:"page" form:"page"`
	PageSize int   `json:"page_size" form:"page_size"`
	Total    int64 `json:"total"`
}

// GetOffset 获取偏移量
func (p *Pagination) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// GetLimit 获取限制数
func (p *Pagination) GetLimit() int {
	return p.PageSize
}

// Normalize 规范化分页参数
func (p *Pagination) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

// GetTotalPages 获取总页数
func (p *Pagination) GetTotalPages() int {
	if p.Total == 0 {
		return 0
	}
	pages := int(p.Total) / p.PageSize
	if int(p.Total)%p.PageSize > 0 {
		pages++
	}
	return pages
}
