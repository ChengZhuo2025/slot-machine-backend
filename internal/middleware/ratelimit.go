// Package middleware 提供 HTTP 中间件
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// SmsRateLimit 短信发送限流（支持 JSON 和 Form 请求）
// 限制规则：每分钟1条，每天10条
func SmsRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从 JSON 或 Form 获取手机号
		phone := extractPhoneFromRequest(c)
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

// extractPhoneFromRequest 从请求中提取手机号（支持 JSON 和 Form）
func extractPhoneFromRequest(c *gin.Context) string {
	// 优先尝试 Form
	if phone := c.PostForm("phone"); phone != "" {
		return phone
	}

	// 尝试从 JSON body 读取
	// 读取 body 后需要放回，以便后续 handler 继续使用
	bodyBytes, err := c.GetRawData()
	if err != nil || len(bodyBytes) == 0 {
		return ""
	}

	// 将 body 放回，供后续使用
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.Unmarshal(bodyBytes, &req); err == nil && req.Phone != "" {
		return req.Phone
	}

	return ""
}

// LoginRateLimit 登录限流中间件
// 限制规则：每 IP 每分钟最多 10 次登录尝试，每小时最多 30 次
func LoginRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := context.Background()

		// 每分钟限制
		minuteKey := fmt.Sprintf("ratelimit:login:minute:%s", ip)
		minuteCount, err := redisClient.Incr(ctx, minuteKey).Result()
		if err != nil {
			c.Next()
			return
		}

		if minuteCount == 1 {
			redisClient.Expire(ctx, minuteKey, time.Minute)
		}

		if minuteCount > 10 {
			response.TooManyRequests(c, "登录尝试过于频繁，请稍后再试")
			c.Abort()
			return
		}

		// 每小时限制
		hourKey := fmt.Sprintf("ratelimit:login:hour:%s", ip)
		hourCount, _ := redisClient.Incr(ctx, hourKey).Result()
		if hourCount == 1 {
			redisClient.Expire(ctx, hourKey, time.Hour)
		}

		if hourCount > 30 {
			response.TooManyRequests(c, "登录尝试次数过多，请1小时后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// PaymentRateLimit 支付限流中间件
// 限制规则：每用户每分钟最多 5 次支付请求
func PaymentRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			// 未登录用户使用 IP 限流
			return
		}

		ctx := context.Background()
		key := fmt.Sprintf("ratelimit:payment:%d", userID)

		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			redisClient.Expire(ctx, key, time.Minute)
		}

		if count > 5 {
			response.TooManyRequests(c, "支付请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminLoginRateLimit 管理员登录限流中间件
// 限制规则：每 IP 每分钟最多 5 次，每小时最多 20 次
func AdminLoginRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := context.Background()

		// 每分钟限制（更严格）
		minuteKey := fmt.Sprintf("ratelimit:admin:login:minute:%s", ip)
		minuteCount, err := redisClient.Incr(ctx, minuteKey).Result()
		if err != nil {
			c.Next()
			return
		}

		if minuteCount == 1 {
			redisClient.Expire(ctx, minuteKey, time.Minute)
		}

		if minuteCount > 5 {
			response.TooManyRequests(c, "登录尝试过于频繁，请稍后再试")
			c.Abort()
			return
		}

		// 每小时限制
		hourKey := fmt.Sprintf("ratelimit:admin:login:hour:%s", ip)
		hourCount, _ := redisClient.Incr(ctx, hourKey).Result()
		if hourCount == 1 {
			redisClient.Expire(ctx, hourKey, time.Hour)
		}

		if hourCount > 20 {
			response.TooManyRequests(c, "登录尝试次数过多，请1小时后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
