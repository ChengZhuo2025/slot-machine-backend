# Go Model开发规范

**适用项目**: 爱上杜美人智能开锁管理系统后端
**版本**: v1.0
**日期**: 2026-01-03

---

## 一、开发流程 (必须遵循)

### 1.1 标准开发顺序

```
┌─────────────────────────────────────────────────────────────┐
│ Step 1: 查阅设计文档                                          │
│ 打开 specs/001-smart-locker-backend/data-model.md           │
│ 找到对应表的完整定义                                          │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: 查看Migration文件                                    │
│ 打开 migrations/000XXX_create_xxx.up.sql                    │
│ 复制完整的 CREATE TABLE 语句                                 │
│ 对照 data-model.md 确认一致性                                │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: 编写Go Model                                         │
│ 严格按照migration定义编写                                     │
│ 每个字段都添加 column: 标签                                   │
│ 使用正确的Go类型映射                                          │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 4: 验证正确性                                           │
│ 运行 go build 检查编译                                       │
│ 编写基础CRUD单元测试                                          │
│ 手动测试插入和查询                                            │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 5: Code Review检查                                      │
│ 使用Model开发验证Checklist                                    │
│ 确保与data-model.md和migration完全一致                        │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 开发前必读文档

1. **data-model.md** - 数据模型设计规范 (权威定义)
2. **对应的migration文件** - 数据库实际结构 (实现参照)
3. **本规范文档** - Go Model编写规范 (编码标准)

---

## 二、Model定义模板

### 2.1 标准Model结构

```go
package models

import "time"

// TableName 表说明(对应data-model.md中的表名)
type TableName struct {
    // ==================== 主键字段 ====================
    ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

    // ==================== 业务字段 ====================
    // 必填字符串字段 - 使用column标签明确映射
    FieldName string `gorm:"column:field_name;type:varchar(100);not null" json:"field_name"`

    // 外键字段 - 明确指定column、index和关联
    ForeignID int64 `gorm:"column:foreign_id;index;not null" json:"foreign_id"`

    // 可空字符串字段 - 使用指针类型
    OptionalField *string `gorm:"column:optional_field;type:varchar(255)" json:"optional_field,omitempty"`

    // 状态字段 - 必须使用string类型(符合data-model.md规范)
    Status string `gorm:"column:status;type:varchar(20);not null" json:"status"`

    // 布尔字段 - 明确指定default值
    IsActive bool `gorm:"column:is_active;not null;default:true" json:"is_active"`

    // 整数字段
    Count int `gorm:"column:count;not null;default:0" json:"count"`

    // 金额字段 - 使用float64映射DECIMAL
    Amount float64 `gorm:"column:amount;type:decimal(12,2);not null" json:"amount"`

    // ==================== 时间字段 ====================
    CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    DeletedAt *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"` // 可空

    // ==================== 关联关系 ====================
    // 使用omitempty避免循环引用和N+1查询
    Related *RelatedModel `gorm:"foreignKey:ForeignID" json:"related,omitempty"`
}

// TableName 指定数据库表名(必须与migration中的表名一致)
func (TableName) TableName() string {
    return "table_names"
}

// ==================== 状态常量 ====================
// 使用字符串常量(符合data-model.md规范)
const (
    TableNameStatusActive   = "active"
    TableNameStatusInactive = "inactive"
    TableNameStatusPending  = "pending"
)

// ==================== 类型常量 ====================
const (
    TableNameTypeA = "type_a"
    TableNameTypeB = "type_b"
)
```

### 2.2 实际示例: Rental Model

**参考**: [migrations/000005_create_rentals.up.sql](../../../migrations/000005_create_rentals.up.sql)

```go
package models

import "time"

