---
name: Testing and Debugging
description: This skill should be used when the user asks to "write tests", "add unit tests", "create integration tests", "debug an issue", "fix a bug", "run tests", "test coverage", "mock dependencies", or needs guidance on testing strategies, debugging techniques, and quality assurance for the Go backend.
version: 1.0.0
---

# Testing and Debugging Skill

This skill provides guidance for testing and debugging the smart locker backend system.

## Testing Framework

| Tool | Purpose |
|------|---------|
| `testing` | Go standard testing |
| `testify` | Assertions and mocking |
| `dockertest` | Integration tests with containers |
| `testcontainers-go` | Database containers |
| `httptest` | HTTP handler testing |
| `go-resty` | API client testing |

## Test Directory Structure

```
tests/
├── unit/              # Unit tests (mirrors internal/)
├── integration/       # Integration tests
└── e2e/               # End-to-end tests

internal/
├── service/
│   └── user/
│       ├── service.go
│       └── service_test.go    # Unit tests alongside code
```

## Unit Testing

### Service Layer Testing

```go
package user_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "backend/internal/models"
    "backend/internal/service/user"
)

type MockUserRepo struct {
    mock.Mock
}

func (m *MockUserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.User), args.Error(1)
}

func TestGetProfile_Success(t *testing.T) {
    mockRepo := new(MockUserRepo)
    svc := user.NewService(mockRepo)

    expectedUser := &models.User{
        ID:       1,
        Phone:    "13800138000",
        Nickname: "Test User",
    }

    mockRepo.On("GetByID", mock.Anything, int64(1)).Return(expectedUser, nil)

    result, err := svc.GetProfile(context.Background(), 1)

    assert.NoError(t, err)
    assert.Equal(t, expectedUser.Nickname, result.Nickname)
    mockRepo.AssertExpectations(t)
}

func TestGetProfile_NotFound(t *testing.T) {
    mockRepo := new(MockUserRepo)
    svc := user.NewService(mockRepo)

    mockRepo.On("GetByID", mock.Anything, int64(999)).Return(nil, gorm.ErrRecordNotFound)

    result, err := svc.GetProfile(context.Background(), 999)

    assert.Error(t, err)
    assert.Nil(t, result)
}
```

### Handler Testing

```go
package user_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "backend/internal/handler/user"
)

func setupRouter(h *user.Handler) *gin.Engine {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.GET("/user/profile", h.GetProfile)
    return r
}

func TestGetProfile_Handler(t *testing.T) {
    mockSvc := new(MockUserService)
    handler := user.NewHandler(mockSvc)
    router := setupRouter(handler)

    mockSvc.On("GetProfile", mock.Anything, int64(1)).Return(&models.User{
        ID:       1,
        Nickname: "Test User",
    }, nil)

    req, _ := http.NewRequest("GET", "/user/profile", nil)
    req.Header.Set("Authorization", "Bearer valid_token")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "Test User")
}
```

### Table-Driven Tests

```go
func TestCalculateRentalFee(t *testing.T) {
    tests := []struct {
        name          string
        durationHours int
        expectedFee   decimal.Decimal
        expectError   bool
    }{
        {"1 hour", 1, decimal.NewFromFloat(10.00), false},
        {"2 hours", 2, decimal.NewFromFloat(18.00), false},
        {"6 hours", 6, decimal.NewFromFloat(50.00), false},
        {"invalid duration", 5, decimal.Zero, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            fee, err := rental.CalculateFee(tt.durationHours)

            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.True(t, tt.expectedFee.Equal(fee))
            }
        })
    }
}
```

## Integration Testing

### Database Integration Tests

