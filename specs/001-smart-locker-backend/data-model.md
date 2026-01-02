# Data Model: 爱上杜美人智能开锁管理系统

**Date**: 2026-01-02 | **Branch**: `001-smart-locker-backend`

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              USER DOMAIN                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────┐      ┌──────────────┐      ┌─────────────┐                     │
│  │  User   │─────>│  UserWallet  │      │ MemberLevel │                     │
│  └────┬────┘      └──────────────┘      └─────────────┘                     │
│       │                                        ▲                             │
│       ├────────────────────────────────────────┘                             │
│       │                                                                      │
│       ├─────>┌─────────────┐      ┌─────────────┐                           │
│       │      │   Address   │      │UserFeedback │                           │
│       │      └─────────────┘      └─────────────┘                           │
│       │                                                                      │
│  ┌────┴────┐      ┌──────────────┐      ┌─────────────┐                     │
│  │  Admin  │─────>│     Role     │─────>│ Permission  │                     │
│  └─────────┘      └──────────────┘      └─────────────┘                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                             DEVICE DOMAIN                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────┐      ┌─────────┐      ┌───────────────┐                       │
│  │ Merchant │─────>│  Venue  │─────>│    Device     │                       │
│  └──────────┘      └─────────┘      └───────┬───────┘                       │
│                                             │                                │
│                                    ┌────────┴────────┐                       │
│                              ┌─────┴─────┐    ┌──────┴──────┐               │
│                              │DeviceLog  │    │DeviceMaint  │               │
│                              └───────────┘    └─────────────┘               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                             ORDER DOMAIN                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────┐      ┌───────────┐      ┌─────────────┐      ┌────────┐       │
│  │  Order  │─────>│ OrderItem │      │   Payment   │      │ Refund │       │
│  └────┬────┘      └───────────┘      └──────┬──────┘      └────────┘       │
│       │                                      │                               │
│       └──────────────────────────────────────┘                               │
│                                                                              │
│  ┌─────────┐      ┌───────────────┐                                         │
│  │ Rental  │─────>│ RentalPricing │                                         │
│  └─────────┘      └───────────────┘                                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                             HOTEL DOMAIN                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────┐      ┌─────────┐      ┌──────────────┐      ┌───────────┐     │
│  │  Hotel  │─────>│  Room   │─────>│ RoomTimeSlot │      │  Booking  │     │
│  └─────────┘      └────┬────┘      └──────────────┘      └───────────┘     │
│                        │                                        ▲           │
│                        └────────────────────────────────────────┘           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                             MALL DOMAIN                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────┐      ┌─────────┐      ┌───────────┐      ┌────────────┐       │
│  │ Category │─────>│ Product │<─────│ CartItem  │      │  Review    │       │
│  └──────────┘      └────┬────┘      └───────────┘      └────────────┘       │
│                         │                                                    │
│                         └─────>┌────────────┐                                │
│                                │ ProductSku │                                │
│                                └────────────┘                                │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                          DISTRIBUTION DOMAIN                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────┐      ┌────────────┐      ┌────────────┐                    │
│  │ Distributor │─────>│ Commission │      │ Withdrawal │                    │
│  └─────────────┘      └────────────┘      └────────────┘                    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                          MARKETING DOMAIN                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────┐      ┌────────────┐      ┌────────────┐                       │
│  │  Coupon  │─────>│ UserCoupon │      │  Campaign  │                       │
│  └──────────┘      └────────────┘      └────────────┘                       │
│                                                                              │
│  ┌────────────────┐                                                          │
│  │ MemberPackage  │                                                          │
│  └────────────────┘                                                          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                          SYSTEM DOMAIN                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌─────────┐   ┌────────┐            │
│  │ SystemConfig │   │ OperationLog │   │ SmsCode │   │ Banner │            │
│  └──────────────┘   └──────────────┘   └─────────┘   └────────┘            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 1. User Domain

