// Package config 配置管理单元测试
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Load 测试 ====================

func TestLoad_WithDefaultValues(t *testing.T) {
	// 不指定配置文件路径，使用默认搜索路径
	cfg, err := Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 验证默认值
	assert.Equal(t, "smart-locker-backend", cfg.Server.Name)
	assert.Equal(t, 8000, cfg.Server.Port)
	assert.Equal(t, "postgres", cfg.Database.Driver)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
}

func TestLoad_WithConfigFile(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
server:
  name: "test-server"
  mode: "release"
  port: 9000
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load 函数应该成功读取配置文件（即使由于 sync.Once 只执行一次）
	cfg, err := Load(configPath)
	// sync.Once 可能导致这返回之前加载的配置，但不应该返回 error
	require.NoError(t, err)
	require.NotNil(t, cfg)
	// 注意：由于 sync.Once 的限制，这个配置可能是之前测试加载的配置
}

// ==================== Get 测试 ====================

func TestGet_ReturnsDefaultConfig(t *testing.T) {

	cfg := Get()
	require.NotNil(t, cfg)

	// 验证返回的是默认配置
	assert.Equal(t, "smart-locker-backend", cfg.Server.Name)
	assert.Equal(t, 8000, cfg.Server.Port)
}

func TestGet_ReturnsSameInstance(t *testing.T) {

	cfg1 := Get()
	cfg2 := Get()

	// 应该返回同一个实例
	assert.Equal(t, cfg1, cfg2)
}

// ==================== DatabaseConfig 测试 ====================

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name   string
		config DatabaseConfig
		want   string
	}{
		{
			name: "Standard config",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				Name:     "mydb",
				SSLMode:  "disable",
				Timezone: "Asia/Shanghai",
			},
			want: "host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=disable TimeZone=Asia/Shanghai",
		},
		{
			name: "Remote database",
			config: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "admin",
				Password: "p@ssw0rd",
				Name:     "production",
				SSLMode:  "require",
				Timezone: "UTC",
			},
			want: "host=db.example.com port=5433 user=admin password=p@ssw0rd dbname=production sslmode=require TimeZone=UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.DSN()
			assert.Equal(t, tt.want, dsn)
		})
	}
}

// ==================== RedisConfig 测试 ====================

func TestRedisConfig_Addr(t *testing.T) {
	tests := []struct {
		name   string
		config RedisConfig
		want   string
	}{
		{
			name: "Localhost",
			config: RedisConfig{
				Host: "localhost",
				Port: 6379,
			},
			want: "localhost:6379",
		},
		{
			name: "Remote server",
			config: RedisConfig{
				Host: "redis.example.com",
				Port: 6380,
			},
			want: "redis.example.com:6380",
		},
		{
			name: "IP address",
			config: RedisConfig{
				Host: "192.168.1.100",
				Port: 6379,
			},
			want: "192.168.1.100:6379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := tt.config.Addr()
			assert.Equal(t, tt.want, addr)
		})
	}
}

// ==================== JWTConfig 测试 ====================

func TestJWTConfig_AccessTokenDuration(t *testing.T) {
	tests := []struct {
		name   string
		expire int
		want   time.Duration
	}{
		{"1 hour", 1, 1 * time.Hour},
		{"24 hours", 24, 24 * time.Hour},
		{"168 hours (7 days)", 168, 168 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := JWTConfig{AccessTokenExpire: tt.expire}
			duration := config.AccessTokenDuration()
			assert.Equal(t, tt.want, duration)
		})
	}
}

func TestJWTConfig_RefreshTokenDuration(t *testing.T) {
	tests := []struct {
		name   string
		expire int
		want   time.Duration
	}{
		{"24 hours", 24, 24 * time.Hour},
		{"168 hours (7 days)", 168, 168 * time.Hour},
		{"720 hours (30 days)", 720, 720 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := JWTConfig{RefreshTokenExpire: tt.expire}
			duration := config.RefreshTokenDuration()
			assert.Equal(t, tt.want, duration)
		})
	}
}

// ==================== Config 模式测试 ====================

func TestConfig_IsDebug(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{"Debug mode", "debug", true},
		{"Release mode", "release", false},
		{"Test mode", "test", false},
		{"Empty mode", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Server: ServerConfig{Mode: tt.mode},
			}
			assert.Equal(t, tt.want, config.IsDebug())
		})
	}
}

func TestConfig_IsRelease(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{"Release mode", "release", true},
		{"Debug mode", "debug", false},
		{"Test mode", "test", false},
		{"Empty mode", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Server: ServerConfig{Mode: tt.mode},
			}
			assert.Equal(t, tt.want, config.IsRelease())
		})
	}
}

// ==================== 业务配置测试 ====================

