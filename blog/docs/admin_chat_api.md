# 后台助手会话管理 API 文档

## 概述

查看全站 AI 助手的对话记录：列表、详情（含每条消息的 `refs` / `refused` / `error`）、删除。

与前台的 [chat_api.md](./chat_api.md) 是两个视角：

| | 前台 | 后台 |
|---|---|---|
| 范围 | 只有自己创建的会话 | 全站 |
| 归属校验 | 有（不符返回 404） | 无 |
| 文章状态 | 只认已发布 | **不过滤**（草稿、已删文章的会话也要能看到） |
| user 消息渲染 | 还原成 pill | 直接显示 `/标题` 原文 |

所有接口均要求管理员认证（`role_id = 1`）。

## 基础 URL

```
/api/v1/admin/chat-sessions
```

## 统一响应格式

```json
{ "code": 0, "message": "成功", "data": {} }
```

## 认证说明

- **强制认证**：必须携带有效的管理员 Token
- **权限校验**：`role_id = 1`，非管理员由 `middleware.Auth(1)` 拦下

---

## 1. 会话列表

`GET /api/v1/admin/chat-sessions?user=&article=&page=&page_size=`

| 参数 | 默认 | 说明 |
|---|---|---|
| `user` | 空 | **一个输入框两种匹配**：纯数字先按 `user_id` 精确匹配（同时仍参与用户名模糊匹配）；非数字只按用户名 `ILIKE` |
| `article` | 空 | 文章标题 `ILIKE` |
| `page` | 1 | — |
| `page_size` | 20 | 上限 100 |

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 8,
        "article_id": 12,
        "article_title": "Redis 发布订阅和 Stream",
        "article_deleted": false,
        "user_id": 3,
        "user_name": "zxp",
        "title": "Stream 和发布订阅有什么区别",
        "summary": "Stream 和发布订阅有什么区别？",
        "message_count": 6,
        "last_message_at": "2026-11-05T16:20:11Z",
        "created_at": "2026-11-05T15:02:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20
  }
}
```

| 字段 | 说明 |
|---|---|
| `summary` | 首条 user 消息原文的前 40 字。比 `title` 更可信——标题可能被用户改过 |
| `article_deleted` | 文章已被硬删（join 不到）。列表显示「文章已删除」而不是空标题，否则管理员会以为是自己筛选条件写错了 |
| `user_name` | 用户被删除时为空串 |

**排序**：`last_message_at DESC NULLS LAST, id DESC`。带 id 兜底是因为刚创建的会话 `last_message_at` 为 NULL，只按它排序结果不稳定，翻页会重复或漏行。

---

## 2. 会话详情

`GET /api/v1/admin/chat-sessions/:id`

```json
{
  "code": 0,
  "data": {
    "session": { "id": 8, "article_title": "…", "user_name": "zxp", "message_count": 6 },
    "messages": [
      {
        "id": 88,
        "role": "user",
        "content": "/Redis 发布订阅和 Stream 和本文有什么区别？",
        "model": "",
        "refs": null,
        "article_refs": [{ "article_id": 8, "title": "Redis 发布订阅和 Stream" }],
        "prompt_tokens": 0,
        "completion_tokens": 0,
        "interrupted": false,
        "refused": false,
        "error": "",
        "created_at": "2026-11-05T16:20:03Z"
      }
    ]
  }
}
```

**消息不分页**：单会话有 400 条上限（约 200 轮），一次返回可控。真出现几百条消息，
那本身就是需要人工介入的异常。

三个徽章字段：`refused`（拒答，内容为后端固定文案）、`error`（失败原因）、
`interrupted`（用户中途停止）。后台要靠它们一眼定位「这次为什么没答好」。

---

## 3. 删除会话

`DELETE /api/v1/admin/chat-sessions/:id`

**硬删**（先删消息、再删会话，同一事务），**不可恢复**。前端必须二次确认。

删除后：后台列表不再显示；用户端访问该会话立刻 404（前端会自动清掉本地记录并回退到空态，不弹错）。

---

## 相关文档

- [chat_api.md](./chat_api.md)：前台 AI 助手接口
- [admin_chunk_api.md](./admin_chunk_api.md)：按 `chunk_id` 查切分块原文
- [admin_llmconfig_api.md](./admin_llmconfig_api.md)：模型配置（对话模型停用后，会话会回退到首个启用模型）
