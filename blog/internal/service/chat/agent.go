package chat

import (
	"context"
	"sync"

	"blog/pkg/llmclient"
	"blog/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"go.uber.org/zap"
)

// agentName agent 名字。必须是稳定常量：它会被写进 tool 调用的来源标记里，
// 改了不影响功能，但会让日志和排查时的对照关系断掉。
const agentName = "article-tutor"

// agentEntry 一个模型对应的 agent 与 runner。
//
// 两者都是**无状态**的（请求级数据走 ctx 和消息，不进 Instruction），
// 所以可以跨请求复用。这是「按模型缓存」成立的前提。
type agentEntry struct {
	agent  adk.Agent
	runner *adk.Runner
	// modelName 模型名，落库时写进 chat_messages.model
	modelName string
}

// agentBuilder 按模型配置构建并缓存 agent。
//
// 为什么必须缓存：agent 构建时会把 ChatModel 与 tool 列表编译进内部结构，
// 每个请求都重建等于每问一句都重新初始化一次模型客户端。QPS 一上来这是纯浪费。
//
// 为什么 key 是模型配置 id：会话可以选不同模型（便宜的/强的），而模型是在
// agent 构建时绑死的，不能只换一次调用。所以只能一个模型一个 agent。
type agentBuilder struct {
	mu      sync.Mutex
	entries map[uint]*agentEntry
	llmCfg  llmconfigResolver
	tool    tool.BaseTool
}

// llmconfigResolver 只取 agentBuilder 真正用到的那一个方法。
// 用最小接口而不是整个 llmconfig.Service：这里不需要增删改查配置的能力，
// 依赖面越小，这个文件改动的理由就越少。
type llmconfigResolver interface {
	Resolve(id uint) (llmclient.Config, error)
}

func newAgentBuilder(llmCfg llmconfigResolver) *agentBuilder {
	return &agentBuilder{
		entries: make(map[uint]*agentEntry, 4),
		llmCfg:  llmCfg,
		tool:    newSearchTool(),
	}
}

// get 取（必要时构建）指定模型的 agent。
//
// 加锁范围覆盖整个构建过程：两个并发请求同时首次命中同一个模型时，重复构建虽然
// 无害（结果一样）但会白花两次初始化，而且 map 并发写会直接 panic。
func (b *agentBuilder) get(ctx context.Context, configID uint) (*agentEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if e, ok := b.entries[configID]; ok {
		return e, nil
	}

	cfg, err := b.llmCfg.Resolve(configID)
	if err != nil {
		// 这里不缓存失败：模型可能刚被管理员重新启用，缓存错误会让它一直不可用。
		return nil, err
	}

	chatModel, err := newChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        agentName,
		Description: "解答关于用户正在阅读的博客文章的问题",
		Instruction: tutorInstruction,
		Model:       chatModel,
		// MaxIterations 刻意不设，用 ADK 默认的 20。
		// 它只当「agent 彻底转死」的安全阀，正常永远碰不到；碰不到才是对的，
		// 碰到就说明 Instruction 或 tool 返回的内容出了 bug。
		// 业务上的「最多检索 5 次」由 toolGuard 那个中间件管（P4 接入）。
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{b.tool},
			},
		},
	})
	if err != nil {
		logger.Error("构建 agent 失败", zap.Uint("model_config_id", configID), zap.Error(err))
		return nil, err
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
		// 必须开：关掉的话每个事件只给完整消息，前端就不是逐字上屏了
		EnableStreaming: true,
	})

	e := &agentEntry{agent: agent, runner: runner, modelName: cfg.Model}
	b.entries[configID] = e
	logger.Info("已构建并缓存 agent", zap.Uint("model_config_id", configID),
		zap.String("model", cfg.Model))
	return e, nil
}

// invalidate 让某个模型的 agent 失效。后台改了配置（换密钥、换模型名、禁用）时调用。
//
// P8 接后台配置变更时用。现在先留好口子，避免到时候在配置服务里反向依赖 chat 包。
func (b *agentBuilder) invalidate(configID uint) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.entries, configID)
}

// invalidateAll 全量失效（例如切换向量模型这类会影响全局的配置变更）。
func (b *agentBuilder) invalidateAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = make(map[uint]*agentEntry, 4)
}
