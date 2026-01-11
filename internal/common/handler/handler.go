// Package handler 提供 API Handler 的通用辅助函数
// 用于减少 Handler 层的代码重复，统一错误处理、认证检查、参数解析等操作
package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/common/utils"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
)

// ============================================================================
// Phase 1: 统一错误处理
// ============================================================================

// HandleError 处理错误并发送适当的响应
// 如果 err 为 nil，返回 false（表示无错误需要处理）
// 如果 err 不为 nil，发送错误响应并返回 true（表示已处理错误，调用方应该 return）
//
// 使用示例:
//
//	result, err := service.DoSomething()
//	if HandleError(c, err) {
//	    return
//	}
func HandleError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if appErr, ok := err.(*errors.AppError); ok {
		response.Error(c, appErr.Code, appErr.Message)
		return true
	}
	response.InternalError(c, err.Error())
	return true
}

// HandleErrorWithMessage 处理错误，对非 AppError 使用自定义消息
// 适用于需要隐藏内部错误详情的场景
//
// 使用示例:
//
//	result, err := service.DoSomething()
//	if HandleErrorWithMessage(c, err, "操作失败") {
//	    return
//	}
func HandleErrorWithMessage(c *gin.Context, err error, message string) bool {
	if err == nil {
		return false
	}
	if appErr, ok := err.(*errors.AppError); ok {
		response.Error(c, appErr.Code, appErr.Message)
		return true
	}
	response.InternalError(c, message)
	return true
}

// MustSucceed 便捷封装：如果有错误则返回错误响应，否则返回成功响应
// 适用于简单的「调用服务 -> 返回结果」场景
//
// 使用示例:
//
//	result, err := service.GetData()
//	MustSucceed(c, err, result)
//	return  // 注意：调用 MustSucceed 后必须 return
func MustSucceed(c *gin.Context, err error, data interface{}) {
	if HandleError(c, err) {
		return
	}
	response.Success(c, data)
}

// MustSucceedWithMessage 便捷封装：带自定义成功消息
func MustSucceedWithMessage(c *gin.Context, err error, message string, data interface{}) {
	if HandleError(c, err) {
		return
	}
	response.SuccessWithMessage(c, message, data)
}

// MustSucceedPage 便捷封装：分页响应版本
//
// 使用示例:
//
//	list, total, err := service.GetList(offset, limit)
//	MustSucceedPage(c, err, list, total, page, pageSize)
//	return
func MustSucceedPage(c *gin.Context, err error, list interface{}, total int64, page, pageSize int) {
	if HandleError(c, err) {
		return
	}
	response.SuccessPage(c, list, total, page, pageSize)
}

// ============================================================================
// Phase 2: 用户认证检查
// ============================================================================

// RequireUserID 获取当前用户ID，如果未登录则返回401响应
// 返回 (userID, true) 表示已登录
// 返回 (0, false) 表示未登录（已发送响应，调用方应该 return）
//
// 使用示例:
//
//	userID, ok := handler.RequireUserID(c)
//	if !ok {
//	    return
//	}
func RequireUserID(c *gin.Context) (int64, bool) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return 0, false
	}
	return userID, true
}

// RequireAdminID 获取当前管理员ID，如果未登录则返回401响应
// 语义上用于管理员 Handler，实际实现与 RequireUserID 相同
func RequireAdminID(c *gin.Context) (int64, bool) {
	adminID := middleware.GetAdminID(c)
	if adminID == 0 {
		response.Unauthorized(c, "请先登录")
		return 0, false
	}
	return adminID, true
}

// GetOptionalUserID 获取当前用户ID（可选）
// 如果未登录返回0，不会发送错误响应
// 适用于认证可选的接口（如商品列表可以不登录访问，但登录后可显示个性化内容）
func GetOptionalUserID(c *gin.Context) int64 {
	return middleware.GetUserID(c)
}

// ============================================================================
// Phase 3: ID 参数解析
// ============================================================================

