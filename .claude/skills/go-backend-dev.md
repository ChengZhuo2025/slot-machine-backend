# Go 后端开发技能

## 概述

本技能用于生成符合项目规范的 Go 后端代码，包括 Model、Repository、Service、中间件和公共组件。

## 技术栈

- **语言**: Go 1.25+
- **Web 框架**: Gin v1.10+
- **ORM**: GORM v1.25+
- **配置管理**: Viper
- **日志**: Zap / Zerolog
- **验证**: go-playground/validator

## 项目结构

```
backend/
├── cmd/api-gateway/        # 应用入口
├── internal/               # 内部实现（不对外暴露）
│   ├── common/             # 公共组件
│   │   ├── config/         # 配置管理
│   │   ├── database/       # 数据库连接
│   │   ├── cache/          # Redis 缓存
│   │   ├── mq/             # 消息队列
│   │   ├── mqtt/           # MQTT 客户端
│   │   ├── logger/         # 日志组件
│   │   ├── middleware/     # 中间件
│   │   └── utils/          # 工具函数
│   ├── models/             # 数据模型
│   ├── repository/         # 数据访问层
│   ├── service/            # 业务逻辑层
│   │   ├── auth/
│   │   ├── user/
│   │   ├── device/
│   │   ├── order/
│   │   └── ...
│   └── handler/            # HTTP 处理器
└── pkg/                    # 可复用公共包
    ├── auth/               # JWT 认证
    ├── crypto/             # 加密解密
    ├── payment/            # 支付集成
    ├── response/           # 统一响应
    └── ...
```

## 代码规范

### 1. Model 定义规范

```go
// internal/models/user.go
package models

import (
    "time"
    "gorm.io/gorm"
)

// User 用户模型
type User struct {
    ID                  int64          `gorm:"primaryKey;autoIncrement" json:"id"`
    Phone               string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"phone"`
    OpenID              *string        `gorm:"type:varchar(64);uniqueIndex" json:"openid,omitempty"`
    UnionID             *string        `gorm:"type:varchar(64)" json:"unionid,omitempty"`
    Nickname            string         `gorm:"type:varchar(50);not null" json:"nickname"`
    Avatar              *string        `gorm:"type:varchar(255)" json:"avatar,omitempty"`
    Gender              int8           `gorm:"default:0" json:"gender"` // 0未知 1男 2女
    Birthday            *time.Time     `gorm:"type:date" json:"birthday,omitempty"`
    MemberLevelID       int64          `gorm:"default:1" json:"member_level_id"`
    Points              int            `gorm:"default:0" json:"points"`
    IsVerified          bool           `gorm:"default:false" json:"is_verified"`
    RealNameEncrypted   *string        `gorm:"type:text" json:"-"`
    IDCardEncrypted     *string        `gorm:"type:text" json:"-"`
    ReferrerID          *int64         `gorm:"index" json:"referrer_id,omitempty"`
    Status              int8           `gorm:"default:1" json:"status"` // 0禁用 1正常
    CreatedAt           time.Time      `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt           time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`

    // 关联
    MemberLevel *MemberLevel `gorm:"foreignKey:MemberLevelID" json:"member_level,omitempty"`
    Referrer    *User        `gorm:"foreignKey:ReferrerID" json:"referrer,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
    return "users"
}

// 状态常量
const (
    UserStatusDisabled = 0
    UserStatusActive   = 1
)
```

### 2. Repository 规范

```go
// internal/repository/user_repo.go
package repository

import (
    "context"
    "errors"

    "gorm.io/gorm"
    "backend/internal/models"
)

var (
    ErrUserNotFound = errors.New("user not found")
)

type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    GetByID(ctx context.Context, id int64) (*models.User, error)
    GetByPhone(ctx context.Context, phone string) (*models.User, error)
    GetByOpenID(ctx context.Context, openID string) (*models.User, error)
    Update(ctx context.Context, user *models.User) error
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context, opts ListOptions) ([]*models.User, int64, error)
}

type userRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
    var user models.User
    err := r.db.WithContext(ctx).
        Preload("MemberLevel").
        First(&user, id).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrUserNotFound
    }
    return &user, err
}

func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
    var user models.User
    err := r.db.WithContext(ctx).
        Where("phone = ?", phone).
        First(&user).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrUserNotFound
    }
    return &user, err
}

// ... 其他方法
```

### 3. Service 规范

```go
// internal/service/user/user_service.go
package user

import (
    "context"
    "errors"

    "backend/internal/models"
    "backend/internal/repository"
)

var (
    ErrUserDisabled = errors.New("user is disabled")
    ErrPhoneExists  = errors.New("phone already exists")
)

type UserService interface {
    GetProfile(ctx context.Context, userID int64) (*models.User, error)
    UpdateProfile(ctx context.Context, userID int64, req *UpdateProfileRequest) error
    GetByPhone(ctx context.Context, phone string) (*models.User, error)
}

type userService struct {
    userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
    return &userService{userRepo: userRepo}
}

func (s *userService) GetProfile(ctx context.Context, userID int64) (*models.User, error) {
    user, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    if user.Status == models.UserStatusDisabled {
        return nil, ErrUserDisabled
    }
    return user, nil
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
    Nickname *string `json:"nickname" validate:"omitempty,max=50"`
    Avatar   *string `json:"avatar" validate:"omitempty,url"`
    Gender   *int8   `json:"gender" validate:"omitempty,oneof=0 1 2"`
    Birthday *string `json:"birthday" validate:"omitempty,datetime=2006-01-02"`
}

func (s *userService) UpdateProfile(ctx context.Context, userID int64, req *UpdateProfileRequest) error {
    user, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return err
    }

    // 更新字段
    if req.Nickname != nil {
        user.Nickname = *req.Nickname
    }
    if req.Avatar != nil {
        user.Avatar = req.Avatar
    }
    // ... 其他字段

    return s.userRepo.Update(ctx, user)
}
```

