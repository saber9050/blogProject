package entity

// Like 文章点赞实体
type Like struct {
	BaseEntity
	ArticleID uint `gorm:"bigint;not null;comment:文章id" json:"article_id"`
	UserID    uint `gorm:"bigint;not null;comment:用户id" json:"user_id"`
}

func (Like) TableName() string {
	return "likes"
}
