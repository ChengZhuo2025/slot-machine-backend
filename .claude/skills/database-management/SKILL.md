---
name: Database Management
description: This skill should be used when the user asks to "create a migration", "add database table", "modify schema", "optimize query", "add index", "database design", "GORM model", "PostgreSQL query", or needs guidance on database operations, migrations, schema design, and query optimization for the smart locker backend.
version: 1.0.0
---

# Database Management Skill

This skill provides guidance for PostgreSQL database management in the smart locker backend system.

## Database Stack

| Component | Version | Purpose |
|-----------|---------|---------|
| PostgreSQL | 15+ | Primary database |
| GORM | v1.25+ | ORM |
| golang-migrate | - | Schema migrations |
| Redis | 7+ | Caching layer |

## Migration Management

### Migration File Naming

```
migrations/
├── 000001_init_users.up.sql
├── 000001_init_users.down.sql
├── 000002_init_devices.up.sql
├── 000002_init_devices.down.sql
└── ...
```

### Create Migration

```bash
# Using golang-migrate
migrate create -ext sql -dir migrations -seq init_users

# Or using make command
make migrate-create name=init_users
```

### Example Migration

```sql
-- 000001_init_users.up.sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    phone VARCHAR(20) UNIQUE NOT NULL,
    openid VARCHAR(64) UNIQUE,
    unionid VARCHAR(64),
    nickname VARCHAR(50) NOT NULL,
    avatar VARCHAR(255),
    gender SMALLINT DEFAULT 0,
    birthday DATE,
    member_level_id BIGINT DEFAULT 1,
    points INT DEFAULT 0,
    is_verified BOOLEAN DEFAULT FALSE,
    real_name_encrypted TEXT,
    id_card_encrypted TEXT,
    referrer_id BIGINT REFERENCES users(id),
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_user_phone ON users(phone);
CREATE INDEX idx_user_openid ON users(openid);
CREATE INDEX idx_user_referrer ON users(referrer_id);

-- 000001_init_users.down.sql
DROP TABLE IF EXISTS users;
```

### Run Migrations

```bash
# Apply all migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check migration status
make migrate-status

# Force version (use carefully)
migrate -path migrations -database "$DB_URL" force 1
```

## GORM Model Definitions

**⚠️ CRITICAL**: Always define models to match migration files EXACTLY.

**MANDATORY RULES**:
1. **ALL fields MUST have `column:` tag** to ensure correct mapping
2. **Field names must match database column names**
3. **Status fields MUST use `string` type** (not int/smallint) - improves readability and maintainability
4. **Refer to `data-model.md` and migration files** before writing any Model

### Model with Correct Field Mapping

```go
package models

import (
    "time"
)

// Order model - Refer to migrations/000003_create_orders.up.sql and data-model.md
type Order struct {
    // Primary key
    ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

    // Business fields - ALL require column: tag
    OrderNo        string     `gorm:"column:order_no;type:varchar(64);uniqueIndex;not null" json:"order_no"`
    UserID         int64      `gorm:"column:user_id;index;not null" json:"user_id"`
    Type           string     `gorm:"column:type;type:varchar(20);index;not null" json:"type"`
    OriginalAmount float64    `gorm:"column:original_amount;type:decimal(12,2);not null" json:"original_amount"`
    DiscountAmount float64    `gorm:"column:discount_amount;type:decimal(12,2);not null;default:0" json:"discount_amount"`
    ActualAmount   float64    `gorm:"column:actual_amount;type:decimal(12,2);not null" json:"actual_amount"`
    DepositAmount  float64    `gorm:"column:deposit_amount;type:decimal(12,2);not null;default:0" json:"deposit_amount"`

    // Status field - Use string type for clarity (NOT int/smallint!)
    Status         string     `gorm:"column:status;type:varchar(20);index;not null" json:"status"`

    // Optional fields - Use pointer types for NULLABLE columns
    CouponID       *int64     `gorm:"column:coupon_id;index" json:"coupon_id,omitempty"`
    Remark         *string    `gorm:"column:remark;type:varchar(255)" json:"remark,omitempty"`

    // Timestamps
    PaidAt         *time.Time `gorm:"column:paid_at" json:"paid_at,omitempty"`
    CompletedAt    *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
    CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
    UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    DeletedAt      *time.Time `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

    // Relations
    User      *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Items     []OrderItem  `gorm:"foreignKey:OrderID" json:"items,omitempty"`
    Payments  []Payment    `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
}

func (Order) TableName() string {
    return "orders"
}

// OrderStatus constants - Use strings for readability
const (
    OrderStatusPending   = "pending"
    OrderStatusPaid      = "paid"
    OrderStatusShipping  = "shipping"
    OrderStatusCompleted = "completed"
    OrderStatusCancelled = "cancelled"
    OrderStatusRefunded  = "refunded"
)
```