// ParseID 解析路径参数 "id" 为 int64
// 返回 (id, true) 表示解析成功
// 返回 (0, false) 表示解析失败（已发送400响应，调用方应该 return）
//
// 使用示例:
//
//	id, ok := handler.ParseID(c, "订单")
//	if !ok {
//	    return
//	}
func ParseID(c *gin.Context, resourceName string) (int64, bool) {
	return ParseParamID(c, "id", resourceName)
}

// ParseParamID 解析指定路径参数为 int64
// paramName: 路径参数名称（如 "id", "hotel_id", "room_id"）
// resourceName: 资源名称，用于错误消息（如 "酒店", "房间"）
//
// 使用示例:
//
//	hotelID, ok := handler.ParseParamID(c, "hotel_id", "酒店")
//	if !ok {
//	    return
//	}
func ParseParamID(c *gin.Context, paramName, resourceName string) (int64, bool) {
	idStr := c.Param(paramName)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的"+resourceName+"ID")
		return 0, false
	}
	return id, true
}

// ParseQueryID 解析查询参数中的可选 ID
// 如果参数为空返回 (nil, true)
// 如果解析失败返回 (nil, false)（已发送400响应）
// 如果解析成功返回 (*id, true)
//
// 使用示例:
//
//	venueID, ok := handler.ParseQueryID(c, "venue_id", "场地")
//	if !ok {
//	    return
//	}
//	if venueID != nil {
//	    filters["venue_id"] = *venueID
//	}
func ParseQueryID(c *gin.Context, paramName, resourceName string) (*int64, bool) {
	idStr := c.Query(paramName)
	if idStr == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的"+resourceName+"ID")
		return nil, false
	}
	return &id, true
}

// ParseRequiredQueryID 解析查询参数中的必填 ID
// 如果参数为空或解析失败返回 (0, false)（已发送400响应）
//
// 使用示例:
//
//	deviceID, ok := handler.ParseRequiredQueryID(c, "device_id", "设备")
//	if !ok {
//	    return
//	}
func ParseRequiredQueryID(c *gin.Context, paramName, resourceName string) (int64, bool) {
	idStr := c.Query(paramName)
	if idStr == "" {
		response.BadRequest(c, "请提供"+resourceName+"ID")
		return 0, false
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的"+resourceName+"ID")
		return 0, false
	}
	return id, true
}

// ============================================================================
// Phase 4: 时间解析辅助
// ============================================================================

// 时间格式常量
const (
	DateFormat         = "2006-01-02"
	DateTimeFormat     = "2006-01-02 15:04:05"
	DateTimeFormatISO  = "2006-01-02T15:04:05Z07:00"
	DateTimeFormatISO2 = "2006-01-02T15:04:05"
	DateTimeFormatMin  = "2006-01-02 15:04"
)

// 支持的日期时间格式列表
var dateTimeFormats = []string{
	DateTimeFormatISO,
	DateTimeFormat,
	DateTimeFormatISO2,
	DateTimeFormatMin,
}

// ParseDate 解析日期字符串 (YYYY-MM-DD)
func ParseDate(s string) (time.Time, error) {
	return time.Parse(DateFormat, s)
}

// ParseDateTime 解析日期时间字符串，支持多种格式
func ParseDateTime(s string) (time.Time, error) {
	for _, format := range dateTimeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.ErrInvalidParams.WithMessage("时间格式错误")
}

// ParseQueryDate 从查询参数解析日期
// 返回 (nil, true) 如果参数为空
// 返回 (nil, false) 如果解析失败（已发送400响应）
// 返回 (*time, true) 如果解析成功
func ParseQueryDate(c *gin.Context, paramName, errorMsg string) (*time.Time, bool) {
	dateStr := c.Query(paramName)
	if dateStr == "" {
		return nil, true
	}
	t, err := ParseDate(dateStr)
	if err != nil {
		response.BadRequest(c, errorMsg)
		return nil, false
	}
	return &t, true
}

