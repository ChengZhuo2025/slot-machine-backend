// Package middleware 提供 HTTP 中间件
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
)

// 上下文键
const (
	ContextKeyRequestID = "request_id"
)

// RequestID 请求 ID 中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用请求头中的 ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 设置到上下文和响应头
		c.Set(ContextKeyRequestID, requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// GetRequestID 获取请求 ID
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(ContextKeyRequestID); exists {
		return requestID.(string)
	}
	return ""
}

// Recovery 恢复中间件
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取堆栈信息
				stack := string(debug.Stack())

				// 记录日志
				logger.Error("Panic recovered",
					zap.String("request_id", GetRequestID(c)),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
					zap.Any("error", err),
					zap.String("stack", stack),
				)

				// 返回错误响应
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
					Code:    500,
					Message: "服务器内部错误",
				})
			}
		}()

		c.Next()
	}
}

// Timeout 超时中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用通道来处理超时
		done := make(chan struct{})

		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// 正常完成
		case <-time.After(timeout):
			// 超时
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, response.Response{
				Code:    504,
				Message: "请求超时",
			})
		}
	}
}

// SecureHeaders 安全头中间件
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")
		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")
		// XSS 保护
		c.Header("X-XSS-Protection", "1; mode=block")
		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}

// NoCache 禁用缓存中间件
func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Next()
	}
}

// RealIP 真实 IP 中间件
func RealIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从 X-Real-IP 获取
		realIP := c.GetHeader("X-Real-IP")
		if realIP != "" {
			c.Request.RemoteAddr = realIP
		} else {
			// 其次从 X-Forwarded-For 获取第一个 IP
			xff := c.GetHeader("X-Forwarded-For")
			if xff != "" {
				// X-Forwarded-For 格式: client, proxy1, proxy2
				for i := 0; i < len(xff); i++ {
					if xff[i] == ',' {
						c.Request.RemoteAddr = xff[:i]
						break
					}
				}
				if c.Request.RemoteAddr == xff {
					c.Request.RemoteAddr = xff
				}
			}
		}

		c.Next()
	}
}

// RequestSizeLimiter 请求大小限制中间件
func RequestSizeLimiter(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			response.BadRequest(c, fmt.Sprintf("请求体过大，最大允许 %d 字节", maxSize))
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}
