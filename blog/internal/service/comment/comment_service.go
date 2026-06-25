package comment

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	commentRepo "blog/internal/repository/comment"
	"blog/pkg/errors"
	"fmt"

	"gorm.io/gorm"
)

type commentService struct {
	commentRepo commentRepo.CommentRepository
	articleRepo interface {
		IncrementCommentCount(articleID uint) error
		DecrementCommentCount(articleID uint) error
	}
}

// NewCommentService 创建评论服务实例
func NewCommentService(repo commentRepo.CommentRepository) CommentService {
	return &commentService{commentRepo: repo}
}

// SetArticleRepo 设置文章仓库依赖（避免循环依赖）
func (s *commentService) SetArticleRepo(articleRepo interface {
	IncrementCommentCount(articleID uint) error
	DecrementCommentCount(articleID uint) error
}) {
	s.articleRepo = articleRepo
}

// ListComments 获取一级评论列表
func (s *commentService) ListComments(articleID uint, page, pageSize int) (*response.CommentListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}

	comments, total, err := s.commentRepo.ListComments(articleID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询评论列表失败: %w", err)
	}
	if len(comments) == 0 {
		return &response.CommentListResponse{
			List:     []response.CommentItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// 批量查询子评论数
	parentIDs := make([]uint, len(comments))
	for i, c := range comments {
		parentIDs[i] = c.ID
	}
	childrenCounts, err := s.commentRepo.GetChildrenCounts(parentIDs)
	if err != nil {
		return nil, fmt.Errorf("查询子评论数失败: %w", err)
	}

	list := make([]response.CommentItem, len(comments))
	for i, c := range comments {
		list[i] = s.entityToItem(&c, childrenCounts[c.ID])
	}

	return &response.CommentListResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListReplies 获取回复列表
func (s *commentService) ListReplies(parentID uint, page, pageSize int) (*response.CommentListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 30 {
		pageSize = 30
	}

	comments, total, err := s.commentRepo.ListReplies(parentID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询回复列表失败: %w", err)
	}
	if len(comments) == 0 {
		return &response.CommentListResponse{
			List:     []response.CommentItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	list := make([]response.CommentItem, len(comments))
	for i, c := range comments {
		list[i] = s.entityToItem(&c, 0)
	}

	return &response.CommentListResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// CreateComment 发表评论
func (s *commentService) CreateComment(articleID, userID uint, req *request.CreateCommentReq) (*response.CommentItem, error) {
	if req.Content == "" {
		return nil, errors.NewDefault(errors.CodeBadRequest)
	}

	// 如果是回复，验证父评论存在且属于该文章
	if req.ParentID != nil && *req.ParentID != 0 {
		parent, err := s.commentRepo.GetCommentByID(*req.ParentID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.NewDefault(errors.CodeNotFound)
			}
			return nil, fmt.Errorf("查询父评论失败: %w", err)
		}
		if parent.ArticleID != articleID {
			return nil, errors.New(errors.CodeBadRequest, "父评论不属于该文章")
		}
	}

	comment := &entity.Comment{
		ArticleID:   articleID,
		UserID:      userID,
		ParentID:    req.ParentID,
		Content:     req.Content,
		ReplyToName: req.ReplyToUserName,
	}

	if err := s.commentRepo.CreateComment(comment); err != nil {
		return nil, fmt.Errorf("发表评论失败: %w", err)
	}

	// 增加文章评论计数
	if s.articleRepo != nil {
		if err := s.articleRepo.IncrementCommentCount(articleID); err != nil {
			fmt.Printf("增加文章评论计数失败: %v\n", err)
		}
	}

	// 构建返回项（使用 GORM 自动填充的 ID 和 CreatedAt）
	item := &response.CommentItem{
		ID:          comment.ID,
		Content:     comment.Content,
		UserID:      userID,
		ParentID:    comment.ParentID,
		ReplyToName: comment.ReplyToName,
		CreatedAt:   comment.CreatedAt,
	}

	return item, nil
}

// DeleteComment 删除评论
func (s *commentService) DeleteComment(commentID, userID uint, roleID int8) error {
	comment, err := s.commentRepo.GetCommentByID(commentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewDefault(errors.CodeNotFound)
		}
		return fmt.Errorf("查询评论失败: %w", err)
	}

	// 权限检查：只有评论作者或管理员可以删除
	if comment.UserID != userID && roleID != 1 {
		return errors.NewDefault(errors.CodeForbidden)
	}

	if err := s.commentRepo.DeleteComment(commentID); err != nil {
		return fmt.Errorf("删除评论失败: %w", err)
	}

	// 减少文章评论计数
	if s.articleRepo != nil {
		if err := s.articleRepo.DecrementCommentCount(comment.ArticleID); err != nil {
			fmt.Printf("减少文章评论计数失败: %v\n", err)
		}
	}

	return nil
}

// entityToItem 将实体转为响应DTO
func (s *commentService) entityToItem(c *entity.Comment, childrenTotal int) response.CommentItem {
	isDeleted := c.DeletedAt.Valid
	item := response.CommentItem{
		ID:            c.ID,
		Content:       c.Content,
		UserID:        c.UserID,
		UserName:      c.UserName,
		AvatarURL:     c.AvatarURL,
		ParentID:      c.ParentID,
		ReplyToName:   c.ReplyToName,
		IsDeleted:     isDeleted,
		CreatedAt:     c.CreatedAt,
		ChildrenTotal: childrenTotal,
	}
	// 已删除评论不展示内容
	if isDeleted {
		item.Content = ""
	}
	return item
}
