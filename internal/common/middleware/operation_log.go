// Package middleware 提供 HTTP 中间件
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// OperationLogger 操作日志中间件
type OperationLogger struct {
	repo *repository.OperationLogRepository
}

// NewOperationLogger 创建操作日志中间件
func NewOperationLogger(repo *repository.OperationLogRepository) *OperationLogger {
	return &OperationLogger{repo: repo}
}

// OperationConfig 操作配置
type OperationConfig struct {
	Module     string
	Action     string
	TargetType string
	GetTargetID func(*gin.Context) *int64
	GetBeforeData func(context.Context, *gin.Context) interface{}
}

// ModuleAction 模块操作映射
var moduleActionMap = map[string]OperationConfig{
	"POST /admin/devices": {
		Module: "device",
		Action: "create",
		TargetType: "device",
	},
	"PUT /admin/devices/:id": {
		Module: "device",
		Action: "update",
		TargetType: "device",
	},
	"PUT /admin/devices/:id/status": {
		Module: "device",
		Action: "update_status",
		TargetType: "device",
	},
	"DELETE /admin/devices/:id": {
		Module: "device",
		Action: "delete",
		TargetType: "device",
	},
	"POST /admin/devices/:id/unlock": {
		Module: "device",
		Action: "remote_unlock",
		TargetType: "device",
	},
	"POST /admin/devices/:id/lock": {
		Module: "device",
		Action: "remote_lock",
		TargetType: "device",
	},
	"POST /admin/venues": {
		Module: "venue",
		Action: "create",
		TargetType: "venue",
	},
	"PUT /admin/venues/:id": {
		Module: "venue",
		Action: "update",
		TargetType: "venue",
	},
	"PUT /admin/venues/:id/status": {
		Module: "venue",
		Action: "update_status",
		TargetType: "venue",
	},
	"DELETE /admin/venues/:id": {
		Module: "venue",
		Action: "delete",
		TargetType: "venue",
	},
	"POST /admin/merchants": {
		Module: "merchant",
		Action: "create",
		TargetType: "merchant",
	},
	"PUT /admin/merchants/:id": {
		Module: "merchant",
		Action: "update",
		TargetType: "merchant",
	},
	"PUT /admin/merchants/:id/status": {
		Module: "merchant",
		Action: "update_status",
		TargetType: "merchant",
	},
	"DELETE /admin/merchants/:id": {
		Module: "merchant",
		Action: "delete",
		TargetType: "merchant",
	},
	"POST /admin/auth/login": {
		Module: "auth",
		Action: "login",
	},
	"POST /admin/auth/logout": {
		Module: "auth",
		Action: "logout",
	},
	"PUT /admin/auth/password": {
		Module: "auth",
		Action: "change_password",
	},
}

// Log 操作日志中间件处理函数
func (l *OperationLogger) Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只记录写操作
		if !l.shouldLog(c) {
			c.Next()
			return
		}

		// 读取请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 执行处理
		c.Next()

		// 记录日志（异步）
		go l.logOperation(c, requestBody)
	}
}

// shouldLog 判断是否需要记录日志
func (l *OperationLogger) shouldLog(c *gin.Context) bool {
	method := c.Request.Method
	// 只记录写操作
	return method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH"
}

// logOperation 记录操作日志
func (l *OperationLogger) logOperation(c *gin.Context, requestBody []byte) {
	if l.repo == nil {
		return
	}

	// 获取路由配置
	path := c.FullPath()
	routeKey := c.Request.Method + " " + path
	config, ok := moduleActionMap[routeKey]
	if !ok && strings.HasPrefix(path, "/api/") {
		// 兼容路由组前缀差异：Gin full path 可能包含 /api 前缀
		altKey := c.Request.Method + " " + strings.TrimPrefix(path, "/api")
		config, ok = moduleActionMap[altKey]
	}
	if !ok {
		// 尝试获取通用配置
		config = l.getDefaultConfig(c)
	}

	// 获取管理员 ID
	adminID, ok := l.getAdminID(c)
	if !ok {
		return
	}

	// 构建日志记录
	log := &models.OperationLog{
		AdminID:   adminID,
		Module:    config.Module,
		Action:    config.Action,
		IP:        c.ClientIP(),
	}

	// 设置 User-Agent
	userAgent := c.Request.UserAgent()
	if userAgent != "" {
		log.UserAgent = &userAgent
	}

	// 设置目标类型和 ID
	if config.TargetType != "" {
		log.TargetType = &config.TargetType
		if targetID := l.getTargetID(c); targetID != nil {
			log.TargetID = targetID
		}
	}

	// 设置请求数据
	if len(requestBody) > 0 {
		var data interface{}
		if err := json.Unmarshal(requestBody, &data); err == nil {
			// 过滤敏感字段
			filteredData := l.filterSensitiveData(data)
			if mapData, ok := filteredData.(map[string]interface{}); ok {
				log.AfterData = mapData
			}
		}
	}

	// 保存日志
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = l.repo.Create(ctx, log)
}

