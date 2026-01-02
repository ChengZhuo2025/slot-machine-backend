# Implementation Plan: 爱上杜美人智能开锁管理系统后端服务

**Branch**: `001-smart-locker-backend` | **Date**: 2026-01-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-smart-locker-backend/spec.md`

## Summary

构建一个 B2B2C 智能柜租借与电商平台后端系统，核心功能包括：
- 用户扫码租借成人情趣用品（P1 核心业务）
- 智能柜设备监控与远程控制
- 酒店房间智能柜预订与开锁
- 商城购物与订单管理
- 分销推广与佣金计算
- 多角色权限管理

技术方案采用 Go 微服务架构，基于 Gin + GORM + PostgreSQL + Redis + MQTT 技术栈。

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**:
- Web 框架: Gin v1.10+
- ORM: GORM v1.25+
- 配置管理: Viper
- MQTT 客户端: paho.mqtt.golang
- 二维码生成: go-qrcode
- JWT 认证: golang-jwt/jwt/v5
- 支付 SDK: wechatpay-go, alipay-sdk-go

**Storage**:
- 主数据库: PostgreSQL 15+
- 缓存: Redis 7+ (会话、热点数据、分布式锁)
- 消息队列: Redis Stream (异步任务、事件驱动)
- 对象存储: 阿里云 OSS / 腾讯云 COS

**IoT Communication**:
- 协议: MQTT 3.1.1/5.0
- Broker: EMQX 5.0+
- QoS: 至少 QoS 1 (确保消息送达)

**Testing**:
- 单元测试: Go testing + testify
- 集成测试: dockertest + testcontainers-go
- API 测试: httptest + go-resty

**Target Platform**: Linux (Docker/Kubernetes)

**Project Type**: Microservices Backend (多服务架构)

**Architecture Strategy**: 模块化单体优先 (Modular Monolith First)

> **当前阶段**: 采用模块化单体架构，所有业务模块通过单一 API Gateway 对外提供服务。各模块在 `internal/` 目录下按业务领域划分，保持清晰的边界和独立性。
>
> **演进路径**: 项目结构 `cmd/` 目录预留了 14 个服务入口，为未来微服务拆分做好准备。当业务规模增长到需要独立扩展或团队规模需要独立部署时，可按模块边界逐步拆分为独立微服务。
>
> **选择理由**:
> - 初期开发效率更高，避免分布式系统复杂性
> - 单一部署单元，运维成本低
> - 模块边界清晰，未来拆分成本可控
> - 符合 "做大之前先做好" 的实践原则

**Performance Goals**:
- API 响应时间: < 200ms (P95)
- 支持 10,000 并发用户
- 数据库查询: < 50ms (P95)
- 扫码到开锁: < 60s 完整流程

**Constraints**:
- 系统可用性: ≥ 99.9%
- 设备状态同步: < 30s
- 支付成功率: ≥ 99.5%
- 敏感数据 100% 加密存储

**Scale/Scope**:
- 初期支持 1,000+ 智能柜设备
- 10,000+ 注册用户
- 14 个业务模块
- 65+ API 端点

**Development Data**:
- 数据库迁移需包含 seed 脚本，使用前端 mock 数据作为初始化测试数据
- 数据来源: `admin-frontend` 和 `user-frontend` 的 mock 模拟数据
- 覆盖范围: 用户、设备、场地、商户、酒店、房间、商品、订单等核心业务数据
- 提供一键初始化命令，便于开发环境快速搭建

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Check

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. 需求一致性 | ✅ PASS | 65 条功能需求已明确定义，覆盖前端所有交互功能 |
| II. 安全优先 | ✅ PASS | 敏感数据加密、JWT 认证、RBAC 权限、审计日志已纳入设计 |
| III. 微服务边界 | ✅ PASS | 14 个业务模块按职责划分，服务间通过 API 通信 |
| IV. API 质量标准 | ✅ PASS | 响应时间 < 200ms、OpenAPI 3.0 文档、语义化版本 |
| V. 可测试性 | ✅ PASS | 单测覆盖率 > 80%、关键业务 > 90%、集成测试 |
| VI. 可观测性 | ✅ PASS | 结构化日志、Prometheus 监控、分布式追踪 |
| VII. 代码质量门禁 | ✅ PASS | CI/CD 流水线、Code Review、静态分析、安全扫描 |

### Quality Attributes Check

| Attribute | Requirement | Design Target |
|-----------|-------------|---------------|
| 性能 | API < 200ms (P95) | 数据库索引优化 + Redis 缓存 |
| 可用性 | ≥ 99.9% | 多副本部署 + 健康检查 + 自动恢复 |
| 安全 | 0 严重漏洞 | 输入验证 + SQL 注入防护 + XSS 防护 |
| 可扩展 | 水平扩展 | 无状态服务 + 容器化部署 |

### Compliance Check

| Requirement | Status | Implementation |
|-------------|--------|----------------|
| PIPL 个人信息保护 | ✅ PASS | 用户同意机制、数据加密、导出/删除接口 |
| PCI DSS 支付安全 | ✅ PASS | 支付数据加密、访问控制、审计日志 |

## Project Structure

### Documentation (this feature)

```text
specs/001-smart-locker-backend/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (OpenAPI specs)
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
backend/
├── cmd/                          # 服务入口
│   ├── api-gateway/              # API 网关
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
│   │   ├── middleware/           # 中间件
│   │   └── utils/                # 工具函数
│   │
│   ├── models/                   # 数据模型
│   │   ├── user.go
│   │   ├── device.go
│   │   ├── order.go
│   │   └── ...
│   │
│   ├── repository/               # 数据访问层
│   │   ├── user_repo.go
│   │   ├── device_repo.go
│   │   └── ...
│   │
│   ├── service/                  # 业务逻辑层
│   │   ├── user/
│   │   ├── device/
│   │   ├── order/
│   │   └── ...
│   │
│   └── handler/                  # HTTP 处理器
│       ├── user/
│       ├── device/
│       └── ...
│
├── pkg/                          # 可复用公共包
│   ├── auth/                     # JWT 认证
│   ├── crypto/                   # 加密解密
│   ├── payment/                  # 支付集成
│   │   ├── wechat/
│   │   └── alipay/
│   ├── sms/                      # 短信服务
│   ├── oss/                      # 对象存储
│   ├── qrcode/                   # 二维码生成
│   └── response/                 # 统一响应格式
│
├── api/                          # API 定义
│   └── openapi/                  # OpenAPI 3.0 规范
│       ├── user.yaml
│       ├── device.yaml
│       └── ...
│
├── migrations/                   # 数据库迁移
│   ├── 000001_init_users.up.sql
│   ├── 000001_init_users.down.sql
│   └── ...
│
├── configs/                      # 配置文件
│   ├── config.yaml
│   ├── config.dev.yaml
│   └── config.prod.yaml
│
├── deployments/                  # 部署配置
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   └── k8s/
│       ├── deployment.yaml
│       └── service.yaml
│
├── scripts/                      # 脚本
│   ├── build.sh
│   └── migrate.sh
│
└── tests/                        # 测试
    ├── unit/
    ├── integration/
    └── e2e/
```

**Structure Decision**: 采用微服务架构，每个业务模块独立服务。使用 `internal/` 目录隔离内部实现，`pkg/` 目录放置可复用组件。服务间通过 HTTP API 和消息队列通信。

## Complexity Tracking

> 无违规项，所有设计符合 Constitution 原则。

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
