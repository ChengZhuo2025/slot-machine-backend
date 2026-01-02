---
name: API Development
description: This skill should be used when the user asks to "create API endpoint", "add route", "implement REST API", "API documentation", "Swagger", "OpenAPI", "request validation", "response format", "pagination", "error handling", or needs guidance on RESTful API design, request/response handling, and API documentation for the smart locker backend.
version: 1.0.0
---

# API Development Skill

This skill provides guidance for developing RESTful APIs in the smart locker backend system.

## API Design Principles

| Principle | Implementation |
|-----------|---------------|
| RESTful | Resource-oriented URLs |
| Versioning | URL path: `/api/v1/` |
| Authentication | JWT Bearer token |
| Format | JSON request/response |
| Documentation | OpenAPI 3.0 (Swagger) |

## Route Structure

### URL Patterns

```
/api/v1/{resource}              # Collection
/api/v1/{resource}/{id}         # Single resource
/api/v1/{resource}/{id}/{action} # Resource action
```

### Route Examples

```go
package router

import (
    "github.com/gin-gonic/gin"
    "backend/internal/handler/user"
    "backend/internal/common/middleware"
)

func SetupRoutes(r *gin.Engine, handlers *Handlers) {
    api := r.Group("/api/v1")

    // Public routes
    auth := api.Group("/auth")
    {
        auth.POST("/sms/send", handlers.Auth.SendSmsCode)
        auth.POST("/login/sms", handlers.Auth.LoginBySms)
        auth.POST("/login/wechat", handlers.Auth.LoginByWechat)
        auth.POST("/token/refresh", handlers.Auth.RefreshToken)
    }

    // Protected routes
    protected := api.Group("")
    protected.Use(middleware.JWTAuth())
    {
        // User
        user := protected.Group("/user")
        {
            user.GET("/profile", handlers.User.GetProfile)
            user.PUT("/profile", handlers.User.UpdateProfile)
            user.GET("/wallet", handlers.User.GetWallet)
            user.GET("/wallet/transactions", handlers.User.GetWalletTransactions)
        }

        // Devices
        devices := protected.Group("/devices")
        {
            devices.GET("/:device_no", handlers.Device.GetByNo)
            devices.GET("/:device_no/pricing", handlers.Device.GetPricing)
        }

        // Rentals
        rentals := protected.Group("/rentals")
        {
            rentals.POST("", handlers.Rental.Create)
            rentals.GET("", handlers.Rental.List)
            rentals.GET("/:id", handlers.Rental.GetByID)
            rentals.POST("/:id/return", handlers.Rental.Return)
        }

        // Orders
        orders := protected.Group("/orders")
        {
            orders.GET("", handlers.Order.List)
            orders.GET("/:id", handlers.Order.GetByID)
            orders.POST("/:id/cancel", handlers.Order.Cancel)
        }

        // Payments
        payments := protected.Group("/payments")
        {
            payments.POST("/create", handlers.Payment.Create)
        }
    }

    // Payment callbacks (no auth)
    callbacks := api.Group("/payments/callback")
    {
        callbacks.POST("/wechat", handlers.Payment.WechatCallback)
        callbacks.POST("/alipay", handlers.Payment.AlipayCallback)
    }
}
```

## Request Handling

### Request Binding

```go
type CreateRentalRequest struct {
    DeviceNo      string `json:"device_no" binding:"required"`
    DurationHours int    `json:"duration_hours" binding:"required,oneof=1 2 3 6 12 24"`
    CouponID      *int64 `json:"coupon_id"`
}

func (h *RentalHandler) Create(c *gin.Context) {
    var req CreateRentalRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ValidationError(c, err)
        return
    }

    userID := c.GetInt64("user_id")

    result, err := h.svc.CreateRental(c.Request.Context(), userID, &req)
    if err != nil {
        response.Error(c, err)
        return
    }

    response.Success(c, result)
}
```

### Custom Validators

```go
package validator

import (
    "github.com/go-playground/validator/v10"
    "regexp"
)

func RegisterCustomValidators(v *validator.Validate) {
    v.RegisterValidation("phone", validatePhone)
    v.RegisterValidation("order_no", validateOrderNo)
}

func validatePhone(fl validator.FieldLevel) bool {
    phone := fl.Field().String()
    matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, phone)
    return matched
}

func validateOrderNo(fl validator.FieldLevel) bool {
    orderNo := fl.Field().String()
    matched, _ := regexp.MatchString(`^[A-Z]{2}\d{14}$`, orderNo)
    return matched
}
```

### Path Parameters

```go
func (h *OrderHandler) GetByID(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        response.BadRequest(c, "invalid order id")
        return
    }

    userID := c.GetInt64("user_id")

    order, err := h.svc.GetByID(c.Request.Context(), userID, id)
    if err != nil {
        response.Error(c, err)
        return
    }

    response.Success(c, order)
}
```

### Query Parameters

```go
type ListOrdersQuery struct {
    Page     int    `form:"page" binding:"min=1"`
    PageSize int    `form:"page_size" binding:"min=1,max=100"`
    Type     string `form:"type" binding:"omitempty,oneof=rental hotel mall"`
    Status   string `form:"status"`
}

func (h *OrderHandler) List(c *gin.Context) {
    var query ListOrdersQuery
    query.Page = 1
    query.PageSize = 20

    if err := c.ShouldBindQuery(&query); err != nil {
        response.ValidationError(c, err)
        return
    }

    userID := c.GetInt64("user_id")

    orders, total, err := h.svc.List(c.Request.Context(), userID, &query)
    if err != nil {
        response.Error(c, err)
        return
    }

    response.Paginated(c, orders, total, query.Page, query.PageSize)
}
```

