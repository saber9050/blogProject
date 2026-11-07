package chat

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// fakeToolCallModel 一个只会「先调工具、再作答」的假模型。
//
// 存在的意义：把「框架有没有把 tool 交给模型、模型的 tool_call 会不会真的被执行、
// 执行结果会不会回填」这条链验证掉，**不需要真实的 LLM**。
// 真实模型不调工具时，你无法区分是「模型/provider 的问题」还是「我们接线错了」——
// 而这个测试能把后者排除掉。
type fakeToolCallModel struct {
	rounds int
	// sawTools 模型收到的 tools 列表长度（模拟 provider 通过 option 收到工具）
	sawTools []*schema.ToolInfo
}

func (f *fakeToolCallModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return f.next(ctx, in)
}

func (f *fakeToolCallModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	m, err := f.next(ctx, in)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{m}), nil
}

func (f *fakeToolCallModel) next(ctx context.Context, in []*schema.Message) (*schema.Message, error) {
	f.rounds++
	if f.rounds == 1 {
		// 第一轮：要求检索
		return &schema.Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: schema.FunctionCall{
					Name:      searchToolName,
					Arguments: `{"query":"Stream 消费组"}`,
				},
			}},
		}, nil
	}
	// 第二轮：拿到工具结果后作答
	return &schema.Message{Role: schema.Assistant, Content: "消费组是独立消费视角。"}, nil
}

// fakeSearcher 记录被调用次数的检索实现。
type fakeSearcher struct {
	calls int
	query string
}

func (f *fakeSearcher) Search(ctx context.Context, sc *Scope, query string) (SearchResult, error) {
	f.calls++
	f.query = query
	return SearchResult{
		Text:     "[1] 《测试文章》· 二，Stream\n消费组是独立消费视角。",
		Hits:     1,
		Articles: []string{"测试文章"},
	}, nil
}

// TestAgentActuallyCallsTool 端到端验证「工具真的会被调用」。
//
// 这条测试是排查「AI 从不检索」的分水岭：
//   - 通过 ⇒ 我们的接线（tool 定义、ToolsConfig、事件消费、scope 传递）没问题，
//     问题出在模型/网关不支持或不使用 tools
//   - 不通过 ⇒ 是代码问题，测试会指出断在哪一环
func TestAgentActuallyCallsTool(t *testing.T) {
	ctx := context.Background()
	searcher := &fakeSearcher{}
	tl := newSearchTool(searcher)

	mdl := &fakeToolCallModel{}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        agentName,
		Instruction: tutorInstruction,
		Model:       mdl,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               []tool.BaseTool{tl},
				ToolCallMiddlewares: []compose.ToolMiddleware{{Invokable: toolGuard}},
			},
		},
	})
	if err != nil {
		t.Fatalf("构建 agent 失败: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true})

	// scope 必须放进 ctx：tool 通过它拿检索范围。
	// 这一步同时也是在验证「scope 真的能穿透到工具里」——穿透不了的话
	// runSearch 会返回「检索范围无效」，测试会在 hits 断言上失败。
	sc := &Scope{CurrentArticleID: 1, CurrentArticleTitle: "测试文章"}
	sc.EnsureSeen()
	runCtx := withScope(ctx, sc)

	iter := runner.Run(runCtx, []*schema.Message{{Role: schema.User, Content: "消费组是什么？"}})
	sink := &recordSink{}
	res := consumeAgentStream(runCtx, iter, sink, sc)

	if res.err != nil {
		t.Fatalf("agent 执行出错: %v", res.err)
	}
	if searcher.calls == 0 {
		t.Fatalf("工具一次都没被调用。事件序列: %v\n"+
			"可能原因：ToolsConfig 没传进去 / tool 定义无效 / scope 解析失败", sink.events)
	}
	if searcher.query != "Stream 消费组" {
		t.Fatalf("工具收到的 query = %q，期望 %q", searcher.query, "Stream 消费组")
	}
	if len(sc.Steps) != 1 {
		t.Fatalf("检索步骤数 = %d，期望 1（说明 tool 里的 BeginStep/EndStep 没走到）", len(sc.Steps))
	}
	if sc.Steps[0].Hits != 1 {
		t.Fatalf("步骤命中数 = %d，期望 1", sc.Steps[0].Hits)
	}
	if res.content == "" {
		t.Fatal("最终回答为空")
	}

	t.Logf("事件序列: %v", sink.events)
	t.Logf("检索步骤: %+v", sc.Steps[0])
}
