# 种子数据生成技能

## 概述

本技能用于从前端 mock 数据中提取初始化数据，生成 PostgreSQL 种子数据 SQL 文件，便于开发和测试环境快速搭建。

## 数据来源

### 前端 Mock 数据位置

```
admin-frontend/src/mock/           # 管理后台 mock 数据
├── user.ts                        # 用户数据
├── device.ts                      # 设备数据
├── venue.ts                       # 场地数据
├── merchant.ts                    # 商户数据
├── hotel.ts                       # 酒店数据
├── room.ts                        # 房间数据
├── product.ts                     # 商品数据
├── order.ts                       # 订单数据
├── coupon.ts                      # 优惠券数据
└── ...

user-frontend/src/mock/            # 用户端 mock 数据
├── home.ts                        # 首页数据
├── rental.ts                      # 租借数据
├── mall.ts                        # 商城数据
└── ...
```

## 种子文件结构

```
backend/migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── ...
└── seeds/                         # 种子数据目录
    ├── 001_seed_basic.sql         # 基础配置数据
    ├── 002_seed_users.sql         # 用户数据
    ├── 003_seed_devices.sql       # 设备数据
    ├── 004_seed_venues.sql        # 场地数据
    ├── 005_seed_merchants.sql     # 商户数据
    ├── 006_seed_hotels.sql        # 酒店和房间数据
    ├── 007_seed_products.sql      # 商品数据
    ├── 008_seed_orders.sql        # 订单数据
    └── 009_seed_marketing.sql     # 营销数据
```

## 种子数据模板

### 001 基础配置数据

```sql
-- migrations/seeds/001_seed_basic.sql
-- 基础配置数据（会员等级、系统配置、角色权限）

-- 会员等级
INSERT INTO member_levels (id, name, min_points, discount, description, created_at, updated_at) VALUES
(1, '普通会员', 0, 1.00, '基础会员，无特殊权益', NOW(), NOW()),
(2, '银卡会员', 1000, 0.95, '95折优惠', NOW(), NOW()),
(3, '金卡会员', 5000, 0.90, '9折优惠', NOW(), NOW()),
(4, '钻石会员', 20000, 0.85, '85折优惠 + 专属客服', NOW(), NOW());

-- 重置序列
SELECT setval('member_levels_id_seq', (SELECT MAX(id) FROM member_levels));

-- 系统配置
INSERT INTO system_configs (key, value, description, created_at, updated_at) VALUES
('site_name', '"爱上杜美人"', '站点名称', NOW(), NOW()),
('customer_service_phone', '"400-888-8888"', '客服电话', NOW(), NOW()),
('rental_deposit', '99.00', '租借押金（元）', NOW(), NOW()),
('rental_overtime_rate', '10.00', '超时费率（元/小时）', NOW(), NOW()),
('max_rental_hours', '24', '最大租借时长（小时）', NOW(), NOW()),
('commission_rate_level1', '0.10', '一级分销佣金比例', NOW(), NOW()),
('commission_rate_level2', '0.05', '二级分销佣金比例', NOW(), NOW()),
('min_withdraw_amount', '100.00', '最低提现金额（元）', NOW(), NOW()),
('jwt_access_token_expire', '7200', 'Access Token 有效期（秒）', NOW(), NOW()),
('jwt_refresh_token_expire', '2592000', 'Refresh Token 有效期（秒）', NOW(), NOW());

-- 管理员角色
INSERT INTO roles (id, name, code, description, created_at, updated_at) VALUES
(1, '超级管理员', 'super_admin', '拥有所有权限', NOW(), NOW()),
(2, '平台管理员', 'platform_admin', '平台日常管理', NOW(), NOW()),
(3, '运营管理员', 'operation_admin', '运营相关权限', NOW(), NOW()),
(4, '财务管理员', 'finance_admin', '财务相关权限', NOW(), NOW()),
(5, '合作商', 'merchant', '商户管理权限', NOW(), NOW()),
(6, '客服', 'customer_service', '客服权限', NOW(), NOW());

SELECT setval('roles_id_seq', (SELECT MAX(id) FROM roles));

-- 租借价格配置
INSERT INTO rental_pricings (id, duration_hours, price, is_active, created_at, updated_at) VALUES
(1, 1, 19.90, true, NOW(), NOW()),
(2, 2, 29.90, true, NOW(), NOW()),
(3, 3, 39.90, true, NOW(), NOW()),
(4, 6, 59.90, true, NOW(), NOW()),
(5, 12, 89.90, true, NOW(), NOW()),
(6, 24, 129.90, true, NOW(), NOW());

SELECT setval('rental_pricings_id_seq', (SELECT MAX(id) FROM rental_pricings));
```

