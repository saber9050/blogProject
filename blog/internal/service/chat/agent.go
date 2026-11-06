package chat

import (
	"context"
	stderrors "errors"
	"fmt"
	"sync"

	"time"

	llmconfigSvc "blog/internal/service/llmconfig"
	"blog/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
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
	// builtAt 构建时间，用于 TTL 过期
	builtAt time.Time
}

// agentCacheTTL 缓存的存活时间。
//
// 为什么需要 TTL：agent 在构建时把「密钥、模型名、温度」都编进了内部结构。
// 管理员改了配置（换了 key、换了模型名）后，缓存不失效就会一直用旧值，
// 而 id 没变——从外部完全看不出问题，只会表现为「配置改了但没生效」。
//
// 为什么不用「配置变更时主动失效」：那需要 llmconfig 服务反向依赖 chat，
// 或者引入一套事件总线。为了一个几十秒的延迟引入这种耦合不值得。
// 60 秒的代价是「改完配置等一分钟」，换来的是零耦合。
const agentCacheTTL = 60 * time.Second

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
	llmCfg  llmconfigSvc.Service
	tool    tool.BaseTool
}

func newAgentBuilder(llmCfg llmconfigSvc.Service, searcher contextSearcher) *agentBuilder {
	return &agentBuilder{
		entries: make(map[uint]*agentEntry, 4),
		llmCfg:  llmCfg,
		tool:    newSearchTool(searcher),
	}
}

// get 取（必要时构建）指定模型的 agent。
//
// 加锁范围覆盖整个构建过程：两个并发请求同时首次命中同一个模型时，重复构建虽然
// 无害（结果一样）但会白花两次初始化，而且 map 并发写会直接 panic。
func (b *agentBuilder) get(ctx context.Context, configID uint) (*agentEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if e, ok := b.entries[configID]; ok && time.Since(e.builtAt) < agentCacheTTL {
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
				// 次数闸 + 单次超时 + 「最后一次」提示，全部在工具层（见 toolguard.go）
				ToolCallMiddlewares: []compose.ToolMiddleware{{Invokable: toolGuard}},
			},
		},
		// 第 1 层兜底：单次模型调用失败自动重试 2 次。
		ModelRetryConfig: &adk.TypedModelRetryConfig[*schema.Message]{
			MaxRetries: 2,
			ShouldRetry: func(callCtx context.Context, rc *adk.TypedRetryContext[*schema.Message]) *adk.TypedRetryDecision[*schema.Message] {
				// 没有错误 = 调用成功，不该重试（重试会白花一次调用）
				if rc.Err == nil {
					return nil
				}
				// ctx 已取消（用户点停止 / 整体超时）时重试毫无意义，只会拖慢退出
				if callCtx.Err() != nil {
					return nil
				}
				return &adk.TypedRetryDecision[*schema.Message]{Retry: true}
			},
		},
		// 第 2 层兜底：重试耗尽后换一个启用的对话模型再试一次。
		ModelFailoverConfig: &adk.ModelFailoverConfig[*schema.Message]{
			MaxRetries: 1,
			ShouldFailover: func(callCtx context.Context, _ *schema.Message, err error) bool {
				if err == nil || callCtx.Err() != nil {
					return false
				}
				// 用户主动中断不算「故障」，换模型重来只会让他再等一次
				return !stderrors.Is(err, context.Canceled)
			},
			GetFailoverModel: func(callCtx context.Context, _ *adk.FailoverContext[*schema.Message]) (
				model.BaseModel[*schema.Message], []*schema.Message, error) {
				m, err := b.fallbackModel(callCtx, configID)
				if err != nil {
					// 返回错误 = 停止故障转移，原始错误照常抛出，不会把真实原因盖掉
					return nil, nil, err
				}
				logger.Warn("主模型调用失败，已切换到备用模型",
					zap.Uint("primary_config_id", configID))
				return m, nil, nil
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

	e := &agentEntry{agent: agent, runner: runner, modelName: cfg.Model, builtAt: time.Now()}
	b.entries[configID] = e
	logger.Info("已构建并缓存 agent", zap.Uint("model_config_id", configID),
		zap.String("model", cfg.Model))
	return e, nil
}

// fallbackModel 取一个「配置文件里启用、且不是当前这个」的对话模型。
//
// 局限（已知并接受）：failover 成功后，落库的 chat_messages.model 仍是主模型的
// 名字——框架不会告诉我们最终用了哪个。要精确记录就得在 failover 时写回 scope，
// 但那个回调拿不到请求级 ctx 的可靠保证，收益也不大（用户关心的是「有没有答」，
// 不是「哪个模型兜的底」）。日志里记录了切换动作，排查够用。
func (b *agentBuilder) fallbackModel(ctx context.Context, excludeID uint) (model.BaseModel[*schema.Message], error) {
	list, err := b.llmCfg.ListByType(1)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if item.ID == excludeID || item.Status != 1 {
			continue
		}
		cfg, err := b.llmCfg.Resolve(item.ID)
		if err != nil {
			// 单条解析失败（密钥坏了之类）不该让整个转移过程失败，继续看下一个
			continue
		}
		m, err := newChatModel(ctx, cfg)
		if err != nil {
			continue
		}
		return m, nil
	}
	return nil, fmt.Errorf("没有可用的备用对话模型")
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
