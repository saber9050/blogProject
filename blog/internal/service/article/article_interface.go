package article

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"mime/multipart"
)

// ArticleService 文章服务接口
type ArticleService interface {
	// ListPublic 前台获取文章列表
	ListPublic(page, pageSize int, sort string, categoryID uint, tagIDs []uint, keyword string, userID uint) (*response.ArticleListResponse, error)
	// GetDetail 获取文章详情
	GetDetail(id uint, userID uint) (*response.ArticleDetailResponse, error)
	// LikeArticle 点赞文章
	LikeArticle(articleID, userID uint) error
	// UnlikeArticle 取消点赞
	UnlikeArticle(articleID, userID uint) error
	// AdminList 后台获取文章列表
	AdminList(page, pageSize int, status *int, categoryID uint, tagIDs []uint, keyword string) (*response.ArticleListResponse, error)
	// AdminCreate 后台创建文章
	AdminCreate(req *request.CreateArticleRequest, userID uint) (*response.CreateArticleResponse, error)
	// AdminUpdate 后台更新文章
	AdminUpdate(id uint, req *request.UpdateArticleRequest) error
	// AdminDelete 后台删除文章
	AdminDelete(id uint) error
	// UploadImage 上传图片（返回完整路径）
	UploadImage(fileHeader *multipart.FileHeader) (string, error)
	// GetStats 获取已发布文章的统计数据
	GetStats() (*response.ArticleStatsResponse, error)
	// TransferCategory 将一个分类下的所有文章转移到另一个分类
	TransferCategory(fromTypeID, toTypeID uint) (int64, error)
	// GetRandomArticleID 随机获取一篇已发布文章的 ID
	GetRandomArticleID() (uint, error)
}
