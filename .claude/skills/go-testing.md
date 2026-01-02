# Go 测试开发技能

## 概述

本技能用于生成符合项目规范的 Go 测试代码，包括单元测试、集成测试和端到端测试。

## 技术栈

- **测试框架**: Go testing + testify v1.8+
- **Mock 生成**: mockery v2.40+
- **集成测试**: testcontainers-go v0.27+
- **HTTP 测试**: httptest + go-resty
- **覆盖率工具**: go test -cover

## 测试目录结构

```
backend/
├── internal/
│   ├── service/
│   │   └── user/
│   │       ├── user_service.go
│   │       └── user_service_test.go    # 单元测试（同目录）
│   └── repository/
│       ├── user_repo.go
│       └── user_repo_test.go
├── tests/
│   ├── integration/                     # 集成测试
│   │   ├── setup_test.go               # 测试环境初始化
│   │   ├── user_test.go
│   │   ├── device_test.go
│   │   └── order_test.go
│   └── e2e/                            # 端到端测试
│       ├── rental_flow_test.go
│       └── booking_flow_test.go
└── mocks/                               # Mock 文件
    ├── user_repository.go
    ├── device_repository.go
    └── ...
```

## 单元测试模板

### Service 层测试

```go
// internal/service/user/user_service_test.go
package user

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    "backend/internal/models"
    "backend/mocks"
)

func TestUserService_GetProfile(t *testing.T) {
    // 测试用例表驱动
    tests := []struct {
        name      string
        userID    int64
        mockSetup func(*mocks.UserRepository)
        want      *models.User
        wantErr   error
    }{
        {
            name:   "成功获取用户信息",
            userID: 1,
            mockSetup: func(m *mocks.UserRepository) {
                m.On("GetByID", mock.Anything, int64(1)).Return(&models.User{
                    ID:       1,
                    Phone:    "13800138000",
                    Nickname: "测试用户",
                    Status:   models.UserStatusActive,
                }, nil)
            },
            want: &models.User{
                ID:       1,
                Phone:    "13800138000",
                Nickname: "测试用户",
                Status:   models.UserStatusActive,
            },
            wantErr: nil,
        },
        {
            name:   "用户不存在",
            userID: 999,
            mockSetup: func(m *mocks.UserRepository) {
                m.On("GetByID", mock.Anything, int64(999)).Return(nil, repository.ErrUserNotFound)
            },
            want:    nil,
            wantErr: repository.ErrUserNotFound,
        },
        {
            name:   "用户已禁用",
            userID: 2,
            mockSetup: func(m *mocks.UserRepository) {
                m.On("GetByID", mock.Anything, int64(2)).Return(&models.User{
                    ID:     2,
                    Status: models.UserStatusDisabled,
                }, nil)
            },
            want:    nil,
            wantErr: ErrUserDisabled,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            mockRepo := new(mocks.UserRepository)
            tt.mockSetup(mockRepo)
            svc := NewUserService(mockRepo)

            // Act
            got, err := svc.GetProfile(context.Background(), tt.userID)

            // Assert
            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                assert.Nil(t, got)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.want.ID, got.ID)
                assert.Equal(t, tt.want.Nickname, got.Nickname)
            }
            mockRepo.AssertExpectations(t)
        })
    }
}

func TestUserService_UpdateProfile(t *testing.T) {
    t.Run("更新昵称成功", func(t *testing.T) {
        mockRepo := new(mocks.UserRepository)
        mockRepo.On("GetByID", mock.Anything, int64(1)).Return(&models.User{
            ID:       1,
            Nickname: "旧昵称",
        }, nil)
        mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

        svc := NewUserService(mockRepo)
        nickname := "新昵称"
        err := svc.UpdateProfile(context.Background(), 1, &UpdateProfileRequest{
            Nickname: &nickname,
        })

        assert.NoError(t, err)
        mockRepo.AssertExpectations(t)
    })
}
```

### Repository 层测试

