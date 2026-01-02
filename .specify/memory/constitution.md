<!--
## Sync Impact Report
- Version change: 0.0.0 → 1.0.0
- Modified principles: N/A (initial creation)
- Added sections:
  - Core Principles (7 principles)
  - Quality Attributes (性能、可用性、安全、可扩展)
  - Compliance Requirements (个人信息保护法、PCI DSS)
  - Governance
- Removed sections: None
- Templates requiring updates:
  - ✅ .specify/templates/plan-template.md (Constitution Check section compatible)
  - ✅ .specify/templates/spec-template.md (Requirements section compatible)
  - ✅ .specify/templates/tasks-template.md (Phase structure compatible)
- Follow-up TODOs: None
-->

# 杜美人智能开锁管理系统后端 Constitution

## Core Principles

### I. 需求一致性原则

后端系统 MUST 完整支持前端已演示的所有功能。

**非协商条款**:
- 后端 API MUST 覆盖前端界面中展示的每一个用户交互功能
- 任何前端功能在后端没有对应 API 支持前，该功能 MUST NOT 被视为完成
- 前后端接口契约 MUST 在开发前明确定义并文档化
- 接口变更 MUST 同步通知前端团队并获得确认

**理由**: 确保系统交付的完整性，避免前端展示功能而后端无法支持的情况。

### II. 安全优先原则

所有涉及用户数据、认证授权和系统操作的功能 MUST 将安全性作为首要考量。

**非协商条款**:
- 敏感数据（密码、密钥、个人信息）MUST 使用行业标准加密算法存储和传输
- 所有 API 端点 MUST 实施认证和授权检查
- 关键操作（开锁、用户管理、配置变更）MUST 记录完整审计日志
- 审计日志 MUST 包含：操作时间、操作人、操作类型、操作对象、操作结果
- 安全漏洞 MUST 在发现后 24 小时内启动修复流程

**理由**: 智能锁系统涉及物理安全，安全漏洞可能导致财产损失或人身安全风险。

### III. 微服务边界清晰原则

每个微服务 MUST 具有明确的职责边界和独立的生命周期。

**非协商条款**:
- 每个服务 MUST 遵循单一职责原则，只处理一个业务领域
- 服务间通信 MUST 通过定义良好的 API 契约进行
- 服务 MUST 能够独立部署、扩展和回滚
- 共享数据库 MUST NOT 在服务间直接访问，MUST 通过 API 获取
- 服务边界划分 MUST 在设计阶段明确并文档化

**理由**: 清晰的服务边界支持团队独立开发、降低耦合度、提高系统可维护性。

### IV. API 质量标准原则

所有对外 API MUST 满足性能和可靠性标准。

**非协商条款**:
- API 响应时间 MUST < 200ms（P95）
- 所有 API MUST 返回结构化错误响应，包含：错误码、错误消息、请求追踪 ID
- API MUST 实现幂等性设计（适用于 POST/PUT/DELETE 操作）
- API 文档 MUST 使用 OpenAPI 3.0+ 规范
- API 版本管理 MUST 遵循语义化版本控制
- 破坏性变更 MUST 通过新版本发布，旧版本 MUST 保持至少 6 个月兼容期

**理由**: 高质量 API 是系统可靠性和开发者体验的基础。

### V. 可测试性原则

代码 MUST 设计为易于测试，测试覆盖率 MUST 达到规定标准。

**非协商条款**:
- 单元测试覆盖率 MUST > 80%
- 关键业务逻辑（开锁、支付、权限）单元测试覆盖率 MUST > 90%
- 所有 API 端点 MUST 有对应的集成测试
- 测试 MUST 在 CI/CD 流水线中自动执行
- 测试失败 MUST 阻断代码合并和部署
- 测试数据 MUST 与生产数据隔离

**理由**: 高测试覆盖率是代码质量和系统稳定性的保障。

### VI. 可观测性原则

