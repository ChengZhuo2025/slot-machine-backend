// Package middleware 提供 HTTP 中间件
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
)

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	HasPermission(roleCode, permissionCode string) bool
	HasAnyPermission(roleCode string, permissionCodes []string) bool
	HasAllPermissions(roleCode string, permissionCodes []string) bool
}

// PermissionConfig 权限配置
type PermissionConfig struct {
	Checker PermissionChecker
}

// RequirePermission 要求指定权限
func RequirePermission(checker PermissionChecker, permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetRole(c)
		if role == "" {
			response.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		if !checker.HasPermission(role, permissionCode) {
			response.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission 要求任一权限
func RequireAnyPermission(checker PermissionChecker, permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetRole(c)
		if role == "" {
			response.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		if !checker.HasAnyPermission(role, permissionCodes) {
			response.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllPermissions 要求全部权限
func RequireAllPermissions(checker PermissionChecker, permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetRole(c)
		if role == "" {
			response.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		if !checker.HasAllPermissions(role, permissionCodes) {
			response.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRoles 要求指定角色
func RequireRoles(roles ...string) gin.HandlerFunc {
	roleSet := make(map[string]struct{})
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role := GetRole(c)
		if role == "" {
			response.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		if _, ok := roleSet[role]; !ok {
			response.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireSuperAdmin 要求超级管理员权限
func RequireSuperAdmin() gin.HandlerFunc {
	return RequireRoles("super_admin")
}

// PermissionCodes 预定义权限码
const (
	// 用户管理
	PermissionUserList   = "user:list"
	PermissionUserCreate = "user:create"
	PermissionUserUpdate = "user:update"
	PermissionUserDelete = "user:delete"

	// 设备管理
	PermissionDeviceList   = "device:list"
	PermissionDeviceCreate = "device:create"
	PermissionDeviceUpdate = "device:update"
	PermissionDeviceDelete = "device:delete"
	PermissionDeviceControl = "device:control"

	// 订单管理
	PermissionOrderList   = "order:list"
	PermissionOrderView   = "order:view"
	PermissionOrderUpdate = "order:update"
	PermissionOrderRefund = "order:refund"

	// 商品管理
	PermissionProductList   = "product:list"
	PermissionProductCreate = "product:create"
	PermissionProductUpdate = "product:update"
	PermissionProductDelete = "product:delete"

	// 财务管理
	PermissionFinanceView      = "finance:view"
	PermissionFinanceSettle    = "finance:settle"
	PermissionFinanceWithdraw  = "finance:withdraw"

	// 营销管理
	PermissionMarketingList   = "marketing:list"
	PermissionMarketingCreate = "marketing:create"
	PermissionMarketingUpdate = "marketing:update"
	PermissionMarketingDelete = "marketing:delete"

	// 系统管理
	PermissionSystemConfig = "system:config"
	PermissionSystemLog    = "system:log"
	PermissionSystemAdmin  = "system:admin"
	PermissionSystemRole   = "system:role"
)
