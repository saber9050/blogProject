# API 文档

后端 HTTP 接口清单。所有接口挂在 `/api/v1` 下，请求与响应均为 JSON。

- [1. 通用约定](#1-通用约定)
- [2. 认证 `/auth`](#2-认证-auth)
- [3. 用户 `/user`](#3-用户-user)
- [4. 前台：文章 / 分类 / 标签 / 评论 / 关于](#4-前台接口)
- [5. 后台管理 `/admin`](#5-后台管理-admin)
- [6. 附录](#6-附录)

---

## 1. 通用约定

### 1.1 响应结构

所有接口统一返回：

```json
{
  "code": 0,
  "message": "成功",
  "data": {}
}
```

`code` 为 `0` 表示成功；非 0 时 `message` 是给用户看的中文提示，`data` 通常为 `null`。

### 1.2 错误码

| 段位 | 含义 | 示例 |
|---|---|---|
| `0` | 成功 | — |
| `400/401/403/404/500/503` | 直接沿用 HTTP 语义 | `401` 未登录、`503` 依赖未就绪 |
| `1xxx` | 用户相关 | `1001` 用户不存在 |
| `2xxx` | 参数相关 | `2001` 参数校验失败 |
| `3xxx` | 资源相关 | `3001` 文章不存在 |

### 1.3 认证

- 登录成功后拿到 `access_token` 与 `refresh_token`
- 业务接口带请求头：`Authorization: Bearer <access_token>`
- `access_token` 过期（`401`）时，用 `POST /auth/refresh` 换新的
- 部分前台接口是**可选认证**（`OptionalAuth`）：带了 token 就解析出 `user_id`（用于判断是否点赞），不带也正常返回

### 1.4 ⚠️ 文章正文是 Markdown

**`content` 字段存的是 Markdown 源码，不是 HTML。**

```json
{
  "title": "Redis 持久化策略",
  "content": "## 一、RDB\n\nRDB 是把内存快照写入磁盘。\n\n| 方案 | 优点 |\n| --- | --- |\n| RDB | 文件紧凑 |"
}
```

- 编辑端用 `md-editor-v3`，左栏写源码、右栏实时预览，落库就是编辑框里的文本
- 前台用 `MdPreview` 渲染，**不要再用 `v-html`**
- 后端做切分/摘要时会先剥掉 Markdown 标记（`#`、`**`、`|`、```` ``` ```` 等），代码块内容保留

### 1.5 分页

列表接口统一用 `page`（从 1 起）与 `page_size`，响应形如：

```json
{
  "list": [],
  "total": 128,
  "page": 1,
  "page_size": 10
}
```

---

## 2. 认证 `/auth`

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| POST | `/auth/register` | 否 | 注册 |
| POST | `/auth/login` | 否 | 账号密码登录 |
| POST | `/auth/email_login` | 否 | 邮箱验证码登录 |
| POST | `/auth/captcha` | 否 | 发送邮箱验证码 |
| GET | `/auth/image_captcha` | 否 | 获取图形验证码 |
| POST | `/auth/reset_password` | 否 | 重设密码 |
| POST | `/auth/refresh` | 否 | 用 refresh_token 换新的 access_token |
| POST | `/auth/logout` | 是 | 登出（token 拉黑） |
| GET | `/auth/is_exists_name` | 否 | 昵称唯一性检查 |
| GET | `/auth/is_exists_account` | 否 | 账号唯一性检查 |

### POST /auth/login

```json
{ "account": "admin", "password": "***", "captcha_id": "xxx", "captcha_code": "8f3a" }
```

响应 `data`：

```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "user": { "id": 1, "name": "admin", "avatar_url": "", "role_id": 1 }
}
```

> `role_id = 1` 是管理员，后台接口都要求这个角色。

---

## 3. 用户 `/user`

全部需要登录。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/user/info` | 当前用户信息 |
| POST | `/user/profile` | 修改昵称等资料 |
| POST | `/user/avatar` | 更换头像（multipart） |
| POST | `/user/password` | 修改密码 |
| POST | `/user/add_email` | 添加邮箱 |
| POST | `/user/email_ack` | 修改邮箱请求 |
| GET | `/user/is_exists_email` | 邮箱唯一性检查 |
| GET | `/user/email` | 邮箱确认链接落地（无登录态，靠 token） |

---

## 4. 前台接口

### 4.1 文章 `/articles`

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/articles` | 可选 | 文章列表，支持 `keyword`（匹配标题与摘要）、`type_id`、`tag_id`、分页 |
| GET | `/articles/:id` | 可选 | 文章详情（`content` 为 Markdown） |
| GET | `/articles/random` | 否 | 随机一篇 |
| GET | `/articles/stats` | 否 | 文章总数等统计 |
| POST | `/articles/:id/like` | 是 | 点赞 |
| DELETE | `/articles/:id/like` | 是 | 取消点赞 |

> 列表与详情**只返回已发布文章**（`status = 1`）。详情接口每次调用会给浏览量 +1。

### 4.2 分类 / 标签

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/categories` | 启用的分类列表 |
| GET | `/categories/count` | 启用的分类数量 |
| GET | `/tags` | 启用的标签列表 |
| GET | `/tags/count` | 启用的标签数量 |

### 4.3 评论 `/articles/:id/comments`

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/articles/:id/comments` | 否 | 一级评论列表 |
| GET | `/articles/:id/comments/:commentId/replies` | 否 | 二级回复列表 |
| POST | `/articles/:id/comments` | 是 | 发表评论 |
| DELETE | `/articles/:id/comments/:commentId` | 是 | 删除自己的评论 |

### 4.4 关于 `/about`

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/about` | 否 | 关于页内容 |

---

## 5. 后台管理 `/admin`

**全部需要管理员**（`role_id = 1`），未登录或权限不足返回 `401`。

### 5.1 用户 `/admin/users`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/admin/users` | 用户列表 |
| POST | `/admin/users` | 新建用户 |
| PUT | `/admin/users/:id` | 更新用户 |
| DELETE | `/admin/users/:id` | 删除用户 |

### 5.2 文章 `/admin/articles`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/admin/articles` | 列表（含草稿） |
| POST | `/admin/articles` | 新建 |
| PUT | `/admin/articles/:id` | 更新 |
| DELETE | `/admin/articles/:id` | 删除 |
| POST | `/admin/articles/generate-summary` | 一键生成摘要 |
| PUT | `/admin/articles/transfer` | 批量转移分类 |
| POST | `/admin/articles/:id/vectorize` | 提交向量化 |
| GET | `/admin/articles/:id/vector-status` | 查询向量化状态 |
| POST | `/admin/upload` | 上传图片（封面与正文共用） |

#### POST /admin/articles

```json
{
  "title": "Redis 持久化策略",
  "type_id": 3,
  "tag_ids": [1, 5],
  "cover_url": "https://.../cover.png",
  "summary": "RDB 与 AOF 的取舍",
  "content": "## 一、RDB\n\nRDB 是把内存快照写入磁盘……",
  "status": 0
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `title` | 是 | 最长 200 |
| `type_id` | 是 | 分类 ID |
| `content` | 是 | **Markdown 源码** |
| `summary` | 否 | 最长 255，可调 AI 生成 |
| `status` | 否 | `0` 草稿（默认）、`1` 发布 |

#### POST /admin/articles/:id/vectorize

无请求体。把文章投进内存队列，异步切分 + 向量化。

- 无需等待，`202` 语义：立即返回，用 `vector-status` 轮询
- 已排队/处理中会拒绝重复提交
- **未配置向量模型时返回 `503`**

#### GET /admin/articles/:id/vector-status

```json
{
  "article_id": 19,
  "status": 3,
  "status_text": "已完成",
  "chunk_count": 12,
  "error": "",
  "vectorized_at": "2026-10-29T17:20:11+08:00"
}
```

`status` 取值：

| 值 | 含义 |
|---|---|
| `0` | 未处理 / 待更新（正文改过） |
| `1` | 排队中 |
| `2` | 处理中 |
| `3` | 已完成 |
| `4` | 失败（`error` 有原因） |

#### POST /admin/articles/generate-summary

```json
{ "config_id": 2, "title": "文章标题", "content": "Markdown 正文" }
```

`config_id` 必须是**已启用的对话模型**（`type = 1`），传向量模型会返回 400。

### 5.3 分类 / 标签 / 评论

| 方法 | 路径 | 说明 |
|---|---|---|
| GET / POST | `/admin/categories` | 列表 / 新建 |
| PUT / DELETE | `/admin/categories/:id` | 更新 / 删除 |
| GET / POST | `/admin/tags` | 列表 / 新建 |
| PUT / DELETE | `/admin/tags/:id` | 更新 / 删除 |
| GET | `/admin/comments` | 评论列表 |
| DELETE | `/admin/comments/:id` | 删除单条 |
| DELETE | `/admin/comments` | 批量删除（body 传 ID 数组） |

### 5.4 AI 模型配置 `/admin/llm-configs`

模型配置存库，**不进配置文件**。新增时必须先测试连接通过才能保存。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/admin/llm-configs` | 列表；`?type=1` 只取对话模型，`?type=2` 只取向量模型 |
| GET | `/admin/llm-configs/:id` | 详情 |
| POST | `/admin/llm-configs/test` | 测试连接（不落库） |
| POST | `/admin/llm-configs` | 新建 |
| PUT | `/admin/llm-configs/:id` | 更新（仅超时/token/温度/状态，`base_url`/`api_key`/`model` 不可改） |
| DELETE | `/admin/llm-configs/:id` | 删除 |

#### POST /admin/llm-configs

```json
{
  "base_url": "https://api.deepseek.com/v1",
  "api_key": "sk-***",
  "model": "deepseek-chat",
  "type": 1,
  "timeout_sec": 60,
  "max_tokens": 2048,
  "temperature": 0.7,
  "status": 1
}
```

向量模型（`type = 2`）额外必填 `dimension`：

```json
{
  "base_url": "https://api.siliconflow.cn/v1",
  "api_key": "sk-***",
  "model": "BAAI/bge-m3",
  "type": 2,
  "dimension": 1024
}
```

| 约束 | 说明 |
|---|---|
| `dimension` | 128~4096，必须与模型实际输出一致 |
| 向量模型数量 | **全局只允许一条** |
| 维度一旦建表 | 后续不可直接改；换维度要先清空向量数据 |
| 创建即建表 | 保存成功后自动创建 `article_chunks`，否则返回 400 与具体原因 |

### 5.5 向量表 `/admin/vector`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/admin/vector/schema` | 表是否存在、当前维度、块数 |
| POST | `/admin/vector/reset` | **只删表不重建**，供换维度时手工调用（前端无入口） |

```json
{ "table_exists": true, "dimension": 1024, "chunk_count": 148 }
```

### 5.6 切分管理 `/admin/chunks`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/admin/chunks` | 块明细，支持 `article_id`、`keyword`、分页 |
| GET | `/admin/chunks/stat` | 按文章聚合的统计 |
| GET | `/admin/chunks/section` | **取整节内容（父子块的父块）** |
| GET | `/admin/chunks/:id` | 单块详情 |
| POST | `/admin/chunks/articles/delete` | 批量删除文章向量 |

#### GET /admin/chunks

```
GET /api/v1/admin/chunks?article_id=19&keyword=持久化&page=1&page_size=20
```

```json
{
  "list": [
    {
      "id": 301,
      "article_id": 19,
      "article_title": "Redis 持久化策略",
      "heading_path": "Redis 持久化策略 > 一、RDB",
      "section_index": 1,
      "chunk_index": 0,
      "char_count": 415,
      "content": "Redis 持久化策略 > 一、RDB\n\nRDB 是把内存快照写入磁盘……",
      "content_truncated": true,
      "created_at": "2026-10-29T17:20:11+08:00"
    }
  ],
  "total": 148,
  "page": 1,
  "page_size": 20
}
```

| 字段 | 说明 |
|---|---|
| `heading_path` | 标题路径，同时拼在 `content` 首行参与 embedding |
| `section_index` | 小节序号，**同一小节切出的多块共享此值** |
| `chunk_index` | 全局块序号 |

#### GET /admin/chunks/section

把同一小节的块拼回整节——「小块检索、整节喂模型」里的父块。

```
GET /api/v1/admin/chunks/section?article_id=19&section_index=2
```

```json
{
  "article_id": 19,
  "article_title": "Redis 持久化策略",
  "section_index": 2,
  "heading_path": "Redis 持久化策略 > 二、AOF",
  "chunk_count": 3,
  "content": "Redis 持久化策略 > 二、AOF\n\nAOF 是追加写日志……",
  "char_count": 2036
}
```

拼接时会去掉每块重复的路径前缀，并裁掉相邻块的重叠部分。

#### POST /admin/chunks/articles/delete

```json
{ "article_ids": [19, 20] }
```

单次最多 200 篇。逐篇独立事务，冲突或不存在记进 `skipped`，不整批回滚。

```json
{ "deleted": 2, "skipped": [] }
```

---

## 6. 附录

### 6.1 切分规则

正文按 Markdown 标题切小节，每块前置标题路径：

```
Redis 持久化策略 > 二、AOF

AOF 是追加写日志……
```

- 小节内超过 800 字才降级到「段落 → 行 → 句号 → 逗号」逐级切分
- 小于 100 字的碎块并入同小节前一块
- 相邻块有 80 字重叠（防止句子卡在边界上）
- **表格与代码块是原子单位**，永不切断，允许超长

### 6.2 健康检查

```
GET /api/v1/health
```

```json
{ "status": "ok", "message": "core-coach API is running" }
```

无认证。用于容器健康检查与部署脚本探活。