### 002 用户数据

```sql
-- migrations/seeds/002_seed_users.sql
-- 用户数据（从前端 mock 提取）

-- 测试用户
INSERT INTO users (id, phone, openid, nickname, avatar, gender, member_level_id, points, is_verified, status, created_at, updated_at) VALUES
(1, '13800138001', 'wx_openid_001', '张三', 'https://example.com/avatar/1.jpg', 1, 2, 1500, true, 1, NOW(), NOW()),
(2, '13800138002', 'wx_openid_002', '李四', 'https://example.com/avatar/2.jpg', 1, 1, 500, false, 1, NOW(), NOW()),
(3, '13800138003', 'wx_openid_003', '王五', 'https://example.com/avatar/3.jpg', 2, 3, 8000, true, 1, NOW(), NOW()),
(4, '13800138004', 'wx_openid_004', '赵六', NULL, 0, 1, 0, false, 1, NOW(), NOW()),
(5, '13800138005', 'wx_openid_005', '测试用户', 'https://example.com/avatar/5.jpg', 1, 4, 25000, true, 1, NOW(), NOW());

SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));

-- 用户钱包
INSERT INTO user_wallets (id, user_id, balance, frozen_balance, total_recharged, total_consumed, version, created_at, updated_at) VALUES
(1, 1, 100.00, 0.00, 500.00, 400.00, 1, NOW(), NOW()),
(2, 2, 50.00, 0.00, 100.00, 50.00, 1, NOW(), NOW()),
(3, 3, 500.00, 99.00, 1000.00, 401.00, 1, NOW(), NOW()),
(4, 4, 0.00, 0.00, 0.00, 0.00, 1, NOW(), NOW()),
(5, 5, 1000.00, 0.00, 2000.00, 1000.00, 1, NOW(), NOW());

SELECT setval('user_wallets_id_seq', (SELECT MAX(id) FROM user_wallets));

-- 管理员账号
INSERT INTO admins (id, username, password_hash, real_name, phone, email, role_id, status, created_at, updated_at) VALUES
(1, 'admin', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqVmHQZKC.M/RqQb95Y7PqKXq.e9jHm', '系统管理员', '13900139000', 'admin@example.com', 1, 1, NOW(), NOW()),
(2, 'operator', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqVmHQZKC.M/RqQb95Y7PqKXq.e9jHm', '运营小王', '13900139001', 'operator@example.com', 3, 1, NOW(), NOW()),
(3, 'finance', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqVmHQZKC.M/RqQb95Y7PqKXq.e9jHm', '财务小李', '13900139002', 'finance@example.com', 4, 1, NOW(), NOW());

SELECT setval('admins_id_seq', (SELECT MAX(id) FROM admins));

-- 注：密码 hash 对应明文密码 "admin123"
```

### 003 设备数据

