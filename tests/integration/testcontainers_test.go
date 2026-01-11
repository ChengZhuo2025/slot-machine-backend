//go:build integration

// Package integration testcontainers-go 使用示例测试
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTestContainers_Example 演示如何使用 TestContainers
func TestTestContainers_Example(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	tc := NewTestContainers(ctx)

	// 启动所有容器
	err := tc.StartAll()
	require.NoError(t, err, "failed to start containers")

	// 确保清理容器
	t.Cleanup(func() {
		_ = tc.Cleanup()
	})

	// 测试 Postgres 连接
	t.Run("Postgres", func(t *testing.T) {
		db, err := tc.GetPostgresDB()
		require.NoError(t, err)

		// 创建测试表
		err = db.Exec(`CREATE TABLE IF NOT EXISTS test_users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`).Error
		assert.NoError(t, err)

		// 插入数据
		err = db.Exec(`INSERT INTO test_users (name, email) VALUES (?, ?)`, "张三", "zhangsan@test.com").Error
		assert.NoError(t, err)

		// 查询数据
		var count int64
		err = db.Raw(`SELECT COUNT(*) FROM test_users`).Scan(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	// 测试 Redis 连接
	t.Run("Redis", func(t *testing.T) {
		client, err := tc.GetRedisClient()
		require.NoError(t, err)

		ctx := context.Background()

		// 设置值
		err = client.Set(ctx, "test:key", "hello", time.Minute).Err()
		assert.NoError(t, err)

		// 获取值
		val, err := client.Get(ctx, "test:key").Result()
		assert.NoError(t, err)
		assert.Equal(t, "hello", val)

		// 删除值
		err = client.Del(ctx, "test:key").Err()
		assert.NoError(t, err)
	})
}

// TestTestContainers_PostgresOnly 仅启动 Postgres
func TestTestContainers_PostgresOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	tc := NewTestContainers(ctx)

	err := tc.StartPostgres(DefaultPostgresConfig())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tc.Cleanup()
	})

	db, err := tc.GetPostgresDB()
	require.NoError(t, err)

	// 验证连接
	sqlDB, err := db.DB()
	require.NoError(t, err)

	err = sqlDB.Ping()
	assert.NoError(t, err)
}

// TestTestContainers_RedisOnly 仅启动 Redis
func TestTestContainers_RedisOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	tc := NewTestContainers(ctx)

	err := tc.StartRedis(DefaultRedisConfig())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tc.Cleanup()
	})

	client, err := tc.GetRedisClient()
	require.NoError(t, err)

	// 验证连接
	pong, err := client.Ping(ctx).Result()
	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)
}

// TestTestContainers_CustomConfig 使用自定义配置
func TestTestContainers_CustomConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	tc := NewTestContainers(ctx)

	// 自定义 Postgres 配置
	pgCfg := PostgresConfig{
		Database: "custom_db",
		User:     "custom_user",
		Password: "custom_password",
		Image:    "postgres:14-alpine",
	}

	err := tc.StartPostgres(pgCfg)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tc.Cleanup()
	})

	// 验证 DSN 包含自定义配置
	assert.Contains(t, tc.PostgresDSN, "custom_db")
	assert.Contains(t, tc.PostgresDSN, "custom_user")
	assert.Contains(t, tc.PostgresDSN, "custom_password")
}

// TestTestContainers_GetDBBeforeStart 在启动前获取 DB 应该失败
func TestTestContainers_GetDBBeforeStart(t *testing.T) {
	ctx := context.Background()
	tc := NewTestContainers(ctx)

	_, err := tc.GetPostgresDB()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres container not started")

	_, err = tc.GetRedisClient()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis container not started")
}

// TestTestContainers_CleanupWithoutStart 清理未启动的容器应该成功
func TestTestContainers_CleanupWithoutStart(t *testing.T) {
	ctx := context.Background()
	tc := NewTestContainers(ctx)

	err := tc.Cleanup()
	assert.NoError(t, err)
}
