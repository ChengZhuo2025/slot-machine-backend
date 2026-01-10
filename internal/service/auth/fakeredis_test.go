package auth

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type memRedisValue struct {
	value     string
	expiresAt *time.Time
}

type memRedis struct {
	now func() time.Time

	mu   sync.Mutex
	data map[string]memRedisValue
}

func newMemRedis(now func() time.Time) *memRedis {
	return &memRedis{
		now:  now,
		data: make(map[string]memRedisValue),
	}
}

func (m *memRedis) purgeExpiredLocked(key string) bool {
	v, ok := m.data[key]
	if !ok {
		return false
	}
	if v.expiresAt == nil || m.now().Before(*v.expiresAt) {
		return true
	}
	delete(m.data, key)
	return false
}

func (m *memRedis) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "EXISTS", keys)
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for _, key := range keys {
		if m.purgeExpiredLocked(key) {
			count++
		}
	}
	cmd.SetVal(count)
	return cmd
}

func (m *memRedis) Incr(ctx context.Context, key string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "INCR", key)
	m.mu.Lock()
	defer m.mu.Unlock()

	var existingExpiresAt *time.Time
	var current int64
	if m.purgeExpiredLocked(key) {
		v := m.data[key]
		existingExpiresAt = v.expiresAt
		n, err := strconv.ParseInt(v.value, 10, 64)
		if err != nil {
			cmd.SetErr(fmt.Errorf("value is not an integer"))
			return cmd
		}
		current = n
	}
	current++
	m.data[key] = memRedisValue{value: strconv.FormatInt(current, 10), expiresAt: existingExpiresAt}
	cmd.SetVal(current)
	return cmd
}

func (m *memRedis) ExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx, "EXPIREAT", key, tm.Unix())
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.purgeExpiredLocked(key) {
		cmd.SetVal(false)
		return cmd
	}
	v := m.data[key]
	v.expiresAt = &tm
	m.data[key] = v
	cmd.SetVal(true)
	return cmd
}

func (m *memRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "SET", key)
	m.mu.Lock()
	defer m.mu.Unlock()

	val := fmt.Sprint(value)
	var expiresAt *time.Time
	if expiration > 0 {
		t := m.now().Add(expiration)
		expiresAt = &t
	}
	m.data[key] = memRedisValue{value: val, expiresAt: expiresAt}
	cmd.SetVal("OK")
	return cmd
}

func (m *memRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "DEL", keys)
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		if !m.purgeExpiredLocked(key) {
			continue
		}
		delete(m.data, key)
		deleted++
	}
	cmd.SetVal(deleted)
	return cmd
}

func (m *memRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "GET", key)
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.purgeExpiredLocked(key) {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(m.data[key].value)
	return cmd
}

type testClock struct {
	now atomic.Value // time.Time
}

func newTestClock(start time.Time) *testClock {
	c := &testClock{}
	c.now.Store(start)
	return c
}

func (c *testClock) Now() time.Time {
	return c.now.Load().(time.Time)
}

func (c *testClock) Advance(d time.Duration) {
	c.now.Store(c.Now().Add(d))
}

func newTestRedisClient(t *testing.T) (redisCmdable, *testClock) {
	t.Helper()
	clock := newTestClock(time.Date(2026, 1, 10, 10, 0, 0, 0, time.Local))
	return newMemRedis(clock.Now), clock
}
