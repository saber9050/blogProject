# 用户模块 API 文档

## 基础信息

- **Base URL**: `/api/v1/user`
- **认证方式**: 除 `/email` 接口外，其余接口均需在 Header 中携带 JWT Token：`Authorization: Bearer <token>`
- **统一响应格式**:

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，200 表示成功 |
| message | string | 响应消息 |
| data | object/array | 响应数据（成功时返回） |

---

## 接口列表

### 1. 获取用户信息

> GET /api/v1/user/info

**认证**: 需要登录

**请求参数**: 无（用户ID从 Token 中解析）

**成功响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "user_id": 1,
    "user_name": "string",
    "account": "string",
    "email": "string",
    "avatar_url": "string",
    "introduction": "string",
    "role_id": 1,
    "staus": 1,
    "create_at": "2024-01-01T00:00:00Z"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | uint | 用户ID |
| user_name | string | 昵称 |
| account | string | 账号 |
| email | string | 邮箱 |
| avatar_url | string | 头像URL |
| introduction | string | 个人简介 |
| role_id | int8 | 角色ID |
| staus | int8 | 用户状态 |
| create_at | datetime | 创建时间 |

---

### 2. 编辑用户信息

> POST /api/v1/user/profile

**认证**: 需要登录

**请求参数** (JSON Body):

```json
{
  "nick_name": "string",
  "introduction": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| nick_name | string | 否 | 新昵称 |
| introduction | string | 否 | 个人简介 |

**成功响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": "更新成功"
}
```

---

### 3. 更换头像

> POST /api/v1/user/avatar

**认证**: 需要登录

**请求参数** (multipart/form-data):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 头像图片文件 |

**成功响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "avatar_url": "string"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| avatar_url | string | 新头像的URL |

---

### 4. 修改邮箱请求

> POST /api/v1/user/email_ack

**认证**: 需要登录

**请求参数** (JSON Body):

```json
{
  "password": "string",
  "captcha": "string",
  "new_email": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| password | string | 是 | 当前密码，用于验证身份 |
| captcha | string | 是 | 邮箱验证码，6位（发送到新邮箱） |
| new_email | string | 是 | 新邮箱地址 |

**成功响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "string"
  }
}
```

> 调用成功后，系统会向新邮箱发送确认链接。

---

### 5. 确认修改邮箱

> GET /api/v1/user/email?token=xxx

**认证**: 无需登录（通过 Token 验证）

**请求参数** (Query):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| token | string | 是 | 邮箱确认 Token（由邮件中的链接携带） |

**成功响应**: 返回 HTML 结果页面，显示操作结果。

---

### 6. 添加邮箱

> POST /api/v1/user/add_email

**认证**: 需要登录

**请求参数** (JSON Body):

```json
{
  "new_email": "string",
  "captcha": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| new_email | string | 是 | 要添加的邮箱地址 |
| captcha | string | 是 | 邮箱验证码，6位 |

**成功响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "添加成功"
  }
}
```

---

### 7. 检查邮箱是否存在

> GET /api/v1/user/is_exists_email?new_email=xxx

**认证**: 需要登录

**请求参数** (Query):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| new_email | string | 是 | 要检查的邮箱地址 |

**成功响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "is_exists": true
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| is_exists | bool | true 表示已存在，false 表示不存在（可用） |