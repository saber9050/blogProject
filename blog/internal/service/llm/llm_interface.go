package llm

import "blog/internal/model/dto/response"

// LLMService LLM 服务接口
type LLMService interface {
	// GenerateSummary 使用指定模型配置生成摘要
	GenerateSummary(configID uint, title string, content string) (*response.GenerateSummaryResponse, error)
}
