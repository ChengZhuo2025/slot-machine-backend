# 智能开锁管理系统 - 开发技能调度器

## 概述

本技能作为开发过程中的总调度器，根据当前任务自动选择合适的专项技能。

## 项目背景

- **项目名称**: 爱上杜美人智能开锁管理系统
- **技术栈**: Go 1.25+ / Gin / GORM / PostgreSQL / Redis / MQTT
- **规范文档**: `backend/specs/001-smart-locker-backend/`

## 技能清单

| 技能 | 文件 | 用途 |
|------|------|------|
| Go 后端开发 | `go-backend-dev.md` | 生成 Go 业务代码 |
| 数据库迁移 | `db-migration.md` | 创建数据库迁移文件 |
| API Handler | `api-handler.md` | 生成 API 处理器代码 |
| 测试开发 | `go-testing.md` | 编写单元/集成测试 |
| 种子数据 | `seed-data.md` | 生成测试数据 SQL |

## 技能调度规则

### Phase 1: Setup（项目初始化）
```
任务 T001-T008 → 无需特定技能，按 plan.md 执行
```

### Phase 2: Foundational（基础设施）

| 任务范围 | 调度技能 |
|----------|----------|
| T009-T012 配置/数据库/缓存/日志 | `go-backend-dev` |
| T013-T029 数据库迁移 | `db-migration` |
| T030 迁移脚本 | `go-backend-dev` |
| T031-T039 种子数据 | `seed-data` |
| T040-T045 模型定义 | `go-backend-dev` |
| T046-T050 公共组件 | `go-backend-dev` |
| T051-T056 中间件 | `go-backend-dev` |
| T057-T059 API Gateway | `go-backend-dev` + `api-handler` |

### Phase 3-10: User Stories（业务开发）

| 代码类型 | 调度技能 |
|----------|----------|
| `internal/models/*.go` | `go-backend-dev` |
| `internal/repository/*.go` | `go-backend-dev` |
| `internal/service/**/*.go` | `go-backend-dev` |
| `internal/handler/**/*.go` | `api-handler` |
| `pkg/**/*.go` | `go-backend-dev` |

### Phase 11: Polish（完善优化）
```
同 Phase 3-10 规则
```

### Phase 12: Testing（测试）

| 任务范围 | 调度技能 |
|----------|----------|
| T240-T242 测试基础设施 | `go-testing` |
| T243-T254 单元测试 | `go-testing` |
| T255-T260 集成测试 | `go-testing` |
| T261-T263 E2E 测试 | `go-testing` |
| T264-T267 API 测试 | `go-testing` |

## 快速判断流程

```
开始任务
    │
    ├─ 是否创建迁移文件？ ─Yes→ 使用 db-migration
    │
    ├─ 是否生成种子数据？ ─Yes→ 使用 seed-data
    │
    ├─ 是否编写测试代码？ ─Yes→ 使用 go-testing
    │
    ├─ 是否编写 Handler？ ─Yes→ 使用 api-handler
    │
    └─ 其他 Go 代码 ─────────→ 使用 go-backend-dev
```

## 多技能组合场景

### 场景1: 实现一个完整的 API 端点
```
1. go-backend-dev → 定义 Model
2. db-migration   → 创建表迁移（如需要）
3. go-backend-dev → 实现 Repository
4. go-backend-dev → 实现 Service
5. api-handler    → 实现 Handler
6. go-testing     → 编写测试
```

### 场景2: 新增一个数据表
```
1. db-migration   → 创建迁移文件
2. seed-data      → 添加测试数据
3. go-backend-dev → 定义 Model
4. go-backend-dev → 实现 Repository
```

### 场景3: 修复一个 Bug
```
1. go-testing     → 先写失败的测试用例
2. go-backend-dev → 修复代码
3. go-testing     → 验证测试通过
```

## 使用示例

当用户说：
- "帮我创建 User 表的迁移" → 调用 `db-migration`
- "实现用户登录接口" → 调用 `api-handler`
- "写一个用户服务" → 调用 `go-backend-dev`
- "帮我写单元测试" → 调用 `go-testing`
- "生成商品的测试数据" → 调用 `seed-data`

## 参考文档

在使用任何技能前，请确保已阅读：
- `backend/specs/001-smart-locker-backend/plan.md` - 技术架构
- `backend/specs/001-smart-locker-backend/data-model.md` - 数据模型
- `backend/specs/001-smart-locker-backend/contracts/user-api.yaml` - API 规范
- `backend/specs/001-smart-locker-backend/tasks.md` - 任务清单
