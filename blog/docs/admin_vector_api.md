# 后台向量表管理 API 文档

## 概述

维护 `article_chunks` 向量表：查看现状、清空重建。

向量表**由向量模型配置驱动**——表是否存在、有几维，取决于后台是否配置了已启用的向量模型。
所以这一模块只有两个接口，创建表不在这里做（在 `POST /admin/llm-configs` 里自动完成）。

所有接口均要求管理员认证（`role_id = 1`）。

## 基础 URL

```
/api/v1/admin/vector
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

## 接口列表

### 1. 查询向量表状态

**请求**

```
GET /vector/schema
```

**认证**：强制认证

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "table_exists": true,
    "dimension": 1024,
    "chunk_total": 148,
    "article_total": 18,
    "extension_ready": true,
    "model": {
      "id": 3,
      "model": "BAAI/bge-m3",
      "dimension": 1024,
      "status": 1
    },
    "match": true,
    "message": "向量表已就绪，维度与模型一致"
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `table_exists` | bool | `article_chunks` 是否已创建 |
| `dimension` | int | 表的实际列维度；表不存在时为 `0` |
| `chunk_total` | int64 | 块总数 |
| `article_total` | int64 | 有向量的文章数 |
| `extension_ready` | bool | pgvector 扩展是否可用 |
| `model` | object / null | 当前向量模型摘要，未配置时为 `null` |
| `match` | bool | 表维度与模型维度是否一致 |
| `message` | string | 给人看的状态说明 |

> 字段名是 `extension_ready` 而不是 `index_ready`：本模块**不建 HNSW 索引**，
> 叫 index 会让人误以为索引已就绪。

**这个接口的用途**

1. 切分管理页展示维度与块数
2. 「AI 模型」新增向量模型时，判断维度输入框是否应该只读
3. 排查「为什么报维度不一致」

**常见状态组合**

| table_exists | model | match | 含义 |
|---|---|---|---|
| false | null | false | 还没配向量模型，向量化不可用 |
| false | 有 | false | 配了模型但表没建（异常，需查日志） |
| true | 有 | true | 正常 |
| true | 有 | false | 换过维度，需要先 `reset` 再重建 |

### 2. 清空向量数据

**删除向量表，但不重建。**

**请求**

```
POST /vector/reset
```

**认证**：强制认证

**请求体**：无

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "deleted_chunks": 148,
    "reset_articles": 18
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `deleted_chunks` | int64 | 删除的块数（删前统计，便于核对） |
| `reset_articles` | int64 | 状态被重置为「未处理」的文章数 |

## 为什么不重建表

换维度时的正确顺序是：

```
reset（删表）→ 删除旧向量模型 → 新建向量模型（填新维度）→ 自动建表
```

如果 `reset` 按**当前模型维度**重建，就会死锁：

1. 当前模型 1024 维，表也是 1024 维
2. 用户想换 768 维模型 → 先 `reset`
3. `reset` 按 1024 重建表
4. 新建 768 维模型 → 报「表已是 1024 维，维度冲突」
5. 回到第 2 步，永远换不成

所以 `reset` 只负责**拆**，建表的职责留给「新建向量模型」那一步。

## 前端入口说明

`reset` 在**后台页面没有按钮**——换维度是低频运维操作，
做成按钮容易被误点。需要时直接调接口：

```bash
curl -X POST 'https://<域名>/api/v1/admin/vector/reset' \
  -H 'Authorization: Bearer <admin token>'
```

接口保留可调用，不要当死代码删。

## 相关文档

- 向量模型配置：[admin_llmconfig_api.md](./admin_llmconfig_api.md)
- 切分结果查询：[admin_chunk_api.md](./admin_chunk_api.md)
