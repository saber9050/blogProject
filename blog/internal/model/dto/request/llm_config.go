package request

// 模型类型。与 entity.ModelTypeChat / ModelTypeEmbedding 对应，
// 但 DTO 层不直接引用 entity，避免 API 契约被内部结构绑死。
const (
	TypeChat      = 1
	TypeEmbedding = 2
)

// TestLLMConfigRequest 测试连接请求（不落库）
type TestLLMConfigRequest struct {
	ID         uint   `json:"id"` // >0 且 APIKey 为空时，复用该配置已存的密钥
	BaseURL    string `json:"base_url" binding:"required"`
	APIKey     string `json:"api_key"` // 可为空
	Model      string `json:"model" binding:"required"`
	TimeoutSec int    `json:"timeout_sec"`                        // 测试用超时，服务端上限 15s
	Type       int8   `json:"type" binding:"omitempty,oneof=1 2"` // 缺省 1=对话模型
	Dimension  int    `json:"dimension"`                          // 向量模型期望维度，用于比对实际输出
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

	// Type 1=对话模型（缺省） 2=向量模型。创建后不可改——
	// 同 base_url/api_key/model 的规则：换类型等于换用途，应当删了重建。
	Type int8 `json:"type" binding:"omitempty,oneof=1 2"`
	// Dimension 向量维度，仅 Type=2 时必填。它决定 article_chunks.embedding 列的维度，
	// 是 pgvector DDL 的一部分，所以必须与库表现有维度一致。
	Dimension int `json:"dimension"`
}

// UpdateLLMConfigRequest 更新模型配置请求（仅允许修改运行参数与状态，base_url/api_key/model 不可改）
type UpdateLLMConfigRequest struct {
	TimeoutSec  int     `json:"timeout_sec"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Status      *int8   `json:"status" binding:"omitempty,oneof=0 1"` // 缺省表示不修改
}