### 4. 错误处理规范

```go
// pkg/response/errors.go
package response

// 错误码定义
const (
    CodeSuccess       = 0
    CodeBadRequest    = 400
    CodeUnauthorized  = 401
    CodeForbidden     = 403
    CodeNotFound      = 404
    CodeConflict      = 409
    CodeTooManyReqs   = 429
    CodeInternalError = 500
)

// 业务错误码 (10000+)
const (
    CodeUserNotFound     = 10001
    CodeUserDisabled     = 10002
    CodePhoneExists      = 10003
    CodeInvalidCode      = 10004
    CodeCodeExpired      = 10005
    CodeDeviceOffline    = 10101
    CodeDeviceInUse      = 10102
    CodeOrderNotFound    = 10201
    CodePaymentFailed    = 10301
    // ...
)

// 错误消息映射
var codeMessages = map[int]string{
    CodeSuccess:       "success",
    CodeBadRequest:    "bad request",
    CodeUnauthorized:  "unauthorized",
    CodeUserNotFound:  "用户不存在",
    CodeUserDisabled:  "用户已被禁用",
    CodeDeviceOffline: "设备离线",
    // ...
}

type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

func (e *AppError) Error() string {
    return e.Message
}

func NewError(code int) *AppError {
    return &AppError{
        Code:    code,
        Message: codeMessages[code],
    }
}
```

### 5. 统一响应格式

```go
// pkg/response/response.go
package response

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

type Response struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data,omitempty"`
    RequestID string      `json:"request_id,omitempty"`
}

type PaginatedData struct {
    Total    int64       `json:"total"`
    Page     int         `json:"page"`
    PageSize int         `json:"page_size"`
    Items    interface{} `json:"items"`
}

func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:      CodeSuccess,
        Message:   "success",
        Data:      data,
        RequestID: c.GetString("request_id"),
    })
}

func SuccessPaginated(c *gin.Context, items interface{}, total int64, page, pageSize int) {
    Success(c, PaginatedData{
        Total:    total,
        Page:     page,
        PageSize: pageSize,
        Items:    items,
    })
}

func Error(c *gin.Context, httpCode int, appCode int, message string) {
    c.JSON(httpCode, Response{
        Code:      appCode,
        Message:   message,
        RequestID: c.GetString("request_id"),
    })
}

func ErrorWithCode(c *gin.Context, err *AppError) {
    httpCode := http.StatusBadRequest
    if err.Code == CodeUnauthorized {
        httpCode = http.StatusUnauthorized
    } else if err.Code == CodeNotFound {
        httpCode = http.StatusNotFound
    } else if err.Code >= CodeInternalError {
        httpCode = http.StatusInternalServerError
    }
    Error(c, httpCode, err.Code, err.Message)
}
```

### 6. 中间件规范

```go
// internal/common/middleware/auth.go
package middleware

import (
    "strings"

    "github.com/gin-gonic/gin"
    "backend/pkg/auth"
    "backend/pkg/response"
)

func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            response.Error(c, 401, response.CodeUnauthorized, "missing authorization header")
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            response.Error(c, 401, response.CodeUnauthorized, "invalid authorization format")
            c.Abort()
            return
        }

        claims, err := auth.ParseToken(parts[1])
        if err != nil {
            response.Error(c, 401, response.CodeUnauthorized, "invalid token")
            c.Abort()
            return
        }

        // 将用户信息存入上下文
        c.Set("user_id", claims.UserID)
        c.Set("user_type", claims.UserType)
        c.Next()
    }
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) int64 {
    if id, exists := c.Get("user_id"); exists {
        return id.(int64)
    }
    return 0
}
```

## 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 文件名 | 小写下划线 | `user_service.go` |
| 包名 | 小写单词 | `user`, `auth` |
| 接口 | 大驼峰 + 动词/名词 | `UserService`, `Repository` |
| 结构体 | 大驼峰 | `User`, `OrderItem` |
| 方法 | 大驼峰 | `GetByID`, `CreateOrder` |
| 常量 | 大驼峰或全大写下划线 | `UserStatusActive`, `CODE_SUCCESS` |
| 变量 | 小驼峰 | `userRepo`, `orderService` |

## 依赖注入

推荐使用构造函数注入：

```go
// 依赖注入示例
func NewUserHandler(userService user.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

// 在 main.go 或 wire 中组装
db := database.NewPostgres(cfg)
userRepo := repository.NewUserRepository(db)
userService := user.NewUserService(userRepo)
userHandler := handler.NewUserHandler(userService)
```

## 参考文档

- 数据模型: `specs/001-smart-locker-backend/data-model.md`
- API 规范: `specs/001-smart-locker-backend/contracts/user-api.yaml`
- 任务清单: `specs/001-smart-locker-backend/tasks.md`
