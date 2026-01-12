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
	// 设备管理
	"POST /admin/devices": {
		Module:     "device",
		Action:     "create",
		TargetType: "device",
	},
	"PUT /admin/devices/:id": {
		Module:     "device",
		Action:     "update",
		TargetType: "device",
	},
	"PUT /admin/devices/:id/status": {
		Module:     "device",
		Action:     "update_status",
		TargetType: "device",
	},
	"DELETE /admin/devices/:id": {
		Module:     "device",
		Action:     "delete",
		TargetType: "device",
	},
	"POST /admin/devices/:id/unlock": {
		Module:     "device",
		Action:     "remote_unlock",
		TargetType: "device",
	},
	"POST /admin/devices/:id/lock": {
		Module:     "device",
		Action:     "remote_lock",
		TargetType: "device",
	},
	"POST /admin/devices/maintenance": {
		Module:     "device",
		Action:     "create_maintenance",
		TargetType: "device_maintenance",
	},
	"POST /admin/devices/maintenance/:id/complete": {
		Module:     "device",
		Action:     "complete_maintenance",
		TargetType: "device_maintenance",
	},

	// 场地管理
	"POST /admin/venues": {
		Module:     "venue",
		Action:     "create",
		TargetType: "venue",
	},
	"PUT /admin/venues/:id": {
		Module:     "venue",
		Action:     "update",
		TargetType: "venue",
	},
	"PUT /admin/venues/:id/status": {
		Module:     "venue",
		Action:     "update_status",
		TargetType: "venue",
	},
	"DELETE /admin/venues/:id": {
		Module:     "venue",
		Action:     "delete",
		TargetType: "venue",
	},

	// 商户管理
	"POST /admin/merchants": {
		Module:     "merchant",
		Action:     "create",
		TargetType: "merchant",
	},
	"PUT /admin/merchants/:id": {
		Module:     "merchant",
		Action:     "update",
		TargetType: "merchant",
	},
	"PUT /admin/merchants/:id/status": {
		Module:     "merchant",
		Action:     "update_status",
		TargetType: "merchant",
	},
	"DELETE /admin/merchants/:id": {
		Module:     "merchant",
		Action:     "delete",
		TargetType: "merchant",
	},

	// 商品管理
	"POST /admin/products": {
		Module:     "product",
		Action:     "create",
		TargetType: "product",
	},
	"PUT /admin/products/:id": {
		Module:     "product",
		Action:     "update",
		TargetType: "product",
	},
	"PUT /admin/products/:id/status": {
		Module:     "product",
		Action:     "update_status",
		TargetType: "product",
	},
	"DELETE /admin/products/:id": {
		Module:     "product",
		Action:     "delete",
		TargetType: "product",
	},

	// 分类管理
	"POST /admin/categories": {
		Module:     "category",
		Action:     "create",
		TargetType: "category",
	},
	"PUT /admin/categories/:id": {
		Module:     "category",
		Action:     "update",
		TargetType: "category",
	},
	"DELETE /admin/categories/:id": {
		Module:     "category",
		Action:     "delete",
		TargetType: "category",
	},

	// 酒店管理
	"POST /admin/hotels": {
		Module:     "hotel",
		Action:     "create",
		TargetType: "hotel",
	},
	"PUT /admin/hotels/:id": {
		Module:     "hotel",
		Action:     "update",
		TargetType: "hotel",
	},
	"PUT /admin/hotels/:id/status": {
		Module:     "hotel",
		Action:     "update_status",
		TargetType: "hotel",
	},
	"DELETE /admin/hotels/:id": {
		Module:     "hotel",
		Action:     "delete",
		TargetType: "hotel",
	},
	"POST /admin/hotels/:id/rooms": {
		Module:     "hotel",
		Action:     "create_room",
		TargetType: "room",
	},
	"PUT /admin/rooms/:id": {
		Module:     "hotel",
		Action:     "update_room",
		TargetType: "room",
	},
	"DELETE /admin/rooms/:id": {
		Module:     "hotel",
		Action:     "delete_room",
		TargetType: "room",
	},

	// 营销管理 - 优惠券
	"POST /admin/marketing/coupons": {
		Module:     "marketing",
		Action:     "create_coupon",
		TargetType: "coupon",
	},
	"PUT /admin/marketing/coupons/:id": {
		Module:     "marketing",
		Action:     "update_coupon",
		TargetType: "coupon",
	},
	"PUT /admin/marketing/coupons/:id/status": {
		Module:     "marketing",
		Action:     "update_coupon_status",
		TargetType: "coupon",
	},
	"DELETE /admin/marketing/coupons/:id": {
		Module:     "marketing",
		Action:     "delete_coupon",
		TargetType: "coupon",
	},

	// 营销管理 - 活动
	"POST /admin/marketing/campaigns": {
		Module:     "marketing",
		Action:     "create_campaign",
		TargetType: "campaign",
	},
	"PUT /admin/marketing/campaigns/:id": {
		Module:     "marketing",
		Action:     "update_campaign",
		TargetType: "campaign",
	},
	"PUT /admin/marketing/campaigns/:id/status": {
		Module:     "marketing",
		Action:     "update_campaign_status",
		TargetType: "campaign",
	},
	"DELETE /admin/marketing/campaigns/:id": {
		Module:     "marketing",
		Action:     "delete_campaign",
		TargetType: "campaign",
	},

	// 会员管理
	"POST /admin/member/levels": {
		Module:     "member",
		Action:     "create_level",
		TargetType: "member_level",
	},
	"PUT /admin/member/levels/:id": {
		Module:     "member",
		Action:     "update_level",
		TargetType: "member_level",
	},
	"DELETE /admin/member/levels/:id": {
		Module:     "member",
		Action:     "delete_level",
		TargetType: "member_level",
	},
	"POST /admin/member/packages": {
		Module:     "member",
		Action:     "create_package",
		TargetType: "member_package",
	},
	"PUT /admin/member/packages/:id": {
		Module:     "member",
		Action:     "update_package",
		TargetType: "member_package",
	},
	"PUT /admin/member/packages/:id/status": {
		Module:     "member",
		Action:     "update_package_status",
		TargetType: "member_package",
	},
	"DELETE /admin/member/packages/:id": {
		Module:     "member",
		Action:     "delete_package",
		TargetType: "member_package",
	},

	// 分销管理
	"POST /admin/distribution/distributors/:id/approve": {
		Module:     "distribution",
		Action:     "approve_distributor",
		TargetType: "distributor",
	},
	"POST /admin/distribution/withdrawals/:id/handle": {
		Module:     "distribution",
		Action:     "handle_withdrawal",
		TargetType: "withdrawal",
	},

	// 财务管理
	"POST /admin/finance/settlements": {
		Module:     "finance",
		Action:     "create_settlement",
		TargetType: "settlement",
	},
	"POST /admin/finance/settlements/generate": {
		Module:     "finance",
		Action:     "generate_settlements",
		TargetType: "settlement",
	},
	"POST /admin/finance/settlements/:id/process": {
		Module:     "finance",
		Action:     "process_settlement",
		TargetType: "settlement",
	},
	"POST /admin/finance/withdrawals/:id/handle": {
		Module:     "finance",
		Action:     "handle_withdrawal",
		TargetType: "withdrawal",
	},
	"POST /admin/finance/withdrawals/batch": {
		Module:     "finance",
		Action:     "batch_handle_withdrawals",
		TargetType: "withdrawal",
	},

	// 认证管理
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

	// 用户管理
	"PUT /admin/users/:id/status": {
		Module:     "user",
		Action:     "update_status",
		TargetType: "user",
	},

	// 订单管理
	"POST /admin/orders/:id/refund": {
		Module:     "order",
		Action:     "refund",
		TargetType: "order",
	},

	// 系统管理 - 管理员
	"POST /admin/admins": {
		Module:     "system",
		Action:     "create_admin",
		TargetType: "admin",
	},
	"PUT /admin/admins/:id": {
		Module:     "system",
		Action:     "update_admin",
		TargetType: "admin",
	},
	"DELETE /admin/admins/:id": {
		Module:     "system",
		Action:     "delete_admin",
		TargetType: "admin",
	},

	// 系统管理 - 角色
	"POST /admin/roles": {
		Module:     "system",
		Action:     "create_role",
		TargetType: "role",
	},
	"PUT /admin/roles/:id": {
		Module:     "system",
		Action:     "update_role",
		TargetType: "role",
	},
	"DELETE /admin/roles/:id": {
		Module:     "system",
		Action:     "delete_role",
		TargetType: "role",
	},

	// 系统管理 - 配置
	"PUT /admin/configs": {
		Module: "system",
		Action: "update_config",
	},

	// 系统管理 - 轮播图
	"POST /admin/banners": {
		Module:     "content",
		Action:     "create_banner",
		TargetType: "banner",
	},
	"PUT /admin/banners/:id": {
		Module:     "content",
		Action:     "update_banner",
		TargetType: "banner",
	},
	"DELETE /admin/banners/:id": {
		Module:     "content",
		Action:     "delete_banner",
		TargetType: "banner",
	},

	// 系统管理 - 文章
	"POST /admin/articles": {
		Module:     "content",
		Action:     "create_article",
		TargetType: "article",
	},
	"PUT /admin/articles/:id": {
		Module:     "content",
		Action:     "update_article",
		TargetType: "article",
	},
	"DELETE /admin/articles/:id": {
		Module:     "content",
		Action:     "delete_article",
		TargetType: "article",
	},

	// 系统管理 - 反馈
	"PUT /admin/feedbacks/:id/reply": {
		Module:     "content",
		Action:     "reply_feedback",
		TargetType: "feedback",
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

	// 从路径推断模块（按优先级排序）
	module := "unknown"
	switch {
	case strings.Contains(path, "/devices"):
		module = "device"
	case strings.Contains(path, "/venues"):
		module = "venue"
	case strings.Contains(path, "/merchants"):
		module = "merchant"
	case strings.Contains(path, "/products"):
		module = "product"
	case strings.Contains(path, "/categories"):
		module = "category"
	case strings.Contains(path, "/hotels"):
		module = "hotel"
	case strings.Contains(path, "/rooms"):
		module = "hotel"
	case strings.Contains(path, "/bookings"):
		module = "booking"
	case strings.Contains(path, "/marketing"):
		module = "marketing"
	case strings.Contains(path, "/member"):
		module = "member"
	case strings.Contains(path, "/distribution"):
		module = "distribution"
	case strings.Contains(path, "/finance"):
		module = "finance"
	case strings.Contains(path, "/orders"):
		module = "order"
	case strings.Contains(path, "/users"):
		module = "user"
	case strings.Contains(path, "/admins"):
		module = "system"
	case strings.Contains(path, "/roles"):
		module = "system"
	case strings.Contains(path, "/permissions"):
		module = "system"
	case strings.Contains(path, "/configs"):
		module = "system"
	case strings.Contains(path, "/banners"):
		module = "content"
	case strings.Contains(path, "/articles"):
		module = "content"
	case strings.Contains(path, "/feedbacks"):
		module = "content"
	case strings.Contains(path, "/auth"):
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
