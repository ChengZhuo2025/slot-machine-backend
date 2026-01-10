// Package tracing 提供 OpenTelemetry 分布式追踪单元测试
package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestInit(t *testing.T) {
	t.Run("使用默认配置", func(t *testing.T) {
		tracer, err := Init(nil)
		require.NoError(t, err)
		require.NotNil(t, tracer)
		assert.NotNil(t, tracer.config)
		assert.Equal(t, "smart-locker-backend", tracer.config.ServiceName)
	})

	t.Run("使用自定义配置", func(t *testing.T) {
		cfg := &Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			SampleRate:     0.5,
			Enabled:        true,
		}
		tracer, err := Init(cfg)
		require.NoError(t, err)
		require.NotNil(t, tracer)
		assert.Equal(t, "test-service", tracer.config.ServiceName)
	})

	t.Run("禁用追踪", func(t *testing.T) {
		cfg := &Config{
			ServiceName: "disabled-service",
			Enabled:     false,
		}
		tracer, err := Init(cfg)
		require.NoError(t, err)
		require.NotNil(t, tracer)
		assert.Nil(t, tracer.provider)
	})

	t.Run("100%采样率", func(t *testing.T) {
		cfg := &Config{
			ServiceName: "full-sample",
			SampleRate:  1.0,
			Enabled:     true,
		}
		tracer, err := Init(cfg)
		require.NoError(t, err)
		require.NotNil(t, tracer)
	})

	t.Run("0%采样率", func(t *testing.T) {
		cfg := &Config{
			ServiceName: "no-sample",
			SampleRate:  0,
			Enabled:     true,
		}
		tracer, err := Init(cfg)
		require.NoError(t, err)
		require.NotNil(t, tracer)
	})
}

func TestGetTracer(t *testing.T) {
	cfg := &Config{
		ServiceName: "get-tracer-test",
		Enabled:     true,
	}
	_, err := Init(cfg)
	require.NoError(t, err)

	tracer := GetTracer()
	require.NotNil(t, tracer)
}

