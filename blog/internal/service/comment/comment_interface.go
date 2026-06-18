package comment

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
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
}
