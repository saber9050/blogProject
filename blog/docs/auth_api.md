# 认证模块 API 文档

## 基础信息

- **Base URL**: `/api/v1/auth`
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

### Token 机制

采用 **双 Token 机制**：

| Token | 存储位置 | 有效期 | 说明 |
|-------|----------|--------|------|
| Access Token | 客户端（`Authorization: Bearer <token>` Header） | 15 分钟 | JWT，内含 `user_id`、`username`、`user_role_id`、`tid` |
| Refresh Token | JWT `tid` 字段 + Redis | 7 天 | 随机字符串，同时也是会话标识符 |

**关键设计**：
- **Refresh Token = 会话ID**。登录时生成一个随机串，同时放入 JWT claims（`tid` 字段）和 Redis（`refresh:{user_id}`），不再有独立的 sessionID
- Redis 直接存 refresh token 字符串，不再需要结构体 `session_id + token`
- Refresh token 不直接返回前端，而是编码在 JWT 的 `tid` 字段中，刷新时从过期 access token 解析提取

**验证流程**：
1. 客户端使用 `access_token` 正常请求接口
2. 中间件仅验证 JWT 签名与过期时间，不额外校验 refresh token
3. `access_token` 过期后，客户端调用刷新接口，带上旧 access token，服务端从中解析 `tid` 进行轮换

**单设备踢出**：
- 新登录生成新 refresh token 覆盖 Redis
- 旧设备的 refresh token 失效，无法刷新 access token
- 旧设备的 access token 在有效期内仍可使用，过期后无法续期
- 旧设备必须重新登录

**Refresh Token 轮换**：
- 每次刷新生成新的 refresh token，更新 JWT + Redis（旧 refresh token 立即失效）
- 若旧 refresh token 被重放（不匹配 Redis）→ 拒绝刷新，需重新登录（不影响当前设备）

---

## 接口列表

### 1. 获取图形验证码

> GET /api/v1/auth/image_captcha

**请求参数**: 无

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "captcha_id": "xxx",
    "base_64": "base64编码的图片"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| captcha_id | string | 验证码ID，登录时需回传 |
| base_64 | string | 验证码图片 Base64 编码 |

---

### 2. 注册

> POST /api/v1/auth/register

**请求参数** (JSON Body):

```json
{
  "user_name": "string",
  "account": "string",
  "password": "string",
  "ack": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_name | string | 是 | 昵称，1-15字符，不可重复 |
| account | string | 是 | 账号，11位数字 |
| password | string | 是 | 密码，11-20位数字和字母组合 |
| ack | string | 是 | 密码二次确认，需与 password 一致 |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "message": "注册成功"
  }
}
```

---

### 3. 账号密码登录

> POST /api/v1/auth/login

**请求参数** (JSON Body):

```json
{
  "account": "string",
  "password": "string",
  "captcha_key": "string",
  "captcha_code": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account | string | 是 | 账号 |
| password | string | 是 | 密码 |
| captcha_key | string | 是 | 图形验证码ID（由 image_captcha 接口获取） |
| captcha_code | string | 是 | 图形验证码，6位字符 |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "user_name": "string",
    "user_id": 1,
    "user_role_id": 1,
    "access_token": "jwt_token_string"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| user_name | string | 用户名/昵称 |
| user_id | uint | 用户ID |
| user_role_id | uint | 用户角色ID |
| access_token | string | Access Token (JWT)，有效期 15 分钟，后续请求需在 `Authorization: Bearer <token>` 中携带 |

> **注意**：如果该账号已有活跃会话（已在其他设备/浏览器登录），旧会话将被限制：
> - Redis 中的旧 refresh token 被覆盖，旧设备无法刷新 access token
> - 旧设备的 access token 在有效期内仍可使用，过期后无法续期
> - 被踢出的旧设备必须重新登录

---

### 4. 邮箱登录

> POST /api/v1/auth/email_login

**请求参数** (JSON Body):

```json
{
  "email": "string",
  "purpose": "login",
  "captcha": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | 是 | 邮箱地址 |
| purpose | string | 是 | 用途，登录时固定为 `login` |
| captcha | string | 是 | 邮箱验证码，6位 |

**成功响应**: 同 [账号密码登录](#3-账号密码登录)（JSON）

> **注意**：踢出行为同账号密码登录 — 如果该账号已有活跃会话，旧设备 access token 过期后失效。

---

### 5. 发送邮箱验证码

> POST /api/v1/auth/captcha

**请求参数** (JSON Body):

```json
{
  "email": "string",
  "purpose": "login"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | 是 | 接收验证码的邮箱地址 |
| purpose | string | 是 | 用途，可选值：`login`、`reset_password`、`reset_email`、`add_email` |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "message": "已向你的邮箱发送验证码"
  }
}
```

### 6. 重设密码

> POST /api/v1/auth/reset_password

**请求参数** (JSON Body):

```json
{
  "email": "string",
  "captcha": "string",
  "new_password": "string",
  "ack": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | 是 | 邮箱地址 |
| captcha | string | 是 | 邮箱验证码，6位 |
| new_password | string | 是 | 新密码 |
| ack | string | 是 | 密码二次确认，需与 new_password 一致 |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "message": "成功修改密码"
  }
}
```

---

### 7. 登出

> POST /api/v1/auth/logout

**请求头**:

| 参数 | 说明 |
|------|------|
| Authorization | Bearer JWT Token |

**请求参数**: 无 Body，Token 通过 Header 传递

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": null
}
```

> **说明**：登出后仅当该 token 的会话仍是当前活跃会话时才删除 Redis 中的 refresh token。若该账号已在其他设备重新登录（旧会话已失效），登出操作不会影响当前活跃会话。

---

### 8. 刷新 Token

> POST /api/v1/auth/refresh

**请求头**:

| 参数 | 说明 |
|------|------|
| Authorization | Bearer 过期的 Access Token |

**请求参数**: 无 Body

Refresh token 从 Access Token 的 JWT claims 中 `tid` 字段提取，无需额外传递。

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "access_token": "new_jwt_token_string"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| access_token | string | 新的 Access Token (JWT)，有效期 15 分钟 |

> **说明**：每次刷新都会轮换 refresh token（旧 refresh token 立即失效）。若检测到旧 refresh token 被重放（疑似被盗），服务端拒绝刷新，前端清除 token 跳转登录页（不影响当前持有新 token 的设备）。
>
> **错误响应**：
> - `401` — refresh token 已过期、已被使用、或被踢出
> - 前端检测到 401 应清除本地存储的 access_token 并跳转登录页

---

### 9. 检查昵称是否存在

> GET /api/v1/auth/is_exists_name?user_name=xxx

**请求参数** (Query):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_name | string | 是 | 要检查的昵称 |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "is_exists": true
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| is_exists | bool | true 表示已存在，false 表示不存在（可用） |

---

### 10. 检查账号是否存在

> GET /api/v1/auth/is_exists_account?account=xxx

**请求参数** (Query):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account | string | 是 | 要检查的账号 |

**成功响应**:

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "is_exists": true
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| is_exists | bool | true 表示已存在，false 表示不存在（可用） |
