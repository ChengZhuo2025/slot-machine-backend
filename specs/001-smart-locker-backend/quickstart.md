# Quick Start: 爱上杜美人智能开锁管理系统后端

## 环境要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.25+ | 编程语言 |
| PostgreSQL | 15+ | 主数据库 |
| Redis | 7+ | 缓存/消息队列 |
| EMQX | 5.0+ | MQTT Broker |
| Docker | 24+ | 容器化 |
| Docker Compose | 2.20+ | 本地编排 |

## 快速开始

### 1. 克隆项目

```bash
git clone <repository-url>
cd backend
```

### 2. 启动依赖服务

```bash
# 启动 PostgreSQL、Redis、EMQX
docker-compose up -d postgres redis emqx
```

### 3. 配置环境变量

```bash
# 复制配置文件
cp configs/config.example.yaml configs/config.yaml

# 编辑配置（数据库、Redis、MQTT 等）
vim configs/config.yaml
```

### 4. 初始化数据库

```bash
# 执行数据库迁移
make migrate-up

# 或使用脚本
./scripts/migrate.sh up
```

### 5. 运行服务

```bash
# 开发模式（单体运行）
make run

# 或指定服务
go run cmd/api-gateway/main.go
```

### 6. 验证服务

```bash
# 健康检查
curl http://localhost:8080/health

# API 文档
open http://localhost:8080/swagger/index.html
```

## 项目结构

```
backend/
├── cmd/                    # 服务入口
│   └── api-gateway/        # API 网关
├── internal/               # 内部实现
│   ├── common/             # 公共组件
│   ├── models/             # 数据模型
│   ├── repository/         # 数据访问层
│   ├── service/            # 业务逻辑层
│   └── handler/            # HTTP 处理器
├── pkg/                    # 可复用包
├── docs/                   # Swagger 文档（swag 生成，供 /swagger 使用）
├── migrations/             # 数据库迁移
├── configs/                # 配置文件
├── deployments/            # 部署配置
└── tests/                  # 测试文件
```

## 常用命令

```bash
# 构建
make build

# 运行测试
make test

# 生成 API 文档
make swagger

# 代码检查
make lint

# 数据库迁移
make migrate-up
make migrate-down

# Docker 构建
make docker-build
```

## 开发规范

### 代码风格

```bash
# 格式化代码
gofmt -w .

# 静态检查
golangci-lint run
```

### 提交规范

```
feat: 新功能
fix: 修复bug
docs: 文档更新
refactor: 重构
test: 测试
chore: 其他
```

### 分支策略

- `main`: 生产分支
- `develop`: 开发分支
- `feature/*`: 功能分支
- `bugfix/*`: 修复分支
- `hotfix/*`: 紧急修复

## 配置说明

### config.yaml

```yaml
server:
  port: 8080
  mode: debug  # debug/release

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: smart_locker
  sslmode: disable

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

mqtt:
  broker: tcp://localhost:1883
  client_id: smart-locker-backend
  username: ""
  password: ""

jwt:
  secret: your-secret-key
  access_expire: 7200      # 2小时
  refresh_expire: 604800   # 7天

payment:
  wechat:
    app_id: ""
    mch_id: ""
    api_key: ""
    cert_path: ""
  alipay:
    app_id: ""
    private_key: ""
    alipay_public_key: ""

sms:
  provider: aliyun  # aliyun/tencent
  access_key: ""
  secret_key: ""
  sign_name: ""
  template_code: ""

oss:
  provider: aliyun  # aliyun/tencent
  endpoint: ""
  bucket: ""
  access_key: ""
  secret_key: ""
```

## Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: smart_locker
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  emqx:
    image: emqx/emqx:5.0
    ports:
      - "1883:1883"
      - "8083:8083"
      - "18083:18083"

  backend:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
      - emqx
    environment:
      - CONFIG_FILE=/app/configs/config.yaml

volumes:
  postgres_data:
  redis_data:
```

## API 测试

### 用户登录

```bash
# 发送验证码
curl -X POST http://localhost:8080/api/v1/auth/sms/send \
  -H "Content-Type: application/json" \
  -d '{"phone": "13800138000"}'

# 验证码登录
curl -X POST http://localhost:8080/api/v1/auth/login/sms \
  -H "Content-Type: application/json" \
  -d '{"phone": "13800138000", "code": "123456"}'
```

### 扫码获取设备

```bash
curl http://localhost:8080/api/v1/devices/D001 \
  -H "Authorization: Bearer <token>"
```

### 创建租借订单

```bash
curl -X POST http://localhost:8080/api/v1/rentals \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"device_no": "D001", "duration_hours": 2}'
```

## 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查 PostgreSQL 是否启动
   - 确认配置文件中数据库连接信息

2. **Redis 连接失败**
   - 检查 Redis 是否启动
   - 确认密码配置

3. **MQTT 连接失败**
   - 检查 EMQX 是否启动
   - 确认端口未被占用

### 日志查看

```bash
# 查看服务日志
docker-compose logs -f backend

# 查看特定服务
docker-compose logs -f postgres
```

## 参考资料

- [Gin 文档](https://gin-gonic.com/docs/)
- [GORM 文档](https://gorm.io/docs/)
- [EMQX 文档](https://www.emqx.io/docs/)
- [微信支付文档](https://pay.weixin.qq.com/wiki/doc/api/index.html)
- [支付宝文档](https://opendocs.alipay.com/open/)
