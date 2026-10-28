package response

// ChunkTemplateResponse 切分模板响应
type ChunkTemplateResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ChunkSize    int    `json:"chunk_size"`
	ChunkOverlap int    `json:"chunk_overlap"`
	MinChunkSize int    `json:"min_chunk_size"`
	StripHTML    bool   `json:"strip_html"`
	Status       int8   `json:"status"`

	// UsedArticleCount / ChunkCount 是派生字段，不进库，查询时聚合出来。
	// 前端靠 UsedArticleCount 判断参数能不能改（>0 时置灰输入框）。
	UsedArticleCount int64 `json:"used_article_count"`
	ChunkCount       int64 `json:"chunk_count"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
