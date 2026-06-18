package response

import "time"

// CommentItem 评论条目
type CommentItem struct {
	ID            uint      `json:"id"`
	Content       string    `json:"content"`
	UserID        uint      `json:"user_id"`
	UserName      string    `json:"user_name"`
	AvatarURL     string    `json:"avatar_url"`
	ParentID      *uint     `json:"parent_id"`
	ReplyToName   string    `json:"reply_to_name"`
	IsDeleted     bool      `json:"is_deleted"`
	CreatedAt     time.Time `json:"created_at"`
	ChildrenTotal int       `json:"children_total"` // 子评论总数
}

// CommentListResponse 评论列表响应
type CommentListResponse struct {
	List     []CommentItem `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}
