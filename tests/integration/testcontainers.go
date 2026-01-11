//go:build integration

// Package integration 提供 testcontainers-go 集成测试环境配置
package integration

import (
	"context"
	"fmt"
	"time"

	redisClient "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestContainers 管理测试容器
type TestContainers struct {
	PostgresContainer testcontainers.Container
	RedisContainer    testcontainers.Container
	PostgresDSN       string
	RedisAddr         string
	ctx               context.Context
}

// PostgresConfig Postgres 容器配置
type PostgresConfig struct {
	Database string
	User     string
	Password string
	Image    string
}

// DefaultPostgresConfig 返回默认 Postgres 配置
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Database: "test_smart_locker",
		User:     "test_user",
		Password: "test_password",
		Image:    "postgres:15-alpine",
	}
}

// RedisConfig Redis 容器配置
type RedisConfig struct {
	Image string
}

// DefaultRedisConfig 返回默认 Redis 配置
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Image: "redis:7-alpine",
	}
}

// NewTestContainers 创建测试容器管理器
func NewTestContainers(ctx context.Context) *TestContainers {
	return &TestContainers{ctx: ctx}
}

// StartPostgres 启动 Postgres 容器
func (tc *TestContainers) StartPostgres(cfg PostgresConfig) error {
	container, err := tcPostgres.RunContainer(tc.ctx,
		testcontainers.WithImage(cfg.Image),
		tcPostgres.WithDatabase(cfg.Database),
		tcPostgres.WithUsername(cfg.User),
		tcPostgres.WithPassword(cfg.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	tc.PostgresContainer = container

	// 获取连接字符串
	host, err := container.Host(tc.ctx)
	if err != nil {
		return fmt.Errorf("failed to get postgres host: %w", err)
	}

	port, err := container.MappedPort(tc.ctx, "5432")
	if err != nil {
		return fmt.Errorf("failed to get postgres port: %w", err)
	}

	tc.PostgresDSN = fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port.Port(), cfg.User, cfg.Password, cfg.Database,
	)

	return nil
}

// StartRedis 启动 Redis 容器
func (tc *TestContainers) StartRedis(cfg RedisConfig) error {
	container, err := tcRedis.RunContainer(tc.ctx,
		testcontainers.WithImage(cfg.Image),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to start redis container: %w", err)
	}

	tc.RedisContainer = container

	// 获取连接地址
	host, err := container.Host(tc.ctx)
	if err != nil {
		return fmt.Errorf("failed to get redis host: %w", err)
	}

	port, err := container.MappedPort(tc.ctx, "6379")
	if err != nil {
		return fmt.Errorf("failed to get redis port: %w", err)
	}

	tc.RedisAddr = fmt.Sprintf("%s:%s", host, port.Port())

	return nil
}

// GetPostgresDB 获取 GORM 数据库连接
func (tc *TestContainers) GetPostgresDB() (*gorm.DB, error) {
	if tc.PostgresDSN == "" {
		return nil, fmt.Errorf("postgres container not started")
	}

	db, err := gorm.Open(postgres.Open(tc.PostgresDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return db, nil
}

// GetRedisClient 获取 Redis 客户端
func (tc *TestContainers) GetRedisClient() (*redisClient.Client, error) {
	if tc.RedisAddr == "" {
		return nil, fmt.Errorf("redis container not started")
	}

	client := redisClient.NewClient(&redisClient.Options{
		Addr: tc.RedisAddr,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(tc.ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

// Cleanup 清理所有容器
func (tc *TestContainers) Cleanup() error {
	var errs []error

	if tc.PostgresContainer != nil {
		if err := tc.PostgresContainer.Terminate(tc.ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate postgres: %w", err))
		}
	}

	if tc.RedisContainer != nil {
		if err := tc.RedisContainer.Terminate(tc.ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate redis: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}

// StartAll 启动所有容器
func (tc *TestContainers) StartAll() error {
	if err := tc.StartPostgres(DefaultPostgresConfig()); err != nil {
		return err
	}

	if err := tc.StartRedis(DefaultRedisConfig()); err != nil {
		return err
	}

	return nil
}
