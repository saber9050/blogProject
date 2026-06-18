package request

// ListArticlesReq 文章列表请求
type ListArticlesReq struct {
	Page       int    `form:"page" json:"page"`               // 页码，默认1
	PageSize   int    `form:"page_size" json:"page_size"`     // 每页条数，默认10，最大20
	Sort       string `form:"sort" json:"sort"`               // 排序：latest / popular
	CategoryID *uint  `form:"category_id" json:"category_id"` // 分类筛选
	TagIDs     string `form:"tag_ids" json:"tag_ids"`         // 标签筛选，逗号分隔 "1,2,3"
	Keyword    string `form:"keyword" json:"keyword"`         // 标题搜索关键字
}

// GetArticleDetailReq 文章详情请求（可选登录态）
type GetArticleDetailReq struct {
	ID uint `uri:"id" json:"id"`
}

// LikeArticleReq 点赞/取消点赞（参数从路径获取）
type LikeArticleReq struct {
	ArticleID uint `uri:"id" json:"article_id"`
}

// BatchLikeStatusReq 批量查询点赞状态
type BatchLikeStatusReq struct {
	IDs string `form:"ids" json:"ids"` // 逗号分隔的文章ID列表
}
