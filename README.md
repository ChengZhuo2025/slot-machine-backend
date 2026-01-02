# 爱上杜美人智能开锁管理系统后端服务

B2B2C 智能柜租借与电商平台后端系统

## 项目概述

本项目是"爱上杜美人"智能开锁管理系统的后端服务，提供智能柜租借、酒店预订、商城购物、分销推广等核心业务功能。系统采用模块化单体架构，基于 Go 语言开发，为未来微服务拆分做好了准备。

### 核心功能

| 模块 | 功能描述 | 优先级 |
|------|----------|--------|
| 用户扫码租借 | 扫码→支付→开锁→归还→结算的完整租借流程 | P1 |
| 设备监控管理 | 实时监控设备状态、远程控制、场地商户管理 | P1 |
| 商城购物 | 商品浏览、购物车、下单支付、订单管理 | P2 |
| 酒店预订 | 房间预订、核销码/开锁码、智能柜联动 | P2 |
| 分销推广 | 推广链接、佣金计算、团队管理、提现 | P2 |
| 财务结算 | 收入统计、商户/分销结算、报表导出 | P3 |
| 营销活动 | 优惠券、活动配置、会员套餐 | P3 |
| 会员体系 | 积分累积、等级升级、会员权益 | P3 |

### 技术栈

| 组件 | 技术选型 | 版本 |
|------|----------|------|
| 语言 | Go | 1.25+ |
| Web 框架 | Gin | 1.10+ |
| ORM | GORM | 1.25+ |
| 数据库 | PostgreSQL | 15+ |
| 缓存 | Redis | 7+ |
| 消息队列 | Redis Stream | - |
| MQTT Broker | EMQX | 5.0+ |
| 配置管理 | Viper | - |
| JWT 认证 | golang-jwt/jwt | v5 |

## 环境要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.25+ | 编程语言 |
| PostgreSQL | 15+ | 主数据库 |
| Redis | 7+ | 缓存/消息队列/分布式锁 |
| EMQX | 5.0+ | MQTT Broker（设备通信） |
| Docker | 24+ | 容器化 |
| Docker Compose | 2.20+ | 本地编排 |

## 快速开始

### 1. 克隆项目

```bash
git clone <repository-url>
cd backend
```

### 2. 启动依赖服务

```bash
# 启动 PostgreSQL、Redis、EMQX
docker-compose -f deployments/docker/docker-compose.yml up -d postgres redis emqx
```

### 3. 配置环境变量

```bash
# 复制配置文件
cp configs/config.example.yaml configs/config.yaml

# 编辑配置（数据库、Redis、MQTT 等）
vim configs/config.yaml
```

### 4. 初始化数据库

```bash
# 执行数据库迁移
make migrate

# 初始化种子数据（开发环境）
make seed
```

### 5. 运行服务

```bash
# 开发模式
make run

# 或直接运行
go run cmd/api-gateway/main.go
```

### 6. 验证服务

```bash
# 健康检查
curl http://localhost:8000/health

# API 文档
open http://localhost:8000/swagger/index.html
```

## 项目结构

```
backend/
├── cmd/                          # 服务入口
│   ├── api-gateway/              # API 网关（主入口）
│   ├── user-service/             # 用户服务
│   ├── device-service/           # 设备服务
│   ├── order-service/            # 订单服务
│   ├── payment-service/          # 支付服务
│   ├── rental-service/           # 租借服务
│   ├── hotel-service/            # 酒店预订服务
│   ├── mall-service/             # 商城服务
│   ├── distribution-service/     # 分销服务
│   ├── marketing-service/        # 营销服务
│   ├── finance-service/          # 财务服务
│   ├── content-service/          # 内容服务
│   ├── notification-service/     # 消息通知服务
│   └── admin-service/            # 管理后台服务
│
├── internal/                     # 内部包（不对外暴露）
│   ├── common/                   # 公共组件
│   │   ├── config/               # 配置管理
│   │   ├── database/             # 数据库连接
│   │   ├── cache/                # Redis 缓存
│   │   ├── mq/                   # 消息队列
│   │   ├── mqtt/                 # MQTT 客户端
│   │   ├── logger/               # 日志组件
│   │   └── middleware/           # 中间件
│   │
│   ├── models/                   # 数据模型
│   ├── repository/               # 数据访问层
│   ├── service/                  # 业务逻辑层
│   └── handler/                  # HTTP 处理器
│
├── pkg/                          # 可复用公共包
│   ├── auth/                     # JWT 认证
│   ├── crypto/                   # 加密解密
│   ├── payment/                  # 支付集成
│   │   ├── wechat/               # 微信支付
│   │   └── alipay/               # 支付宝
│   ├── sms/                      # 短信服务
│   ├── oss/                      # 对象存储
│   ├── qrcode/                   # 二维码生成
│   └── response/                 # 统一响应格式
│
├── api/openapi/                  # OpenAPI 3.0 规范
├── migrations/                   # 数据库迁移
├── seeds/                        # 种子数据
├── configs/                      # 配置文件
├── deployments/                  # 部署配置
│   ├── docker/                   # Docker 配置
│   └── k8s/                      # Kubernetes 配置
├── scripts/                      # 脚本
└── tests/                        # 测试文件
    ├── unit/                     # 单元测试
    ├── integration/              # 集成测试
    └── e2e/                      # 端到端测试
```