```go
// internal/repository/user_repo_test.go
package repository

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "backend/internal/models"
)

// 使用 SQLite 内存数据库进行快速测试
func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)

    err = db.AutoMigrate(&models.User{})
    require.NoError(t, err)

    return db
}

func TestUserRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := NewUserRepository(db)

    user := &models.User{
        Phone:    "13800138000",
        Nickname: "测试用户",
    }

    err := repo.Create(context.Background(), user)

    assert.NoError(t, err)
    assert.NotZero(t, user.ID)
}

func TestUserRepository_GetByPhone(t *testing.T) {
    db := setupTestDB(t)
    repo := NewUserRepository(db)

    // 准备测试数据
    db.Create(&models.User{Phone: "13800138000", Nickname: "测试"})

    t.Run("找到用户", func(t *testing.T) {
        user, err := repo.GetByPhone(context.Background(), "13800138000")
        assert.NoError(t, err)
        assert.Equal(t, "13800138000", user.Phone)
    })

    t.Run("用户不存在", func(t *testing.T) {
        user, err := repo.GetByPhone(context.Background(), "13900139000")
        assert.ErrorIs(t, err, ErrUserNotFound)
        assert.Nil(t, user)
    })
}
```

## 集成测试模板

### 测试环境初始化

```go
// tests/integration/setup_test.go
package integration

import (
    "context"
    "fmt"
    "os"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

var (
    testDB    *gorm.DB
    testRedis *redis.RedisContainer
    pgContainer *postgres.PostgresContainer
)

func TestMain(m *testing.M) {
    ctx := context.Background()

    // 启动 PostgreSQL 容器
    var err error
    pgContainer, err = postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    if err != nil {
        panic(err)
    }

    // 获取连接字符串
    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        panic(err)
    }

    // 连接数据库
    testDB, err = gorm.Open(postgres.Open(connStr), &gorm.Config{})
    if err != nil {
        panic(err)
    }

    // 运行迁移
    runMigrations(testDB)

    // 启动 Redis 容器
    testRedis, err = redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    if err != nil {
        panic(err)
    }

    // 运行测试
    code := m.Run()

    // 清理
    pgContainer.Terminate(ctx)
    testRedis.Terminate(ctx)

    os.Exit(code)
}

func runMigrations(db *gorm.DB) {
    // 执行数据库迁移
    db.AutoMigrate(
        &models.User{},
        &models.Device{},
        &models.Order{},
        // ... 其他模型
    )
}

// 测试辅助函数
func cleanupTable(t *testing.T, tables ...string) {
    for _, table := range tables {
        testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
    }
}

func createTestUser(t *testing.T) *models.User {
    user := &models.User{
        Phone:    fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000),
        Nickname: "测试用户",
        Status:   models.UserStatusActive,
    }
    err := testDB.Create(user).Error
    require.NoError(t, err)
    return user
}
```

### API 集成测试

```go
// tests/integration/user_test.go
package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserAPI_GetProfile(t *testing.T) {
    // 清理并准备数据
    cleanupTable(t, "users")
    user := createTestUser(t)

    // 初始化路由
    gin.SetMode(gin.TestMode)
    router := setupTestRouter()

    // 生成测试 Token
    token := generateTestToken(user.ID)

    t.Run("获取用户信息成功", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
        req.Header.Set("Authorization", "Bearer "+token)
        w := httptest.NewRecorder()

        router.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)

        var resp map[string]interface{}
        err := json.Unmarshal(w.Body.Bytes(), &resp)
        require.NoError(t, err)
        assert.Equal(t, float64(0), resp["code"])
    })

    t.Run("未授权访问", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
        w := httptest.NewRecorder()

        router.ServeHTTP(w, req)

        assert.Equal(t, http.StatusUnauthorized, w.Code)
    })
}

func TestUserAPI_UpdateProfile(t *testing.T) {
    cleanupTable(t, "users")
    user := createTestUser(t)
    token := generateTestToken(user.ID)

    gin.SetMode(gin.TestMode)
    router := setupTestRouter()

    t.Run("更新昵称成功", func(t *testing.T) {
        body := map[string]interface{}{
            "nickname": "新昵称",
        }
        jsonBody, _ := json.Marshal(body)

        req := httptest.NewRequest(http.MethodPut, "/api/v1/user/profile", bytes.NewReader(jsonBody))
        req.Header.Set("Authorization", "Bearer "+token)
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        router.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)

        // 验证数据库已更新
        var updated models.User
        testDB.First(&updated, user.ID)
        assert.Equal(t, "新昵称", updated.Nickname)
    })

    t.Run("昵称过长", func(t *testing.T) {
        body := map[string]interface{}{
            "nickname": string(make([]byte, 100)), // 超过50字符限制
        }
        jsonBody, _ := json.Marshal(body)

        req := httptest.NewRequest(http.MethodPut, "/api/v1/user/profile", bytes.NewReader(jsonBody))
        req.Header.Set("Authorization", "Bearer "+token)
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        router.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
    })
}
```

