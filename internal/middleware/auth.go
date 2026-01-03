// Package middleware 提供 HTTP 中间件
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
)

// AuthConfig 认证配置
type AuthConfig struct {
	JWTManager *jwt.Manager
	UserType   string // 期望的用户类型
}

// 上下文键
const (
	ContextKeyUserID   = "user_id"
	ContextKeyUserType = "user_type"
	ContextKeyRole     = "role"
	ContextKeyClaims   = "claims"
)

// Auth 认证中间件
func Auth(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		claims, err := config.JWTManager.ParseToken(token)
		if err != nil {
			if err == jwt.ErrTokenExpired {
				response.Unauthorized(c, "登录已过期，请重新登录")
			} else {
				response.Unauthorized(c, "无效的令牌")
			}
			c.Abort()
			return
		}

		// 验证用户类型
		if config.UserType != "" && claims.UserType != config.UserType {
			response.Forbidden(c, "无权访问")
			c.Abort()
			return
		}

		// 设置上下文
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserType, claims.UserType)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyClaims, claims)

		c.Next()
	}
}

// OptionalAuth 可选认证中间件（不强制要求登录）
func OptionalAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token != "" {
			claims, err := jwtManager.ParseToken(token)
			if err == nil {
				c.Set(ContextKeyUserID, claims.UserID)
				c.Set(ContextKeyUserType, claims.UserType)
				c.Set(ContextKeyRole, claims.Role)
				c.Set(ContextKeyClaims, claims)
			}
		}
		c.Next()
	}
}

// UserAuth 用户认证中间件
func UserAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return Auth(&AuthConfig{
		JWTManager: jwtManager,
		UserType:   jwt.UserTypeUser,
	})
}

// AdminAuth 管理员认证中间件
func AdminAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return Auth(&AuthConfig{
		JWTManager: jwtManager,
		UserType:   jwt.UserTypeAdmin,
	})
}

// extractToken 从请求中提取令牌
func extractToken(c *gin.Context) string {
	// 优先从 Authorization 头获取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// 其次从查询参数获取
	token := c.Query("token")
	if token != "" {
		return token
	}

	// 最后从 Cookie 获取
	token, _ = c.Cookie("token")
	return token
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) int64 {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return 0
	}
	return userID.(int64)
}

// GetUserType 从上下文获取用户类型
func GetUserType(c *gin.Context) string {
	userType, exists := c.Get(ContextKeyUserType)
	if !exists {
		return ""
	}
	return userType.(string)
}

// GetRole 从上下文获取角色
func GetRole(c *gin.Context) string {
	role, exists := c.Get(ContextKeyRole)
	if !exists {
		return ""
	}
	return role.(string)
}

// GetClaims 从上下文获取完整的 Claims
func GetClaims(c *gin.Context) *jwt.Claims {
	claims, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil
	}
	return claims.(*jwt.Claims)
}

// IsLoggedIn 判断是否已登录
func IsLoggedIn(c *gin.Context) bool {
	_, exists := c.Get(ContextKeyUserID)
	return exists
}
