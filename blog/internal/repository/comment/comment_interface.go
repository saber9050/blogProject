package comment

import (
	"blog/internal/model/entity"
)

// CommentRepository 评论仓储接口
type CommentRepository interface {
	// ListComments 分页查询一级评论（带用户信息）
	ListComments(articleID uint, page, pageSize int) ([]entity.Comment, int64, error)
	// ListReplies 分页查询二级评论（带用户信息）
	ListReplies(parentID uint, page, pageSize int) ([]entity.Comment, int64, error)
	// CreateComment 创建评论
	CreateComment(comment *entity.Comment) error
	// DeleteComment 软删除评论
	DeleteComment(id uint) error
	// DeleteByArticleID 软删除某文章的所有评论
	DeleteByArticleID(articleID uint) error
	// GetCommentByID 根据ID查询评论（用于权限校验）
	GetCommentByID(id uint) (*entity.Comment, error)
	// GetCommentCount 获取文章评论总数
	GetCommentCount(articleID uint) (int64, error)
	// GetChildrenCounts 批量获取评论子评论数
	GetChildrenCounts(parentIDs []uint) (map[uint]int, error)
	// ListAllComments 分页查询所有未删除评论（后台管理，带文章标题和用户名）
	ListAllComments(page, pageSize int) ([]AdminCommentRow, int64, error)
	// BatchDeleteComments 批量软删除评论
	BatchDeleteComments(ids []uint) (int64, error)
	// GetArticleIDsByCommentIDs 批量查询评论所属文章ID
	GetArticleIDsByCommentIDs(ids []uint) ([]uint, error)
}