## Mock 生成

### mockery 配置

```yaml
# .mockery.yaml
with-expecter: true
packages:
  backend/internal/repository:
    interfaces:
      UserRepository:
      DeviceRepository:
      OrderRepository:
      RentalRepository:
  backend/internal/service/user:
    interfaces:
      UserService:
  backend/internal/service/device:
    interfaces:
      DeviceService:
  backend/pkg/payment:
    interfaces:
      PaymentClient:
```

### 生成 Mock 命令

```bash
# 安装 mockery
go install github.com/vektra/mockery/v2@latest

# 生成所有 Mock
mockery --all --output=mocks --outpkg=mocks

# 生成特定接口的 Mock
mockery --name=UserRepository --dir=internal/repository --output=mocks
```

### Mock 使用示例

```go
// 使用 mockery 生成的 Mock
func TestOrderService_CreateOrder(t *testing.T) {
    mockUserRepo := new(mocks.UserRepository)
    mockOrderRepo := new(mocks.OrderRepository)
    mockPayment := new(mocks.PaymentClient)

    // 设置期望
    mockUserRepo.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.User{
        ID:     1,
        Status: models.UserStatusActive,
    }, nil)

    mockOrderRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("*models.Order")).
        Run(func(ctx context.Context, order *models.Order) {
            order.ID = 100 // 模拟生成的订单ID
        }).
        Return(nil)

    // 创建 Service 并测试
    svc := NewOrderService(mockUserRepo, mockOrderRepo, mockPayment)
    // ...
}
```

## 测试覆盖率

### 运行测试并生成覆盖率报告

```bash
# 运行单元测试并生成覆盖率
go test -v -coverprofile=coverage.out ./internal/...

# 查看覆盖率摘要
go tool cover -func=coverage.out

# 生成 HTML 覆盖率报告
go tool cover -html=coverage.out -o coverage.html

# 运行集成测试
go test -v -tags=integration ./tests/integration/...

# 合并覆盖率报告
go test -v -coverprofile=coverage.out ./...
```

### 覆盖率目标

| 模块 | 最低覆盖率 | 目标覆盖率 |
|------|-----------|-----------|
| Service 层 | 80% | 90% |
| Repository 层 | 70% | 80% |
| Handler 层 | 60% | 75% |
| 工具函数 | 90% | 95% |

## Makefile 集成

```makefile
.PHONY: test test-unit test-integration test-coverage mock

# 运行所有测试
test: test-unit test-integration

# 单元测试
test-unit:
	go test -v -race -short ./internal/...

# 集成测试
test-integration:
	go test -v -race -tags=integration ./tests/integration/...

# 带覆盖率的测试
test-coverage:
	go test -v -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# 生成 Mock
mock:
	mockery --all --output=mocks --outpkg=mocks

# 检查覆盖率门禁
coverage-check:
	@go test -coverprofile=coverage.out ./internal/... > /dev/null 2>&1
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$coverage < 80" | bc) -eq 1 ]; then \
		echo "Coverage $$coverage% is below 80% threshold"; \
		exit 1; \
	fi
	@echo "Coverage check passed"
```

## 测试命名规范

| 类型 | 命名格式 | 示例 |
|------|---------|------|
| 单元测试函数 | `Test{Type}_{Method}` | `TestUserService_GetProfile` |
| 子测试 | 中文描述场景 | `t.Run("用户不存在", ...)` |
| 测试文件 | `{source}_test.go` | `user_service_test.go` |
| 集成测试文件 | `{module}_test.go` | `user_test.go` |
| Mock 文件 | `{interface}.go` | `user_repository.go` |

## 参考文档

- 任务清单: `specs/001-smart-locker-backend/tasks.md` (T240-T269)
- 测试覆盖率要求: 单测 > 80%, 关键业务 > 90%
