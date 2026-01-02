---
name: Project Guide - 智能开锁管理系统
description: This skill should be used when the user asks about "project overview", "how to start", "project architecture", "which module", "what features", "development workflow", "project structure", "business logic", "system design", or needs guidance on understanding the smart locker system, navigating between modules, or making architectural decisions. This is the entry point skill for the project.
version: 1.0.0
---

# 爱上杜美人智能开锁管理系统 - 项目总体指南

This skill serves as the central hub for the smart locker backend project, providing architectural overview, module navigation, and cross-cutting concerns.

## Project Overview

### Business Background

**爱上杜美人** 是一个 B2B2C 智能柜租借与电商平台，核心业务：

| 业务线 | 优先级 | 描述 |
|--------|--------|------|
| 智能柜租借 | P1 | 用户扫码租借成人情趣用品 |
| 设备管理 | P1 | 智能柜监控、远程控制、告警 |
| 商城购物 | P2 | 商品浏览、购物车、下单配送 |
| 酒店预订 | P2 | 房间预订、核销、开锁码 |
| 分销推广 | P2 | 推广链接、佣金计算、提现 |
| 财务结算 | P3 | 对账、分账、报表导出 |
| 营销活动 | P3 | 优惠券、活动、会员套餐 |

### Technology Stack

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend Clients                        │
│              微信小程序 / H5 / 管理后台                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     API Gateway (Gin)                        │
│              认证 / 限流 / 路由 / 日志                         │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  User Module  │    │ Device Module │    │ Order Module  │
│  用户/认证/钱包 │    │ 设备/场地/商户 │    │ 订单/支付/租借 │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Data & Messaging Layer                    │
│  PostgreSQL 15+ │ Redis 7+ │ EMQX (MQTT) │ 阿里云 OSS       │
└─────────────────────────────────────────────────────────────┘
```

### Core Tech Choices

| Layer | Technology | Purpose |
|-------|------------|---------|
| Language | Go 1.25+ | High performance, concurrency |
| Web Framework | Gin v1.10+ | HTTP routing, middleware |
| ORM | GORM v1.25+ | Database operations |
| Database | PostgreSQL 15+ | Primary data store |
| Cache | Redis 7+ | Session, cache, distributed lock |
| Message Queue | Redis Stream | Async tasks, events |
| IoT Protocol | MQTT (EMQX 5.0+) | Device communication |
| Authentication | JWT | Stateless auth |
| Payment | WeChat Pay, Alipay | Payment processing |

## System Architecture

### Modular Monolith Strategy

```
当前阶段: 模块化单体架构
├── 单一部署单元，运维成本低
├── 模块边界清晰，按业务领域划分
└── 预留微服务拆分路径

未来演进: 按需拆分为微服务
├── 当单模块需要独立扩展时
├── 当团队规模需要独立部署时
└── 按 cmd/ 目录结构拆分
```

### 14 Business Modules

| Module | Directory | Responsibility |
|--------|-----------|----------------|
| **认证授权** | `internal/service/auth/` | 登录、JWT、权限 |
| **用户管理** | `internal/service/user/` | 用户信息、钱包、会员 |
| **设备管理** | `internal/service/device/` | 设备、场地、商户 |
| **租借服务** | `internal/service/rental/` | 租借流程、定价 |
| **酒店预订** | `internal/service/hotel/` | 酒店、房间、预订 |
| **商城服务** | `internal/service/mall/` | 商品、购物车、分类 |
| **订单管理** | `internal/service/order/` | 统一订单、状态流转 |
| **支付服务** | `internal/service/payment/` | 支付、回调、退款 |
| **分销服务** | `internal/service/distribution/` | 分销商、佣金、团队 |
| **营销服务** | `internal/service/marketing/` | 优惠券、活动、套餐 |
| **财务服务** | `internal/service/finance/` | 结算、提现、报表 |
| **内容服务** | `internal/service/content/` | 文章、通知、模板 |
| **系统管理** | `internal/service/system/` | 配置、日志、缓存 |
| **仪表盘** | `internal/service/dashboard/` | 数据统计、概览 |

## Development Workflow

### Getting Started

```bash
# 1. Clone and setup
git clone <repository-url>
cd backend

# 2. Start dependencies
docker-compose up -d postgres redis emqx

# 3. Configure
cp configs/config.example.yaml configs/config.yaml

# 4. Run migrations
make migrate-up

# 5. Start server
make run
```

### Development Cycle

```
需求理解 → 查阅 spec.md → 设计方案 → 编码实现 → 测试验证 → 代码审查
    │
    └── 参考文档:
        ├── specs/001-smart-locker-backend/spec.md      # 功能需求
        ├── specs/001-smart-locker-backend/data-model.md # 数据模型
        └── specs/001-smart-locker-backend/contracts/    # API 契约