### 1.1 User（用户）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 用户ID |
| phone | VARCHAR(20) | UNIQUE, INDEX | 手机号 |
| openid | VARCHAR(64) | UNIQUE, INDEX, NULLABLE | 微信OpenID |
| unionid | VARCHAR(64) | NULLABLE | 微信UnionID |
| nickname | VARCHAR(50) | NOT NULL | 昵称 |
| avatar | VARCHAR(255) | NULLABLE | 头像URL |
| gender | TINYINT | DEFAULT 0 | 性别: 0未知 1男 2女 |
| birthday | DATE | NULLABLE | 生日 |
| member_level_id | BIGINT | FK, DEFAULT 1 | 会员等级ID |
| points | INT | DEFAULT 0 | 积分 |
| is_verified | BOOLEAN | DEFAULT FALSE | 是否实名认证 |
| real_name_encrypted | TEXT | NULLABLE | 加密的真实姓名 |
| id_card_encrypted | TEXT | NULLABLE | 加密的身份证号 |
| referrer_id | BIGINT | FK, NULLABLE, INDEX | 推荐人用户ID |
| status | TINYINT | DEFAULT 1 | 状态: 0禁用 1正常 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_user_phone` ON (phone)
- `idx_user_openid` ON (openid)
- `idx_user_referrer` ON (referrer_id)

**State Transitions**:
```
正常(1) ←→ 禁用(0)
```

---

### 1.2 UserWallet（用户钱包）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 钱包ID |
| user_id | BIGINT | FK, UNIQUE | 用户ID |
| balance | DECIMAL(12,2) | DEFAULT 0.00 | 余额 |
| frozen_balance | DECIMAL(12,2) | DEFAULT 0.00 | 冻结余额（押金） |
| total_recharged | DECIMAL(12,2) | DEFAULT 0.00 | 累计充值 |
| total_consumed | DECIMAL(12,2) | DEFAULT 0.00 | 累计消费 |
| total_withdrawn | DECIMAL(12,2) | DEFAULT 0.00 | 累计提现 |
| version | INT | DEFAULT 0 | 乐观锁版本号 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Business Rules**:
- 余额变更使用乐观锁保证并发安全
- 押金冻结在 frozen_balance 中

---

### 1.3 WalletTransaction（钱包交易记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 记录ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| type | VARCHAR(20) | NOT NULL | 类型: recharge/consume/refund/withdraw/deposit/return_deposit |
| amount | DECIMAL(12,2) | NOT NULL | 金额 |
| balance_before | DECIMAL(12,2) | NOT NULL | 变动前余额 |
| balance_after | DECIMAL(12,2) | NOT NULL | 变动后余额 |
| order_no | VARCHAR(64) | NULLABLE, INDEX | 关联订单号 |
| remark | VARCHAR(255) | NULLABLE | 备注 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 1.4 MemberLevel（会员等级）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 等级ID |
| name | VARCHAR(50) | NOT NULL | 等级名称 |
| level | INT | UNIQUE | 等级序号 |
| min_points | INT | NOT NULL | 所需积分下限 |
| discount | DECIMAL(3,2) | DEFAULT 1.00 | 折扣率 (0.80 = 8折) |
| benefits | JSON | NULLABLE | 权益描述 |
| icon | VARCHAR(255) | NULLABLE | 等级图标 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 1.5 Admin（管理员）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 管理员ID |
| username | VARCHAR(50) | UNIQUE | 用户名 |
| password_hash | VARCHAR(255) | NOT NULL | 密码哈希 |
| name | VARCHAR(50) | NOT NULL | 姓名 |
| phone | VARCHAR(20) | NULLABLE | 手机号 |
| email | VARCHAR(100) | NULLABLE | 邮箱 |
| role_id | BIGINT | FK | 角色ID |
| merchant_id | BIGINT | FK, NULLABLE | 关联商户ID (合作商角色) |
| status | TINYINT | DEFAULT 1 | 状态: 0禁用 1正常 |
| last_login_at | TIMESTAMP | NULLABLE | 最后登录时间 |
| last_login_ip | VARCHAR(45) | NULLABLE | 最后登录IP |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

---

### 1.6 Role（角色）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 角色ID |
| code | VARCHAR(50) | UNIQUE | 角色编码 |
| name | VARCHAR(50) | NOT NULL | 角色名称 |
| description | VARCHAR(255) | NULLABLE | 角色描述 |
| is_system | BOOLEAN | DEFAULT FALSE | 是否系统角色 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

**预置角色**:
- super_admin: 超级管理员
- platform_admin: 平台管理员
- operation_admin: 运营管理员
- finance_admin: 财务管理员
- partner: 合作商
- customer_service: 客服

---

### 1.7 Permission（权限）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 权限ID |
| code | VARCHAR(100) | UNIQUE | 权限编码 |
| name | VARCHAR(100) | NOT NULL | 权限名称 |
| type | VARCHAR(20) | NOT NULL | 类型: menu/api |
| parent_id | BIGINT | FK, NULLABLE | 父权限ID |
| path | VARCHAR(255) | NULLABLE | API路径或菜单路径 |
| method | VARCHAR(10) | NULLABLE | HTTP方法 |
| sort | INT | DEFAULT 0 | 排序 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 1.8 RolePermission（角色权限关联）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| role_id | BIGINT | PK, FK | 角色ID |
| permission_id | BIGINT | PK, FK | 权限ID |

---

### 1.9 UserFeedback（用户反馈）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 反馈ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| type | VARCHAR(20) | NOT NULL | 类型: suggestion/bug/complaint/other |
| content | TEXT | NOT NULL | 反馈内容 |
| images | JSON | NULLABLE | 图片列表 |
| contact | VARCHAR(100) | NULLABLE | 联系方式 |
| status | TINYINT | DEFAULT 0 | 状态: 0待处理 1处理中 2已处理 |
| reply | TEXT | NULLABLE | 回复内容 |
| replied_by | BIGINT | FK, NULLABLE | 回复管理员ID |
| replied_at | TIMESTAMP | NULLABLE | 回复时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 1.10 Address（用户收货地址）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 地址ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| receiver_name | VARCHAR(50) | NOT NULL | 收货人姓名 |
| receiver_phone | VARCHAR(20) | NOT NULL | 收货人电话 |
| province | VARCHAR(50) | NOT NULL | 省份 |
| city | VARCHAR(50) | NOT NULL | 城市 |
| district | VARCHAR(50) | NOT NULL | 区县 |
| detail | VARCHAR(255) | NOT NULL | 详细地址 |
| postal_code | VARCHAR(10) | NULLABLE | 邮编 |
| is_default | BOOLEAN | DEFAULT FALSE | 是否默认地址 |
| tag | VARCHAR(20) | NULLABLE | 地址标签: home/company/other |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_address_user` ON (user_id)
- `idx_address_user_default` ON (user_id, is_default)

**Business Rules**:
- 每个用户只能有一个默认地址
- 设置新默认地址时，需取消原默认地址

---

## 2. Device Domain

