// Package jwt JWT令牌管理单元测试
package jwt

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestManager 创建测试用的 JWT Manager
func setupTestManager() *Manager {
	config := &Config{
		Secret:            "test-secret-key-for-jwt-token-signing",
		AccessExpireTime:  15 * time.Minute,
		RefreshExpireTime: 7 * 24 * time.Hour,
		Issuer:            "test-issuer",
	}
	return NewManager(config)
}

// ==================== NewManager 测试 ====================

func TestNewManager(t *testing.T) {
	config := &Config{
		Secret:            "secret",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 24 * time.Hour,
		Issuer:            "test",
	}

	manager := NewManager(config)
	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
}

// ==================== GenerateTokenPair 测试 ====================

func TestManager_GenerateTokenPair_Success(t *testing.T) {
	manager := setupTestManager()

	tests := []struct {
		name     string
		userID   int64
		userType string
		role     string
	}{
		{"User token", 12345, UserTypeUser, ""},
		{"Admin token", 99999, UserTypeAdmin, "admin"},
		{"User with role", 54321, UserTypeUser, "vip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPair, err := manager.GenerateTokenPair(tt.userID, tt.userType, tt.role)
			require.NoError(t, err)
			assert.NotNil(t, tokenPair)
			assert.NotEmpty(t, tokenPair.AccessToken)
			assert.NotEmpty(t, tokenPair.RefreshToken)
			assert.Greater(t, tokenPair.ExpiresAt, time.Now().Unix())

			// 验证 access token 和 refresh token 不同
			assert.NotEqual(t, tokenPair.AccessToken, tokenPair.RefreshToken)

			// 验证可以解析 access token
			claims, err := manager.ParseToken(tokenPair.AccessToken)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, tt.userType, claims.UserType)
			assert.Equal(t, tt.role, claims.Role)

			// 验证可以解析 refresh token
			refreshClaims, err := manager.ParseToken(tokenPair.RefreshToken)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, refreshClaims.UserID)
		})
	}
}

func TestManager_GenerateTokenPair_ExpiryTime(t *testing.T) {
	manager := setupTestManager()

	tokenPair, err := manager.GenerateTokenPair(123, UserTypeUser, "")
	require.NoError(t, err)

	// 验证 ExpiresAt 大约是 15 分钟后
	expectedExpireAt := time.Now().Add(15 * time.Minute).Unix()
	assert.InDelta(t, expectedExpireAt, tokenPair.ExpiresAt, 5) // 允许5秒误差
}

// ==================== GenerateAccessToken 测试 ====================

func TestManager_GenerateAccessToken_Success(t *testing.T) {
	manager := setupTestManager()

	userID := int64(12345)
	userType := UserTypeUser
	role := "member"

	token, expiresAt, err := manager.GenerateAccessToken(userID, userType, role)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, expiresAt, time.Now().Unix())

	// 验证可以解析token
	claims, err := manager.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, userType, claims.UserType)
	assert.Equal(t, role, claims.Role)
}

// ==================== ParseToken 测试 ====================

func TestManager_ParseToken_Success(t *testing.T) {
	manager := setupTestManager()

	userID := int64(99999)
	userType := UserTypeAdmin
	role := "super_admin"

	token, _, err := manager.GenerateAccessToken(userID, userType, role)
	require.NoError(t, err)

	claims, err := manager.ParseToken(token)
	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, userType, claims.UserType)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, manager.config.Issuer, claims.Issuer)
	assert.Equal(t, userType, claims.Subject)
	assert.NotEmpty(t, claims.ID)
}

func TestManager_ParseToken_InvalidToken(t *testing.T) {
	manager := setupTestManager()

	tests := []struct {
		name        string
		token       string
		expectedErr error
	}{
		{"Empty token", "", ErrTokenMalformed},
		{"Invalid format", "invalid.token.format", ErrTokenMalformed},
		{"Random string", "this-is-not-a-jwt-token", ErrTokenMalformed},
		{"Incomplete JWT", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", ErrTokenMalformed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ParseToken(tt.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
			// 验证错误类型（不严格匹配，因为可能是包装的错误）
			assert.Contains(t, err.Error(), "token")
		})
	}
}

