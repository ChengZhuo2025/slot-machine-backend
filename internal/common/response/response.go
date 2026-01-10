// Package response 提供统一的 API 响应格式
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response API 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData 分页数据结构
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// ListData 列表数据结构 (Swagger文档使用)
type ListData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（带消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// SuccessPage 分页成功响应
func SuccessPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// SuccessWithPage 分页成功响应（别名）
func SuccessWithPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	SuccessPage(c, list, total, page, pageSize)
}

// SuccessList 列表成功响应（支持可变参数）
// 调用方式1: SuccessList(c, list, total) - 不带分页
// 调用方式2: SuccessList(c, list, total, page, pageSize) - 带分页
func SuccessList(c *gin.Context, list interface{}, total int64, pageInfo ...int) {
	page := 1
	pageSize := 20
	if len(pageInfo) >= 2 {
		page = pageInfo[0]
		pageSize = pageInfo[1]
	}
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: ListData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithData 错误响应（带数据）
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// BadRequest 请求参数错误
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    400,
		Message: message,
	})
}

// Unauthorized 未授权
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	c.JSON(http.StatusUnauthorized, Response{
		Code:    401,
		Message: message,
	})
}

// Forbidden 禁止访问
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	c.JSON(http.StatusForbidden, Response{
		Code:    403,
		Message: message,
	})
}

// NotFound 资源不存在
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "not found"
	}
	c.JSON(http.StatusNotFound, Response{
		Code:    404,
		Message: message,
	})
}

// InternalError 服务器内部错误
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	c.JSON(http.StatusInternalServerError, Response{
		Code:    500,
		Message: message,
	})
}

// TooManyRequests 请求过于频繁
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "too many requests"
	}
	c.JSON(http.StatusTooManyRequests, Response{
		Code:    429,
		Message: message,
	})
}
