package llmconfig

import (
	"blog/internal/model/entity"
	"errors"

	"gorm.io/gorm"
)

// llmConfigRepository 模型配置数据访问实现
type llmConfigRepository struct {
	db *gorm.DB
}

// NewLLMConfigRepository 创建模型配置仓储实例
func NewLLMConfigRepository(db *gorm.DB) Repository {
	return &llmConfigRepository{db: db}
}

// List 获取全部配置
func (r *llmConfigRepository) List() ([]*entity.AIModelConfig, error) {
	var list []*entity.AIModelConfig
	if err := r.db.Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// FindByID 按 ID 查询
func (r *llmConfigRepository) FindByID(id uint) (*entity.AIModelConfig, error) {
	var m entity.AIModelConfig
	err := r.db.First(&m, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// Create 创建配置
func (r *llmConfigRepository) Create(m *entity.AIModelConfig) error {
	return r.db.Create(m).Error
}

// UpdateFields 更新指定字段
func (r *llmConfigRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&entity.AIModelConfig{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除配置
func (r *llmConfigRepository) Delete(id uint) error {
	return r.db.Delete(&entity.AIModelConfig{}, id).Error
}
