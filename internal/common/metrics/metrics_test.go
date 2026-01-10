// Package metrics 提供 Prometheus 指标收集单元测试
package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestInit(t *testing.T) {
	t.Run("使用默认命名空间", func(t *testing.T) {
		m := Init("")
		require.NotNil(t, m)
		assert.NotNil(t, m.httpRequestsTotal)
		assert.NotNil(t, m.httpRequestDuration)
		assert.NotNil(t, m.httpRequestsInFlight)
		assert.NotNil(t, m.dbQueriesTotal)
		assert.NotNil(t, m.dbQueryDuration)
		assert.NotNil(t, m.cacheHitsTotal)
		assert.NotNil(t, m.cacheMissesTotal)
		assert.NotNil(t, m.mqttMessagesTotal)
		assert.NotNil(t, m.activeDevices)
		assert.NotNil(t, m.activeUsers)
		assert.NotNil(t, m.ordersTotal)
		assert.NotNil(t, m.paymentsTotal)
	})

	t.Run("使用自定义命名空间", func(t *testing.T) {
		m := Init("custom_namespace")
		require.NotNil(t, m)
	})
}

func TestGetMetrics(t *testing.T) {
	t.Run("获取已初始化的指标", func(t *testing.T) {
		Init("test")
		m := GetMetrics()
		require.NotNil(t, m)
	})

	t.Run("获取指标实例", func(t *testing.T) {
		// GetMetrics 应该返回非空指标实例
		m := GetMetrics()
		require.NotNil(t, m)
	})
}

func TestMetrics_RecordDBQuery(t *testing.T) {
	m := Init("test_db")

	t.Run("记录SELECT查询", func(t *testing.T) {
		// 不会panic即为成功
		m.RecordDBQuery("SELECT", "users", 10*time.Millisecond)
	})

	t.Run("记录INSERT查询", func(t *testing.T) {
		m.RecordDBQuery("INSERT", "orders", 5*time.Millisecond)
	})

	t.Run("记录UPDATE查询", func(t *testing.T) {
		m.RecordDBQuery("UPDATE", "devices", 3*time.Millisecond)
	})

	t.Run("记录DELETE查询", func(t *testing.T) {
		m.RecordDBQuery("DELETE", "sessions", 2*time.Millisecond)
	})
}

func TestMetrics_RecordCache(t *testing.T) {
	m := Init("test_cache")

	t.Run("记录缓存命中", func(t *testing.T) {
		m.RecordCacheHit("user_cache")
		m.RecordCacheHit("session_cache")
	})

	t.Run("记录缓存未命中", func(t *testing.T) {
		m.RecordCacheMiss("user_cache")
		m.RecordCacheMiss("config_cache")
	})
}

func TestMetrics_RecordMQTTMessage(t *testing.T) {
	m := Init("test_mqtt")

	t.Run("记录发送消息", func(t *testing.T) {
		m.RecordMQTTMessage("device/status", "outbound")
	})

	t.Run("记录接收消息", func(t *testing.T) {
		m.RecordMQTTMessage("device/command", "inbound")
	})
}

func TestMetrics_SetActiveCounters(t *testing.T) {
	m := Init("test_counters")

	t.Run("设置活跃设备数", func(t *testing.T) {
		m.SetActiveDevices(100)
		m.SetActiveDevices(150)
	})

	t.Run("设置活跃用户数", func(t *testing.T) {
		m.SetActiveUsers(500)
		m.SetActiveUsers(600)
	})
}

func TestMetrics_RecordOrder(t *testing.T) {
	m := Init("test_orders")

	t.Run("记录已创建订单", func(t *testing.T) {
		m.RecordOrder("created")
	})

	t.Run("记录已支付订单", func(t *testing.T) {
		m.RecordOrder("paid")
	})

	t.Run("记录已完成订单", func(t *testing.T) {
		m.RecordOrder("completed")
	})

	t.Run("记录已取消订单", func(t *testing.T) {
		m.RecordOrder("cancelled")
	})
}

func TestMetrics_RecordPayment(t *testing.T) {
	m := Init("test_payments")

	t.Run("记录微信支付成功", func(t *testing.T) {
		m.RecordPayment("wechat", "success")
	})

	t.Run("记录支付宝支付成功", func(t *testing.T) {
		m.RecordPayment("alipay", "success")
	})

	t.Run("记录支付失败", func(t *testing.T) {
		m.RecordPayment("wechat", "failed")
	})

	t.Run("记录支付待处理", func(t *testing.T) {
		m.RecordPayment("alipay", "pending")
	})
}

func TestRecordHTTPRequest(t *testing.T) {
	Init("test_http")

	t.Run("记录HTTP请求", func(t *testing.T) {
		RecordHTTPRequest("GET", "/api/users", "200", 100*time.Millisecond)
		RecordHTTPRequest("POST", "/api/orders", "201", 50*time.Millisecond)
		RecordHTTPRequest("GET", "/api/users/1", "404", 10*time.Millisecond)
		RecordHTTPRequest("POST", "/api/login", "500", 200*time.Millisecond)
	})
}

func TestRecordDBQueryGlobal(t *testing.T) {
	Init("test_global")

	t.Run("全局记录数据库查询", func(t *testing.T) {
		RecordDBQueryGlobal("SELECT", "products", 15*time.Millisecond)
	})
}

func TestRecordCacheGlobal(t *testing.T) {
	Init("test_global_cache")

	t.Run("全局记录缓存命中", func(t *testing.T) {
		RecordCacheHitGlobal("product_cache")
	})

	t.Run("全局记录缓存未命中", func(t *testing.T) {
		RecordCacheMissGlobal("product_cache")
	})
}

func TestMetrics_Middleware(t *testing.T) {
	m := Init("test_middleware")

	router := gin.New()
	router.Use(m.Middleware())

	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "metrics")
	})

	t.Run("记录请求指标", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("跳过/metrics端点", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler(t *testing.T) {
	Init("test_handler")

	router := gin.New()
	router.GET("/metrics", Handler())

	t.Run("返回Prometheus指标", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Prometheus 指标应该包含一些标准内容
		body := w.Body.String()
		assert.Contains(t, body, "go_")  // Go 运行时指标
	})
}
