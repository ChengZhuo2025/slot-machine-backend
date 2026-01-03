---
name: Go Backend Development
description: This skill should be used when the user asks to "create a service", "add an API endpoint", "implement business logic", "create a handler", "add a repository", "write Go code", "implement a feature", or needs guidance on Go backend development patterns, Gin framework usage, GORM operations, or project structure conventions for the smart locker backend system.
version: 1.0.0
---

# Go Backend Development Skill

This skill provides guidance for developing the 爱上杜美人智能开锁管理系统 backend using Go, Gin, and GORM.

## Technology Stack

| Component | Version | Purpose |
|-----------|---------|---------|
| Go | 1.25+ | Programming language |
| Gin | v1.10+ | Web framework |
| GORM | v1.25+ | ORM for PostgreSQL |
| Viper | - | Configuration management |
| JWT | golang-jwt/jwt/v5 | Authentication |

## Project Structure

```
backend/
├── cmd/api-gateway/         # Main entry point
├── internal/
│   ├── common/              # Shared components
│   │   ├── config/          # Configuration
│   │   ├── database/        # Database connection
│   │   ├── cache/           # Redis cache
│   │   ├── middleware/      # HTTP middleware
│   │   └── utils/           # Utilities
│   ├── models/              # Data models (GORM)
│   ├── repository/          # Data access layer
│   ├── service/             # Business logic
│   └── handler/             # HTTP handlers
├── pkg/                     # Reusable packages
│   ├── auth/                # JWT authentication
│   ├── crypto/              # Encryption
│   ├── payment/             # Payment integration
│   ├── response/            # Unified response
│   └── qrcode/              # QR code generation
└── migrations/              # Database migrations
```

## Development Patterns

### Handler Pattern

Create HTTP handlers in `internal/handler/{module}/`:

```go
package user

import (
    "github.com/gin-gonic/gin"
    "backend/pkg/response"
    "backend/internal/service/user"
)

type Handler struct {
    svc *user.Service
}

func NewHandler(svc *user.Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) GetProfile(c *gin.Context) {
    userID := c.GetInt64("user_id")

    profile, err := h.svc.GetProfile(c.Request.Context(), userID)
    if err != nil {
        response.Error(c, err)
        return
    }

    response.Success(c, profile)
}
```

### Service Pattern

Implement business logic in `internal/service/{module}/`:

```go
package user

import (
    "context"
    "backend/internal/models"
    "backend/internal/repository"
)

type Service struct {
    repo *repository.UserRepository
}

func NewService(repo *repository.UserRepository) *Service {
    return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, userID int64) (*models.User, error) {
    return s.repo.GetByID(ctx, userID)
}
```

### Repository Pattern

Implement data access in `internal/repository/`:

```go
package repository

import (
    "context"
    "backend/internal/models"
    "gorm.io/gorm"
)

type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
    var user models.User
    if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}
```

### Model Definition

**CRITICAL**: Define GORM models in `internal/models/` following strict conventions.

**⚠️ IMPORTANT RULES**:
1. **ALL fields MUST have `column:` tag** - Never rely on GORM's automatic naming
2. **Field names must match database column names exactly**
3. **Status fields MUST use `string` type** (not int/int8) for readability
4. **Always refer to migration files** for authoritative schema definitions

```go
package models

import "time"

// User model - ALWAYS refer to migrations/000001_init_users.up.sql
type User struct {
    // Primary key
    ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

    // Business fields - ALL must have column: tag
    Phone           string     `gorm:"column:phone;type:varchar(20);uniqueIndex;not null" json:"phone"`
    OpenID          *string    `gorm:"column:openid;type:varchar(64);uniqueIndex" json:"openid,omitempty"`
    Unionid         *string    `gorm:"column:unionid;type:varchar(64)" json:"unionid,omitempty"`
    Nickname        string     `gorm:"column:nickname;type:varchar(50);not null" json:"nickname"`
    Avatar          *string    `gorm:"column:avatar;type:varchar(255)" json:"avatar,omitempty"`
    Gender          int16      `gorm:"column:gender;type:smallint;default:0" json:"gender"`
    Birthday        *time.Time `gorm:"column:birthday;type:date" json:"birthday,omitempty"`
    MemberLevelID   int64      `gorm:"column:member_level_id;default:1" json:"member_level_id"`
    Points          int        `gorm:"column:points;default:0" json:"points"`
    IsVerified      bool       `gorm:"column:is_verified;not null;default:false" json:"is_verified"`
    ReferrerID      *int64     `gorm:"column:referrer_id;index" json:"referrer_id,omitempty"`

    // Status field - Use string type for readability (NOT int8!)
    Status          string     `gorm:"column:status;type:varchar(20);not null" json:"status"`

    // Timestamps
    CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    DeletedAt       *time.Time `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

    // Relations
    Referrer *User `gorm:"foreignKey:ReferrerID" json:"referrer,omitempty"`
}

