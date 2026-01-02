# Research: 爱上杜美人智能开锁管理系统后端服务

**Date**: 2026-01-02 | **Branch**: `001-smart-locker-backend`

## 1. 微服务架构设计

### Decision: 采用单体先行、服务划分就绪的架构策略

**Rationale**:
- 项目初期团队规模有限，单体架构降低运维复杂度
- 代码结构按微服务边界组织，便于后期拆分
- 共享数据库简化事务处理，避免分布式事务复杂性
- 当单服务性能瓶颈出现时再逐步拆分

**Alternatives Considered**:
| Alternative | Rejected Because |
|-------------|------------------|
| 完全微服务 | 初期运维成本高，团队学习曲线陡峭 |
| 纯单体 | 代码耦合，后期拆分困难 |

**Implementation**:
```
Phase 1: 单体应用 + 模块化代码结构
Phase 2: 按负载拆分高频服务（设备服务、支付服务）
Phase 3: 完全微服务化
```

## 2. 服务划分策略

### Decision: 按业务域划分 + API 网关统一入口

**服务划分**:

| 服务 | 职责 | 优先级 |
|------|------|--------|
| api-gateway | 路由、认证、限流、日志 | P0 |
| user-service | 用户、认证、会员、钱包 | P0 |
| device-service | 设备、场地、商户、MQTT | P0 |
| rental-service | 租借、定价、超时处理 | P0 |
| order-service | 订单创建、状态流转 | P0 |
| payment-service | 支付、退款、对账 | P0 |
| hotel-service | 酒店、房间、预订、核销 | P1 |
| mall-service | 商品、购物车、评价 | P1 |
| distribution-service | 分销、佣金、提现 | P1 |
| marketing-service | 优惠券、活动、会员套餐 | P2 |
| finance-service | 统计、结算、报表 | P2 |
| content-service | 文章、FAQ | P2 |
| notification-service | 短信、推送、站内信 | P2 |
| admin-service | 后台管理、配置、日志 | P1 |

**Rationale**:
- 按业务能力（Business Capability）划分，边界清晰
- 高频服务（设备、租借、支付）独立，便于扩展
- 低频服务（营销、内容）可合并部署

## 3. 数据库设计策略

### Decision: 单库多 Schema + 读写分离

**Rationale**:
- 初期使用单库，降低运维复杂度
- 使用 Schema 隔离不同业务域数据
- 主从复制实现读写分离，提升查询性能
- 预留分库分表能力（订单表按时间分区）

**Schema 划分**:
```sql
-- 用户域
CREATE SCHEMA IF NOT EXISTS users;

-- 设备域
CREATE SCHEMA IF NOT EXISTS devices;

-- 订单域
CREATE SCHEMA IF NOT EXISTS orders;

-- 商城域
CREATE SCHEMA IF NOT EXISTS mall;

-- 酒店域
CREATE SCHEMA IF NOT EXISTS hotel;

-- 财务域
CREATE SCHEMA IF NOT EXISTS finance;

-- 系统域
CREATE SCHEMA IF NOT EXISTS system;
```

**索引策略**:
- 高频查询字段建立索引
- 组合查询使用复合索引
- 避免过度索引影响写入性能
- 定期分析慢查询优化索引

## 4. 缓存策略

### Decision: 多级缓存 + Cache-Aside Pattern

**缓存层次**:
```
L1: 本地缓存 (go-cache) - 热点配置、字典数据
L2: Redis 缓存 - 会话、Token、业务数据
L3: 数据库
```

**缓存场景**:

| 场景 | 策略 | TTL |
|------|------|-----|
| JWT Token | Redis Hash | 7天 |
| 验证码 | Redis String | 5分钟 |
| 用户信息 | Redis Hash | 30分钟 |
| 设备状态 | Redis Hash | 实时更新 |
| 商品列表 | Redis String (JSON) | 10分钟 |
| 系统配置 | 本地缓存 + Redis | 1小时 |
| 分布式锁 | Redis SETNX | 业务相关 |