### 2.1 Merchant（商户）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 商户ID |
| name | VARCHAR(100) | NOT NULL | 商户名称 |
| contact_name | VARCHAR(50) | NOT NULL | 联系人 |
| contact_phone | VARCHAR(20) | NOT NULL | 联系电话 |
| address | VARCHAR(255) | NULLABLE | 地址 |
| business_license | VARCHAR(255) | NULLABLE | 营业执照图片 |
| commission_rate | DECIMAL(5,4) | DEFAULT 0.2000 | 分成比例 (0.20 = 20%) |
| settlement_type | VARCHAR(20) | DEFAULT 'monthly' | 结算周期: weekly/monthly |
| bank_name | VARCHAR(100) | NULLABLE | 银行名称 |
| bank_account_encrypted | TEXT | NULLABLE | 加密的银行账号 |
| bank_holder_encrypted | TEXT | NULLABLE | 加密的持卡人姓名 |
| status | TINYINT | DEFAULT 1 | 状态: 0禁用 1正常 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

---

### 2.2 Venue（场地）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 场地ID |
| merchant_id | BIGINT | FK, INDEX | 商户ID |
| name | VARCHAR(100) | NOT NULL | 场地名称 |
| type | VARCHAR(20) | NOT NULL | 类型: mall/hotel/community/office/other |
| province | VARCHAR(50) | NOT NULL | 省份 |
| city | VARCHAR(50) | NOT NULL | 城市 |
| district | VARCHAR(50) | NOT NULL | 区县 |
| address | VARCHAR(255) | NOT NULL | 详细地址 |
| longitude | DECIMAL(10,7) | NULLABLE | 经度 |
| latitude | DECIMAL(10,7) | NULLABLE | 纬度 |
| contact_name | VARCHAR(50) | NULLABLE | 联系人 |
| contact_phone | VARCHAR(20) | NULLABLE | 联系电话 |
| status | TINYINT | DEFAULT 1 | 状态: 0禁用 1正常 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_venue_merchant` ON (merchant_id)
- `idx_venue_location` ON (province, city, district)

---

### 2.3 Device（智能柜设备）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 设备ID |
| device_no | VARCHAR(64) | UNIQUE | 设备编号 |
| name | VARCHAR(100) | NOT NULL | 设备名称 |
| type | VARCHAR(20) | NOT NULL | 设备类型: standard/mini/premium |
| model | VARCHAR(50) | NULLABLE | 设备型号 |
| venue_id | BIGINT | FK, INDEX | 场地ID |
| qr_code | VARCHAR(255) | NOT NULL | 设备二维码URL |
| product_name | VARCHAR(100) | NOT NULL | 柜内产品名称 |
| product_image | VARCHAR(255) | NULLABLE | 产品图片 |
| slot_count | INT | DEFAULT 1 | 格子数量 |
| available_slots | INT | DEFAULT 1 | 可用格子数 |
| online_status | TINYINT | DEFAULT 0 | 在线状态: 0离线 1在线 |
| lock_status | TINYINT | DEFAULT 0 | 锁状态: 0锁定 1打开 |
| rental_status | TINYINT | DEFAULT 0 | 租借状态: 0空闲 1使用中 |
| current_rental_id | BIGINT | FK, NULLABLE | 当前租借记录ID |
| firmware_version | VARCHAR(20) | NULLABLE | 固件版本 |
| network_type | VARCHAR(20) | DEFAULT 'WiFi' | 网络类型: WiFi/4G/Ethernet |
| signal_strength | INT | NULLABLE | 信号强度(0-100) |
| battery_level | INT | NULLABLE | 电量百分比(0-100) |
| temperature | DECIMAL(5,2) | NULLABLE | 温度(℃) |
| humidity | DECIMAL(5,2) | NULLABLE | 湿度(%) |
| last_heartbeat_at | TIMESTAMP | NULLABLE | 最后心跳时间 |
| last_online_at | TIMESTAMP | NULLABLE | 最后上线时间 |
| last_offline_at | TIMESTAMP | NULLABLE | 最后离线时间 |
| install_time | TIMESTAMP | NULLABLE | 安装时间 |
| status | TINYINT | DEFAULT 1 | 状态: 0禁用 1正常 2维护中 3故障 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_device_venue` ON (venue_id)
- `idx_device_status` ON (online_status, rental_status)
- `idx_device_type` ON (type)

**State Transitions**:
```
租借状态:
空闲(0) → 使用中(1) [用户支付开锁]
使用中(1) → 空闲(0) [用户归还或超时购买]

在线状态:
离线(0) ←→ 在线(1) [心跳超时/收到心跳]

设备状态:
正常(1) → 维护中(2) [管理员设置维护]
正常(1) → 故障(3) [设备上报异常]
维护中(2) → 正常(1) [维护完成]
故障(3) → 正常(1) [故障修复]
```

---

### 2.4 DeviceLog（设备日志）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 日志ID |
| device_id | BIGINT | FK, INDEX | 设备ID |
| type | VARCHAR(20) | NOT NULL | 类型: online/offline/unlock/lock/error/heartbeat |
| content | TEXT | NULLABLE | 日志内容 |
| operator_id | BIGINT | NULLABLE | 操作人ID |
| operator_type | VARCHAR(10) | NULLABLE | 操作人类型: user/admin/system |
| created_at | TIMESTAMP | DEFAULT NOW(), INDEX | 创建时间 |

**Indexes**:
- `idx_device_log_device_time` ON (device_id, created_at DESC)

---

### 2.5 DeviceMaintenance（设备维护记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 记录ID |
| device_id | BIGINT | FK, INDEX | 设备ID |
| type | VARCHAR(20) | NOT NULL | 类型: repair/clean/replace/inspect |
| description | TEXT | NOT NULL | 维护描述 |
| before_images | JSON | NULLABLE | 维护前图片 |
| after_images | JSON | NULLABLE | 维护后图片 |
| cost | DECIMAL(10,2) | DEFAULT 0 | 维护成本 |
| operator_id | BIGINT | FK | 操作人ID |
| status | TINYINT | DEFAULT 0 | 状态: 0进行中 1已完成 |
| started_at | TIMESTAMP | NOT NULL | 开始时间 |
| completed_at | TIMESTAMP | NULLABLE | 完成时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

