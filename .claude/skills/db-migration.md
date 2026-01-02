# 数据库迁移技能

## 概述

本技能用于根据 `data-model.md` 规范生成 PostgreSQL 数据库迁移文件。

## 技术栈

- **数据库**: PostgreSQL 15+
- **迁移工具**: golang-migrate/migrate
- **文件格式**: SQL (up/down 配对)

## 文件命名规范

```
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_user_wallets.up.sql
├── 000002_create_user_wallets.down.sql
└── ...
```

格式: `{序号}_{描述}.{up|down}.sql`
- 序号: 6位数字，从 000001 开始
- 描述: 小写下划线分隔，简明描述操作
- 方向: up (升级) / down (回滚)

## 迁移文件模板

### 创建表 (up)

```sql
-- migrations/000001_create_users.up.sql
-- 创建用户表

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    phone VARCHAR(20) NOT NULL,
    openid VARCHAR(64),
    unionid VARCHAR(64),
    nickname VARCHAR(50) NOT NULL,
    avatar VARCHAR(255),
    gender SMALLINT DEFAULT 0,
    birthday DATE,
    member_level_id BIGINT DEFAULT 1,
    points INTEGER DEFAULT 0,
    is_verified BOOLEAN DEFAULT FALSE,
    real_name_encrypted TEXT,
    id_card_encrypted TEXT,
    referrer_id BIGINT,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_openid ON users(openid) WHERE openid IS NOT NULL;

-- 普通索引
CREATE INDEX IF NOT EXISTS idx_users_referrer ON users(referrer_id);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- 字段注释
COMMENT ON TABLE users IS '用户表';
COMMENT ON COLUMN users.id IS '用户ID';
COMMENT ON COLUMN users.phone IS '手机号';
COMMENT ON COLUMN users.openid IS '微信OpenID';
COMMENT ON COLUMN users.unionid IS '微信UnionID';
COMMENT ON COLUMN users.nickname IS '昵称';
COMMENT ON COLUMN users.avatar IS '头像URL';
COMMENT ON COLUMN users.gender IS '性别: 0未知 1男 2女';
COMMENT ON COLUMN users.birthday IS '生日';
COMMENT ON COLUMN users.member_level_id IS '会员等级ID';
COMMENT ON COLUMN users.points IS '积分';
COMMENT ON COLUMN users.is_verified IS '是否实名认证';
COMMENT ON COLUMN users.real_name_encrypted IS '加密的真实姓名';
COMMENT ON COLUMN users.id_card_encrypted IS '加密的身份证号';
COMMENT ON COLUMN users.referrer_id IS '推荐人用户ID';
COMMENT ON COLUMN users.status IS '状态: 0禁用 1正常';
```

### 删除表 (down)

```sql
-- migrations/000001_create_users.down.sql
-- 回滚：删除用户表

DROP TABLE IF EXISTS users CASCADE;
```

### 添加外键约束

```sql
-- migrations/000003_add_foreign_keys.up.sql
-- 添加外键约束

ALTER TABLE users
    ADD CONSTRAINT fk_users_member_level
    FOREIGN KEY (member_level_id)
    REFERENCES member_levels(id)
    ON DELETE SET NULL;

ALTER TABLE users
    ADD CONSTRAINT fk_users_referrer
    FOREIGN KEY (referrer_id)
    REFERENCES users(id)
    ON DELETE SET NULL;
```

```sql
-- migrations/000003_add_foreign_keys.down.sql
-- 回滚：删除外键约束

ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_member_level;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_referrer;
```

### 创建分区表

```sql
-- migrations/000006_create_orders.up.sql
-- 创建订单表（按月分区）

CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL,
    order_no VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    type VARCHAR(20) NOT NULL,
    original_amount DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    actual_amount DECIMAL(12,2) NOT NULL,
    deposit_amount DECIMAL(12,2) DEFAULT 0,
    status VARCHAR(20) NOT NULL,
    coupon_id BIGINT,
    remark VARCHAR(255),
    address_id BIGINT,
    address_snapshot JSONB,
    express_company VARCHAR(50),
    express_no VARCHAR(64),
    shipped_at TIMESTAMP WITH TIME ZONE,
    received_at TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancel_reason VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建分区
CREATE TABLE orders_2026_01 PARTITION OF orders
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE orders_2026_02 PARTITION OF orders
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
-- ... 其他月份

-- 索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_order_no ON orders(order_no);
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_type_status ON orders(type, status);
CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at DESC);
```

## 数据类型映射

| Go 类型 | PostgreSQL 类型 | 说明 |
|---------|-----------------|------|
| `int64` | `BIGINT` / `BIGSERIAL` | 主键用 BIGSERIAL |
| `int` | `INTEGER` | |
| `int8` / `int16` | `SMALLINT` | 状态/枚举 |
| `string` | `VARCHAR(n)` / `TEXT` | 固定长度用 VARCHAR |
| `bool` | `BOOLEAN` | |
| `float64` | `DECIMAL(p,s)` | 金额必须用 DECIMAL |
| `time.Time` | `TIMESTAMP WITH TIME ZONE` | |
| `*time.Time` | `TIMESTAMP WITH TIME ZONE` | 可空 |
| `[]string` / `map` | `JSONB` | JSON 数据 |

## 索引命名规范

| 类型 | 格式 | 示例 |
|------|------|------|
| 主键 | `{表名}_pkey` | `users_pkey` |
| 唯一索引 | `idx_{表名}_{字段}` | `idx_users_phone` |
| 普通索引 | `idx_{表名}_{字段}` | `idx_users_status` |
| 组合索引 | `idx_{表名}_{字段1}_{字段2}` | `idx_orders_type_status` |
| 外键约束 | `fk_{表名}_{关联表}` | `fk_users_member_level` |

## 常用 SQL 片段

### 自动更新时间戳

```sql
-- 创建触发器函数
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 应用到表
CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```

### 软删除索引

```sql
-- 只索引未删除的记录
CREATE INDEX idx_users_active ON users(id) WHERE deleted_at IS NULL;
```

### 乐观锁字段

```sql
version INTEGER DEFAULT 0 NOT NULL,

-- 更新时检查版本
UPDATE user_wallets
SET balance = balance - 100, version = version + 1
WHERE id = 1 AND version = 5;
```

## 迁移命令

```bash
# 安装 migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 创建迁移文件
migrate create -ext sql -dir migrations -seq create_users

# 执行迁移
migrate -path migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up

# 回滚一步
migrate -path migrations -database "..." down 1

# 回滚所有
migrate -path migrations -database "..." down

# 查看版本
migrate -path migrations -database "..." version

# 强制设置版本（修复脏状态）
migrate -path migrations -database "..." force 5
```

## Makefile 集成

```makefile
DB_URL ?= postgres://postgres:password@localhost:5432/smartlocker?sslmode=disable

.PHONY: migrate-up migrate-down migrate-create

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name
```

## 参考文档

- 数据模型: `specs/001-smart-locker-backend/data-model.md`
- 任务清单: `specs/001-smart-locker-backend/tasks.md` (T013-T029)
