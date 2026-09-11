# Blog Project

个人博客全栈项目，前后端分离，支持 Docker Compose 一键部署，并接入 GitHub Actions 自动测试与发布。

| 子项目 | 目录 | 技术栈 |
|---|---|---|
| 后端服务 | [blog/](./blog/) | Go + Gin + GORM + MySQL + Redis + MinIO |
| 前端页面 | [vue/](./vue/) | Vue 3 + TypeScript + Vite + Pinia |

## 快速导航

- [后端 README](./blog/README.md) — 技术栈、配置项说明、API 路由、中间件
- [前端 README](./vue/README.md) — 技术栈、目录结构、页面路由、Axios 拦截器
- [CI/CD 说明](./docs/CI-CD.md) — 流水线设计、Secrets 清单、部署排错手册

## 架构概览

开发环境下前端独立运行，通过代理访问后端：

```text
┌──────────────┐    HTTP /api     ┌──────────────┐    GORM     ┌───────────┐
│  Vue SPA     │ ───────────────→ │  Gin Backend │ ──────────→ │   MySQL   │
│  :3000       │ ←─────────────── │  :9527       │             └───────────┘
└──────────────┘   JSON Response  └──────────────┘    ↑ 缓存            ↑
                                          │        ┌───────────┐  ┌───────────┐
                                          └──────→ │   Redis   │  │   MinIO   │
                                                   └───────────┘  └───────────┘
```

生产环境（Docker）下前端由 Nginx 托管并监听 `80`，同时把 `/api` 反向代理到后端 `9527`；
后端与 MySQL / Redis / MinIO / Ollama 通过 `blog-network` 内部网络以服务名互访，数据库端口不对公网开放。

- **后端**：Go + Gin + GORM + JWT（双 Token）+ Redis + MinIO + Ollama
- **前端**：Vue 3 + TypeScript + Vite + Pinia + Vue Router + Axios + wangEditor

详情见各子项目 README。

## 目录结构

```text
.
├── blog/                     # 后端（Go）
│   ├── configs/              # 配置模板与本地配置
│   └── cmd/server/           # 程序入口
├── vue/                      # 前端（Vue 3 + Vite）
├── scripts/
│   ├── setup-server.sh       # 服务器初始化（Docker / swap / 镜像加速）
│   └── deploy.sh             # 部署脚本（健康检查 + 失败回滚）
├── docs/CI-CD.md             # CI/CD 完整说明
├── docker-compose.yml        # 本地/自建部署：镜像本地构建
└── docker-compose.prod.yml   # 生产部署：直接拉取 ACR 镜像，供 CI/CD 使用
```

## 快速开始（Docker Compose）

### 1. 准备配置文件

项目有两个**不入库**的配置文件，需要从模板复制后填写真实值：

```bash
# 环境变量：数据库密码、MinIO 凭证等，供 docker compose 读取
cp .env.example .env

# 后端配置：连接信息、JWT 密钥、SMTP 授权码等
cp blog/configs/config.yaml.example blog/configs/config.yaml
```

`.env` 中需要修改的项：

| 变量 | 说明 |
|---|---|
| `MYSQL_PASSWORD` | MySQL root 密码 |
| `MYSQL_DATABASE` | 数据库名，默认 `blog-project` |
| `REDIS_PASSWORD` | Redis 密码，留空表示不启用 |
| `JWT_SECRET` | JWT 签名密钥，建议随机长字符串 |
| `MINIO_ACCESS_KEY` | MinIO 账号 |
| `MINIO_SECRET_KEY` | MinIO 密码，请改为强密码 |
| `MINIO_PUBLIC_URL` | 浏览器访问图片的地址，如 `http://<服务器IP>:9000` |
| `LLM_MODEL_NAME` | Ollama 模型名，默认 `gemma3:270m` |

然后按需修改 `blog/configs/config.yaml`，至少要改 `app.host` 和 `email` 相关项。

> ⚠️ **`.env` 与 `config.yaml` 里的密码必须一致。**
> MySQL / MinIO 容器用 `.env` 的值初始化，而后端是读 `config.yaml` 去连接的，两边对不上会连接失败。

### 2. 启动服务

```bash
docker compose up -d --build

# 查看启动日志
docker compose logs -f
```

### 服务清单

`docker-compose.yml` 包含 **6 个服务**：

