package response

// VectorSchemaResponse 向量表现状
//
// 用途有三个：① 切分管理页展示维度信息；② 「AI 模型」新增向量模型时
// 判断维度输入框是否只读；③ 排查「为什么报维度不一致」。
type VectorSchemaResponse struct {
	TableExists  bool  `json:"table_exists"` // article_chunks 是否已创建
	Dimension    int   `json:"dimension"`    // 表的实际列维度，表不存在时为 0
	ChunkTotal   int64 `json:"chunk_total"`
	ArticleTotal int64 `json:"article_total"`
	// ExtensionReady pgvector 扩展是否可用。叫 extension 而不是 index：
	// 本模块不建 HNSW 索引，说 index 会让人以为索引已经建好了。
	ExtensionReady bool               `json:"extension_ready"`
	Model          *VectorSchemaModel `json:"model"` // 未配置向量模型时为 null
	Match          bool               `json:"match"` // 表维度与模型维度是否一致
	Message        string             `json:"message"`
}

// VectorSchemaModel 当前向量模型的摘要信息（不含密钥）
type VectorSchemaModel struct {
	ID        uint   `json:"id"`
	Model     string `json:"model"`
	Dimension int    `json:"dimension"`
	Status    int8   `json:"status"`
}

// VectorResetResponse 清空并重建向量表的结果
//
// 注意这里没有 dimension：清空后**不重建表**，而是把它留给「新增向量模型」那一步。
// 原因见 service 里 Reset 的注释——如果这里按当前模型维度重建，
// 「先清空再换 768 维模型」就会陷入死循环：表被旧维度重建，新模型又报维度冲突。
type VectorResetResponse struct {
	DeletedChunks int64 `json:"deleted_chunks"` // 删前统计，便于核对
	ResetArticles int64 `json:"reset_articles"` // 被重置状态的文章数
}
