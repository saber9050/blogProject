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
	// Steps 仅 assistant 有值：本次回答的检索过程（查了几次、用什么词、命中几段、耗时多久）。
	// nil 表示这次没有发生过检索（模型直接回答），前端要能区分「没查」和「查了没命中」。
	Steps       []entity.ChatStep `json:"steps"`
	Interrupted bool              `json:"interrupted"`
	Refused     bool              `json:"refused"`
	// NoRetrieval 零检索，Content 是后端固定提示
	NoRetrieval bool   `json:"no_retrieval"`
	Error       string `json:"error"`
	CreatedAt   string `json:"created_at"`
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

// SendChatMessageResponse 非流式发消息的返回（`stream:false` 时用）。
//
// 流式模式下这些字段分散在 SSE 的 meta / tool / delta / done 事件里，
// 两者字段语义完全一致，只是投递方式不同。
type SendChatMessageResponse struct {
	UserMessageID      uint   `json:"user_message_id"`
	AssistantMessageID uint   `json:"assistant_message_id"`
	Content            string `json:"content"`
	Model              string `json:"model"`
	ModelFallback      bool   `json:"model_fallback"`
	// Refs 本次回答用到的切分块；nil 表示没检索或走了全文降级。
	Refs []entity.ChatRefChunk `json:"refs"`
	// ArticleRefs 本轮实际生效的引用文章（回显给前端拼 pill）。
	ArticleRefs []entity.ChatArticleRef `json:"article_refs"`
	// SearchedArticles 实际检索过的文章 id。比 ArticleRefs 少是正常的：
	// ArticleRefs 是「允许检索的范围」，这个才是「真的查了哪几篇」。
	SearchedArticles []uint `json:"searched_articles"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Refused          bool   `json:"refused"`
	// NoRetrieval 一次检索都没成功，内容已被替换成固定文案（不是模型说的）
	NoRetrieval bool `json:"no_retrieval"`
	Interrupted bool `json:"interrupted"`
	Rounds      int  `json:"rounds"`
	ToolCalls   int  `json:"tool_calls"`
	Degraded    bool `json:"degraded"`
	// Steps 本次回答的检索过程。非流式模式下与流式 tool 事件的内容一致，
	// nil 表示一次都没检索（模型直接回答）。
	Steps []entity.ChatStep `json:"steps"`
}

// ===== 后台「助手会话」 =====

// AdminSessionItem 后台会话列表的一行。
//
// 与 ChatSessionItem 的区别：这是「管理员视角」，要多给用户名、文章名、首问摘要，
// 而且**不过滤文章 status**（草稿文章也可能有历史会话，管理员需要能查到）。
type AdminSessionItem struct {
	ID           uint   `json:"id"`
	ArticleID    uint   `json:"article_id"`
	ArticleTitle string `json:"article_title"`
	// ArticleDeleted 文章已被硬删，标题取不到。列表要显示「文章已删除」而不是空标题，
	// 否则管理员会以为是自己筛选条件写错了。
	ArticleDeleted bool   `json:"article_deleted"`
	UserID         uint   `json:"user_id"`
	UserName       string `json:"user_name"`
	Title          string `json:"title"`
	// Summary 首条用户提问的前若干字。管理员在列表上最想知道「这个人问了什么」，
	// title 可能被改过，首问原文不会。
	Summary       string  `json:"summary"`
	MessageCount  int     `json:"message_count"`
	LastMessageAt *string `json:"last_message_at"`
	CreatedAt     string  `json:"created_at"`
}

// AdminSessionListResponse 后台会话分页结果。
type AdminSessionListResponse struct {
	List     []*AdminSessionItem `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// AdminMessageItem 后台看到的一条消息。
//
// 比前台多给 error / refused 的原因：后台是「查问题现场」，要知道哪句被拒了、
// 哪句失败了。user 消息直接显示 `/标题` 原文，**不渲染 pill**——
// 那样会把「用户到底打了什么」藏起来，反而不利于排查。
type AdminMessageItem struct {
	ID          uint                    `json:"id"`
	Role        string                  `json:"role"`
	Content     string                  `json:"content"`
	Model       string                  `json:"model"`
	Refs        []entity.ChatRefChunk   `json:"refs"`
	ArticleRefs []entity.ChatArticleRef `json:"article_refs"`
	// Steps 检索过程。后台靠它判断「这次为什么答得不好」——
	// 是根本没查、查了没命中、还是查到了但降级了。
	Steps            []entity.ChatStep `json:"steps"`
	PromptTokens     int               `json:"prompt_tokens"`
	CompletionTokens int               `json:"completion_tokens"`
	Interrupted      bool              `json:"interrupted"`
	Refused          bool              `json:"refused"`
	// NoRetrieval 零检索，Content 是后端固定提示
	NoRetrieval bool   `json:"no_retrieval"`
	Error       string `json:"error"`
	CreatedAt   string `json:"created_at"`
}

// AdminSessionDetailResponse 会话详情 + 消息。
//
// 消息不分页：单个会话有 400 条上限（见 §9 的会话总量限制），一次返回可控。
// 真出现「一个会话几百条消息」的情况，那本身就是需要人工介入的异常。
type AdminSessionDetailResponse struct {
	Session  *AdminSessionItem   `json:"session"`
	Messages []*AdminMessageItem `json:"messages"`
}
