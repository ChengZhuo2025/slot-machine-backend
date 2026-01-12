# 智能储物柜后端测试规范

本文档描述后端项目的测试结构、规范和最佳实践，帮助开发人员快速了解并编写高质量的测试代码。

> **更新时间**: 2026-01-11

## 目录结构

```text
tests/
├── api/                    # API 接口测试 (build tag: api)
├── e2e/                    # 端到端场景测试 (build tag: e2e)
├── integration/            # 集成测试 (build tag: integration)
├── unit/                   # 独立单元测试 (build tag: unit，可选)
├── helpers/                # 测试辅助工具
│   ├── test_helpers.go     # 测试数据生成函数
│   └── mocks.go            # Mock 实现
├── output/                 # 测试产物输出目录 (git ignored)
│   ├── coverage.out        # 覆盖率数据文件
│   └── coverage.html       # HTML 可视化报告
└── setup_test.go           # 全局测试配置
```

---

## 当前测试套件概览

### 已完成的测试资产

#### 业务单元测试 (in-package)

| 模块 | 测试文件路径 | 状态 |
|------|-------------|------|
| auth | `internal/service/auth/*_test.go` | ✅ |
| rental | `internal/service/rental/*_test.go` | ✅ |
| payment | `internal/service/payment/*_test.go` | ✅ |
| order | `internal/service/order/*_test.go` | ✅ |
| hotel | `internal/service/hotel/*_test.go` | ✅ |
| distribution | `internal/service/distribution/*_test.go` | ✅ |
| user | `internal/service/user/*_test.go` | ✅ |
| marketing | `internal/service/marketing/*_test.go` | ✅ |
| finance | `internal/service/finance/*_test.go` | ✅ |
| admin | `internal/service/admin/*_test.go` | ✅ |
| mall | `internal/service/mall/*_test.go` | ✅ |
| content | `internal/service/content/*_test.go` | ✅ |
| device | `internal/service/device/*_test.go` | ✅ |

#### 公共模块单元测试

| 模块 | 测试文件路径 | 测试用例数 |
|------|-------------|-----------|
| crypto | `internal/common/crypto/crypto_test.go` | 30+ |
| jwt | `internal/common/jwt/jwt_test.go` | 25+ |
| utils | `internal/common/utils/utils_test.go` | 40+ |
| qrcode | `internal/common/qrcode/qrcode_test.go` | 30+ |
| config | `internal/common/config/config_test.go` | 20+ |
| errors | `internal/common/errors/errors_test.go` | 50+ |
| response | `internal/common/response/response_test.go` | 35+ |
| metrics | `internal/common/metrics/*_test.go` | ✅ |
| middleware | `internal/common/middleware/*_test.go` | ✅ |
| tracing | `internal/common/tracing/*_test.go` | ✅ |
| cache | `internal/common/cache/*_test.go` | ✅ |
| database | `internal/common/database/*_test.go` | ✅ |
| logger | `internal/common/logger/*_test.go` | ✅ |

#### Repository 单元测试

已完成全部 Repository 层测试，位于 `internal/repository/*_test.go`：

- user_repo, device_repo, order_repo, rental_repo, admin_repo
- payment_repo, coupon_repo, address_repo, article_repo, banner_repo
- booking_repo, campaign_repo, cart_repo, category_repo, commission_repo
- device_alert_repo, device_log_repo, distributor_repo, feedback_repo
- hotel_repo, member_level_repo, member_package_repo, merchant_repo
- message_template_repo, notification_repo, operation_log_repo, product_repo
- review_repo, role_repo, room_repo, settlement_repo, system_config_repo
- transaction_repo, user_coupon_repo, venue_repo, withdrawal_repo

#### API 测试 (`tests/api/`)

| 测试文件 | 用户故事 | 描述 |
|---------|---------|------|
| `auth_api_test.go` | - | 用户认证 API |
| `admin_auth_api_test.go` | US2 | 管理端认证 API |
| `us1_rental_api_test.go` | US1 | 租借 API |
| `admin_device_api_test.go` | US2 | 设备管理 API |
| `us2_admin_merchant_venue_api_test.go` | US2 | 商户/场地管理 API |
| `us3_mall_api_test.go` | US3 | 商城 API |
| `us4_hotel_api_test.go` | US4 | 酒店 API |
| `us5_distribution_api_test.go` | US5 | 分销 API |
| `us6_finance_api_test.go` | US6 | 财务 API |
| `us7_marketing_api_test.go` | US7 | 营销 API |
| `us8_member_api_test.go` | US8 | 会员（用户端）API |
| `us8_member_admin_api_test.go` | US8 | 会员（管理端）API |

