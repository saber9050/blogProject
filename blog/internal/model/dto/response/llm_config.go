package response

// LLMConfigResponse 模型配置响应（密钥脱敏）
type LLMConfigResponse struct {
	ID           uint    `json:"id"`
	BaseURL      string  `json:"base_url"`
	APIKeyMasked string  `json:"api_key_masked"` // 脱敏展示，绝不返回明文
	Model        string  `json:"model"`
	Type         int8    `json:"type"`      // 1=对话模型 2=向量模型
	Dimension    int     `json:"dimension"` // 向量维度，仅 Type=2 有效
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
	// Dimension 向量模型才有值：模型实际输出的维度。
	// 前端拿它和填写的 dimension 对比，用户能立刻看出自己填错了没有。
	Dimension int    `json:"dimension,omitempty"`
	Error     string `json:"error,omitempty"`
}