```go
package integration

import (
    "context"
    "testing"

    "github.com/ory/dockertest/v3"
    "github.com/stretchr/testify/suite"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type DatabaseTestSuite struct {
    suite.Suite
    pool     *dockertest.Pool
    resource *dockertest.Resource
    db       *gorm.DB
}

func (s *DatabaseTestSuite) SetupSuite() {
    pool, err := dockertest.NewPool("")
    s.Require().NoError(err)

    resource, err := pool.Run("postgres", "15", []string{
        "POSTGRES_PASSWORD=test",
        "POSTGRES_DB=test",
    })
    s.Require().NoError(err)

    s.pool = pool
    s.resource = resource

    // Wait for container to be ready
    err = pool.Retry(func() error {
        dsn := fmt.Sprintf("host=localhost port=%s user=postgres password=test dbname=test sslmode=disable",
            resource.GetPort("5432/tcp"))
        db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
        if err != nil {
            return err
        }
        s.db = db
        return nil
    })
    s.Require().NoError(err)

    // Run migrations
    s.db.AutoMigrate(&models.User{}, &models.Order{})
}

func (s *DatabaseTestSuite) TearDownSuite() {
    s.pool.Purge(s.resource)
}

func (s *DatabaseTestSuite) TestCreateUser() {
    repo := repository.NewUserRepository(s.db)

    user := &models.User{
        Phone:    "13800138000",
        Nickname: "Test User",
    }

    err := repo.Create(context.Background(), user)
    s.NoError(err)
    s.NotZero(user.ID)
}

func TestDatabaseSuite(t *testing.T) {
    suite.Run(t, new(DatabaseTestSuite))
}
```

### API Integration Tests

```go
package integration

import (
    "testing"

    "github.com/go-resty/resty/v2"
    "github.com/stretchr/testify/assert"
)

func TestRentalFlow(t *testing.T) {
    client := resty.New()
    baseURL := "http://localhost:8080/api/v1"

    // Step 1: Login
    var loginResp struct {
        Code int `json:"code"`
        Data struct {
            AccessToken string `json:"access_token"`
        } `json:"data"`
    }

    resp, err := client.R().
        SetHeader("Content-Type", "application/json").
        SetBody(`{"phone": "13800138000", "code": "123456"}`).
        SetResult(&loginResp).
        Post(baseURL + "/auth/login/sms")

    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode())
    token := loginResp.Data.AccessToken

    // Step 2: Get device info
    var deviceResp struct {
        Code int           `json:"code"`
        Data models.Device `json:"data"`
    }

    resp, err = client.R().
        SetHeader("Authorization", "Bearer "+token).
        SetResult(&deviceResp).
        Get(baseURL + "/devices/D001")

    assert.NoError(t, err)
    assert.Equal(t, 0, deviceResp.Code)

    // Step 3: Create rental order
    var orderResp struct {
        Code int `json:"code"`
        Data struct {
            OrderID int64 `json:"order_id"`
        } `json:"data"`
    }

    resp, err = client.R().
        SetHeader("Authorization", "Bearer "+token).
        SetHeader("Content-Type", "application/json").
        SetBody(`{"device_no": "D001", "duration_hours": 2}`).
        SetResult(&orderResp).
        Post(baseURL + "/rentals")

    assert.NoError(t, err)
    assert.Equal(t, 0, orderResp.Code)
    assert.NotZero(t, orderResp.Data.OrderID)
}
```

## Debugging Techniques

### Structured Logging

```go
package logger

import (
    "go.uber.org/zap"
)

var Log *zap.SugaredLogger

func Init(env string) {
    var logger *zap.Logger
    if env == "production" {
        logger, _ = zap.NewProduction()
    } else {
        logger, _ = zap.NewDevelopment()
    }
    Log = logger.Sugar()
}

// Usage
logger.Log.With(
    "user_id", userID,
    "device_no", deviceNo,
    "action", "create_rental",
).Info("Creating rental order")

logger.Log.With(
    "error", err,
    "order_id", orderID,
).Error("Failed to process payment")
```

### Request Tracing

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)
        c.Next()
    }
}
```

### Panic Recovery

```go
package middleware

func Recovery() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if r := recover(); r != nil {
                requestID := c.GetString("request_id")
                logger.Log.With(
                    "request_id", requestID,
                    "panic", r,
                    "stack", string(debug.Stack()),
                ).Error("Panic recovered")

                c.AbortWithStatusJSON(500, gin.H{
                    "code":       500,
                    "message":    "internal server error",
                    "request_id": requestID,
                })
            }
        }()
        c.Next()
    }
}
```

## Test Commands

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/service/user/...

# Run with verbose output
go test -v ./...

# Run integration tests
go test -tags=integration ./tests/integration/...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Coverage Requirements

| Category | Target |
|----------|--------|
| Unit tests | > 80% |
| Critical business logic | > 90% |
| Integration tests | Key flows |
| E2E tests | Happy paths |

## Additional Resources

### Reference Files

- **`references/mocking-guide.md`** - Advanced mocking techniques
- **`references/debugging-tools.md`** - Debugging tools and commands

### Test Data

- Use `specs/001-smart-locker-backend/` mock data for test fixtures
- Seed scripts in `migrations/` for test database setup
