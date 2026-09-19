package llmconfig

import "blog/internal/model/entity"

// Repository 模型配置数据访问接口
type Repository interface {
	// List 获取全部配置（默认使用的排在前）
	List() ([]*entity.AIModelConfig, error)
	// FindByID 按 ID 查询，未找到返回 (nil, nil)
	FindByID(id uint) (*entity.AIModelConfig, error)
	// FindActive 获取当前生效配置（is_default=1 且 status=1），未找到返回 (nil, nil)
	FindActive() (*entity.AIModelConfig, error)
	// Create 创建配置
	Create(m *entity.AIModelConfig) error
	// UpdateFields 更新指定字段
	UpdateFields(id uint, fields map[string]interface{}) error
	// Delete 删除配置
	Delete(id uint) error
	// ClearDefault 清空所有配置的默认标记
	ClearDefault() error
	// SetDefault 将指定配置设为默认
	SetDefault(id uint) error
	// Transaction 在事务中执行，回调内使用的是事务内的 Repository
	Transaction(fn func(tx Repository) error) error
}
