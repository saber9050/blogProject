# 评论模块 API 文档

## 基础信息

- **Base URL**: `/api/v1/articles`
- **认证方式**: 读取接口无需登录，写入接口需在 Header 中携带 JWT Token：`Authorization: Bearer <token>`
- **统一响应格式**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，0 表示成功 |
| message | string | 响应消息 |
| data | object/array | 响应数据（成功时返回） |

---

## 接口列表

### 1. 一级评论列表

> GET /api/v1/articles/:id/comments

**认证**: 无需登录

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| id | uint | 文章ID |

**请求参数** (Query):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "list": [
      {
        "id": 1,
        "content": "string",
        "user_id": 1,
        "user_name": "string",
        "avatar_url": "string",
        "parent_id": null,
        "reply_to_name": "",
        "is_deleted": false,
        "created_at": "2024-01-01T00:00:00Z",
        "children_total": 3
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| list | array | 评论列表 |
| list[].id | uint | 评论ID |
| list[].content | string | 评论内容 |
| list[].user_id | uint | 评论者用户ID |
| list[].user_name | string | 评论者昵称 |
| list[].avatar_url | string | 评论者头像URL |
| list[].parent_id | uint/null | 父评论ID（一级评论为 null） |
| list[].reply_to_name | string | 回复目标用户名（一级评论为空） |
| list[].is_deleted | bool | 是否已删除 |
| list[].created_at | datetime | 创建时间 |
| list[].children_total | int | 子评论（二级回复）总数 |
| total | int64 | 总记录数 |
| page | int | 当前页码 |
| page_size | int | 每页条数 |

---

### 2. 二级回复列表

> GET /api/v1/articles/:id/comments/:commentId/replies

**认证**: 无需登录

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| id | uint | 文章ID |
| commentId | uint | 父评论ID |

**请求参数** (Query):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 10 |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "list": [
      {
        "id": 2,
        "content": "string",
        "user_id": 2,
        "user_name": "string",
        "avatar_url": "string",
        "parent_id": 1,
        "reply_to_name": "string",
        "is_deleted": false,
        "created_at": "2024-01-01T00:00:00Z",
        "children_total": 0
      }
    ],
    "total": 5,
    "page": 1,
    "page_size": 10
  }
}
```

> 字段说明同 [一级评论列表](#1-一级评论列表)，`parent_id` 指向父评论ID，`reply_to_name` 为回复目标用户名。

---

### 3. 发表评论

> POST /api/v1/articles/:id/comments

**认证**: 需要登录

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| id | uint | 文章ID |

**请求参数** (JSON Body):

```json
{
  "content": "string",
  "parent_id": null,
  "reply_to_user_name": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 是 | 评论内容 |
| parent_id | uint/null | 否 | 父评论ID；null 表示一级评论，传值表示二级回复 |
| reply_to_user_name | string | 否 | 回复目标用户名（二级评论时用于展示 @xxx） |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": 1,
    "content": "string",
    "user_id": 1,
    "user_name": "string",
    "avatar_url": "string",
    "parent_id": null,
    "reply_to_name": "",
    "is_deleted": false,
    "created_at": "2024-01-01T00:00:00Z",
    "children_total": 0
  }
}
```

> 返回新创建的评论对象，字段同评论列表项。

---

### 4. 删除评论

> DELETE /api/v1/articles/:id/comments/:commentId

**认证**: 需要登录（仅评论作者或管理员可删除）

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| id | uint | 文章ID |
| commentId | uint | 评论ID |

**请求参数**: 无（用户身份从 Token 解析）

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": "删除成功"
}
```