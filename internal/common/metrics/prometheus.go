// Package metrics 提供 Prometheus 指标收集
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 指标收集器
type Metrics struct {
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge
	dbQueriesTotal      *prometheus.CounterVec
	dbQueryDuration     *prometheus.HistogramVec
	cacheHitsTotal      *prometheus.CounterVec
	cacheMissesTotal    *prometheus.CounterVec
	mqttMessagesTotal   *prometheus.CounterVec
	activeDevices       prometheus.Gauge
	activeUsers         prometheus.Gauge
	ordersTotal         *prometheus.CounterVec
	paymentsTotal       *prometheus.CounterVec
}

var defaultMetrics *Metrics

// Init 初始化指标收集器
func Init(namespace string) *Metrics {
	if namespace == "" {
		namespace = "smart_locker"
	}

	m := &Metrics{
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		httpRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Current number of HTTP requests being processed",
			},
		),
		dbQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "table"},
		),
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation", "table"},
		),
		cacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache"},
		),
		cacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache"},
		),
		mqttMessagesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "mqtt_messages_total",
				Help:      "Total number of MQTT messages",
			},
			[]string{"topic", "direction"},
		),
		activeDevices: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_devices",
				Help:      "Number of currently active devices",
			},
		),
		activeUsers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_users",
				Help:      "Number of currently active users",
			},
		),
		ordersTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "orders_total",
				Help:      "Total number of orders",
			},
			[]string{"status"},
		),
		paymentsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "payments_total",
				Help:      "Total number of payments",
			},
			[]string{"method", "status"},
		),
	}

	defaultMetrics = m
	return m
}

// GetMetrics 获取默认指标收集器
func GetMetrics() *Metrics {
	if defaultMetrics == nil {
		return Init("")
	}
	return defaultMetrics
}

// Middleware 返回 Gin 中间件
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过 metrics 端点本身
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		m.httpRequestsInFlight.Inc()

		c.Next()

		m.httpRequestsInFlight.Dec()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		m.httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		m.httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// Handler 返回 Prometheus HTTP 处理器
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// RecordDBQuery 记录数据库查询
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration) {
	m.dbQueriesTotal.WithLabelValues(operation, table).Inc()
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordCacheHit 记录缓存命中
func (m *Metrics) RecordCacheHit(cache string) {
	m.cacheHitsTotal.WithLabelValues(cache).Inc()
}

// RecordCacheMiss 记录缓存未命中
func (m *Metrics) RecordCacheMiss(cache string) {
	m.cacheMissesTotal.WithLabelValues(cache).Inc()
}

// RecordMQTTMessage 记录 MQTT 消息
func (m *Metrics) RecordMQTTMessage(topic, direction string) {
	m.mqttMessagesTotal.WithLabelValues(topic, direction).Inc()
}

// SetActiveDevices 设置活跃设备数
func (m *Metrics) SetActiveDevices(count float64) {
	m.activeDevices.Set(count)
}

// SetActiveUsers 设置活跃用户数
func (m *Metrics) SetActiveUsers(count float64) {
	m.activeUsers.Set(count)
}

// RecordOrder 记录订单
func (m *Metrics) RecordOrder(status string) {
	m.ordersTotal.WithLabelValues(status).Inc()
}

// RecordPayment 记录支付
func (m *Metrics) RecordPayment(method, status string) {
	m.paymentsTotal.WithLabelValues(method, status).Inc()
}

// RecordHTTPRequest 手动记录 HTTP 请求（用于非中间件场景）
func RecordHTTPRequest(method, path, status string, duration time.Duration) {
	m := GetMetrics()
	m.httpRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordDBQueryGlobal 全局记录数据库查询
func RecordDBQueryGlobal(operation, table string, duration time.Duration) {
	GetMetrics().RecordDBQuery(operation, table, duration)
}

// RecordCacheHitGlobal 全局记录缓存命中
func RecordCacheHitGlobal(cache string) {
	GetMetrics().RecordCacheHit(cache)
}

// RecordCacheMissGlobal 全局记录缓存未命中
func RecordCacheMissGlobal(cache string) {
	GetMetrics().RecordCacheMiss(cache)
}
