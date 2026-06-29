package category

import (
	"blog/internal/model/entity"
	"errors"

	"gorm.io/gorm"
)

// categoryRepository 分类数据访问实现
type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository 创建分类仓储实例
func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

// FindByID 通过 ID 查找分类
func (r *categoryRepository) FindByID(id uint) (*entity.Category, error) {
	var category entity.Category
	err := r.db.First(&category, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

// ListPublic 获取所有启用的分类（前台）
func (r *categoryRepository) ListPublic() ([]*entity.Category, error) {
	var list []*entity.Category
	err := r.db.Model(&entity.Category{}).
		Where("status = ?", 1).
		Order("created_at DESC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// List 获取分页的分类列表（后台）
func (r *categoryRepository) List(page, pageSize int, status *int, keyword string) ([]*entity.Category, int64, error) {
	var list []*entity.Category
	var total int64

	query := r.db.Model(&entity.Category{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if keyword != "" {
		query = query.Where("category_name LIKE ?", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Create 创建分类
func (r *categoryRepository) Create(category *entity.Category) error {
	return r.db.Create(category).Error
}

// UpdateFields 更新分类的指定字段
func (r *categoryRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&entity.Category{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除分类（软删除）
func (r *categoryRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Category{}, id).Error
}

// IsExistsByName 判断分类名称是否存在（排除指定ID）
func (r *categoryRepository) IsExistsByName(name string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.Model(&entity.Category{}).Where("category_name = ?", name)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountByCategoryID 统计分类下的文章数
func (r *categoryRepository) CountByCategoryID(categoryID uint) (int64, error) {
	var count int64
	err := r.db.Model(&entity.Article{}).Where("type_id = ?", categoryID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
