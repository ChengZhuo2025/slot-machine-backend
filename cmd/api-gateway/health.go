// Package main 是应用程序入口
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// healthHandler 健康检查（简单版）
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// pingHandler Ping 检查
func pingHandler(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

// readyHandler 就绪检查（检查依赖服务）
func readyHandler(db *gorm.DB, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		checks := make(map[string]interface{})
		allHealthy := true

		// 检查数据库连接
		dbStatus := "ok"
		sqlDB, err := db.DB()
		if err != nil {
			dbStatus = "error: " + err.Error()
			allHealthy = false
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := sqlDB.PingContext(ctx); err != nil {
				dbStatus = "error: " + err.Error()
				allHealthy = false
			}
		}
		checks["database"] = dbStatus

		// 检查 Redis 连接
		redisStatus := "ok"
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := redisClient.Ping(ctx).Result(); err != nil {
			redisStatus = "error: " + err.Error()
			allHealthy = false
		}
		checks["redis"] = redisStatus

		// 返回结果
		status := http.StatusOK
		statusText := "ready"
		if !allHealthy {
			status = http.StatusServiceUnavailable
			statusText = "not ready"
		}

		c.JSON(status, gin.H{
			"status":    statusText,
			"timestamp": time.Now().Unix(),
			"checks":    checks,
		})
	}
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp int64                  `json:"timestamp"`
	Checks    map[string]interface{} `json:"checks,omitempty"`
}