## 3. Order Domain

### 3.1 Order（订单）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 订单ID |
| order_no | VARCHAR(64) | UNIQUE | 订单号 |
| user_id | BIGINT | FK, INDEX | 用户ID |
| type | VARCHAR(20) | NOT NULL, INDEX | 类型: rental/hotel/mall |
| original_amount | DECIMAL(12,2) | NOT NULL | 原价 |
| discount_amount | DECIMAL(12,2) | DEFAULT 0 | 优惠金额 |
| actual_amount | DECIMAL(12,2) | NOT NULL | 实付金额 |
| deposit_amount | DECIMAL(12,2) | DEFAULT 0 | 押金金额 |
| status | VARCHAR(20) | NOT NULL, INDEX | 状态 |
| coupon_id | BIGINT | FK, NULLABLE | 使用的优惠券ID |
| remark | VARCHAR(255) | NULLABLE | 备注 |
| address_id | BIGINT | FK, NULLABLE | 收货地址ID (商城订单) |
| address_snapshot | JSON | NULLABLE | 收货地址快照 |
| express_company | VARCHAR(50) | NULLABLE | 快递公司 |
| express_no | VARCHAR(64) | NULLABLE | 快递单号 |
| shipped_at | TIMESTAMP | NULLABLE | 发货时间 |
| received_at | TIMESTAMP | NULLABLE | 收货时间 |
| paid_at | TIMESTAMP | NULLABLE | 支付时间 |
| completed_at | TIMESTAMP | NULLABLE | 完成时间 |
| cancelled_at | TIMESTAMP | NULLABLE | 取消时间 |
| cancel_reason | VARCHAR(255) | NULLABLE | 取消原因 |
| created_at | TIMESTAMP | DEFAULT NOW(), INDEX | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Order Status**:
- pending: 待支付
- paid: 已支付
- pending_ship: 待发货 (商城订单)
- shipped: 已发货 (商城订单)
- in_progress: 进行中 (租借/酒店订单)
- completed: 已完成
- cancelled: 已取消
- refunding: 退款中
- refunded: 已退款

**Indexes**:
- `idx_order_user` ON (user_id)
- `idx_order_type_status` ON (type, status)
- `idx_order_created` ON (created_at DESC)
- `idx_order_express` ON (express_no)

---

### 3.2 OrderItem（订单项）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 记录ID |
| order_id | BIGINT | FK, INDEX | 订单ID |
| product_id | BIGINT | FK, NULLABLE | 商品ID (商城订单) |
| product_name | VARCHAR(100) | NOT NULL | 商品名称 |
| product_image | VARCHAR(255) | NULLABLE | 商品图片 |
| sku_info | VARCHAR(255) | NULLABLE | SKU信息 |
| price | DECIMAL(12,2) | NOT NULL | 单价 |
| quantity | INT | NOT NULL | 数量 |
| subtotal | DECIMAL(12,2) | NOT NULL | 小计 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 3.3 Payment（支付记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 支付ID |
| payment_no | VARCHAR(64) | UNIQUE | 支付流水号 |
| order_id | BIGINT | FK, INDEX | 订单ID |
| order_no | VARCHAR(64) | NOT NULL, INDEX | 订单号 |
| user_id | BIGINT | FK, INDEX | 用户ID |
| channel | VARCHAR(20) | NOT NULL | 支付渠道: wechat/alipay/wallet |
| amount | DECIMAL(12,2) | NOT NULL | 支付金额 |
| status | VARCHAR(20) | NOT NULL | 状态: pending/success/failed/refunded |
| trade_no | VARCHAR(64) | NULLABLE | 第三方交易号 |
| pay_time | TIMESTAMP | NULLABLE | 支付时间 |
| callback_data | JSON | NULLABLE | 回调数据 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

---

### 3.4 Refund（退款记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 退款ID |
| refund_no | VARCHAR(64) | UNIQUE | 退款流水号 |
| order_id | BIGINT | FK, INDEX | 订单ID |
| payment_id | BIGINT | FK | 支付ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| amount | DECIMAL(12,2) | NOT NULL | 退款金额 |
| reason | VARCHAR(255) | NOT NULL | 退款原因 |
| status | VARCHAR(20) | NOT NULL | 状态: pending/processing/success/failed |
| trade_refund_no | VARCHAR(64) | NULLABLE | 第三方退款号 |
| operator_id | BIGINT | FK, NULLABLE | 操作人ID |
| processed_at | TIMESTAMP | NULLABLE | 处理时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

---

### 3.5 Rental（租借记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 租借ID |
| order_id | BIGINT | FK, UNIQUE | 关联订单ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| device_id | BIGINT | FK, INDEX | 设备ID |
| duration_hours | INT | NOT NULL | 租借时长(小时) |
| rental_fee | DECIMAL(10,2) | NOT NULL | 租金 |
| deposit | DECIMAL(10,2) | NOT NULL | 押金 |
| overtime_rate | DECIMAL(10,2) | NOT NULL | 超时费率(元/小时) |
| overtime_fee | DECIMAL(10,2) | DEFAULT 0 | 超时费 |
| status | VARCHAR(20) | NOT NULL, INDEX | 状态 |
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

**Business Rules**:
- 超过24小时未归还，自动转为购买
- 押金扣除后，订单状态变更为"已购买"

---

