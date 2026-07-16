package llm

import "blog/internal/model/dto/response"

// LLMService LLM 服务接口
type LLMService interface {
	// GenerateSummary 根据文章标题和内容生成摘要
	GenerateSummary(title string, content string) (*response.GenerateSummaryResponse, error)
}
