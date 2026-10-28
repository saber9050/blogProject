package chunktemplate

import "blog/internal/model/entity"

// TemplateStat 单个模板的使用统计
type TemplateStat struct {
	ArticleCount int64 // 用该模板切过的文章数
	ChunkCount   int64 // 该模板下的块总数
}

// Repository 切分模板数据访问接口
type Repository interface {
	// List 列表；status 为 nil 表示不过滤
	List(status *int8) ([]*entity.ChunkTemplate, error)
	// FindByID 按 ID 查询，未找到返回 (nil, nil)
	FindByID(id uint) (*entity.ChunkTemplate, error)
	// ExistsByName 名称是否已存在；excludeID > 0 时排除自身
	ExistsByName(name string, excludeID uint) (bool, error)
	// Create 创建
	Create(t *entity.ChunkTemplate) error
	// UpdateFields 更新指定字段
	UpdateFields(id uint, fields map[string]interface{}) error
	// Delete 删除
	Delete(id uint) error
	// Stats 按模板聚合使用情况。article_chunks 不存在时返回空 map（不是错误）
	Stats() (map[uint]TemplateStat, error)
	// ChunkCount 某模板下的块数。article_chunks 不存在时返回 0
	ChunkCount(templateID uint) (int64, error)
}
