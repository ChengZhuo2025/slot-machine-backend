// Package middleware 提供 HTTP 中间件
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowOrigins     []string // 允许的源
	AllowMethods     []string // 允许的方法
	AllowHeaders     []string // 允许的头
	ExposeHeaders    []string // 暴露的头
	AllowCredentials bool     // 是否允许携带凭证
	MaxAge           int      // 预检请求缓存时间（秒）
}

// DefaultCORSConfig 默认 CORS 配置
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Request-ID",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 小时
	}
}

// CORS 跨域中间件
func CORS(config *CORSConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCORSConfig()
	}

	allowAllOrigins := len(config.AllowOrigins) == 1 && config.AllowOrigins[0] == "*"
	allowOriginSet := make(map[string]struct{})
	for _, origin := range config.AllowOrigins {
		allowOriginSet[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否允许该源
		var allowOrigin string
		if allowAllOrigins {
			if config.AllowCredentials {
				allowOrigin = origin
			} else {
				allowOrigin = "*"
			}
		} else {
			if _, ok := allowOriginSet[origin]; ok {
				allowOrigin = origin
			}
		}

		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))

			if len(config.ExposeHeaders) > 0 {
				c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
			}

			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", string(rune(config.MaxAge)))
			}
		}

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORSWithOrigins 指定源的 CORS 中间件
func CORSWithOrigins(origins ...string) gin.HandlerFunc {
	config := DefaultCORSConfig()
	config.AllowOrigins = origins
	return CORS(config)
}

// CORSAllowAll 允许所有源的 CORS 中间件
func CORSAllowAll() gin.HandlerFunc {
	return CORS(DefaultCORSConfig())
}