## API 文档

系统提供两套 API：

### 用户端 API (User API)

面向微信小程序/H5 用户，主要接口：

| 模块 | 端点 | 说明 |
|------|------|------|
| 认证 | `POST /auth/sms/send` | 发送短信验证码 |
| 认证 | `POST /auth/login/sms` | 手机验证码登录 |
| 认证 | `POST /auth/login/wechat` | 微信授权登录 |
| 用户 | `GET /user/profile` | 获取用户信息 |
| 用户 | `GET /user/wallet` | 获取钱包信息 |
| 设备 | `GET /devices/{device_no}` | 扫码获取设备信息 |
| 租借 | `POST /rentals` | 创建租借订单 |
| 租借 | `POST /rentals/{id}/return` | 归还租借物品 |
| 订单 | `GET /orders` | 获取订单列表 |
| 支付 | `POST /payments/create` | 创建支付 |
| 酒店 | `GET /hotels` | 获取酒店列表 |
| 酒店 | `POST /bookings` | 创建酒店预订 |
| 商城 | `GET /mall/products` | 获取商品列表 |
| 商城 | `POST /mall/cart` | 添加购物车 |
| 分销 | `POST /distribution/apply` | 申请成为分销商 |
| 营销 | `GET /marketing/coupons/available` | 获取可领取优惠券 |

### 管理端 API (Admin API)

面向后台管理系统，主要接口：

| 模块 | 端点 | 说明 |
|------|------|------|
| 认证 | `POST /admin/auth/login` | 管理员登录 |
| 设备 | `GET /admin/devices` | 设备列表 |
| 设备 | `POST /admin/devices/{id}/unlock` | 远程开锁 |
| 商户 | `GET /admin/merchants` | 商户管理 |
| 订单 | `GET /admin/orders` | 订单管理 |
| 财务 | `GET /admin/finance/statistics` | 财务统计 |
| 分销 | `GET /admin/distributors` | 分销商管理 |
| 营销 | `POST /admin/coupons` | 创建优惠券 |

详细 API 文档请参考 `specs/001-smart-locker-backend/contracts/` 目录下的 OpenAPI 规范文件。

## 常用命令

```bash
# 构建
make build

# 运行
make run

# 运行测试
make test
make test-unit
make test-integration

# 测试覆盖率
make coverage

# 代码检查
make lint

# 数据库迁移
make migrate-up
make migrate-down

# 初始化种子数据
make seed

# 回滚最后一次迁移
make migrate-down

# 重置所有迁移
make migrate-reset

# 查看迁移状态
make migrate-status

# 重置数据库（清空并重建）
make reset-db

# 生成 Swagger 文档
make swagger

# Docker 构建
make docker-build
```

## 配置说明

### config.yaml 配置示例

```yaml
server:
  port: 8000
  mode: debug  # debug/release

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: smart_locker
  sslmode: disable

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

mqtt:
  broker: tcp://localhost:1883
  client_id: smart-locker-backend
  username: ""
  password: ""

jwt:
  secret: your-secret-key
  access_expire: 7200      # 2小时（秒）
  refresh_expire: 604800   # 7天（秒）

payment:
  wechat:
    app_id: ""
    mch_id: ""
    api_key: ""
    cert_path: ""
  alipay:
    app_id: ""
    private_key: ""
    alipay_public_key: ""

sms:
  provider: aliyun  # aliyun/tencent
  access_key: ""
  secret_key: ""
  sign_name: ""
  template_code: ""

oss:
  provider: aliyun
  endpoint: ""
  bucket: ""
  access_key: ""
  secret_key: ""
```

## 数据模型

系统包含以下核心数据域：

