package chat

import (
	"time"

	"blog/internal/model/entity"
)

// SessionRow 后台会话列表的一行：会话 + 所属文章标题 + 所属用户 + 首条提问摘要。
//
// 用独立行结构而不是往 entity.ChatSession 上加字段：文章标题、用户名、摘要都不属于
// 会话本身，只是查询时为了少几次往返带出来的；混进实体会让「哪些列是真的落库」
// 变得模糊，写回时也容易误带只读字段。
type SessionRow struct {
	ID               uint
	ArticleID        uint
	ArticleTitle     string
	ArticleDeleted   bool // 文章已硬删（join 不到标题）时为 true，后台显示「文章已删除」
	UserID           uint
	UserName         string
	Title            string
	MessageCount     int
	LastMessageAt    *time.Time
	CreatedAt        time.Time
	FirstUserContent string // 首条 user 消息原文，后台列表截断展示
}

// AdminQuery 后台会话列表的筛选条件。
type AdminQuery struct {
	// User 一个输入框两种匹配：纯数字时先按 user_id 精确匹配（同时仍参与用户名模糊匹配），
	// 非数字只按用户名模糊匹配。合成一个字段是为了让管理员不必先想「我输入的是 id 还是名字」。
	User string
	// Article 文章标题模糊匹配。
	Article  string
	Page     int
	PageSize int
}

// Repository 助手会话与消息的数据访问接口。
type Repository interface {
	// ===== 会话（前台，一律带 user_id 做归属校验）=====

	// CreateSession 新建会话并回填 ID。
	CreateSession(s *entity.ChatSession) error
	// FindSession 按 (id, userID) 取会话。不存在或不属于该用户都返回 (nil, nil)——
	// 调用方统一回 404，不区分「没有」和「不是你的」，避免被用来探测会话是否存在。
	FindSession(id, userID uint) (*entity.ChatSession, error)
	// ListSessionsByArticle 某用户在某文章下的会话列表，最近活跃优先，最多 limit 条。
	ListSessionsByArticle(userID, articleID uint, limit int) ([]*entity.ChatSession, error)
	// UpdateSession 更新会话的可变字段（title / model_config_id 等），空 map 直接返回。
	UpdateSession(id uint, fields map[string]any) error
	// TouchSession 发完一轮消息后更新计数与活跃时间。用 SQL 自增而不是读改写：
	// 两个请求并发时读改写会互相覆盖，计数就会丢。
	TouchSession(id uint, addMessages int, now time.Time) error
	// DeleteSession 硬删会话及其全部消息（同一事务，先删子表）。
	DeleteSession(id uint) error

	// ===== 消息 =====

	// CreateMessage 插入一条消息并回填 ID。
	CreateMessage(m *entity.ChatMessage) error
	// ListMessages 倒序取 limit 条（id 从大到小），调用方需要正序时自行反转。
	// beforeID = 0 表示取最新一页。
	ListMessages(sessionID, beforeID uint, limit int) ([]*entity.ChatMessage, error)
	// ListAllMessages 取会话全部消息，正序。仅后台详情与历史组装用（会话有 400 条上限，
	// 一次性取完可控；真要流式再看是否需要游标）。
	ListAllMessages(sessionID uint) ([]*entity.ChatMessage, error)

	// ===== 后台 =====

	// AdminListSessions 全站会话分页，带用户 / 文章筛选。
	// ⚠️ 不按 articles.status 或 deleted_at 过滤：草稿文章、已删文章的会话都要能看到，
	// 后台是「查发生过什么」，和「现在能不能检索」是两件事。
	AdminListSessions(q AdminQuery) ([]*SessionRow, int64, error)
	// AdminFindSession 后台取单个会话（不校验归属）。
	AdminFindSession(id uint) (*SessionRow, error)
	// AdminDeleteSession 后台硬删，与 DeleteSession 同语义。
	AdminDeleteSession(id uint) error
}

// 编译期断言：实现必须满足接口，防止改签名单边漂移。
var _ Repository = (*chatRepository)(nil)
