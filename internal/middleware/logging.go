// Package middleware 提供 HTTP 中间件
package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingConfig 日志配置
type LoggingConfig struct {
	Logger          *zap.Logger
	SkipPaths       []string // 跳过日志的路径
	SkipHealthCheck bool     // 跳过健康检查接口
	LogRequestBody  bool     // 是否记录请求体
	LogResponseBody bool     // 是否记录响应体
	MaxBodySize     int      // 最大记录的 body 大小
}

// DefaultLoggingConfig 默认日志配置
func DefaultLoggingConfig(logger *zap.Logger) *LoggingConfig {
	return &LoggingConfig{
		Logger:          logger,
		SkipPaths:       []string{},
		SkipHealthCheck: true,
		LogRequestBody:  false,
		LogResponseBody: false,
		MaxBodySize:     1024,
	}
}

// responseWriter 响应写入器包装
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logging 请求日志中间件
func Logging(config *LoggingConfig) gin.HandlerFunc {
	skipPaths := make(map[string]struct{})
	for _, path := range config.SkipPaths {
		skipPaths[path] = struct{}{}
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 跳过指定路径
		if _, ok := skipPaths[path]; ok {
			c.Next()
			return
		}

		// 跳过健康检查
		if config.SkipHealthCheck && (path == "/health" || path == "/ping" || path == "/ready") {
			c.Next()
			return
		}

		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = c.GetString(ContextKeyRequestID)
		}

		// 记录请求体
		var requestBody string
		if config.LogRequestBody && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if len(bodyBytes) > 0 {
				if len(bodyBytes) > config.MaxBodySize {
					requestBody = string(bodyBytes[:config.MaxBodySize]) + "...(truncated)"
				} else {
					requestBody = string(bodyBytes)
				}
			}
		}

		// 包装响应写入器
		var blw *responseWriter
		if config.LogResponseBody {
			blw = &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
			c.Writer = blw
		}

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// 构建日志字段
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// 添加用户 ID
		if userID := GetUserID(c); userID > 0 {
			fields = append(fields, zap.Int64("user_id", userID))
		}

		// 添加请求体
		if requestBody != "" {
			fields = append(fields, zap.String("request_body", requestBody))
		}

		// 添加响应体
		if config.LogResponseBody && blw != nil {
			responseBody := blw.body.String()
			if len(responseBody) > config.MaxBodySize {
				responseBody = responseBody[:config.MaxBodySize] + "...(truncated)"
			}
			fields = append(fields, zap.String("response_body", responseBody))
		}

		// 添加错误信息
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// 根据状态码选择日志级别
		switch {
		case statusCode >= 500:
			config.Logger.Error("HTTP Request", fields...)
		case statusCode >= 400:
			config.Logger.Warn("HTTP Request", fields...)
		default:
			config.Logger.Info("HTTP Request", fields...)
		}
	}
}

// AccessLog 简化的访问日志中间件
func AccessLog(logger *zap.Logger) gin.HandlerFunc {
	return Logging(DefaultLoggingConfig(logger))
}
