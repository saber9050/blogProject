package entity

// AIModelConfig 大模型配置
type AIModelConfig struct {
	BaseEntity
	// Name 配置名称：后台不再单独配置，创建/更新时自动取「模型 id」填充。
	// 保留该列是为了满足历史 not null 约束（GORM AutoMigrate 不会删列）。
	Name        string  `gorm:"type:varchar(64);not null;comment:配置名称(自动取模型id)" json:"-"`
	BaseURL     string  `gorm:"type:varchar(255);not null;comment:API基址(到/v1)" json:"base_url"`
	APIKey      string  `gorm:"type:varchar(512);not null;comment:密钥" json:"-"` // 绝不序列化
	Model       string  `gorm:"type:varchar(128);not null;comment:模型id" json:"model"`
	TimeoutSec  int     `gorm:"default:30;comment:单次超时秒" json:"timeout_sec"`
	MaxTokens   int     `gorm:"default:2048;comment:最大生成token" json:"max_tokens"`
	Temperature float64 `gorm:"default:0.3;comment:采样温度" json:"temperature"`
	Status      int8    `gorm:"type:tinyint;not null;default:1;comment:状态:1启用,0禁用" json:"status"`
	IsDefault   bool    `gorm:"default:false;comment:是否为当前使用" json:"is_default"`
}

func (AIModelConfig) TableName() string {
	return "ai_model_configs"
}
