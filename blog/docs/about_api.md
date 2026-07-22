# 关于页面 API 文档

## 概述

关于页面模块提供博客关于页面信息的展示与管理功能。关于页面数据为单例模式，全局仅有一条配置记录。

## 基础 URL

- 前台接口：`/api/v1`

## 统一响应格式

```json
{
  "code": 0,
  "message": "成功",
  "data": {} // 具体数据，可能为对象或数组
}
```

- `code`: 状态码，0 表示成功，非 0 表示错误
- `message`: 提示信息
- `data`: 响应数据，接口成功时返回

## 认证说明

- **无需认证**：前台获取接口无需认证，任何人都可以访问
- **强制认证**：修改接口必须携带有效的管理员 Token，否则返回 `401 Unauthorized`
- **管理员权限校验**：修改接口会校验用户是否为管理员（`role_id = 1`），非管理员返回 `403 Forbidden`

## 接口列表

### 1. 获取关于页面信息

获取关于页面的全部展示信息，包括管理员基本信息与关于实体内容。此接口无需认证。

**请求**

```
GET /about
```

**认证**：无需认证

**查询参数**：无

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "admin_name": "管理员昵称",
    "avatar_url": "https://example.com/avatar.jpg",
    "email": "admin@example.com",
    "introduction": "热爱构建高性能、高可维护性的系统",
    "tech_stack": "Go,Vue,AI,Docker,Redis,MySQL",
    "my_story": "大学开始接触编程，最初学习 Java。后来接触 Go...",
    "why": "学习最大的敌人不是不会，而是遗忘...",
    "interest": "Go,Redis,消息队列,系统设计,AI Agent,Docker",
    "git_hub": "https://github.com/yourusername",
    "csdn": "https://blog.csdn.net/yourusername"
  }
}
```

**字段说明**

| 字段名 | 类型 | 说明 |
|--------|------|------|
| admin_name | string | 管理员昵称（来自用户表） |
| avatar_url | string | 管理员头像 URL（来自用户表） |
| email | string | 管理员邮箱（来自用户表） |
| introduction | string | 管理员个人简介（来自用户表） |
| tech_stack | string | 技术栈，多个用逗号隔开 |
| my_story | string | 我的故事（Markdown 格式） |
| why | string | 为什么建立博客 |
| interest | string | 感兴趣的技术，多个用逗号隔开 |
| git_hub | string | GitHub 地址 |
| csdn | string | CSDN 地址 |

**业务逻辑**

1. 查询 abouts 表中的唯一记录
2. 查询用户表中管理员的 user_name、avatar_url、email、introduction 字段
3. 若 abouts 表中没有数据，各字段返回空字符串

### 2. 修改关于页面内容

更新关于页面的配置内容。此为部分更新接口，只发送需要修改的字段，未发送的字段保持不变。

**请求**

```
PUT /about
```

**认证**：强制认证（需携带管理员 Token，`role_id = 1`）

**请求体**

```json
{
  "tech_stack": "Go,Vue,AI,Docker,Redis,MySQL,Kubernetes",
  "my_story": "更新后的我的故事内容..."
}
```

**请求参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| tech_stack | string | 否 | 技术栈，每个用逗号隔开，最大长度 255 |
| my_story | string | 否 | 我的故事（Markdown 格式） |
| why | string | 否 | 为什么建立博客 |
| interest | string | 否 | 感兴趣的技术，每个用逗号隔开，最大长度 255 |
| git_hub | string | 否 | GitHub 地址，最大长度 255 |
| csdn | string | 否 | CSDN 地址，最大长度 255 |

**注意事项**

- 此为部分更新接口，请求体中**只包含需要修改的字段**即可
- 未在请求体中出现的字段将**保持原值不变**
- 请求体为空时，不执行任何更新操作，返回成功

**业务逻辑**

1. 验证用户是否为管理员（`role_id = 1`），非管理员返回 403
2. 查询 abouts 表中是否存在记录：
   - 若不存在，使用默认值创建一条记录，再执行更新
   - 若存在，直接更新指定字段
3. 只更新请求体中包含的字段，未包含的字段保持不变

**响应**

成功：

```json
{
  "code": 0,
  "message": "成功",
  "data": null
}
```

失败（非管理员）：

```json
{
  "code": 403,
  "message": "禁止访问",
  "data": null
}
```

## 错误码参考

| 状态码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未授权（Token 无效或缺失） |
| 403 | 禁止访问（非管理员用户） |
| 500 | 服务器内部错误 |

## 注意事项

1. 关于页面数据为单例模式，abouts 表中仅存在一条记录。
2. 获取接口无需认证，任何人都可以访问。
3. 修改接口仅管理员可操作，需要携带有效的管理员 Token。
4. 修改接口为**部分更新**，只需发送需要修改的字段，未发送的字段保持不变。
5. 若 abouts 表中尚无记录，修改接口会自动创建一条默认记录后再执行更新。
6. 所有时间字段均使用 ISO 8601 格式（UTC 时区）。