### 3.6 RentalPricing（租借定价）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 定价ID |
| venue_id | BIGINT | FK, NULLABLE | 场地ID (NULL表示默认) |
| duration_hours | INT | NOT NULL | 时长(小时) |
| price | DECIMAL(10,2) | NOT NULL | 价格 |
| deposit | DECIMAL(10,2) | NOT NULL | 押金 |
| overtime_rate | DECIMAL(10,2) | NOT NULL | 超时费率(元/小时) |
| is_active | BOOLEAN | DEFAULT TRUE | 是否启用 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_pricing_venue_duration` UNIQUE ON (venue_id, duration_hours)

---

## 4. Hotel Domain

### 4.1 Hotel（酒店）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 酒店ID |
| name | VARCHAR(100) | NOT NULL | 酒店名称 |
| star_rating | TINYINT | NULLABLE | 星级: 1-5 |
| province | VARCHAR(50) | NOT NULL | 省份 |
| city | VARCHAR(50) | NOT NULL | 城市 |
| district | VARCHAR(50) | NOT NULL | 区县 |
| address | VARCHAR(255) | NOT NULL | 详细地址 |
| longitude | DECIMAL(10,7) | NULLABLE | 经度 |
| latitude | DECIMAL(10,7) | NULLABLE | 纬度 |
| phone | VARCHAR(20) | NOT NULL | 联系电话 |
| images | JSON | NULLABLE | 酒店图片列表 |
| facilities | JSON | NULLABLE | 设施列表 |
| description | TEXT | NULLABLE | 酒店描述 |
| check_in_time | TIME | DEFAULT '14:00' | 入住时间 |
| check_out_time | TIME | DEFAULT '12:00' | 退房时间 |
| commission_rate | DECIMAL(5,4) | DEFAULT 0.1500 | 分成比例 |
| status | TINYINT | DEFAULT 1 | 状态: 0下架 1上架 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

---

### 4.2 Room（房间）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 房间ID |
| hotel_id | BIGINT | FK, INDEX | 酒店ID |
| room_no | VARCHAR(20) | NOT NULL | 房间号 |
| room_type | VARCHAR(50) | NOT NULL | 房型名称 |
| device_id | BIGINT | FK, NULLABLE | 关联智能柜设备ID |
| images | JSON | NULLABLE | 房间图片 |
| facilities | JSON | NULLABLE | 房间设施 |
| area | INT | NULLABLE | 面积(平方米) |
| bed_type | VARCHAR(50) | NULLABLE | 床型 |
| max_guests | INT | DEFAULT 2 | 最大入住人数 |
| hourly_price | DECIMAL(10,2) | NOT NULL | 钟点价 |
| daily_price | DECIMAL(10,2) | NOT NULL | 全日价 |
| status | TINYINT | DEFAULT 1 | 状态: 0停用 1可用 2已预订 3使用中 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_room_hotel` ON (hotel_id)
- `idx_room_device` ON (device_id)

---

### 4.3 RoomTimeSlot（房间时段价格）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 时段ID |
| room_id | BIGINT | FK, INDEX | 房间ID |
| duration_hours | INT | NOT NULL | 时长(小时): 1/2/3/6/12/24 |
| price | DECIMAL(10,2) | NOT NULL | 价格 |
| start_time | TIME | NULLABLE | 可用开始时间 |
| end_time | TIME | NULLABLE | 可用结束时间 |
| is_active | BOOLEAN | DEFAULT TRUE | 是否启用 |
| sort | INT | DEFAULT 0 | 排序 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_timeslot_room` ON (room_id)
- `idx_timeslot_room_duration` UNIQUE ON (room_id, duration_hours)

**Business Rules**:
- 同一房间同一时长只能有一个价格配置
- 前端展示时按 sort 排序

---

### 4.4 Booking（预订记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 预订ID |
| booking_no | VARCHAR(64) | UNIQUE | 预订号 |
| order_id | BIGINT | FK, UNIQUE | 关联订单ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| hotel_id | BIGINT | FK, INDEX | 酒店ID |
| room_id | BIGINT | FK, INDEX | 房间ID |
| device_id | BIGINT | FK, NULLABLE | 关联智能柜设备ID |
| check_in_time | TIMESTAMP | NOT NULL | 入住时间 |
| check_out_time | TIMESTAMP | NOT NULL | 退房时间 |
| duration_hours | INT | NOT NULL | 时长(小时) |
| amount | DECIMAL(10,2) | NOT NULL | 金额 |
| verification_code | VARCHAR(20) | NOT NULL | 核销码 |
| unlock_code | VARCHAR(10) | NOT NULL | 开锁码 |
| qr_code | VARCHAR(255) | NOT NULL | 核销二维码URL |
| status | VARCHAR(20) | NOT NULL, INDEX | 状态 |
| verified_at | TIMESTAMP | NULLABLE | 核销时间 |
| verified_by | BIGINT | FK, NULLABLE | 核销人ID |
| unlocked_at | TIMESTAMP | NULLABLE | 开锁时间 |
| completed_at | TIMESTAMP | NULLABLE | 完成时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Booking Status**:
- pending: 待支付
- paid: 已支付/待核销
- verified: 已核销/待使用
- in_use: 使用中
- completed: 已完成
- cancelled: 已取消
- refunded: 已退款
- expired: 已过期

**Business Rules**:
- 开锁码仅在 check_in_time 至 check_out_time 时段内有效
- 核销后，开锁码方可使用
- 超过入住时间未核销，状态变为 expired

---

## 5. Mall Domain

### 5.1 Category（商品分类）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 分类ID |
| parent_id | BIGINT | FK, NULLABLE, INDEX | 父分类ID |
| name | VARCHAR(50) | NOT NULL | 分类名称 |
| icon | VARCHAR(255) | NULLABLE | 分类图标 |
| sort | INT | DEFAULT 0 | 排序 |
| level | TINYINT | DEFAULT 1 | 层级: 1/2/3 |
| is_active | BOOLEAN | DEFAULT TRUE | 是否启用 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 5.2 Product（商品）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 商品ID |
| category_id | BIGINT | FK, INDEX | 分类ID |
| name | VARCHAR(100) | NOT NULL | 商品名称 |
| subtitle | VARCHAR(255) | NULLABLE | 副标题 |
| images | JSON | NOT NULL | 商品图片列表 |
| description | TEXT | NULLABLE | 商品描述 |
| price | DECIMAL(10,2) | NOT NULL | 销售价 |
| original_price | DECIMAL(10,2) | NULLABLE | 原价 |
| stock | INT | DEFAULT 0 | 库存 |
| sales | INT | DEFAULT 0 | 销量 |
| unit | VARCHAR(20) | DEFAULT '件' | 单位 |
| is_on_sale | BOOLEAN | DEFAULT TRUE | 是否上架 |
| is_hot | BOOLEAN | DEFAULT FALSE | 是否热门 |
| is_new | BOOLEAN | DEFAULT FALSE | 是否新品 |
| sort | INT | DEFAULT 0 | 排序 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_product_category` ON (category_id)
- `idx_product_sale` ON (is_on_sale, sort DESC)