| 服务 | 镜像 | 端口 | 说明 |
|---|---|---|---|
| `blog-frontend` | 本地构建（Vue → Nginx） | `80` | SPA 静态资源 + `/api` 反向代理 |
| `blog-backend` | 本地构建（Go → Alpine） | `9527` | Gin RESTful API |
| `mysql` | `mysql:8.0` | `3306` | 持久化数据库 |
| `redis` | `redis:7-alpine` | `6379` | 缓存 + Refresh Token 存储 |
| `minio` | `minio/minio` | `9000 / 9001` | 对象存储（文章图片等） |
| `ollama` | `ollama/ollama` | `11434` | 本地 LLM（一键生成摘要） |

### 访问地址

| 服务 | 地址 |
|---|---|
| 前端页面 | `http://<服务器IP>` |
| 后端 API | `http://<服务器IP>:9527` |
| 健康检查 | `http://<服务器IP>:9527/api/v1/health` |
| MinIO 控制台 | `http://<服务器IP>:9001` |

> 生产环境请在安全组中只放行 `80` 与 `9527`；`3306`、`6379` 以及 MinIO 的 `9000/9001` 不建议对公网开放。

## 配置文件说明

Docker 部署时，需要调整 `config.yaml` 中的这几处（其余保持默认即可）：

| 配置项 | 建议值 | 说明 |
|---|---|---|
| `database.mysql.host` | `mysql` | 使用 Compose 内部服务名 |
| `database.redis.host` | `redis` | 使用 Compose 内部服务名 |
| `minio.endpoint` | `minio:9000` | 容器内部连接地址 |
| `minio.base_url` | `http://<服务器IP>:9000` | 图片的公网访问地址 |
| `llm.base_url` | `http://ollama:11434` | 容器内部连接 Ollama |
| `app.host` | `<服务器IP>` | 邮件链接等场景使用 |

> 其中 `mysql` / `redis` / `minio:9000` / `ollama:11434` 这几项已在 compose 中通过环境变量覆盖，
> 即使 `config.yaml` 里写的是 `127.0.0.1` 也能正常工作；`minio.base_url` 和 `app.host` 因为要暴露给浏览器，必须手改。

## 环境变量

除 `.env` 之外，`docker-compose.yml` 还会向后端容器注入以下变量，用于覆盖配置文件中的连接地址（已预设好，通常无需修改）：

| 环境变量 | 值 | 说明 |
|---|---|---|
| `CORE_COACH_DATABASE_MYSQL_HOST` | `mysql` | MySQL 服务名 |
| `CORE_COACH_DATABASE_MYSQL_PORT` | `3306` | MySQL 端口 |
| `CORE_COACH_DATABASE_REDIS_HOST` | `redis` | Redis 服务名 |
| `CORE_COACH_DATABASE_REDIS_PORT` | `6379` | Redis 端口 |
| `CORE_COACH_MINIO_ENDPOINT` | `minio:9000` | MinIO 内部地址 |
| `LLM_BASE_URL` | `http://ollama:11434` | Ollama API 地址 |
| `LLM_MODEL_NAME` | `gemma3:270m` | LLM 模型 |
| `LLM_KEEP_ALIVE` | `30m` | 模型常驻内存时长 |

生产环境使用的 `docker-compose.prod.yml` 在此基础上额外注入 `MYSQL_USER`、`MYSQL_PASSWORD`、
`REDIS_PASSWORD`、`JWT_SECRET` 以及 MinIO 相关凭证，全部来自服务器上的 `.env`。

## 本地开发

```bash
# 后端
cd blog/cmd/server && go run main.go

# 前端（独立启动，端口 3000）
cd vue && npm install && npm run dev
```

前端开发服务器已配置代理，`/api` 请求会转发到 `http://localhost:9527`。

## CI/CD

项目已接入 GitHub Actions：push 到 `main` 且 CI 全绿后，自动构建镜像推送到阿里云 ACR，
再 SSH 到 ECS 完成滚动更新与健康检查，失败自动回滚到上一个版本。

```text
push → CI（go vet / gofmt / go test -race / vue-tsc / vitest / 构建）
     → 构建镜像并推送 ACR（tag = commit sha）
     → 镜像冒烟测试 → SSH 部署 → 健康检查 → 失败自动回滚
```

完整配置步骤、Secrets 清单与排错手册见 [docs/CI-CD.md](./docs/CI-CD.md)。

## 常用命令

```bash
# 查看容器状态
docker compose ps

# 查看某个服务日志
docker compose logs -f blog-backend

# 重启单个服务（不影响数据库）
docker compose up -d blog-backend blog-frontend

# 停止所有服务
docker compose down

# 停止并删除数据卷（谨慎：会清除数据库与对象存储数据）
docker compose down -v
```

## 镜像加速

中国大陆服务器拉取 Docker Hub 镜像较慢，建议配置加速器：

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
