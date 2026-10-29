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

// ListByType 按类型筛选；t=0 表示不筛选
func (r *llmConfigRepository) ListByType(t int8) ([]*entity.AIModelConfig, error) {
	var list []*entity.AIModelConfig
	q := r.db.Order("id ASC")
	if t != 0 {
		q = q.Where("type = ?", t)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// CountByType 统计指定类型的配置数；excludeID > 0 时排除该条（更新场景用）
func (r *llmConfigRepository) CountByType(t int8, excludeID uint) (int64, error) {
	var n int64
	q := r.db.Model(&entity.AIModelConfig{}).Where("type = ?", t)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// FindActiveEmbedding 查询启用中的向量模型（全局最多一条）
//
// 只取 status=1。已禁用的向量模型不该被用于向量化——那属于「配了但关掉了」，
// 报错比静默使用更合理。
func (r *llmConfigRepository) FindActiveEmbedding() (*entity.AIModelConfig, error) {
	var m entity.AIModelConfig
	err := r.db.Where("type = ? AND status = ?", entity.ModelTypeEmbedding, 1).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
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
