# Blog - 后端服务

基于 Go + Gin + GORM 的个人博客后端 API 服务。

## 技术栈

| 组件 | 技术 |
|---|---|
| 语言 | Go 1.26 |
| 框架 | Gin v1.12 |
| ORM | GORM v1.31 + MySQL |
| 缓存 | Redis (go-redis v9) |
| JWT | golang-jwt v5 |
| 配置 | Viper |
| 日志 | Zap + Lumberjack |
| 存储 | MinIO |
| 邮件 | gomail |
| 验证码 | base64Captcha |

## 目录结构

```
blog/
├── cmd/server/          # 主入口
├── configs/             # 配置文件
├── internal/            # 应用内部代码
│   ├── api/             # HTTP 路由和控制器
│   │   ├── router.go    # 全局路由组装
│   │   └── v1/          # API v1 版本
│   │       ├── admin/   # 后台管理接口
│   │       ├── article/ # 前台文章接口
│   │       ├── auth/    # 认证接口
│   │       ├── category/# 分类接口
│   │       ├── comment/ # 评论接口
│   │       ├── tag/     # 标签接口
│   │       └── user/    # 用户接口
│   ├── app/             # 应用生命周期
│   ├── cache/           # 缓存层 (Redis)
│   ├── constant/        # 常量定义
│   ├── middleware/      # Gin 中间件
│   ├── model/           # 数据模型
│   │   ├── dto/request/ # 请求 DTO
│   │   ├── dto/response/# 响应 DTO
│   │   └── entity/      # GORM 实体
│   ├── repository/      # 数据访问层
│   └── service/         # 业务逻辑层
├── pkg/                 # 公共工具包
│   ├── config/          # 配置加载
│   ├── database/        # MySQL/Redis 初始化
│   ├── email/           # 邮件发送
│   ├── errors/          # 业务错误码
│   ├── jwt/             # JWT 令牌
│   ├── logger/          # 日志
│   ├── minio/           # MinIO 对象存储
│   ├── response/        # 统一 HTTP 响应
│   └── utils/           # 工具函数
├── logs/                # 运行日志
├── docs/                # API 接口文档
└── go.mod
```

## 快速开始

### 前置条件

- Go 1.26+
- MySQL 8.0+
- Redis 7.0+
- MinIO（可选，用于图片/文件存储）

### 配置

复制配置文件并修改：

```bash
cp configs/config.yaml.example configs/config.yaml
```

配置项说明：

| 配置 | 说明 |
|---|---|
| `database.mysql` | MySQL 连接信息 |
| `database.redis` | Redis 连接信息 |
| `jwt.secret` | JWT 签名密钥 |
| `jwt.expire_hours` | Token 过期时间 |
| `email` | SMTP 邮件发送配置 |
| `minio` | MinIO 对象存储配置 |
| `app.port` | 服务端口（默认 9527） |

支持环境变量覆盖敏感字段：`MYSQL_PASSWORD`、`REDIS_PASSWORD`、`JWT_SECRET`、`COZE_API_KEY`、`CRYPTO_RSA_PRIVATE_KEY`。

### 运行

```bash
cd cmd/server
go run main.go
```

## 项目架构

采用分层架构：**控制器 (Controller) → 服务 (Service) → 仓储 (Repository) → 数据库**，通过依赖注入组装。

### 路由结构

- `GET /api/v1/health` — 健康检查
- `/api/v1/auth/*` — 认证（登录、注册、验证码、密码重置）
- `/api/v1/users/*` — 用户信息（需登录）
- `/api/v1/articles/*` — 前台文章（含 `GET /random` 随机一篇文章）
- `/api/v1/categories/*` — 分类
- `/api/v1/tags/*` — 标签
- `/api/v1/articles/comments/*` — 评论
- `/api/v1/admin/*` — 后台管理（需管理员权限）
- `/api/v1/abouts/*` — 关于页面

### API 文档

接口文档位于 `docs/` 目录：

| 文档 | 说明 |
|---|---|
| `article_api.md` | 前台文章（列表、详情、随机文章） |
| `auth_api.md` | 认证相关（登录、注册、验证码、密码重置） |
| `user_api.md` | 用户信息 |
| `comment_api.md` | 评论 |
| `admin_article_api.md` | 后台文章管理 |
| `admin_category_api.md` | 后台分类管理 |
| `admin_tag_api.md` | 后台标签管理 |
| `admin_user_api.md` | 后台用户管理 |
| `about_api.md` | 关于页面 |
| `DEVELOPMENT.md` | 开发指南 |

## 主要功能

- 用户注册/登录/密码重置（邮箱验证码）
- JWT 认证 + 角色鉴权（普通用户 / 管理员）
- 文章 CRUD + 富文本编辑
- 随机一篇文章
- 分类管理（含禁用检查和文章一键转移）
- 评论管理（级联删除）
- 标签管理
- 个人中心（邮箱修改）
- 对象存储（MinIO 图片/文件上传）
- 图形验证码 + 频率限制
- 日志分级归档
