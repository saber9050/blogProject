package response

// VectorizeSubmitResponse 提交向量化的响应（立即返回，不等处理完成）
type VectorizeSubmitResponse struct {
	ArticleID    uint `json:"article_id"`
	TemplateID   uint `json:"template_id"`
	VectorStatus int8 `json:"vector_status"` // 恒为 1（排队中）
}

// VectorStatusResponse 向量化状态（轮询用，只查 articles 一行）
type VectorStatusResponse struct {
	ArticleID        uint   `json:"article_id"`
	VectorStatus     int8   `json:"vector_status"`
	VectorTemplateID uint   `json:"vector_template_id"`
	VectorChunkCount int    `json:"vector_chunk_count"`
	VectorError      string `json:"vector_error"`
	VectorizedAt     string `json:"vectorized_at"`
	UpdatedAt        string `json:"updated_at"`
}