**缓存更新策略**:
- 写操作：先更新数据库，再删除缓存
- 读操作：先读缓存，未命中则读数据库并回填缓存
- 批量操作：使用 Pipeline 提升性能

## 5. MQTT 设备通信设计

### Decision: EMQX + 分层主题设计

**MQTT Broker 选型: EMQX**

**Rationale**:
- 国产开源，社区活跃，中文文档完善
- 支持百万级并发连接
- 内置规则引擎，支持消息路由到 Redis/PostgreSQL
- 支持 WebSocket，便于管理后台实时推送

**主题设计**:
```
# 设备上行（设备 → 服务器）
device/{device_id}/status          # 设备状态上报
device/{device_id}/event           # 设备事件（开锁、关锁、故障）
device/{device_id}/heartbeat       # 心跳

# 设备下行（服务器 → 设备）
device/{device_id}/command         # 控制命令（开锁、重启）
device/{device_id}/config          # 配置下发

# 广播
device/broadcast/config            # 全局配置广播
```

**QoS 策略**:
| 消息类型 | QoS | 原因 |
|----------|-----|------|
| 开锁命令 | 1 | 确保送达，至少一次 |
| 状态上报 | 1 | 确保送达 |
| 心跳 | 0 | 允许丢失 |
| 配置下发 | 2 | 精确一次 |

**设备认证**:
- 使用设备唯一 ID + 密钥进行认证
- 支持 TLS 加密传输
- 设备离线超过阈值（5分钟）标记为离线

## 6. 支付集成设计

### Decision: 统一支付抽象层 + 异步回调处理

**支付流程**:
```
1. 创建订单 → 生成预支付参数
2. 前端调用支付 → 支付平台
3. 支付回调 → 验签 → 更新订单状态 → 触发后续业务
4. 主动查询兜底 → 定时任务查询未确认订单
```

**接口抽象**:
```go
type PaymentProvider interface {
    // 创建支付
    CreatePayment(ctx context.Context, order *Order) (*PaymentResult, error)
    // 查询支付状态
    QueryPayment(ctx context.Context, orderNo string) (*PaymentStatus, error)
    // 申请退款
    Refund(ctx context.Context, refund *RefundRequest) (*RefundResult, error)
    // 验证回调签名
    VerifyCallback(ctx context.Context, data []byte) (*CallbackData, error)
}
```

**幂等性保证**:
- 使用订单号作为支付唯一标识
- 回调处理使用分布式锁防重
- 订单状态机控制状态流转

## 7. 认证授权设计

### Decision: JWT + RBAC + Redis Session

**JWT Token 结构**:
```json
{
  "sub": "user_id",
  "type": "user|admin",
  "role": "super_admin|platform_admin|...",
  "exp": 1704067200,
  "iat": 1703462400,
  "jti": "unique_token_id"
}
```

**Token 刷新机制**:
- Access Token: 2小时有效
- Refresh Token: 7天有效
- Token 存储到 Redis，支持主动失效

**RBAC 权限模型**:
```
用户 → 角色 → 权限
Admin → Role → Permission (API + Menu)
```

**权限检查流程**:
```
1. 请求到达 → 提取 JWT → 验证签名和有效期
2. 从 Redis 获取用户角色和权限
3. 检查当前 API 是否在权限列表中
4. 通过则继续，否则返回 403
```

## 8. 异步任务与事件驱动

### Decision: Redis Stream + 消费者组

**Rationale**:
- 相比 RabbitMQ 运维更简单
- Redis Stream 支持消费者组、消息确认、消息重试
- 与现有 Redis 基础设施复用

**任务队列设计**:

| Stream | 用途 | 消费者 |
|--------|------|--------|
| order:events | 订单事件 | 佣金计算、通知发送 |
| device:events | 设备事件 | 状态同步、告警处理 |
| payment:events | 支付事件 | 订单更新、分账处理 |
| notification:tasks | 通知任务 | 短信、推送发送 |
| rental:timeout | 租借超时 | 超时处理、押金扣除 |

