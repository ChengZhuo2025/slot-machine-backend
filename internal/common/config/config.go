// Package config 提供应用配置管理功能
package config

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var (
	globalConfig *Config
	once         sync.Once
)

// Config 应用配置结构
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Redis       RedisConfig       `mapstructure:"redis"`
	MQTT        MQTTConfig        `mapstructure:"mqtt"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	Crypto      CryptoConfig      `mapstructure:"crypto"`
	SMS         SMSConfig         `mapstructure:"sms"`
	WeChat      WeChatConfig      `mapstructure:"wechat"`
	Alipay      AlipayConfig      `mapstructure:"alipay"`
	OSS         OSSConfig         `mapstructure:"oss"`
	Logger      LoggerConfig      `mapstructure:"logger"`
	Metrics     MetricsConfig     `mapstructure:"metrics"`
	Tracing     TracingConfig     `mapstructure:"tracing"`
	RateLimit   RateLimitConfig   `mapstructure:"ratelimit"`
	CORS        CORSConfig        `mapstructure:"cors"`
	Business    BusinessConfig    `mapstructure:"business"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Name            string `mapstructure:"name"`
	Mode            string `mapstructure:"mode"`
	Port            int    `mapstructure:"port"`
	ReadTimeout     int    `mapstructure:"read_timeout"`
	WriteTimeout    int    `mapstructure:"write_timeout"`
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Name            string `mapstructure:"name"`
	SSLMode         string `mapstructure:"sslmode"`
	Timezone        string `mapstructure:"timezone"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
	LogMode         bool   `mapstructure:"log_mode"`
	SlowThreshold   int    `mapstructure:"slow_threshold"`
}

// DSN 返回数据库连接字符串
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.Timezone,
	)
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
	DialTimeout  int    `mapstructure:"dial_timeout"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// Addr 返回 Redis 地址
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// MQTTConfig MQTT配置
type MQTTConfig struct {
	Broker         string `mapstructure:"broker"`
	ClientIDPrefix string `mapstructure:"client_id_prefix"`
	Username       string `mapstructure:"username"`
	Password       string `mapstructure:"password"`
	KeepAlive      int    `mapstructure:"keep_alive"`
	AutoReconnect  bool   `mapstructure:"auto_reconnect"`
	ConnectTimeout int    `mapstructure:"connect_timeout"`
	QoS            byte   `mapstructure:"qos"`
	Retained       bool   `mapstructure:"retained"`
	TopicPrefix    string `mapstructure:"topic_prefix"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	AccessTokenExpire  int    `mapstructure:"access_token_expire"`
	RefreshTokenExpire int    `mapstructure:"refresh_token_expire"`
	Issuer             string `mapstructure:"issuer"`
}

// AccessTokenDuration 返回访问令牌有效期
func (j *JWTConfig) AccessTokenDuration() time.Duration {
	return time.Duration(j.AccessTokenExpire) * time.Hour
}

// RefreshTokenDuration 返回刷新令牌有效期
func (j *JWTConfig) RefreshTokenDuration() time.Duration {
	return time.Duration(j.RefreshTokenExpire) * time.Hour
}

// CryptoConfig 加密配置
type CryptoConfig struct {
	AESKey     string `mapstructure:"aes_key"`
	BcryptCost int    `mapstructure:"bcrypt_cost"`
}

// SMSConfig 短信配置
type SMSConfig struct {
	Provider        string `mapstructure:"provider"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	SignName        string `mapstructure:"sign_name"`
	TemplateID      string `mapstructure:"template_id"`
	CodeExpire      int    `mapstructure:"code_expire"`
	SendInterval    int    `mapstructure:"send_interval"`
	DailyLimit      int    `mapstructure:"daily_limit"`
}

// WeChatConfig 微信配置
type WeChatConfig struct {
	AppID          string `mapstructure:"app_id"`
	AppSecret      string `mapstructure:"app_secret"`
	MchID          string `mapstructure:"mch_id"`
	APIv3Key       string `mapstructure:"api_v3_key"`
	SerialNo       string `mapstructure:"serial_no"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	NotifyURL      string `mapstructure:"notify_url"`
}

// AlipayConfig 支付宝配置
type AlipayConfig struct {
	AppID               string `mapstructure:"app_id"`
	PrivateKeyPath      string `mapstructure:"private_key_path"`
	AlipayPublicKeyPath string `mapstructure:"alipay_public_key_path"`
	IsSandbox           bool   `mapstructure:"is_sandbox"`
	NotifyURL           string `mapstructure:"notify_url"`
}

// OSSConfig 对象存储配置
type OSSConfig struct {
	Provider        string `mapstructure:"provider"`
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Bucket          string `mapstructure:"bucket"`
	CustomDomain    string `mapstructure:"custom_domain"`
	UploadDir       string `mapstructure:"upload_dir"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
	Caller     bool   `mapstructure:"caller"`
}

