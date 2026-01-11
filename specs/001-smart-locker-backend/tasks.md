# Tasks: 爱上杜美人智能开锁管理系统后端服务

**Input**: Design documents from `/specs/001-smart-locker-backend/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: 包含单元测试、集成测试和 E2E 测试任务（Phase 12），目标覆盖率：单测 > 80%，关键业务 > 90%。

**Seed Data**: 使用前端 mock 数据作为初始化测试数据，支持 `make seed` 一键初始化开发环境。

**Organization**: 任务按用户故事组织，支持独立实现和测试每个故事。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 所属用户故事（US1, US2, US3...）
- 描述中包含具体文件路径

## Path Conventions

基于 plan.md 定义的项目结构：
- 服务入口: `cmd/`
- 内部实现: `internal/`
- 公共包: `pkg/`
- 数据库迁移: `migrations/`
- 配置: `configs/`
- 部署: `deployments/`

---

## Phase 1: Setup (项目初始化)

**Purpose**: 项目基础结构和开发环境配置

- [x] T001 创建项目目录结构，按 plan.md 中的结构创建所有目录
- [x] T002 初始化 Go Module，创建 `go.mod` 文件
- [x] T003 [P] 添加核心依赖到 `go.mod`：gin, gorm, viper, jwt, redis, mqtt
- [x] T004 [P] 创建 Makefile 定义常用命令（build, run, test, lint, migrate）
- [x] T005 [P] 创建 `.gitignore` 文件
- [x] T006 [P] 配置 golangci-lint，创建 `.golangci.yml`
- [x] T007 [P] 创建 Docker Compose 开发环境配置 `deployments/docker/docker-compose.yml`
- [x] T008 创建配置文件模板 `configs/config.example.yaml`

---

## Phase 2: Foundational (基础设施)

**Purpose**: 所有用户故事都依赖的核心基础设施

**⚠️ CRITICAL**: 此阶段必须完成后才能开始任何用户故事

### 配置与数据库

- [x] T009 实现配置管理模块 `internal/common/config/config.go`
- [x] T010 实现数据库连接模块 `internal/common/database/postgres.go`
- [x] T011 [P] 实现 Redis 连接模块 `internal/common/cache/redis.go`
- [x] T012 [P] 实现日志模块 `internal/common/logger/logger.go`

### 数据库迁移 - 核心表

- [x] T013 创建 User 表迁移 `migrations/000001_create_users.up.sql`
- [x] T014 [P] 创建 UserWallet 表迁移 `migrations/000001_create_users.up.sql` (合并到用户迁移)
- [x] T015 [P] 创建 MemberLevel 表迁移 `migrations/000001_create_users.up.sql` (合并到用户迁移)
- [x] T016 [P] 创建 Admin/Role/Permission 表迁移 `migrations/000002_create_admins.up.sql`
- [x] T017 [P] 创建 Merchant/Venue/Device 表迁移 `migrations/000003_create_devices.up.sql`
- [x] T018 [P] 创建 Order/OrderItem/Payment 表迁移 `migrations/000004_create_orders.up.sql`
- [x] T019 [P] 创建 Rental/RentalPricing 表迁移 `migrations/000005_create_rentals.up.sql`
- [x] T020 [P] 创建 Hotel/Room/Booking 表迁移 `migrations/000006_create_hotels.up.sql`
- [x] T021 [P] 创建 Category/Product/ProductSku/CartItem 表迁移 `migrations/000007_create_products.up.sql`
- [x] T022 [P] 创建 Distributor/Commission/Withdrawal 表迁移 `migrations/000008_create_distribution.up.sql`
- [x] T023 [P] 创建 Coupon/UserCoupon/Campaign 表迁移 `migrations/000009_create_marketing.up.sql`
- [x] T024 [P] 创建 Settlement/WalletTransaction 表迁移 `migrations/000010_create_finance.up.sql`
- [x] T025 [P] 创建 Article/Notification/SystemConfig/OperationLog 表迁移 `migrations/000011_create_system.up.sql`
- [x] T026 [P] 创建 Address 表迁移 `migrations/000001_create_users.up.sql` (合并到用户迁移)
- [x] T027 [P] 创建 RoomTimeSlot 表迁移 `migrations/000006_create_hotels.up.sql` (合并到酒店迁移)
- [x] T028 [P] 创建 SmsCode 表迁移 `migrations/000011_create_system.up.sql` (合并到系统迁移)
- [x] T029 [P] 创建 Banner 表迁移 `migrations/000011_create_system.up.sql` (合并到系统迁移)
- [x] T030 创建数据库迁移脚本 `scripts/migrate.sh`

### 种子数据（开发测试）

- [x] T031 创建种子数据目录结构和加载脚本 `seeds/` + `scripts/seed.sh`
- [x] T032 从 admin-frontend/user-frontend mock 数据提取用户/管理员种子数据 `seeds/001_users.sql`
- [x] T033 [P] 提取会员等级/角色/权限种子数据 `seeds/002_rbac.sql`
- [x] T034 [P] 提取商户/场地/设备种子数据 `seeds/003_devices.sql`
- [x] T035 [P] 提取酒店/房间/时段价格种子数据 `seeds/004_hotels.sql`
- [x] T036 [P] 提取商品分类/商品/SKU 种子数据 `seeds/005_products.sql`
- [x] T037 [P] 提取优惠券/活动种子数据 `seeds/006_marketing.sql`
- [x] T038 [P] 提取租借定价/Banner/系统配置种子数据 `seeds/007_system.sql`
- [x] T039 更新 Makefile 添加 `make seed` 和 `make reset-db` 命令

### 核心模型定义

- [x] T040 [P] 定义 User 模型 `internal/models/user.go`
- [x] T041 [P] 定义 Admin/Role/Permission 模型 `internal/models/admin.go`
- [x] T042 [P] 定义 Merchant/Venue 模型 `internal/models/venue.go`
- [x] T043 [P] 定义 Device 模型 `internal/models/device.go`
- [x] T044 [P] 定义 Order/OrderItem 模型 `internal/models/order.go`
- [x] T045 [P] 定义 Payment/Refund 模型 `internal/models/payment.go`

### 公共组件

- [x] T046 实现统一响应格式 `internal/common/response/response.go`
- [x] T047 [P] 实现错误码定义 `internal/common/errors/errors.go`
- [x] T048 [P] 实现 JWT 工具 `internal/common/jwt/jwt.go`
- [x] T049 [P] 实现加密工具（AES-256-GCM）`internal/common/crypto/crypto.go`
- [x] T050 [P] 实现密码哈希工具（bcrypt）`internal/common/crypto/crypto.go` (合并到加密模块)

### 中间件

- [x] T051 实现认证中间件 `internal/middleware/auth.go`
- [x] T052 [P] 实现 RBAC 权限中间件 `internal/middleware/permission.go`
- [x] T053 [P] 实现请求日志中间件 `internal/middleware/logging.go`
- [x] T054 [P] 实现限流中间件 `internal/middleware/ratelimit.go`
- [x] T055 [P] 实现跨域中间件 `internal/middleware/cors.go`
- [x] T056 [P] 实现请求 ID 中间件 `internal/middleware/common.go`

### API Gateway 入口

- [x] T057 创建 API Gateway 主入口 `cmd/api/main.go`
- [x] T058 实现路由注册 `cmd/api/router.go`
- [x] T059 实现健康检查端点 `cmd/api/health.go`

**Checkpoint**: ✅ 基础设施就绪，可以开始用户故事实现

---

## Phase 3: User Story 1 - 用户扫码租借智能柜 (Priority: P1) 🎯 MVP

**Goal**: 实现用户扫码→支付→开锁→归还→结算的完整租借流程

**Independent Test**: 模拟用户完整租借流程，验证从扫码到结算的完整链路

### 认证模块

- [x] T060 [P] [US1] 实现短信验证码服务 `pkg/sms/aliyun.go`
- [x] T061 [P] [US1] 实现验证码存储（Redis）`internal/service/auth/code_service.go`
- [x] T062 [US1] 实现用户注册/登录服务 `internal/service/auth/auth_service.go`
- [x] T063 [P] [US1] 实现微信授权登录服务 `internal/service/auth/wechat_service.go`
- [x] T064 [US1] 实现认证 API Handler `internal/handler/auth/auth_handler.go`

### 用户模块

- [x] T065 [US1] 实现用户 Repository `internal/repository/user_repo.go`
- [x] T066 [US1] 实现用户服务 `internal/service/user/user_service.go`
- [x] T067 [US1] 实现用户钱包服务 `internal/service/user/wallet_service.go`
- [x] T068 [US1] 实现用户 API Handler `internal/handler/user/user_handler.go`

### 设备与场地模块

- [x] T069 [P] [US1] 定义 Rental/RentalPricing 模型 `internal/models/rental.go`
- [x] T070 [US1] 实现设备 Repository `internal/repository/device_repo.go`
- [x] T071 [US1] 实现场地 Repository `internal/repository/venue_repo.go`
- [x] T072 [US1] 实现设备查询服务 `internal/service/device/device_service.go`
- [x] T073 [US1] 实现设备 API Handler（用户端）`internal/handler/device/device_handler.go`

### MQTT 设备通信

- [x] T074 [US1] 实现 MQTT 客户端 `internal/common/mqtt/client.go`
- [x] T075 [US1] 实现设备控制服务（开锁/状态查询）`internal/service/device/control_service.go`
- [x] T076 [US1] 实现设备状态订阅处理 `internal/service/device/status_handler.go`

### 租借模块

- [x] T077 [US1] 实现租借定价 Repository `internal/repository/rental_pricing_repo.go`
- [x] T078 [US1] 实现租借 Repository `internal/repository/rental_repo.go`
- [x] T079 [US1] 实现租借服务（创建/归还/超时处理）`internal/service/rental/rental_service.go`
- [x] T080 [US1] 实现租借 API Handler `internal/handler/rental/rental_handler.go`

### 订单模块

- [x] T081 [US1] 实现订单 Repository `internal/repository/order_repo.go`
- [x] T082 [US1] 实现订单服务 `internal/service/order/order_service.go`
- [x] T083 [US1] 实现订单 API Handler `internal/handler/order/order_handler.go`

### 支付模块

- [x] T084 [P] [US1] 实现微信支付 SDK 封装 `pkg/payment/wechat/wechat.go`
- [x] T085 [P] [US1] 实现支付宝 SDK 封装 `pkg/payment/alipay/alipay.go`
- [x] T086 [US1] 实现支付 Repository `internal/repository/payment_repo.go`
- [x] T087 [US1] 实现统一支付服务 `internal/service/payment/payment_service.go`
- [x] T088 [US1] 实现支付回调处理 `internal/handler/payment/callback_handler.go`
- [x] T089 [US1] 实现支付 API Handler `internal/handler/payment/payment_handler.go`

### 定时任务

- [x] T090 [US1] 实现租借超时检查任务 `internal/service/rental/timeout_checker.go`
- [x] T091 [US1] 实现超过24小时自动购买逻辑 `internal/service/rental/auto_purchase.go`

### 路由注册

- [x] T092 [US1] 注册 User Story 1 所有路由到 API Gateway

**Checkpoint**: User Story 1 完成，用户可以完整体验扫码租借流程

---

## Phase 4: User Story 2 - 管理员设备监控与管理 (Priority: P1)

**Goal**: 管理员可实时监控设备状态、远程控制设备、管理场地和商户

**Independent Test**: 通过管理后台进行设备状态查看和远程控制操作

### 管理员认证

- [x] T093 [US2] 实现管理员 Repository `internal/repository/admin_repo.go`
- [x] T094 [US2] 实现管理员登录服务 `internal/service/admin/admin_auth_service.go`
- [x] T095 [US2] 实现管理员认证 API Handler `internal/handler/admin/auth_handler.go`

### 权限管理

- [x] T096 [US2] 实现角色权限 Repository `internal/repository/role_repo.go`
- [x] T097 [US2] 实现权限服务 `internal/service/admin/permission_service.go`

### 设备管理（管理端）

- [x] T098 [P] [US2] 定义 DeviceLog/DeviceMaintenance 模型 `internal/models/device_log.go`
- [x] T099 [US2] 实现设备日志 Repository `internal/repository/device_log_repo.go`
- [x] T100 [US2] 实现设备管理服务（CRUD/远程控制）`internal/service/admin/device_admin_service.go`
- [x] T101 [US2] 实现设备管理 API Handler `internal/handler/admin/device_handler.go`

### 场地管理

- [x] T102 [US2] 实现场地管理服务 `internal/service/admin/venue_admin_service.go`
- [x] T103 [US2] 实现场地管理 API Handler `internal/handler/admin/venue_handler.go`

### 商户管理

- [x] T104 [US2] 实现商户 Repository `internal/repository/merchant_repo.go`
- [x] T105 [US2] 实现商户管理服务 `internal/service/admin/merchant_admin_service.go`
- [x] T106 [US2] 实现商户管理 API Handler `internal/handler/admin/merchant_handler.go`

### 二维码生成

- [x] T107 [US2] 实现二维码生成工具 `pkg/qrcode/generator.go`
- [x] T108 [US2] 实现设备二维码生成逻辑 `internal/service/device/qrcode_service.go`

### 设备告警

- [x] T109 [US2] 实现设备异常告警服务 `internal/service/device/alert_service.go`

### 操作日志

- [x] T110 [P] [US2] 定义 OperationLog 模型 `internal/models/operation_log.go`
- [x] T111 [US2] 实现操作日志 Repository `internal/repository/operation_log_repo.go`
- [x] T112 [US2] 实现操作日志中间件 `internal/common/middleware/operation_log.go`

### 路由注册

- [x] T113 [US2] 注册 User Story 2 所有管理端路由

**Checkpoint**: User Story 2 完成，管理员可监控和管理设备

---

## Phase 5: User Story 3 - 用户商城购物 (Priority: P2)

**Goal**: 用户可浏览商品、加购、下单支付、查看订单、申请退款

**Independent Test**: 完整购物流程从商品浏览到支付完成

### 商品模块

- [x] T114 [P] [US3] 定义 Category/Product/ProductSku 模型 `internal/models/product.go`
- [x] T115 [P] [US3] 定义 CartItem 模型 `internal/models/cart.go`
- [x] T116 [P] [US3] 定义 Review 模型 `internal/models/review.go`
- [x] T117 [US3] 实现分类 Repository `internal/repository/category_repo.go`
- [x] T118 [US3] 实现商品 Repository `internal/repository/product_repo.go`
- [x] T119 [US3] 实现商品服务 `internal/service/mall/product_service.go`
- [x] T120 [US3] 实现商品搜索服务 `internal/service/mall/search_service.go`
- [x] T121 [US3] 实现商品 API Handler `internal/handler/mall/product_handler.go`

### 购物车模块

- [x] T122 [US3] 实现购物车 Repository `internal/repository/cart_repo.go`
- [x] T123 [US3] 实现购物车服务 `internal/service/mall/cart_service.go`
- [x] T124 [US3] 实现购物车 API Handler `internal/handler/mall/cart_handler.go`

### 商城订单

- [x] T125 [US3] 实现商城订单服务 `internal/service/mall/mall_order_service.go`
- [x] T126 [US3] 实现商城订单 API Handler `internal/handler/mall/order_handler.go`

### 商品评价

- [x] T127 [US3] 实现评价 Repository `internal/repository/review_repo.go`
- [x] T128 [US3] 实现评价服务 `internal/service/mall/review_service.go`
- [x] T129 [US3] 实现评价 API Handler `internal/handler/mall/review_handler.go`

### 退款处理

- [x] T130 [P] [US3] 定义 Refund 模型 `internal/models/refund.go` (已合并到 payment.go)
- [x] T131 [US3] 实现退款 Repository `internal/repository/refund_repo.go` (已合并到 payment_repo.go)
- [x] T132 [US3] 实现退款服务 `internal/service/order/refund_service.go`
- [x] T133 [US3] 实现退款 API Handler `internal/handler/order/refund_handler.go`

### 商品管理（管理端）

- [x] T134 [US3] 实现商品管理服务 `internal/service/admin/product_admin_service.go`
- [x] T135 [US3] 实现商品管理 API Handler `internal/handler/admin/product_handler.go`

### 路由注册

- [x] T136 [US3] 注册 User Story 3 所有路由

**Checkpoint**: User Story 3 完成，商城购物功能可用 ✅

---

## Phase 6: User Story 4 - 酒店房间智能柜租借 (Priority: P2)

**Goal**: 用户可预订酒店房间，获取核销码和开锁码，到店核销后使用开锁码开锁

**Independent Test**: 预订→支付→核销→开锁的完整流程

### 酒店模块

- [x] T137 [P] [US4] 定义 Hotel/Room/Booking 模型 `internal/models/hotel.go`
- [x] T138 [US4] 实现酒店 Repository `internal/repository/hotel_repo.go`
- [x] T139 [US4] 实现房间 Repository `internal/repository/room_repo.go`
- [x] T140 [US4] 实现酒店服务 `internal/service/hotel/hotel_service.go`
- [x] T141 [US4] 实现酒店 API Handler `internal/handler/hotel/hotel_handler.go`

### 预订模块

- [x] T142 [US4] 实现预订 Repository `internal/repository/booking_repo.go`
- [x] T143 [US4] 实现预订服务（创建/核销/开锁码验证）`internal/service/hotel/booking_service.go`
- [x] T144 [US4] 实现开锁码生成与验证 `internal/service/hotel/code_service.go` (合并 T144/T145)
- [x] T145 [US4] 实现核销码生成 `internal/service/hotel/code_service.go` (合并到 T144)
- [x] T146 [US4] 实现预订 API Handler `internal/handler/hotel/booking_handler.go`

### 酒店管理（管理端）

- [x] T147 [US4] 实现酒店管理服务 `internal/service/admin/hotel_admin_service.go`
- [x] T148 [US4] 实现酒店管理 API Handler `internal/handler/admin/hotel_handler.go`
- [x] T149 [US4] 实现前台核销 API Handler `internal/handler/admin/booking_verify_handler.go`

### 路由注册

- [x] T150 [US4] 注册 User Story 4 所有路由

**Checkpoint**: User Story 4 完成，酒店预订功能可用 ✅

---

## Phase 7: User Story 5 - 分销商推广与佣金管理 (Priority: P2)

**Goal**: 分销商可生成推广链接，推广用户消费后获得佣金

**Independent Test**: 推广链接→用户注册→消费→佣金计算→提现的完整流程

### 分销模块

- [x] T151 [P] [US5] 定义 Distributor/Commission/Withdrawal 模型 `internal/models/distribution.go`
- [x] T152 [US5] 实现分销商 Repository `internal/repository/distributor_repo.go`
- [x] T153 [US5] 实现佣金 Repository `internal/repository/commission_repo.go`
- [x] T154 [US5] 实现提现 Repository `internal/repository/withdrawal_repo.go`
- [x] T155 [US5] 实现分销商服务（申请/审核/团队）`internal/service/distribution/distributor_service.go`
- [x] T156 [US5] 实现佣金计算服务（按实付金额）`internal/service/distribution/commission_service.go`
- [x] T157 [US5] 实现推广链接生成服务 `internal/service/distribution/invite_service.go`
- [x] T158 [US5] 实现提现服务 `internal/service/distribution/withdraw_service.go`
- [x] T159 [US5] 实现分销 API Handler（用户端）`internal/handler/distribution/distribution_handler.go`

### 分销管理（管理端）

- [x] T160 [US5] 实现分销管理服务 `internal/service/admin/distribution_admin_service.go`
- [x] T161 [US5] 实现分销管理 API Handler `internal/handler/admin/distribution_handler.go`

### 佣金设置

- [x] T162 [US5] 实现佣金设置服务 `internal/service/admin/commission_setting_service.go`

### 订单完成触发佣金

- [x] T163 [US5] 在订单完成时触发佣金计算 `internal/service/order/order_complete_hook.go`

### 路由注册

- [x] T164 [US5] 注册 User Story 5 所有路由

**Checkpoint**: User Story 5 完成，分销体系可用 ✅

---

## Phase 8: User Story 6 - 财务对账与结算 (Priority: P3)

**Goal**: 财务管理员可查看统计、执行结算、审核提现、导出报表

**Independent Test**: 财务报表查询、结算操作、提现审核独立测试

### 财务模块

- [x] T165 [P] [US6] 定义 Settlement/WalletTransaction 模型 `internal/models/finance.go`
- [x] T166 [US6] 实现结算 Repository `internal/repository/settlement_repo.go`
- [x] T167 [US6] 实现交易流水 Repository `internal/repository/transaction_repo.go`
- [x] T168 [US6] 实现财务统计服务 `internal/service/finance/statistics_service.go`
- [x] T169 [US6] 实现结算服务（商户/分销商）`internal/service/finance/settlement_service.go`
- [x] T170 [US6] 实现提现审核服务 `internal/service/finance/withdrawal_audit_service.go`
- [x] T171 [US6] 实现报表导出服务 `internal/service/finance/export_service.go`
- [x] T172 [US6] 实现财务 API Handler `internal/handler/admin/finance_handler.go`

### 路由注册

- [x] T173 [US6] 注册 User Story 6 所有路由

**Checkpoint**: User Story 6 完成，财务管理功能可用 ✅

---

## Phase 9: User Story 7 - 营销活动与优惠券管理 (Priority: P3)

**Goal**: 运营管理员可创建优惠券、管理活动，用户可领取和使用

**Independent Test**: 优惠券创建→发放→领取→下单使用的完整流程

### 优惠券模块

- [x] T174 [P] [US7] 定义 Coupon/UserCoupon/Campaign 模型 `internal/models/marketing.go`
- [x] T175 [US7] 实现优惠券 Repository `internal/repository/coupon_repo.go`
- [x] T176 [US7] 实现用户优惠券 Repository `internal/repository/user_coupon_repo.go`
- [x] T177 [US7] 实现优惠券服务 `internal/service/marketing/coupon_service.go`
- [x] T178 [US7] 实现用户优惠券服务（领取/使用/过期）`internal/service/marketing/user_coupon_service.go`
- [x] T179 [US7] 实现营销 API Handler（用户端）`internal/handler/marketing/coupon_handler.go`

### 营销活动

- [x] T180 [US7] 实现活动 Repository `internal/repository/campaign_repo.go`
- [x] T181 [US7] 实现活动服务 `internal/service/marketing/campaign_service.go`

### 营销管理（管理端）

- [x] T182 [US7] 实现优惠券管理服务 `internal/service/admin/marketing_admin_service.go`
- [x] T183 [US7] 实现营销管理 API Handler `internal/handler/admin/marketing_handler.go`

### 订单优惠计算

- [x] T184 [US7] 在订单创建时计算优惠 `internal/service/order/discount_calculator.go`

### 路由注册

- [x] T185 [US7] 注册 User Story 7 所有路由

**Checkpoint**: User Story 7 完成，营销功能可用 ✅

---

## Phase 10: User Story 8 - 会员体系与权益管理 (Priority: P3)

**Goal**: 用户消费积分累积、等级升级、享受会员权益

**Independent Test**: 消费→积分累积→等级升级→权益生效的流程测试

### 会员模块

- [x] T186 [P] [US8] 定义 MemberPackage 模型 `internal/models/member.go`
- [x] T187 [US8] 实现会员等级 Repository `internal/repository/member_level_repo.go`
- [x] T188 [US8] 实现会员套餐 Repository `internal/repository/member_package_repo.go`
- [x] T189 [US8] 实现积分服务 `internal/service/user/points_service.go`
- [x] T190 [US8] 实现会员等级服务（升级检测）`internal/service/user/member_level_service.go`
- [x] T191 [US8] 实现会员套餐购买服务 `internal/service/user/member_package_service.go`
- [x] T192 [US8] 实现会员 API Handler `internal/handler/user/member_handler.go`

### 会员管理（管理端）

- [x] T193 [US8] 实现会员管理服务 `internal/service/admin/member_admin_service.go`
- [x] T194 [US8] 实现会员管理 API Handler `internal/handler/admin/member_handler.go`

### 订单完成触发积分

- [x] T195 [US8] 在订单完成时累加积分 `internal/service/order/points_hook.go`

### 会员折扣计算

- [x] T196 [US8] 在订单创建时应用会员折扣 `internal/service/order/member_discount.go`

### 路由注册

- [x] T197 [US8] 注册 User Story 8 所有路由

**Checkpoint**: User Story 8 完成，会员体系可用 ✅

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事的优化和完善

### 仪表盘

- [x] T198 [P] 实现平台管理员仪表盘数据服务 `internal/service/admin/dashboard_service.go`
- [x] T199 [P] 实现分销商仪表盘数据服务 `internal/service/distribution/dashboard_service.go`
- [x] T200 [P] 实现财务仪表盘数据服务 `internal/service/finance/dashboard_service.go`
- [x] T201 [P] 实现运营仪表盘数据服务 `internal/service/admin/operation_dashboard_service.go`
- [x] T202 实现仪表盘 API Handler `internal/handler/admin/dashboard_handler.go`

### 内容管理

- [x] T203 [P] 定义 Article/Notification/MessageTemplate 模型 `internal/models/system.go`
- [x] T204 [P] 实现文章 Repository `internal/repository/article_repo.go`
- [x] T205 [P] 实现通知 Repository `internal/repository/notification_repo.go`
- [x] T206 实现内容服务 `internal/service/content/content_service.go`
- [x] T207 实现通知服务 `internal/service/content/notification_service.go`
- [x] T208 实现内容 API Handler `internal/handler/content/content_handler.go`

### 系统管理

- [x] T209 [P] 定义 SystemConfig 模型 `internal/models/system_config.go`
- [x] T210 实现系统配置 Repository `internal/repository/system_config_repo.go`
- [x] T211 实现系统配置服务 `internal/service/admin/system_config_service.go`
- [x] T212 实现系统管理 API Handler `internal/handler/admin/system_handler.go`

### 用户管理（管理端）

- [x] T213 实现用户管理服务 `internal/service/admin/user_admin_service.go`
- [x] T214 实现用户管理 API Handler `internal/handler/admin/user_handler.go`

### 订单管理（管理端）

- [x] T215 实现订单管理服务 `internal/service/admin/order_admin_service.go`
- [x] T216 实现订单管理 API Handler `internal/handler/admin/order_handler.go`

### 用户反馈

- [x] T217 [P] 定义 UserFeedback 模型 `internal/models/feedback.go`
- [x] T218 实现反馈 Repository `internal/repository/feedback_repo.go`
- [x] T219 实现反馈服务 `internal/service/user/feedback_service.go`
- [x] T220 实现反馈 API Handler `internal/handler/user/feedback_handler.go`

### 用户收货地址

- [x] T221 [P] 定义 Address 模型 `internal/models/address.go`
- [x] T222 实现 Address Repository `internal/repository/address_repo.go`
- [x] T223 实现地址服务（CRUD/设置默认）`internal/service/user/address_service.go`
- [x] T224 实现地址 API Handler `internal/handler/user/address_handler.go`

### Banner 轮播图管理

- [x] T225 [P] 定义 Banner 模型 `internal/models/banner.go`
- [x] T226 实现 Banner Repository `internal/repository/banner_repo.go`
- [x] T227 实现 Banner 服务（用户端查询）`internal/service/content/banner_service.go`
- [x] T228 实现 Banner 管理服务（管理端 CRUD）`internal/service/admin/banner_admin_service.go`
- [x] T229 实现 Banner API Handler（用户端）`internal/handler/content/banner_handler.go`
- [x] T230 实现 Banner 管理 API Handler `internal/handler/admin/banner_handler.go`

### 消息推送

- [x] T231 实现短信推送服务 `pkg/sms/sender.go`
- [x] T232 实现消息模板服务 `internal/service/content/template_service.go`

### 对象存储

- [x] T233 实现阿里云 OSS 上传 `pkg/oss/aliyun.go`

### 可观测性

- [x] T233a [P] 集成 Prometheus 指标收集 `internal/common/metrics/prometheus.go`，暴露 `/metrics` 端点，收集 API 请求量、响应时间、错误率、数据库连接池状态等核心指标
- [x] T233b [P] 集成 OpenTelemetry 分布式追踪 `internal/common/tracing/opentelemetry.go`，支持请求链路追踪、跨服务调用追踪、数据库查询追踪
- [x] T233c [P] 实现追踪中间件 `internal/common/middleware/tracing.go`，自动为每个请求生成 Trace ID 并传递到下游

### API 文档

- [x] T234 集成 Swagger 文档生成 `cmd/api-gateway/swagger.go`
- [x] T235 生成 OpenAPI 文档到 `docs/`

### 部署配置

- [x] T236 [P] 创建 Dockerfile `deployments/docker/Dockerfile`
- [x] T237 [P] 创建 Kubernetes 部署配置 `deployments/k8s/deployment.yaml`
- [x] T238 [P] 创建 Kubernetes Service 配置 `deployments/k8s/service.yaml`

### 文档完善

- [x] T239 更新 quickstart.md 验证所有功能

---

## Phase 12: Testing (测试)

**Purpose**: 确保代码质量和业务逻辑正确性

**⚠️ NOTE**: 测试任务可与功能开发并行，建议每完成一个模块即编写对应测试

---

### 如何生成“测试覆盖率报告”

- 生成单元测试覆盖率报告（默认统计 `./internal/...` + `./pkg/...`，输出 HTML 报告）：
  - `make coverage`
  - 产物：`coverage.out`、`coverage.html`
  - 查看：`open coverage.html`（macOS）/ `xdg-open coverage.html`（Linux）/ 直接用浏览器打开
  - 命令行摘要：`go tool cover -func=coverage.out | tail -n 20`
- 运行覆盖率门禁（用于 CI/CD；默认运行带 tag 的 `api,e2e` 测试并统计关键模块覆盖率）：
  - `make coverage-gate`
  - 可通过环境变量调整：`OVERALL_MIN`、`KEY_MODULE_MIN`、`GO_TEST_TAGS`、`GO_TEST_TARGETS`、`COVERPKG`

> 说明：`scripts/coverage.sh` 会把 `GOCACHE` 指向仓库内的 `.gocache/`，避免在受限环境下写入系统缓存目录导致权限问题。

### 当前测试资产盘点（已创建）

- 业务单元测试（in-package）：`internal/service/{admin,auth,content,device,distribution,finance,hotel,mall,marketing,order,payment,rental,user}/*_test.go`
- 公共模块单元测试：`internal/common/{metrics,middleware,tracing}/*_test.go`
- Repository 单元测试：`internal/repository/{admin_repo,coupon_repo,device_repo,order_repo,payment_repo,rental_repo,user_repo}_test.go`（更多 repo 已补齐）
- 端到端/场景测试：
  - API 测试：`tests/api/*_test.go`（`//go:build api`）
  - 集成测试：`tests/integration/*_test.go`（`//go:build integration`）
  - E2E 测试：`tests/e2e/*_test.go`（`//go:build e2e`）
  - 额外单测（tests 目录内）：`tests/unit/*_test.go`（无 build tag）

### 当前覆盖率现状（实测）

> 说明：以下为本仓库内置脚本的实测结果（以当前代码为准），用于 Phase 12 的"达标/差距"追踪。
> **更新时间：2026-01-11** (最新更新：高优先级冲刺完成)

#### 📊 关键模块覆盖率现状

**覆盖率门禁（key modules = auth/payment/order/rental/booking）：整体约 76.8% ⬆️**

**优秀模块 (≥85%)：**

- payment `92.4%`            ✅
- auth `90.6%`               ✅
- order `90.4%`              ✅
- content `89.6%`
- hotel (booking) `87.6%`
- distribution `86.8%`
- rental `85.3%`

**良好模块 (70-85%)：**

- marketing `84.1%`
- user `80.1%`

**中等模块 (50-70%)：**

- device `62.6%`
- mall `55.5%`

**需改进模块 (<50%)：**

- admin `49.6%`
- finance `36.2%`

## ⚠️ CRITICAL: Model开发验证Checklist

**所有涉及Model开发的任务必须遵循以下Checklist，在PR提交前逐项检查：**

### 📋 开发前检查 (必须完成)

- 已查阅 `specs/001-smart-locker-backend/data-model.md` 对应表的完整定义
- 已查看对应的 `migrations/000XXX_create_xxx.up.sql` 文件
- 理解了表的业务含义和字段用途
- 了解了该表与其他表的关联关系

### 📝 编码中检查 (逐项验证)

- Model中**所有字段**都添加了 `column:` 标签
- 字段名与数据库列名**完全一致**
- 字段类型与数据库类型正确映射:
  - VARCHAR → string
  - BIGINT → int64
  - INT → int
  - DECIMAL → float64
  - BOOLEAN → bool
  - TIMESTAMP (必填) → time.Time
  - TIMESTAMP (可空) → *time.Time
- 状态字段使用 `string` 类型(而非int/int8)
- 所有NOT NULL字段都定义为值类型
- 所有NULLABLE字段都定义为指针类型
- 没有添加数据库中不存在的字段
- 没有遗漏数据库中的必填字段
- 外键字段正确定义了关联关系
- TableName()方法返回正确的表名

### ✅ 开发后检查 (必须通过)

- 已运行 `go build ./internal/models/...` 验证编译通过
- 已编写基础CRUD单元测试
- 单元测试能够成功插入数据
- 单元测试能够成功查询数据
- 单元测试能够成功更新数据
- 所有测试用例通过
- 已手动测试在实际数据库中的CRUD操作

### 📚 必读参考文档

开发时必须参考:
1. **`specs/001-smart-locker-backend/data-model.md`** - 数据模型定义(权威参照)
2. **`specs/001-smart-locker-backend/model-development-guide.md`** - Go Model开发规范(开发标准)
3. **对应的migration文件** - 数据库实际结构(实现参照)

**⚠️ 重要**: 只有通过以上所有检查项，Model开发任务才算完成！

---

### 测试基础设施

- [x] T240 配置测试框架和 mock 工具 `tests/setup_test.go`
- [ ] T241 [P] 配置 testcontainers-go 集成测试环境 `tests/integration/testcontainers.go`（当前集成测试主要使用 sqlite in-memory，可后续补齐 Postgres/Redis 容器化测试）
- [x] T242 [P] 创建测试工具函数（数据库清理、mock 数据生成）`tests/helpers/`

### 单元测试 - 核心业务

- [x] T243 [P] 编写 auth_service 单元测试 `internal/service/auth/auth_service_test.go`
- [x] T244 [P] 补齐验证码发送/校验单元测试（频率限制/每日上限/发送失败回滚/一次性校验）`internal/service/auth/code_service_test.go`
- [x] T245 [P] 补齐微信登录/绑定手机号单元测试（code2Session 成功/失败、老用户更新/新用户创建、邀请码、绑定手机号冲突）`internal/service/auth/wechat_service_test.go`
- [x] T246 [P] 编写 rental_service 单元测试 `internal/service/rental/rental_service_test.go`
- [x] T247 [P] 补齐租借核心流程边界测试（CancelRental/GetRental/ListRentals/GenerateRentalNo/超时与异常分支）`internal/service/rental/{rental_service_test.go,rental_service_extra_test.go}`
- [x] T248 [P] 编写 payment_service 单元测试 `internal/service/payment/payment_service_test.go`
- [x] T249 [P] 补齐支付回调与状态机单元测试（HandlePaymentCallback、重复回调幂等、失败分支）`internal/service/payment/payment_service_test.go`
- [x] T250 [P] 编写订单域相关单元测试 `internal/service/order/{refund_service_test.go,member_discount_test.go,points_hook_test.go,order_complete_hook_test.go}`
- [x] T251 [P] 补齐 order_service 主流程单元测试 `internal/service/order/discount_calculator_test.go`
- [x] T252 [P] 补齐订单折扣/积分/退款关键分支（CalculateWithMemberDiscount、积分Hook组合器、退款审核通过/拒绝/列表/详情）`internal/service/order/{member_discount_test.go,points_hook_test.go,refund_service_test.go}`
- [x] T253 [P] 编写酒店业务单元测试 `internal/service/hotel/{booking_service_test.go,hotel_service_test.go,code_service_test.go}`
- [x] T254 [P] 补齐酒店预订关键分支（GetBookingByNo/UnlockByCode/到期与完成任务处理/jsonToStringSlice）`internal/service/hotel/{booking_service_test.go,hotel_service_test.go}`
- [x] T255 [P] 编写分销业务单元测试 `internal/service/distribution/{commission_service_test.go,distributor_service_test.go,withdraw_service_test.go}`
- [x] T256 [P] 补齐分销邀请/推广链路单元测试 `internal/service/distribution/invite_service_test.go`
- [x] T257 [P] 补齐 wallet_service 单元测试 `internal/service/user/wallet_service_test.go`
- [x] T258 [P] 补齐用户核心服务单元测试（user_service 关键分支：注册资料/状态/手机号等）`internal/service/user/user_service_test.go`
- [x] T259 [P] 编写营销相关单元测试 `tests/unit/{coupon_service_test.go,campaign_service_test.go,user_coupon_service_test.go}`
- [x] T260 [P] 将 marketing 单测迁移/补齐到 in-package（便于覆盖率统计）`internal/service/marketing/marketing_service_test.go`
- [x] T261 [P] 补齐 finance 单元测试 `internal/service/finance/finance_service_test.go`
- [x] T262 [P] 补齐管理端后台核心服务单元测试（dashboard/permission/merchant/hotel/member/product/marketing 等）`internal/service/admin/*_test.go`
- [x] T263 [P] 补齐商城订单/搜索服务单元测试（mall_order_service/search_service）`internal/service/mall/*_test.go`
- [x] T264 [P] 补齐内容/通知服务单元测试（content_service/notification_service）`internal/service/content/*_test.go`
- [ ] T265 [P] 补齐通用基础模块单元测试 `internal/common/*_test.go`
  - ✅ crypto 模块（AES加密/解密、密码哈希、数据脱敏）`internal/common/crypto/crypto_test.go` - 30+ 测试用例
  - ✅ jwt 模块（令牌生成/解析/验证/刷新）`internal/common/jwt/jwt_test.go` - 25+ 测试用例
  - ✅ utils 模块（订单号生成、验证函数、金额格式化、分页）`internal/common/utils/utils_test.go` - 40+ 测试用例
  - ✅ qrcode 模块（二维码生成、格式转换、批量生成）`internal/common/qrcode/qrcode_test.go` - 30+ 测试用例
  - ✅ config 模块（配置加载、默认值、辅助方法）`internal/common/config/config_test.go` - 20+ 测试用例
  - ✅ errors 模块（错误码定义、错误包装、错误链）`internal/common/errors/errors_test.go` - 50+ 测试用例
  - ✅ response 模块（统一响应格式、HTTP状态码）`internal/common/response/response_test.go` - 35+ 测试用例
  - ⏳ 待完成：logger/cache/database（需要 mock 外部依赖，建议后续补齐）

### 单元测试 - Repository 层

- [x] T266 [P] 编写 user_repo 单元测试 `internal/repository/user_repo_test.go`
- [x] T267 [P] 编写 device_repo 单元测试 `internal/repository/device_repo_test.go`
- [x] T268 [P] 编写 order_repo 单元测试 `internal/repository/order_repo_test.go`
- [x] T269 [P] 编写 rental_repo 单元测试 `internal/repository/rental_repo_test.go`
- [x] T270 [P] 编写 admin_repo 单元测试 `internal/repository/admin_repo_test.go`
- [x] T271 [P] 补齐其余关键 repo 单元测试（payment/coupon）`internal/repository/{payment_repo_test.go,coupon_repo_test.go}`
- [x] T272 [P] 补齐剩余 repository 单元测试（按仓储文件逐一补齐 CRUD/列表/过滤/排序/边界条件）`internal/repository/*_repo_test.go`
  - 补齐清单（当前对应 `*_test.go`）：`address_repo`、`article_repo`、`banner_repo`、`booking_repo`、`campaign_repo`、`cart_repo`、`category_repo`、`commission_repo`、`device_alert_repo`、`device_log_repo`、`distributor_repo`、`feedback_repo`、`hotel_repo`、`member_level_repo`、`member_package_repo`、`merchant_repo`、`message_template_repo`、`notification_repo`、`operation_log_repo`、`product_repo`、`review_repo`、`role_repo`、`room_repo`、`settlement_repo`、`system_config_repo`、`transaction_repo`、`user_coupon_repo`、`venue_repo`、`withdrawal_repo`

### 集成测试

- [x] T273 [P] 编写租借流程集成测试（扫码→支付→开锁→归还）`tests/integration/rental_flow_test.go`
- [x] T274 [P] 编写支付流程集成测试（创建→回调→状态更新）`tests/integration/payment_flow_test.go`
- [x] T275 [P] 编写酒店预订集成测试（预订→核销→开锁）`tests/integration/us4_hotel_booking_flow_test.go`
- [x] T276 [P] 编写商城订单集成测试（加购→下单→支付）`tests/integration/us3_mall_order_flow_test.go`
- [x] T277 [P] 编写分销流程集成测试（推广→消费→计算佣金）`tests/integration/distribution_flow_test.go`
- [x] T278 [P] 编写管理端基础流程集成测试 `tests/integration/admin_flow_test.go`
- [x] T279 [P] 编写 US2 权限/设备监控集成测试 `tests/integration/{us2_permission_flow_test.go,us2_device_monitoring_flow_test.go}`
- [x] T280 [P] 编写 US6 财务/US7 营销/US8 会员集成测试 `tests/integration/{us6_finance_flow_test.go,us7_marketing_flow_test.go,us8_membership_flow_test.go}`

### E2E 测试

- [x] T281 [P] 编写扫码租借完整流程 E2E 测试 `tests/e2e/us1_scan_rent_flow_test.go`
- [x] T282 [P] 编写酒店预订完整流程 E2E 测试 `tests/e2e/us4_hotel_booking_flow_test.go`
- [x] T283 [P] 编写商城购物完整流程 E2E 测试 `tests/e2e/us3_mall_shopping_flow_test.go`
- [x] T284 [P] 编写 US2 管理端设备管理 E2E 测试 `tests/e2e/us2_admin_device_monitor_manage_flow_test.go`
- [x] T285 [P] 编写 US5 分销推广 E2E 测试 `tests/e2e/us5_distribution_flow_test.go`
- [x] T286 [P] 编写 US6 财务结算 E2E 测试 `tests/e2e/us6_finance_settlement_flow_test.go`
- [x] T287 [P] 编写 US7 营销优惠 E2E 测试 `tests/e2e/us7_marketing_flow_test.go`
- [x] T288 [P] 编写 US8 会员体系 E2E 测试 `tests/e2e/us8_membership_flow_test.go`

### API 测试

- [x] T289 编写 Auth API 测试 `tests/api/auth_api_test.go`
- [x] T290 [P] 编写管理端 Auth API 测试 `tests/api/admin_auth_api_test.go`
- [x] T291 [P] 编写 US1 租借 API 测试 `tests/api/us1_rental_api_test.go`
- [x] T292 [P] 编写 US2 管理端设备/商户/场地 API 测试 `tests/api/{admin_device_api_test.go,us2_admin_merchant_venue_api_test.go}`
- [x] T293 [P] 编写 US3 商城 API 测试 `tests/api/us3_mall_api_test.go`
- [x] T294 [P] 编写 US4 酒店 API 测试 `tests/api/us4_hotel_api_test.go`
- [x] T295 [P] 编写 US5 分销 API 测试 `tests/api/us5_distribution_api_test.go`
- [x] T296 [P] 编写 US6 财务 API 测试 `tests/api/us6_finance_api_test.go`
- [x] T297 [P] 编写 US7 营销 API 测试 `tests/api/us7_marketing_api_test.go`
- [x] T298 [P] 编写 US8 会员（用户端/管理端）API 测试 `tests/api/{us8_member_api_test.go,us8_member_admin_api_test.go}`
- [x] T299 [P] 补齐内容/通知相关 API 测试（Banner/Article/Notification）`tests/api/*_test.go`

### 测试覆盖率报告

- [x] T300 配置测试覆盖率收集和报告 `scripts/coverage.sh`
- [x] T301 更新 Makefile 添加 `make test`, `make test-unit`, `make test-integration`, `make coverage` 命令
- [x] T302 实现覆盖率门禁验证脚本 `scripts/coverage-gate.sh`，验证：（1）整体单元测试覆盖率 ≥ 80%；（2）关键业务模块（auth/payment/order/rental/booking）覆盖率 ≥ 90%；不满足条件时返回非零退出码阻止 CI/CD 流水线继续执行
- [ ] T303 跑通并达标覆盖率门禁：`make coverage-gate`（补齐缺失的单测/场景测试，直到满足阈值）
- [ ] T304 关键模块覆盖率冲刺：将 auth/payment/order/rental/booking 单测覆盖率提升到 ≥ 90%（以 `make coverage-gate` 为准）
- [ ] T305 整体单测覆盖率冲刺：将 `make coverage` 覆盖率提升到 ≥ 80%（补齐低覆盖包：admin/user/mall/content/finance/repository/pkg 等）

**Checkpoint**: 测试覆盖率达标（单测 > 80%，关键业务 > 90%）

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖，可立即开始
- **Foundational (Phase 2)**: 依赖 Phase 1 完成，**阻塞所有用户故事**
- **User Stories (Phase 3-10)**: 依赖 Phase 2 完成
  - US1 和 US2 是 P1 优先级，应先完成
  - US3-US5 是 P2 优先级
  - US6-US8 是 P3 优先级
- **Polish (Phase 11)**: 依赖所有核心用户故事完成
- **Testing (Phase 12)**: 可与 Phase 3-11 并行进行，建议每完成一个模块即编写测试

### User Story Dependencies

| Story | Priority | 可并行 | 依赖 |
|-------|----------|--------|------|
| US1 扫码租借 | P1 | ✅ | 仅依赖 Phase 2 |
| US2 设备管理 | P1 | ✅ | 仅依赖 Phase 2 |
| US3 商城购物 | P2 | ✅ | 仅依赖 Phase 2 |
| US4 酒店预订 | P2 | ✅ | 仅依赖 Phase 2 |
| US5 分销推广 | P2 | ⚠️ | 依赖 US1 的订单完成逻辑 |
| US6 财务结算 | P3 | ⚠️ | 依赖 US5 的分销数据 |
| US7 营销优惠 | P3 | ✅ | 仅依赖 Phase 2 |
| US8 会员体系 | P3 | ✅ | 仅依赖 Phase 2 |

### Parallel Opportunities

- **Phase 1**: T003-T008 可并行
- **Phase 2**: T011-T029, T033~T038, T040-T045, T047-T050, T052-T056 可并行
- **Phase 3+**: 不同用户故事可由不同开发者并行开发
- **Phase 12**: 测试任务可与功能开发并行（建议 TDD 或功能完成后立即测试）

---

## Parallel Example: Phase 2 Foundation

```bash
# 并行执行数据库迁移脚本（不同文件）:
Task: "创建 User 表迁移 migrations/000001_create_users.up.sql"
Task: "创建 Device 表迁移 migrations/000005_create_devices.up.sql"
Task: "创建 Order 表迁移 migrations/000006_create_orders.up.sql"

# 并行执行模型定义（不同文件）:
Task: "定义 User 模型 internal/models/user.go"
Task: "定义 Device 模型 internal/models/device.go"
Task: "定义 Order 模型 internal/models/order.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational（关键阻塞）
3. 执行 `make seed` 初始化测试数据
4. 完成 Phase 3: User Story 1（扫码租借）
5. 编写 US1 相关测试（Phase 12 部分任务）
6. **验证点**: 测试完整租借流程
7. 可部署/演示 MVP

### Incremental Delivery

1. Setup + Foundational + Seed Data → 基础就绪
2. 添加 US1 → 编写测试 → 部署（MVP！）
3. 添加 US2 → 编写测试 → 部署（设备管理）
4. 添加 US3/US4/US5 → 编写测试 → 部署（商城/酒店/分销）
5. 添加 US6/US7/US8 → 编写测试 → 部署（财务/营销/会员）
6. Polish + 完整测试覆盖 → 完整版本

### Parallel Team Strategy

多开发者并行：

1. 团队共同完成 Setup + Foundational + Seed Data
2. Foundational 完成后：
   - 开发者 A: User Story 1（扫码租借）+ 对应测试
   - 开发者 B: User Story 2（设备管理）+ 对应测试
   - 开发者 C: User Story 3（商城购物）+ 对应测试
3. 各故事独立完成和集成
4. 最后统一补充集成测试和 E2E 测试

---

## Notes

- [P] 任务 = 不同文件，无依赖，可并行
- [Story] 标签映射任务到具体用户故事
- 每个用户故事应可独立完成和测试
- 每个任务或逻辑组完成后提交代码
- 在任何检查点暂停以独立验证故事
- 避免：模糊任务、同一文件冲突、破坏独立性的跨故事依赖
- **Seed Data**: 种子数据来源于 `admin-frontend` 和 `user-frontend` 的 mock 数据，确保开发测试有真实数据支持
- **Testing**: 建议采用 TDD 或功能完成后立即编写测试，确保测试覆盖率达标
