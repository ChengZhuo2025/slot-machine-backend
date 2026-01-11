// Package crypto 加密工具单元测试
package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== AES 加密/解密测试 ====================

func TestNewAES_ValidKey(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		keyLen int
	}{
		{"AES-128", "1234567890123456", 16},
		{"AES-192", "123456789012345678901234", 24},
		{"AES-256", "12345678901234567890123456789012", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aes, err := NewAES(tt.key)
			require.NoError(t, err)
			assert.NotNil(t, aes)
			assert.Equal(t, tt.keyLen, len(aes.key))
		})
	}
}

func TestNewAES_InvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"Too short", "12345"},
		{"Invalid length", "12345678901234567"},
		{"Empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aes, err := NewAES(tt.key)
			assert.Error(t, err)
			assert.Equal(t, ErrInvalidKeySize, err)
			assert.Nil(t, aes)
		})
	}
}

func TestAES_EncryptDecrypt_Success(t *testing.T) {
	key := "12345678901234567890123456789012" // 32 bytes for AES-256
	aes, err := NewAES(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"Simple text", "Hello, World!"},
		{"Chinese text", "你好，世界！"},
		{"Empty string", ""},
		{"Long text", strings.Repeat("测试数据", 100)},
		{"Special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"Numbers", "1234567890"},
		{"Mixed", "Test123测试!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			ciphertext, err := aes.Encrypt(tt.plaintext)
			require.NoError(t, err)
			assert.NotEmpty(t, ciphertext)
			assert.NotEqual(t, tt.plaintext, ciphertext)

			// 解密
			decrypted, err := aes.Decrypt(ciphertext)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestAES_Encrypt_RandomIV(t *testing.T) {
	key := "12345678901234567890123456789012"
	aes, err := NewAES(key)
	require.NoError(t, err)

	plaintext := "Test data"

	// 多次加密同一数据应该产生不同的密文（因为IV是随机的）
	ciphertext1, err := aes.Encrypt(plaintext)
	require.NoError(t, err)

	ciphertext2, err := aes.Encrypt(plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, ciphertext1, ciphertext2, "相同明文加密应产生不同密文（随机IV）")

	// 但都应该能正确解密
	decrypted1, err := aes.Decrypt(ciphertext1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := aes.Decrypt(ciphertext2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

func TestAES_Decrypt_InvalidCiphertext(t *testing.T) {
	key := "12345678901234567890123456789012"
	aes, err := NewAES(key)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext string
		wantErr    error
	}{
		{"Invalid base64", "not-a-valid-base64!", nil}, // base64解码会失败
		{"Too short", "YWJj", ErrCiphertextShort},      // 短于 AES block size
		{"Empty", "", nil},                              // base64解码空字符串
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := aes.Decrypt(tt.ciphertext)
			assert.Error(t, err)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			}
		})
	}
}

func TestAES_Decrypt_WrongKey(t *testing.T) {
	key1 := "12345678901234567890123456789012"
	key2 := "abcdefghijklmnopqrstuvwxyz123456"

	aes1, err := NewAES(key1)
	require.NoError(t, err)

	aes2, err := NewAES(key2)
	require.NoError(t, err)

	plaintext := "Secret data"

	// 用 key1 加密
	ciphertext, err := aes1.Encrypt(plaintext)
	require.NoError(t, err)

	// 用 key2 解密应该得到错误的数据
	decrypted, err := aes2.Decrypt(ciphertext)
	require.NoError(t, err) // 不会报错，但数据不正确
	assert.NotEqual(t, plaintext, decrypted)
}

// ==================== 密码哈希测试 ====================

func TestHashPassword_Success(t *testing.T) {
	passwords := []string{
		"password123",
		"StrongP@ssw0rd!",
		"简单密码",
		"12345678",
		strings.Repeat("x", 72), // bcrypt最大长度
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			hash, err := HashPassword(password)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, password, hash)
			assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"))
		})
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "password123"

	// 多次哈希同一密码应该产生不同的哈希值（因为salt是随机的）
	hash1, err := HashPassword(password)
	require.NoError(t, err)

	hash2, err := HashPassword(password)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "相同密码应产生不同哈希值（随机salt）")

	// 但都应该能验证通过
	assert.True(t, VerifyPassword(password, hash1))
	assert.True(t, VerifyPassword(password, hash2))
}

