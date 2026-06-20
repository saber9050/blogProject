package article

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	minioPkg "blog/pkg/minio"
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
	// UploadImage 上传图片（返回 URL）
	UploadImage(fileHeader *multipart.FileHeader) (string, error)
	// SetDeps 设置依赖（用户、分类、标签仓库和 MinIO）
	SetDeps(userRepo interface {
		FindByID(id uint) (*entity.User, error)
	}, categoryRepo interface {
		FindByID(id uint) (*entity.Category, error)
	}, tagRepo interface {
		FindByIDs(ids []uint) ([]*entity.Tag, error)
	}, minio *minioPkg.Client)
}
