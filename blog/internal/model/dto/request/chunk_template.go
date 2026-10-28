package request

// CreateChunkTemplateRequest 新建切分模板请求
type CreateChunkTemplateRequest struct {
	Name         string `json:"name" binding:"required,max=64"`
	Description  string `json:"description" binding:"max=255"`
	ChunkSize    int    `json:"chunk_size" binding:"required"`
	ChunkOverlap int    `json:"chunk_overlap"`
	MinChunkSize int    `json:"min_chunk_size"`
	// StripHTML 指针：缺省 true（当前正文是 HTML），显式传 false 才不剥标签
	StripHTML *bool `json:"strip_html"`
	Status    *int8 `json:"status" binding:"omitempty,oneof=0 1"`
}

// UpdateChunkTemplateRequest 更新切分模板请求
//
// 三个数值参数用指针：要区分「没传」和「传了 0」。
// 用值类型的话，前端只想改名称时，请求里省略 chunk_size 会被当成 0，
// 然后被校验规则拒掉，用户会莫名其妙。
type UpdateChunkTemplateRequest struct {
	Name         string `json:"name" binding:"omitempty,max=64"`
	Description  string `json:"description" binding:"omitempty,max=255"`
	ChunkSize    *int   `json:"chunk_size"`
	ChunkOverlap *int   `json:"chunk_overlap"`
	MinChunkSize *int   `json:"min_chunk_size"`
	StripHTML    *bool  `json:"strip_html"`
	Status       *int8  `json:"status" binding:"omitempty,oneof=0 1"`
}

// ResetVectorRequest 清空并重建向量表请求。
//
// Confirm 必须是字面量 "RESET"。这不是形式主义——该操作会永久删除全部向量块，
// 而它通常是在改配置时顺手点的，一个误触的代价是几十篇文章要重新向量化。
type ResetVectorRequest struct {
	Confirm string `json:"confirm"`
}