```sql
-- migrations/seeds/003_seed_devices.sql
-- 设备数据

INSERT INTO devices (id, device_no, qrcode_url, name, model, status, venue_id, merchant_id, firmware_version, last_heartbeat, created_at, updated_at) VALUES
(1, 'DV20260001', 'https://example.com/qr/DV20260001.png', '1号柜', 'SL-100', 'online', 1, 1, 'v2.1.0', NOW(), NOW(), NOW()),
(2, 'DV20260002', 'https://example.com/qr/DV20260002.png', '2号柜', 'SL-100', 'online', 1, 1, 'v2.1.0', NOW(), NOW(), NOW()),
(3, 'DV20260003', 'https://example.com/qr/DV20260003.png', '3号柜', 'SL-200', 'offline', 2, 1, 'v2.0.5', NOW() - INTERVAL '2 hours', NOW(), NOW()),
(4, 'DV20260004', 'https://example.com/qr/DV20260004.png', '大堂柜', 'SL-200', 'online', 3, 2, 'v2.1.0', NOW(), NOW(), NOW()),
(5, 'DV20260005', 'https://example.com/qr/DV20260005.png', '房间柜-101', 'SL-100', 'online', 4, 2, 'v2.1.0', NOW(), NOW(), NOW()),
(6, 'DV20260006', 'https://example.com/qr/DV20260006.png', '房间柜-102', 'SL-100', 'maintenance', 4, 2, 'v2.0.5', NOW() - INTERVAL '1 day', NOW(), NOW());

SELECT setval('devices_id_seq', (SELECT MAX(id) FROM devices));
```

### 004 场地数据

```sql
-- migrations/seeds/004_seed_venues.sql
-- 场地数据

INSERT INTO venues (id, name, type, address, latitude, longitude, contact_name, contact_phone, status, created_at, updated_at) VALUES
(1, '万达广场店', 'mall', '北京市朝阳区建国路88号万达广场B1层', 39.9087, 116.4716, '张经理', '13800001111', 1, NOW(), NOW()),
(2, '国贸商城店', 'mall', '北京市朝阳区建国门外大街1号国贸商城', 39.9089, 116.4588, '李经理', '13800002222', 1, NOW(), NOW()),
(3, '如家酒店大堂', 'hotel', '北京市海淀区中关村大街1号', 39.9833, 116.3167, '王店长', '13800003333', 1, NOW(), NOW()),
(4, '如家酒店客房区', 'hotel', '北京市海淀区中关村大街1号', 39.9833, 116.3167, '王店长', '13800003333', 1, NOW(), NOW());

SELECT setval('venues_id_seq', (SELECT MAX(id) FROM venues));
```

### 005 商户数据

```sql
-- migrations/seeds/005_seed_merchants.sql
-- 商户数据

INSERT INTO merchants (id, name, contact_name, contact_phone, email, commission_rate, settlement_account, settlement_type, status, created_at, updated_at) VALUES
(1, '北京爱尚科技有限公司', '张总', '13800001000', 'zhang@example.com', 0.70, '{"bank": "工商银行", "account": "6222001234567890", "name": "北京爱尚科技有限公司"}', 'bank', 1, NOW(), NOW()),
(2, '上海情趣生活商贸有限公司', '李总', '13800002000', 'li@example.com', 0.65, '{"bank": "建设银行", "account": "6227001234567890", "name": "上海情趣生活商贸有限公司"}', 'bank', 1, NOW(), NOW());

SELECT setval('merchants_id_seq', (SELECT MAX(id) FROM merchants));
```

### 006 酒店和房间数据

