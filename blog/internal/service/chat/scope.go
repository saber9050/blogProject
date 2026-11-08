package chat

import (
	"context"

	"blog/internal/model/entity"
)

// scopeKey ctx 里存放请求级检索范围的键。
//
// 用非导出空结构体做键，避免与任何三方包（eino、gin）往 ctx 里塞的键冲突。
type scopeKey struct{}

// Scope 一次提问的请求级上下文。
//
// ⚠️ **绝对不能把这些字段拼进 agent 的 Instruction**：agent 与 runner 是按模型缓存、
// 跨请求复用的（见 agent.go），Instruction 一旦带上文章标题/ID，两个用户并发就会互相
// 串号——而且只在高并发下复现，本地单人测试永远测不出来。
// 请求级数据只有两个合法去处：① 首条 user 消息的前缀（给模型看）② 这里的 ctx（给 tool 用）。
type Scope struct {
	SessionID uint
	// AssistantMessageID 本次回答对应的助手消息 id（开流前就建好了占位消息）。
	//
	// tool 落库要用它做归属：一条助手消息可能对应多轮检索，
	// 事后按 message_id 就能把「这一轮回答到底查了什么」整组取出来。
	AssistantMessageID uint
	// CurrentArticleID 会话所属文章，永远是检索主体。
	CurrentArticleID    uint
	CurrentArticleTitle string
	// RefArticleIDs 用户在本轮用 `/` 额外引用的文章（不含当前文章）。
	RefArticleIDs []uint
	// RefTitles 引用文章 id → 标题，用于在资料里标注出处。
	RefTitles map[uint]string

	// Seen 已经返回过的 chunk，第二轮检索必须排除，否则模型看到重复内容会误判「资料就这些」。
	Seen map[uint]struct{}
	// Refs 全部轮次命中的块，流结束后落库。
	Refs []entity.ChatRefChunk
	// Steps 每次工具调用的过程记录，流结束后落库。
	// 这是「这次到底检索了没、查了什么」的唯一凭证，刷新页面后还能看。
	Steps []entity.ChatStep

	// ToolCalls 实际执行的检索次数（含并行）。只有一个 tool 时通常等于 Rounds。
	ToolCalls int
	// Rounds 真正发生过检索的轮数。
	Rounds int
	// UsedChars 累计已装配的资料字数，受总预算 6000 约束。
	UsedChars int
	// Degraded 本次回答出现过降级（检索失败 / 走了全文兜底）。
	Degraded bool
}

// withScope 把 scope 放进 ctx。
//
// 放**指针**是刻意的：tool 执行时要往 Seen / Refs / ToolCalls 里写回结果，
// 值拷贝会让这些写入全部丢失（而且不会报错，只是 refs 永远为空）。只读的
// 「文章范围」在 tool 侧通过 ArticleIDs() 返回副本，避免被误改。
func withScope(ctx context.Context, s *Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, s)
}

// scopeFromCtx 取 scope。取不到时返回空 scope（而不是 nil），
// 让 tool 里的判空逻辑退化成「没有可检索的范围」，走正常的空结果分支。
func scopeFromCtx(ctx context.Context) *Scope {
	if v, ok := ctx.Value(scopeKey{}).(*Scope); ok && v != nil {
		return v
	}
	return &Scope{}
}

// ArticleIDs 本次检索允许涉及的文章集合（当前文章 + 引用，已去重）。
//
// 返回副本：调用方拿到的是值，改它不会影响 scope。这是「范围只读」这条约束的
// 代码级保障——只靠注释约定，迟早有人在 tool 里 append 一下就把范围改了。
func (s *Scope) ArticleIDs() []uint {
	ids := make([]uint, 0, len(s.RefArticleIDs)+1)
	if s.CurrentArticleID != 0 {
		ids = append(ids, s.CurrentArticleID)
	}
	for _, id := range s.RefArticleIDs {
		if id != s.CurrentArticleID {
			ids = append(ids, id)
		}
	}
	return ids
}

// ArticleTitlesInScope 检索范围内涉及的文章标题（当前文章在前，引用在后）。
// 用于 SSE 的 tool end 事件里展示「这次查了哪几篇」。
func (s *Scope) ArticleTitlesInScope() []string {
	out := make([]string, 0, len(s.RefArticleIDs)+1)
	if s.CurrentArticleTitle != "" {
		out = append(out, s.CurrentArticleTitle)
	}
	for _, id := range s.RefArticleIDs {
		if title, ok := s.RefTitles[id]; ok {
			out = append(out, title)
		}
	}
	return out
}

// BeginStep 追加一条「开始检索」的记录，返回它的下标供结束时回填。
//
// 先占位再回填（而不是等检索完再 append）：检索过程中如果进程被杀 / 用户中断，
// 这条记录至少留下了「它当时在查什么」，比什么都没有强。
func (s *Scope) BeginStep(tool, query string) int {
	s.Steps = append(s.Steps, entity.ChatStep{
		Round: len(s.Steps) + 1,
		Tool:  tool,
		Query: query,
	})
	return len(s.Steps) - 1
}

// EndStep 回填一次检索的结果。
func (s *Scope) EndStep(idx int, hits, chars int, ms int64, articles []string, degraded bool, note string) {
	if idx < 0 || idx >= len(s.Steps) {
		return
	}
	st := &s.Steps[idx]
	st.Hits = hits
	st.Chars = chars
	st.Ms = ms
	st.Articles = articles
	st.Degraded = degraded
	st.Note = note
}

// EnsureSeen 初始化 Seen，避免 tool 里反复判空。
func (s *Scope) EnsureSeen() {
	if s.Seen == nil {
		s.Seen = make(map[uint]struct{}, 32)
	}
}