## Response Format

### Unified Response Structure

```go
package response

import (
    "github.com/gin-gonic/gin"
    "net/http"
)

type Response struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data,omitempty"`
    RequestID string      `json:"request_id,omitempty"`
}

type PaginatedResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
    Meta    *Pagination `json:"meta,omitempty"`
}

type Pagination struct {
    Total    int64 `json:"total"`
    Page     int   `json:"page"`
    PageSize int   `json:"page_size"`
    Pages    int   `json:"pages"`
}

func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:      0,
        Message:   "success",
        Data:      data,
        RequestID: c.GetString("request_id"),
    })
}

func Paginated(c *gin.Context, data interface{}, total int64, page, pageSize int) {
    pages := int(total) / pageSize
    if int(total)%pageSize > 0 {
        pages++
    }

    c.JSON(http.StatusOK, PaginatedResponse{
        Code:    0,
        Message: "success",
        Data:    data,
        Meta: &Pagination{
            Total:    total,
            Page:     page,
            PageSize: pageSize,
            Pages:    pages,
        },
    })
}

func Error(c *gin.Context, err error) {
    code, message := mapError(err)
    c.JSON(http.StatusOK, Response{
        Code:      code,
        Message:   message,
        RequestID: c.GetString("request_id"),
    })
}

func ValidationError(c *gin.Context, err error) {
    c.JSON(http.StatusOK, Response{
        Code:      400,
        Message:   formatValidationError(err),
        RequestID: c.GetString("request_id"),
    })
}
```

### Error Codes

```go
package errors

const (
    // General errors (1-999)
    CodeSuccess       = 0
    CodeBadRequest    = 400
    CodeUnauthorized  = 401
    CodeForbidden     = 403
    CodeNotFound      = 404
    CodeServerError   = 500

    // User errors (1000-1999)
    CodeUserNotFound     = 1001
    CodePhoneExists      = 1002
    CodeInvalidCode      = 1003
    CodeAccountDisabled  = 1004

    // Device errors (2000-2999)
    CodeDeviceNotFound   = 2001
    CodeDeviceOffline    = 2002
    CodeDeviceBusy       = 2003
    CodeUnlockFailed     = 2004

    // Order errors (3000-3999)
    CodeOrderNotFound    = 3001
    CodeOrderPaid        = 3002
    CodeOrderCancelled   = 3003
    CodeInsufficientStock = 3004

    // Payment errors (4000-4999)
    CodePaymentFailed    = 4001
    CodeRefundFailed     = 4002
    CodeInsufficientBalance = 4003
)
```

## Idempotency

### Idempotency Key Header

```go
func Idempotency() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "POST" {
            key := c.GetHeader("X-Idempotency-Key")
            if key != "" {
                c.Set("idempotency_key", key)
            }
        }
        c.Next()
    }
}
```

### Usage in Handler

```go
func (h *RentalHandler) Create(c *gin.Context) {
    idempotencyKey := c.GetString("idempotency_key")

    // Check for existing result
    if idempotencyKey != "" {
        cached, err := h.cache.Get(c.Request.Context(), "idempotency:"+idempotencyKey)
        if err == nil {
            c.Data(http.StatusOK, "application/json", cached)
            return
        }
    }

    // Process request...
    result, err := h.svc.CreateRental(...)

    // Cache result
    if idempotencyKey != "" && err == nil {
        data, _ := json.Marshal(result)
        h.cache.Set(c.Request.Context(), "idempotency:"+idempotencyKey, data, 24*time.Hour)
    }
}
```

## OpenAPI Documentation

### Swagger Setup

```go
package main

import (
    "github.com/gin-gonic/gin"
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "backend/docs" // Generated docs
)

func setupSwagger(r *gin.Engine) {
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

### Handler Annotations

```go
// CreateRental godoc
// @Summary Create rental order
// @Description Create a new rental order for a device
// @Tags Rental
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param X-Idempotency-Key header string false "Idempotency key"
// @Param request body CreateRentalRequest true "Rental request"
// @Success 200 {object} Response{data=RentalOrderResponse}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Router /rentals [post]
func (h *RentalHandler) Create(c *gin.Context) {
    // Implementation
}
```

### Generate Documentation

```bash
# Install swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs
swag init -g cmd/api-gateway/main.go -o docs

# Or use make
make swagger
```

## Middleware Stack

```go
func SetupMiddleware(r *gin.Engine) {
    r.Use(gin.Recovery())
    r.Use(middleware.RequestID())
    r.Use(middleware.RequestLogger())
    r.Use(middleware.CORS())
    r.Use(middleware.RateLimiter())
    r.Use(middleware.Idempotency())
}
```

## API Testing

### Using httptest

```go
func TestCreateRental(t *testing.T) {
    router := setupTestRouter()

    body := `{"device_no": "D001", "duration_hours": 2}`
    req, _ := http.NewRequest("POST", "/api/v1/rentals", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+testToken)
    req.Header.Set("X-Idempotency-Key", "test-key-123")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)

    var resp Response
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, 0, resp.Code)
}
```

## Additional Resources

### Reference Files

- **`references/api-conventions.md`** - API design conventions
- **`references/error-codes.md`** - Complete error code reference

### Project Documentation

- **`specs/001-smart-locker-backend/contracts/user-api.yaml`** - User API specification
- **`specs/001-smart-locker-backend/contracts/admin-api.yaml`** - Admin API specification
