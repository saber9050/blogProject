package category

import "blog/internal/model/entity"

// CategoryRepository 分类数据访问接口
type CategoryRepository interface {
	// FindByID 通过 ID 查找分类
	FindByID(id uint) (*entity.Category, error)
	// ListPublic 获取所有启用的分类（前台）
	ListPublic() ([]*entity.Category, error)
	// List 获取分页的分类列表（后台）
	List(page, pageSize int, status *int, keyword string) ([]*entity.Category, int64, error)
	// Create 创建分类
	Create(category *entity.Category) error
	// UpdateFields 更新分类的指定字段
	UpdateFields(id uint, fields map[string]interface{}) error
	// Delete 删除分类（软删除）
	Delete(id uint) error
	// IsExistsByName 判断分类名称是否存在（排除指定ID）
	IsExistsByName(name string, excludeID uint) (bool, error)
	// CountByCategoryID 统计分类下的文章数
	CountByCategoryID(categoryID uint) (int64, error)
}