```sql
-- migrations/seeds/006_seed_hotels.sql
-- 酒店和房间数据

-- 酒店
INSERT INTO hotels (id, name, address, latitude, longitude, phone, description, facilities, images, status, created_at, updated_at) VALUES
(1, '如家精选酒店(中关村店)', '北京市海淀区中关村大街1号', 39.9833, 116.3167, '010-88886666', '位于中关村核心地段，交通便利', '["wifi", "parking", "breakfast"]', '["https://example.com/hotel/1/1.jpg", "https://example.com/hotel/1/2.jpg"]', 1, NOW(), NOW()),
(2, '汉庭酒店(国贸店)', '北京市朝阳区建国门外大街2号', 39.9089, 116.4600, '010-88887777', '紧邻国贸CBD，商务出行首选', '["wifi", "parking", "gym"]', '["https://example.com/hotel/2/1.jpg"]', 1, NOW(), NOW());

SELECT setval('hotels_id_seq', (SELECT MAX(id) FROM hotels));

-- 房间
INSERT INTO rooms (id, hotel_id, room_no, room_type, description, images, device_id, status, created_at, updated_at) VALUES
(1, 1, '101', '大床房', '25平米，1.8米大床', '["https://example.com/room/1/1.jpg"]', 5, 'available', NOW(), NOW()),
(2, 1, '102', '双床房', '28平米，两张1.2米床', '["https://example.com/room/2/1.jpg"]', 6, 'maintenance', NOW(), NOW()),
(3, 1, '201', '豪华套房', '45平米，带客厅', '["https://example.com/room/3/1.jpg"]', NULL, 'available', NOW(), NOW()),
(4, 2, '301', '商务大床房', '30平米，1.8米大床', '["https://example.com/room/4/1.jpg"]', NULL, 'available', NOW(), NOW());

SELECT setval('rooms_id_seq', (SELECT MAX(id) FROM rooms));

-- 房间时段价格
INSERT INTO room_time_slots (id, room_id, time_slot, price, created_at, updated_at) VALUES
(1, 1, '2h', 68.00, NOW(), NOW()),
(2, 1, '4h', 98.00, NOW(), NOW()),
(3, 1, '6h', 128.00, NOW(), NOW()),
(4, 1, 'overnight', 188.00, NOW(), NOW()),
(5, 2, '2h', 78.00, NOW(), NOW()),
(6, 2, '4h', 118.00, NOW(), NOW()),
(7, 3, '2h', 128.00, NOW(), NOW()),
(8, 3, '4h', 188.00, NOW(), NOW());

SELECT setval('room_time_slots_id_seq', (SELECT MAX(id) FROM room_time_slots));
```

### 007 商品数据

```sql
-- migrations/seeds/007_seed_products.sql
-- 商品分类和商品数据

-- 商品分类
INSERT INTO product_categories (id, parent_id, name, sort_order, status, created_at, updated_at) VALUES
(1, NULL, '情趣用品', 1, 1, NOW(), NOW()),
(2, 1, '振动棒', 1, 1, NOW(), NOW()),
(3, 1, '跳蛋', 2, 1, NOW(), NOW()),
(4, 1, '延时用品', 3, 1, NOW(), NOW()),
(5, NULL, '安全套', 2, 1, NOW(), NOW()),
(6, NULL, '润滑液', 3, 1, NOW(), NOW());

SELECT setval('product_categories_id_seq', (SELECT MAX(id) FROM product_categories));

-- 商品
INSERT INTO products (id, category_id, name, description, main_image, images, price, original_price, stock, sales, status, created_at, updated_at) VALUES
(1, 2, '智能加温振动棒', '10频振动，USB充电，防水设计', 'https://example.com/product/1/main.jpg', '["https://example.com/product/1/1.jpg", "https://example.com/product/1/2.jpg"]', 199.00, 299.00, 100, 500, 1, NOW(), NOW()),
(2, 2, '迷你便携振动棒', '小巧便携，静音设计', 'https://example.com/product/2/main.jpg', '["https://example.com/product/2/1.jpg"]', 99.00, 129.00, 200, 800, 1, NOW(), NOW()),
(3, 3, '无线遥控跳蛋', '10米遥控距离，12种模式', 'https://example.com/product/3/main.jpg', '["https://example.com/product/3/1.jpg"]', 159.00, 199.00, 150, 300, 1, NOW(), NOW()),
(4, 5, '超薄安全套 10只装', '001超薄，玻尿酸润滑', 'https://example.com/product/4/main.jpg', '["https://example.com/product/4/1.jpg"]', 49.90, 69.00, 500, 2000, 1, NOW(), NOW()),
(5, 6, '水溶性润滑液 100ml', '水溶性配方，温和不刺激', 'https://example.com/product/5/main.jpg', '["https://example.com/product/5/1.jpg"]', 39.90, 49.00, 300, 1500, 1, NOW(), NOW());

SELECT setval('products_id_seq', (SELECT MAX(id) FROM products));
```

### 008 订单数据

