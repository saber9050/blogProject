# ===== Stage 1: Build Vue =====
FROM node:18-alpine AS builder

WORKDIR /build

# 安装依赖（利用 Docker 缓存层）
COPY package.json package-lock.json ./
RUN npm ci

# 复制源码并构建
COPY . .
RUN npm run build

# ===== Stage 2: Nginx 运行 =====
FROM nginx:alpine

RUN apk --no-cache add tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 复制构建产物
COPY --from=builder /build/dist /usr/share/nginx/html

# Nginx 配置：SPA 路由 + API 代理
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
