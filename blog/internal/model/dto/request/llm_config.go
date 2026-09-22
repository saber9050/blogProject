package request

// TestLLMConfigRequest 测试连接请求（不落库）
type TestLLMConfigRequest struct {
	ID         uint   `json:"id"` // >0 且 APIKey 为空时，复用该配置已存的密钥
	BaseURL    string `json:"base_url" binding:"required"`
	APIKey     string `json:"api_key"` // 可为空
	Model      string `json:"model" binding:"required"`
	TimeoutSec int    `json:"timeout_sec"` // 测试用超时，服务端上限 15s
}

// CreateLLMConfigRequest 新建模型配置请求
type CreateLLMConfigRequest struct {
	BaseURL     string  `json:"base_url" binding:"required"`
	APIKey      string  `json:"api_key" binding:"required"`
	Model       string  `json:"model" binding:"required"`
	TimeoutSec  int     `json:"timeout_sec"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Status      *int8   `json:"status" binding:"omitempty,oneof=0 1"` // 缺省视为启用
}

// UpdateLLMConfigRequest 更新模型配置请求（仅允许修改运行参数与状态，base_url/api_key/model 不可改）
type UpdateLLMConfigRequest struct {
	TimeoutSec  int     `json:"timeout_sec"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Status      *int8   `json:"status" binding:"omitempty,oneof=0 1"` // 缺省表示不修改
}