func TestVerifyPassword_Success(t *testing.T) {
	password := "MySecretPassword123!"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	assert.True(t, VerifyPassword(password, hash))
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	password := "correct_password"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	tests := []string{
		"wrong_password",
		"Correct_password", // 大小写敏感
		"correct_passwor",  // 少一个字符
		"correct_password ", // 多一个空格
		"",
	}

	for _, wrongPassword := range tests {
		t.Run(wrongPassword, func(t *testing.T) {
			assert.False(t, VerifyPassword(wrongPassword, hash))
		})
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	assert.False(t, VerifyPassword("password", "invalid-hash"))
	assert.False(t, VerifyPassword("password", ""))
	assert.False(t, VerifyPassword("password", "$2a$10$invalid"))
}

// ==================== 随机字符串/字节测试 ====================

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"Length 8", 8},
		{"Length 16", 16},
		{"Length 32", 32},
		{"Length 64", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, err := GenerateRandomString(tt.length)
			require.NoError(t, err)
			assert.Equal(t, tt.length, len(str))
			assert.NotEmpty(t, str)
		})
	}
}

func TestGenerateRandomString_Uniqueness(t *testing.T) {
	length := 16
	iterations := 100
	seen := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		str, err := GenerateRandomString(length)
		require.NoError(t, err)
		assert.False(t, seen[str], "生成的随机字符串应该是唯一的")
		seen[str] = true
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	tests := []int{8, 16, 32, 64, 128}

	for _, n := range tests {
		t.Run(string(rune(n)), func(t *testing.T) {
			bytes, err := GenerateRandomBytes(n)
			require.NoError(t, err)
			assert.Equal(t, n, len(bytes))
		})
	}
}

func TestGenerateRandomBytes_Uniqueness(t *testing.T) {
	n := 16
	iterations := 100
	seen := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		bytes, err := GenerateRandomBytes(n)
		require.NoError(t, err)
		key := string(bytes)
		assert.False(t, seen[key], "生成的随机字节应该是唯一的")
		seen[key] = true
	}
}

// ==================== 数据脱敏测试 ====================

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{"Valid phone", "13812345678", "138****5678"},
		{"Another valid phone", "18600001111", "186****1111"},
		{"Too short", "1234567", "1234567"},
		{"Too long", "138123456789", "138123456789"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskPhone(tt.phone)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskIDCard(t *testing.T) {
	tests := []struct {
		name     string
		idCard   string
		expected string
	}{
		{"Valid ID", "110101199001011234", "110101********1234"},
		{"Another valid ID", "320102198501012345", "320102********2345"},
		{"Too short", "1234567890", "1234567890"},
		{"Too long", "1234567890123456789", "1234567890123456789"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskIDCard(tt.idCard)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskBankCard(t *testing.T) {
	tests := []struct {
		name     string
		cardNo   string
		expected string
	}{
		{"16-digit card", "6222021234567890", "6222 **** **** 7890"},
		{"19-digit card", "6222021234567890123", "6222 **** **** 0123"},
		{"Short card", "123456", "123456"},
		{"Minimum length", "12345678", "1234 **** **** 5678"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskBankCard(tt.cardNo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{"Normal email", "user@example.com", "us***@example.com"},
		{"Short email", "ab@test.com", "ab@test.com"}, // 太短不脱敏
		{"Long email", "verylongemail@test.com", "ve***@test.com"},
		{"No @ sign", "notanemail", "notanemail"},
		{"Empty", "", ""},
		{"Single char before @", "a@test.com", "a@test.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskEmail(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== 性能测试 ====================

func BenchmarkAESEncrypt(b *testing.B) {
	key := "12345678901234567890123456789012"
	aes, _ := NewAES(key)
	plaintext := "This is a test message for encryption benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = aes.Encrypt(plaintext)
	}
}

func BenchmarkAESDecrypt(b *testing.B) {
	key := "12345678901234567890123456789012"
	aes, _ := NewAES(key)
	plaintext := "This is a test message for encryption benchmark"
	ciphertext, _ := aes.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = aes.Decrypt(ciphertext)
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "MySecretPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "MySecretPassword123!"
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyPassword(password, hash)
	}
}
