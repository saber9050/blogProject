package request

// VectorizeRequest 提交文章向量化
type VectorizeRequest struct {
	// TemplateID 用哪个切分模板。必填——不同模板切出来的块完全不同，
	// 不给默认值（比如「取第一个启用的」）会让用户不知道这次是怎么切的。
	TemplateID uint `json:"template_id" binding:"required"`
}
