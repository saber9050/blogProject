package chat

import (
	"encoding/json"
	"fmt"
	"net/http"

	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	chatSvc "blog/internal/service/chat"

	"github.com/gin-gonic/gin"
)

// sseSink 把 service 推来的事件编码成 SSE 帧。
//
// 关键设计：**延迟写 header**（ensureStarted）。
//
// 如果一进来就 WriteHeader(200)，那么「会话不存在」「提问为空」「没有可用模型」这些
// 在 meta 之前就能判定的错误就再也没法返回正常的 404/400 了——HTTP 状态码已经定型。
// 前端只能收到 200 + 一个 error 事件，弹出来的还是红色错误，体验和排查都更糟。
//
// 延迟之后：第一个事件（一定是 meta）到来时才切到流式；在那之前失败就走普通 JSON。
type sseSink struct {
	c *gin.Context
	// started header 是否已写出（一旦写出就不能再改状态码，也不能再返回 JSON）
	started bool
	// wroteDelta 是否已经推过正文。用于 error 事件的 partial 标记：
	// 推过就必须让前端保留已显示的内容，不能清空气泡。
	wroteDelta bool
}

func newSSESink(c *gin.Context) *sseSink {
	return &sseSink{c: c}
}

// ensureStarted 第一次真正要写事件时才切到 SSE。
func (s *sseSink) ensureStarted() {
	if s.started {
		return
	}
	s.started = true
	h := s.c.Writer.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	// 告诉 nginx 不要缓冲。即使 nginx 侧漏配了 proxy_buffering off，
	// 有这行也能兜住（nginx 认得 X-Accel-Buffering）。
	h.Set("X-Accel-Buffering", "no")
	s.c.Status(http.StatusOK)
	s.c.Writer.Flush()
}

// write 写一帧。每次写完立刻 Flush，否则全攒在缓冲区里，
// 用户看到的是「转圈 10 秒，然后整段答案一次出现」。
func (s *sseSink) write(event string, data any) error {
	s.ensureStarted()
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// 手拼帧头而不是用 JSON 包一层：SSE 协议要求 "event:" 与 "data:" 分两行，
	// data 里只能是单行 JSON（换行由 json.Marshal 转义成 \n）。
	if _, err := fmt.Fprintf(s.c.Writer, "event: %s\ndata: %s\n\n", event, b); err != nil {
		return err
	}
	s.c.Writer.Flush()
	return nil
}

func (s *sseSink) OnMeta(m chatSvc.MetaEvent) error {
	// meta 只给「校验通过的引用文章」。
	// ⚠️ 这里给不出「参与检索的篇数」——此刻 agent 还没决定要不要检索、查几篇，
	// 硬塞一个数字只会是错的。真实统计在 done 事件里。
	refs := m.ArticleRefs
	if refs == nil {
		refs = []entity.ChatArticleRef{}
	}
	return s.write("meta", gin.H{
		"session_id":           m.SessionID,
		"user_message_id":      m.UserMessageID,
		"assistant_message_id": m.AssistantMessageID,
		"model":                m.Model,
		"model_fallback":       m.ModelFallback,
		"article_refs":         refs,
	})
}

func (s *sseSink) OnToolStart(round int, name, query string) error {
	return s.write("tool", gin.H{
		"phase": "start",
		"round": round,
		"name":  name,
		"query": query,
	})
}

func (s *sseSink) OnToolEnd(round, hits, chars int, articles []string) error {
	if articles == nil {
		articles = []string{}
	}
	return s.write("tool", gin.H{
		"phase":    "end",
		"round":    round,
		"hits":     hits,
		"chars":    chars,
		"articles": articles,
	})
}

func (s *sseSink) OnDelta(text string) error {
	s.wroteDelta = true
	return s.write("delta", gin.H{"content": text})
}

func (s *sseSink) OnError(code int, message string, partial bool) error {
	return s.write("error", gin.H{
		"code":    code,
		"message": message,
		// partial=true 表示之前已经推过内容，前端必须保留、只在末尾加一行「回答中断」
		"partial": partial,
	})
}

// done 发收尾帧：先 refs（如果有），再 done。
//
// refs 放在 done 之前而不是每轮检索后发一次：多轮检索只有全部跑完才知道总和，
// 中间发的话前端会看到「引用块越滚越多」，而且还要自己去重。
func (s *sseSink) done(res *response.SendChatMessageResponse) error {
	if len(res.Refs) > 0 {
		if err := s.write("refs", gin.H{"chunks": res.Refs}); err != nil {
			return err
		}
	}
	searched := res.SearchedArticles
	if searched == nil {
		searched = []uint{}
	}
	// steps 放进 done：前端流式期间的过程面板是**靠 tool 事件拼**的，
	// 一旦某帧事件在传输/解析中丢了，用户就会永久看不到这次检索——
	// 而这恰恰是他判断「答得靠不靠谱」的唯一依据。done 里带一份服务端记录，
	// 前端以它为准覆盖本地拼的，过程就不会因为丢帧而消失。
	steps := res.Steps
	if steps == nil {
		steps = []entity.ChatStep{}
	}
	return s.write("done", gin.H{
		"assistant_message_id": res.AssistantMessageID,
		"prompt_tokens":        res.PromptTokens,
		"completion_tokens":    res.CompletionTokens,
		"refused":              res.Refused,
		"interrupted":          res.Interrupted,
		"rounds":               res.Rounds,
		"tool_calls":           res.ToolCalls,
		"searched_articles":    searched,
		"degraded":             res.Degraded,
		"steps":                steps,
	})
}