// Rental 租借订单
type Rental struct {
    ID                int64      `gorm:"primaryKey;autoIncrement" json:"id"`
    OrderID           int64      `gorm:"column:order_id;uniqueIndex;not null" json:"order_id"`
    UserID            int64      `gorm:"column:user_id;index;not null" json:"user_id"`
    DeviceID          int64      `gorm:"column:device_id;index;not null" json:"device_id"`
    DurationHours     int        `gorm:"column:duration_hours;not null" json:"duration_hours"`
    RentalFee         float64    `gorm:"column:rental_fee;type:decimal(10,2);not null" json:"rental_fee"`
    Deposit           float64    `gorm:"column:deposit;type:decimal(10,2);not null" json:"deposit"`
    OvertimeRate      float64    `gorm:"column:overtime_rate;type:decimal(10,2);not null" json:"overtime_rate"`
    OvertimeFee       float64    `gorm:"column:overtime_fee;type:decimal(10,2);not null;default:0" json:"overtime_fee"`
    Status            string     `gorm:"column:status;type:varchar(20);not null" json:"status"`
    UnlockedAt        *time.Time `gorm:"column:unlocked_at" json:"unlocked_at,omitempty"`
    ExpectedReturnAt  *time.Time `gorm:"column:expected_return_at" json:"expected_return_at,omitempty"`
    ReturnedAt        *time.Time `gorm:"column:returned_at" json:"returned_at,omitempty"`
    IsPurchased       bool       `gorm:"column:is_purchased;not null;default:false" json:"is_purchased"`
    PurchasedAt       *time.Time `gorm:"column:purchased_at" json:"purchased_at,omitempty"`
    CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

    // 关联
    Order  *Order  `gorm:"foreignKey:OrderID" json:"order,omitempty"`
    User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

func (Rental) TableName() string {
    return "rentals"
}

// RentalStatus 租借状态(字符串)
const (
    RentalStatusPending   = "pending"    // 待支付
    RentalStatusPaid      = "paid"       // 已支付(待取货)
    RentalStatusInUse     = "in_use"     // 使用中
    RentalStatusReturned  = "returned"   // 已归还
    RentalStatusCompleted = "completed"  // 已完成
    RentalStatusCancelled = "cancelled"  // 已取消
    RentalStatusRefunding = "refunding"  // 退款中
    RentalStatusRefunded  = "refunded"   // 已退款
)
```

---

## 三、字段类型映射规范

### 3.1 PostgreSQL → Go类型映射表

| PostgreSQL类型 | Go类型 | GORM标签示例 | 使用场景 |
|---------------|--------|--------------|----------|
| **BIGINT** | `int64` | `gorm:"type:bigint"` | 主键、外键、ID类字段 |
| **INT** | `int` | `gorm:"type:int"` | 数量、序号、小整数 |
| **SMALLINT** | `int16` | `gorm:"type:smallint"` | 小范围整数(不推荐用于状态) |
| **VARCHAR(n)** | `string` | `gorm:"type:varchar(100)"` | 短文本字段 |
| **TEXT** | `string` | `gorm:"type:text"` | 长文本、描述字段 |
| **DECIMAL(m,n)** | `float64` | `gorm:"type:decimal(12,2)"` | 金额、价格字段 |
| **BOOLEAN** | `bool` | `gorm:"type:boolean"` | 布尔标志 |
| **TIMESTAMP** | `time.Time` | `gorm:"type:timestamp"` | 必填时间字段 |
| **TIMESTAMP (可空)** | `*time.Time` | `gorm:"type:timestamp"` | 可空时间字段 |
| **JSON/JSONB** | `json.RawMessage` | `gorm:"type:jsonb"` | JSON数据 |
| **DATE** | `time.Time` | `gorm:"type:date"` | 日期字段 |
| **TIME** | `time.Time` | `gorm:"type:time"` | 时间字段 |

### 3.2 必填字段 vs 可空字段

```go
// ✅ 正确 - 必填字段使用值类型
Name        string     `gorm:"column:name;type:varchar(100);not null"`
Amount      float64    `gorm:"column:amount;type:decimal(10,2);not null"`
IsActive    bool       `gorm:"column:is_active;not null;default:true"`
CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`

// ✅ 正确 - 可空字段使用指针类型
Description *string    `gorm:"column:description;type:text"`
DeletedAt   *time.Time `gorm:"column:deleted_at"`
Remark      *string    `gorm:"column:remark;type:varchar(255)"`

// ❌ 错误 - 必填字段不应使用指针
Name        *string    `gorm:"column:name;not null"` // ← 错误!
```

### 3.3 状态字段必须使用字符串

**原则**: 所有状态字段必须使用 `string` 类型,而不是 `int`/`int8`/`int16`

**原因**:
1. ✅ **可读性强**: SQL查询结果直接可读
2. ✅ **自文档化**: 无需查询状态码映射表
3. ✅ **易扩展**: 新增状态不会有数字冲突
4. ✅ **日志友好**: 日志中直接显示状态名称
5. ✅ **API友好**: 返回JSON无需转换

```go
// ❌ 错误 - 不要使用整数状态码
Status int8 `gorm:"column:status;type:smallint"`

const (
    StatusPending  = 0
    StatusActive   = 1
    StatusInactive = 2
)

// ✅ 正确 - 使用字符串状态(符合data-model.md规范)
Status string `gorm:"column:status;type:varchar(20);not null"`

const (
    StatusPending  = "pending"
    StatusActive   = "active"
    StatusInactive = "inactive"
)
```

---

## 四、GORM标签规范

### 4.1 必须使用的标签

**所有字段都必须添加以下标签**:

1. **column**: 明确指定数据库列名
2. **type**: 明确指定数据库字段类型
3. **json**: 指定JSON序列化字段名

```go
// ✅ 完整的标签示例
DeviceID int64 `gorm:"column:device_id;index;not null" json:"device_id"`
```

### 4.2 常用GORM标签

| 标签 | 作用 | 示例 |
|------|------|------|
| `column:` | 指定列名 | `column:user_id` |
| `type:` | 指定数据库类型 | `type:varchar(100)` |
| `primaryKey` | 主键 | `primaryKey;autoIncrement` |
| `uniqueIndex` | 唯一索引 | `uniqueIndex` |
| `index` | 普通索引 | `index` |
| `not null` | 非空约束 | `not null` |
| `default:` | 默认值 | `default:0` |
| `autoCreateTime` | 自动创建时间 | `autoCreateTime` |
| `autoUpdateTime` | 自动更新时间 | `autoUpdateTime` |
| `foreignKey:` | 外键关联 | `foreignKey:UserID` |

### 4.3 完整示例

```go
type Order struct {
    // 主键
    ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

    // 唯一索引字段
    OrderNo string `gorm:"column:order_no;type:varchar(64);uniqueIndex;not null" json:"order_no"`

    // 外键索引字段
    UserID int64 `gorm:"column:user_id;index;not null" json:"user_id"`

    // 普通字段with默认值
    Status string `gorm:"column:status;type:varchar(20);not null;default:'pending'" json:"status"`

    // 金额字段
    Amount float64 `gorm:"column:amount;type:decimal(12,2);not null" json:"amount"`

    // 布尔字段with默认值
    IsActive bool `gorm:"column:is_active;not null;default:true" json:"is_active"`

    // 可空文本字段
    Remark *string `gorm:"column:remark;type:varchar(255)" json:"remark,omitempty"`

    // 自动时间戳
    CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

    // 关联关系
    User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

---

## 五、常见错误和避免方法

### 5.1 ❌ 错误1: 忘记添加column标签

**问题**: GORM会自动将驼峰命名转为蛇形命名,但可能转换错误

```go
// ❌ 错误 - GORM会将DeviceID转换为device_i_d(错误!)
DeviceID int64 `gorm:"index;not null" json:"device_id"`

// ✅ 正确 - 明确指定列名
DeviceID int64 `gorm:"column:device_id;index;not null" json:"device_id"`
```

---

### 5.2 ❌ 错误2: 字段名与数据库不一致

**问题**: 使用了不符合数据库的字段名

```go
// ❌ 错误 - 数据库是rental_fee,不是rental_amount
RentalAmount float64 `gorm:"column:rental_amount;type:decimal(10,2)" json:"rental_amount"`

// ✅ 正确 - 与数据库列名完全一致
RentalFee float64 `gorm:"column:rental_fee;type:decimal(10,2)" json:"rental_fee"`
```

**检查方法**:
1. 打开对应的migration文件
2. 复制完整的列名
3. 粘贴到column标签中

---

### 5.3 ❌ 错误3: 添加数据库中不存在的字段

**问题**: Model中定义了数据库没有的字段

```go
// ❌ 错误 - 数据库中没有rental_no字段
RentalNo string `gorm:"column:rental_no;type:varchar(64);uniqueIndex" json:"rental_no"`

// ✅ 正确 - 如果需要业务ID,应先在migration中添加字段
// Step 1: 在migration中添加字段
ALTER TABLE rentals ADD COLUMN rental_no VARCHAR(64) UNIQUE;

// Step 2: 然后在Model中定义
RentalNo string `gorm:"column:rental_no;type:varchar(64);uniqueIndex" json:"rental_no"`
```

---

### 5.4 ❌ 错误4: 遗漏必填字段

**问题**: Model中缺少数据库的必填字段

```go
// ❌ 错误 - 遗漏order_id必填外键
type Rental struct {
    ID       int64  `gorm:"primaryKey"`
    UserID   int64  `gorm:"column:user_id"`
    DeviceID int64  `gorm:"column:device_id"`
    // 缺少 OrderID - 创建时会失败!
}

// ✅ 正确 - 包含所有必填字段
type Rental struct {
    ID       int64 `gorm:"primaryKey"`
    OrderID  int64 `gorm:"column:order_id;uniqueIndex;not null"` // ← 必填外键
    UserID   int64 `gorm:"column:user_id;index;not null"`
    DeviceID int64 `gorm:"column:device_id;index;not null"`
}
```

**检查方法**:
```sql
-- 在migration文件中查找所有NOT NULL字段
-- 确保Model中都有对应定义
```

---

### 5.5 ❌ 错误5: 字段类型映射错误

```go
// ❌ 错误 - PostgreSQL BOOLEAN应映射为Go bool,不是int
IsActive int `gorm:"column:is_active;type:boolean"`

// ✅ 正确
IsActive bool `gorm:"column:is_active;not null;default:true"`

// ❌ 错误 - DECIMAL应映射为float64,不是int
Amount int `gorm:"column:amount;type:decimal(12,2)"`

// ✅ 正确
Amount float64 `gorm:"column:amount;type:decimal(12,2);not null"`
```

---

### 5.6 ❌ 错误6: 时间字段使用错误

```go
// ❌ 错误 - 必填时间字段不应使用指针
CreatedAt *time.Time `gorm:"column:created_at;autoCreateTime"`

// ✅ 正确 - 必填时间使用值类型
CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`

// ✅ 正确 - 可空时间使用指针类型
DeletedAt *time.Time `gorm:"column:deleted_at"`
```

---

## 六、Model开发验证Checklist

### 6.1 开发前检查 (必须完成)

- [ ] 已查阅 `data-model.md` 对应表的完整定义
- [ ] 已查看对应的 `migrations/000XXX_create_xxx.up.sql` 文件
- [ ] 理解了表的业务含义和字段用途
- [ ] 了解了该表与其他表的关联关系

### 6.2 编码中检查 (逐项验证)

- [ ] Model中所有字段都添加了 `column:` 标签
- [ ] 字段名与数据库列名完全一致
- [ ] 字段类型与数据库类型正确映射:
  - [ ] VARCHAR → string
  - [ ] BIGINT → int64
  - [ ] INT → int
  - [ ] DECIMAL → float64
  - [ ] BOOLEAN → bool
  - [ ] TIMESTAMP (必填) → time.Time
  - [ ] TIMESTAMP (可空) → *time.Time
- [ ] 状态字段使用 `string` 类型(而非int)
- [ ] 所有NOT NULL字段都定义为值类型
- [ ] 所有NULLABLE字段都定义为指针类型
- [ ] 没有添加数据库中不存在的字段
- [ ] 没有遗漏数据库中的必填字段
- [ ] 外键字段正确定义了关联关系
- [ ] TableName()方法返回正确的表名

### 6.3 开发后检查 (必须通过)

- [ ] 已运行 `go build ./internal/models/...` 验证编译通过
- [ ] 已编写基础CRUD单元测试
- [ ] 单元测试能够成功插入数据
- [ ] 单元测试能够成功查询数据
- [ ] 单元测试能够成功更新数据
- [ ] 所有测试用例通过
- [ ] 已手动测试在实际数据库中的CRUD操作

---

## 七、单元测试编写规范

### 7.1 基础CRUD测试模板

每个Model都必须编写基础CRUD测试:

```go
package models_test

import (
    "testing"
    "time"

    "github.com/dumeirei/smart-locker-backend/internal/models"
    "github.com/stretchr/testify/assert"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func TestRentalModel_CRUD(t *testing.T) {
    // 连接测试数据库
    dsn := "host=localhost user=postgres password=postgres dbname=smart_locker_test port=5432"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    assert.NoError(t, err)

    // 1. 测试创建
    rental := &models.Rental{
        OrderID:           1,
        UserID:            1,
        DeviceID:          1,
        DurationHours:     4,
        RentalFee:         18.00,
        Deposit:           50.00,
        OvertimeRate:      5.00,
        OvertimeFee:       0,
        Status:            models.RentalStatusPending,
        IsPurchased:       false,
    }
    err = db.Create(rental).Error
    assert.NoError(t, err)
    assert.NotZero(t, rental.ID)
    assert.NotZero(t, rental.CreatedAt)

    // 2. 测试查询
    var found models.Rental
    err = db.First(&found, rental.ID).Error
    assert.NoError(t, err)
    assert.Equal(t, rental.OrderID, found.OrderID)
    assert.Equal(t, rental.UserID, found.UserID)
    assert.Equal(t, rental.Status, found.Status)

    // 3. 测试更新
    now := time.Now()
    err = db.Model(&found).Updates(map[string]interface{}{
        "status":      models.RentalStatusPaid,
        "unlocked_at": now,
    }).Error
    assert.NoError(t, err)

    // 验证更新
    err = db.First(&found, rental.ID).Error
    assert.NoError(t, err)
    assert.Equal(t, models.RentalStatusPaid, found.Status)
    assert.NotNil(t, found.UnlockedAt)

    // 4. 测试删除
    err = db.Delete(&found).Error
    assert.NoError(t, err)

    // 验证删除
    err = db.First(&found, rental.ID).Error
    assert.Error(t, err) // 应该找不到记录
}

func TestRentalModel_Associations(t *testing.T) {
    // 测试关联加载
    dsn := "host=localhost user=postgres password=postgres dbname=smart_locker_test port=5432"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    assert.NoError(t, err)

    var rental models.Rental
    err = db.Preload("Order").Preload("User").Preload("Device").First(&rental, 1).Error
    assert.NoError(t, err)
    assert.NotNil(t, rental.Order)
    assert.NotNil(t, rental.User)
    assert.NotNil(t, rental.Device)
}
```

### 7.2 测试覆盖要求

- **必须测试**: Create, Read, Update, Delete
- **建议测试**: 关联加载, 批量操作, 事务操作
- **覆盖率要求**: >80%

---

## 八、Code Review检查点

### 8.1 Review时必须检查

1. **Model定义一致性**
   - [ ] Model定义与data-model.md完全一致
   - [ ] Model定义与migration文件完全一致
   - [ ] 所有字段都有column标签

2. **字段映射正确性**
   - [ ] 字段类型映射正确
   - [ ] 必填/可空字段使用正确的类型(值类型/指针类型)
   - [ ] 状态字段使用string类型

3. **完整性检查**
   - [ ] 没有多余字段(数据库中不存在的)
   - [ ] 没有遗漏必填字段
   - [ ] TableName()返回正确的表名

4. **测试覆盖**
   - [ ] 有对应的单元测试文件
   - [ ] CRUD操作都有测试
   - [ ] 所有测试都能通过

### 8.2 Review通过标准

只有满足以下所有条件,PR才能被批准:

✅ Model定义与data-model.md和migration完全一致
✅ 所有字段都有正确的column标签
✅ 字段类型映射正确
✅ 没有多余或遗漏的字段
✅ 有完整的单元测试
✅ 所有测试通过

---

## 九、工具和自动化 (计划中)

### 9.1 Migration to Model Generator (未来)

**功能**: 从migration文件自动生成Model代码

```bash
./tools/gen-model-from-migration.sh migrations/000005_create_rentals.up.sql \
  > internal/models/rental_generated.go
```

### 9.2 Schema Consistency Checker (未来)

**功能**: 自动检查Model定义与migration的一致性

```bash
./tools/check-model-schema-consistency.sh

# 输出示例:
# ❌ Error: Field mismatch in Rental model
#   - Missing required field: order_id (BIGINT NOT NULL)
#   - Extra field found: rental_no (not in database)
#   - Type mismatch: Status (int8 vs VARCHAR(20))
```

### 9.3 CI/CD集成 (未来)

在CI pipeline中自动检查:

```yaml
- name: Check Model-Schema Consistency
  run: ./tools/check-model-schema-consistency.sh
```

---

## 十、参考资料

1. **data-model.md** - 数据模型设计规范(权威参照)
2. **migrations/*.up.sql** - 数据库迁移文件(实际结构)
3. **GORM文档**: https://gorm.io/docs/
4. **PostgreSQL文档**: https://www.postgresql.org/docs/

---

## 十一、FAQ

### Q1: 为什么状态字段必须用string而不是int?

**A**: 原因见第3.3节。简而言之:
- ✅ 更易读、易维护、易调试
- ✅ 符合data-model.md的设计规范
- ✅ API和日志更友好

### Q2: 什么时候字段用指针,什么时候用值类型?

**A**: 规则很简单:
- **必填字段(NOT NULL)**: 使用值类型 (如 `string`, `int64`, `bool`)
- **可空字段(NULLABLE)**: 使用指针类型 (如 `*string`, `*time.Time`)

### Q3: 如果我需要添加数据库中没有的字段怎么办?

**A**: 正确的流程:
1. 先创建migration文件添加字段
2. 运行migration更新数据库
3. 然后在Model中添加对应字段

**不要**直接在Model中添加数据库没有的字段!

### Q4: column标签可以省略吗?

**A**: **不可以!** 所有字段都必须添加column标签,原因:
1. 避免GORM自动转换错误
2. 代码可读性更好
3. 便于Code Review检查

### Q5: 如何确保我的Model定义是正确的?

**A**: 使用本规范第六节的Checklist逐项检查。

---

**文档版本**: v1.0
**最后更新**: 2026-01-03
**维护者**: 后端开发团队
