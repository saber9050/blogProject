package response

import "time"

// TagInfo 标签简要信息
type TagInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// CategoryInfo 分类简要信息
type CategoryInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// ArticleListItem 文章列表项
type ArticleListItem struct {
	ID           uint         `json:"id"`
	Title        string       `json:"title"`
	Summary      string       `json:"summary"`
	CoverURL     string       `json:"cover_url"`
	AuthorID     uint         `json:"author_id"`
	AuthorName   string       `json:"author_name"`
	AuthorAvatar string       `json:"author_avatar"`
	Category     CategoryInfo `json:"category"`
	Tags         []TagInfo    `json:"tags"`
	ViewCount    uint         `json:"view_count"`
	LikeCount    uint         `json:"like_count"`
	CommentCount uint         `json:"comment_count"`
	IsLiked      bool         `json:"is_liked"`
	CreatedAt    time.Time    `json:"created_at"`
}

// ArticleListResponse 文章列表响应
type ArticleListResponse struct {
	List     []ArticleListItem `json:"list"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// ArticleDetail 文章详情
type ArticleDetail struct {
	ArticleListItem
	Content     string          `json:"content"`
	PrevArticle *ArticleNavLink `json:"prev_article,omitempty"`
	NextArticle *ArticleNavLink `json:"next_article,omitempty"`
}

// ArticleNavLink 上一篇/下一篇导航
type ArticleNavLink struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

// LikeStatusMap 批量点赞状态
type LikeStatusMap struct {
	LikedMap map[uint]bool `json:"liked_map"`
}
