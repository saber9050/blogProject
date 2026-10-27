package entity

// 模型类型
const (
	ModelTypeChat      int8 = 1 // 对话模型：打 /chat/completions
	ModelTypeEmbedding int8 = 2 // 向量模型：打 /embeddings，全局只允许一条
)

// AIModelConfig 大模型配置
type AIModelConfig struct {
	BaseEntity
	// Name 配置名称：后台不再单独配置，创建/更新时自动取「模型 id」填充。
	// 保留该列是为了满足历史 not null 约束（GORM AutoMigrate 不会删列）。
	Name    string `gorm:"type:varchar(64);not null;comment:配置名称(自动取模型id)" json:"-"`
	BaseURL string `gorm:"type:varchar(255);not null;comment:API基址(到/v1)" json:"base_url"`
	APIKey  string `gorm:"type:varchar(512);not null;comment:密钥" json:"-"` // 绝不序列化
	Model   string `gorm:"type:varchar(128);not null;comment:模型id" json:"model"`
	// Type 模型类型，存量数据经 default:1 自动成为对话模型
	Type int8 `gorm:"type:smallint;not null;default:1;comment:类型:1对话模型,2向量模型" json:"type"`
	// Dimension 向量维度。不只是记录字段，它是 article_chunks.embedding 列的维度来源：
	// pgvector 的 vector(n) 维度写在 DDL 里，改不了，所以必须与建表时的值一致。
	// 仅 Type=2 时有效，对话模型恒为 0。
	Dimension   int     `gorm:"not null;default:0;comment:向量维度,仅type=2有效" json:"dimension"`
	TimeoutSec  int     `gorm:"default:30;comment:单次超时秒" json:"timeout_sec"`
	MaxTokens   int     `gorm:"default:2048;comment:最大生成token" json:"max_tokens"`
	Temperature float64 `gorm:"default:0.3;comment:采样温度" json:"temperature"`
	Status      int8    `gorm:"type:smallint;not null;comment:状态:1启用,0禁用" json:"status"`
}

func (AIModelConfig) TableName() string {
	return "ai_model_configs"
}