系统运行状态 MUST 可被完整监控和追踪。

**非协商条款**:
- 所有服务 MUST 输出结构化日志（JSON 格式）
- 日志 MUST 包含：时间戳、服务名、请求追踪 ID、日志级别、消息
- 关键业务指标 MUST 暴露 Prometheus 兼容的监控端点
- 分布式追踪 MUST 覆盖所有服务间调用
- 告警规则 MUST 覆盖：服务可用性、响应时间、错误率、资源使用率
- 审计追踪 MUST 保留至少 1 年

**理由**: 可观测性是快速定位问题、保障系统稳定运行的基础。

### VII. 代码质量门禁原则

代码变更 MUST 通过质量检查后方可合并。

**非协商条款**:
- 所有代码 MUST 通过静态分析工具检查（无严重/高危告警）
- 代码变更 MUST 经过至少一位团队成员 Code Review
- CI 流水线检查 MUST 包含：编译、测试、静态分析、安全扫描
- CI 检查失败 MUST 阻断代码合并
- 代码风格 MUST 符合团队统一规范（使用自动格式化工具）
- 安全扫描发现的高危漏洞 MUST 在合并前修复

**理由**: 质量门禁是保障代码质量、减少生产问题的最后防线。

## Quality Attributes

系统 MUST 满足以下质量属性要求：

### 性能

- API 响应时间 MUST < 200ms（P95）
- 系统 MUST 支持 10,000 并发用户
- 数据库查询 MUST < 50ms（P95）
- 后台任务处理延迟 MUST < 5 秒

### 可用性

- 系统可用性 MUST ≥ 99.9%（每月停机时间 < 43 分钟）
- 计划维护 MUST 提前 72 小时通知
- 故障恢复时间（RTO）MUST < 15 分钟
- 数据恢复点（RPO）MUST < 5 分钟

### 安全

- 严重安全漏洞数量 MUST = 0
- 高危安全漏洞 MUST 在 7 天内修复
- 渗透测试 MUST 每季度执行一次
- 安全审计 MUST 每年执行一次

### 可扩展

- 系统 MUST 支持水平扩展
- 系统 MUST 无单点故障（SPOF）
- 新服务实例 MUST 能在 60 秒内启动并接收流量
- 扩缩容操作 MUST 不影响现有请求

## Compliance Requirements

系统 MUST 遵守以下法规和标准：

### 个人信息保护法（PIPL）

- 用户个人信息收集 MUST 获得明确同意
- 用户 MUST 能够查看、导出、删除个人数据
- 个人信息存储 MUST 加密
- 跨境数据传输 MUST 符合法规要求
- 数据泄露 MUST 在 72 小时内上报

### PCI DSS 支付安全标准

- 支付卡数据 MUST 使用强加密存储
- 支付系统 MUST 实施访问控制
- 支付相关操作 MUST 记录完整审计日志
- 网络 MUST 实施分段隔离
- 安全评估 MUST 每年执行

## Governance

### 宪章优先级

本宪章是项目开发的最高指导文件，所有开发实践、技术决策和代码审查 MUST 以本宪章为准。

### 修订程序

1. 任何团队成员可以提出修订建议
2. 修订建议 MUST 包含：变更内容、变更理由、影响评估
3. 修订 MUST 经过技术负责人审批
4. 重大变更（原则增删、质量标准调整）MUST 经过项目经理批准
5. 修订生效后 MUST 更新版本号和修订日期

### 版本控制策略

- MAJOR: 原则删除或根本性重定义
- MINOR: 新原则添加或现有原则重大扩展
- PATCH: 措辞澄清、格式调整、非实质性修改

### 合规审查

- 每个 Sprint 结束时 MUST 进行宪章合规自查
- 代码审查 MUST 包含宪章合规检查项
- 违规情况 MUST 记录并制定改进计划

**Version**: 1.0.0 | **Ratified**: 2026-01-02 | **Last Amended**: 2026-01-02
