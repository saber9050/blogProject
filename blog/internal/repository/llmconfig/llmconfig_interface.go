package llmconfig

import "blog/internal/model/entity"

// Repository 模型配置数据访问接口
type Repository interface {
	// List 获取全部配置
	List() ([]*entity.AIModelConfig, error)
	// FindByID 按 ID 查询，未找到返回 (nil, nil)
	FindByID(id uint) (*entity.AIModelConfig, error)
	// Create 创建配置
	Create(m *entity.AIModelConfig) error
	// UpdateFields 更新指定字段
	UpdateFields(id uint, fields map[string]interface{}) error
	// Delete 删除配置
	Delete(id uint) error
}
