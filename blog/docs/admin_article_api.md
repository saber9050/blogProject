# 后台文章管理 API 文档

## 概述

后台文章管理模块提供管理员对博客文章的增删改查功能。所有接口均要求管理员认证（`role_id = 1`）。

## 基础 URL

```
/api/v1/admin
```

## 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {} // 具体数据，可能为对象或数组
}
```

- `code`: 状态码，0 表示成功，非 0 表示错误
- `message`: 提示信息
- `data`: 响应数据，接口成功时返回

## 认证说明

- **强制认证**：所有接口必须携带有效的管理员 Token，否则返回 `401 Unauthorized`
- **管理员权限校验**：接口会校验用户是否为管理员（`role_id = 1`），非管理员返回 `403 Forbidden`

## 接口列表

### 1. 获取文章列表

获取分页的文章列表，支持按分类、标签、状态和关键词筛选。管理员可以查看所有状态的文章（包括草稿）。

**请求**

```
GET /articles
```

**认证**：强制认证（需携带管理员 Token）

**查询参数**

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| page | uint | 否 | 页码，默认为 1 | `1` |
| page_size | uint | 否 | 每页数量，默认为 10，最大 50 | `10` |
| status | int | 否 | 文章状态，1-已发布，0-草稿，不传则返回所有状态 | `1` |
| category)id | uint | 否 | 分类 ID，筛选指定分类的文章 | `5` |
| tag_ids | string | 否 | 标签 ID 列表，多个用英文逗号分隔，筛选包含任意标签的文章 | `1,2,3` |
| keyword | string | 否 | 搜索关键词，匹配文章标题或摘要 | `Go语言` |

**排序规则**

- 按文章创建时间倒序排列（最新创建的文章在前）

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 123,
        "title": "文章标题",
        "summary": "文章摘要",
        "cover_url": "https://example.com/cover.jpg",
        "status": 1,
        "views": 1000,
        "like_count": 50,
        "comment_count": 12,
        "author_name": "作者昵称",
        "category": {
          "id": 5,
          "name": "技术"
        },
        "tags": [
          { "id": 1, "name": "Go" },
          { "id": 2, "name": "后端" }
        ],
        "created_at": "2025-01-01T12:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 10
  }
}
```

**字段说明**

