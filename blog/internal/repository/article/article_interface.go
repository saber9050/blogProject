package article

import (
	"blog/internal/model/entity"
)

// ArticleRepository 文章数据访问接口
type ArticleRepository interface {
	// FindByID 通过 ID 查找文章
	FindByID(id uint) (*entity.Article, error)
	// List 获取文章列表
	List(page, pageSize int, status *int, sort string, categoryID uint, tagIDs []uint, keyword string) ([]*entity.Article, int64, error)
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
	// CountLikes 获取文章点赞数
	CountLikes(articleID uint) (int64, error)
	// CountComments 获取文章评论数（未删除的）
	CountComments(articleID uint) (int64, error)
	// GetStats 获取已发布文章的统计数据（文章数、总阅读量、总点赞量）
	GetStats() (articleCount, totalViews, totalLikes int64, err error)

	// FindArticleImages 获取文章关联的所有图片
	FindArticleImages(articleID uint) ([]entity.ArticleImage, error)
	// CreateArticleImages 批量创建文章图片引用
	CreateArticleImages(images []entity.ArticleImage) error
	// DeleteArticleImages 批量删除文章图片引用（按 ID）
	DeleteArticleImages(ids []uint) error
	// DeleteArticleImagesByArticleID 删除文章的所有图片引用
	DeleteArticleImagesByArticleID(articleID uint) error
	// TransferCategory 将一个分类下的所有文章转移到另一个分类，返回受影响行数
	TransferCategory(fromTypeID, toTypeID uint) (int64, error)

	// GetRandomID 随机获取一篇已发布文章的 ID
	GetRandomID() (uint, error)
}