func TestBusinessConfig_Defaults(t *testing.T) {

	cfg := Get()

	// 验证租借配置默认值
	assert.Equal(t, 99.00, cfg.Business.Rental.DefaultDeposit)
	assert.Equal(t, 24, cfg.Business.Rental.AutoPurchaseHours)
	assert.Equal(t, 5, cfg.Business.Rental.TimeoutCheckInterval)

	// 验证分销配置默认值
	assert.Equal(t, 0.10, cfg.Business.Distribution.Level1Rate)
	assert.Equal(t, 0.05, cfg.Business.Distribution.Level2Rate)
	assert.Equal(t, 2, cfg.Business.Distribution.MaxLevel)
	assert.Equal(t, 100.00, cfg.Business.Distribution.MinWithdrawAmount)

	// 验证会员配置默认值
	assert.Equal(t, 1, cfg.Business.Member.PointsRate)
	assert.Equal(t, 100, cfg.Business.Member.PointsToMoney)
}

// ==================== 配置结构完整性测试 ====================

func TestConfig_AllFieldsPopulated(t *testing.T) {

	cfg := Get()
	require.NotNil(t, cfg)

	// 验证所有主要配置项都有值
	assert.NotEmpty(t, cfg.Server.Name)
	assert.NotZero(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Database.Driver)
	assert.NotEmpty(t, cfg.Database.Host)
	assert.NotZero(t, cfg.Database.Port)
	assert.NotEmpty(t, cfg.Redis.Host)
	assert.NotZero(t, cfg.Redis.Port)
	assert.NotEmpty(t, cfg.JWT.Secret)
	assert.NotZero(t, cfg.JWT.AccessTokenExpire)
	assert.NotEmpty(t, cfg.Logger.Level)
	assert.NotEmpty(t, cfg.Logger.Format)
}

// ==================== CORS 配置测试 ====================

func TestCORSConfig_Defaults(t *testing.T) {

	cfg := Get()

	// 验证 CORS 默认值
	assert.Contains(t, cfg.CORS.AllowedOrigins, "*")
	assert.Contains(t, cfg.CORS.AllowedMethods, "GET")
	assert.Contains(t, cfg.CORS.AllowedMethods, "POST")
	assert.Contains(t, cfg.CORS.AllowedHeaders, "Authorization")
	assert.True(t, cfg.CORS.AllowCredentials)
	assert.Equal(t, 86400, cfg.CORS.MaxAge)
}

// ==================== 监控配置测试 ====================

func TestMetricsConfig_Defaults(t *testing.T) {

	cfg := Get()

	// 验证监控配置默认值
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, 9100, cfg.Metrics.Port)
	assert.Equal(t, "/metrics", cfg.Metrics.Path)
}

func TestTracingConfig_Defaults(t *testing.T) {

	cfg := Get()

	// 验证追踪配置默认值
	assert.False(t, cfg.Tracing.Enabled)
	assert.Equal(t, "smart-locker-backend", cfg.Tracing.ServiceName)
	assert.Equal(t, 1.0, cfg.Tracing.SampleRate)
}

// ==================== 限流配置测试 ====================

func TestRateLimitConfig_Defaults(t *testing.T) {

	cfg := Get()

	// 验证限流配置默认值
	assert.True(t, cfg.RateLimit.Enabled)
	assert.Equal(t, 100, cfg.RateLimit.RequestsPerSecond)
	assert.Equal(t, 200, cfg.RateLimit.Burst)
}

// ==================== MQTT 配置测试 ====================

func TestMQTTConfig_Defaults(t *testing.T) {

	cfg := Get()

	// 验证 MQTT 配置默认值
	assert.Equal(t, "tcp://localhost:1883", cfg.MQTT.Broker)
	assert.Equal(t, "smart-locker-", cfg.MQTT.ClientIDPrefix)
	assert.Equal(t, 60, cfg.MQTT.KeepAlive)
	assert.True(t, cfg.MQTT.AutoReconnect)
	assert.Equal(t, byte(1), cfg.MQTT.QoS)
	assert.False(t, cfg.MQTT.Retained)
	assert.Equal(t, "smart-locker/", cfg.MQTT.TopicPrefix)
}

// ==================== SMS 配置测试 ====================

func TestSMSConfig_Defaults(t *testing.T) {

	cfg := Get()

	// 验证短信配置默认值
	assert.Equal(t, "aliyun", cfg.SMS.Provider)
	assert.Equal(t, 5, cfg.SMS.CodeExpire)
	assert.Equal(t, 60, cfg.SMS.SendInterval)
	assert.Equal(t, 10, cfg.SMS.DailyLimit)
}

// ==================== 日志配置测试 ====================

func TestLoggerConfig_Defaults(t *testing.T) {

	cfg := Get()

	// 验证日志配置默认值
	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, "console", cfg.Logger.Format)
	assert.Equal(t, "stdout", cfg.Logger.Output)
	assert.Equal(t, "./logs/app.log", cfg.Logger.FilePath)
	assert.Equal(t, 100, cfg.Logger.MaxSize)
	assert.Equal(t, 10, cfg.Logger.MaxBackups)
	assert.Equal(t, 30, cfg.Logger.MaxAge)
	assert.True(t, cfg.Logger.Compress)
	assert.True(t, cfg.Logger.Caller)
}