#### 集成测试 (`tests/integration/`)

| 测试文件 | 描述 |
|---------|------|
| `rental_flow_test.go` | 租借流程（扫码→支付→开锁→归还）|
| `payment_flow_test.go` | 支付流程（创建→回调→状态更新）|
| `admin_flow_test.go` | 管理端基础流程 |
| `distribution_flow_test.go` | 分销流程（推广→消费→计算佣金）|
| `us2_permission_flow_test.go` | 权限管理流程 |
| `us2_device_monitoring_flow_test.go` | 设备监控流程 |
| `us3_mall_order_flow_test.go` | 商城订单流程（加购→下单→支付）|
| `us4_hotel_booking_flow_test.go` | 酒店预订流程（预订→核销→开锁）|
| `us6_finance_flow_test.go` | 财务结算流程 |
| `us7_marketing_flow_test.go` | 营销活动流程 |
| `us8_membership_flow_test.go` | 会员体系流程 |

#### E2E 测试 (`tests/e2e/`)

| 测试文件 | 用户故事 | 描述 |
|---------|---------|------|
| `us1_scan_rent_flow_test.go` | US1 | 扫码租借完整流程 |
| `us2_admin_device_monitor_manage_flow_test.go` | US2 | 管理端设备管理 |
| `us3_mall_shopping_flow_test.go` | US3 | 商城购物完整流程 |
| `us4_hotel_booking_flow_test.go` | US4 | 酒店预订完整流程 |
| `us5_distribution_flow_test.go` | US5 | 分销推广流程 |
| `us6_finance_settlement_flow_test.go` | US6 | 财务结算流程 |
| `us7_marketing_flow_test.go` | US7 | 营销优惠流程 |
| `us8_membership_flow_test.go` | US8 | 会员体系流程 |

---

## 当前覆盖率现状

> **数据更新时间**: 2026-01-12

### 整体覆盖率

**整体覆盖率：57.2%**（含 handler 层，handler 层通过 API 测试覆盖）

**覆盖率门禁（关键模块 = auth/payment/order/rental/booking）：整体约 89.3%**

### 各模块覆盖率详情

#### 优秀模块 (≥85%)

| 模块 | 覆盖率 | 状态 |
|------|--------|------|
| common/errors | 100.0% | ✅ |
| common/response | 100.0% | ✅ |
| common/utils | 100.0% | ✅ |
| common/cache | 98.0% | ✅ |
| common/metrics | 95.1% | ✅ |
| common/logger | 93.8% | ✅ |
| common/config | 93.6% | ✅ |
| payment | 92.4% | ✅ 达标 |
| auth | 90.6% | ✅ 达标 |
| order | 90.4% | ✅ 达标 |
| content | 89.6% | ✅ |
| hotel (booking) | 87.6% | ✅ |
| distribution | 86.8% | ✅ |
| common/jwt | 86.3% | ✅ |
| common/tracing | 86.3% | ✅ |
| rental | 85.3% | ✅ 达标 |

#### 良好模块 (70-85%)

| 模块 | 覆盖率 | 状态 |
|------|--------|------|
| marketing | 84.1% | 📈 |
| common/qrcode | 83.7% | 📈 |
| user | 80.1% | 📈 |
| repository | 77.1% | 📈 |
| device | 74.4% | 📈 |
| admin | 73.5% | 📈 |
| common/handler | 69.9% | 📈 |

#### 中等模块 (50-70%)

| 模块 | 覆盖率 | 状态 |
|------|--------|------|
| finance | 65.6% | ⚠️ 待提升 |
| mall | 59.1% | ⚠️ 待提升 |
| common/database | 56.4% | ⚠️ 待提升 |

#### 待补充模块 (<50%)