### Field Type Mapping Rules

| PostgreSQL Type | Go Type | GORM Tag Example | Notes |
|----------------|---------|------------------|-------|
| BIGINT | `int64` | `gorm:"type:bigint"` | Primary keys, foreign keys |
| INT | `int` | `gorm:"type:int"` | Counts, quantities |
| SMALLINT | `int16` | `gorm:"type:smallint"` | Small numbers (NOT for status!) |
| VARCHAR(n) | `string` | `gorm:"column:name;type:varchar(100)"` | Text fields |
| TEXT | `string` | `gorm:"column:content;type:text"` | Long text |
| DECIMAL(m,n) | `float64` | `gorm:"column:amount;type:decimal(12,2)"` | Money, prices |
| BOOLEAN | `bool` | `gorm:"column:is_active;type:boolean"` | Flags |
| TIMESTAMP (required) | `time.Time` | `gorm:"column:created_at"` | Required timestamps |
| TIMESTAMP (nullable) | `*time.Time` | `gorm:"column:deleted_at"` | Optional timestamps |

### Development Workflow

**BEFORE writing any Model code**:

1. **Read data-model.md**: Check `specs/001-smart-locker-backend/data-model.md` for table definition
2. **Read migration file**: Open corresponding `migrations/000XXX_create_xxx.up.sql`
3. **Copy column names**: Copy exact column names from CREATE TABLE statement
4. **Add column: tags**: Every field must have `column:` tag
5. **Match types**: Use correct Go type mapping (see table above)
6. **Status as string**: Never use int/int8 for status fields
7. **Test immediately**: Write basic CRUD test to verify Model works

**Reference**: See `specs/001-smart-locker-backend/model-development-guide.md` for complete standards.

### Soft Delete

```go
// Query includes soft deleted
db.Unscoped().Where("user_id = ?", userID).Find(&orders)

// Permanently delete
db.Unscoped().Delete(&order)

// Restore soft deleted
db.Model(&order).Update("deleted_at", nil)
```

## Query Patterns

### Efficient Queries

```go
// Select specific columns
db.Select("id", "nickname", "avatar").Find(&users)

// Preload relations efficiently
db.Preload("Items").Preload("Payments").First(&order, orderID)

// Conditional preload
db.Preload("Items", "quantity > ?", 1).Find(&orders)

// Joins for better performance
db.Joins("User").Find(&orders)
```

### Pagination

```go
func (r *OrderRepository) List(ctx context.Context, userID int64, page, pageSize int) ([]models.Order, int64, error) {
    var orders []models.Order
    var total int64

    query := r.db.WithContext(ctx).
        Model(&models.Order{}).
        Where("user_id = ?", userID)

    // Get total count
    query.Count(&total)

    // Get paginated results
    err := query.
        Order("created_at DESC").
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&orders).Error

    return orders, total, err
}
```

### Batch Operations

