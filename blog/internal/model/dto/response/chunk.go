package response

// ChunkItem 切分块列表项
type ChunkItem struct {
	ID               uint   `json:"id"`
	ArticleID        uint   `json:"article_id"`
	ArticleTitle     string `json:"article_title"`
	TemplateID       uint   `json:"template_id"`
	TemplateName     string `json:"template_name"`
	ChunkIndex       int    `json:"chunk_index"`
	CharCount        int    `json:"char_count"`
	Content          string `json:"content"`
	ContentTruncated bool   `json:"content_truncated"` // true 表示还有更多，需调详情
	EmbeddingModel   string `json:"embedding_model"`
	CreatedAt        string `json:"created_at"`
}

// ChunkListResponse 切分块分页列表
type ChunkListResponse struct {
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	List     []*ChunkItem `json:"list"`
}

// ChunkStatSummary 切分统计概览（页面顶部三张卡片）
type ChunkStatSummary struct {
	ChunkTotal         int64   `json:"chunk_total"`
	ArticleTotal       int64   `json:"article_total"`
	AvgChunkPerArticle float64 `json:"avg_chunk_per_article"`
}

// ChunkTemplateStat 按模板聚合
type ChunkTemplateStat struct {
	TemplateID     uint   `json:"template_id"`
	TemplateName   string `json:"template_name"`
	TemplateStatus int8   `json:"template_status"`
	ArticleCount   int64  `json:"article_count"`
	ChunkCount     int64  `json:"chunk_count"`
	AvgChars       int    `json:"avg_chars"`
	LastChunkAt    string `json:"last_chunk_at"`
}

// ChunkArticleStat 按文章聚合
type ChunkArticleStat struct {
	ArticleID    uint   `json:"article_id"`
	ArticleTitle string `json:"article_title"`
	TemplateID   uint   `json:"template_id"`
	TemplateName string `json:"template_name"`
	ChunkCount   int64  `json:"chunk_count"`
	AvgChars     int    `json:"avg_chars"`
	LastChunkAt  string `json:"last_chunk_at"`
}

// ChunkStatResponse 切分统计
type ChunkStatResponse struct {
	GroupBy string           `json:"group_by"` // template / article
	Summary ChunkStatSummary `json:"summary"`
	List    interface{}      `json:"list"`
}
