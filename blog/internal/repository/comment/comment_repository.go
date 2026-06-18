package comment

import (
	"blog/internal/model/entity"

	"gorm.io/gorm"
)

type commentRepository struct {
	db *gorm.DB
}

// NewCommentRepository 创建评论仓储实例
func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

// commentWithUser 评论 + 用户信息的查询结果
type commentWithUser struct {
	entity.Comment
	UserName  string `json:"user_name"`
	AvatarURL string `json:"avatar_url"`
}

// ListComments 分页查询一级评论
func (r *commentRepository) ListComments(articleID uint, page, pageSize int) ([]entity.Comment, int64, error) {
	base := r.db.Model(&entity.Comment{}).
		Where("article_id = ?", articleID).
		Where("parent_id IS NULL OR parent_id = 0")

	// 总数
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	offset := (page - 1) * pageSize
	var rows []commentWithUser
	err := r.db.Table("comments").
		Select("comments.*, users.user_name, users.avatar_url").
		Joins("LEFT JOIN users ON users.id = comments.user_id").
		Where("comments.article_id = ?", articleID).
		Where("comments.parent_id IS NULL OR comments.parent_id = 0").
		Order("comments.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	comments := make([]entity.Comment, len(rows))
	for i, row := range rows {
		c := row.Comment
		c.UserName = row.UserName
		c.AvatarURL = row.AvatarURL
		comments[i] = c
	}

	return comments, total, nil
}

// ListReplies 分页查询二级评论
func (r *commentRepository) ListReplies(parentID uint, page, pageSize int) ([]entity.Comment, int64, error) {
	base := r.db.Model(&entity.Comment{}).Where("parent_id = ?", parentID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	offset := (page - 1) * pageSize
	var rows []commentWithUser
	err := r.db.Table("comments").
		Select("comments.*, users.user_name, users.avatar_url").
		Joins("LEFT JOIN users ON users.id = comments.user_id").
		Where("comments.parent_id = ?", parentID).
		Order("comments.created_at ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	comments := make([]entity.Comment, len(rows))
	for i, row := range rows {
		c := row.Comment
		c.UserName = row.UserName
		c.AvatarURL = row.AvatarURL
		comments[i] = c
	}

	return comments, total, nil
}

// CreateComment 创建评论
func (r *commentRepository) CreateComment(comment *entity.Comment) error {
	return r.db.Create(comment).Error
}

// DeleteComment 软删除评论
func (r *commentRepository) DeleteComment(id uint) error {
	return r.db.Delete(&entity.Comment{}, id).Error
}

// GetCommentByID 根据ID查询评论
func (r *commentRepository) GetCommentByID(id uint) (*entity.Comment, error) {
	var comment entity.Comment
	err := r.db.First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetCommentCount 获取文章评论总数
func (r *commentRepository) GetCommentCount(articleID uint) (int64, error) {
	var count int64
	err := r.db.Model(&entity.Comment{}).
		Where("article_id = ?", articleID).
		Count(&count).Error
	return count, err
}

// GetChildrenCounts 批量获取评论子评论数
func (r *commentRepository) GetChildrenCounts(parentIDs []uint) (map[uint]int, error) {
	if len(parentIDs) == 0 {
		return map[uint]int{}, nil
	}

	type row struct {
		ParentID uint `gorm:"column:parent_id"`
		Count    int  `gorm:"column:cnt"`
	}

	var rows []row
	err := r.db.Model(&entity.Comment{}).
		Select("parent_id, COUNT(*) as cnt").
		Where("parent_id IN ?", parentIDs).
		Group("parent_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint]int, len(rows))
	for _, row := range rows {
		result[row.ParentID] = row.Count
	}
	return result, nil
}
