// Package jwt 提供 JWT 令牌管理功能
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义 JWT 声明
type Claims struct {
	UserID   int64  `json:"user_id"`
	UserType string `json:"user_type"` // user, admin
	Role     string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// Config JWT 配置
type Config struct {
	Secret           string
	AccessExpireTime time.Duration
	RefreshExpireTime time.Duration
	Issuer           string
}

// Manager JWT 管理器
type Manager struct {
	config *Config
}

// TokenPair 令牌对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// 预定义错误
var (
	ErrTokenInvalid   = errors.New("invalid token")
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenMalformed = errors.New("token malformed")
	ErrTokenNotActive = errors.New("token not active yet")
)

// NewManager 创建 JWT 管理器
func NewManager(config *Config) *Manager {
	return &Manager{
		config: config,
	}
}

// GenerateTokenPair 生成令牌对
func (m *Manager) GenerateTokenPair(userID int64, userType, role string) (*TokenPair, error) {
	now := time.Now()
	accessExpireAt := now.Add(m.config.AccessExpireTime)
	refreshExpireAt := now.Add(m.config.RefreshExpireTime)

	// 生成访问令牌
	accessToken, err := m.generateToken(userID, userType, role, accessExpireAt)
	if err != nil {
		return nil, err
	}

	// 生成刷新令牌
	refreshToken, err := m.generateToken(userID, userType, role, refreshExpireAt)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpireAt.Unix(),
	}, nil
}

// GenerateAccessToken 生成访问令牌
func (m *Manager) GenerateAccessToken(userID int64, userType, role string) (string, int64, error) {
	expireAt := time.Now().Add(m.config.AccessExpireTime)
	token, err := m.generateToken(userID, userType, role, expireAt)
	return token, expireAt.Unix(), err
}

// generateToken 生成令牌
func (m *Manager) generateToken(userID int64, userType, role string, expireAt time.Time) (string, error) {
	claims := &Claims{
		UserID:   userID,
		UserType: userType,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   userType,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expireAt),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// ParseToken 解析令牌
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrTokenMalformed
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotActive
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// RefreshToken 刷新令牌
func (m *Manager) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := m.ParseToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	return m.GenerateTokenPair(claims.UserID, claims.UserType, claims.Role)
}

// ValidateToken 验证令牌
func (m *Manager) ValidateToken(tokenString string) (bool, error) {
	_, err := m.ParseToken(tokenString)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetUserIDFromToken 从令牌获取用户 ID
func (m *Manager) GetUserIDFromToken(tokenString string) (int64, error) {
	claims, err := m.ParseToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// UserType 用户类型常量
const (
	UserTypeUser  = "user"
	UserTypeAdmin = "admin"
)
