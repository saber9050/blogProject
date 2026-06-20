package tag

import (
	"blog/internal/model/entity"
	"errors"

	"gorm.io/gorm"
)

// tagRepository 标签数据访问实现
type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository 创建标签仓储实例
func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

// FindByID 通过 ID 查找标签
func (r *tagRepository) FindByID(id uint) (*entity.Tag, error) {
	var tag entity.Tag
	err := r.db.First(&tag, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tag, nil
}

// FindByIDs 批量查找标签
func (r *tagRepository) FindByIDs(ids []uint) ([]*entity.Tag, error) {
	var list []*entity.Tag
	err := r.db.Where("id IN ?", ids).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListPublic 获取所有启用的标签（前台）
func (r *tagRepository) ListPublic() ([]*entity.Tag, error) {
	var list []*entity.Tag
	err := r.db.Model(&entity.Tag{}).
		Where("status = ?", 1).
		Order("created_at DESC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// List 获取分页的标签列表（后台）
func (r *tagRepository) List(page, pageSize int, status *int, keyword string) ([]*entity.Tag, int64, error) {
	var list []*entity.Tag
	var total int64

	query := r.db.Model(&entity.Tag{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
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

// Create 创建标签
func (r *tagRepository) Create(tag *entity.Tag) error {
	return r.db.Create(tag).Error
}

// Update 更新标签
func (r *tagRepository) Update(tag *entity.Tag) error {
	return r.db.Save(tag).Error
}

// Delete 删除标签（软删除）
func (r *tagRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Tag{}, id).Error
}

// IsExistsByName 判断标签名称是否存在（排除指定ID）
func (r *tagRepository) IsExistsByName(name string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.Model(&entity.Tag{}).Where("name = ?", name)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteTagArticles 删除标签与文章的关联关系
func (r *tagRepository) DeleteTagArticles(tagID uint) error {
	return r.db.Where("tag_id = ?", tagID).Delete(&entity.TagArticle{}).Error
}
