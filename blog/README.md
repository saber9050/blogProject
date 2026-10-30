# Blog - 后端服务

基于 Go + Gin + GORM 的个人博客后端 API 服务。

## 技术栈

| 组件 | 技术 |
|---|---|
| 语言 | Go 1.26 |
| 框架 | Gin v1.12 |
| ORM | GORM v1.31 + PostgreSQL 17（pgvector） |
| 缓存 | Redis (go-redis v9) |
| JWT | golang-jwt v5 |
| 配置 | Viper |
| 日志 | Zap + Lumberjack |
| 存储 | MinIO（图片/文件） |
| 邮件 | gomail |
| 验证码 | base64Captcha |
| AI 能力 | 大模型（OpenAI 兼容协议，后台可配置）：文章摘要生成、正文向量化（pgvector） |

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
│       ├── chunker/     # 正文切分（纯函数，可按单测钉死边界）
│       ├── vectorize/   # 向量化任务链路（队列 + worker）
│       ├── vectorschema/# 向量表 DDL 与维度维护
│       ├── llmconfig/   # 模型配置与连接测试
│       ├── chunk/       # 切分结果查询与整节拼接
│       ├── article/     # 文章
│       ├── auth/        # 认证
│       ├── user/        # 用户
│       └── llm/         # 摘要生成
├── pkg/                 # 公共工具包
│   ├── config/          # 配置加载
│   ├── database/        # PostgreSQL/Redis 初始化、向量表 DDL
│   ├── email/           # 邮件发送
│   ├── errors/          # 业务错误码
│   ├── jwt/             # JWT 令牌
│   ├── logger/          # 日志
│   ├── minio/           # MinIO 对象存储
│   ├── llmclient/       # 大模型客户端（chat/completions + embeddings）
│   ├── response/        # 统一 HTTP 响应
│   └── utils/           # 工具函数（密码哈希、RSA 加密等）
├── logs/                # 运行日志
├── docs/                # API 文档（按模块拆分，含 DEVELOPMENT.md）
└── go.mod
```

## 快速开始

### 前置条件

- Go 1.26+
- PostgreSQL 17+（需支持 pgvector 扩展，推荐镜像 `pgvector/pgvector:pg17`）
- Redis 7.0+
- MinIO（可选，用于图片/文件存储）
- 大模型（AI 摘要等用）**在后台「AI 模型」页配置**，不走配置文件

### 配置

复制配置文件并修改：

```bash
cp configs/config.yaml.example configs/config.yaml
```

| 配置 | 说明 |
|---|---|
| `app.port` | 服务端口（默认 9527） |
| `app.mode` | 运行模式（debug / release / test） |
| `database.postgres` | PostgreSQL 连接信息（含连接池大小） |
| `database.redis` | Redis 连接信息（含 DB 编号、连接池大小） |
| `jwt.secret` | JWT 签名密钥 |
| `jwt.access_expire_minutes` | Access Token 过期时间（分钟），默认 15 |
| `jwt.refresh_expire_hours` | Refresh Token 过期时间（小时），默认 72 |
| `log.*` | 日志级别、文件路径、轮转策略 |
| `cors.*` | CORS 跨域配置 |
| `email` | SMTP 邮件发送配置 |
| `minio` | MinIO 对象存储配置 |
| `default_admin` | 首次启动自动创建管理员 |

> 大模型**不通过配置文件配置**：请在后台「AI 模型」页维护（存库，需测试连接通过才能保存）。
> 未配置任何模型时，调用需要模型的接口（如一键生成摘要）会返回「请先配置模型」。

支持环境变量覆盖敏感字段：`PG_HOST`、`PG_PORT`、`PG_USER`、`PG_PASSWORD`、`PG_DATABASE`、
`REDIS_PASSWORD`、`JWT_SECRET`、`COZE_API_KEY`、`CRYPTO_RSA_PRIVATE_KEY`、`MINIO_PUBLIC_URL`。

### 运行

```bash
cd cmd/server
go run main.go
```

## 项目架构

采用分层架构：**控制器 (Controller) → 服务 (Service) → 仓储 (Repository) → 数据库**，通过依赖注入组装。

## API 路由

### 健康检查

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/health` | 健康检查 |

### 认证 `/api/v1/auth/`

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/register` | 注册 |
| POST | `/auth/login` | 密码登录 |
| POST | `/auth/email_login` | 邮箱验证码登录 |
| GET | `/auth/image_captcha` | 图形验证码 |
| POST | `/auth/captcha` | 发送邮箱验证码 |
| POST | `/auth/reset_password` | 重置密码 |
| POST | `/auth/logout` | 登出（TID 比对，仅清当前会话） |
| POST | `/auth/refresh` | 刷新令牌（双 Token 轮换） |
| GET | `/auth/is_exists_name` | 检查用户名是否已存在 |
| GET | `/auth/is_exists_account` | 检查账号是否已存在 |

### 用户 `/api/v1/user/`（需登录）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/user/info` | 获取个人信息 |
| POST | `/user/profile` | 编辑个人信息 |
| POST | `/user/avatar` | 更换头像（MinIO 上传） |
| POST | `/user/password` | 修改密码 |
| POST | `/user/email_ack` | 请求修改邮箱（发送验证码） |
| POST | `/user/add_email` | 添加邮箱 |
| GET | `/user/email` | 确认邮箱变更（无状态链接） |
| GET | `/user/is_exists_email` | 检查邮箱是否已存在 |

