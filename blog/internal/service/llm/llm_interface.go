package llm

// LLMService LLM 服务接口
type LLMService interface {
	// GenerateSummary 根据文章内容生成摘要
	GenerateSummary(content string) (string, error)
}
