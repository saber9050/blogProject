# Blog Project

个人博客全栈项目，分为前后端两个独立子项目：

| 子项目 | 目录 | 说明 |
|---|---|---|
| 后端服务 | [blog/](./blog/) | Go + Gin + GORM + MySQL/Redis，提供 RESTful API |
| 前端页面 | [vue/](./vue/) | Vue 3 + TypeScript + Vite，浏览器端 SPA |

## 快速导航

- [后端 README — 技术栈、配置、API 路由、中间件](./blog/README.md)
- [前端 README — 技术栈、目录结构、页面路由、Axios 拦截器](./vue/README.md)

## 项目架构概览

```
┌──────────┐     HTTP/API      ┌──────────┐     ORM      ┌─────────┐
│   Vue    │ ────────────────→  │   Gin    │ ──────────→  │  MySQL  │
│  SPA     │ ←──────────────── │  Backend │ ←──────────  │         │
│ :3000    │     JSON Resp     │ :9527    │     /Redis    │         │
└──────────┘                   └──────────┘               └─────────┘
```

- **前端**：Vue 3 + Pinia + Vue Router + Axios + wangEditor
- **后端**：Go + Gin + GORM + JWT + Redis + MinIO

详情见各子项目 README。

## Docker 部署

项目提供 `docker-compose.yml` 一键部署，包含 **6 个服务**：

| 服务 | 镜像 | 端口 | 说明 |
|---|---|---|---|
| `blog-frontend` | 本地构建（Vue → Nginx） | `80` | SPA + API 反向代理 |
| `blog-backend` | 本地构建（Go → Alpine） | `9527` | Gin RESTful API |
| `mysql` | `mysql:8.0` | `3306` | 持久化数据库 |
| `redis` | `redis:7-alpine` | `6379` | 缓存 + Refresh Token 存储 |
| `minio` | `minio/minio` | `9000 / 9001` | 对象存储（图片等） |
| `ollama` | `ollama/ollama` | `11434` | 本地 LLM（AI 摘要生成） |

### 快速启动

```bash
# 1. 后端配置（选 configs/config2.yaml 覆盖默认配置）
cp blog/configs/config2.yaml blog/configs/config.yaml

# 2. （可选）配置文件按需修改
#     - config.yaml 中 mysql/redis/minio/ollama 的 host 需用 Docker 内部服务名（已配置好）
#     - app.host 和 minio.base_url 改为你的服务器公网 IP

# 3. 一键构建并启动所有服务
docker compose up -d --build

# 4. 查看日志
docker compose logs -f
```

### 配置文件说明

Docker 部署专用配置：

- **数据库连接**：`database.mysql.host: mysql`、`database.redis.host: redis`（Docker 网络内部服务名）
- **MinIO**：`minio.endpoint: minio:9000`（内部连接）、`minio.base_url` 需改为服务器公网 IP（外部浏览器访问）
- **LLM**：`llm.base_url: http://ollama:11434`（Docker 网络内连接 Ollama）
- **应用地址**：`app.host` 需改为你的服务器公网 IP（邮件链接等使用）

> `configs/config2.yaml` 中所有服务连接地址已预设 Docker 内部服务名，复制为 `config.yaml` 后即可直接使用。

### 环境变量

docker-compose 支持通过 `.env` 文件或环境变量覆盖默认值：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `MYSQL_PASSWORD` | `123456` | MySQL root 密码 |
| `MYSQL_DATABASE` | `blog-project` | MySQL 数据库名 |
| `MINIO_ACCESS_KEY` | `admin` | MinIO Access Key |
| `MINIO_SECRET_KEY` | `1223349392Zxp` | MinIO Secret Key |

后端服务还通过环境变量覆盖部分配置（已预设，通常无需修改）：

| 环境变量 | 值 | 说明 |
|---|---|---|
| `CORE_COACH_DATABASE_MYSQL_HOST` | `mysql` | MySQL 服务名 |
| `CORE_COACH_DATABASE_REDIS_HOST` | `redis` | Redis 服务名 |
| `CORE_COACH_MINIO_ENDPOINT` | `minio:9000` | MinIO 内部地址 |
| `LLM_BASE_URL` | `http://ollama:11434` | Ollama API 地址 |
| `LLM_MODEL_NAME` | `gemma3:270m` | LLM 模型 |

### 访问地址

| 服务 | 地址 |
|---|---|
| 前端页面 | `http://<服务器IP>`（端口 80） |
| 后端 API | `http://<服务器IP>:9527` |
| MinIO 控制台 | `http://<服务器IP>:9001` |

### 停止与清理

```bash
# 停止所有服务
docker compose down

# 停止并删除数据卷（谨慎：会清除数据库和存储数据）
docker compose down -v
```

### 镜像加速

中国大陆服务器拉取 Docker Hub 镜像较慢，建议配置镜像加速器：

```bash
sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://dockerproxy.cn",
    "https://docker.nju.edu.cn"
  ]
}
EOF
sudo systemctl daemon-reload
sudo systemctl restart docker
```