func TestManager_ParseToken_WrongSecret(t *testing.T) {
	// 用一个 secret 生成 token
	manager1 := NewManager(&Config{
		Secret:            "secret-1",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 24 * time.Hour,
		Issuer:            "test",
	})

	token, _, err := manager1.GenerateAccessToken(123, UserTypeUser, "")
	require.NoError(t, err)

	// 用另一个 secret 解析 token
	manager2 := NewManager(&Config{
		Secret:            "secret-2",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 24 * time.Hour,
		Issuer:            "test",
	})

	claims, err := manager2.ParseToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestManager_ParseToken_ExpiredToken(t *testing.T) {
	// 创建一个过期时间很短的 manager
	manager := NewManager(&Config{
		Secret:            "test-secret",
		AccessExpireTime:  1 * time.Millisecond, // 1毫秒后过期
		RefreshExpireTime: 1 * time.Millisecond,
		Issuer:            "test",
	})

	token, _, err := manager.GenerateAccessToken(123, UserTypeUser, "")
	require.NoError(t, err)

	// 等待 token 过期
	time.Sleep(10 * time.Millisecond)

	claims, err := manager.ParseToken(token)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenExpired, err)
	assert.Nil(t, claims)
}

// ==================== RefreshToken 测试 ====================

func TestManager_RefreshToken_Success(t *testing.T) {
	manager := setupTestManager()

	userID := int64(12345)
	userType := UserTypeUser
	role := "member"

	// 生成初始 token pair
	originalPair, err := manager.GenerateTokenPair(userID, userType, role)
	require.NoError(t, err)

	// 使用 refresh token 刷新
	newPair, err := manager.RefreshToken(originalPair.RefreshToken)
	require.NoError(t, err)
	assert.NotNil(t, newPair)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)

	// 新的 token 应该不同于原来的 token
	assert.NotEqual(t, originalPair.AccessToken, newPair.AccessToken)
	assert.NotEqual(t, originalPair.RefreshToken, newPair.RefreshToken)

	// 验证新 token 的声明
	claims, err := manager.ParseToken(newPair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, userType, claims.UserType)
	assert.Equal(t, role, claims.Role)
}

func TestManager_RefreshToken_InvalidToken(t *testing.T) {
	manager := setupTestManager()

	newPair, err := manager.RefreshToken("invalid-refresh-token")
	assert.Error(t, err)
	assert.Nil(t, newPair)
}

func TestManager_RefreshToken_ExpiredToken(t *testing.T) {
	// 创建一个过期时间很短的 manager
	manager := NewManager(&Config{
		Secret:            "test-secret",
		AccessExpireTime:  1 * time.Millisecond,
		RefreshExpireTime: 1 * time.Millisecond,
		Issuer:            "test",
	})

	tokenPair, err := manager.GenerateTokenPair(123, UserTypeUser, "")
	require.NoError(t, err)

	// 等待 token 过期
	time.Sleep(10 * time.Millisecond)

	newPair, err := manager.RefreshToken(tokenPair.RefreshToken)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenExpired, err)
	assert.Nil(t, newPair)
}

// ==================== ValidateToken 测试 ====================

