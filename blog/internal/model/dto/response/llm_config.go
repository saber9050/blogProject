package response

// LLMConfigResponse 模型配置响应（密钥脱敏）
type LLMConfigResponse struct {
	ID           uint    `json:"id"`
	BaseURL      string  `json:"base_url"`
	APIKeyMasked string  `json:"api_key_masked"` // 脱敏展示，绝不返回明文
	Model        string  `json:"model"`
	TimeoutSec   int     `json:"timeout_sec"`
	MaxTokens    int     `json:"max_tokens"`
	Temperature  float64 `json:"temperature"`
	Status       int8    `json:"status"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// TestConnectionResponse 测试连接结果
type TestConnectionResponse struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latency_ms"`
	Reply     string `json:"reply,omitempty"`
	Error     string `json:"error,omitempty"`
}
