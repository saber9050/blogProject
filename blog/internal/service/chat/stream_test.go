package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// 这组测试直接构造 ADK 事件流喂给 consumeAgentStream，不依赖模型、不依赖数据库。
//
// 为什么值得单独测：事件映射是本模块最容易写错、且错了最难发现的一层。
// 写错的表现不是报错，而是「前端只看到最终答案、没有检索进度」或者
// 「tool 事件和 delta 顺序颠倒」——这类问题在真实调用里要盯很久才看得出来，
// 而构造事件只需要几行。
//
// 三类事件靠 MessageOutput.Role 区分（不是猜内容）：
//   - Assistant + ToolCalls  → tool start
//   - Tool                   → tool end
//   - Assistant + Content    → delta

// recordSink 记录收到的事件顺序，用于断言。
type recordSink struct {
	events []string
	deltas string
}

func (r *recordSink) OnMeta(MetaEvent) error { r.events = append(r.events, "meta"); return nil }

func (r *recordSink) OnToolStart(round int, name, query string) error {
	r.events = append(r.events, "tool_start:"+name+":"+query)
	return nil
}

func (r *recordSink) OnToolEnd(round, hits, chars int, articles []string) error {
	r.events = append(r.events, "tool_end")
	return nil
}

func (r *recordSink) OnDelta(text string) error {
	r.events = append(r.events, "delta")
	r.deltas += text
	return nil
}

func (r *recordSink) OnError(code int, message string, partial bool) error {
	r.events = append(r.events, "error")
	return nil
}

func assistantEvent(chunks ...*schema.Message) *adk.AgentEvent {
	return &adk.AgentEvent{Output: &adk.AgentOutput{
		MessageOutput: &adk.MessageVariant{
			IsStreaming:   true,
			MessageStream: schema.StreamReaderFromArray(chunks),
			Role:          schema.Assistant,
		},
	}}
}

func toolEvent(content string) *adk.AgentEvent {
	return &adk.AgentEvent{Output: &adk.AgentOutput{
		MessageOutput: &adk.MessageVariant{
			IsStreaming: false,
			Message:     &schema.Message{Role: schema.Tool, Content: content},
			Role:        schema.Tool,
		},
	}}
}

func TestConsumeAgentStreamToolThenAnswer(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	// 轮 1：模型决定检索（只带 ToolCall，没有正文）
	gen.Send(assistantEvent(&schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID:       "call-1",
			Function: schema.FunctionCall{Name: searchToolName, Arguments: `{"query":"Stream 消费组"}`},
		}},
	}))
	// 工具结果
	gen.Send(toolEvent("[1] 《Redis 发布订阅和 Stream》· 二，Stream\n正文……"))
	// 轮 2：模型输出正文
	gen.Send(assistantEvent(
		&schema.Message{Role: schema.Assistant, Content: "两者的"},
		&schema.Message{Role: schema.Assistant, Content: "定位不同。"},
	))
	gen.Close()

	sink := &recordSink{}
	sc := &Scope{CurrentArticleID: 12, CurrentArticleTitle: "本文", RefTitles: map[uint]string{}}
	// tool 自身会累加 Rounds/ToolCalls，这里模拟它执行过一次。
	// ⚠️ 必须连 ToolCalls 一起模拟：零检索守卫看的就是它，
	// 只设 Rounds 会被判定成「没检索过」，正文直接被替换成固定文案。
	sc.Rounds = 1
	sc.ToolCalls = 1

	res := consumeAgentStream(context.Background(), iter, sink, sc)

	// 注意：只有 1 个 delta。拒答守卫会缓冲前 40 字（此回答才 8 字），
	// 到 flush 时一次性吐出——这是刻意的代价，换来的是「不会先闪半截标记再被替换」。
	want := []string{"tool_start:search_article_context:Stream 消费组", "tool_end", "delta"}
	if len(sink.events) != len(want) {
		t.Fatalf("事件序列不符:\n got %v\nwant %v", sink.events, want)
	}
	for i := range want {
		if sink.events[i] != want[i] {
			t.Fatalf("第 %d 个事件不符:\n got %v\nwant %v", i, sink.events, want)
		}
	}
	if res.content != "两者的定位不同。" {
		t.Fatalf("拼出的正文不对: %q", res.content)
	}
	if res.err != nil {
		t.Fatalf("不应有错误: %v", res.err)
	}
}

// TestConsumeAgentStreamNoRetrieval 零检索守卫。
//
// 这是本模块最重要的一条防线：模型的上下文里**只有文章标题，没有正文**
// （见 buildMessages）。一次检索都没发生时，它输出的任何"文章内容"都是
// 顺着标题编的，还会仿照消息前缀说"出自《xxx》"——用户完全分辨不出来。
//
// 所以这里断言的不是"答案对不对"，而是"编造的内容根本出不去"。
func TestConsumeAgentStreamNoRetrieval(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(assistantEvent(
		&schema.Message{Role: schema.Assistant, Content: "这篇文章主要介绍了 Go 语言中处理 Web"},
		&schema.Message{Role: schema.Assistant, Content: "表单的方法，出自《goweb表单》。"},
	))
	gen.Close()

	sink := &recordSink{}
	res := consumeAgentStream(context.Background(), iter, sink, &Scope{})

	if !res.noRetrieval {
		t.Fatal("一次检索都没有，应判定为 noRetrieval")
	}
	if res.content != NoRetrievalReply {
		t.Fatalf("编造的内容必须被整段替换，实际: %q", res.content)
	}
	if strings.Contains(sink.deltas, "Web") {
		t.Fatalf("编造内容泄漏到了前端: %q", sink.deltas)
	}
}

// TestConsumeAgentStreamRefusedBeforeNoRetrieval 拒答优先于零检索。
//
// 跑题时本来就不会调工具，ToolCalls 同样是 0。如果顺序反了，
// 「问天气」会被说成「没读到文章」，用户会以为站点坏了。
func TestConsumeAgentStreamRefusedBeforeNoRetrieval(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(assistantEvent(&schema.Message{Role: schema.Assistant, Content: offTopicMarker}))
	gen.Close()

	sink := &recordSink{}
	res := consumeAgentStream(context.Background(), iter, sink, &Scope{})

	if !res.refused {
		t.Fatal("应判定为跑题")
	}
	if res.noRetrieval {
		t.Fatal("跑题不该被判成「没检索到资料」")
	}
	if res.content != OffTopicReply {
		t.Fatalf("文案应为拒答文案，实际: %q", res.content)
	}
}

func TestConsumeAgentStreamError(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(assistantEvent(&schema.Message{Role: schema.Assistant, Content: "已经生成了一部分"}))
	gen.Send(&adk.AgentEvent{Err: context.DeadlineExceeded})
	gen.Close()

	sink := &recordSink{}
	// 已检索过才谈得上"保留已生成内容"：零检索的内容本来就该被替换
	sc := &Scope{ToolCalls: 1}
	res := consumeAgentStream(context.Background(), iter, sink, sc)

	// 关键：出错时**已生成的内容不能丢**，否则用户既白等又白烧 token
	if res.content != "已经生成了一部分" {
		t.Fatalf("出错时已生成内容应为 %q，实际 %q", "已经生成了一部分", res.content)
	}
	if res.err == nil {
		t.Fatal("应记录错误")
	}
}
