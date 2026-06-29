package article

import (
	"blog/internal/model/entity"
)

// ArticleRepository 文章数据访问接口
type ArticleRepository interface {
	// FindByID 通过 ID 查找文章
	FindByID(id uint) (*entity.Article, error)
	// ListPublic 前台获取已发布的文章列表
	ListPublic(page, pageSize int, sort string, categoryID uint, tagIDs []uint, keyword string) ([]*entity.Article, int64, error)
	// ListAdmin 后台获取文章列表（可查看所有状态）
	ListAdmin(page, pageSize int, status *int, categoryID uint, tagIDs []uint, keyword string) ([]*entity.Article, int64, error)
	// Create 创建文章
	Create(article *entity.Article) error
	// UpdateFields 更新文章的指定字段
	UpdateFields(id uint, fields map[string]interface{}) error
	// Delete 删除文章（软删除）
	Delete(id uint) error
	// IncrementViews 增加浏览量
	IncrementViews(id uint) error
	// GetTagsByArticleID 获取文章关联的标签
	GetTagsByArticleID(articleID uint) ([]*entity.Tag, error)
	// SetArticleTags 设置文章标签（先删后插）
	SetArticleTags(articleID uint, tagIDs []uint) error
	// FindLiked 查询用户是否已点赞
	FindLiked(articleID, userID uint) (bool, error)
	// CreateLike 创建点赞
	CreateLike(articleID, userID uint) error
	// DeleteLike 取消点赞
	DeleteLike(articleID, userID uint) error
	// IncrementLikeCount 增加点赞计数
	IncrementLikeCount(articleID uint) error
	// DecrementLikeCount 减少点赞计数
	DecrementLikeCount(articleID uint) error
	// IncrementCommentCount 增加评论计数
	IncrementCommentCount(articleID uint) error
	// DecrementCommentCount 减少评论计数
	DecrementCommentCount(articleID uint) error
}