func (User) TableName() string {
    return "users"
}

// UserStatus constants - Use strings for clarity
const (
    UserStatusActive   = "active"
    UserStatusInactive = "inactive"
    UserStatusBanned   = "banned"
)
```

**Development Checklist** (MUST complete before PR):
- [ ] Checked `specs/001-smart-locker-backend/data-model.md` for table definition
- [ ] Checked corresponding migration file in `migrations/`
- [ ] All fields have `column:` tags
- [ ] Field types match database (VARCHAR→string, BIGINT→int64, BOOLEAN→bool, TIMESTAMP→time.Time)
- [ ] Status fields use string type (not int/int8)
- [ ] NOT NULL fields use value types, NULLABLE fields use pointer types
- [ ] No extra fields that don't exist in database
- [ ] No missing required fields from database
- [ ] Wrote basic CRUD unit tests

**Reference**: See `specs/001-smart-locker-backend/model-development-guide.md` for complete standards.

## Unified Response Format

Use `pkg/response` for consistent API responses:

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

func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:    0,
        Message: "success",
        Data:    data,
    })
}

func Error(c *gin.Context, err error) {
    // Map error to appropriate response
    code, message := mapError(err)
    c.JSON(http.StatusOK, Response{
        Code:    code,
        Message: message,
    })
}
```

## Middleware Usage

### Authentication Middleware

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "backend/pkg/auth"
)

func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "unauthorized"})
            return
        }

        claims, err := auth.ParseToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "invalid token"})
            return
        }

        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

### Request Logging

```go
func RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        latency := time.Since(start)

        log.Printf("[%s] %s %s %d %v",
            c.Request.Method,
            c.Request.URL.Path,
            c.ClientIP(),
            c.Writer.Status(),
            latency,
        )
    }
}
```

## Error Handling

Define custom error types:

```go
package errors

type AppError struct {
    Code    int
    Message string
    Err     error
}

func (e *AppError) Error() string {
    return e.Message
}

var (
    ErrNotFound      = &AppError{Code: 404, Message: "resource not found"}
    ErrUnauthorized  = &AppError{Code: 401, Message: "unauthorized"}
    ErrBadRequest    = &AppError{Code: 400, Message: "bad request"}
    ErrDeviceBusy    = &AppError{Code: 1001, Message: "device is in use"}
    ErrPaymentFailed = &AppError{Code: 2001, Message: "payment failed"}
)
```

## Configuration Management

Use Viper for configuration:

```go
package config

import "github.com/spf13/viper"

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    JWT      JWTConfig
    MQTT     MQTTConfig
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./configs")

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

## Code Quality Standards

### Naming Conventions

- Package names: lowercase, single word (`user`, `order`, `payment`)
- Interface names: describe behavior (`Repository`, `Service`)
- Struct names: PascalCase (`UserService`, `OrderHandler`)
- Function names: PascalCase for exported, camelCase for internal

### Code Style

- Run `gofmt -w .` before committing
- Run `golangci-lint run` for static analysis
- Maximum function length: 50 lines
- Maximum cyclomatic complexity: 10

### Documentation

- Add package-level documentation
- Document exported functions and types
- Use meaningful variable names

## Additional Resources

### Reference Files

For detailed patterns and techniques, consult:
- **`references/patterns.md`** - Advanced Go patterns
- **`references/gin-best-practices.md`** - Gin framework conventions

### Project Documentation

- **`specs/001-smart-locker-backend/spec.md`** - Feature specification
- **`specs/001-smart-locker-backend/data-model.md`** - Data model definitions
- **`specs/001-smart-locker-backend/contracts/user-api.yaml`** - API contracts