---

### 5.3 ProductSku（商品SKU）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | SKU ID |
| product_id | BIGINT | FK, INDEX | 商品ID |
| sku_code | VARCHAR(64) | UNIQUE | SKU编码 |
| attributes | JSON | NOT NULL | SKU属性 {"颜色": "红色", "尺码": "M"} |
| price | DECIMAL(10,2) | NOT NULL | SKU价格 |
| stock | INT | DEFAULT 0 | SKU库存 |
| image | VARCHAR(255) | NULLABLE | SKU图片 |
| is_active | BOOLEAN | DEFAULT TRUE | 是否启用 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 5.4 CartItem（购物车项）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 记录ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| product_id | BIGINT | FK | 商品ID |
| sku_id | BIGINT | FK, NULLABLE | SKU ID |
| quantity | INT | NOT NULL | 数量 |
| selected | BOOLEAN | DEFAULT TRUE | 是否选中 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_cart_user` ON (user_id)
- `idx_cart_user_product` UNIQUE ON (user_id, product_id, sku_id)

---

### 5.5 Review（商品评价）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 评价ID |
| order_id | BIGINT | FK, INDEX | 订单ID |
| product_id | BIGINT | FK, INDEX | 商品ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| rating | TINYINT | NOT NULL | 评分: 1-5 |
| content | TEXT | NULLABLE | 评价内容 |
| images | JSON | NULLABLE | 评价图片 |
| is_anonymous | BOOLEAN | DEFAULT FALSE | 是否匿名 |
| reply | TEXT | NULLABLE | 商家回复 |
| replied_at | TIMESTAMP | NULLABLE | 回复时间 |
| status | TINYINT | DEFAULT 1 | 状态: 0隐藏 1显示 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

## 6. Distribution Domain

### 6.1 Distributor（分销商）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 分销商ID |
| user_id | BIGINT | FK, UNIQUE | 关联用户ID |
| parent_id | BIGINT | FK, NULLABLE, INDEX | 上级分销商ID |
| level | TINYINT | DEFAULT 1 | 层级: 1直推 2间推 |
| invite_code | VARCHAR(20) | UNIQUE | 邀请码 |
| total_commission | DECIMAL(12,2) | DEFAULT 0 | 累计佣金 |
| available_commission | DECIMAL(12,2) | DEFAULT 0 | 可提现佣金 |
| frozen_commission | DECIMAL(12,2) | DEFAULT 0 | 冻结佣金 |
| withdrawn_commission | DECIMAL(12,2) | DEFAULT 0 | 已提现佣金 |
| team_count | INT | DEFAULT 0 | 团队人数 |
| direct_count | INT | DEFAULT 0 | 直推人数 |
| status | TINYINT | DEFAULT 0 | 状态: 0待审核 1已通过 2已拒绝 |
| approved_at | TIMESTAMP | NULLABLE | 审核通过时间 |
| approved_by | BIGINT | FK, NULLABLE | 审核人ID |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

---

### 6.2 Commission（佣金记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 记录ID |
| distributor_id | BIGINT | FK, INDEX | 分销商ID |
| order_id | BIGINT | FK, INDEX | 订单ID |
| from_user_id | BIGINT | FK | 消费用户ID |
| type | VARCHAR(20) | NOT NULL | 类型: direct/indirect |
| order_amount | DECIMAL(12,2) | NOT NULL | 订单实付金额 |
| rate | DECIMAL(5,4) | NOT NULL | 佣金比例 |
| amount | DECIMAL(12,2) | NOT NULL | 佣金金额 |
| status | TINYINT | DEFAULT 0 | 状态: 0待结算 1已结算 2已失效 |
| settled_at | TIMESTAMP | NULLABLE | 结算时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

**Business Rules**:
- 佣金按订单实付金额计算
- 订单完成后佣金进入待结算状态
- 退款订单佣金失效

---

### 6.3 Withdrawal（提现申请）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 提现ID |
| withdrawal_no | VARCHAR(64) | UNIQUE | 提现单号 |
| user_id | BIGINT | FK, INDEX | 用户ID |
| type | VARCHAR(20) | NOT NULL | 类型: wallet/commission |
| amount | DECIMAL(12,2) | NOT NULL | 提现金额 |
| fee | DECIMAL(10,2) | DEFAULT 0 | 手续费 |
| actual_amount | DECIMAL(12,2) | NOT NULL | 实际到账金额 |
| withdraw_to | VARCHAR(20) | NOT NULL | 提现方式: wechat/alipay/bank |
| account_info_encrypted | TEXT | NOT NULL | 加密的账户信息 |
| status | VARCHAR(20) | NOT NULL | 状态 |
| operator_id | BIGINT | FK, NULLABLE | 操作人ID |
| processed_at | TIMESTAMP | NULLABLE | 处理时间 |
| reject_reason | VARCHAR(255) | NULLABLE | 拒绝原因 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Withdrawal Status**:
- pending: 待审核
- approved: 已通过
- processing: 打款中
- success: 已完成
- rejected: 已拒绝

---

## 7. Marketing Domain

### 7.1 Coupon（优惠券模板）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 优惠券ID |
| name | VARCHAR(100) | NOT NULL | 优惠券名称 |
| type | VARCHAR(20) | NOT NULL | 类型: fixed/percent |
| value | DECIMAL(10,2) | NOT NULL | 面额/折扣率 |
| min_amount | DECIMAL(10,2) | DEFAULT 0 | 最低消费金额 |
| max_discount | DECIMAL(10,2) | NULLABLE | 最高优惠金额 |
| applicable_type | VARCHAR(20) | DEFAULT 'all' | 适用范围: all/rental/hotel/mall |
| total_count | INT | NOT NULL | 发放总量 |
| issued_count | INT | DEFAULT 0 | 已发放数量 |
| used_count | INT | DEFAULT 0 | 已使用数量 |
| per_user_limit | INT | DEFAULT 1 | 每人限领 |
| start_time | TIMESTAMP | NOT NULL | 开始时间 |
| end_time | TIMESTAMP | NOT NULL | 结束时间 |
| validity_days | INT | NULLABLE | 领取后有效天数 |
| status | TINYINT | DEFAULT 1 | 状态: 0禁用 1启用 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 7.2 UserCoupon（用户优惠券）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 记录ID |
| user_id | BIGINT | FK, INDEX | 用户ID |
| coupon_id | BIGINT | FK, INDEX | 优惠券ID |
| status | TINYINT | DEFAULT 0 | 状态: 0未使用 1已使用 2已过期 |
| order_id | BIGINT | FK, NULLABLE | 使用的订单ID |
| received_at | TIMESTAMP | DEFAULT NOW() | 领取时间 |
| expire_at | TIMESTAMP | NOT NULL | 过期时间 |
| used_at | TIMESTAMP | NULLABLE | 使用时间 |

---

### 7.3 Campaign（营销活动）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 活动ID |
| name | VARCHAR(100) | NOT NULL | 活动名称 |
| type | VARCHAR(20) | NOT NULL | 类型: discount/gift/points |
| description | TEXT | NULLABLE | 活动描述 |
| rules | JSON | NOT NULL | 活动规则 |
| start_time | TIMESTAMP | NOT NULL | 开始时间 |
| end_time | TIMESTAMP | NOT NULL | 结束时间 |
| status | TINYINT | DEFAULT 1 | 状态: 0禁用 1启用 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 7.4 MemberPackage（会员套餐）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 套餐ID |
| name | VARCHAR(100) | NOT NULL | 套餐名称 |
| target_level_id | BIGINT | FK | 目标会员等级ID |
| price | DECIMAL(10,2) | NOT NULL | 套餐价格 |
| duration_days | INT | NOT NULL | 有效天数 |
| bonus_points | INT | DEFAULT 0 | 赠送积分 |
| description | TEXT | NULLABLE | 套餐描述 |
| is_active | BOOLEAN | DEFAULT TRUE | 是否启用 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

## 8. Finance Domain

### 8.1 Settlement（结算记录）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 结算ID |
| settlement_no | VARCHAR(64) | UNIQUE | 结算单号 |
| type | VARCHAR(20) | NOT NULL | 类型: merchant/distributor |
| target_id | BIGINT | NOT NULL, INDEX | 商户ID或分销商ID |
| period_start | DATE | NOT NULL | 结算周期开始 |
| period_end | DATE | NOT NULL | 结算周期结束 |
| total_amount | DECIMAL(12,2) | NOT NULL | 结算总金额 |
| fee | DECIMAL(10,2) | DEFAULT 0 | 手续费 |
| actual_amount | DECIMAL(12,2) | NOT NULL | 实际结算金额 |
| order_count | INT | NOT NULL | 订单数量 |
| status | VARCHAR(20) | NOT NULL | 状态 |
| operator_id | BIGINT | FK, NULLABLE | 操作人ID |
| settled_at | TIMESTAMP | NULLABLE | 结算时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

**Settlement Status**:
- pending: 待结算
- processing: 结算中
- completed: 已完成
- failed: 结算失败

---

## 9. Content Domain

### 9.1 Article（文章）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 文章ID |
| category | VARCHAR(20) | NOT NULL, INDEX | 分类: help/faq/notice/about |
| title | VARCHAR(200) | NOT NULL | 标题 |
| content | TEXT | NOT NULL | 内容 |
| cover_image | VARCHAR(255) | NULLABLE | 封面图 |
| sort | INT | DEFAULT 0 | 排序 |
| view_count | INT | DEFAULT 0 | 浏览量 |
| is_published | BOOLEAN | DEFAULT TRUE | 是否发布 |
| published_at | TIMESTAMP | NULLABLE | 发布时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

---

### 9.2 Notification（通知）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 通知ID |
| user_id | BIGINT | FK, INDEX | 用户ID (NULL表示全体) |
| type | VARCHAR(20) | NOT NULL | 类型: system/order/marketing |
| title | VARCHAR(100) | NOT NULL | 标题 |
| content | TEXT | NOT NULL | 内容 |
| link | VARCHAR(255) | NULLABLE | 跳转链接 |
| is_read | BOOLEAN | DEFAULT FALSE | 是否已读 |
| read_at | TIMESTAMP | NULLABLE | 阅读时间 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

### 9.3 MessageTemplate（消息模板）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 模板ID |
| code | VARCHAR(50) | UNIQUE | 模板编码 |
| name | VARCHAR(100) | NOT NULL | 模板名称 |
| type | VARCHAR(20) | NOT NULL | 类型: sms/push/wechat |
| content | TEXT | NOT NULL | 模板内容 |
| variables | JSON | NULLABLE | 变量列表 |
| is_active | BOOLEAN | DEFAULT TRUE | 是否启用 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

---

## 10. System Domain

### 10.1 SystemConfig（系统配置）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 配置ID |
| group | VARCHAR(50) | NOT NULL, INDEX | 配置分组 |
| key | VARCHAR(100) | NOT NULL | 配置键 |
| value | TEXT | NOT NULL | 配置值 |
| type | VARCHAR(20) | DEFAULT 'string' | 值类型: string/number/boolean/json |
| description | VARCHAR(255) | NULLABLE | 描述 |
| is_public | BOOLEAN | DEFAULT FALSE | 是否公开 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_config_group_key` UNIQUE ON (group, key)

