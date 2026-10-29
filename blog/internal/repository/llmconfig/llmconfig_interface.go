package llmconfig

import "blog/internal/model/entity"

// Repository 模型配置数据访问接口
type Repository interface {
	// List 获取全部配置
	List() ([]*entity.AIModelConfig, error)
	// ListByType 按类型筛选；t=0 表示不筛选
	ListByType(t int8) ([]*entity.AIModelConfig, error)
	// FindByID 按 ID 查询，未找到返回 (nil, nil)
	FindByID(id uint) (*entity.AIModelConfig, error)
	// CountByType 统计指定类型的配置数；excludeID > 0 时排除该条
	CountByType(t int8, excludeID uint) (int64, error)
	// FindActiveEmbedding 查询启用中的向量模型（全局最多一条），未配置返回 (nil, nil)
	FindActiveEmbedding() (*entity.AIModelConfig, error)
	// Create 创建配置
	Create(m *entity.AIModelConfig) error
	// UpdateFields 更新指定字段
	UpdateFields(id uint, fields map[string]interface{}) error
	// Delete 删除配置
	Delete(id uint) error
}
