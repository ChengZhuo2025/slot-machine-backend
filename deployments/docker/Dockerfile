# =====================================================
# 多阶段构建 Dockerfile
# 爱上杜美人智能开锁管理系统后端服务
# =====================================================

# Stage 1: Build
FROM golang:1.25-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
ARG SERVICE=api-gateway
ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT_HASH

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH}" \
    -o /app/bin/${SERVICE} \
    ./cmd/${SERVICE}

# Stage 2: Runtime
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata curl

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从 builder 阶段复制二进制文件
ARG SERVICE=api-gateway
COPY --from=builder /app/bin/${SERVICE} /app/server
COPY --from=builder /app/configs /app/configs

# 创建日志目录
RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 启动命令
ENTRYPOINT ["/app/server"]