```

### Key Commands

| Command | Purpose |
|---------|---------|
| `make run` | Start development server |
| `make build` | Build binary |
| `make test` | Run all tests |
| `make lint` | Code quality check |
| `make swagger` | Generate API docs |
| `make migrate-up` | Apply migrations |
| `make migrate-down` | Rollback migration |

## Skill Navigation Guide

根据任务类型选择合适的专业 skill：

### 编码开发

| 任务 | 推荐 Skill | 触发词 |
|------|-----------|--------|
| 创建服务/Handler | **go-backend-dev** | "创建服务", "实现功能", "Go代码" |
| 数据库/迁移 | **database-management** | "数据库", "迁移", "查询优化" |
| API 端点 | **api-development** | "API", "路由", "接口" |

### 特定领域

| 任务 | 推荐 Skill | 触发词 |
|------|-----------|--------|
| 设备通信 | **iot-mqtt** | "MQTT", "设备控制", "开锁" |
| 支付集成 | **payment-integration** | "支付", "微信支付", "退款" |

### 质量保障

| 任务 | 推荐 Skill | 触发词 |
|------|-----------|--------|
| 测试/调试 | **testing-debugging** | "测试", "单元测试", "调试" |

### 跨领域决策

当问题涉及多个领域时，本 skill 提供协调指导：

```
"实现用户扫码租借完整流程"
  ├── 设备扫码 → iot-mqtt (获取设备信息)
  ├── 创建订单 → go-backend-dev (业务逻辑)
  ├── 支付处理 → payment-integration (支付集成)
  ├── 开锁指令 → iot-mqtt (设备控制)
  └── 数据存储 → database-management (事务处理)
```

## Cross-Cutting Concerns

### Error Handling Standard

```go
// 统一错误码规范
const (
    // 通用错误 (1-999)
    CodeSuccess      = 0
    CodeBadRequest   = 400
    CodeUnauthorized = 401

    // 用户模块 (1000-1999)
    CodeUserNotFound = 1001

    // 设备模块 (2000-2999)
    CodeDeviceBusy   = 2003

    // 订单模块 (3000-3999)
    CodeOrderNotFound = 3001

    // 支付模块 (4000-4999)
    CodePaymentFailed = 4001
)
```

### Logging Convention

```go
// 使用结构化日志
logger.Log.With(
    "module", "rental",
    "action", "create_order",
    "user_id", userID,
    "device_no", deviceNo,
).Info("Creating rental order")
```

### Security Requirements

| 要求 | 实现方式 |
|------|---------|
| 认证 | JWT + Refresh Token |
| 授权 | RBAC 角色权限 |
| 数据加密 | 敏感字段 AES 加密 |
| SQL 注入 | GORM 参数化查询 |
| 审计日志 | 管理员操作全记录 |

### Performance Targets

| Metric | Target |
|--------|--------|
| API 响应时间 | < 200ms (P95) |
| 数据库查询 | < 50ms (P95) |
| 扫码到开锁 | < 60s 完整流程 |
| 并发用户 | 10,000+ |
| 系统可用性 | ≥ 99.9% |

## Core Business Flows

### 租借流程 (P1 核心)

```
用户扫码 → 获取设备信息 → 选择时长 → 创建订单 → 支付
    │                                              │
    │                                              ▼
    │                                         支付成功
    │                                              │
    │                                              ▼
    │                                    发送开锁指令 (MQTT)
    │                                              │
    │                                              ▼
    │                                         设备开锁
    │                                              │
    │                                              ▼
    └──────────────────────────────────────── 使用中
                                                   │
                                                   ▼
                                             用户归还扫码
                                                   │
                                                   ▼
                                         检测关锁 → 结算 → 退押金
```

### 酒店预订流程 (P2)

```
选择酒店/房间 → 选择时段 → 创建预订 → 支付
                                         │
                                         ▼
                              生成核销码 + 开锁码
                                         │
                                         ▼
                              前台核销 → 状态更新
                                         │
                                         ▼
                              输入开锁码 → 开锁 → 使用
                                         │
                                         ▼
                              关锁 → 完成订单
```

## Additional Resources

### Project Documentation

| Document | Location | Content |
|----------|----------|---------|
| 功能规范 | `specs/001-smart-locker-backend/spec.md` | 完整需求定义 |
| 数据模型 | `specs/001-smart-locker-backend/data-model.md` | 实体定义 |
| 实施计划 | `specs/001-smart-locker-backend/plan.md` | 技术方案 |
| API 契约 | `specs/001-smart-locker-backend/contracts/` | OpenAPI 规范 |
| 快速开始 | `specs/001-smart-locker-backend/quickstart.md` | 环境搭建 |

### Reference Files

- **`references/module-dependencies.md`** - 模块依赖关系
- **`references/deployment-guide.md`** - 部署指南
