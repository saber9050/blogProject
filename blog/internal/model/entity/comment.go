package entity

// Comment 评论实体
type Comment struct {
	BaseEntity
	ArticleID uint   `gorm:"type:bigint;not null;index;comment:所属文章ID" json:"article_id"`
	UserID    uint   `gorm:"type:bigint;not null;index;comment:评论用户ID（登录用户）" json:"user_id"`
	Content   string `gorm:"type:text;not null;comment:评论内容" json:"content"`
}

func (Comment) TableName() string {
	return "comments"
}