```go
// Batch insert
users := []models.User{{...}, {...}, {...}}
db.CreateInBatches(users, 100)

// Batch update
db.Model(&models.Product{}).
    Where("category_id = ?", categoryID).
    Updates(map[string]interface{}{"is_on_sale": false})

// Batch delete
db.Where("status = ? AND created_at < ?", "expired", deadline).
    Delete(&models.UserCoupon{})
```

## Index Strategy

### High-Priority Indexes

```sql
-- User lookups
CREATE UNIQUE INDEX idx_user_phone ON users(phone);
CREATE UNIQUE INDEX idx_user_openid ON users(openid) WHERE openid IS NOT NULL;

-- Order queries
CREATE INDEX idx_order_user ON orders(user_id);
CREATE INDEX idx_order_type_status ON orders(type, status);
CREATE INDEX idx_order_created ON orders(created_at DESC);

-- Device queries
CREATE INDEX idx_device_venue ON devices(venue_id);
CREATE INDEX idx_device_status ON devices(online_status, rental_status);

-- Payment queries
CREATE INDEX idx_payment_order ON payments(order_id);
CREATE INDEX idx_payment_no ON payments(payment_no);
```

### Composite Indexes

```sql
-- For queries: WHERE user_id = ? AND status = ? ORDER BY created_at DESC
CREATE INDEX idx_order_user_status_time ON orders(user_id, status, created_at DESC);

-- For queries: WHERE venue_id = ? AND duration_hours = ?
CREATE UNIQUE INDEX idx_pricing_venue_duration ON rental_pricings(venue_id, duration_hours);
```

## Table Partitioning

### Orders Table Partitioning

```sql
-- Create partitioned table
CREATE TABLE orders (
    id BIGSERIAL,
    order_no VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    -- other columns...
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE orders_2026_01 PARTITION OF orders
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE orders_2026_02 PARTITION OF orders
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- Automate partition creation
CREATE OR REPLACE FUNCTION create_monthly_partition()
RETURNS void AS $$
DECLARE
    start_date DATE := date_trunc('month', NOW() + interval '1 month');
    end_date DATE := start_date + interval '1 month';
    partition_name TEXT := 'orders_' || to_char(start_date, 'YYYY_MM');
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF orders
         FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;
```

## Query Optimization

### EXPLAIN ANALYZE

```sql
EXPLAIN ANALYZE
SELECT o.*, u.nickname
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE o.status = 'completed'
  AND o.created_at > NOW() - INTERVAL '30 days'
ORDER BY o.created_at DESC
LIMIT 20;
```

### Common Optimizations

1. **Use covering indexes** for frequently queried columns
2. **Avoid SELECT *** - specify needed columns
3. **Use LIMIT** for pagination
4. **Index foreign keys** used in JOINs
5. **Batch large updates** to avoid long locks

### Query Performance Tips

```go
// Use raw SQL for complex queries
var results []struct {
    UserID      int64
    TotalAmount decimal.Decimal
    OrderCount  int
}

db.Raw(`
    SELECT user_id,
           SUM(actual_amount) as total_amount,
           COUNT(*) as order_count
    FROM orders
    WHERE status = 'completed'
      AND created_at >= ?
    GROUP BY user_id
    HAVING COUNT(*) > 5
    ORDER BY total_amount DESC
    LIMIT 100
`, startDate).Scan(&results)
```

## Connection Pool

### Configuration

```go
func SetupDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        return nil, err
    }

    sqlDB, _ := db.DB()

    // Connection pool settings
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)
    sqlDB.SetConnMaxIdleTime(10 * time.Minute)

    return db, nil
}
```

## Data Model Reference

For complete data model definitions, see:
- **`specs/001-smart-locker-backend/data-model.md`** - Entity definitions
- **`references/schema-reference.md`** - Full schema details

## Additional Resources

### Reference Files

- **`references/schema-reference.md`** - Complete database schema
- **`references/query-patterns.md`** - Advanced query patterns
