package tag

import "blog/internal/model/entity"

// TagRepository 标签数据访问接口
type TagRepository interface {
	// FindByID 通过 ID 查找标签
	FindByID(id uint) (*entity.Tag, error)
	// FindByIDs 批量查找标签
	FindByIDs(ids []uint) ([]*entity.Tag, error)
	// ListPublic 获取所有启用的标签（前台）
	ListPublic() ([]*entity.Tag, error)
	// List 获取分页的标签列表（后台）
	List(page, pageSize int, status *int, keyword string) ([]*entity.Tag, int64, error)
	// Create 创建标签
	Create(tag *entity.Tag) error
	// Update 更新标签
	Update(tag *entity.Tag) error
	// Delete 删除标签（软删除）
	Delete(id uint) error
	// IsExistsByName 判断标签名称是否存在（排除指定ID）
	IsExistsByName(name string, excludeID uint) (bool, error)
	// DeleteTagArticles 删除标签与文章的关联关系
	DeleteTagArticles(tagID uint) error
}
