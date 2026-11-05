package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
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

// newSearchTool 构造检索工具。
//
// P2 阶段返回**假数据**：先把「ADK 能不能跑通、事件能不能正确映射成 SSE、
// 无关问题会不会真的不调工具」这三件事验完，再接真检索（P3）。
// 混在一起做的话，结果不对时你分不清是 agent 配置问题还是 SQL 问题。
func newSearchTool() tool.InvokableTool {
	t, err := utils.InferTool(
		searchToolName,
		"在用户当前阅读的文章及其显式引用的文章中检索相关段落，返回带出处的小节内容。",
		runSearch,
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
func runSearch(ctx context.Context, args *searchToolArgs) (string, error) {
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

	sc.ToolCalls++
	sc.Rounds++

	// ===== P2：假数据（P3 换成真实的混合检索）=====
	// 保留 query 在文案里，方便本地一眼看出「模型到底拿什么词去查」——
	// 调 Instruction 时这个信息比什么都重要。
	return fmt.Sprintf(`[1] 《%s》· 示例小节
（P2 阶段为假数据，真实检索将在后续接入）模型本次使用的检索词是「%s」。
这段内容用于验证 agent 与 SSE 事件链路是否打通。`, sc.CurrentArticleTitle, query), nil
}