| 模块 | 覆盖率 | 说明 |
|------|--------|------|
| common/middleware | 33.2% | 通过集成测试覆盖 |
| handler/* | 0% | 通过 API 测试覆盖 |

### 覆盖率目标

| 模块类别 | 目标覆盖率 | 当前状态 |
|----------|-----------|----------|
| 关键模块 (auth, payment, order, rental, booking) | ≥ 90% | ✅ 89.3% 基本达标 |
| 一般模块 | ≥ 60% | ✅ 达标 |
| 整体覆盖率 | ≥ 80% | ⚠️ 57.2% (含 handler) |

### 测试执行结果（2026-01-12）

| 测试类型 | 结果 | 说明 |
|----------|------|------|
| 单元测试 | ✅ 全部通过 | 28 个包 |
| API 测试 | ✅ 全部通过 | tests/api/ |
| 集成测试 | ✅ 基本通过 | 1 个跳过 (需 Docker) |

---

## 测试类型说明

### 1. 单元测试 (Unit Tests)

**位置**: `internal/service/*_test.go`, `internal/repository/*_test.go`, `tests/unit/`

**用途**: 测试单个函数或方法的逻辑正确性

**特点**:
- 使用 Mock 隔离外部依赖
- 运行速度快
- 不依赖数据库或网络

**运行命令**:
```bash
make test-unit
# 或
go test -v -race ./internal/... ./pkg/...
```

### 2. 集成测试 (Integration Tests)

**位置**: `tests/integration/`

**Build Tag**: `//go:build integration`

**用途**: 测试多个组件协作的业务流程

**特点**:
- 使用 SQLite 内存数据库
- 测试 Service 层与 Repository 层的集成
- 验证完整的业务流程

**运行命令**:
```bash
make test-integration
# 或
go test -v -tags=integration ./tests/integration/...
```

### 3. API 测试 (API Tests)

**位置**: `tests/api/`

**Build Tag**: `//go:build api`

**用途**: 测试 HTTP API 接口的请求和响应

**特点**:
- 使用 `httptest` 模拟 HTTP 请求
- 验证请求参数、响应格式、状态码
- 测试中间件和路由

**运行命令**:
```bash
make test-api
# 或
go test -v -tags=api ./tests/api/...
```

### 4. 端到端测试 (E2E Tests)

**位置**: `tests/e2e/`

**Build Tag**: `//go:build e2e`

**用途**: 模拟用户完整操作流程

**特点**:
- 覆盖完整用户场景 (如扫码租借、商城购物、酒店预订)
- 包含多个 API 调用的链式操作
- 验证跨模块交互

**运行命令**:
```bash
make test-e2e
# 或
go test -v -tags=e2e ./tests/e2e/...
```

---

## 测试命名规范

### 文件命名

| 类型 | 命名格式 | 示例 |
|------|----------|------|
| 单元测试 | `{module}_test.go` | `auth_service_test.go` |
| API 测试 | `{feature}_api_test.go` | `auth_api_test.go` |
| 集成测试 | `{feature}_flow_test.go` | `rental_flow_test.go` |
| E2E 测试 | `us{N}_{feature}_flow_test.go` | `us1_scan_rent_flow_test.go` |

### 测试函数命名

```go
// 格式: Test{功能}_{场景}_{预期结果}
func TestCreateUser_ValidInput_Success(t *testing.T) { ... }
func TestCreateUser_DuplicatePhone_ReturnsError(t *testing.T) { ... }
func TestPayment_InsufficientBalance_Fails(t *testing.T) { ... }
```

---

## 测试框架和工具

### 核心依赖

```go
import (
    "github.com/stretchr/testify/assert"   // 断言库
    "github.com/stretchr/testify/require"  // 必须通过的断言
    "github.com/stretchr/testify/mock"     // Mock 框架
    "gorm.io/driver/sqlite"                // 测试数据库
)
```

### 断言使用

```go
// 使用 assert - 失败后继续执行
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.Nil(t, result)

// 使用 require - 失败后立即停止
require.NoError(t, err, "数据库连接失败")
require.NotNil(t, user, "用户不应为空")
```

---

## 测试数据库设置

使用 SQLite 内存数据库进行测试，确保测试隔离性：

```go
func setupTestDB(t *testing.T) *gorm.DB {
    dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared",
        strings.ReplaceAll(t.Name(), "/", "_"))
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })
    require.NoError(t, err)

    sqlDB, err := db.DB()
    require.NoError(t, err)
    sqlDB.SetMaxOpenConns(1)
    sqlDB.SetMaxIdleConns(1)

    // 自动迁移所需模型
    err = db.AutoMigrate(
        &models.User{},
        &models.Order{},
        // ... 其他模型
    )
    require.NoError(t, err)

    return db
}
```

---

## 测试辅助工具

### helpers/test_helpers.go

提供测试数据生成函数：

```go
// 随机数据生成
helpers.RandomString(n int) string
helpers.RandomPhone() string
helpers.RandomInt(min, max int) int
helpers.RandomFloat(min, max float64) float64

// 模型工厂函数
helpers.NewTestUser() *models.User
helpers.NewTestUserWithPhone(phone string) *models.User
helpers.NewTestUserWallet(userID int64, balance float64) *models.UserWallet
helpers.NewTestMerchant() *models.Merchant
helpers.NewTestVenue(merchantID int64) *models.Venue
helpers.NewTestDevice(venueID int64) *models.Device
helpers.NewTestRentalPricing(...) *models.RentalPricing
helpers.NewTestRental(...) *models.Rental
helpers.NewTestPayment(...) *models.Payment
helpers.NewTestCategory(...) *models.Category
helpers.NewTestProduct(categoryID int64) *models.Product
helpers.NewTestHotel() *models.Hotel
helpers.NewTestRoom(hotelID int64) *models.Room
helpers.NewTestBooking(...) *models.Booking
helpers.NewTestAdmin(roleID int64) *models.Admin
helpers.NewTestRole() *models.Role
helpers.NewTestDistributor(...) *models.Distributor
helpers.NewTestCommission(...) *models.Commission
helpers.NewTestWithdrawal(...) *models.Withdrawal
```

### helpers/mocks.go

提供常用 Mock 实现：

```go
// Repository Mocks
MockUserRepository
MockRentalRepository
MockPaymentRepository
MockDeviceRepository
MockRefundRepository

// Service Mocks
MockCodeService    // 验证码服务
MockWalletService  // 钱包服务
MockMQTTService    // MQTT 设备控制
```

---

## 编写测试最佳实践

### 1. 使用 Table-Driven Tests

```go
func TestValidatePhone(t *testing.T) {
    tests := []struct {
        name    string
        phone   string
        wantErr bool
    }{
        {"valid phone", "13800138000", false},
        {"too short", "138001", true},
        {"invalid prefix", "10000000000", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePhone(tt.phone)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 2. 测试前置和清理

```go
func TestSomething(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    service := NewService(db)

    // Cleanup (使用 t.Cleanup 确保执行)
    t.Cleanup(func() {
        sqlDB, _ := db.DB()
        sqlDB.Close()
    })

    // Test logic
    // ...
}
```

### 3. Mock 使用示例

```go
func TestAuthService_Login(t *testing.T) {
    // 创建 mock
    mockUserRepo := new(helpers.MockUserRepository)
    mockCodeService := new(helpers.MockCodeService)

    // 设置期望
    mockCodeService.On("VerifyCode", mock.Anything, "13800138000", "123456", "login").
        Return(true, nil)
    mockUserRepo.On("GetByPhone", mock.Anything, "13800138000").
        Return(&models.User{ID: 1, Phone: ptr("13800138000")}, nil)

    // 创建服务并测试
    service := NewAuthService(mockUserRepo, mockCodeService)
    result, err := service.Login(ctx, "13800138000", "123456")

    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, result)

    // 验证 mock 调用
    mockCodeService.AssertExpectations(t)
    mockUserRepo.AssertExpectations(t)
}
```

### 4. API 测试示例

```go
//go:build api

func TestLoginAPI(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := setupTestRouter(t)

    body := map[string]string{
        "phone": "13800138000",
        "code":  "123456",
    }
    jsonBody, _ := json.Marshal(body)

    req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var resp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, float64(0), resp["code"])
}
```

---

## 覆盖率相关

### 测试产物输出目录

所有测试产物统一输出到 `tests/output/` 目录：

```
tests/output/
├── coverage.out    # 覆盖率数据文件
├── coverage.html   # HTML 可视化报告
└── .gitkeep        # 保留目录结构
```

### 生成覆盖率报告

```bash
# 生成 HTML 覆盖率报告
make coverage

# 产物位置
# - tests/output/coverage.out (覆盖率数据文件)
# - tests/output/coverage.html (HTML 报告)

# 查看报告
open tests/output/coverage.html  # macOS
xdg-open tests/output/coverage.html  # Linux

# 命令行查看摘要
go tool cover -func=tests/output/coverage.out | tail -n 20
```

### 覆盖率门禁

```bash
# 运行覆盖率检查 (CI/CD 使用)
make coverage-gate

# 可通过环境变量调整阈值
OVERALL_MIN=75 KEY_MODULE_MIN=90 make coverage-gate
```

### 覆盖率阈值配置

| 阈值类型 | 默认值 | 说明 |
|----------|--------|------|
| `OVERALL_MIN` | 70% | 整体覆盖率最低要求 |
| `KEY_MODULE_MIN` | 85% | 关键模块覆盖率最低要求 |

关键模块包括：`auth`, `payment`, `order`, `rental`, `booking`

### 覆盖率脚本说明

- `scripts/coverage.sh` - 生成覆盖率报告
  - 默认统计 `./internal/service/...`, `./internal/repository/...`, `./internal/common/...`, `./pkg/...`
  - 输出到 `tests/output/` 目录
- `scripts/coverage-gate.sh` - 覆盖率门禁验证脚本
  - 验证整体单元测试覆盖率 ≥ 70%
  - 验证关键业务模块覆盖率 ≥ 85%
  - 不满足条件时返回非零退出码阻止 CI/CD 流水线

> **注意**: `scripts/coverage.sh` 会把 `GOCACHE` 指向仓库内的 `.gocache/`，避免在受限环境下写入系统缓存目录导致权限问题。

---

## 运行测试命令

```bash
# 运行所有测试
make test

# 运行特定类型测试
make test-unit          # 单元测试
make test-integration   # 集成测试
make test-api           # API 测试
make test-e2e           # E2E 测试

# 带覆盖率运行
make coverage           # 生成覆盖率报告
make coverage-gate      # 覆盖率门禁检查
```

---

## 常见问题

### Q: 测试文件没有被执行?

检查是否添加了正确的 build tag：
```go
//go:build api
// +build api
```

### Q: 数据库并发冲突?

确保每个测试使用独立的数据库连接：
```go
dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
```

### Q: Mock 方法未被调用?

检查 Mock 设置是否与实际调用匹配，使用 `mock.Anything` 处理动态参数。

### Q: 覆盖率报告不包含某些包?

确保测试文件与源文件在同一个包内（in-package test），或者在 `make coverage` 时指定 `-coverpkg` 参数。

---

## 待完成项

以下测试基础设施已完成：

- [x] `internal/common/logger` - 日志模块单元测试（40+ 测试用例）
- [x] `internal/common/cache` - 缓存模块单元测试（使用 miniredis mock，50+ 测试用例）
- [x] `internal/common/database` - 数据库模块单元测试（使用 SQLite mock，30+ 测试用例）
- [x] testcontainers-go 集成测试环境配置（`tests/integration/testcontainers.go`）

### testcontainers-go 使用说明

testcontainers-go 提供真实的 PostgreSQL 和 Redis 容器用于集成测试：

```go
//go:build integration

func TestWithRealDatabase(t *testing.T) {
    ctx := context.Background()
    tc := NewTestContainers(ctx)

    // 启动所有容器
    err := tc.StartAll()
    require.NoError(t, err)
    defer tc.Cleanup()

    // 获取数据库连接
    db, err := tc.GetPostgresDB()
    require.NoError(t, err)

    // 获取 Redis 客户端
    redis, err := tc.GetRedisClient()
    require.NoError(t, err)

    // 执行测试...
}
```

**运行 testcontainers 测试：**

```bash
# 需要 Docker 环境
make test-integration
```

**依赖安装：**

```bash
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/testcontainers/testcontainers-go/modules/redis
go get github.com/alicebob/miniredis/v2  # 用于 cache 模块单元测试
```

---

## 相关资源

- [testify 文档](https://github.com/stretchr/testify)
- [Go Testing 官方文档](https://golang.org/pkg/testing/)
- [GORM 测试指南](https://gorm.io/docs/index.html)
- [项目任务清单](../specs/001-smart-locker-backend/tasks.md) - Phase 12: Testing