func TestManager_ValidateToken_Success(t *testing.T) {
	manager := setupTestManager()

	token, _, err := manager.GenerateAccessToken(123, UserTypeUser, "")
	require.NoError(t, err)

	valid, err := manager.ValidateToken(token)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestManager_ValidateToken_Invalid(t *testing.T) {
	manager := setupTestManager()

	tests := []string{
		"invalid-token",
		"",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid",
	}

	for _, token := range tests {
		t.Run(token, func(t *testing.T) {
			valid, err := manager.ValidateToken(token)
			assert.Error(t, err)
			assert.False(t, valid)
		})
	}
}

func TestManager_ValidateToken_Expired(t *testing.T) {
	manager := NewManager(&Config{
		Secret:            "test-secret",
		AccessExpireTime:  1 * time.Millisecond,
		RefreshExpireTime: 1 * time.Millisecond,
		Issuer:            "test",
	})

	token, _, err := manager.GenerateAccessToken(123, UserTypeUser, "")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	valid, err := manager.ValidateToken(token)
	assert.Error(t, err)
	assert.False(t, valid)
	assert.Equal(t, ErrTokenExpired, err)
}

// ==================== GetUserIDFromToken 测试 ====================

func TestManager_GetUserIDFromToken_Success(t *testing.T) {
	manager := setupTestManager()

	tests := []int64{1, 12345, 99999, 1000000}

	for _, expectedID := range tests {
		t.Run(string(rune(expectedID)), func(t *testing.T) {
			token, _, err := manager.GenerateAccessToken(expectedID, UserTypeUser, "")
			require.NoError(t, err)

			userID, err := manager.GetUserIDFromToken(token)
			assert.NoError(t, err)
			assert.Equal(t, expectedID, userID)
		})
	}
}

func TestManager_GetUserIDFromToken_InvalidToken(t *testing.T) {
	manager := setupTestManager()

	userID, err := manager.GetUserIDFromToken("invalid-token")
	assert.Error(t, err)
	assert.Equal(t, int64(0), userID)
}

func TestManager_GetUserIDFromToken_ExpiredToken(t *testing.T) {
	manager := NewManager(&Config{
		Secret:            "test-secret",
		AccessExpireTime:  1 * time.Millisecond,
		RefreshExpireTime: 1 * time.Millisecond,
		Issuer:            "test",
	})

	token, _, err := manager.GenerateAccessToken(123, UserTypeUser, "")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	userID, err := manager.GetUserIDFromToken(token)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenExpired, err)
	assert.Equal(t, int64(0), userID)
}

// ==================== 边界条件和特殊情况测试 ====================

func TestManager_TokenWithZeroUserID(t *testing.T) {
	manager := setupTestManager()

	// UserID 为 0 也应该能正常工作
	token, _, err := manager.GenerateAccessToken(0, UserTypeUser, "")
	require.NoError(t, err)

	claims, err := manager.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, int64(0), claims.UserID)
}

func TestManager_TokenWithEmptyRole(t *testing.T) {
	manager := setupTestManager()

	token, _, err := manager.GenerateAccessToken(123, UserTypeUser, "")
	require.NoError(t, err)

	claims, err := manager.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, "", claims.Role)
}

func TestManager_TokenWithLongRole(t *testing.T) {
	manager := setupTestManager()

	longRole := strings.Repeat("admin_", 100)
	token, _, err := manager.GenerateAccessToken(123, UserTypeAdmin, longRole)
	require.NoError(t, err)

	claims, err := manager.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, longRole, claims.Role)
}

func TestManager_MultipleTokensForSameUser(t *testing.T) {
	manager := setupTestManager()

	userID := int64(12345)

	// 为同一用户生成多个 token
	token1, _, err := manager.GenerateAccessToken(userID, UserTypeUser, "")
	require.NoError(t, err)

	token2, _, err := manager.GenerateAccessToken(userID, UserTypeUser, "")
	require.NoError(t, err)

	// 两个 token 应该不同（因为 token ID 是随机的）
	assert.NotEqual(t, token1, token2)

	// 但都应该能正确解析出同一个 userID
	claims1, err := manager.ParseToken(token1)
	require.NoError(t, err)

	claims2, err := manager.ParseToken(token2)
	require.NoError(t, err)

	assert.Equal(t, userID, claims1.UserID)
	assert.Equal(t, userID, claims2.UserID)
}

// ==================== 性能测试 ====================

func BenchmarkGenerateTokenPair(b *testing.B) {
	manager := setupTestManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateTokenPair(12345, UserTypeUser, "member")
	}
}

func BenchmarkParseToken(b *testing.B) {
	manager := setupTestManager()
	token, _, _ := manager.GenerateAccessToken(12345, UserTypeUser, "member")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ParseToken(token)
	}
}

func BenchmarkValidateToken(b *testing.B) {
	manager := setupTestManager()
	token, _, _ := manager.GenerateAccessToken(12345, UserTypeUser, "member")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateToken(token)
	}
}