- `list`: 文章对象数组，每个对象包含：
  - `id`: 文章 ID
  - `title`: 标题
  - `summary`: 摘要
  - `cover_url`: 封面图 URL（可能为空）
  - `status`: 文章状态，1-已发布，0-草稿
  - `views`: 浏览量
  - `like_count`: 点赞数
  - `comment_count: 评论数
  - `author_name`: 作者昵称
  - `category`: 分类对象（可能为空）
  - `tags`: 标签对象数组（可能为空数组）
  - `created_at`: 创建时间（ISO 8601 格式）
- `total`: 符合条件的文章总数
- `page`: 当前页码
- `page_size`: 每页数量

### 2. 新增文章

创建一篇新文章。

**请求**

```
POST /articles
```

**认证**：强制认证（需携带管理员 Token）

**请求体**

```json
{
  "title": "文章标题",
  "type_id": 5,
  "tag_ids": [1, 2, 3],
  "cover_url": "https://example.com/cover.jpg",
  "summary": "文章摘要",
  "content": "文章完整内容（HTML）",
  "status": 1
}
```

**请求参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| title | string | 是 | 文章标题，最大长度 200 |
| type_id | uint | 是 | 分类 ID |
| tag_ids | []uint | 否 | 标签 ID 数组，不传则不关联标签 |
| cover_url | string | 否 | 封面图片 URL |
| summary | string | 否 | 文章摘要，最大长度 500 |
| content | string | 是 | 文章内容（HTML 格式） |
| status | int | 是 | 文章状态，1-已发布，0-草稿 |

**业务逻辑**

1. 验证用户是否为管理员，非管理员返回 403
2. 验证分类是否存在，不存在则返回 400
3. 验证标签是否存在，不存在的标签会被忽略
4. 创建文章，关联分类和标签
5. 自动设置作者为当前管理员用户
6. **同步图片引用**：解析 `content` HTML，提取所有 `<img>` 标签中的 `src` URL，转换为相对路径后批量插入 `article_images` 表

**响应**

成功：

```json
{
  "code": 0,
  "message": "创建成功",
  "data": {
    "id": 123
  }
}
```

失败：

```json
{
  "code": 400,
  "message": "分类不存在",
  "data": null
}
```

### 3. 编辑文章

更新指定文章的信息。

**请求**

```
PUT /articles/{id}
```

**认证**：强制认证（需携带管理员 Token）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | uint | 是 | 文章 ID |

**请求体**

```json
{
  "title": "文章标题",
  "type_id": 5,
  "tag_ids": [1, 2, 3],
  "cover_url": "https://example.com/cover.jpg",
  "summary": "文章摘要",
  "content": "文章完整内容（HTML）",
  "status": 1
}
```

**请求参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| title | string | 是 | 文章标题，最大长度 200 |
| type_id | uint | 是 | 分类 ID |
| tag_ids | []uint | 否 | 标签 ID 数组，不传则不关联标签 |
| cover_url | string | 否 | 封面图片 URL |
| summary | string | 否 | 文章摘要，最大长度 500 |
| content | string | 是 | 文章内容（HTML 格式） |
| status | int | 是 | 文章状态，1-已发布，0-草稿 |

**业务逻辑**

1. 验证用户是否为管理员，非管理员返回 403
2. 验证文章是否存在，不存在则返回 404
3. 验证分类是否存在，不存在则返回 400
4. 验证标签是否存在，不存在的标签会被忽略
5. 更新文章信息，重新关联分类和标签
6. **同步图片引用**：
   - 解析新的 `content` HTML，提取所有 `<img>` 标签中的 `src` URL，转换为相对路径
   - 查询 `article_images` 表获取该文章原有图片记录
   - **差集计算**：新提取的图片集合与原有记录对比
     - 需删除的图片（原有中有、新内容中没有）→ 从 MinIO 删除文件 + 删除 DB 记录
     - 需新增的图片（新内容中有、原有中没有）→ 插入 DB 记录
   - 图片已在原有集合中的跳过（不变）

**响应**

成功：

```json
{
  "code": 0,
  "message": "更新成功",
  "data": null
}
```

失败（文章不存在）：

```json
{
  "code": 404,
  "message": "文章不存在",
  "data": null
}
```

### 4. 删除文章

删除指定文章（软删除）。

**请求**

```
DELETE /articles/{id}
```

**认证**：强制认证（需携带管理员 Token）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | uint | 是 | 文章 ID |

**业务逻辑**

1. 验证用户是否为管理员，非管理员返回 403
2. 验证文章是否存在，不存在则返回 404
3. 查询 `article_images` 表获取该文章关联的所有图片相对路径
4. 遍历图片列表，逐一调用 MinIO 的删除接口移除文件
5. 删除 `article_images` 表中的关联记录
6. 删除封面图（`cover_url`）对应的 MinIO 文件
7. 软删除该文章关联的所有评论（将 `comments` 表中 `article_id` 匹配的记录设置 `deleted_at` 字段）
8. 执行软删除操作（设置 `deleted_at` 字段）

> **注意**：若 MinIO 中某张图片删除失败，不会中断整体流程（记录日志），确保文章删除操作不受文件清理影响。

**响应**

成功：

```json
{
  "code": 0,
  "message": "删除成功",
  "data": null
}
```

失败（文章不存在）：

```json
{
  "code": 404,
  "message": "文章不存在",
  "data": null
}
```

### 5. 一键转移分类

将指定源分类下的所有文章批量转移到目标分类。

**请求**

```
PUT /articles/transfer
```

**认证**：强制认证（需携带管理员 Token）

**请求体**

```json
{
  "from_type_id": 1,
  "to_type_id": 2
}
```

**请求参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| from_type_id | uint | 是 | 源分类 ID，需要转移的文章所属分类 |
| to_type_id | uint | 是 | 目标分类 ID，文章将转移至此分类 |

**业务逻辑**

1. 验证用户是否为管理员，非管理员返回 403
2. 验证源分类是否存在，不存在则返回 400
3. 验证目标分类是否存在，不存在则返回 400
4. 验证源分类和目标分类不能相同，相同则返回 400
5. 将 `articles` 表中 `type_id = from_type_id` 且未删除的所有记录的 `type_id` 更新为 `to_type_id`
6. 返回受影响的文章数量

**响应**

成功：

```json
{
  "code": 0,
  "message": "转移成功",
  "data": {
    "affected_count": 5
  }
}
```

失败（源分类不存在）：

```json
{
  "code": 400,
  "message": "源分类不存在",
  "data": null
}
```

失败（目标分类不存在）：

```json
{
  "code": 400,
  "message": "目标分类不存在",
  "data": null
}
```

失败（分类相同）：

```json
{
  "code": 400,
  "message": "源分类和目标分类不能相同",
  "data": null
}
```

### 6. 上传图片

上传文章内容中引用的图片文件，返回可访问的完整 URL。

**请求**

```
POST /upload
```

**认证**：强制认证（需携带管理员 Token）

**请求体**：`multipart/form-data`

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | file | 是 | 图片文件，支持 jpeg/png/gif/webp，最大 5MB |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "url": "http://minio.example.com:9000/bucket/all/20260101/abc123.jpg"
  }
}
```