---

### 10.2 OperationLog（操作日志）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 日志ID |
| admin_id | BIGINT | FK, INDEX | 管理员ID |
| module | VARCHAR(50) | NOT NULL | 模块 |
| action | VARCHAR(50) | NOT NULL | 操作 |
| target_type | VARCHAR(50) | NULLABLE | 操作对象类型 |
| target_id | BIGINT | NULLABLE | 操作对象ID |
| before_data | JSON | NULLABLE | 操作前数据 |
| after_data | JSON | NULLABLE | 操作后数据 |
| ip | VARCHAR(45) | NOT NULL | 操作IP |
| user_agent | VARCHAR(255) | NULLABLE | User Agent |
| created_at | TIMESTAMP | DEFAULT NOW(), INDEX | 创建时间 |

**Indexes**:
- `idx_oplog_admin_time` ON (admin_id, created_at DESC)
- `idx_oplog_module_action` ON (module, action)

---

### 10.3 SmsCode（短信验证码）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | 记录ID |
| phone | VARCHAR(20) | NOT NULL, INDEX | 手机号 |
| code | VARCHAR(10) | NOT NULL | 验证码 |
| type | VARCHAR(20) | NOT NULL | 类型: login/register/bind/reset |
| expire_at | TIMESTAMP | NOT NULL | 过期时间 |
| is_used | BOOLEAN | DEFAULT FALSE | 是否已使用 |
| used_at | TIMESTAMP | NULLABLE | 使用时间 |
| ip | VARCHAR(45) | NULLABLE | 请求IP |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |

