package chat

import (
	"context"
	stderrors "errors"
	"io"
	"strings"

	"blog/internal/model/entity"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// MetaEvent 开流第一帧。收到它前端才建助手气泡。
type MetaEvent struct {
	SessionID          uint
	UserMessageID      uint
	AssistantMessageID uint
	Model              string
	// ModelFallback 会话原绑定的模型已不可用，本次自动换了模型。
	ModelFallback bool
	// ArticleRefs 校验通过的引用文章（不存在/未发布的已剔除）。
	ArticleRefs []entity.ChatArticleRef
}

// DoneEvent 流结束。这些字段只有到结束才准：
// 开流时 agent 还没决定要不要检索，rounds/tool_calls 都无从得知。
type DoneEvent struct {
	AssistantMessageID uint
	PromptTokens       int
	CompletionTokens   int
	Refused            bool
	Interrupted        bool
	Rounds             int
	ToolCalls          int
	SearchedArticles   []uint
	Degraded           bool
}

// StreamSink 流式事件接收方。
//
// 抽成接口而不是直接写 HTTP：这样「非流式」只需要换一个把内容攒起来的实现，
// agent 事件消费逻辑（下面 consumeAgentStream）两种模式完全共用一份。
// 否则流式和非流式会变成两套容易跑偏的代码。
type StreamSink interface {
	OnMeta(MetaEvent) error
	// OnToolStart 模型决定调用检索工具的那一刻。
	OnToolStart(round int, name, query string) error
	// OnToolEnd 检索返回。
	OnToolEnd(round, hits, chars int, articles []string) error
	OnDelta(text string) error
	// OnError 流中途出错。partial 表示此前已经推过内容，前端不能清空气泡。
	OnError(code int, message string, partial bool) error
}

// consumeResult agent 事件消费的产物。
type consumeResult struct {
	content          string
	promptTokens     int
	completionTokens int
	interrupted      bool
	// refused 本次回答被判为跑题（内容已被替换成 OffTopicReply）
	refused bool
	err     error
}

// consumeAgentStream 把 ADK 的事件流翻译成 SSE 事件。
//
// 事件的两类来源（都靠 MessageOutput 区分，不靠消息内容猜）：
//   - Role == schema.Tool：工具执行完的结果 → tool end
//   - Role == schema.Assistant：模型输出 → delta；若这一轮带 ToolCalls 则是 tool start
//
// 顺序天然正确：模型先说「我要调工具」（assistant + ToolCalls）→ 框架执行 →
// 工具结果事件（Tool）。所以 start 一定在 end 之前，不需要额外排序。
//
// 用命名返回值 + defer 收尾，是为了保证**任何**返回路径都带上已生成的内容：
// 中途失败时那几百字对用户有价值，丢了等于既白烧 token 又白等时间。
func consumeAgentStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent],
	sink StreamSink, sc *Scope) (res consumeResult) {

	var buf strings.Builder
	filter := &deltaFilter{}
	defer func() {
		// 必须把缓冲区吐干净：短回答（不足 40 字）全在缓冲里，
		// 不 flush 的话前端一个字都收不到——而「回答很短」是最常见的情况之一。
		for _, out := range filter.flush() {
			buf.WriteString(out)
			_ = sink.OnDelta(out)
		}
		res.content = buf.String()
		res.refused = filter.Refused()
		// ctx 被取消 ⇒ 用户点了停止或整体超时。这不是错误，已生成内容照常保留。
		if ctx.Err() != nil && res.err == nil {
			res.interrupted = true
		}
	}()

	for {
		event, ok := iter.Next()
		if !ok {
			return res
		}
		if event.Err != nil {
			res.err = event.Err
			return res
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mo := event.Output.MessageOutput

		// 工具执行结果
		if mo.Role == schema.Tool {
			content := ""
			if msg, err := mo.GetMessage(); err == nil && msg != nil {
				content = msg.Content
			}
			// round 取「已发生的检索次数」：tool 执行时已自增过 sc.Rounds，
			// 与 start 时预估的序号刚好对上。
			_ = sink.OnToolEnd(maxInt(sc.Rounds, 1), countSegments(content), len([]rune(content)), sc.ArticleTitlesInScope())
			continue
		}

		if mo.IsStreaming && mo.MessageStream != nil {
			var toolCalls []schema.ToolCall
			for {
				chunk, err := mo.MessageStream.Recv()
				if stderrors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					// 模型调用失败但框架会自动重试：这不是致命错误。
					// 当成致命错误返回的话，一次 429 就会把整条回答掐断。
					// 这里只跳出内层继续读下一个事件，不发 error 事件——
					// 重试成功后前端完全感知不到，体验更好。
					var wr *adk.WillRetryError
					if stderrors.As(err, &wr) {
						break
					}
					res.err = err
					return res
				}
				if chunk == nil {
					continue
				}
				if chunk.Content != "" {
					for _, out := range filter.feed(chunk.Content) {
						buf.WriteString(out)
						_ = sink.OnDelta(out)
					}
				}
				if len(chunk.ToolCalls) > 0 {
					toolCalls = append(toolCalls, chunk.ToolCalls...)
				}
				collectUsage(chunk, &res)
			}
			emitToolStart(sink, sc, toolCalls)
			continue
		}

		// 非流式整块消息
		if mo.Message != nil {
			msg := mo.Message
			if msg.Content != "" {
				for _, out := range filter.feed(msg.Content) {
					buf.WriteString(out)
					_ = sink.OnDelta(out)
				}
			}
			emitToolStart(sink, sc, msg.ToolCalls)
			collectUsage(msg, &res)
		}
	}
}