```sql
-- migrations/seeds/008_seed_orders.sql
-- 订单数据（示例订单）

-- 订单
INSERT INTO orders (id, order_no, user_id, type, original_amount, discount_amount, actual_amount, deposit_amount, status, created_at, updated_at) VALUES
(1, 'R20260101000001', 1, 'rental', 29.90, 0.00, 29.90, 99.00, 'completed', NOW() - INTERVAL '7 days', NOW() - INTERVAL '6 days'),
(2, 'R20260101000002', 3, 'rental', 59.90, 5.99, 53.91, 99.00, 'in_progress', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours'),
(3, 'M20260101000001', 1, 'mall', 248.90, 20.00, 228.90, 0.00, 'paid', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
(4, 'B20260101000001', 5, 'booking', 68.00, 0.00, 68.00, 0.00, 'completed', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days');

SELECT setval('orders_id_seq', (SELECT MAX(id) FROM orders));

-- 租借记录
INSERT INTO rentals (id, order_id, user_id, device_id, duration_hours, start_time, end_time, actual_end_time, rental_fee, deposit, overtime_fee, status, created_at, updated_at) VALUES
(1, 1, 1, 1, 2, NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days' + INTERVAL '2 hours', NOW() - INTERVAL '7 days' + INTERVAL '1 hour 50 minutes', 29.90, 99.00, 0.00, 'returned', NOW() - INTERVAL '7 days', NOW() - INTERVAL '6 days'),
(2, 2, 3, 2, 6, NOW() - INTERVAL '2 hours', NOW() + INTERVAL '4 hours', NULL, 59.90, 99.00, 0.00, 'renting', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours');

SELECT setval('rentals_id_seq', (SELECT MAX(id) FROM rentals));

-- 支付记录
INSERT INTO payments (id, order_id, payment_no, channel, amount, status, paid_at, created_at, updated_at) VALUES
(1, 1, 'PAY20260101000001', 'wechat', 128.90, 'success', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days'),
(2, 2, 'PAY20260101000002', 'wechat', 152.91, 'success', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours'),
(3, 3, 'PAY20260101000003', 'alipay', 228.90, 'success', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day');

SELECT setval('payments_id_seq', (SELECT MAX(id) FROM payments));
```

### 009 营销数据

```sql
-- migrations/seeds/009_seed_marketing.sql
-- 营销数据（优惠券、活动）

-- 优惠券模板
INSERT INTO coupons (id, name, type, value, min_amount, max_discount, total_count, used_count, start_time, end_time, status, created_at, updated_at) VALUES
(1, '新人专享10元券', 'fixed', 10.00, 50.00, NULL, 10000, 500, NOW() - INTERVAL '30 days', NOW() + INTERVAL '60 days', 1, NOW(), NOW()),
(2, '全场9折券', 'percentage', 10.00, 100.00, 50.00, 5000, 200, NOW() - INTERVAL '7 days', NOW() + INTERVAL '30 days', 1, NOW(), NOW()),
(3, '满200减30', 'fixed', 30.00, 200.00, NULL, 2000, 100, NOW(), NOW() + INTERVAL '14 days', 1, NOW(), NOW());

SELECT setval('coupons_id_seq', (SELECT MAX(id) FROM coupons));

-- 用户优惠券
INSERT INTO user_coupons (id, user_id, coupon_id, status, used_at, expire_at, created_at, updated_at) VALUES
(1, 1, 1, 'used', NOW() - INTERVAL '5 days', NOW() + INTERVAL '25 days', NOW() - INTERVAL '10 days', NOW() - INTERVAL '5 days'),
(2, 1, 2, 'unused', NULL, NOW() + INTERVAL '23 days', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
(3, 2, 1, 'unused', NULL, NOW() + INTERVAL '50 days', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
(4, 3, 3, 'unused', NULL, NOW() + INTERVAL '14 days', NOW(), NOW());

SELECT setval('user_coupons_id_seq', (SELECT MAX(id) FROM user_coupons));

-- 首页轮播图
INSERT INTO banners (id, title, image_url, link_type, link_url, sort_order, status, start_time, end_time, created_at, updated_at) VALUES
(1, '新年特惠', 'https://example.com/banner/1.jpg', 'product', '/product/1', 1, 1, NOW(), NOW() + INTERVAL '30 days', NOW(), NOW()),
(2, '会员日', 'https://example.com/banner/2.jpg', 'page', '/member', 2, 1, NOW(), NOW() + INTERVAL '7 days', NOW(), NOW()),
(3, '新品上市', 'https://example.com/banner/3.jpg', 'category', '/category/2', 3, 1, NOW(), NOW() + INTERVAL '14 days', NOW(), NOW());

SELECT setval('banners_id_seq', (SELECT MAX(id) FROM banners));
```

