// Package tracing 提供 OpenTelemetry 分布式追踪
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config 追踪配置
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string // OTLP endpoint, empty for stdout
	SampleRate     float64
	Enabled        bool
}

// Tracer 追踪器包装
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	config   *Config
}

var defaultTracer *Tracer

// Init 初始化追踪器
func Init(cfg *Config) (*Tracer, error) {
	if cfg == nil {
		cfg = &Config{
			ServiceName: "smart-locker-backend",
			Environment: "development",
			SampleRate:  1.0,
			Enabled:     true,
		}
	}

	if !cfg.Enabled {
		defaultTracer = &Tracer{config: cfg}
		return defaultTracer, nil
	}

	// 创建资源
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("创建资源失败: %w", err)
	}

	// 创建导出器
	var exporter sdktrace.SpanExporter
	if cfg.Endpoint != "" {
		// 使用 OTLP gRPC 导出器
		client := otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
			otlptracegrpc.WithInsecure(),
		)
		exporter, err = otlptrace.New(context.Background(), client)
		if err != nil {
			return nil, fmt.Errorf("创建 OTLP 导出器失败: %w", err)
		}
	} else {
		// 使用 stdout 导出器（开发环境）
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("创建 stdout 导出器失败: %w", err)
		}
	}

	// 创建采样器
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// 创建 TracerProvider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(provider)

	// 设置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := &Tracer{
		provider: provider,
		tracer:   provider.Tracer(cfg.ServiceName),
		config:   cfg,
	}

	defaultTracer = tracer
	return tracer, nil
}

// GetTracer 获取默认追踪器
func GetTracer() *Tracer {
	return defaultTracer
}

// Shutdown 关闭追踪器
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// Start 开始一个新的 span
func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t.tracer == nil {
		return ctx, noopSpan{}
	}
	return t.tracer.Start(ctx, spanName, opts...)
}

// StartSpan 开始一个带属性的 span
func (t *Tracer) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if t.tracer == nil {
		return ctx, noopSpan{}
	}
	return t.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// SpanFromContext 从上下文获取当前 span
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent 添加事件到当前 span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetError 设置 span 错误
func SetError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// SetAttributes 设置 span 属性
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// noopSpan 空操作 span（追踪未启用时使用）
type noopSpan struct{}

func (noopSpan) End(...trace.SpanEndOption)                                 {}
func (noopSpan) AddEvent(string, ...trace.EventOption)                      {}
func (noopSpan) IsRecording() bool                                           { return false }
func (noopSpan) RecordError(error, ...trace.EventOption)                    {}
func (noopSpan) SpanContext() trace.SpanContext                              { return trace.SpanContext{} }
func (noopSpan) SetStatus(trace.Status)                                      {}
func (noopSpan) SetName(string)                                              {}
func (noopSpan) SetAttributes(...attribute.KeyValue)                         {}
func (noopSpan) TracerProvider() trace.TracerProvider                        { return nil }
func (noopSpan) AddLink(trace.Link)                                          {}

// 常用属性键
var (
	AttrUserID      = attribute.Key("user.id")
	AttrDeviceID    = attribute.Key("device.id")
	AttrOrderID     = attribute.Key("order.id")
	AttrMerchantID  = attribute.Key("merchant.id")
	AttrVenueID     = attribute.Key("venue.id")
	AttrOperation   = attribute.Key("operation")
	AttrDBTable     = attribute.Key("db.table")
	AttrDBOperation = attribute.Key("db.operation")
	AttrCacheKey    = attribute.Key("cache.key")
	AttrMQTTTopic   = attribute.Key("mqtt.topic")
)

// WithUserID 添加用户 ID 属性
func WithUserID(id int64) attribute.KeyValue {
	return AttrUserID.Int64(id)
}

// WithDeviceID 添加设备 ID 属性
func WithDeviceID(id int64) attribute.KeyValue {
	return AttrDeviceID.Int64(id)
}

// WithOrderID 添加订单 ID 属性
func WithOrderID(id int64) attribute.KeyValue {
	return AttrOrderID.Int64(id)
}

// WithOperation 添加操作属性
func WithOperation(op string) attribute.KeyValue {
	return AttrOperation.String(op)
}

// WithDBTable 添加数据库表属性
func WithDBTable(table string) attribute.KeyValue {
	return AttrDBTable.String(table)
}
