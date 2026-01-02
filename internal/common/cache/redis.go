// Package cache 提供 Redis 缓存功能
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/common/config"
	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

// Init 初始化 Redis 连接
func Init(cfg *config.RedisConfig) (*redis.Client, error) {
	rdb = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	return rdb, nil
}

// GetClient 获取 Redis 客户端
func GetClient() *redis.Client {
	return rdb
}

// Close 关闭 Redis 连接
func Close() error {
	if rdb != nil {
		return rdb.Close()
	}
	return nil
}

// Set 设置缓存
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return rdb.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
func Get(ctx context.Context, key string, dest interface{}) error {
	data, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// GetString 获取字符串缓存
func GetString(ctx context.Context, key string) (string, error) {
	return rdb.Get(ctx, key).Result()
}

// SetString 设置字符串缓存
func SetString(ctx context.Context, key string, value string, expiration time.Duration) error {
	return rdb.Set(ctx, key, value, expiration).Err()
}

// Delete 删除缓存
func Delete(ctx context.Context, keys ...string) error {
	return rdb.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	n, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Expire 设置过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return rdb.Expire(ctx, key, expiration).Err()
}

// TTL 获取剩余过期时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	return rdb.TTL(ctx, key).Result()
}

// Incr 自增
func Incr(ctx context.Context, key string) (int64, error) {
	return rdb.Incr(ctx, key).Result()
}

// IncrBy 自增指定值
func IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return rdb.IncrBy(ctx, key, value).Result()
}

// Decr 自减
func Decr(ctx context.Context, key string) (int64, error) {
	return rdb.Decr(ctx, key).Result()
}

// SetNX 设置如果不存在（分布式锁基础）
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return rdb.SetNX(ctx, key, value, expiration).Result()
}

// HSet 设置哈希字段
func HSet(ctx context.Context, key string, values ...interface{}) error {
	return rdb.HSet(ctx, key, values...).Err()
}

// HGet 获取哈希字段
func HGet(ctx context.Context, key, field string) (string, error) {
	return rdb.HGet(ctx, key, field).Result()
}

// HGetAll 获取所有哈希字段
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return rdb.HGetAll(ctx, key).Result()
}

// HDel 删除哈希字段
func HDel(ctx context.Context, key string, fields ...string) error {
	return rdb.HDel(ctx, key, fields...).Err()
}

// SAdd 集合添加成员
func SAdd(ctx context.Context, key string, members ...interface{}) error {
	return rdb.SAdd(ctx, key, members...).Err()
}

// SMembers 获取集合所有成员
func SMembers(ctx context.Context, key string) ([]string, error) {
	return rdb.SMembers(ctx, key).Result()
}

// SIsMember 检查是否是集合成员
func SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return rdb.SIsMember(ctx, key, member).Result()
}

// SRem 移除集合成员
func SRem(ctx context.Context, key string, members ...interface{}) error {
	return rdb.SRem(ctx, key, members...).Err()
}

// LPush 列表左侧插入
func LPush(ctx context.Context, key string, values ...interface{}) error {
	return rdb.LPush(ctx, key, values...).Err()
}

// RPush 列表右侧插入
func RPush(ctx context.Context, key string, values ...interface{}) error {
	return rdb.RPush(ctx, key, values...).Err()
}

// LPop 列表左侧弹出
func LPop(ctx context.Context, key string) (string, error) {
	return rdb.LPop(ctx, key).Result()
}

// RPop 列表右侧弹出
func RPop(ctx context.Context, key string) (string, error) {
	return rdb.RPop(ctx, key).Result()
}

// LRange 获取列表范围
func LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return rdb.LRange(ctx, key, start, stop).Result()
}

// LLen 获取列表长度
func LLen(ctx context.Context, key string) (int64, error) {
	return rdb.LLen(ctx, key).Result()
}

// CacheKey 常用缓存键前缀
const (
	KeyPrefixUser        = "user:"
	KeyPrefixDevice      = "device:"
	KeyPrefixOrder       = "order:"
	KeyPrefixSession     = "session:"
	KeyPrefixSMSCode     = "sms:code:"
	KeyPrefixRateLimit   = "ratelimit:"
	KeyPrefixLock        = "lock:"
	KeyPrefixDeviceState = "device:state:"
)

// BuildKey 构建缓存键
func BuildKey(prefix string, parts ...string) string {
	key := prefix
	for _, part := range parts {
		key += part + ":"
	}
	return key[:len(key)-1]
}
