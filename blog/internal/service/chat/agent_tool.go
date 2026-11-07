package chat

import (
	"context"
	"strings"
	"time"

	"blog/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"go.uber.org/zap"
)

// searchToolName 检索工具名。写死常量，因为：
//   - SSE 的 tool 事件要带上它，前端按它展示「正在检索资料…」
//   - 日志里靠它统计「一轮问了几次」
const searchToolName = "search_article_context"

// searchToolArgs 工具的入参。
//
// ⚠️ 刻意**没有 article_ids**：模型可以编造任意 id，等于给了它「读任意文章」的洞。
// 检索范围只能来自服务端校验后放进 ctx 的 scope（见 scope.go），
// 这是「范围不越界」的第一层防御。
type searchToolArgs struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"检索关键词，用关键名词短语，不要整句照抄用户原话"`
}

// contextSearcher 检索工具唯一依赖的能力。
//
// 不直接依赖 chatService：工具只需要「按范围 + 关键词检索」，
// 用最小接口的好处是它可以在单测里换成假实现，不必造一整个 service。
type contextSearcher interface {
	Search(ctx context.Context, sc *Scope, query string) (SearchResult, error)
}

// newSearchTool 构造检索工具（P3 起接真实混合检索）。
func newSearchTool(searcher contextSearcher) tool.InvokableTool {
	t, err := utils.InferTool(
		searchToolName,
		"在用户当前阅读的文章及其显式引用的文章中检索相关段落，返回带出处的小节内容。",
		func(ctx context.Context, args *searchToolArgs) (string, error) {
			return runSearch(ctx, searcher, args)
		},
	)
	if err != nil {
		// 工具定义是编译期常量，这里出错只可能是代码写错了（struct tag 非法之类）。
		// 直接 panic 比带着一个坏工具跑更早暴露问题。
		panic("构造检索工具失败: " + err.Error())
	}
	return t
}

// runSearch 检索工具的实现，返回给模型的是一段**人可读文本**而不是结构化 JSON。
//
// 为什么返回文本：tool 结果会原样进模型上下文，返回结构体还要序列化，
// 既费 token 也更容易让模型把「字段名」当成正文复述出来。
//
// ⚠️ 约束：**除 ctx 取消外，永远返回 (文本, nil)**。
// 框架对「tool 返回 error」的处理没有稳定契约（可能中断整个 agent，也可能把错误
// 回填给模型继续循环）。把正确性押在未确认的行为上不可接受，所以所有异常都转成
// 「信息 + 下一步建议」的短句，让模型按建议行动。
func runSearch(ctx context.Context, searcher contextSearcher, args *searchToolArgs) (string, error) {
	// ctx 被取消（用户点停止 / 整体超时）时必须让循环尽快停下来，
	// 继续「假装成功」只会让模型在已取消的 ctx 上继续烧 token。
	if err := ctx.Err(); err != nil {
		return "", err
	}

	sc := scopeFromCtx(ctx)
	if len(sc.ArticleIDs()) == 0 {
		// 正常情况下不可能发生（scope 在 service 里就填好了），出现即 bug 信号。
		return "检索范围无效，请直接根据已有信息回答；如果没有任何信息，请如实告知无法回答。", nil
	}
	query := strings.TrimSpace(args.Query)
	if len([]rune(query)) < 2 {
		return "query 不合法，请用 2 个字以上的关键词重新检索。", nil
	}
	return doSearch(ctx, searcher, query)
}

// doSearch 一次检索的完整执行：计次 → 记步骤 → 调检索 → 回填结果。
//
// 抽出来是因为有**两个入口**：正常的 tool 调用（runSearch），
// 和「模型给的工具名不在注册表里」的兜底入口（handleUnknownTool）。
// 两条路必须走同一份计次与记录逻辑，否则步骤面板会缺记录、次数闸也会失效。
func doSearch(ctx context.Context, searcher contextSearcher, query string) (string, error) {
	sc := scopeFromCtx(ctx)
	sc.ToolCalls++
	sc.Rounds++

	// 先占位再回填：万一检索中途进程被杀，这条记录至少留下了「它当时在查什么」
	stepIdx := sc.BeginStep(searchToolName, query)
	start := time.Now()
	res, err := searcher.Search(ctx, sc, query)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		sc.EndStep(stepIdx, 0, 0, elapsed, nil, true, "检索失败")
		return "", err
	}
	sc.EndStep(stepIdx, res.Hits, len([]rune(res.Text)), elapsed, res.Articles, res.Degraded, res.Note)
	return res.Text, nil
}

// handleUnknownTool 兜底：模型给的工具名不在注册表里（含空字符串）。
//
// 真实触发场景：Qwen2.5-7B 这类小模型在流式返回 tool_call 时会把 function.name
// 丢掉，框架拿空名字去查索引查不到，直接抛
// `[NodeRunError] tool not found in toolsNode indexes`，整轮回答中断——
// 用户看到的是红框错误，而且**永远不会有一次检索**。
//
// 我们只有一个工具，所以「调用了不存在的工具」在语义上就等于「想检索」。
// 从参数里取出 query 照常执行，比让整轮回答失败好得多。
//
// ⚠️ 这条路径**不经过 toolGuard 中间件**（框架只在命中索引后才走中间件），
// 所以次数闸与「最后一次」提示要在这里再做一遍，否则这道兜底会变成绕过限流的后门。
func (b *agentBuilder) handleUnknownTool(ctx context.Context, name, input string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	sc := scopeFromCtx(ctx)
	query := strings.TrimSpace(extractQuery(input))

	// 这行日志是兜底的最大价值：它能证明「模型到底给没给名字」。
	// 没有它，就只能猜是模型的问题还是框架的问题。
	logger.Warn("模型调用了未注册的工具名，已按检索兜底执行",
		zap.String("tool_name", name),
		zap.String("query", query),
		zap.Int("args_len", len(input)))

	if len([]rune(query)) < 2 {
		// 连检索词都取不出来：不猜要查什么，让模型基于已有信息收尾。
		return "检索工具不可用。请基于已有信息回答；如果没有信息，请如实说明无法回答。", nil
	}
	if sc.ToolCalls >= maxToolCalls {
		return "检索次数已用完，请基于现有资料作答。不要再调用检索工具。", nil
	}

	out, err := doSearch(ctx, b.searcher, query)
	if err != nil {
		// 父 ctx 取消必须原样抛出：用户已经不想等了，继续跑只是白烧 token
		if ctx.Err() != nil {
			return "", err
		}
		return "检索失败，请基于已有资料作答。", nil
	}
	if sc.ToolCalls == maxToolCalls {
		out = "（本次检索已是最后一次，请基于已有资料作答）\n" + out
	}
	return out, nil
}
