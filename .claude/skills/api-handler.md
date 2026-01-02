# API Handler 开发技能

## 概述

本技能用于根据 OpenAPI 规范生成 Gin HTTP Handler 代码，确保 API 实现与规范文档一致。

## 技术栈

- **Web 框架**: Gin v1.10+
- **参数验证**: go-playground/validator/v10
- **API 规范**: OpenAPI 3.0 (`contracts/user-api.yaml`)

## Handler 文件结构

```
internal/handler/
├── auth/
│   └── auth_handler.go       # 认证相关
├── user/
│   └── user_handler.go       # 用户相关
├── device/
│   └── device_handler.go     # 设备相关
├── rental/
│   └── rental_handler.go     # 租借相关
├── order/
│   ├── order_handler.go      # 订单相关
│   └── refund_handler.go     # 退款相关
├── payment/
│   ├── payment_handler.go    # 支付相关
│   └── callback_handler.go   # 支付回调
├── hotel/
│   ├── hotel_handler.go      # 酒店相关
│   └── booking_handler.go    # 预订相关
├── mall/
│   ├── product_handler.go    # 商品相关
│   ├── cart_handler.go       # 购物车
│   └── order_handler.go      # 商城订单
├── distribution/
│   └── distribution_handler.go
├── marketing/
│   └── coupon_handler.go
├── admin/
│   ├── auth_handler.go       # 管理员认证
│   ├── user_handler.go       # 用户管理
│   ├── device_handler.go     # 设备管理
│   └── ...
└── health/
    └── health_handler.go     # 健康检查
```

## Handler 代码模板

### 基础结构

```go
// internal/handler/user/user_handler.go
package user

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    userService "backend/internal/service/user"
    "backend/internal/common/middleware"
    "backend/pkg/response"
)

type UserHandler struct {
    userService userService.UserService
}

func NewUserHandler(userService userService.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

// RegisterRoutes 注册路由
func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
    // 需要认证的路由
    auth := r.Group("").Use(middleware.JWTAuth())
    {
        auth.GET("/user/profile", h.GetProfile)
        auth.PUT("/user/profile", h.UpdateProfile)
        auth.GET("/user/wallet", h.GetWallet)
        auth.GET("/user/wallet/transactions", h.GetWalletTransactions)
    }
}
```

### GET 请求处理 - 获取详情

```go
// GetProfile 获取用户信息
// @Summary 获取用户信息
// @Tags User
// @Security bearerAuth
// @Produce json
// @Success 200 {object} response.Response{data=UserProfileResponse}
// @Failure 401 {object} response.Response
// @Router /user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
    userID := middleware.GetUserID(c)
    if userID == 0 {
        response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
        return
    }

    user, err := h.userService.GetProfile(c.Request.Context(), userID)
    if err != nil {
        handleServiceError(c, err)
        return
    }

    response.Success(c, toUserProfileResponse(user))
}

// UserProfileResponse 用户信息响应
type UserProfileResponse struct {
    ID           int64        `json:"id"`
    Phone        string       `json:"phone"`
    Nickname     string       `json:"nickname"`
    Avatar       *string      `json:"avatar,omitempty"`
    Gender       int8         `json:"gender"`
    MemberLevel  *MemberLevel `json:"member_level,omitempty"`
    Points       int          `json:"points"`
    IsVerified   bool         `json:"is_verified"`
    IsDistributor bool        `json:"is_distributor"`
}

type MemberLevel struct {
    ID       int64   `json:"id"`
    Name     string  `json:"name"`
    Discount float64 `json:"discount"`
}

func toUserProfileResponse(user *models.User) *UserProfileResponse {
    resp := &UserProfileResponse{
        ID:         user.ID,
        Phone:      maskPhone(user.Phone),
        Nickname:   user.Nickname,
        Avatar:     user.Avatar,
        Gender:     user.Gender,
        Points:     user.Points,
        IsVerified: user.IsVerified,
    }
    if user.MemberLevel != nil {
        resp.MemberLevel = &MemberLevel{
            ID:       user.MemberLevel.ID,
            Name:     user.MemberLevel.Name,
            Discount: user.MemberLevel.Discount,
        }
    }
    return resp
}
```

### PUT 请求处理 - 更新数据

```go
// UpdateProfile 更新用户信息
// @Summary 更新用户信息
// @Tags User
// @Security bearerAuth
// @Accept json
// @Produce json
// @Param body body UpdateProfileRequest true "更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /user/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
    userID := middleware.GetUserID(c)

    var req UpdateProfileRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
        return
    }

    // 参数验证
    if err := validate.Struct(&req); err != nil {
        response.Error(c, http.StatusBadRequest, response.CodeBadRequest, formatValidationError(err))
        return
    }

    err := h.userService.UpdateProfile(c.Request.Context(), userID, &req)
    if err != nil {
        handleServiceError(c, err)
        return
    }

    response.Success(c, nil)
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
    Nickname *string `json:"nickname" validate:"omitempty,max=50"`
    Avatar   *string `json:"avatar" validate:"omitempty,url"`
    Gender   *int8   `json:"gender" validate:"omitempty,oneof=0 1 2"`
    Birthday *string `json:"birthday" validate:"omitempty,datetime=2006-01-02"`
}
```

### GET 请求处理 - 分页列表

