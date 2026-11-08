package chat

import (
	"context"
	stderrors "errors"
	"io"
	"strconv"
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
	// noRetrieval 一次检索都没成功（内容已被替换成 NoRetrievalReply）
	noRetrieval bool
	err         error
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
	// 零检索守卫：见到第一段正文时若 ToolCalls 还是 0，就判定它在凭标题编造
	filter := &deltaFilter{hasRetrieval: func() bool { return sc.ToolCalls > 0 }}
	defer func() {
		// 必须把缓冲区吐干净：短回答（不足 40 字）全在缓冲里，
		// 不 flush 的话前端一个字都收不到——而「回答很短」是最常见的情况之一。
		for _, out := range filter.flush() {
			buf.WriteString(out)
			_ = sink.OnDelta(out)
		}
		res.content = buf.String()
		res.refused = filter.Refused()
		res.noRetrieval = filter.NoRetrieval()
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
			//
			// hits / articles 一律取 tool 自己记的那条 step，不要在这里重新数：
			// 用 countSegments(content) 数的是「资料文本里有几段」，而 tool 记的是
			// 「实际命中几个块」，两者在退回全文降级时会对不上。
			last := sc.LastStep()
			hits, chars, articles := last.Hits, last.Chars, last.Articles
			if hits == 0 && chars == 0 {
				// 兜底：tool 因为某些原因没记上步骤时，至少把内容长度报出去
				chars = len([]rune(content))
				hits = countSegments(content)
			}
			_ = sink.OnToolEnd(maxInt(sc.Rounds, 1), hits, chars, articles)
			continue
		}

		if mo.IsStreaming && mo.MessageStream != nil {
			acc := newToolCallAcc()
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
				acc.add(chunk.ToolCalls)
				collectUsage(chunk, &res)
			}
			acc.emit(sink, sc)
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
			acc := newToolCallAcc()
			acc.add(msg.ToolCalls)
			acc.emit(sink, sc)
			collectUsage(msg, &res)
		}
	}
}

// toolCallAcc 合并流式 tool_call 的**分片**后再发事件。
//
// 为什么必须合并：OpenAI 协议里 delta.tool_calls 是增量——第一片带 id 和
// function.name，后续片只带 arguments 片段（name/id 都为空），
// `schema.ToolCall.Index` 的注释说得很明确：流式下它就是「第几个调用的哪一片」。
// eino-ext 把每一片原样转成一个 schema.ToolCall（chat_model.go 的 toMessageToolCalls）。
//
// 不合并的后果（线上实测）：一次检索被拆成 3 片 → 发 3 次 tool_start →
// 前端列出 2~3 个同一条检索，编号乱跳，而且 query 永远是空串
// （单片解析不出完整 JSON）。用户看到的是「检索 4 次却列了 6 条」。
type toolCallAcc struct {
	order []string
	calls map[string]*schema.ToolCall
}

func newToolCallAcc() *toolCallAcc {
	return &toolCallAcc{calls: make(map[string]*schema.ToolCall, 2)}
}

func (a *toolCallAcc) add(list []schema.ToolCall) {
	for _, tc := range list {
		key := toolCallKey(tc)
		if key == "" {
			// 纯参数片段：归到最近一个已知调用上。
			// 归不到（第一片就是裸参数）时按 0 处理——只有一个工具的常态适用。
			if len(a.order) > 0 {
				key = a.order[len(a.order)-1]
			} else {
				key = "0"
			}
		}

		cur, ok := a.calls[key]
		if !ok {
			cp := tc
			a.calls[key] = &cp
			a.order = append(a.order, key)
			continue
		}
		// 分片合并：非空字段才覆盖，参数一律拼接
		if tc.ID != "" {
			cur.ID = tc.ID
		}
		if tc.Type != "" {
			cur.Type = tc.Type
		}
		if tc.Function.Name != "" {
			cur.Function.Name = tc.Function.Name
		}
		cur.Function.Arguments += tc.Function.Arguments
	}
}

// emit 合并完成后，每个调用只发一次 tool_start。
func (a *toolCallAcc) emit(sink StreamSink, sc *Scope) {
	for i, key := range a.order {
		c := a.calls[key]
		name := c.Function.Name
		if name == "" {
			// 名字丢了（Qwen2.5-7B 这类小模型常见）。前端只需要知道"在检索"，
			// 真正执行时由 UnknownToolsHandler 按参数兜底，不影响功能。
			name = searchToolName
		}
		// round = Rounds+1+i：并行调用时同一轮会返回多个 tool_call，
		// 加 i 才能保证每个 start 的 round 唯一（前端按它配对 end）。
		_ = sink.OnToolStart(sc.Rounds+1+i, name, extractQuery(c.Function.Arguments))
	}
}

// toolCallKey 决定一个分片归属哪个调用。
//
// 优先用 Index（流式合并的正规标识）；没有 Index 的提供方退化为 ID / 名字；
// 三者都没有就是纯参数片段，由调用方归并到最近一个。
func toolCallKey(tc schema.ToolCall) string {
	if tc.Index != nil {
		return "i" + strconv.Itoa(*tc.Index)
	}
	if tc.ID != "" {
		return "id:" + tc.ID
	}
	if tc.Function.Name != "" {
		return "name:" + tc.Function.Name
	}
	return ""
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

// LastStep 最后一条检索记录。没有检索时返回零值，调用方不必判空。
func (s *Scope) LastStep() entity.ChatStep {
	if len(s.Steps) == 0 {
		return entity.ChatStep{}
	}
	return s.Steps[len(s.Steps)-1]
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