### 文章 `/api/v1/articles/`

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| GET | `/articles` | 文章列表（支持 sort / category_id / tag_ids / keyword 参数） | 可选 |
| GET | `/articles/:id` | 文章详情 | 可选 |
| GET | `/articles/random` | 随机一篇文章 | - |
| GET | `/articles/stats` | 文章统计（总文章数、浏览量、点赞数） | - |
| POST | `/articles/:id/like` | 点赞文章 | 需登录 |
| DELETE | `/articles/:id/like` | 取消点赞 | 需登录 |

### 分类 `/api/v1/categories/`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/categories` | 分类列表 |
| GET | `/categories/count` | 启用分类数量 |

### 标签 `/api/v1/tags/`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/tags` | 标签列表 |
| GET | `/tags/count` | 启用标签数量 |

### 评论 `/api/v1/articles/comments/`

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| GET | `/articles/:id/comments` | 一级评论列表 | - |
| GET | `/articles/:id/comments/:commentId/replies` | 二级回复列表 | - |
| POST | `/articles/:id/comments` | 发表评论 | 需登录 |
| DELETE | `/articles/:id/comments/:commentId` | 删除评论 | 需登录 |

### 关于 `/api/v1/about/`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/about` | 获取关于页面信息 |
| PUT | `/about` | 更新关于页面（管理员） |

### 后台管理 `/api/v1/admin/`（需管理员）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET/POST/PUT/DELETE | `/admin/articles` | 文章 CRUD |
| PUT | `/admin/articles/transfer` | 文章分类一键转移 |
| GET/POST/PUT/DELETE | `/admin/categories` | 分类 CRUD |
| GET/POST/PUT/DELETE | `/admin/tags` | 标签 CRUD |
| GET | `/admin/comments` | 评论列表 |
| DELETE | `/admin/comments` | 批量删除评论 |
| DELETE | `/admin/comments/:id` | 单个删除评论 |
| GET/POST/PUT/DELETE | `/admin/users` | 用户 CRUD |
| POST | `/admin/upload` | 图片上传（MinIO） |
| POST | `/admin/articles/generate-summary` | AI 生成文章摘要（请求体指定 `config_id`，选择使用哪条模型配置） |
| GET/POST/PUT/DELETE | `/admin/llm-configs` | 大模型配置 CRUD（`?type=1` 对话模型 / `?type=2` 向量模型） |
| POST | `/admin/llm-configs/test` | 测试模型连接（不落库） |
| POST | `/admin/articles/:id/vectorize` | 提交向量化（异步，立即返回） |
| GET | `/admin/articles/:id/vector-status` | 查询向量化状态与块数 |
| GET | `/admin/vector/schema` | 向量表是否存在、当前维度、块数 |
| POST | `/admin/vector/reset` | 删除向量表（仅换维度时手工调用，前端无入口） |
| GET | `/admin/chunks` | 切分块明细（支持 `article_id`、`keyword`） |
| GET | `/admin/chunks/stat` | 按文章聚合的切分统计 |
| GET | `/admin/chunks/section` | 取整节内容（父子块的父块） |
| POST | `/admin/chunks/articles/delete` | 批量删除文章向量 |

> 请求/响应示例见各模块文档，见下方「API 文档」一节。

## 中间件

| 中间件 | 说明 |
|---|---|
| Recovery | panic 恢复，Zap 记录堆栈 |
| Logger | HTTP 请求日志（方法、状态码、延迟、IP） |
| CORS | 动态跨域配置，支持预检请求和凭据 |
| Auth(roleID) | JWT 验证 + 角色鉴权 |
| OptionalAuth() | 可选认证——有 token 注入用户上下文，无 token 也放行 |

## 主要功能