func (l *OperationLogger) getAdminID(c *gin.Context) (int64, bool) {
	// 优先使用明确的 admin_id
	if v, ok := c.Get("admin_id"); ok {
		if id, ok := v.(int64); ok {
			return id, true
		}
	}

	// 兼容内部 JWT 中间件：AdminAuth 仅设置 user_id / user_type
	userType, _ := c.Get("user_type")
	if userTypeStr, ok := userType.(string); ok && userTypeStr == "admin" {
		if v, ok := c.Get("user_id"); ok {
			if id, ok := v.(int64); ok {
				return id, true
			}
		}
	}

	return 0, false
}

// getDefaultConfig 获取默认配置
func (l *OperationLogger) getDefaultConfig(c *gin.Context) OperationConfig {
	path := c.FullPath()
	method := c.Request.Method

	// 从路径推断模块
	module := "unknown"
	if strings.Contains(path, "/devices") {
		module = "device"
	} else if strings.Contains(path, "/venues") {
		module = "venue"
	} else if strings.Contains(path, "/merchants") {
		module = "merchant"
	} else if strings.Contains(path, "/admins") {
		module = "admin"
	} else if strings.Contains(path, "/roles") {
		module = "role"
	} else if strings.Contains(path, "/auth") {
		module = "auth"
	}

	// 从方法推断操作
	action := "unknown"
	switch method {
	case "POST":
		action = "create"
	case "PUT", "PATCH":
		action = "update"
	case "DELETE":
		action = "delete"
	}

	return OperationConfig{
		Module: module,
		Action: action,
	}
}

// getTargetID 从路径参数获取目标 ID
func (l *OperationLogger) getTargetID(c *gin.Context) *int64 {
	idStr := c.Param("id")
	if idStr == "" {
		return nil
	}

	var id int64
	if _, err := json.Number(idStr).Int64(); err == nil {
		id, _ = json.Number(idStr).Int64()
		return &id
	}
	return nil
}

// filterSensitiveData 过滤敏感数据
func (l *OperationLogger) filterSensitiveData(data interface{}) interface{} {
	sensitiveFields := []string{
		"password", "old_password", "new_password", "confirm_password",
		"token", "access_token", "refresh_token",
		"secret", "api_key", "api_secret",
		"bank_account", "bank_holder", "id_card",
	}

	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			lowerKey := strings.ToLower(key)
			isSensitive := false
			for _, sf := range sensitiveFields {
				if strings.Contains(lowerKey, sf) {
					isSensitive = true
					break
				}
			}
			if isSensitive {
				result[key] = "***"
			} else {
				result[key] = l.filterSensitiveData(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = l.filterSensitiveData(item)
		}
		return result
	default:
		return data
	}
}

// LogWithConfig 使用自定义配置记录操作日志
func (l *OperationLogger) LogWithConfig(config OperationConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 读取请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 执行处理
		c.Next()

		// 记录日志
		go l.logOperationWithConfig(c, requestBody, config)
	}
}

// logOperationWithConfig 使用自定义配置记录操作日志
func (l *OperationLogger) logOperationWithConfig(c *gin.Context, requestBody []byte, config OperationConfig) {
	if l.repo == nil {
		return
	}

	// 获取管理员 ID
	adminID, exists := c.Get("admin_id")
	if !exists {
		return
	}

	// 构建日志记录
	log := &models.OperationLog{
		AdminID: adminID.(int64),
		Module:  config.Module,
		Action:  config.Action,
		IP:      c.ClientIP(),
	}

	// 设置 User-Agent
	userAgent := c.Request.UserAgent()
	if userAgent != "" {
		log.UserAgent = &userAgent
	}

	// 设置目标类型和 ID
	if config.TargetType != "" {
		log.TargetType = &config.TargetType
	}
	if config.GetTargetID != nil {
		log.TargetID = config.GetTargetID(c)
	} else if targetID := l.getTargetID(c); targetID != nil {
		log.TargetID = targetID
	}

	// 设置请求数据
	if len(requestBody) > 0 {
		var data interface{}
		if err := json.Unmarshal(requestBody, &data); err == nil {
			filteredData := l.filterSensitiveData(data)
			if mapData, ok := filteredData.(map[string]interface{}); ok {
				log.AfterData = mapData
			}
		}
	}

	// 保存日志
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = l.repo.Create(ctx, log)
}
