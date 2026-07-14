# syntax=docker/dockerfile:1
# ===== Stage 1: Build Vue =====
FROM node:18-alpine AS builder

WORKDIR /build

# 使用国内 npm 镜像
RUN npm config set registry https://registry.npmmirror.com

# 安装依赖（利用 Docker 缓存层 + BuildKit 持久缓存）
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

# 复制源码并构建
COPY . .
RUN --mount=type=cache,target=/root/.npm \
    --mount=type=cache,target=/build/node_modules/.cache \
    npm run build

# ===== Stage 2: Nginx 运行 =====
FROM nginx:alpine

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk --no-cache add tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

COPY --from=builder /build/dist /usr/share/nginx/html

COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
