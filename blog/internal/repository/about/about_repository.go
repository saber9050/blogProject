package about

import (
	"blog/internal/model/entity"
	"errors"

	"gorm.io/gorm"
)

// aboutRepository 关于页面数据访问实现
type aboutRepository struct {
	db *gorm.DB
}

// NewAboutRepository 创建关于页面仓储实例
func NewAboutRepository(db *gorm.DB) AboutRepository {
	return &aboutRepository{db: db}
}

// GetAbout 获取关于页面配置（单例）
func (r *aboutRepository) GetAbout() (*entity.About, error) {
	var about entity.About
	err := r.db.First(&about).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &about, nil
}

// UpdateAbout 更新关于页面配置（部分更新）
func (r *aboutRepository) UpdateAbout(about *entity.About) error {
	return r.db.Save(about).Error
}

// CreateAbout 创建关于页面配置（单例）
func (r *aboutRepository) CreateAbout(about *entity.About) error {
	return r.db.Create(about).Error
}