| 域 | 主要实体 |
|----|----------|
| 用户域 | User, UserWallet, MemberLevel, Admin, Role, Permission |
| 设备域 | Merchant, Venue, Device, DeviceLog, DeviceMaintenance |
| 订单域 | Order, OrderItem, Payment, Refund, Rental, RentalPricing |
| 酒店域 | Hotel, Room, RoomTimeSlot, Booking |
| 商城域 | Category, Product, ProductSku, CartItem, Review |
| 分销域 | Distributor, Commission, Withdrawal |
| 营销域 | Coupon, UserCoupon, Campaign, MemberPackage |
| 系统域 | SystemConfig, OperationLog, SmsCode, Banner |

详细数据模型设计请参考 `specs/001-smart-locker-backend/data-model.md`。

## 设备通信协议

系统通过 MQTT 协议与智能柜设备通信：

### 主题设计

```
# 设备上行（设备 → 服务器）
device/{device_id}/status          # 设备状态上报
device/{device_id}/event           # 设备事件（开锁、关锁、故障）
device/{device_id}/heartbeat       # 心跳

# 设备下行（服务器 → 设备）
device/{device_id}/command         # 控制命令（开锁、重启）
device/{device_id}/config          # 配置下发
```

### QoS 策略

| 消息类型 | QoS | 说明 |
|----------|-----|------|
| 开锁命令 | 1 | 至少一次送达 |
| 状态上报 | 1 | 至少一次送达 |
| 心跳 | 0 | 允许丢失 |
| 配置下发 | 2 | 精确一次 |

## 开发规范

### 代码风格

```bash
# 格式化代码
gofmt -w .

# 静态检查
golangci-lint run
```

### 提交规范

```
feat: 新功能
fix: 修复bug
docs: 文档更新
refactor: 重构
test: 测试
chore: 其他
```

### 分支策略

- `main`: 生产分支
- `develop`: 开发分支
- `feature/*`: 功能分支
- `bugfix/*`: 修复分支
- `hotfix/*`: 紧急修复

### 测试覆盖率要求

- 整体单元测试覆盖率：≥ 80%
- 关键业务模块（auth/payment/order/rental/booking）：≥ 90%

## 部署

### Docker Compose（开发环境）

```bash
cd deployments/docker
docker-compose up -d
```

### Kubernetes（生产环境）

```bash
kubectl apply -f deployments/k8s/
```

### 生产环境建议

- 服务至少 2 副本部署
- PostgreSQL 主从复制
- Redis Sentinel 高可用
- EMQX 集群模式
- Nginx Ingress 负载均衡

## 性能目标

| 指标 | 目标值 |
|------|--------|
| API 响应时间 (P95) | < 200ms |
| 并发用户数 | 10,000+ |
| 数据库查询 (P95) | < 50ms |
| 扫码到开锁 | < 60s |
| 系统可用性 | ≥ 99.9% |
| 支付成功率 | ≥ 99.5% |
| 设备状态同步延迟 | < 30s |

## 安全设计

- **传输安全**: 所有接口强制 HTTPS
- **认证授权**: JWT Token + RBAC 权限控制
- **数据加密**: 敏感数据使用 AES-256-GCM 加密
- **密码存储**: bcrypt 哈希
- **防刷策略**: 验证码 1次/分钟，登录 5次/分钟/IP
- **审计日志**: 记录所有管理员操作

## 文档索引

| 文档 | 路径 | 说明 |
|------|------|------|
| 功能规格 | `specs/001-smart-locker-backend/spec.md` | 详细功能需求 |
| 实现计划 | `specs/001-smart-locker-backend/plan.md` | 技术方案和架构 |
| 数据模型 | `specs/001-smart-locker-backend/data-model.md` | 数据库设计 |
| 技术研究 | `specs/001-smart-locker-backend/research.md` | 技术选型分析 |
| 任务清单 | `specs/001-smart-locker-backend/tasks.md` | 开发任务分解 |
| 用户 API | `specs/001-smart-locker-backend/contracts/user-api.yaml` | 用户端 API 规范 |
| 管理 API | `specs/001-smart-locker-backend/contracts/admin-api.yaml` | 管理端 API 规范 |

## 参考资料

- [Gin 文档](https://gin-gonic.com/docs/)
- [GORM 文档](https://gorm.io/docs/)
- [EMQX 文档](https://www.emqx.io/docs/)
- [微信支付文档](https://pay.weixin.qq.com/wiki/doc/api/index.html)
- [支付宝文档](https://opendocs.alipay.com/open/)

## License

Copyright 2026. All rights reserved.
