package request

// ===== 模块 05 · 文章页 AI 助手 =====
//
// 前台接口全部需要登录（Auth()），不涉及游客。所以这些请求体里没有任何
// 身份字段——归属一律取自 token 里的 user_id，绝不接受前端传 user_id。

// CreateChatSessionRequest 新建会话。
type CreateChatSessionRequest struct {
	ArticleID uint `json:"article_id" binding:"required"`
	// ModelConfigID 0 表示「用默认」（服务端取首个启用的对话模型）。
	// 允许 0 而不是强制必填：前端首次进来就是拿着 context 返回的 default_model_id 发请求，
	// 万一那个值因为后台并发改配置失效了，用 0 走兜底比直接 400 体验好。
	ModelConfigID uint `json:"model_config_id"`
}

// UpdateChatSessionRequest 修改会话。
//
// ⚠️ 刻意没有 ArticleID：会话一旦创建就绑定文章，不允许改。
// 允许改的话「不同文章的对话不互串」就靠前端自觉了，而前端是最不可信的一层。
// 用指针区分「没传」和「传了空值」——标题清空是合法操作。
type UpdateChatSessionRequest struct {
	Title         *string `json:"title"`
	ModelConfigID *uint   `json:"model_config_id"`
}

// ChatMessageListQuery 历史消息分页（游标）。
type ChatMessageListQuery struct {
	// BeforeID 取 id 小于它的消息；0 表示从最新开始。
	// 用 id 游标而不是 offset：新消息不断插入，offset 会让翻页出现重复/漏行。
	BeforeID uint `form:"before_id"`
	Limit    int  `form:"limit"`
}

// ChatArticleCandidateQuery `/` 选择器的候选文章查询。
type ChatArticleCandidateQuery struct {
	// Keyword 为空表示「刚打出 /」，返回最近的文章供直接选。
	Keyword string `form:"keyword"`
	// ExcludeID 当前文章 id，从候选里排除——它本来就在检索范围内，不需要 pill。
	ExcludeID uint `form:"exclude_id"`
	Limit     int  `form:"limit"`
}

// SendChatMessageRequest 发消息。
type SendChatMessageRequest struct {
	// Content 用户提问原文。pill 已在前端序列化成 /文章标题，后端按普通文本处理。
	Content string `json:"content" binding:"required"`
	// ArticleIDs 本轮用 `/` 额外引用的文章（不含当前文章）。
	// ⚠️ 与 tool 参数是两回事：这里是「用户点名了什么」，服务端要逐条校验；
	// tool 那侧永远不接受模型传范围。
	ArticleIDs []uint `json:"article_ids"`
	// Stream 默认 true。用指针区分「没传」和「显式传 false」——
	// 用 bool 的话零值 false 会让「没传」变成「非流式」，前端得每次都显式传。
	Stream *bool `json:"stream"`
}

// WantsStream 默认走流式；只有显式传 false 才走一次性 JSON。
func (r *SendChatMessageRequest) WantsStream() bool {
	return r.Stream == nil || *r.Stream
}

// AdminChatSessionQuery 后台会话列表筛选。
type AdminChatSessionQuery struct {
	// User 一个输入框两种匹配：纯数字先按 user_id 精确匹配（同时也参与用户名模糊），
	// 非数字只按用户名模糊。合成一个字段，管理员不用先想「我输入的是 id 还是名字」。
	User string `form:"user"`
	// Article 文章标题模糊匹配。
	Article  string `form:"article"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}