**定时任务**:
- 使用 cron 表达式配置
- 支持分布式锁防止重复执行
- 关键任务：订单超时取消、租借超时处理、支付状态查询

## 9. 日志与监控

### Decision: 结构化日志 + Prometheus + Jaeger

**日志规范**:
```json
{
  "timestamp": "2026-01-02T10:00:00Z",
  "level": "info",
  "service": "device-service",
  "trace_id": "abc123",
  "span_id": "def456",
  "message": "Device unlock command sent",
  "device_id": "D001",
  "user_id": "U001",
  "latency_ms": 50
}
```

**监控指标**:
- 业务指标：订单量、支付成功率、设备在线率
- 性能指标：QPS、响应时间、错误率
- 资源指标：CPU、内存、连接数

**告警规则**:
| 指标 | 阈值 | 级别 |
|------|------|------|
| API 错误率 | > 1% | Warning |
| API 响应时间 P95 | > 500ms | Warning |
| 设备离线数 | > 10% | Critical |
| 支付失败率 | > 0.5% | Critical |

## 10. 安全设计

### Decision: 多层安全防护

**API 安全**:
- 所有接口强制 HTTPS
- JWT Token 认证
- 请求签名防篡改
- IP 白名单（管理后台）

**数据安全**:
- 敏感数据使用 AES-256-GCM 加密
- 密码使用 bcrypt 哈希
- 数据库连接使用 SSL
- 定期数据备份

**防刷策略**:
- 验证码接口：1次/分钟/手机号
- 登录接口：5次/分钟/IP
- 支付接口：Token 认证 + 幂等

**审计日志**:
- 记录所有管理员操作
- 记录敏感数据访问
- 日志保留至少 1 年

## 11. 部署架构

### Decision: Docker Compose (Dev) + Kubernetes (Prod)

**开发环境**:
```yaml
services:
  - postgres
  - redis
  - emqx
  - api-gateway
  - backend (all-in-one)
```

**生产环境**:
```
- Kubernetes 集群
- 服务多副本部署（至少 2 副本）
- PostgreSQL 主从复制
- Redis Sentinel 高可用
- EMQX 集群
- Nginx Ingress 负载均衡
```

**CI/CD 流程**:
```
代码提交 → 单元测试 → 静态分析 → 构建镜像 → 集成测试 → 部署 Staging → E2E 测试 → 部署 Production
```

## 12. 第三方服务集成

### Decision: 适配器模式 + 降级策略

**短信服务**:
- 主：阿里云短信
- 备：腾讯云短信
- 降级：失败自动切换备用通道

**对象存储**:
- 阿里云 OSS
- 图片上传限制：5MB
- 支持格式：jpg, png, webp
- 自动生成缩略图

**二维码生成**:
- 使用 go-qrcode 本地生成
- 设备二维码包含设备 ID
- 酒店核销码包含预订信息

## 13. 错误处理规范

### Decision: 统一错误码 + 结构化响应

**错误码设计**:
```
10xxx - 系统错误
20xxx - 认证授权错误
30xxx - 参数验证错误
40xxx - 业务逻辑错误
50xxx - 第三方服务错误
```

**响应格式**:
```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "request_id": "uuid"
}

{
  "code": 40001,
  "message": "设备不存在",
  "data": null,
  "request_id": "uuid"
}
```

## 14. 开发规范

### Decision: Go 标准规范 + 项目约定

**代码风格**:
- 使用 gofmt 格式化
- 使用 golangci-lint 静态检查
- 变量命名：camelCase
- 常量命名：全大写下划线
- 包命名：小写单词

**Git 规范**:
- 分支：feature/xxx, bugfix/xxx, hotfix/xxx
- 提交：feat: xxx, fix: xxx, docs: xxx
- PR 必须通过 CI 检查和 Code Review

**目录规范**:
- `cmd/` - 程序入口
- `internal/` - 内部实现
- `pkg/` - 可复用包
- `api/` - API 定义
- `configs/` - 配置文件
- `migrations/` - 数据库迁移
- `tests/` - 测试文件
