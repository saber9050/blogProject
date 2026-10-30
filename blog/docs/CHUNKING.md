# 正文清洗与切分方案

向量化链路的第一个环节：把文章正文变成「适合喂给 embedding 模型的块」。

本文覆盖两件事——**清洗**（去掉对语义无贡献的标记）和**切分**（按 Markdown 结构切成块），
以及它们为什么这么做、代码在哪里、怎么验证。

- [1. 为什么需要这一层](#1-为什么需要这一层)
- [2. 数据流](#2-数据流)
- [3. 清洗](#3-清洗)
- [4. 切分](#4-切分)
- [5. 父子块](#5-父子块)
- [6. 代码文件索引](#6-代码文件索引)
- [7. 测试与验证](#7-测试与验证)
- [8. 已知限制](#8-已知限制)

---

## 1. 为什么需要这一层

块是 RAG 里**同时承担三个角色**的基本单位：

| 角色 | 影响 |
|---|---|
| 向量化的单位 | 一个块产出一个向量，块里混两件事，向量就代表不好任何一件 |
| 检索的单位 | 系统找的不是「哪篇文章相关」，是「哪个块相关」 |
| 喂给模型的单位 | 检索到什么，模型就只能看到什么 |

所以块的质量直接决定检索上限。而块的质量由两步决定：

- **清洗**：块里不该有 `**` `|` `](http://...)` 这类对语义无贡献的字符
- **切分**：块的边界不该落在句子中间，也不该把两个不相干的小节揉在一起

> 依据：Prem AI《RAG Chunking Strategies: The 2026 Benchmark Guide》对技术文档的建议是
> 「按标题层级切 + 小节内递归，512 token/节」；Chroma 的实测显示纯语义切分只比递归切分
> 多 2–3 个 recall 点，成本却高一个量级。

---

## 2. 数据流

```text
articles.content（Markdown 源码）
        │
        │  ① 清洗：MarkdownToPlainText
        ▼
纯文本（保留段落边界，代码块内容保留）
        │
        │  ② 结构解析：parseSections
        ▼
小节序列（标题 + 正文，带祖先链）
        │
        │  ③ 逐节切分：splitSectionBody
        │     · 表格/代码块 → 原子段，不切
        │     · 超长段落 → splitRecursive 逐级降级
        ▼
小节内的块
        │
        │  ④ 合并碎块 mergeSmall  →  ⑤ 加重叠 applyOverlap
        ▼
        │  ⑥ 前置标题路径 buildHeadingPath
        ▼
[]chunker.Result{Content, CharCount, HeadingPath, SectionIndex}
        │
        │  ⑦ 批量调 embedding
        ▼
article_chunks（pgvector）
```

入口是 `chunker.Split`，位于 **`internal/service/chunker/chunker.go`**。

调用方：

| 调用点 | 位置 | 用途 |
|---|---|---|
| 主链路 | `internal/service/vectorize/vectorize_service.go:266` | 切分后调 embedding 写库 |
| 长度预校验 | `internal/service/vectorize/vectorize_service.go:98` | 提交前判断正文是否为空/过短 |
| 摘要生成 | `internal/service/llm/llm_service.go:62` | 只清洗不切分，喂给对话模型 |

---

## 3. 清洗

### 3.1 核心原则

**去掉标记，保留内容。**

`## 一、背景` 去掉 `##` 但要留「一、背景」；`[点这里](https://x.com)` 留「点这里」丢掉 URL。
唯一的例外是图片——`![示意图](url)` 整段丢掉，因为 alt 通常为空或与正文无关，URL 更是纯噪声。

### 3.2 处理规则

实现在 **`internal/service/chunker/chunker.go`** 的 `MarkdownToPlainText`（约 395 行），
正则集中在同文件 368 行的 `var` 块。

| 标记 | 处理 | 正则变量 | 理由 |
|---|---|---|---|
| ` ``` ` 代码围栏 | 删除围栏，**保留代码内容** | `reMdCodeBlock` | 代码本身是正文；标识符对向量有区分度 |
| `![alt](url)` | **整段删除** | `reMdImage` | alt 多为空，URL 无语义价值 |
| `[text](url)` | 只留 `text` | `reMdLink` | URL 对 embedding 没有贡献，还占预算 |
| `#` 标题前缀 | 删 `#` 留标题文字 | `reMdHeading` | 标题文字是很好的语义信号 |
| `>` 引用 | 删 `>` | `reMdQuote` | — |
| `---` 分隔线 | 整行删除 | `reMdHR` | 纯装饰 |
| `\| --- \|` 表格分隔行 | 整行删除 | `reMdTableSep` | 纯装饰 |
| `-` `*` `+` 列表标记 | 删标记 | `reMdBullet` | — |
| `1.` 有序列表标记 | 删标记 | `reMdOrdered` | 要求「数字 + 点 + 空白」，`1.消息` 这种正文不会被误删 |
| `~~删除线~~` | 留文字 | `reMdStrike` | — |
| `**粗体**` | 留文字 | `reMdBold` | — |
| `*斜体*` | 留文字 | `reMdItalic` | 只认 `*`，**不认 `_`**——`snake_case` 里的下划线不能动 |
| `` `行内代码` `` | 删反引号 | `reMdInlineCode` | — |

### 3.3 代码块为什么要先占位

代码块里的内容是**不能动的**：`// 这是注释` 的 `//`、`a * b` 的 `*`、shell 管道 `cmd | grep` 的 `|`，
如果直接跑上面的正则，这些符号会被当成标记删掉。

所以 `MarkdownToPlainText` 的第一步是把所有代码块**抽出来换成占位符**：

```go
s := reMdCodeBlock.ReplaceAllStringFunc(md, func(m string) string {
    codes = append(codes, strings.TrimRight(body, "\n"))
    return fmt.Sprintf("\x00%d\x00", len(codes)-1)   // 占位
})
// ... 跑完所有清洗正则 ...
for i, c := range codes {                            // 最后还原
    s = strings.ReplaceAll(s, fmt.Sprintf("\x00%d\x00", i), c)
}
```

用 `\x00` 而不是「特殊字符串」占位：它不会是正文内容，也不会被 `normalizeSpace` 当空白压缩。

### 3.4 零宽字符

富文本转过来的内容常夹着看不见的字符，它们不显示但**占字符数**，会让块首出现脏字符，
也让 `char_count` 虚高。清洗末尾统一剔除（`chunker.go:457` 的 `zeroWidth`）：

`\u200b` 零宽空格、`\u200c` 零宽不连字、`\u200d` 零宽连字、`\u2060` 词连接符、`\ufeff` BOM。

### 3.5 空白归一化

`normalizeSpace`（`chunker.go:469`）逐行处理：

- 行内连续空白（含全角空格、制表符）压成一个半角空格
- 行首尾去空白
- 统一 `\r\n` → `\n`
- 3 个以上连续换行压成 2 个（一个空行）

**保留了单换行和空行**——这是切分时判断段落边界的基础，不能压平。

> 反例：`llm_service` 早期用 `\s+ → 空格` 一行正则做清洗，整篇文章变成一行，
> 切出来的块毫无段落结构。这也是后来清洗逻辑集中到 `chunker` 包的原因。

### 3.6 效果

同一篇文章（`#30 redis发布订阅和Stream`，4350 字 Markdown）在切分时对比：

| | 块数 | 总字数 | 含标记的块 |
|---|---|---|---|
| 不洗（`StripMarkup=false`） | 6 | 4303 | **6 块全部有** |
| 洗完 | 6 | 4061 | **0** |

不洗时残留分布：`**` 出现在 5 块、`##` 出现在 6 块、`](` 1 块、`> ` 1 块。

---

## 4. 切分

### 4.1 从「定长」到「结构感知」

旧实现只看字符分隔符（段落 → 行 → 句号 → 逗号 → 空格），按 800 字往上堆。
它**不知道 Markdown 有结构**，标题 `## 一、先看困境` 对它来说和普通文本没区别。

在 14 篇真实文章（80 块）上实测的损耗：

| 问题 | 数量 |
|---|---|
| 块跨越小节边界 | **29 块（36%）** |
| 代码块被切断 | 10 处 |
| 表格被切断 | 4 处 |
| 块以标题残片开头 | 8 处 |

现在的做法：**先按标题切小节，小节内部才降级到字符切分**。
理由很直接——小节边界是作者亲手划的主题边界，比任何字符分隔符都准。

### 4.2 整体流程

实现在 `chunker.go` 的 `Split`（87 行）：

```go
func Split(content, docTitle string, p Params) ([]Result, error) {
    // 1. 整体校验（用清洗后的全文判断空/过短）
    // 2. parseSections 按标题切片
    // 3. 逐节 splitSectionBody → mergeSmall → applyOverlap
    // 4. buildHeadingPath 拼标题路径，组装 Result
}
```

### 4.3 解析标题

`parseSections`（`chunker.go:159`）逐行扫描，遇到 `^#{1,6} ` 就开始新小节。

两个容易踩坑的地方：

**① 代码块里的 `#` 不是标题**

````markdown
```bash
# 这是注释，不是标题
```
````

用 `inFence` 标记跳过围栏内所有行，否则代码块会被切成十几个「小节」。

**② 同级标题要顶掉上一个，不是叠加**

`## A` 后面紧跟 `## B`，两者都是第二级，B 要替换 A。
实现按 **level 弹栈**而不是按栈深度：

```go
for len(stack) > 0 && stack[len(stack)-1].level >= lv {
    stack = stack[:len(stack)-1]
}
stack = append(stack, headItem{level: lv, title: title})
```

用栈深度会出错：`## A` 前面没有 `#` 时栈里只占一层，深度判断不成立，
B 就叠在 A 后面变成「A > B」。

> 一级标题（`#`）**也当小节标题**。不少文章直接用它分大节（实测 `#30` 是 `# 一，发布订阅`），
> 早期实现把 `#` 当成「文章标题的重复」跳过，这些块的小节信息全丢。

### 4.4 小节内切分

`splitSectionBody`（`chunker.go:240`）按顺序处理每个段落：

```text
段落 → 清洗 → 判断是不是原子块
                    │
        ┌───────────┴───────────┐
        │ 原子块                │ 普通段落
        ▼                       ▼
  能塞进当前块就塞          长度 ≤ 800 → 塞进当前块
  塞不下就单独成块          长度 > 800 → splitRecursive 逐级降级
  （允许超长）
```

### 4.5 表格与代码块是原子单位

`isAtomicBlock`（`chunker.go:335`）识别两类：

- 代码块：以 ` ``` ` 开头
- 表格：至少两行以 `|` 开头

**永不切断，允许超长。** 半张表格喂给 embedding 是纯噪声——`| 命令 | 说明 |` 的表头在一个块里、
数据行在另一个块里，两边都读不出完整信息。

代价是可能出现远超 800 字的块（实测最长 1678 字），这是刻意接受的：
bge-m3 支持 8192 token，长度不是瓶颈，语义完整才是。

### 4.6 超长段落降级

只有**普通段落**超过 800 字时，才走 `splitRecursive`（`chunker.go:517`）。
分隔符优先级（`chunker.go:500` 的 `separators`）：

```text
\n\n（段落） → \n（行） → 。！？； → ". " "! " "? " "; " → ，, → 空格
```

逐级尝试，某级切完还超长就降一级。全部用完仍超长时用 `hardSplit`（570 行）按字数硬切——
**按 rune 切不是按 byte**，否则中文会被切成半个字符，存进库变成 U+FFFD。

### 4.7 碎块合并

`mergeSmall`（`chunker.go:593`）：小于 `MinChunkSize`（100 字）的块并进前一块，
但合并后不能超过 `ChunkSize + MinChunkSize`。

只是「小标题单独一行」「列表项很短」这类碎块，几个字的块拿去 embedding 是纯浪费——
向量里几乎没有信息，却占一次 API 调用和一行存储。

> 合并**只在小节内进行**，不跨小节。跨小节合并会把上一节的话题带进下一节的块。

### 4.8 相邻块重叠

`applyOverlap`（`chunker.go:631`）：给每块前置上一块尾部 80 字，防止完整句子卡在边界上两边都读不到。

拼接时补一个 `\n\n` 分隔符。原来直接字符串拼接，实测出现过：

```text
1.消息队列的相关命令| 命令 | 基本语法格式 | 功能描述 |
```

上一块的结尾和本块开头的 Markdown 表格前半行黏成一行，喂给 embedding 就是噪声。

### 4.9 标题路径

`buildHeadingPath`（`chunker.go:214`）拼出 `文章标题 > 上一级 > 当前小节`，**拼进块正文参与 embedding**。

```text
redis发布订阅和Stream > 二，Stream > 2.消费组相关命令

消费组是依赖Stream消息队列来存在的，不能独自创建。
```

这是整个方案里性价比最高的一步：中文技术博客的查询词往往就是小节标题，
用户搜「消费组」，块里明晃晃写着「消费组相关命令」，语义匹配直接命中。

三个约束：

| 约束 | 原因 |
|---|---|
| 最多两级小节 | 路径会挤占 800 字的预算，太长反而稀释正文 |
| 取「当前 + 上一级」而非「最顶层两级」 | 三级标题下的小节若只取顶层，当前标题就丢了 |
| 与 `docTitle` 重复的首段跳过 | 正文首行常写成 `# 标题`（与 `articles.title` 相同），不跳会出现「我的文章 > 我的文章 > 节一」 |

路径同时单独存到 `article_chunks.heading_path`，供 UI 展示与将来按章节过滤。

### 4.10 参数

参数固化为 `DefaultParams()`（`chunker.go:70`），没有「切分模板」概念：

| 常量 | 值 | 位置 | 说明 |
|---|---|---|---|
| `DefaultChunkSize` | 800 | `chunker.go:20` | 目标块大小（字） |
| `DefaultChunkOverlap` | 80 | `chunker.go:21` | 相邻块重叠 |
| `DefaultMinChunkSize` | 100 | `chunker.go:22` | 碎块阈值 |
| `MinContentChars` | 10 | `chunker.go:30` | 低于此值直接拒绝向量化 |

`Params.withDefaults`（435 行）会兜住非法值（0 或负数），静默修正而不是报错。

### 4.11 实测效果

改前改后对比（14 篇真实文章）：

| 指标 | 定长切分 | 结构感知 |
|---|---|---|
| 跨小节块 | 29 | **0** |
| 代码块断裂 | 10 | **0** |
| 表格断裂 | 4 | **0** |
| 块数 | 80 | 150（平均 400 字/块） |
| 单块最长 | — | 1678 字（原子表格，刻意允许） |

---

## 5. 父子块

### 5.1 要解决的问题

结构感知切分让块变细（平均 400 字），检索更准；但命中一个 300 字的块后，
模型可能拿不到完整答案——「找到了，但回答缺一块」是最常见的 RAG 失败模式。

父子块的解法：**索引存小块（检索准），命中后把所属整节交给模型（上下文全）**。

### 5.2 实现

不需要新表：同一小节切出的所有块共享 `section_index`（`article_chunks.section_index`）。

```sql
SELECT * FROM article_chunks
WHERE article_id = ? AND section_index = ?
ORDER BY chunk_index
```

拼接逻辑在 **`internal/service/chunk/chunk_service.go`** 的 `Section` 方法，
仓储层是 **`internal/repository/articlechunk/articlechunk_repository.go`** 的 `ListBySection`。

拼接要做两件事，缺一件结果就是错的：

1. **去掉每块的标题路径前缀** —— 同一小节的块前缀完全相同，不去会重复 N 次
2. **裁掉相邻块的重叠** —— 每块前置了上一块尾部 80 字

重叠裁剪用「从大到小试匹配」：拿上一块末尾 N 字对本块开头 N 字，找到最大匹配就裁掉。
少于 8 字的重合不裁——那多半是「的」「我们」这类巧合，裁了反而吃正文。

---

## 6. 代码文件索引

### 6.1 核心实现

| 文件 | 内容 |
|---|---|
| `internal/service/chunker/chunker.go` | 清洗与切分的全部实现，纯函数、无 IO |
| `internal/service/chunker/chunker_test.go` | 23 组单测 |

`chunker.go` 内的函数分工：

| 函数 | 行号 | 职责 |
|---|---|---|
| `Split` | 87 | 入口：解析 → 切分 → 合并 → 重叠 → 拼路径 |
| `parseSections` | 159 | 按标题切小节，维护层级栈 |
| `buildHeadingPath` | 214 | 拼 `文章标题 > 上一级 > 当前小节` |
| `splitSectionBody` | 240 | 小节内切分，处理原子块 |
| `splitParagraphs` | 309 | 按空行切段，代码块内部不切 |
| `isAtomicBlock` | 335 | 判断表格/代码块 |
| `ExtractText` | 357 | 对外入口：按需清洗 |
| `MarkdownToPlainText` | 395 | Markdown 清洗，代码块先占位 |
| `normalizeSpace` | 469 | 空白归一化，保留段落边界 |
| `splitRecursive` | 517 | 按分隔符优先级降级切分 |
| `hardSplit` | 570 | 按 rune 硬切，兜底 |
| `mergeSmall` | 593 | 碎块合并 |
| `applyOverlap` | 631 | 相邻块加重叠 |

### 6.2 上下游

| 环节 | 文件 | 说明 |
|---|---|---|
| 触发 | `internal/api/v1/admin/article/controller.go` | `POST /articles/:id/vectorize` |
| 任务链路 | `internal/service/vectorize/vectorize_service.go` | 队列、worker、事务写块 |
| 长度预校验 | 同上 `:98` | 提交前用 `ExtractText` 判断 |
| 表结构 | `pkg/database/vector.go` | `article_chunks` DDL、维度校验、补列 |
| 整节拼接 | `internal/service/chunk/chunk_service.go` | 父子块的父块 |
| 摘要复用 | `internal/service/llm/llm_service.go:62` | 只清洗不切分 |
| 图片处理 | `internal/service/article/article_service.go` | 图片引用同步（与清洗无关，但同属正文解析） |

### 6.3 数据落点

| 字段 | 表 | 说明 |
|---|---|---|
| `content` | `article_chunks` | 块正文（含标题路径前缀） |
| `char_count` | `article_chunks` | rune 数 |
| `section_index` | `article_chunks` | 小节序号，父子块靠它 |
| `heading_path` | `article_chunks` | 标题路径，UI 展示与按章节过滤 |
| `embedding` | `article_chunks` | `vector(N)`，维度来自向量模型配置 |
| `vector_status` | `articles` | 0 未处理 / 1 排队 / 2 处理中 / 3 完成 / 4 失败 |

---

## 7. 测试与验证

### 7.1 单元测试

`internal/service/chunker/chunker_test.go`，23 组用例，覆盖：

| 类别 | 用例 |
|---|---|
| 异常输入 | 空内容、纯空白、纯零宽字符、只有分隔线、只有图片、正文过短 |
| 结构切分 | 按小节切、路径含文章标题、一级标题当小节、同级标题不叠加、文档标题去重 |
| 原子块 | 表格不被切断（60 行完整保留）、代码块不被切断（50 行完整保留） |
| 清洗 | 各类标记、代码块内容保留、代码里的 `*` 不被误删、段落边界不丢 |
| 边界 | 无标题退化、超长无标点硬切、碎块合并、重叠、块长上限 |

跑法：

```bash
cd blog && go test ./internal/service/chunker/ -v
```

### 7.2 真实数据验收

写一个临时程序对全库跑一遍，统计结构破坏：

```go
// 用 StripMarkup=false 切，块里保留标记才能看出结构是否被切断
chunks, _ := chunker.Split(content, title,
    chunker.Params{ChunkSize: 800, ChunkOverlap: 0, MinChunkSize: 100, StripMarkup: false})

// 断言三个指标都应为 0
//  · 块内出现 ≥2 个 `^##` 标题    → 跨小节
//  · 块内 ``` 数量为奇数           → 代码块被切断
//  · 有 ≥2 行以 | 开头但没有 |---|  → 表格被切断
```

### 7.3 验证整节拼接

后台「切分管理」页点「所属小节」列，或直接调接口：

```bash
curl 'https://<域名>/api/v1/admin/chunks/section?article_id=19&section_index=2' \
  -H 'Authorization: Bearer <admin token>'
```

返回的整节内容应该**通读无重复段落**。有重复说明重叠没裁干净，
有断裂说明路径前缀没去干净。

---

## 8. 已知限制

| 限制 | 影响 | 现状 |
|---|---|---|
| 引用式图片 `![x][ref]` | 清洗时不会被删除，会以原样留在块里 | 博客图片都是行内式，暂不处理 |
| URL 含空格的图片 `![](<a b.png>)` | 清洗正则匹配不到，会残留 | MinIO 生成的 key 不含空格 |
| HTML 内嵌在 Markdown 里 | 清洗不剥 HTML 标签，会原样留在块里 | 编辑器是 Markdown，正常写不会出现 |
| 只识别 ATX 标题（`##`） | Setext 标题（`===` 下划线式）不认 | 编辑器只产出 ATX |
| 表格必须是 `|` 开头 | 无边框表格不认，按普通段落切 | 同上 |
| 原子块可能超长 | 单块最长实测 1678 字 | 刻意接受的代价 |

> 关于「清洗不剥 HTML」：正文统一是 Markdown 后，HTML 清洗那套（剥标签、实体反转义、td 分隔）
> 已经删除。如果将来要支持「Markdown 里嵌 HTML」，需要在 `MarkdownToPlainText` 之前
> 补一道 HTML 剥离，并注意 `ADD COLUMN` 式的迁移顺序问题。
