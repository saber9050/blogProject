package response

// CategoryInfo 分类信息
type CategoryInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// TagInfo 标签信息
type TagInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// ArticleItem 文章列表项
type ArticleItem struct {
	ID           uint          `json:"id"`
	Title        string        `json:"title"`
	Summary      string        `json:"summary"`
	CoverURL     string        `json:"cover_url"`
	Status       int8          `json:"status"`
	Views        uint          `json:"views"`
	LikeCount    uint          `json:"like_count"`
	CommentCount uint          `json:"comment_count"`
	IsLiked      bool          `json:"is_liked"`
	AuthorName   string        `json:"author_name"`
	Category     *CategoryInfo `json:"category"`
	Tags         []*TagInfo    `json:"tags"`
	CreatedAt    string        `json:"created_at"`
}

// ArticleListResponse 文章列表响应
type ArticleListResponse struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// AdminArticleItem 后台文章条目：在公共结构上追加管理侧字段。
//
// 刻意用独立结构而不是往 ArticleItem 里加字段：ArticleItem 被 ListPublic 和
// GetDetail 共用，加进去前台文章列表的 JSON 里也会带出 vector_status 等内部状态。
// 虽然不算敏感信息，但没必要暴露，也增加响应体积。
type AdminArticleItem struct {
	ArticleItem
	VectorStatus       int8   `json:"vector_status"`
	VectorTemplateID   uint   `json:"vector_template_id"`
	VectorTemplateName string `json:"vector_template_name"`
	VectorChunkCount   int    `json:"vector_chunk_count"`
	VectorError        string `json:"vector_error"`
	VectorizedAt       string `json:"vectorized_at"`
	UpdatedAt          string `json:"updated_at"`
}

// AdminArticleListResponse 后台文章列表响应
type AdminArticleListResponse struct {
	List     []*AdminArticleItem `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// ArticleDetailResponse 文章详情响应
type ArticleDetailResponse struct {
	ArticleItem
	Content string `json:"content"`
}

// CreateArticleResponse 创建文章响应
type CreateArticleResponse struct {
	ID uint `json:"id"`
}

// ArticleStatsResponse 文章统计响应
type ArticleStatsResponse struct {
	ArticleCount int64 `json:"article_count"`
	TotalViews   int64 `json:"total_views"`
	TotalLikes   int64 `json:"total_likes"`
}