**Indexes**:
- `idx_smscode_phone_type` ON (phone, type)
- `idx_smscode_expire` ON (expire_at)

**Business Rules**:
- 验证码有效期5分钟
- 同一手机号每分钟最多发送1次
- 同一IP每小时最多发送10次

---

### 10.4 Banner（轮播图/广告位）

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | BIGINT | PK, AUTO_INCREMENT | Banner ID |
| title | VARCHAR(100) | NOT NULL | 标题 |
| image | VARCHAR(255) | NOT NULL | 图片URL |
| link_type | VARCHAR(20) | NULLABLE | 链接类型: url/product/hotel/activity/none |
| link_value | VARCHAR(255) | NULLABLE | 链接值 |
| position | VARCHAR(20) | NOT NULL, INDEX | 位置: home/mall/hotel |
| sort | INT | DEFAULT 0 | 排序(越大越靠前) |
| start_time | TIMESTAMP | NULLABLE | 开始展示时间 |
| end_time | TIMESTAMP | NULLABLE | 结束展示时间 |
| is_active | BOOLEAN | DEFAULT TRUE | 是否启用 |
| click_count | INT | DEFAULT 0 | 点击次数 |
| created_at | TIMESTAMP | DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT NOW() | 更新时间 |

**Indexes**:
- `idx_banner_position` ON (position, is_active, sort DESC)
- `idx_banner_time` ON (start_time, end_time)

**Business Rules**:
- 展示时按 position 筛选、is_active = true、当前时间在 start_time 和 end_time 之间
- 按 sort 降序排列

---

## Database Indexes Summary

### High-Priority Indexes (P0)

| Table | Index | Columns | Type |
|-------|-------|---------|------|
| users | idx_user_phone | phone | UNIQUE |
| users | idx_user_openid | openid | UNIQUE |
| devices | idx_device_venue | venue_id | B-TREE |
| orders | idx_order_user | user_id | B-TREE |
| orders | idx_order_type_status | type, status | B-TREE |
| rentals | idx_rental_device | device_id | B-TREE |
| payments | idx_payment_order | order_id | B-TREE |

### Partition Strategy

**orders 表按月分区**:
```sql
CREATE TABLE orders (
  ...
) PARTITION BY RANGE (created_at);

CREATE TABLE orders_2026_01 PARTITION OF orders
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
```

**operation_logs 表按月分区**:
```sql
CREATE TABLE operation_logs (
  ...
) PARTITION BY RANGE (created_at);
```
