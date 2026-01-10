// Package middleware 提供 HTTP 中间件
package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig 追踪中间件配置
type TracingConfig struct {
	ServiceName string
	SkipPaths   []string
}

// Tracing 返回追踪中间件
func Tracing(cfg *TracingConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = &TracingConfig{
			ServiceName: "smart-locker-backend",
		}
	}

	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	tracer := otel.Tracer(cfg.ServiceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// 跳过指定路径
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 从请求头提取追踪上下文
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 创建 span 名称
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		if c.FullPath() == "" {
			spanName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}

		// 开始 span
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Request.Method),
				semconv.HTTPTarget(c.Request.URL.Path),
				semconv.HTTPScheme(c.Request.URL.Scheme),
				semconv.NetHostName(c.Request.Host),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
			),
		)
		defer span.End()

		// 更新请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 处理请求
		c.Next()

		// 记录响应状态
		status := c.Writer.Status()
		span.SetAttributes(
			semconv.HTTPStatusCode(status),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// 记录错误
		if len(c.Errors) > 0 {
			span.SetAttributes(attribute.String("http.error", c.Errors.String()))
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}

		// 设置 span 状态
		if status >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}

// InjectTraceContext 注入追踪上下文到响应头
func InjectTraceContext() gin.HandlerFunc {
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		c.Next()

		// 注入追踪上下文到响应头
		propagator.Inject(c.Request.Context(), propagation.HeaderCarrier(c.Writer.Header()))
	}
}

// GetTraceID 从上下文获取追踪 ID
func GetTraceID(c *gin.Context) string {
	span := trace.SpanFromContext(c.Request.Context())
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID 从上下文获取 Span ID
func GetSpanID(c *gin.Context) string {
	span := trace.SpanFromContext(c.Request.Context())
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// AddSpanEvent 添加 span 事件
func AddSpanEvent(c *gin.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(c.Request.Context())
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanAttributes 设置 span 属性
func SetSpanAttributes(c *gin.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(attrs...)
}

// RecordSpanError 记录 span 错误
func RecordSpanError(c *gin.Context, err error) {
	span := trace.SpanFromContext(c.Request.Context())
	span.RecordError(err)
}