// ParseQueryDateRange 从查询参数解析日期范围（start_date, end_date）
// 结束日期会自动调整为当天结束时间（23:59:59）
// 返回 (nil, nil, true) 如果两个参数都为空
// 返回 (nil, nil, false) 如果解析失败（已发送400响应）
func ParseQueryDateRange(c *gin.Context) (*time.Time, *time.Time, bool) {
	var start, end *time.Time

	startStr := c.Query("start_date")
	if startStr != "" {
		t, err := ParseDate(startStr)
		if err != nil {
			response.BadRequest(c, "无效的开始日期格式")
			return nil, nil, false
		}
		start = &t
	}

	endStr := c.Query("end_date")
	if endStr != "" {
		t, err := ParseDate(endStr)
		if err != nil {
			response.BadRequest(c, "无效的结束日期格式")
			return nil, nil, false
		}
		// 设置为当天结束时间
		endOfDay := t.Add(24*time.Hour - time.Second)
		end = &endOfDay
	}

	return start, end, true
}

// ParseRequiredQueryDateRange 从查询参数解析必填的日期范围
// 返回 (zero, zero, false) 如果任一参数为空或解析失败（已发送400响应）
func ParseRequiredQueryDateRange(c *gin.Context) (time.Time, time.Time, bool) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	if startStr == "" || endStr == "" {
		response.BadRequest(c, "请指定开始和结束日期")
		return time.Time{}, time.Time{}, false
	}

	startDate, err := ParseDate(startStr)
	if err != nil {
		response.BadRequest(c, "无效的开始日期格式")
		return time.Time{}, time.Time{}, false
	}

	endDate, err := ParseDate(endStr)
	if err != nil {
		response.BadRequest(c, "无效的结束日期格式")
		return time.Time{}, time.Time{}, false
	}

	// 设置结束日期为当天结束时间
	endDate = endDate.Add(24*time.Hour - time.Second)

	return startDate, endDate, true
}

// ============================================================================
// Phase 5: 分页处理
// ============================================================================

// BindPagination 从查询参数绑定并规范化分页参数
// 默认 page=1, pageSize=10, 最大 pageSize=100
//
// 使用示例:
//
//	p := handler.BindPagination(c)
//	list, total, err := service.GetList(p.GetOffset(), p.GetLimit())
//	MustSucceedPage(c, err, list, total, p.Page, p.PageSize)
func BindPagination(c *gin.Context) utils.Pagination {
	var p utils.Pagination
	p.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	p.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "10"))
	p.Normalize()
	return p
}

// BindPaginationWithDefaults 从查询参数绑定分页参数，使用自定义默认值
func BindPaginationWithDefaults(c *gin.Context, defaultPage, defaultPageSize int) utils.Pagination {
	var p utils.Pagination
	p.Page, _ = strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(defaultPage)))
	p.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultPageSize)))
	p.Normalize()
	return p
}

// ============================================================================
// 组合辅助函数
// ============================================================================

// RequireUserAndParseID 组合：检查用户登录 + 解析ID参数
// 适用于大多数需要登录且操作特定资源的接口
//
// 使用示例:
//
//	userID, resourceID, ok := handler.RequireUserAndParseID(c, "订单")
//	if !ok {
//	    return
//	}
func RequireUserAndParseID(c *gin.Context, resourceName string) (userID, resourceID int64, ok bool) {
	userID, ok = RequireUserID(c)
	if !ok {
		return 0, 0, false
	}
	resourceID, ok = ParseID(c, resourceName)
	if !ok {
		return 0, 0, false
	}
	return userID, resourceID, true
}

// RequireAdminAndParseID 组合：检查管理员登录 + 解析ID参数
func RequireAdminAndParseID(c *gin.Context, resourceName string) (adminID, resourceID int64, ok bool) {
	adminID, ok = RequireAdminID(c)
	if !ok {
		return 0, 0, false
	}
	resourceID, ok = ParseID(c, resourceName)
	if !ok {
		return 0, 0, false
	}
	return adminID, resourceID, true
}
