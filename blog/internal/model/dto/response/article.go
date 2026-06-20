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

// ArticleDetailResponse 文章详情响应
type ArticleDetailResponse struct {
	ArticleItem
	Content string `json:"content"`
}

// CreateArticleResponse 创建文章响应
type CreateArticleResponse struct {
	ID uint `json:"id"`
}
