# syntax=docker/dockerfile:1
# ===== Stage 1: Build Go Binary =====
FROM golang:1.26-alpine AS builder

WORKDIR /build

# 使用国内 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct

# 安装依赖（利用 Docker 缓存层 + BuildKit 持久缓存）
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download

# 复制源码并编译（利用 BuildKit 编译缓存，二次构建秒级完成）
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /build/blog-server ./cmd/server

# ===== Stage 2: 运行 =====
FROM alpine:3.19

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk --no-cache add ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY --from=builder /build/blog-server .
COPY --from=builder /build/configs ./configs

# 兜底：真实 config.yaml 已被 .dockerignore 排除（含密钥，不入镜像），
# 这里用模板补一份，保证镜像自身可启动、CI 冒烟测试能跑。
# 生产环境由 docker-compose 挂载宿主机的 configs/config.yaml 覆盖。
RUN cd /app/configs \
    && if [ ! -f config.yaml ]; then cp config.yaml.example config.yaml; fi \
    && mkdir -p /app/logs

EXPOSE 9527

CMD ["./blog-server"]
