package chat

import (
	"context"
	"fmt"

	"blog/pkg/llmclient"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// toOpenAIConfig 把项目自己的 llmclient.Config 映射成 eino-ext 的 openai 配置。
//
// 单独抽一个函数而不是散在 agent 构建里，是因为这里有几个**静默出错**的转换：
//
//   - Temperature：llmclient 用 float64，eino-ext 要 *float32。直接 &cfg.Temperature
//     拿到的是 *float64，编译就过不了；但写成 float32(0) 又会让「温度配 0」和
//     「没配温度」混为一谈。
//   - MaxTokens：eino-ext 是 *int。传 &0 的含义是「最多生成 0 个 token」，
//     不是「不限制」——会让模型一个字都吐不出来。
//   - Timeout：eino-ext 的语义是 time.Duration，0 表示**不超时**（官方注释明确）。
//     我们靠 ctx 的 120s 控住整条链，所以这里传 0 是主动选择，不是漏配。
func toOpenAIConfig(cfg llmclient.Config) *openai.ChatModelConfig {
	conf := &openai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	}

	if cfg.MaxTokens > 0 {
		mt := cfg.MaxTokens
		conf.MaxTokens = &mt
	}
	// Temperature 的取值范围是 0~2，0 是合法且有意义的值（求确定性输出）。
	// 配置表里默认 0.3，真拿到 0 说明管理员就是要确定性，照传。
	if cfg.Temperature >= 0 {
		t := float32(cfg.Temperature)
		conf.Temperature = &t
	}

	// 不用 cfg.TimeoutSec 做客户端超时：那会把「整条 agent 链」卡在单次调用上。
	// 超时统一由 SendMessage 里的 context.WithTimeout 控制（120s），
	// 好处是它同时能中断工具执行和后续轮次，而不只是 HTTP 请求。

	return conf
}

// newChatModel 按项目配置创建一个 eino ChatModel。
//
// 单独一层是为了把「配置映射」（toOpenAIConfig）与「组件构造」分开：
// 前者是纯函数、可以直接单测；后者要网络上下文，只能集成测。
func newChatModel(ctx context.Context, cfg llmclient.Config) (model.BaseModel[*schema.Message], error) {
	cm, err := openai.NewChatModel(ctx, toOpenAIConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("创建模型客户端失败: %w", err)
	}
	return cm, nil
}
