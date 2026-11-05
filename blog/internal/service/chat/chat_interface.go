package chat

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
)

// MaxRefArticles 单次提问可涉及的文章总篇数上限（含当前文章）。
//
// 放在服务端并随 context 下发，前端据此算 pill 上限（8-1=7），不在前端写死：
// 将来调这个数只改一处，不会出现「后端拒了但前端还让选」的错位。
const MaxRefArticles = 8

// MaxSessionsPerArticle 单篇文章下单用户的会话展示上限。
//
// 只在 context 的 sessions 列表里生效（不分页）。单篇下超过 50 个会话是很反常的
// 使用方式，真出现也该由用户自己清理，接口先按 50 截断、不报错。
const MaxSessionsPerArticle = 50

// Service 助手会话服务接口。
//
// P1 只包含会话 CRUD 与候选文章查询；发消息（含 agent 编排、检索、流式）在 P2 起接。
type Service interface {
	// Context 面板初始化。文章不存在 / 未发布返回 404。
	Context(userID, articleID uint) (*response.ChatContextResponse, error)

	// CreateSession 新建会话。article_id 必须存在且已发布；
	// model_config_id 为 0 时用「首个启用的对话模型」。
	CreateSession(userID uint, req *request.CreateChatSessionRequest) (*response.ChatSessionItem, error)

	// ListMessages 历史消息（游标分页，id 倒序取一页后反转成正序）。
	// 会话不存在或不属于该用户，统一 404。
	ListMessages(userID, sessionID uint, beforeID uint, limit int) (*response.ChatMessageListResponse, error)

	// UpdateSession 改标题 / 换模型。会话归属不符 404。
	UpdateSession(userID, sessionID uint, req *request.UpdateChatSessionRequest) error

	// DeleteSession 硬删会话及其消息。会话归属不符 404。
	DeleteSession(userID, sessionID uint) error

	// ListArticleCandidates `/` 选择器的候选源，只返回已发布文章。
	ListArticleCandidates(keyword string, excludeID uint, limit int) ([]*response.ChatArticleCandidateItem, error)
}

// 编译期断言：防止实现与接口签名单边漂移。
var _ Service = (*chatService)(nil)
