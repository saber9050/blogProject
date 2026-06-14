package entity

// Tag 标签实体
type Tag struct {
	BaseEntity
	TagName string `gorm:"type:varchar(100);not null;comment:标签名" json:"tag_name"`
	Status  int8   `gorm:"type:tinyint;not null;default:1;comment:状态:1启用,0禁用" json:"status"`
}

func (Tag) TableName() string {
	return "tags"
}

// TagArticle 文章标签关联
type TagArticle struct {
	BaseEntity
	ArticleID uint `gorm:"bigint;not null;comment:文章id" json:"article_id"`
	TagID     uint `gorm:"bigint;not null;comment:标签id" json:"tag_id"`
}

func (TagArticle) TableName() string {
	return "tag_articles"
}
