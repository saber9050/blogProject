# syntax=docker/dockerfile:1
# ===== Stage 1: Build Go Binary =====
FROM golang:1.26-alpine AS builder

WORKDIR /build

# 使用国内 Go 模块代理，大幅加速依赖下载
ENV GOPROXY=https://goproxy.cn,direct

# 安装依赖（利用 Docker 缓存层）
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /build/blog-server ./cmd/server

# ===== Stage 2: 运行 =====
FROM alpine:3.19

# 使用阿里云镜像加速 apk 下载
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk --no-cache add ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

# 复制编译产物和配置文件
COPY --from=builder /build/blog-server .
COPY --from=builder /build/configs ./configs

# 创建日志目录
RUN mkdir -p /app/logs

EXPOSE 9527

CMD ["./blog-server"]
