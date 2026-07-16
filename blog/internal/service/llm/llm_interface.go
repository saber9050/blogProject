package llm

// LLMService LLM 服务接口
type LLMService interface {
	// GenerateSummary 根据文章标题和内容生成摘要
	GenerateSummary(title string, content string) (string, error)
}
