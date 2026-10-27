package entity

// Like 文章点赞实体
type Like struct {
	BaseEntity
	ArticleID uint `gorm:"type:bigint;not null;index;comment:文章id" json:"article_id"`
	UserID    uint `gorm:"type:bigint;not null;index;comment:用户id" json:"user_id"`
}

func (Like) TableName() string {
	return "likes"
}
