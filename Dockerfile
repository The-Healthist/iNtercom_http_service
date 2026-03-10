# Build stage
FROM docker.1ms.run/golang:1.23.0-alpine AS builder

# 设置Go环境变量
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

LABEL version="2.3.0" \
    description="intercom_http_service" \
    maintainer="Stone Sea"

WORKDIR /app

# 复制全部源码（含 vendor 目录），无需网络下载依赖
COPY . .

# 使用 vendor 模式构建，零网络依赖
RUN go build -mod=vendor -ldflags="-s -w" -o main ./cmd/server

# Final stage
FROM docker.1ms.run/alpine:latest

WORKDIR /app

LABEL version="2.3.0" \
    description="intercom_http_service" \
    maintainer="Stone Sea"

# 从 builder 复制 TLS 证书（golang:alpine 已预装 ca-certificates）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 使用 UTC 时区，无需 tzdata 包
ENV TZ=UTC

# 创建目录结构
RUN mkdir -p /app/cmd/server /app/logs /app/docs

# Copy binary and docs from builder
COPY --from=builder /app/main /app/cmd/server/main
COPY --from=builder /app/docs /app/docs

# Set executable permissions
RUN chmod +x /app/cmd/server/main

EXPOSE 8080

# 更全面的健康检查
# wget 是 alpine busybox 内置的，无需额外安装
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/ping || exit 1

# 使用重构后的入口点
ENTRYPOINT ["/app/cmd/server/main"] 