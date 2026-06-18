package request

// CreateCommentReq 发表评论请求
type CreateCommentReq struct {
	Content         string `json:"content" binding:"required"` // 评论内容
	ParentID        *uint  `json:"parent_id"`                  // 父评论ID（回复时使用，null为一级评论）
	ReplyToUserName string `json:"reply_to_user_name"`         // 回复目标用户名（二级评论展示用）
}

// ListCommentsReq 评论列表请求
type ListCommentsReq struct {
	Page     int `form:"page" json:"page"`           // 页码，默认1
	PageSize int `form:"page_size" json:"page_size"` // 每页条数，默认20
}

// ListRepliesReq 回复列表请求（二级评论）
type ListRepliesReq struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}