```go
// GetWalletTransactions 获取钱包交易记录
// @Summary 获取钱包交易记录
// @Tags User
// @Security bearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param type query string false "类型" Enums(recharge,consume,refund,withdraw,deposit,return_deposit)
// @Success 200 {object} response.Response{data=response.PaginatedData}
// @Router /user/wallet/transactions [get]
func (h *UserHandler) GetWalletTransactions(c *gin.Context) {
    userID := middleware.GetUserID(c)

    // 解析分页参数
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    txType := c.Query("type")

    // 参数校验
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 20
    }

    transactions, total, err := h.userService.GetWalletTransactions(
        c.Request.Context(),
        userID,
        txType,
        page,
        pageSize,
    )
    if err != nil {
        handleServiceError(c, err)
        return
    }

    response.SuccessPaginated(c, toTransactionResponses(transactions), total, page, pageSize)
}
```

### POST 请求处理 - 创建资源

```go
// CreateRental 创建租借订单
// @Summary 创建租借订单
// @Tags Rental
// @Security bearerAuth
// @Accept json
// @Produce json
// @Param body body CreateRentalRequest true "租借信息"
// @Success 200 {object} response.Response{data=RentalOrderResponse}
// @Failure 400 {object} response.Response
// @Router /rentals [post]
func (h *RentalHandler) CreateRental(c *gin.Context) {
    userID := middleware.GetUserID(c)

    var req CreateRentalRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
        return
    }

    // 幂等性检查
    idempotencyKey := c.GetHeader("X-Idempotency-Key")
    if idempotencyKey != "" {
        if result, found := h.cache.Get(idempotencyKey); found {
            response.Success(c, result)
            return
        }
    }

    result, err := h.rentalService.CreateRental(c.Request.Context(), userID, &req)
    if err != nil {
        handleServiceError(c, err)
        return
    }

    // 缓存幂等结果
    if idempotencyKey != "" {
        h.cache.Set(idempotencyKey, result, 24*time.Hour)
    }

    response.Success(c, result)
}

// CreateRentalRequest 创建租借请求
type CreateRentalRequest struct {
    DeviceNo      string `json:"device_no" validate:"required"`
    DurationHours int    `json:"duration_hours" validate:"required,oneof=1 2 3 6 12 24"`
    CouponID      *int64 `json:"coupon_id" validate:"omitempty,gt=0"`
}
```

### 路径参数处理

```go
// GetOrderDetail 获取订单详情
// @Summary 获取订单详情
// @Tags Order
// @Security bearerAuth
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response{data=OrderDetailResponse}
// @Failure 404 {object} response.Response
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrderDetail(c *gin.Context) {
    userID := middleware.GetUserID(c)

    // 解析路径参数
    orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid order id")
        return
    }

    order, err := h.orderService.GetByID(c.Request.Context(), orderID, userID)
    if err != nil {
        handleServiceError(c, err)
        return
    }

    response.Success(c, toOrderDetailResponse(order))
}
```

## 错误处理

```go
// internal/handler/common.go
package handler

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"
    "backend/internal/repository"
    "backend/internal/service/user"
    "backend/pkg/response"
)

func handleServiceError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, repository.ErrUserNotFound):
        response.Error(c, http.StatusNotFound, response.CodeUserNotFound, "用户不存在")
    case errors.Is(err, user.ErrUserDisabled):
        response.Error(c, http.StatusForbidden, response.CodeUserDisabled, "用户已被禁用")
    case errors.Is(err, repository.ErrDeviceNotFound):
        response.Error(c, http.StatusNotFound, response.CodeDeviceNotFound, "设备不存在")
    case errors.Is(err, repository.ErrOrderNotFound):
        response.Error(c, http.StatusNotFound, response.CodeOrderNotFound, "订单不存在")
    default:
        // 记录未知错误日志
        logger.Error("unhandled error", zap.Error(err))
        response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务器内部错误")
    }
}

func formatValidationError(err error) string {
    // 格式化验证错误为用户友好的消息
    // ...
    return err.Error()
}
```

## 路由注册

```go
// cmd/api-gateway/router.go
package main

import (
    "github.com/gin-gonic/gin"
    "backend/internal/handler/auth"
    "backend/internal/handler/user"
    "backend/internal/handler/device"
    // ...
)

func setupRouter(
    authHandler *auth.AuthHandler,
    userHandler *user.UserHandler,
    deviceHandler *device.DeviceHandler,
    // ...
) *gin.Engine {
    r := gin.New()

    // 全局中间件
    r.Use(middleware.RequestID())
    r.Use(middleware.Logger())
    r.Use(middleware.Recovery())
    r.Use(middleware.Cors())

    // API v1
    v1 := r.Group("/api/v1")
    {
        // 公开路由
        authHandler.RegisterRoutes(v1)

        // 需认证路由
        userHandler.RegisterRoutes(v1)
        deviceHandler.RegisterRoutes(v1)
        // ...
    }

    return r
}
```

## 验证规则

| 规则 | 说明 | 示例 |
|------|------|------|
| `required` | 必填 | `validate:"required"` |
| `omitempty` | 可选 | `validate:"omitempty"` |
| `max=n` | 最大长度 | `validate:"max=50"` |
| `min=n` | 最小长度 | `validate:"min=6"` |
| `oneof` | 枚举值 | `validate:"oneof=1 2 3"` |
| `email` | 邮箱格式 | `validate:"email"` |
| `url` | URL格式 | `validate:"url"` |
| `gt=n` | 大于 | `validate:"gt=0"` |
| `gte=n` | 大于等于 | `validate:"gte=1"` |
| `datetime` | 日期格式 | `validate:"datetime=2006-01-02"` |

## 参考文档

- API 规范: `specs/001-smart-locker-backend/contracts/user-api.yaml`
- 响应格式: `go-backend-dev.md` 统一响应部分
