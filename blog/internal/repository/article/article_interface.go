package article

import (
	"blog/internal/model/entity"
)

// ArticleRepository 文章数据访问接口
type ArticleRepository interface {
	// ListArticles 分页查询文章列表（含热度排序）
	ListArticles(params *ArticleListParams) ([]entity.Article, int64, error)
	// FindByID 根据 ID 查找文章
	FindByID(id uint) (*entity.Article, error)
	// IncrementViewCount 增加文章浏览量
	IncrementViewCount(articleID uint) error
	// LikeArticle 点赞
	LikeArticle(articleID, userID uint) error
	// IsLike 检查是否点赞过
	IsLike(articleID, userID uint) (bool, error)
	// UnlikeArticle 取消点赞
	UnlikeArticle(articleID, userID uint) error
	// BatchLikeStatus 批量查询用户是否点赞
	BatchLikeStatus(userID uint, articleIDs []uint) (map[uint]bool, error)
	// GetLikeCounts 批量获取点赞数
	GetLikeCounts(articleIDs []uint) (map[uint]uint, error)
	// GetCommentCounts 批量获取评论数
	GetCommentCounts(articleIDs []uint) (map[uint]uint, error)
	// ListCategories 获取所有分类
	ListCategories() ([]entity.Category, error)
	// ListTags 获取所有标签
	ListTags() ([]entity.Tag, error)
}

// ArticleListParams 文章列表查询参数
type ArticleListParams struct {
	Page       int
	PageSize   int
	Sort       string
	CategoryID *uint
	TagIDs     []uint
	Keyword    string
	UserID     uint // 登录用户ID，用于查询点赞状态
}

// PrevNextArticle 上一篇/下一篇简要信息
type PrevNextArticle struct {
	ID    uint
	Title string
}
