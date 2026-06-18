package article

import (
	"blog/internal/model/dto/response"
	articleRepo "blog/internal/repository/article"
)

// ArticleService 文章服务接口
type ArticleService interface {
	// ListArticles 文章列表
	ListArticles(params *articleRepo.ArticleListParams) (*response.ArticleListResponse, error)
	// GetArticleDetail 文章详情
	GetArticleDetail(articleID, userID uint) (*response.ArticleDetail, error)
	// LikeArticle 点赞
	LikeArticle(articleID, userID uint) error
	// UnlikeArticle 取消点赞
	UnlikeArticle(articleID, userID uint) error
	// BatchLikeStatus 批量查询点赞状态
	BatchLikeStatus(userID uint, articleIDs []uint) (*response.LikeStatusMap, error)
	// IncrementViewCount 增加文章浏览量
	IncrementViewCount(articleID uint) error
	// ListCategories 获取所有分类
	ListCategories() ([]response.CategoryInfo, error)
	// ListTags 获取所有标签
	ListTags() ([]response.TagInfo, error)
}
