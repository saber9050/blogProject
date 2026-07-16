package request

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Title    string `json:"title" binding:"required,max=200"`
	TypeID   uint   `json:"type_id" binding:"required"`
	TagIDs   []uint `json:"tag_ids"`
	CoverURL string `json:"cover_url"`
	Summary  string `json:"summary" binding:"max=255"`
	Content  string `json:"content" binding:"required"`
	Status   int8   `json:"status" binding:"oneof=0 1"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	Title    string `json:"title" binding:"max=200"`
	TypeID   uint   `json:"type_id"`
	TagIDs   []uint `json:"tag_ids"`
	CoverURL string `json:"cover_url"`
	Summary  string `json:"summary" binding:"max=255"`
	Content  string `json:"content"`
	Status   int8   `json:"status" binding:"oneof=0 1"`
}

// GenerateSummaryRequest 一键生成摘要请求
type GenerateSummaryRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// TransferCategoryReq 一键转移分类请求
type TransferCategoryReq struct {
	FromTypeID uint `json:"from_type_id" binding:"required"`
	ToTypeID   uint `json:"to_type_id" binding:"required"`
}

// ArticleListQuery 文章列表查询参数
type ArticleListQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Sort       string `form:"sort"`
	CategoryID uint   `form:"category_id"`
	TagIDs     string `form:"tag_ids"`
	Keyword    string `form:"keyword"`
	Status     int    `form:"status"`
}