func TestTracer_Shutdown(t *testing.T) {
	t.Run("关闭已启用的追踪器", func(t *testing.T) {
		cfg := &Config{
			ServiceName: "shutdown-test",
			Enabled:     true,
		}
		tracer, err := Init(cfg)
		require.NoError(t, err)

		err = tracer.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("关闭未启用的追踪器", func(t *testing.T) {
		cfg := &Config{
			ServiceName: "shutdown-disabled",
			Enabled:     false,
		}
		tracer, err := Init(cfg)
		require.NoError(t, err)

		err = tracer.Shutdown(context.Background())
		require.NoError(t, err)
	})
}

func TestTracer_Start(t *testing.T) {
	cfg := &Config{
		ServiceName: "start-span-test",
		Enabled:     true,
	}
	tracer, err := Init(cfg)
	require.NoError(t, err)

	t.Run("启动新span", func(t *testing.T) {
		ctx, span := tracer.Start(context.Background(), "test-operation")
		require.NotNil(t, ctx)
		require.NotNil(t, span)
		span.End()
	})

	t.Run("禁用时返回默认span", func(t *testing.T) {
		disabledTracer := &Tracer{config: &Config{Enabled: false}}
		ctx, span := disabledTracer.Start(context.Background(), "noop-test")
		require.NotNil(t, ctx)
		require.NotNil(t, span)
		// 禁用时返回的span应该是安全可用的（不会panic）
		span.AddEvent("test-event")
		span.End()
	})
}

func TestTracer_StartSpan(t *testing.T) {
	cfg := &Config{
		ServiceName: "start-span-attrs-test",
		Enabled:     true,
	}
	tracer, err := Init(cfg)
	require.NoError(t, err)

	t.Run("启动带属性的span", func(t *testing.T) {
		ctx, span := tracer.StartSpan(context.Background(), "db-query",
			attribute.String("db.table", "users"),
			attribute.String("db.operation", "SELECT"),
		)
		require.NotNil(t, ctx)
		require.NotNil(t, span)
		span.End()
	})
}

func TestSpanFromContext(t *testing.T) {
	cfg := &Config{
		ServiceName: "span-context-test",
		Enabled:     true,
	}
	tracer, err := Init(cfg)
	require.NoError(t, err)

	ctx, span := tracer.Start(context.Background(), "parent-span")
	defer span.End()

	retrievedSpan := SpanFromContext(ctx)
	require.NotNil(t, retrievedSpan)
}

func TestAddEvent(t *testing.T) {
	cfg := &Config{
		ServiceName: "add-event-test",
		Enabled:     true,
	}
	tracer, err := Init(cfg)
	require.NoError(t, err)

	ctx, span := tracer.Start(context.Background(), "event-test")
	defer span.End()

	// 不会panic即为成功
	AddEvent(ctx, "user-login", attribute.String("username", "testuser"))
	AddEvent(ctx, "cache-miss")
}

func TestSetError(t *testing.T) {
	cfg := &Config{
		ServiceName: "set-error-test",
		Enabled:     true,
	}
	tracer, err := Init(cfg)
	require.NoError(t, err)

	ctx, span := tracer.Start(context.Background(), "error-test")
	defer span.End()

	// 不会panic即为成功
	SetError(ctx, errors.New("test error"))
}

func TestSetAttributes(t *testing.T) {
	cfg := &Config{
		ServiceName: "set-attrs-test",
		Enabled:     true,
	}
	tracer, err := Init(cfg)
	require.NoError(t, err)

	ctx, span := tracer.Start(context.Background(), "attrs-test")
	defer span.End()

	// 不会panic即为成功
	SetAttributes(ctx,
		attribute.Int64("user_id", 123),
		attribute.String("action", "login"),
	)
}

func TestDisabledTracerSpan(t *testing.T) {
	// 测试禁用追踪时返回的span是安全可用的
	disabledTracer := &Tracer{config: &Config{Enabled: false}}
	ctx, span := disabledTracer.Start(context.Background(), "noop-test")
	require.NotNil(t, ctx)
	require.NotNil(t, span)

	t.Run("AddEvent不会panic", func(t *testing.T) {
		span.AddEvent("test-event")
	})

	t.Run("End不会panic", func(t *testing.T) {
		span.End()
	})
}

func TestAttributeHelpers(t *testing.T) {
	t.Run("WithUserID", func(t *testing.T) {
		attr := WithUserID(123)
		assert.Equal(t, "user.id", string(attr.Key))
		assert.Equal(t, int64(123), attr.Value.AsInt64())
	})

	t.Run("WithDeviceID", func(t *testing.T) {
		attr := WithDeviceID(456)
		assert.Equal(t, "device.id", string(attr.Key))
		assert.Equal(t, int64(456), attr.Value.AsInt64())
	})

	t.Run("WithOrderID", func(t *testing.T) {
		attr := WithOrderID(789)
		assert.Equal(t, "order.id", string(attr.Key))
		assert.Equal(t, int64(789), attr.Value.AsInt64())
	})

	t.Run("WithOperation", func(t *testing.T) {
		attr := WithOperation("create")
		assert.Equal(t, "operation", string(attr.Key))
		assert.Equal(t, "create", attr.Value.AsString())
	})

	t.Run("WithDBTable", func(t *testing.T) {
		attr := WithDBTable("users")
		assert.Equal(t, "db.table", string(attr.Key))
		assert.Equal(t, "users", attr.Value.AsString())
	})
}

func TestAttributeKeys(t *testing.T) {
	t.Run("预定义属性键", func(t *testing.T) {
		assert.Equal(t, attribute.Key("user.id"), AttrUserID)
		assert.Equal(t, attribute.Key("device.id"), AttrDeviceID)
		assert.Equal(t, attribute.Key("order.id"), AttrOrderID)
		assert.Equal(t, attribute.Key("merchant.id"), AttrMerchantID)
		assert.Equal(t, attribute.Key("venue.id"), AttrVenueID)
		assert.Equal(t, attribute.Key("operation"), AttrOperation)
		assert.Equal(t, attribute.Key("db.table"), AttrDBTable)
		assert.Equal(t, attribute.Key("db.operation"), AttrDBOperation)
		assert.Equal(t, attribute.Key("cache.key"), AttrCacheKey)
		assert.Equal(t, attribute.Key("mqtt.topic"), AttrMQTTTopic)
	})
}

func TestConfig_Defaults(t *testing.T) {
	cfg := &Config{}

	t.Run("空配置值", func(t *testing.T) {
		assert.Empty(t, cfg.ServiceName)
		assert.Empty(t, cfg.ServiceVersion)
		assert.Empty(t, cfg.Environment)
		assert.Empty(t, cfg.Endpoint)
		assert.Zero(t, cfg.SampleRate)
		assert.False(t, cfg.Enabled)
	})
}