**返回的 URL 说明**

- `url` 是图片的完整访问 URL，前端应将其直接嵌入文章 HTML 中（`<img src="url">`）
- 后端保存文章时会自动从此 URL 中提取相对路径存入 `article_images` 表

### 7. 一键生成摘要

根据文章内容（HTML），调用本地部署的 LLM 模型自动生成摘要。

**请求**

```
POST /articles/generate-summary
```

**认证**：强制认证（需携带管理员 Token）

**请求体**

```json
{
  "title": "文章标题",
  "content": "文章完整内容（HTML）"
}
```

**请求参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| title | string | 是 | 文章标题，传给模型辅助生成更准确的摘要 |
| content | string | 是 | 文章内容（HTML 格式），后端会自动去除 HTML 标签后传给模型 |

**业务逻辑**

1. 验证用户是否为管理员，非管理员返回 403
2. 接收 `title` 和 `content` 参数
3. 对于已去除 HTML 标签的纯文本内容，如果超过 8000 字符，截断到 8000 字符
4. 构造 prompt（包含文章标题与内容），调用本地部署的 LLM 模型 API（如 Ollama、LM Studio 等）
5. 模型返回摘要文本
6. 清理摘要，截断不超过 255 字符，返回给前端

**响应**

成功：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "summary": "这是由 AI 自动生成的文章摘要..."
  }
}
```

失败：

```json
{
  "code": 500,
  "message": "摘要生成失败，请稍后重试",
  "data": null
}
```

---

## 数据实体

### ArticleImage（文章引用图片）

| 字段名 | 类型 | 说明 |
|--------|------|------|
| id | uint | 主键 ID |
| article_id | uint | 文章 ID，关联 articles 表 |
| url | string | 图片相对路径，格式：`/bucket/目录/文件名` |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

### URL 格式转换说明

```
完整访问 URL（存于 HTML content 中）：
  http://minio.example.com:9000/bucket/all/20260101/abc.jpg

相对路径（存于 article_images.url）：
  /bucket/all/20260101/abc.jpg
```

后端通过 MinIO Client 的 `ParseFileKey` 方法将完整 URL 中的路径部分提取为相对路径。

## 图片生命周期

```
用户上传图片
    │
    ▼
POST /upload → 返回完整 URL → 前端插入编辑器 HTML
    │
    ▼
保存文章（POST/PUT /articles）
    │
    ▼
后端解析 content → 提取 img src → 转相对路径 → 写入 article_images 表
    │
    ▼
编辑文章 → 重新同步 article_images（增/删）
    │
    ▼
删除文章 → 清理 MinIO 文件 + 清除 article_images 记录
```

## 错误码参考

| 状态码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未授权（Token 无效或缺失） |
| 403 | 禁止访问（非管理员用户） |
| 404 | 文章不存在 |
| 500 | 服务器内部错误 |

## 注意事项

1. 所有接口均要求管理员认证（`role_id = 1`），非管理员用户访问返回 403。
2. 获取文章列表接口可以查看所有状态的文章（包括草稿），通过 `status` 参数筛选。
3. 新增和编辑文章时，`tag_ids` 数组中不存在的标签会被忽略，不会导致请求失败。
4. 删除文章为软删除，数据不会从数据库中物理删除。
5. 文章内容支持 HTML 格式，需确保内容安全，防止 XSS 攻击。
6. 所有时间字段均使用 ISO 8601 格式（UTC 时区）。
7. **图片引用同步**：文章新增/编辑时后端自动解析 `content` HTML 中的 `<img>` 标签，同步 `article_images` 表，前端无需额外处理。
8. **图片清理**：删除文章时，后端同时清理 MinIO 中的图片文件和 `article_images` 记录。若某张图片删除失败（如 MinIO 连接异常），仅记录日志，不阻塞文章删除流程。
9. **封面图不纳入 `article_images` 管理**：封面图（`cover_url`）在删除文章时单独删除，与内容图片路径无关。