// emitToolStart 这一轮如果带 ToolCalls，说明模型要调检索。
//
// round 用「下一次检索的序号」：此刻工具还没执行，sc.Rounds 还是上一次的值。
func emitToolStart(sink StreamSink, sc *Scope, calls []schema.ToolCall) {
	if len(calls) == 0 {
		return
	}
	for _, c := range calls {
		name := c.Function.Name
		if name == "" {
			name = searchToolName
		}
		_ = sink.OnToolStart(sc.Rounds+1, name, extractQuery(c.Function.Arguments))
	}
}

// extractQuery 从 tool 参数 JSON 里取 query，只用于前端展示「正在检索 xxx」。
//
// 解析失败返回空串而不是报错：这只是展示信息，不该因为模型偶尔吐出畸形 JSON
// 就让整条回答失败（真正的参数校验在 tool 内部做）。
func extractQuery(arguments string) string {
	const key = `"query"`
	idx := strings.Index(arguments, key)
	if idx < 0 {
		return ""
	}
	rest := arguments[idx+len(key):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if len(rest) < 2 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// collectUsage 累加 token 用量。流式下通常只有最后一个 chunk 带 ResponseMeta，
// 累加而不是覆盖是为了兼容「每块都带」的实现。
func collectUsage(msg *schema.Message, res *consumeResult) {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return
	}
	u := msg.ResponseMeta.Usage
	res.promptTokens += u.PromptTokens
	res.completionTokens += u.CompletionTokens
}

// countSegments 数工具结果里有几段（以 `[数字]` 开头的行）。
// 只用于前端展示「已检索 N 段」，数不准也不影响功能。
func countSegments(content string) int {
	n := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 3 || line[0] != '[' {
			continue
		}
		end := strings.IndexByte(line, ']')
		if end <= 1 {
			continue
		}
		// `[` 与 `]` 之间必须全是数字：正文里 [] 的其他用法（如「[注意]」）不该被算成一段
		ok := true
		for _, c := range line[1:end] {
			if c < '0' || c > '9' {
				ok = false
				break
			}
		}
		if ok {
			n++
		}
	}
	return n
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
