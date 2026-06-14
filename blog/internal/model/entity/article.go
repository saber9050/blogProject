package entity

import "gorm.io/gorm"

// Article 文章实体
type Article struct {
	BaseEntity
	UserID    uint           `gorm:"type:bigint;not null;comment:文章作者id" json:"user_id"`
	Title     string         `gorm:"type:varchar(100);not null;comment:文章标题" json:"title"`
	Content   string         `gorm:"type:longtext;not null;comment:文章内容" json:"content"`
	CoverURL  string         `gorm:"type:varchar(255);comment:封面图片url" json:"cover_url"`
	Summary   string         `gorm:"type:varchar(255);comment:摘要" json:"summary"`
	Views     uint           `gorm:"type:bigint;comment:浏览量" json:"views"`
	TypeID    uint           `gorm:"type:bigint;not null;comment:文章分类id" json:"type_id"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Article) TableName() string {
	return "articles"
}
