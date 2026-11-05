package response

import "blog/internal/model/entity"

// ===== 模块 05 · 文章页 AI 助手 =====

// ChatModelItem 可选的对话模型。只给 id + model：
// base_url / api_key 属于后台配置，前台没有理由知道，泄露了就是白给攻击面。
type ChatModelItem struct {
	ID    uint   `json:"id"`
	Model string `json:"model"`
}

// ChatSessionItem 会话列表项。
type ChatSessionItem struct {
	ID            uint    `json:"id"`
	Title         string  `json:"title"`
	ModelConfigID uint    `json:"model_config_id"`
	Model         string  `json:"model"`
	MessageCount  int     `json:"message_count"`
	LastMessageAt *string `json:"last_message_at"`
	CreatedAt     string  `json:"created_at"`
}

// ChatContextResponse 面板初始化：能不能聊、可用模型、该文章下的会话列表。
type ChatContextResponse struct {
	// Enabled 为 false 时前端置灰输入框并展示 Reason。原文「未登录」不走这个字段——
	// 未登录直接被 Auth() 挡成 401，前端弹「请先登录」。
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`

	ArticleID    uint   `json:"article_id"`
	ArticleTitle string `json:"article_title"`

	// RAGReady 本文能否走向量检索（表存在 + 向量模型可用 + vector_status=3）。
	// false 不报错，回答会自动降级为关键词检索 / 全文，前端最多提一句「资料可能不完整」。
	RAGReady bool `json:"rag_ready"`
	// MaxRefArticles 单次提问涉及的文章总篇数上限（含当前文章）。
	// 前端据此算 pill 上限（8-1=7），不要在前端写死数字。
	MaxRefArticles int `json:"max_ref_articles"`

	Models         []*ChatModelItem   `json:"models"`
	DefaultModelID uint               `json:"default_model_id"`
	LastSessionID  uint               `json:"last_session_id"`
	Sessions       []*ChatSessionItem `json:"sessions"`
}

// ChatMessageItem 一条消息。
type ChatMessageItem struct {
	ID   uint   `json:"id"`
	Role string `json:"role"`
	// Content 用户消息是「用户看到的原文」（pill 序列化成 /文章标题）；
	// 助手消息是完整回答；拒答时是后端固定文案。
	Content string `json:"content"`
	Model   string `json:"model"`
	// Refs 仅 assistant 有值，null 表示没检索或走了全文降级。
	// 块内容不返回：正文会被重新向量化整体替换，冗余进消息只会让数据翻倍且可能过期。
	Refs []entity.ChatRefChunk `json:"refs"`
	// ArticleRefs 仅 user 有值：本轮显式引用的其他文章。
	ArticleRefs []entity.ChatArticleRef `json:"article_refs"`
	Interrupted bool                    `json:"interrupted"`
	Refused     bool                    `json:"refused"`
	Error       string                  `json:"error"`
	CreatedAt   string                  `json:"created_at"`
}

// ChatMessageListResponse 历史消息分页结果。
type ChatMessageListResponse struct {
	List []*ChatMessageItem `json:"list"`
	// HasMore 是否还有更早的消息（用于「加载更早」按钮）。
	HasMore bool `json:"has_more"`
}

// ChatArticleCandidateItem `/` 选择器的候选项。只返回 id + title：
// 候选浮层只显示标题，多给字段等于把「文章列表接口」的演进绑到聊天上。
type ChatArticleCandidateItem struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}
