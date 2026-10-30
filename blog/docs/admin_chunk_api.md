# 后台切分管理 API 文档

## 概述

查询文章向量化的切分结果：块明细、按文章聚合的统计、以及把同一小节的块拼回整节。

切分由向量化任务在后台完成（见 [admin_article_api.md](./admin_article_api.md) 的向量化接口），
本模块只负责**读**和**删除**。

所有接口均要求管理员认证（`role_id = 1`）。

## 基础 URL

```
/api/v1/admin/chunks
```

## 统一响应格式

```json
{
  "code": 0,
  "message": "成功",
  "data": {}
}
```

## 认证说明

- **强制认证**：必须携带有效的管理员 Token，否则返回 `401 Unauthorized`
- **管理员权限校验**：`role_id = 1`，非管理员返回 `403 Forbidden`

## 切分规则速查

理解返回字段之前，先知道块是怎么切出来的：

```text
Redis 持久化策略 > 二、AOF

AOF 是追加写日志……
```

- 按 Markdown 标题切小节，每块正文前拼**标题路径**（参与 embedding）
- 小节内超过 800 字才降级到「段落 → 行 → 句号 → 逗号」逐级切
- 小于 100 字的碎块并入同小节前一块
- 相邻块有 80 字重叠
- **表格与代码块是原子单位**，永不切断，允许超长

## 接口列表

### 1. 查询块明细

**请求**

```
GET /chunks
```

**认证**：强制认证

**查询参数**

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|---|---|---|---|---|
| page | int | 否 | 页码，默认 1 | `1` |
| page_size | int | 否 | 每页数量，默认 20 | `20` |
| article_id | uint | 否 | 只看某篇文章的块 | `19` |
| keyword | string | 否 | 关键词，匹配块内容 | `持久化` |

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "total": 148,
    "page": 1,
    "page_size": 20,
    "list": [
      {
        "id": 301,
        "article_id": 19,
        "article_title": "Redis 持久化策略",
        "heading_path": "Redis 持久化策略 > 二、AOF",
        "section_index": 2,
        "chunk_index": 4,
        "char_count": 415,
        "content": "Redis 持久化策略 > 二、AOF\n\nAOF 是追加写日志……",
        "content_truncated": true,
        "embedding_model": "BAAI/bge-m3",
        "created_at": "2026-10-29 17:20:11"
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `heading_path` | string | 标题路径，`文章标题 > 上级小节 > 当前小节`（最多两级小节） |
| `section_index` | int | **小节序号**，同一小节切出的多块共享此值 |
| `chunk_index` | int | 全局块序号，`0` 起 |
| `char_count` | int | 字符数（含标题路径前缀） |
| `content` | string | 块正文，首行是标题路径 |
| `content_truncated` | bool | `true` 表示列表里只返回了预览，完整内容需调详情 |
| `embedding_model` | string | 产出该向量的模型 |

### 2. 按文章聚合统计

**请求**

```
GET /chunks/stat
```

**认证**：强制认证

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "summary": {
      "chunk_total": 148,
      "article_total": 14
    },
    "list": [
      {
        "article_id": 19,
        "article_title": "Redis 持久化策略",
        "chunk_count": 12,
        "avg_chars": 386,
        "last_chunk_at": "2026-10-29 17:20:11"
      }
    ]
  }
}
```

> 只有「按文章」一个维度：切分参数已全局固定，按模板分组的结果只剩一行「默认参数」，没有信息量。

### 3. 取整节内容（父子块）

把同一小节的块拼回整节原文。

**请求**

```
GET /chunks/section
```

**认证**：强制认证

**查询参数**

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|---|---|---|---|---|
| article_id | uint | 是 | 文章 ID | `19` |
| section_index | int | 否 | 小节序号，默认 0 | `2` |

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "article_id": 19,
    "article_title": "Redis 持久化策略",
    "section_index": 2,
    "heading_path": "Redis 持久化策略 > 二、AOF",
    "chunk_count": 3,
    "content": "Redis 持久化策略 > 二、AOF\n\nAOF 是追加写日志……",
    "char_count": 2036
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `chunk_count` | int | 该小节由几块拼成 |
| `content` | string | 拼接后的整节，首行是标题路径 |
| `char_count` | int | 整节字符数 |

**拼接做两件事**（缺一件结果就是错的）：

1. 去掉每块重复的标题路径前缀——同一小节的块前缀完全相同
2. 裁掉相邻块的重叠部分——每块前置了上一块尾部 80 字

重叠裁剪用「从大到小试匹配」：拿上一块末尾 N 字对本块开头 N 字，找到最大匹配就裁掉。
少于 8 字的重合不裁，那多半是「的」「我们」这类巧合。

**为什么保留这个接口**

「小块检索、整节喂模型」——检索用小块的向量命中最准，但交给模型时需要完整上下文。
现在没有检索出口，所以它同时是**验证切分质量**的手段：块拼不回通顺原文，父子块方案就是错的。

### 4. 查询单块详情

列表里 `content_truncated = true` 时，用这个接口拿完整内容。

**请求**

```
GET /chunks/:id
```

**认证**：强制认证

**响应**：单个块对象，结构同列表项，`content` 为完整内容、`content_truncated` 恒为 `false`。

### 5. 批量删除文章向量

一次性删掉多篇文章的所有块，并把文章状态重置为「未处理」。

**请求**

```
POST /chunks/articles/delete
```

**认证**：强制认证

**请求体**

```json
{ "article_ids": [19, 20, 21] }
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `article_ids` | uint[] | 是 | 文章 ID 列表，**单次最多 200 篇** |

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "deleted_articles": 2,
    "deleted_chunks": 25,
    "skipped": [
      {
        "article_id": 21,
        "title": "Redis 持久化策略",
        "reason": "正在向量化中，请稍后再试"
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `deleted_articles` | int | 成功删除的文章数 |
| `deleted_chunks` | int64 | 删除的块总数 |
| `skipped` | array | 被跳过的文章及原因 |

**为什么是「跳过」而不是「整批失败」**

用户一次勾十篇，其中一篇可能正在向量化。整批回滚会让他一篇都删不掉，还得自己找出是哪一篇。
所以逐篇独立事务：失败的记进 `skipped`，其余照常删。

**为什么不做单块删除**

块是「整篇替换」的产物——重新向量化时会整批删旧插新，单独删一个块会让这篇的块集合不完整。
需要清理就整篇删。

## 路由说明

`/stat` 与 `/section` 是**静态路径段**，`/:id` 是**参数段**。两者同级共存依赖 gin ≥ v1.7
（当前用 v1.12），静态段优先匹配，所以 `/chunks/section` 不会被 `/chunks/:id` 截走。

## 相关文档

- 向量模型配置：[admin_llmconfig_api.md](./admin_llmconfig_api.md)
- 向量表状态与重置：[admin_vector_api.md](./admin_vector_api.md)
- 文章向量化提交：[admin_article_api.md](./admin_article_api.md)
