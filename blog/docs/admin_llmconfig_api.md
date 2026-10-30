# 后台 AI 模型配置 API 文档

## 概述

提供大模型配置的增删改查与连接测试。配置**存库**，不写进 `config.yaml`。

系统里有两类模型，用 `type` 区分：

| type | 类型 | 用途 |
|---|---|---|
| `1` | 对话模型 | 文章摘要生成等 |
| `2` | 向量模型 | 正文向量化（embedding），**全局只允许一条** |

所有接口均要求管理员认证（`role_id = 1`）。

## 基础 URL

```
/api/v1/admin/llm-configs
```

## 统一响应格式

```json
{
  "code": 0,
  "message": "成功",
  "data": {}
}
```

- `code`: 状态码，0 表示成功，非 0 表示错误
- `message`: 提示信息
- `data`: 响应数据

## 认证说明

- **强制认证**：所有接口必须携带有效的管理员 Token，否则返回 `401 Unauthorized`
- **管理员权限校验**：接口会校验用户是否为管理员（`role_id = 1`），非管理员返回 `403 Forbidden`

## 通用字段

**配置对象**（`api_key` 一律脱敏返回，绝不返回明文）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | uint | 配置 ID |
| `base_url` | string | OpenAI 兼容接口地址，如 `https://api.deepseek.com/v1` |
| `api_key_masked` | string | 脱敏后的密钥，如 `sk-****abcd` |
| `model` | string | 模型标识，同时用作配置名称 |
| `type` | int8 | `1` 对话模型 / `2` 向量模型 |
| `dimension` | int | 向量维度，仅 `type=2` 有意义，否则为 0 |
| `timeout_sec` | int | 请求超时（秒） |
| `max_tokens` | int | 最大输出 token |
| `temperature` | float | 温度 |
| `status` | int8 | `1` 启用 / `0` 禁用 |
| `created_at` / `updated_at` | string | 时间戳 |

> 配置名称不单独维护，创建时自动取 `model`。

## 接口列表

### 1. 获取配置列表

**请求**

```
GET /llm-configs
```

**认证**：强制认证

**查询参数**

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|---|---|---|---|---|
| type | int | 否 | 只返回指定类型；不传返回全部 | `2` |

> `type` 只能是 `1` 或 `2`，其他值返回 `400`。
> 文章编辑页的摘要模型下拉必须传 `type=1`，否则向量模型会混进选项。

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "id": 2,
      "base_url": "https://api.deepseek.com/v1",
      "api_key_masked": "sk-****a1b2",
      "model": "deepseek-chat",
      "type": 1,
      "dimension": 0,
      "timeout_sec": 60,
      "max_tokens": 2048,
      "temperature": 0.7,
      "status": 1,
      "created_at": "2026-10-29 17:12:03",
      "updated_at": "2026-10-29 17:12:03"
    }
  ]
}
```

### 2. 获取配置详情

**请求**

```
GET /llm-configs/:id
```

**认证**：强制认证

**响应**：单个配置对象，结构同上。

### 3. 测试连接

不落库，仅验证地址、密钥、模型是否可用。

**请求**

```
POST /llm-configs/test
```

**认证**：强制认证

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `base_url` | string | 是 | 接口地址 |
| `api_key` | string | 否 | 为空且 `id > 0` 时复用该配置已存的密钥 |
| `model` | string | 是 | 模型标识 |
| `type` | int8 | 否 | `1` 或 `2`，缺省 `1` |
| `timeout_sec` | int | 否 | 测试超时，服务端上限 15 秒 |
| `dimension` | int | 否 | 仅向量模型：期望维度，用于与实际输出比对 |

```json
{
  "base_url": "https://api.siliconflow.cn/v1",
  "api_key": "sk-***",
  "model": "BAAI/bge-m3",
  "type": 2,
  "dimension": 1024
}
```

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "ok": true,
    "latency_ms": 312,
    "reply": "OK",
    "dimension": 1024
  }
}
```

| 字段 | 说明 |
|---|---|
| `ok` | 是否连通 |
| `latency_ms` | 往返耗时 |
| `reply` | 对话模型的回复片段，向量模型无此字段 |
| `dimension` | **向量模型实际输出的维度**，前端拿它和用户填写的值比对 |
| `error` | 失败原因 |

### 4. 新建配置

**请求**

```
POST /llm-configs
```

**认证**：强制认证

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `base_url` | string | 是 | 接口地址 |
| `api_key` | string | 是 | 密钥 |
| `model` | string | 是 | 模型标识 |
| `type` | int8 | 否 | `1` 对话（缺省）/ `2` 向量 |
| `dimension` | int | 条件 | **`type=2` 时必填**，128~4096 |
| `timeout_sec` | int | 否 | 缺省 60 |
| `max_tokens` | int | 否 | 缺省 2048 |
| `temperature` | float | 否 | 缺省 0.7 |
| `status` | int8 | 否 | `0` 或 `1`，缺省启用 |

**对话模型示例**

```json
{
  "base_url": "https://api.deepseek.com/v1",
  "api_key": "sk-***",
  "model": "deepseek-chat",
  "type": 1
}
```

**向量模型示例**

```json
{
  "base_url": "https://api.siliconflow.cn/v1",
  "api_key": "sk-***",
  "model": "BAAI/bge-m3",
  "type": 2,
  "dimension": 1024
}
```

**业务规则**

1. **保存前必须测试连接通过**，否则拒绝入库
2. 向量模型 `dimension` 必须在 `128~4096`
3. 模型实际输出的维度必须与填写的 `dimension` 一致
4. **向量模型全局只允许一条**，重复创建返回 `400` 并提示已存在的配置
5. 向量模型保存成功后会**自动创建 `article_chunks` 表**（维度取自 `dimension`）

**响应**：创建成功的配置对象。

**错误响应**

| 场景 | code | message |
|---|---|---|
| 参数缺失或格式错 | 400 | `请求参数错误` |
| 连接失败 | 400 | `配置不可用，测试连接失败：<具体原因>` |
| 维度超范围 | 400 | `向量维度需在 128~4096 之间，当前填写 0` |
| 维度不一致 | 400 | `模型实际输出 1024 维，与你填写的 512 不一致；...` |
| 向量模型已存在 | 400 | `已存在向量模型（BAAI/bge-m3, id=3），全局只允许一条，请先删除或改为对话模型` |
| 建表失败 | 400 | `补充向量表字段失败: ...` 等具体原因 |

### 5. 更新配置

只能改运行参数与状态。`base_url` / `api_key` / `model` / `type` 不可改——
换地址等于换服务、换类型等于换用途，都应当删了重建。

**请求**

```
PUT /llm-configs/:id
```

**认证**：强制认证

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `timeout_sec` | int | 否 | 超时 |
| `max_tokens` | int | 否 | 最大 token |
| `temperature` | float | 否 | 温度 |
| `status` | int8 | 否 | `0` / `1`，不传表示不修改 |

**响应**：更新后的配置对象。

> 更新不重新测试连接。

### 6. 删除配置

**请求**

```
DELETE /llm-configs/:id
```

**认证**：强制认证

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": null
}
```

> 删除向量模型**不会**自动删除 `article_chunks` 表与已有向量数据。
> 需要清空时调 [向量表接口](./admin_vector_api.md) 的 `reset`。
