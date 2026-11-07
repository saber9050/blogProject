# 文章页 AI 助手 API 文档

## 概述

文章详情页的 AI 答疑助手：基于文章内容（可额外引用其他文章）回答读者提问。

实现上是 **agentic RAG**：模型自己决定「要不要检索、用什么词再检索、资料够不够」，
检索工具内部是「语义 + 关键词 → RRF 融合」的混合检索。

设计文档见 `docs/up/05-article-chat/chat-plan.md`（不入库，仅本地留存）。

**所有接口都要求登录**，游客不可使用。前端在未登录时点击入口会提示先登录。

## 基础 URL

```
/api/v1
```

## 统一响应格式

```json
{ "code": 0, "message": "成功", "data": {} }
```

发消息接口在 `stream=true` 时返回 `text/event-stream`，不走这个结构。

## 认证说明

- **强制认证**：`Authorization: Bearer <access_token>`，缺失或过期返回 `401`
- 会话归属一律取自 token 里的 `user_id`，**不接受前端传 user_id**
- 访问别人的会话返回 **404**（不是 403）——403 等于承认「存在但不给你看」，可被用来枚举 id

## 错误码

沿用 `pkg/errors/code.go` 的四位码，`chat` 不新增业务码：

| code | 场景 |
|---|---|
| 400 | 内容为空 / 超过 2000 rune；引用超过 7 篇；引用的文章不存在或未发布 |
| 401 | 未登录 / token 失效 |
| 404 | 会话不存在或不属于当前用户；文章不存在或未发布 |
| 409 | 单会话消息总数达上限（400 条 ≈ 200 轮） |
| 429 | 提问过于频繁（每用户每分钟 6 条） |
| 503 | 没有可用的对话模型；模型调用失败 |

---

## 1. 面板初始化

`GET /api/v1/articles/:id/chat/context`

返回「能不能聊、有哪些模型、这篇文章下的历史会话」。

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "enabled": true,
    "reason": "",
    "article_id": 12,
    "article_title": "Redis 发布订阅和 Stream",
    "rag_ready": true,
    "max_ref_articles": 8,
    "models": [{ "id": 1, "model": "deepseek-chat" }],
    "default_model_id": 1,
    "last_session_id": 8,
    "sessions": [
      {
        "id": 8,
        "title": "Stream 和发布订阅有什么区别",
        "model_config_id": 1,
        "model": "deepseek-chat",
        "message_count": 6,
        "last_message_at": "2026-11-05T16:20:11Z",
        "created_at": "2026-11-05T15:02:00Z"
      }
    ]
  }
}
```

| 字段 | 说明 |
|---|---|
| `enabled` | 能否聊天。`false` 时前端置灰输入并展示 `reason`，**仍是 200**——后台没配模型是配置问题，不是请求错误 |
| `rag_ready` | 本文能否走向量检索（表存在 + 向量模型可用 + `vector_status=3`）。`false` 不报错，回答自动降级为关键词检索或全文 |
| `max_ref_articles` | 单次提问可涉及的文章总篇数（含当前文章）。前端据此算 pill 上限（8-1=7），不要硬编码 |
| `models` | 启用的对话模型，**只含 id + model**，不返回 base_url / api_key |
| `default_model_id` | 新建会话的默认模型：优先沿用上次会话的模型，失效则回退到 `models[0]` |
| `last_session_id` | 最近活跃的会话，`0` 表示没有 |
| `sessions` | 该文章下当前用户的会话，按 `last_message_at` 倒序，最多 50 条（无分页） |

---

## 2. 新建会话

`POST /api/v1/chat/sessions`

```json
{ "article_id": 12, "model_config_id": 1 }
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `article_id` | 是 | 文章必须存在且已发布 |
| `model_config_id` | 否 | `0` 表示用默认模型；传了就必须是「启用中的对话模型」，不做静默替换 |

返回新建的会话对象（结构同 `sessions[]`）。

---

## 3. 历史消息

`GET /api/v1/chat/sessions/:sessionId/messages?before_id=&limit=`

