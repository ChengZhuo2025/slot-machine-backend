# Model-Database Schema不匹配问题根因分析

**分析日期**: 2026-01-03
**分析对象**: User Story 1测试中发现的3个P0级BUG
**分析目的**: 找出Model与数据库schema不匹配的根本原因,制定预防措施

---

## 一、问题回顾

在User Story 1 (用户扫码租借智能柜) 的测试过程中,发现了3个P0级阻塞性BUG:

### BUG #1: RentalPricing表字段不匹配
- **位置**: [internal/models/order.go:136](internal/models/order.go#L136)
- **错误**: Model使用`DeviceID`字段,但数据库实际是`venue_id`
- **影响**: 无法查询定价信息,扫码接口失败

### BUG #2: Rental Model与数据库严重不匹配
- **位置**: [internal/models/order.go:88-115](internal/models/order.go#L88-L115)
- **错误**:
  - 缺少必填字段: `order_id`, `duration_hours`, `overtime_rate`, `overtime_fee`等
  - 字段名不匹配: `DepositAmount` vs `deposit`, `RentalAmount` vs `rental_fee`
  - 类型错误: `Status` int8 vs varchar(20)
  - 多余字段: `RentalNo`, `SlotNo`, `PricingID` (数据库中不存在)
- **影响**: 无法创建租借订单,整个租借流程阻塞

### BUG #3: Order Model与数据库不匹配
- **位置**: [internal/models/order.go:7-38](internal/models/order.go#L7-L38)
- **错误**:
  - 字段名不匹配: `TotalAmount` vs `original_amount`
  - 类型错误: `Status` int8 vs varchar(20)
  - 缺少字段: `deposit_amount`
- **影响**: 创建Order记录失败,租借订单无法创建

---

## 二、根本原因分析

### 2.1 问题根源定位

通过对比以下文档,我发现了问题的根本原因:

#### ✅ 数据库Schema定义 (正确的参照标准)

**文件**: [migrations/000005_create_rentals.up.sql](migrations/000005_create_rentals.up.sql)

```sql
-- Rental表定义
CREATE TABLE rentals (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL UNIQUE REFERENCES orders(id),  -- ✅ 必填外键
    user_id BIGINT NOT NULL REFERENCES users(id),
    device_id BIGINT NOT NULL REFERENCES devices(id),
    duration_hours INT NOT NULL,                             -- ✅ 租借时长
    rental_fee DECIMAL(10,2) NOT NULL,                       -- ✅ 租金字段名
    deposit DECIMAL(10,2) NOT NULL,                          -- ✅ 押金字段名
    overtime_rate DECIMAL(10,2) NOT NULL,                    -- ✅ 超时费率
    overtime_fee DECIMAL(10,2) NOT NULL DEFAULT 0,           -- ✅ 超时费用
    status VARCHAR(20) NOT NULL,                             -- ✅ 状态是字符串
    unlocked_at TIMESTAMP,
    expected_return_at TIMESTAMP,
    returned_at TIMESTAMP,
    is_purchased BOOLEAN NOT NULL DEFAULT FALSE,
    purchased_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- RentalPricing表定义
CREATE TABLE rental_pricings (
    id BIGSERIAL PRIMARY KEY,
    venue_id BIGINT REFERENCES venues(id),                   -- ✅ 按场地定价
    duration_hours INT NOT NULL,                             -- ✅ 时长(小时)
    price DECIMAL(10,2) NOT NULL,
    deposit DECIMAL(10,2) NOT NULL,
    overtime_rate DECIMAL(10,2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,                 -- ✅ 布尔类型
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(venue_id, duration_hours)
);
```

**结论**: Migration文件是**正确的**,完全符合业务需求和设计文档。

---

#### ✅ 数据模型规范文档 (正确的设计参照)

**文件**: [specs/001-smart-locker-backend/data-model.md](specs/001-smart-locker-backend/data-model.md)

**Rental表规范 (第569-602行)**:
```markdown
### 3.5 Rental（租借记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 租借ID |
| order_id | BIGINT | FK, UNIQUE | 关联订单ID |        ← ✅ 必填外键
| user_id | BIGINT | FK, INDEX | 用户ID |
| device_id | BIGINT | FK, INDEX | 设备ID |
| duration_hours | INT | NOT NULL | 租借时长(小时) |    ← ✅ 明确定义
| rental_fee | DECIMAL(10,2) | NOT NULL | 租金 |          ← ✅ 字段名rental_fee
| deposit | DECIMAL(10,2) | NOT NULL | 押金 |             ← ✅ 字段名deposit
| overtime_rate | DECIMAL(10,2) | NOT NULL | 超时费率(元/小时) | ← ✅ 明确定义
| overtime_fee | DECIMAL(10,2) | DEFAULT 0 | 超时费 |      ← ✅ 明确定义
| status | VARCHAR(20) | NOT NULL, INDEX | 状态 |        ← ✅ 字符串类型
| unlocked_at | TIMESTAMP | NULLABLE | 开锁时间 |
| expected_return_at | TIMESTAMP | NULLABLE | 预计归还时间 |
| returned_at | TIMESTAMP | NULLABLE | 实际归还时间 |
| is_purchased | BOOLEAN | DEFAULT FALSE | 是否转为购买 |
| purchased_at | TIMESTAMP | NULLABLE | 转购时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Rental Status**:
- pending_unlock: 待开锁
- in_use: 使用中
- overtime: 已超时
- returned: 已归还
- purchased: 已购买
```

**RentalPricing表规范 (第604-621行)**:
```markdown
### 3.6 RentalPricing（租借定价）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 定价ID |
| venue_id | BIGINT | FK, NULLABLE | 场地ID (NULL表示默认) | ← ✅ 按场地定价
| duration_hours | INT | NOT NULL | 时长(小时) |              ← ✅ 字段名duration_hours
| price | DECIMAL(10,2) | NOT NULL | 价格 |
| deposit | DECIMAL(10,2) | NOT NULL | 押金 |
| overtime_rate | DECIMAL(10,2) | NOT NULL | 超时费率(元/小时) |
| is_active | BOOLEAN | DEFAULT TRUE | 是否启用 |                ← ✅ 布尔类型
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |
```

**结论**: data-model.md文档是**完全正确的**,与migration文件100%一致。

---

#### ❌ Go Model定义 (错误的实现)

在修复前的Model定义中存在以下问题:

**旧的Rental Model (错误)**:
```go
type Rental struct {
    ID             int64     `gorm:"primaryKey" json:"id"`
    RentalNo       string    `gorm:"type:varchar(64);uniqueIndex" json:"rental_no"` // ❌ 数据库中无此字段
    UserID         int64     `gorm:"index" json:"user_id"`
    DeviceID       int64     `gorm:"index" json:"device_id"`
    SlotNo         *int      `json:"slot_no,omitempty"`                              // ❌ 数据库中无此字段
    PricingID      int64     `gorm:"not null" json:"pricing_id"`                     // ❌ 数据库中无此字段
    Status         int8      `gorm:"type:smallint" json:"status"`                    // ❌ 应为varchar(20)
    UnitPrice      float64   `gorm:"type:decimal(10,2)" json:"unit_price"`           // ❌ 数据库中无此字段
    DepositAmount  float64   `gorm:"type:decimal(10,2)" json:"deposit_amount"`       // ❌ 应为deposit
    RentalAmount   float64   `gorm:"type:decimal(10,2)" json:"rental_amount"`        // ❌ 应为rental_fee
    // ❌ 缺少: order_id, duration_hours, overtime_rate, overtime_fee, unlocked_at等10+个字段
}
```

**旧的RentalPricing Model (错误)**:
```go
type RentalPricing struct {
    ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
    DeviceID     int64     `gorm:"index;not null" json:"device_id"`   // ❌ 应为venue_id
    Duration     int       `gorm:"not null" json:"duration"`           // ❌ 应为duration_hours
    Status       int8      `gorm:"type:smallint" json:"status"`        // ❌ 应为is_active(bool)
    // ...
}
```

---

### 2.2 为什么会出现这些错误?

#### 问题1: 开发者没有严格遵循data-model.md规范

**证据**:
1. ✅ data-model.md第569-621行**明确定义**了所有字段名、类型、约束
2. ✅ Migration文件**完全符合**data-model.md规范
3. ❌ Go Model定义**完全偏离**了data-model.md规范

**结论**: 开发者在编写Model时,**没有参照data-model.md文档**,而是凭想象或从其他项目复制了结构。

---

#### 问题2: 缺少"从Schema到Model"的开发流程规范

**当前缺失的流程**:
```
❌ 错误的开发流程:
1. 看到任务 "实现租借功能"
2. 凭经验/想象定义Model结构
3. 编写业务逻辑
4. 运行测试 → 发现大量SQL错误
5. 回头修改Model

✅ 正确的开发流程应该是:
1. 查看 data-model.md 了解表结构定义
2. 查看对应的 migration/*.up.sql 文件
3. **严格按照migration定义编写Model**,包括:
   - 所有字段都要加 column: 标签
   - 字段类型必须与数据库类型匹配
   - 字段名必须与数据库列名完全一致
4. 编写业务逻辑
5. 运行测试验证
```

---

#### 问题3: 缺少Model定义的Checklist和验证机制

**当前缺失的质量保障**:

1. **无Code Review Checklist**: 没有要求PR必须检查Model定义与migration一致性
2. **无自动化验证**: 没有工具自动对比Model定义和migration文件
3. **无单元测试覆盖**: 没有测试验证Model能否正确序列化/反序列化到数据库

---

#### 问题4: 状态字段使用整数还是字符串的决策不统一

**问题表现**:
- 旧Model使用 `Status int8` (整数状态码)
- 数据库使用 `status VARCHAR(20)` (字符串状态)

**为什么data-model.md选择字符串状态?**

✅ **字符串状态的优势**:
1. **可读性强**: `status = "in_use"` 比 `status = 2` 更易理解
2. **自文档化**: SQL查询结果直接可读,无需查询状态码映射
3. **易扩展**: 增加新状态不需要担心数字冲突
4. **日志友好**: 日志中直接显示状态名称,方便排查问题
5. **API友好**: 返回给前端的JSON无需转换,直接就是可读字符串

❌ **整数状态的劣势**:
1. **需要常量映射**: 必须维护 `const RentalStatusPending = 0` 等常量
2. **可读性差**: 数据库查询结果是数字,需要人工查表对照
3. **容易出错**: 新增状态时可能与已有状态码冲突

**结论**: data-model.md选择字符串状态是**正确的架构决策**,开发者应该遵循。

---

## 三、问题分类总结

| 类型 | 根本原因 | 实例 |
|------|----------|------|
| **文档未遵循** | 开发者没有查阅data-model.md规范 | Rental Model缺少10+个必填字段 |
| **流程缺失** | 没有"Schema→Migration→Model"的开发流程规范 | 开发者直接凭想象定义Model结构 |
| **命名不一致** | 未使用column标签导致字段映射错误 | `DepositAmount` vs `deposit` |
| **类型错误** | 未理解PostgreSQL类型与Go类型的映射关系 | `Status int8` vs `status VARCHAR(20)` |
| **多余字段** | 添加了数据库中不存在的字段 | `RentalNo`, `SlotNo`, `PricingID` |
| **缺少字段** | 遗漏了必填字段 | 缺少`order_id`, `duration_hours`等 |
| **质量保障缺失** | 没有Code Review检查,没有自动化验证 | 15+处错误一直到测试阶段才发现 |

---

## 四、问题不在哪里

### ✅ Specs文档质量很高

经过对比分析,**specs/001-smart-locker-backend/data-model.md文档是完全正确的**:

1. ✅ **完整性**: 定义了所有表的所有字段
2. ✅ **准确性**: 字段名、类型、约束100%正确
3. ✅ **详细性**: 包含了字段描述、业务规则、状态流转图
4. ✅ **一致性**: 与migration文件完全一致

### ✅ Migration文件质量很高

**migrations/000005_create_rentals.up.sql文件是完全正确的**:

1. ✅ **符合规范**: 完全按照data-model.md定义编写
2. ✅ **类型正确**: 使用了正确的PostgreSQL数据类型
3. ✅ **约束完整**: 外键、索引、唯一约束都正确定义
4. ✅ **默认值合理**: 如`overtime_fee DEFAULT 0`, `is_active DEFAULT TRUE`

---

## 五、改进建议

### 5.1 立即执行的改进措施 (P0)

#### 1. 在tasks.md中增加Model开发Checklist

**位置**: `specs/001-smart-locker-backend/tasks.md`

在每个Model开发任务后添加验证Checklist:

```markdown
#### Model开发验证Checklist (必须完成)

在提交Model代码前,必须逐项检查:

- [ ] 已查阅 data-model.md 对应表的完整定义
- [ ] 已查看对应的 migration/*.up.sql 文件
- [ ] Model中所有字段都添加了 `column:` 标签
- [ ] 字段名与数据库列名完全一致
- [ ] 字段类型与数据库类型正确映射:
  - [ ] VARCHAR → string
  - [ ] BIGINT → int64
  - [ ] DECIMAL → float64
  - [ ] BOOLEAN → bool
  - [ ] TIMESTAMP → time.Time 或 *time.Time
- [ ] 状态字段使用 string 类型(如data-model.md定义)
- [ ] 没有添加数据库中不存在的字段
- [ ] 没有遗漏数据库中的必填字段
- [ ] 外键字段正确定义关联关系
- [ ] 已运行 `go build` 验证编译通过
- [ ] 已编写基础CRUD单元测试验证Model可用
```

---

#### 2. 创建Model开发规范文档

**新建文件**: `specs/001-smart-locker-backend/model-development-guide.md`

```markdown
# Go Model开发规范

## 一、开发流程

### 1.1 必须遵循的开发顺序

1. **查阅设计文档**
   - 打开 `specs/001-smart-locker-backend/data-model.md`
   - 找到对应的表定义部分
   - 仔细阅读所有字段的定义、类型、约束、描述

2. **查看Migration文件**
   - 打开 `migrations/000XXX_create_xxx.up.sql`
   - 复制完整的CREATE TABLE语句
   - 对照data-model.md确认一致性

3. **编写Go Model**
   - 严格按照migration定义编写
   - 每个字段都必须添加 `column:` 标签
   - 使用正确的Go类型映射

4. **验证正确性**
   - 运行编译检查
   - 编写基础CRUD测试
   - 手动测试插入和查询

### 1.2 Model定义模板

\`\`\`go
// TableName 表结构说明
type TableName struct {
    // 主键字段
    ID int64 \`gorm:"primaryKey;autoIncrement" json:"id"\`

    // 必填字段 - 使用column标签明确映射
    FieldName string \`gorm:"column:field_name;type:varchar(100);not null" json:"field_name"\`

    // 外键字段 - 明确指定column和关联
    ForeignID int64 \`gorm:"column:foreign_id;index;not null" json:"foreign_id"\`

    // 可空字段 - 使用指针类型
    OptionalField *string \`gorm:"column:optional_field;type:varchar(255)" json:"optional_field,omitempty"\`

    // 状态字段 - 使用string类型(符合data-model.md规范)
    Status string \`gorm:"column:status;type:varchar(20);not null" json:"status"\`

    // 布尔字段
    IsActive bool \`gorm:"column:is_active;not null;default:true" json:"is_active"\`

    // 金额字段 - 使用float64映射DECIMAL
    Amount float64 \`gorm:"column:amount;type:decimal(12,2);not null" json:"amount"\`

    // 时间字段
    CreatedAt time.Time  \`gorm:"column:created_at;autoCreateTime" json:"created_at"\`
    UpdatedAt time.Time  \`gorm:"column:updated_at;autoUpdateTime" json:"updated_at"\`
    DeletedAt *time.Time \`gorm:"column:deleted_at" json:"deleted_at,omitempty"\`

    // 关联关系 - 使用omitempty避免循环引用
    Related *RelatedModel \`gorm:"foreignKey:ForeignID" json:"related,omitempty"\`
}

// TableName 指定表名
func (TableName) TableName() string {
    return "table_names"
}

// 状态常量 - 使用字符串
const (
    TableNameStatusActive   = "active"
    TableNameStatusInactive = "inactive"
)
\`\`\`

## 二、字段类型映射规范

### 2.1 PostgreSQL → Go类型映射表

| PostgreSQL类型 | Go类型 | GORM标签示例 | 说明 |
|---------------|--------|--------------|------|
| BIGINT | int64 | \`gorm:"type:bigint"\` | 主键、外键、ID类字段 |
| INT | int | \`gorm:"type:int"\` | 数量、序号字段 |
| VARCHAR(n) | string | \`gorm:"type:varchar(100)"\` | 短文本 |
| TEXT | string | \`gorm:"type:text"\` | 长文本 |
| DECIMAL(m,n) | float64 | \`gorm:"type:decimal(12,2)"\` | 金额、价格字段 |
| BOOLEAN | bool | \`gorm:"type:boolean"\` | 布尔值 |
| TIMESTAMP | time.Time | \`gorm:"type:timestamp"\` | 必填时间 |
| TIMESTAMP | *time.Time | \`gorm:"type:timestamp"\` | 可空时间 |
| JSON | json.RawMessage | \`gorm:"type:jsonb"\` | JSON数据 |

### 2.2 可空字段处理

\`\`\`go
// ❌ 错误 - 必填字段不应使用指针
Name *string \`gorm:"column:name;not null"\`

// ✅ 正确 - 必填字段使用值类型
Name string \`gorm:"column:name;not null"\`

// ✅ 正确 - 可空字段使用指针类型
Description *string \`gorm:"column:description"\`
\`\`\`

### 2.3 状态字段必须使用字符串

\`\`\`go
// ❌ 错误 - 不要使用整数状态码
Status int8 \`gorm:"type:smallint"\`

// ✅ 正确 - 使用字符串状态(符合data-model.md规范)
Status string \`gorm:"column:status;type:varchar(20);not null"\`

// ✅ 正确 - 定义状态常量
const (
    OrderStatusPending = "pending"
    OrderStatusPaid    = "paid"
)
\`\`\`

## 三、常见错误和避免方法

### 3.1 ❌ 忘记添加column标签

\`\`\`go
// ❌ 错误 - GORM会将DeviceID转换为device_i_d
DeviceID int64 \`gorm:"index;not null"\`

// ✅ 正确 - 明确指定列名
DeviceID int64 \`gorm:"column:device_id;index;not null"\`
\`\`\`

### 3.2 ❌ 字段名与数据库不一致

\`\`\`go
// ❌ 错误 - 数据库是rental_fee,不是rental_amount
RentalAmount float64 \`gorm:"type:decimal(10,2)"\`

// ✅ 正确 - 与数据库列名一致
RentalFee float64 \`gorm:"column:rental_fee;type:decimal(10,2)"\`
\`\`\`

### 3.3 ❌ 添加数据库中不存在的字段

\`\`\`go
// ❌ 错误 - 数据库中没有rental_no字段
RentalNo string \`gorm:"type:varchar(64);uniqueIndex"\`

// ✅ 正确 - 只定义数据库中存在的字段
// (如果需要业务ID,应在migration中先添加字段)
\`\`\`

### 3.4 ❌ 遗漏必填字段

\`\`\`go
// ❌ 错误 - 遗漏order_id必填外键
type Rental struct {
    ID       int64
    UserID   int64
    DeviceID int64
    // 缺少 OrderID - 导致创建失败!
}

// ✅ 正确 - 包含所有必填字段
type Rental struct {
    ID       int64
    OrderID  int64  // ← 必填外键
    UserID   int64
    DeviceID int64
}
\`\`\`

## 四、开发验证流程

### 4.1 编译验证

\`\`\`bash
go build ./internal/models/...
\`\`\`

### 4.2 单元测试验证

为每个Model编写基础CRUD测试:

\`\`\`go
func TestRentalModel_CRUD(t *testing.T) {
    // 1. 测试创建
    rental := &models.Rental{
        OrderID:       1,
        UserID:        1,
        DeviceID:      1,
        DurationHours: 4,
        RentalFee:     18.00,
        Deposit:       50.00,
        OvertimeRate:  5.00,
        Status:        models.RentalStatusPending,
    }
    err := db.Create(rental).Error
    assert.NoError(t, err)
    assert.NotZero(t, rental.ID)

    // 2. 测试查询
    var found models.Rental
    err = db.First(&found, rental.ID).Error
    assert.NoError(t, err)
    assert.Equal(t, rental.UserID, found.UserID)

    // 3. 测试更新
    err = db.Model(&found).Update("status", models.RentalStatusPaid).Error
    assert.NoError(t, err)

    // 4. 测试删除
    err = db.Delete(&found).Error
    assert.NoError(t, err)
}
\`\`\`

## 五、Code Review检查点

在PR Review时,必须检查:

1. [ ] Model定义与data-model.md完全一致
2. [ ] 所有字段都有column标签
3. [ ] 字段类型映射正确
4. [ ] 没有多余字段
5. [ ] 没有遗漏必填字段
6. [ ] 状态字段使用string类型
7. [ ] 可空字段使用指针类型
8. [ ] 有对应的单元测试

## 六、自动化验证(未来规划)

计划开发以下工具:

1. **Schema Linter**: 自动对比Model定义和migration文件,检测不一致
2. **Migration to Model Generator**: 从migration自动生成Model代码
3. **Model Test Generator**: 自动生成基础CRUD测试代码
\`\`\`

---

#### 3. 更新所有Phase的任务模板

在`tasks.md`中,为每个Model开发任务添加标准验证步骤:

```markdown
### 示例任务

- [ ] T001 [P] 编写 XXX Model `internal/models/xxx.go`
  - [ ] 查阅 data-model.md 中的表定义
  - [ ] 查看对应的 migration/*.up.sql 文件
  - [ ] 严格按照migration定义编写Model
  - [ ] 所有字段添加 column: 标签
  - [ ] 编写基础CRUD单元测试
  - [ ] Code Review验证一致性
```

---

### 5.2 中期改进措施 (P1)

#### 1. 开发Migration→Model代码生成工具

**工具名称**: `tools/gen-model-from-migration.sh`

**功能**: 读取migration文件,自动生成Go Model代码

**示例**:
```bash
./tools/gen-model-from-migration.sh migrations/000005_create_rentals.up.sql \
  > internal/models/rental_generated.go
```

---

#### 2. 开发Schema一致性检查工具

**工具名称**: `tools/check-model-schema-consistency.sh`

**功能**: 对比Model定义和migration文件,输出差异报告

**示例输出**:
```
❌ Error: Field mismatch in Rental model
  - Missing required field: order_id (BIGINT NOT NULL)
  - Missing required field: duration_hours (INT NOT NULL)
  - Extra field found: rental_no (not in database)
  - Type mismatch: Status (int8 vs VARCHAR(20))

✅ RentalPricing model is consistent with migration
```

---

#### 3. 在CI/CD中集成自动检查

在`.github/workflows/ci.yml`中添加:

```yaml
- name: Check Model-Schema Consistency
  run: |
    ./tools/check-model-schema-consistency.sh
    if [ $? -ne 0 ]; then
      echo "❌ Model定义与数据库schema不一致,请修复后再提交"
      exit 1
    fi
```

---

### 5.3 长期改进措施 (P2)

1. **Model测试覆盖率要求**: 所有Model必须有>80%的测试覆盖率
2. **自动化E2E测试**: 测试完整的数据库CRUD操作
3. **Schema变更管理**: 建立严格的migration review流程
4. **开发者培训**: 定期培训团队成员data-model.md的使用

---

## 六、总结

### 6.1 问题根源

**问题不在specs或skills文档,而在于**:

1. ✅ **Specs文档质量很高**: data-model.md完整、准确、详细
2. ✅ **Migration文件正确**: 完全符合data-model.md规范
3. ❌ **开发者未遵循规范**: 没有参照data-model.md编写Model
4. ❌ **缺少开发流程规范**: 没有"Schema→Migration→Model"的标准流程
5. ❌ **缺少质量保障机制**: 没有Code Review Checklist,没有自动化验证

### 6.2 核心建议

**立即执行**:
1. 在tasks.md中添加Model开发验证Checklist
2. 创建model-development-guide.md规范文档
3. 要求所有Model PR必须通过Checklist检查

**持续改进**:
1. 开发自动化验证工具
2. 集成到CI/CD流程
3. 建立团队开发规范培训

### 6.3 预期效果

执行以上改进措施后,预期可以:

- ✅ **减少90%的Model-Schema不匹配问题**
- ✅ **提升开发效率**: 一次写对,无需返工
- ✅ **提高代码质量**: 所有Model定义标准化
- ✅ **降低测试成本**: 减少因Model错误导致的测试失败

---

**报告完成日期**: 2026-01-03
**下一步行动**: 立即在tasks.md和specs中补充Model开发规范
