// Package crypto 提供加密工具
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// AES 加密管理器
type AES struct {
	key []byte
}

// 预定义错误
var (
	ErrInvalidKeySize   = errors.New("invalid key size: must be 16, 24, or 32 bytes")
	ErrCiphertextShort  = errors.New("ciphertext too short")
	ErrDecryptionFailed = errors.New("decryption failed")
)

// NewAES 创建 AES 加密管理器
// key 长度必须是 16（AES-128）、24（AES-192）或 32（AES-256）字节
func NewAES(key string) (*AES, error) {
	keyBytes := []byte(key)
	keyLen := len(keyBytes)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, ErrInvalidKeySize
	}
	return &AES{key: keyBytes}, nil
}

// Encrypt 加密数据
func (a *AES) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", err
	}

	plaintextBytes := []byte(plaintext)
	ciphertext := make([]byte, aes.BlockSize+len(plaintextBytes))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintextBytes)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密数据
func (a *AES) Decrypt(ciphertext string) (string, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	if len(ciphertextBytes) < aes.BlockSize {
		return "", ErrCiphertextShort
	}

	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", err
	}

	iv := ciphertextBytes[:aes.BlockSize]
	ciphertextBytes = ciphertextBytes[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertextBytes, ciphertextBytes)

	return string(ciphertextBytes), nil
}

// HashPassword 对密码进行哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateRandomBytes 生成随机字节
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// MaskPhone 手机号脱敏
func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// MaskIDCard 身份证号脱敏
func MaskIDCard(idCard string) string {
	if len(idCard) != 18 {
		return idCard
	}
	return idCard[:6] + "********" + idCard[14:]
}

// MaskBankCard 银行卡号脱敏
func MaskBankCard(cardNo string) string {
	if len(cardNo) < 8 {
		return cardNo
	}
	return cardNo[:4] + " **** **** " + cardNo[len(cardNo)-4:]
}

// MaskEmail 邮箱脱敏
func MaskEmail(email string) string {
	for i, c := range email {
		if c == '@' {
			if i <= 2 {
				return email
			}
			return email[:2] + "***" + email[i:]
		}
	}
	return email
}
