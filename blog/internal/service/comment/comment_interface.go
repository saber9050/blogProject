package comment

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
)

// CommentService 评论服务接口
type CommentService interface {
	// ListComments 获取文章一级评论列表
	ListComments(articleID uint, page, pageSize int) (*response.CommentListResponse, error)
	// ListReplies 获取评论的回复列表
	ListReplies(parentID uint, page, pageSize int) (*response.CommentListResponse, error)
	// CreateComment 发表评论
	CreateComment(articleID, userID uint, req *request.CreateCommentReq) (*response.CommentItem, error)
	// DeleteComment 删除评论
	DeleteComment(commentID, userID uint, roleID int8) error
	// SetArticleRepo 设置文章仓库依赖（避免循环依赖）
	SetArticleRepo(articleRepo interface {
		IncrementCommentCount(articleID uint) error
		DecrementCommentCount(articleID uint) error
	})
	// SetMinioClient 设置MinIO客户端依赖（用于生成完整头像URL）
	SetMinioClient(minioClient interface {
		GetFileURL(fileKey string) string
	})
	// SetUserRepo 设置用户仓库依赖（用于获取用户信息）
	SetUserRepo(userRepo interface {
		FindByID(id uint) (*entity.User, error)
	})
}