- 用户注册/登录/密码重置（邮箱验证码）
- 邮箱验证码登录 / 双 Token 认证（Access Token + Refresh Token 自动轮换）
- JWT 认证 + 角色鉴权（普通用户 / 管理员），Refresh Token 校验实现单设备踢出
- Token 自动刷新：Access Token 过期后通过过期 JWT 中的 `tid` 字段提取 refresh token 进行轮换
- 文章 CRUD + **Markdown 编辑与渲染**（`content` 存 Markdown 源码，不再是 HTML）
- 随机一篇文章
- 文章点赞/取消点赞
- 文章统计（总文章数、总浏览量、总点赞数）
- 嵌套评论/回复（支持二级回复）
- 分类管理（含禁用检查和文章一键转移）
- 标签管理
- 个人中心（头像上传、邮箱修改）
- 关于页面管理
- 对象存储（MinIO 图片/文件上传）
- 图形验证码 + 频率限制
- 日志分级归档 + 轮转
- 启动时自动创建默认管理员
- AI 生成文章摘要（后台编辑器调用；使用哪条模型配置由调用方在选择框中指定）
- 大模型配置管理（后台「AI 模型」页：CRUD + 测试连接；新增配置测试通过才允许入库）
- **正文向量化**：按 Markdown 结构切分 → embedding → 存 pgvector，为将来的检索问答做准备
- **切分管理**：块明细分页、按文章统计、整节回看（验证「小块拼回原文」是否通顺）

## 数据格式与向量化

### 正文是 Markdown

`articles.content` 存的是 **Markdown 源码**，不是 HTML。

- 编辑端用 `md-editor-v3`（左源码右预览），落库就是编辑框里的文本
- 前台用 `MdPreview` 渲染，不用 `v-html`
- 做切分/摘要前会先剥掉 Markdown 标记（`#` `**` `|` ```` ``` ```` 等），代码块内容保留

### 切分规则

按 Markdown 标题切小节，每块前置标题路径：

```text
Redis 持久化策略 > 二、AOF

AOF 是追加写日志……
```

| 参数 | 值 | 说明 |
|---|---|---|
| 块大小 | 800 字 | 小节内超长才降级到「段落 → 行 → 句号 → 逗号」逐级切 |
| 重叠 | 80 字 | 防止句子卡在块边界上 |
| 碎块阈值 | 100 字 | 低于此值并入同小节前一块 |
| 表格/代码块 | 原子 | 永不切断，允许超长 |

标题路径会拼进块正文参与 embedding——中文技术博客的查询词往往就是小节标题。

### 父子块

同一小节切出的多块共享 `section_index`，检索命中任一块后可按小节取回整节：

```
GET /api/v1/admin/chunks/section?article_id=19&section_index=2
```

拼接时自动去掉重复的路径前缀、裁掉块间重叠。这是「小块检索、整节喂模型」的数据基础。

### 处理链路

```text
POST /admin/articles/:id/vectorize
  → 条件 UPDATE 抢锁（防重复提交）
  → 内存队列（容量 64）
  → 单 worker 串行：切分 → 调 embedding → 事务内「删旧块 + 插新块 + 回写状态」
  → GET /admin/articles/:id/vector-status 轮询
```

未配置向量模型时提交会返回 `503`。

## API 文档

接口文档按模块拆分，位于 `docs/` 目录：

| 文档 | 说明 |
|---|---|
| [article_api.md](./docs/article_api.md) | 前台文章（列表、详情、随机、点赞、统计） |
| [auth_api.md](./docs/auth_api.md) | 认证（注册、登录、验证码、密码重置、双 Token） |
| [user_api.md](./docs/user_api.md) | 用户信息 |
| [comment_api.md](./docs/comment_api.md) | 评论与回复 |
| [about_api.md](./docs/about_api.md) | 关于页面 |
| [admin_article_api.md](./docs/admin_article_api.md) | 后台文章（CRUD、摘要、**向量化提交与状态**） |
| [admin_category_api.md](./docs/admin_category_api.md) | 后台分类 |
| [admin_tag_api.md](./docs/admin_tag_api.md) | 后台标签 |
| [admin_comment_api.md](./docs/admin_comment_api.md) | 后台评论 |
| [admin_user_api.md](./docs/admin_user_api.md) | 后台用户 |
| [admin_llmconfig_api.md](./docs/admin_llmconfig_api.md) | **后台 AI 模型配置**（对话/向量模型、连接测试） |
| [admin_vector_api.md](./docs/admin_vector_api.md) | **后台向量表**（状态查询、清空重建） |
| [admin_chunk_api.md](./docs/admin_chunk_api.md) | **后台切分管理**（块明细、统计、整节回看、批量删除） |
| [DEVELOPMENT.md](./docs/DEVELOPMENT.md) | 开发指南 |

## 扩展开发

如果你想在此基础上开发新功能，项目提供了分层架构和完整的开发示例。

开发新功能的标准流程：

1. **定义数据模型** — `internal/model/entity/` 创建 GORM 实体
2. **创建 DTO** — `internal/model/dto/request/` 和 `internal/model/dto/response/` 定义请求/响应结构
3. **实现 Repository** — `internal/repository/` 数据访问层
4. **实现 Service** — `internal/service/` 业务逻辑层
5. **创建 Controller + 路由** — `internal/api/v1/<name>/` 控制器和路由注册
6. **注册到应用** — 更新 `internal/app/app.go` 依赖注入

详细的分步示例（含完整代码）见 [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md)。