| 参数 | 默认 | 说明 |
|---|---|---|
| `before_id` | 0 | 取 id 小于它的消息，`0` 表示从最新开始 |
| `limit` | 30 | 上限 100 |

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 88,
        "role": "user",
        "content": "/Redis 发布订阅和 Stream 和本文有什么区别？",
        "model": "",
        "refs": null,
        "article_refs": [{ "article_id": 8, "title": "Redis 发布订阅和 Stream" }],
        "steps": null,
        "interrupted": false,
        "refused": false,
        "error": "",
        "created_at": "2026-11-05T16:20:03Z"
      }
    ],
    "has_more": false
  }
}
```

| 字段 | 说明 |
|---|---|
| `list` | **正序**（旧 → 新）。分页游标用 `before_id`，不用 offset（新消息插入会让 offset 重复/漏行） |
| `refs` | 仅 assistant 有值：本次回答用到的切分块（`chunk_id` / `article_id` / `section_index` / `heading_path` / `rrf_score`）。`null` 表示没检索或走了全文降级 |
| `article_refs` | 仅 user 有值：本轮显式引用的其他文章。`content` 里对应位置是 `/标题` 纯文本，前端按它还原 pill |
| `refused` | 被判为跑题，`content` 是后端固定文案而非模型输出 |
| `steps` | **仅 assistant**：本次回答的检索过程，见下节。`null` 表示一次都没检索 |

**`steps` 结构**（前端渲染成可展开的过程面板）：

```json
[
  {
    "round": 1,
    "tool": "search_article_context",
    "query": "Stream 消费组",
    "hits": 4,
    "chars": 1820,
    "ms": 380,
    "articles": ["Redis 发布订阅和 Stream"],
    "degraded": false,
    "note": ""
  }
]
```

| 字段 | 说明 |
|---|---|
| `query` | 模型实际用的检索词。**这是调提示词时最有价值的信息**——能看出它有没有把用户原句照抄进去 |
| `hits` | 命中的资料段数，`0` 表示没查到 |
| `ms` | 单次检索耗时。用户能据此确认「检索确实发生了，只是很快」 |
| `degraded` / `note` | 本次检索是否降级（向量不可用 / 关键词路失败 / 退回全文） |

> 为什么要落库而不是只走 SSE：刷新页面后这些步骤是「这次到底查没查、查了什么」的唯一凭证。
> 而事后追问「它为什么这么答」恰恰最需要看这个。

---

## 4. 发消息

`POST /api/v1/chat/sessions/:sessionId/messages`

```json
{
  "content": "/Redis 发布订阅和 Stream 和本文有什么区别？",
  "article_ids": [8],
  "stream": true
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `content` | 是 | 1~2000 rune。pill 已在前端序列化成 `/标题` 纯文本 |
| `article_ids` | 否 | 本轮额外引用的文章（不含当前文章），最多 7 个。**逐条校验存在 + 已发布，任一条不合法整体 400**，不做静默过滤 |
| `stream` | 否 | 默认 `true` |

### 4.1 流式（`stream: true`）

响应头：

```
Content-Type: text/event-stream; charset=utf-8
Cache-Control: no-cache
X-Accel-Buffering: no
```

事件序列：

```
event: meta
data: {"session_id":9,"user_message_id":90,"assistant_message_id":91,"model":"deepseek-chat","model_fallback":false,"article_refs":[{"article_id":8,"title":"…"}]}

event: tool
data: {"phase":"start","round":1,"name":"search_article_context","query":"Stream 消费组"}

event: tool
data: {"phase":"end","round":1,"hits":4,"chars":1820,"articles":["Redis 发布订阅和 Stream"]}

event: refs
data: {"chunks":[{"chunk_id":815,"article_id":8,"section_index":2,"heading_path":"…","rrf_score":0.0326}]}

event: delta
data: {"content":"Stream 的"}

event: done
data: {"assistant_message_id":91,"prompt_tokens":2100,"completion_tokens":286,"refused":false,"interrupted":false,"rounds":1,"tool_calls":1,"searched_articles":[12,8],"degraded":false}
```

| 事件 | 说明 |
|---|---|
| `meta` | **必发且首个**。`article_refs` 只是「校验通过的引用范围」，此时还不知道 agent 会检索几篇 |
| `tool` | 检索进度，`phase` 为 `start` / `end`，`round` 从 1 递增，最多 5。前端据此显示「正在检索资料…（第 N 轮）」——agentic 首字要 3~5 秒，没有反馈用户会以为卡死 |
| `refs` | 所有轮次跑完、`done` 之前**只发一次**（合并去重后的结果）。`rrf_score` 只表示相对名次，**不是相似度**，不要当百分比展示 |
| `delta` | 增量文本 |
| `done` | **必发且最后**。`rounds` = 实际检索轮数（0 = 一次都没检索）；`searched_articles` = 真正查过的文章 id；`degraded` = 出现过降级但最终答出来了 |
| `error` | `{"code":503,"message":"…","partial":true}`。`partial=true` 表示此前已推过内容，**前端必须保留、只在末尾加一行提示，不要清空气泡** |

> 拒答（`refused=true`）是 **200 + 正常 `done`**，不是错误码。前端用低调灰条展示，
> 不要弹 toast、不要红框——那是设计好的边界，不是故障。

**`meta` 之前失败** → 返回普通 JSON 错误 + 非 200（服务端刻意延迟写 SSE header，
就是为了这种情况还能给出正确的 404/400/503）。

### 4.2 非流式（`stream: false`）

同样字段一次性返回，用于 curl 排查：

```json
{
  "code": 0,
  "data": {
    "user_message_id": 90,
    "assistant_message_id": 91,
    "content": "Stream 的消费组…",
    "model": "deepseek-chat",
    "model_fallback": false,
    "refs": [],
    "steps": [],
    "article_refs": [],
    "searched_articles": [12, 8],
    "prompt_tokens": 2100,
    "completion_tokens": 286,
    "refused": false,
    "interrupted": false,
    "rounds": 1,
    "tool_calls": 1,
    "degraded": false
  }
}
```

---

## 5. 修改会话

`PATCH /api/v1/chat/sessions/:sessionId`

```json
{ "title": "新标题", "model_config_id": 2 }
```

两个字段都可选，但至少传一个。**不接受 `article_id`**：会话一旦创建就绑定文章，
允许改的话「不同文章的对话不互串」就只能靠前端自觉了。

换模型不影响历史消息：旧消息保留各自生成时的模型名，新消息才用新模型。

---

## 6. 删除会话

`DELETE /api/v1/chat/sessions/:sessionId`

**硬删**（消息 + 会话，同一事务），不可恢复。只能删自己的会话。

---

## 7. 候选文章

`GET /api/v1/chat/article-candidates?keyword=&exclude_id=&limit=`

供输入框的 `/` 选择器使用。只返回已发布文章，`exclude_id` 用于排除当前文章。

```json
{ "code": 0, "data": [{ "id": 8, "title": "Redis 发布订阅和 Stream" }] }
```

| 参数 | 默认 | 说明 |
|---|---|---|
| `keyword` | 空 | 为空时返回最近的 20 篇 |
| `exclude_id` | 0 | 当前文章 id |
| `limit` | 20 | 上限 50 |

---

## 8. 排查：助手从不检索资料

现象：回答很快、内容看着也通顺，但 `steps` 始终为空（前端显示「未检索资料（直接作答）」）。

**先确认接线没问题。** 有一条不需要真实模型的测试专门验证这件事：

```bash
go test ./internal/service/chat/ -run 'TestAgentActuallyCallsTool|TestSearchToolInfo' -v
```

它会用一个「必定先调工具再作答」的假模型跑完整条 ADK 链路，断言
`tool_start` / `tool_end` 事件与 `steps` 落库都正常。**通过 ⇒ 代码接线没问题，
问题在模型或网关侧。**

### 8.1 三个按概率排序的原因

| # | 原因 | 怎么确认 |
|---|---|---|
| 1 | **模型不支持 function calling** | 用下面的 curl 直接打 provider。返回里没有 `tool_calls` 字段 ⇒ 不支持 |
| 2 | **`base_url` 是第三方网关，`tools` 字段被丢弃** | 同上。小模型（如 7B 以下）和部分中转站是重灾区 |
| 3 | **Instruction 被忽略** | 换一个更强的模型对比；或看日志里有没有「未发生任何检索」的 warn |

`deepseek-reasoner`（R1 系列）、`o1` 这类推理模型**不支持 tools**，配了它们必然不检索。

### 8.2 一行 curl 判定模型能力

```bash
curl -s https://<base_url>/chat/completions \
  -H "Authorization: Bearer <api_key>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "<model>",
    "messages": [{"role":"user","content":"帮我查一下 Redis Stream 的消费组"}],
    "tools": [{
      "type": "function",
      "function": {
        "name": "search_article_context",
        "description": "在文章中检索相关段落",
        "parameters": {
          "type": "object",
          "properties": {"query": {"type":"string","description":"检索关键词"}},
          "required": ["query"]
        }
      }
    }],
    "tool_choice": "auto"
  }' | jq '.choices[0].message | {content, tool_calls}'
```

| 返回 | 结论 |
|---|---|
| `tool_calls` 非空 | 模型支持，问题在别处（看 §8.3 的日志） |
| `tool_calls` 为 null、`content` 是正常回答 | **模型忽略了 tools** ⇒ 这就是根因，换模型 |
| 400 / 报错提到 tools | **网关不支持** ⇒ 换 base_url 或换模型 |

### 8.3 日志里看什么

| 日志 | 含义 |
|---|---|
| `已构建并缓存 agent` 带 `tools=1` | 工具确实交给 agent 了 |
| `本次回答未发生任何检索（模型未调用检索工具）` | 模型没调工具，同时打出模型名与 base_url，直接对照 §8.1 |
| 完全没有上面两条 | 请求没走到 `SendMessage`（检查限流、模型配置是否启用） |

> 这条 warn 是刻意加的：不检索的表现是「回答很快、看起来正常」，
> 不打日志就永远不会有人发现——直到用户发现答案和文章对不上。

---

## 相关文档

- [admin_chat_api.md](./admin_chat_api.md)：后台「助手会话」管理
- [admin_chunk_api.md](./admin_chunk_api.md)：切分块的查看（`refs` 里的 `chunk_id` 可在这里查原文）
- [admin_llmconfig_api.md](./admin_llmconfig_api.md)：对话 / 向量模型配置
