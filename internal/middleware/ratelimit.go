// Package middleware 提供 HTTP 中间件
package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	RedisClient *redis.Client
	KeyPrefix   string        // Redis 键前缀
	Limit       int           // 限制次数
	Window      time.Duration // 时间窗口
	KeyFunc     func(*gin.Context) string // 自定义键生成函数
}

// DefaultRateLimitConfig 默认限流配置
func DefaultRateLimitConfig(redisClient *redis.Client) *RateLimitConfig {
	return &RateLimitConfig{
		RedisClient: redisClient,
		KeyPrefix:   "ratelimit:",
		Limit:       100,
		Window:      time.Minute,
		KeyFunc:     nil,
	}
}

// RateLimit 限流中间件
func RateLimit(config *RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var key string
		if config.KeyFunc != nil {
			key = config.KeyFunc(c)
		} else {
			// 默认使用 IP + 路径作为键
			key = fmt.Sprintf("%s%s:%s", config.KeyPrefix, c.ClientIP(), c.Request.URL.Path)
		}

		ctx := context.Background()

		// 使用 Redis 实现滑动窗口限流
		count, err := config.RedisClient.Incr(ctx, key).Result()
		if err != nil {
			// Redis 错误时放行
			c.Next()
			return
		}

		// 首次请求设置过期时间
		if count == 1 {
			config.RedisClient.Expire(ctx, key, config.Window)
		}

		// 超过限制
		if int(count) > config.Limit {
			// 获取剩余时间
			ttl, _ := config.RedisClient.TTL(ctx, key).Result()
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(ttl).Unix()))
			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))

			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		// 设置响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", config.Limit-int(count)))

		c.Next()
	}
}

// IPRateLimit IP 限流中间件
func IPRateLimit(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	config := &RateLimitConfig{
		RedisClient: redisClient,
		KeyPrefix:   "ratelimit:ip:",
		Limit:       limit,
		Window:      window,
		KeyFunc: func(c *gin.Context) string {
			return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
		},
	}
	return RateLimit(config)
}

// UserRateLimit 用户限流中间件
func UserRateLimit(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	config := &RateLimitConfig{
		RedisClient: redisClient,
		KeyPrefix:   "ratelimit:user:",
		Limit:       limit,
		Window:      window,
		KeyFunc: func(c *gin.Context) string {
			userID := GetUserID(c)
			if userID > 0 {
				return fmt.Sprintf("ratelimit:user:%d", userID)
			}
			return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
		},
	}
	return RateLimit(config)
}

// APIRateLimit API 接口限流中间件
func APIRateLimit(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	config := &RateLimitConfig{
		RedisClient: redisClient,
		KeyPrefix:   "ratelimit:api:",
		Limit:       limit,
		Window:      window,
		KeyFunc: func(c *gin.Context) string {
			userID := GetUserID(c)
			if userID > 0 {
				return fmt.Sprintf("ratelimit:api:%d:%s", userID, c.Request.URL.Path)
			}
			return fmt.Sprintf("ratelimit:api:%s:%s", c.ClientIP(), c.Request.URL.Path)
		},
	}
	return RateLimit(config)
}

// SmsRateLimit 短信发送限流
func SmsRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		phone := c.PostForm("phone")
		if phone == "" {
			c.Next()
			return
		}

		ctx := context.Background()
		key := fmt.Sprintf("ratelimit:sms:%s", phone)

		// 每分钟最多发送 1 条
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			redisClient.Expire(ctx, key, time.Minute)
		}

		if count > 1 {
			response.TooManyRequests(c, "短信发送过于频繁，请稍后再试")
			c.Abort()
			return
		}

		// 每天最多发送 10 条
		dayKey := fmt.Sprintf("ratelimit:sms:day:%s", phone)
		dayCount, _ := redisClient.Incr(ctx, dayKey).Result()
		if dayCount == 1 {
			// 设置到当天结束的过期时间
			now := time.Now()
			endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
			redisClient.ExpireAt(ctx, dayKey, endOfDay)
		}

		if dayCount > 10 {
			response.TooManyRequests(c, "今日短信发送次数已达上限")
			c.Abort()
			return
		}

		c.Next()
	}
}
