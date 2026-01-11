// Package cache Redis 缓存模块单元测试
package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/dumeirei/smart-locker-backend/internal/common/config"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMiniRedis 创建 miniredis 测试实例
func setupMiniRedis(t *testing.T) *miniredis.Miniredis {
	s, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})
	return s
}

// setupTestRedis 初始化测试 Redis 客户端
func setupTestRedis(t *testing.T, s *miniredis.Miniredis) {
	rdb = redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	t.Cleanup(func() {
		_ = rdb.Close()
		rdb = nil
	})
}

// ==================== Init 函数测试 ====================

func TestInit_Success(t *testing.T) {
	s := setupMiniRedis(t)

	cfg := &config.RedisConfig{
		Host:         s.Host(),
		Port:         s.Server().Addr().Port,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5,
		ReadTimeout:  3,
		WriteTimeout: 3,
	}

	client, err := Init(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	t.Cleanup(func() {
		_ = Close()
	})
}

func TestInit_ConnectionFailed(t *testing.T) {
	cfg := &config.RedisConfig{
		Host:        "invalid-host",
		Port:        9999,
		DialTimeout: 1,
	}

	client, err := Init(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect redis")
}

// ==================== GetClient / Close 测试 ====================

func TestGetClient(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)

	client := GetClient()
	assert.NotNil(t, client)
	assert.Equal(t, rdb, client)
}

func TestClose_WithNilClient(t *testing.T) {
	rdb = nil
	err := Close()
	assert.NoError(t, err)
}

func TestClose_WithClient(t *testing.T) {
	s := setupMiniRedis(t)
	rdb = redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	err := Close()
	assert.NoError(t, err)
}

// ==================== Set / Get 测试 ====================

func TestSet_And_Get(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	// 设置值
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	data := TestData{Name: "test", Value: 123}

	err := Set(ctx, "test:key", data, time.Minute)
	assert.NoError(t, err)

	// 获取值
	var result TestData
	err = Get(ctx, "test:key", &result)
	assert.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestGet_NotFound(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	var result string
	err := Get(ctx, "nonexistent:key", &result)
	assert.Error(t, err)
	assert.Equal(t, redis.Nil, err)
}

func TestSet_MarshalError(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	// 无法序列化的值（包含 channel）
	ch := make(chan int)
	err := Set(ctx, "test:channel", ch, time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal value")
}

// ==================== SetString / GetString 测试 ====================

func TestSetString_And_GetString(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	err := SetString(ctx, "str:key", "hello world", time.Minute)
	assert.NoError(t, err)

	result, err := GetString(ctx, "str:key")
	assert.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestGetString_NotFound(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_, err := GetString(ctx, "nonexistent:str")
	assert.Error(t, err)
	assert.Equal(t, redis.Nil, err)
}

// ==================== Delete 测试 ====================

func TestDelete(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	// 设置值
	_ = SetString(ctx, "del:key1", "value1", time.Minute)
	_ = SetString(ctx, "del:key2", "value2", time.Minute)

	// 删除
	err := Delete(ctx, "del:key1", "del:key2")
	assert.NoError(t, err)

	// 验证删除
	_, err = GetString(ctx, "del:key1")
	assert.Equal(t, redis.Nil, err)

	_, err = GetString(ctx, "del:key2")
	assert.Equal(t, redis.Nil, err)
}

// ==================== Exists 测试 ====================

func TestExists(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	// 不存在
	exists, err := Exists(ctx, "check:key")
	assert.NoError(t, err)
	assert.False(t, exists)

	// 设置后存在
	_ = SetString(ctx, "check:key", "value", time.Minute)
	exists, err = Exists(ctx, "check:key")
	assert.NoError(t, err)
	assert.True(t, exists)
}

// ==================== Expire / TTL 测试 ====================

func TestExpire_And_TTL(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_ = SetString(ctx, "ttl:key", "value", time.Hour)

	// 修改过期时间
	err := Expire(ctx, "ttl:key", time.Minute)
	assert.NoError(t, err)

	// 获取 TTL
	ttl, err := TTL(ctx, "ttl:key")
	assert.NoError(t, err)
	assert.LessOrEqual(t, ttl, time.Minute)
	assert.Greater(t, ttl, time.Duration(0))
}

// ==================== Incr / IncrBy / Decr 测试 ====================

func TestIncr(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	val, err := Incr(ctx, "counter:incr")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = Incr(ctx, "counter:incr")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)
}

func TestIncrBy(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	val, err := IncrBy(ctx, "counter:incrby", 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)

	val, err = IncrBy(ctx, "counter:incrby", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), val)
}

func TestDecr(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_ = SetString(ctx, "counter:decr", "10", 0)

	val, err := Decr(ctx, "counter:decr")
	assert.NoError(t, err)
	assert.Equal(t, int64(9), val)
}

// ==================== SetNX 测试 ====================

func TestSetNX(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	// 第一次设置成功
	ok, err := SetNX(ctx, "lock:key", "owner1", time.Minute)
	assert.NoError(t, err)
	assert.True(t, ok)

	// 第二次设置失败（键已存在）
	ok, err = SetNX(ctx, "lock:key", "owner2", time.Minute)
	assert.NoError(t, err)
	assert.False(t, ok)
}

// ==================== Hash 操作测试 ====================

func TestHSet_And_HGet(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	err := HSet(ctx, "hash:key", "field1", "value1", "field2", "value2")
	assert.NoError(t, err)

	val, err := HGet(ctx, "hash:key", "field1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	val, err = HGet(ctx, "hash:key", "field2")
	assert.NoError(t, err)
	assert.Equal(t, "value2", val)
}

func TestHGetAll(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_ = HSet(ctx, "hash:all", "a", "1", "b", "2", "c", "3")

	result, err := HGetAll(ctx, "hash:all")
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "1", result["a"])
	assert.Equal(t, "2", result["b"])
	assert.Equal(t, "3", result["c"])
}

func TestHDel(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_ = HSet(ctx, "hash:del", "f1", "v1", "f2", "v2")

	err := HDel(ctx, "hash:del", "f1")
	assert.NoError(t, err)

	_, err = HGet(ctx, "hash:del", "f1")
	assert.Equal(t, redis.Nil, err)

	val, err := HGet(ctx, "hash:del", "f2")
	assert.NoError(t, err)
	assert.Equal(t, "v2", val)
}

// ==================== Set 操作测试 ====================

func TestSAdd_And_SMembers(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	err := SAdd(ctx, "set:key", "a", "b", "c")
	assert.NoError(t, err)

	members, err := SMembers(ctx, "set:key")
	assert.NoError(t, err)
	assert.Len(t, members, 3)
	assert.Contains(t, members, "a")
	assert.Contains(t, members, "b")
	assert.Contains(t, members, "c")
}

func TestSIsMember(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_ = SAdd(ctx, "set:check", "member1", "member2")

	isMember, err := SIsMember(ctx, "set:check", "member1")
	assert.NoError(t, err)
	assert.True(t, isMember)

	isMember, err = SIsMember(ctx, "set:check", "nonexistent")
	assert.NoError(t, err)
	assert.False(t, isMember)
}

func TestSRem(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_ = SAdd(ctx, "set:rem", "x", "y", "z")

	err := SRem(ctx, "set:rem", "y")
	assert.NoError(t, err)

	members, _ := SMembers(ctx, "set:rem")
	assert.Len(t, members, 2)
	assert.NotContains(t, members, "y")
}

// ==================== List 操作测试 ====================

func TestLPush_And_LPop(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	err := LPush(ctx, "list:lpush", "c", "b", "a")
	assert.NoError(t, err)

	val, err := LPop(ctx, "list:lpush")
	assert.NoError(t, err)
	assert.Equal(t, "a", val)
}

func TestRPush_And_RPop(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	err := RPush(ctx, "list:rpush", "a", "b", "c")
	assert.NoError(t, err)

	val, err := RPop(ctx, "list:rpush")
	assert.NoError(t, err)
	assert.Equal(t, "c", val)
}

func TestLRange(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_ = RPush(ctx, "list:range", "1", "2", "3", "4", "5")

	// 获取全部
	result, err := LRange(ctx, "list:range", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, result)

	// 获取部分
	result, err = LRange(ctx, "list:range", 1, 3)
	assert.NoError(t, err)
	assert.Equal(t, []string{"2", "3", "4"}, result)
}

func TestLLen(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	_ = RPush(ctx, "list:len", "a", "b", "c")

	length, err := LLen(ctx, "list:len")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)
}

// ==================== BuildKey 测试 ====================

func TestBuildKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		parts    []string
		expected string
	}{
		{
			name:     "single part",
			prefix:   KeyPrefixUser,
			parts:    []string{"12345"},
			expected: "user:12345",
		},
		{
			name:     "multiple parts",
			prefix:   KeyPrefixOrder,
			parts:    []string{"2026", "01", "11"},
			expected: "order:2026:01:11",
		},
		{
			name:     "sms code key",
			prefix:   KeyPrefixSMSCode,
			parts:    []string{"13800138000", "login"},
			expected: "sms:code:13800138000:login",
		},
		{
			name:     "rate limit key",
			prefix:   KeyPrefixRateLimit,
			parts:    []string{"api", "v1", "orders"},
			expected: "ratelimit:api:v1:orders",
		},
		{
			name:     "lock key",
			prefix:   KeyPrefixLock,
			parts:    []string{"payment", "ORD123"},
			expected: "lock:payment:ORD123",
		},
		{
			name:     "device state key",
			prefix:   KeyPrefixDeviceState,
			parts:    []string{"D20260111001"},
			expected: "device:state:D20260111001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildKey(tt.prefix, tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== 缓存键前缀常量测试 ====================

func TestCacheKeyPrefixes(t *testing.T) {
	assert.Equal(t, "user:", KeyPrefixUser)
	assert.Equal(t, "device:", KeyPrefixDevice)
	assert.Equal(t, "order:", KeyPrefixOrder)
	assert.Equal(t, "session:", KeyPrefixSession)
	assert.Equal(t, "sms:code:", KeyPrefixSMSCode)
	assert.Equal(t, "ratelimit:", KeyPrefixRateLimit)
	assert.Equal(t, "lock:", KeyPrefixLock)
	assert.Equal(t, "device:state:", KeyPrefixDeviceState)
}

// ==================== 复杂数据结构测试 ====================

func TestSet_ComplexStruct(t *testing.T) {
	s := setupMiniRedis(t)
	setupTestRedis(t, s)
	ctx := context.Background()

	type Address struct {
		City   string `json:"city"`
		Street string `json:"street"`
	}
	type User struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		Addresses []Address `json:"addresses"`
		Tags      []string  `json:"tags"`
		CreatedAt time.Time `json:"created_at"`
	}

	user := User{
		ID:    12345,
		Name:  "张三",
		Email: "zhangsan@example.com",
		Addresses: []Address{
			{City: "北京", Street: "长安街"},
			{City: "上海", Street: "南京路"},
		},
		Tags:      []string{"vip", "active"},
		CreatedAt: time.Now().Truncate(time.Second),
	}

	err := Set(ctx, "user:complex", user, time.Hour)
	assert.NoError(t, err)

	var result User
	err = Get(ctx, "user:complex", &result)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Name, result.Name)
	assert.Equal(t, user.Email, result.Email)
	assert.Len(t, result.Addresses, 2)
	assert.Equal(t, user.Addresses[0].City, result.Addresses[0].City)
	assert.Equal(t, user.Tags, result.Tags)
}
