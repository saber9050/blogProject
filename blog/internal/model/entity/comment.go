package entity

import "gorm.io/gorm"

// Comment 评论实体
type Comment struct {
	BaseEntity
	ArticleID uint `gorm:"type:bigint;not null;index;comment:所属文章ID" json:"article_id"`
	UserID    uint `gorm:"type:bigint;not null;index;comment:评论用户ID（登录用户）" json:"user_id"`
	// 为空或0是一级评论,否则是二级评论，即回复某人的评论
	ParentID    *uint          `gorm:"type:bigint;default:0;index;comment:父评论ID" json:"parent_id"`
	Content     string         `gorm:"type:text;not null;comment:评论内容" json:"content"`
	ReplyToName string         `gorm:"-" json:"reply_to_name,omitempty"` // 回复目标用户名（非DB字段，查询时填充）
	UserName    string         `gorm:"-" json:"user_name,omitempty"`     // 评论用户名（非DB字段，查询时填充）
	AvatarURL   string         `gorm:"-" json:"avatar_url,omitempty"`    // 评论用户头像（非DB字段，查询时填充）
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Comment) TableName() string {
	return "comments"
}
