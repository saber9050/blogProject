package response

// ChunkItem 切分块列表项
type ChunkItem struct {
	ID           uint   `json:"id"`
	ArticleID    uint   `json:"article_id"`
	ArticleTitle string `json:"article_title"`
	// HeadingPath 标题路径，如 "redis发布订阅和Stream > 二，Stream"
	HeadingPath      string `json:"heading_path"`
	SectionIndex     int    `json:"section_index"`
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

// ChunkStatSummary 切分统计概览（页面顶部卡片）
type ChunkStatSummary struct {
	ChunkTotal   int64 `json:"chunk_total"`
	ArticleTotal int64 `json:"article_total"`
}

// ChunkArticleStat 按文章聚合
//
// 只有「按文章」一个维度：模板概念已下线，切分参数全局固定，
// 按模板分组的结果只剩一行「默认参数」，没有信息量。
type ChunkArticleStat struct {
	ArticleID    uint   `json:"article_id"`
	ArticleTitle string `json:"article_title"`
	ChunkCount   int64  `json:"chunk_count"`
	AvgChars     int    `json:"avg_chars"`
	LastChunkAt  string `json:"last_chunk_at"`
}

// ChunkStatResponse 切分统计
type ChunkStatResponse struct {
	Summary ChunkStatSummary `json:"summary"`
	List    interface{}      `json:"list"`
}

// ChunkDeleteSkipped 批量删除时被跳过的一篇及原因
//
// 之所以「跳过」而不是「整批失败」：用户一次勾十篇，其中一篇可能正在向量化，
// 整批回滚会让他一篇都删不掉，还得自己找出是哪一篇。
type ChunkDeleteSkipped struct {
	ArticleID uint   `json:"article_id"`
	Title     string `json:"title"`
	Reason    string `json:"reason"`
}

// ChunkDeleteBatchResponse 批量删除文章向量的结果
type ChunkDeleteBatchResponse struct {
	DeletedArticles int                  `json:"deleted_articles"`
	DeletedChunks   int64                `json:"deleted_chunks"`
	Skipped         []ChunkDeleteSkipped `json:"skipped"`
}