// MetricsConfig 监控配置
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// TracingConfig 链路追踪配置
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	Endpoint    string  `mapstructure:"endpoint"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerSecond int  `mapstructure:"requests_per_second"`
	Burst             int  `mapstructure:"burst"`
}

// CORSConfig 跨域配置
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

// BusinessConfig 业务配置
type BusinessConfig struct {
	Rental       RentalConfig       `mapstructure:"rental"`
	Distribution DistributionConfig `mapstructure:"distribution"`
	Member       MemberConfig       `mapstructure:"member"`
}

// RentalConfig 租借配置
type RentalConfig struct {
	DefaultDeposit       float64 `mapstructure:"default_deposit"`
	AutoPurchaseHours    int     `mapstructure:"auto_purchase_hours"`
	TimeoutCheckInterval int     `mapstructure:"timeout_check_interval"`
}

// DistributionConfig 分销配置
type DistributionConfig struct {
	Level1Rate        float64 `mapstructure:"level1_rate"`
	Level2Rate        float64 `mapstructure:"level2_rate"`
	MaxLevel          int     `mapstructure:"max_level"`
	MinWithdrawAmount float64 `mapstructure:"min_withdraw_amount"`
}

// MemberConfig 会员配置
type MemberConfig struct {
	PointsRate    int `mapstructure:"points_rate"`
	PointsToMoney int `mapstructure:"points_to_money"`
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	var err error
	once.Do(func() {
		v := viper.New()

		// 设置配置文件路径
		if configPath != "" {
			v.SetConfigFile(configPath)
		} else {
			v.SetConfigName("config")
			v.SetConfigType("yaml")
			v.AddConfigPath("./configs")
			v.AddConfigPath(".")
		}

		// 环境变量支持
		v.AutomaticEnv()
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		// 设置默认值
		setDefaults(v)

		// 读取配置文件
		if err = v.ReadInConfig(); err != nil {
			// 如果配置文件不存在，使用默认值
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return
			}
			err = nil
		}

		// 解析配置
		globalConfig = &Config{}
		if err = v.Unmarshal(globalConfig); err != nil {
			return
		}
	})

	return globalConfig, err
}

// Get 获取全局配置
func Get() *Config {
	if globalConfig == nil {
		// 使用默认配置
		globalConfig = &Config{}
		v := viper.New()
		setDefaults(v)
		_ = v.Unmarshal(globalConfig)
	}
	return globalConfig
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.name", "smart-locker-backend")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.port", 8000)
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)
	v.SetDefault("server.shutdown_timeout", 10)

	// Database defaults
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.name", "smart_locker")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.timezone", "Asia/Shanghai")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime", 60)
	v.SetDefault("database.log_mode", true)
	v.SetDefault("database.slow_threshold", 200)

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 100)
	v.SetDefault("redis.min_idle_conns", 10)
	v.SetDefault("redis.dial_timeout", 5)
	v.SetDefault("redis.read_timeout", 3)
	v.SetDefault("redis.write_timeout", 3)

	// MQTT defaults
	v.SetDefault("mqtt.broker", "tcp://localhost:1883")
	v.SetDefault("mqtt.client_id_prefix", "smart-locker-")
	v.SetDefault("mqtt.keep_alive", 60)
	v.SetDefault("mqtt.auto_reconnect", true)
	v.SetDefault("mqtt.connect_timeout", 10)
	v.SetDefault("mqtt.qos", 1)
	v.SetDefault("mqtt.retained", false)
	v.SetDefault("mqtt.topic_prefix", "smart-locker/")

	// JWT defaults
	v.SetDefault("jwt.secret", "your-super-secret-key-change-in-production")
	v.SetDefault("jwt.access_token_expire", 168)
	v.SetDefault("jwt.refresh_token_expire", 720)
	v.SetDefault("jwt.issuer", "smart-locker")

	// Crypto defaults
	v.SetDefault("crypto.bcrypt_cost", 10)

	// SMS defaults
	v.SetDefault("sms.provider", "aliyun")
	v.SetDefault("sms.code_expire", 5)
	v.SetDefault("sms.send_interval", 60)
	v.SetDefault("sms.daily_limit", 10)

	// Logger defaults
	v.SetDefault("logger.level", "debug")
	v.SetDefault("logger.format", "console")
	v.SetDefault("logger.output", "stdout")
	v.SetDefault("logger.file_path", "./logs/app.log")
	v.SetDefault("logger.max_size", 100)
	v.SetDefault("logger.max_backups", 10)
	v.SetDefault("logger.max_age", 30)
	v.SetDefault("logger.compress", true)
	v.SetDefault("logger.caller", true)

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.port", 9100)
	v.SetDefault("metrics.path", "/metrics")

	// Tracing defaults
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.service_name", "smart-locker-backend")
	v.SetDefault("tracing.sample_rate", 1.0)

	// Rate limit defaults
	v.SetDefault("ratelimit.enabled", true)
	v.SetDefault("ratelimit.requests_per_second", 100)
	v.SetDefault("ratelimit.burst", 200)

	// CORS defaults
	v.SetDefault("cors.allowed_origins", []string{"*"})
	v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allowed_headers", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"})
	v.SetDefault("cors.exposed_headers", []string{"X-Request-ID"})
	v.SetDefault("cors.allow_credentials", true)
	v.SetDefault("cors.max_age", 86400)

	// Business defaults
	v.SetDefault("business.rental.default_deposit", 99.00)
	v.SetDefault("business.rental.auto_purchase_hours", 24)
	v.SetDefault("business.rental.timeout_check_interval", 5)
	v.SetDefault("business.distribution.level1_rate", 0.10)
	v.SetDefault("business.distribution.level2_rate", 0.05)
	v.SetDefault("business.distribution.max_level", 2)
	v.SetDefault("business.distribution.min_withdraw_amount", 100.00)
	v.SetDefault("business.member.points_rate", 1)
	v.SetDefault("business.member.points_to_money", 100)
}

// IsDebug 是否为调试模式
func (c *Config) IsDebug() bool {
	return c.Server.Mode == "debug"
}

// IsRelease 是否为发布模式
func (c *Config) IsRelease() bool {
	return c.Server.Mode == "release"
}