## 执行种子数据

### 命令行执行

```bash
# 执行所有种子数据
psql -U postgres -d smartlocker -f migrations/seeds/001_seed_basic.sql
psql -U postgres -d smartlocker -f migrations/seeds/002_seed_users.sql
# ... 依次执行

# 或使用脚本批量执行
for f in migrations/seeds/*.sql; do
    psql -U postgres -d smartlocker -f "$f"
done
```

### Makefile 集成

```makefile
DB_URL ?= postgres://postgres:password@localhost:5432/smartlocker?sslmode=disable

.PHONY: seed seed-all seed-clean

# 执行所有种子数据
seed-all:
	@for f in migrations/seeds/*.sql; do \
		echo "Executing $$f..."; \
		psql "$(DB_URL)" -f "$$f"; \
	done

# 清理并重新填充种子数据
seed-clean:
	@echo "Truncating all tables..."
	psql "$(DB_URL)" -c "TRUNCATE TABLE banners, user_coupons, coupons, payments, rentals, orders, products, product_categories, room_time_slots, rooms, hotels, merchants, venues, devices, admins, user_wallets, users, rental_pricings, roles, system_configs, member_levels RESTART IDENTITY CASCADE;"
	@$(MAKE) seed-all

# 开发环境一键初始化
dev-init: migrate-up seed-all
	@echo "Development database initialized!"
```

## 数据提取流程

### 从前端 Mock 提取数据

1. **定位 Mock 文件**
   ```bash
   # 查找所有 mock 数据文件
   find admin-frontend/src/mock -name "*.ts" -o -name "*.js"
   find user-frontend/src/mock -name "*.ts" -o -name "*.js"
   ```

2. **解析 Mock 数据结构**
   - 识别数据字段与数据库表的映射关系
   - 处理 TypeScript 类型定义
   - 转换日期格式和特殊字段

3. **生成 SQL INSERT 语句**
   - 遵循外键依赖顺序
   - 处理序列重置
   - 添加注释说明数据来源

### 字段映射规则

| Mock 字段类型 | PostgreSQL 类型 | 转换说明 |
|--------------|-----------------|----------|
| `string` | `VARCHAR`/`TEXT` | 直接映射，注意引号转义 |
| `number` | `INTEGER`/`DECIMAL` | 金额保留2位小数 |
| `boolean` | `BOOLEAN` | `true`→`TRUE`, `false`→`FALSE` |
| `Date`/`timestamp` | `TIMESTAMP` | 使用 `NOW()` 或相对时间 |
| `array` | `JSONB` | 使用 `'["a","b"]'` 格式 |
| `object` | `JSONB` | 使用 `'{}'::jsonb` 格式 |
| `null` | `NULL` | 直接使用 `NULL` |

## 注意事项

1. **外键依赖顺序**: 种子数据文件按数字顺序执行，确保被依赖的表先填充
2. **序列重置**: 每个表插入后需重置序列，避免后续插入主键冲突
3. **密码处理**: 管理员密码使用 bcrypt hash，不存储明文
4. **敏感数据**: 测试数据使用虚构信息，不使用真实用户数据
5. **幂等性**: 种子脚本应支持重复执行，使用 `ON CONFLICT DO NOTHING` 或先清理再插入

## 参考文档

- 数据模型: `specs/001-smart-locker-backend/data-model.md`
- 任务清单: `specs/001-smart-locker-backend/tasks.md` (T026-1 ~ T026-9)
- 前端 Mock: `admin-frontend/src/mock/`, `user-frontend/src/mock/`
