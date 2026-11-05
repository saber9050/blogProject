package entity

import "time"

// 消息角色。只允许这两个值，不设数据库 CHECK——写错 role 的后果是前端渲染异常，
// 比「插入失败」更容易被测试发现，加 CHECK 的收益低于维护成本。
const (
	ChatRoleUser      = "user"
	ChatRoleAssistant = "assistant"
)

// ChatRefChunk 助手回答引用到的一个切分块，存进 chat_messages.refs（jsonb 数组）。
//
// 不存 content：块内容会被重新向量化整体替换，把正文冗余进消息表会让数据翻倍
// 且可能过期。要展示时按 chunk_id 回查，查不到就降级只显示 heading_path。
type ChatRefChunk struct {
	ChunkID      uint    `json:"chunk_id"`
	ArticleID    uint    `json:"article_id"`
	SectionIndex int     `json:"section_index"`
	HeadingPath  string  `json:"heading_path"`
	RRFScore     float64 `json:"rrf_score"`
}

// ChatArticleRef 用户在某一轮显式引用（`/` 选择）的另一篇文章，存进
// chat_messages.article_refs。只记录「额外引用」的文章，当前文章是隐式的、永远在内。
type ChatArticleRef struct {
	ArticleID uint   `json:"article_id"`
	Title     string `json:"title"`
}

// ChatMessage 会话里的一条消息。
//
// 不嵌 BaseEntity：消息不可编辑，没有 updated_at 的语义。
//
// refs 与 article_refs 是两类完全不同的东西，绝不能合并：
//   - refs        助手侧「模型看了哪些块」，产生方是检索
//   - article_refs 用户侧「点名了哪些文章」，产生方是输入框的 `/` 选择
//
// 合成一个字段会让前端必须靠 role 二分叉解析，且「用户引用」与「实际命中」
// 会混在一起，后端统计「用户引了但没查到」时无从下手。
type ChatMessage struct {
	ID        uint   `gorm:"primaryKey;index:idx_chat_messages_session,priority:2" json:"id"`
	SessionID uint   `gorm:"type:bigint;not null;comment:所属会话id;index:idx_chat_messages_session,priority:1" json:"session_id"`
	Role      string `gorm:"type:varchar(16);not null;comment:角色:user/assistant" json:"role"`
	// Content 用户消息存「用户看到的原文」（pill 序列化成 /文章标题），
	// 助手消息存完整回答。拒答时存后端固定文案。
	Content string `gorm:"type:text;not null;default:'';comment:消息正文" json:"content"`
	// Model 生成该条的模型 id，仅 assistant 有值。
	Model string `gorm:"type:varchar(128);not null;default:'';comment:生成该条的模型id" json:"model"`
	// Refs 助手回答用到的切分块。nil ⇒ SQL NULL ⇒ 接口输出 null（表示没检索或走了全文降级）。
	Refs []ChatRefChunk `gorm:"serializer:json;type:jsonb;comment:助手回答用到的切分块" json:"refs"`
	// ArticleRefs 用户本轮引用的其他文章。nil ⇒ SQL NULL。
	ArticleRefs      []ChatArticleRef `gorm:"serializer:json;type:jsonb;comment:用户本轮引用的其他文章" json:"article_refs"`
	PromptTokens     int              `gorm:"not null;default:0;comment:输入token数" json:"prompt_tokens"`
	CompletionTokens int              `gorm:"not null;default:0;comment:输出token数" json:"completion_tokens"`
	// Error 失败原因（已脱敏）。非空表示这条助手消息没答完，但已生成的内容仍然保留——
	// 清空已生成的几百字等于既白烧 token 又白等时间。
	Error string `gorm:"type:varchar(255);not null;default:'';comment:失败原因(脱敏后)" json:"error"`
	// Interrupted 用户主动停止或断开连接。
	Interrupted bool `gorm:"not null;default:false;comment:用户中途停止/断开" json:"interrupted"`
	// Refused 助手判定为跑题。此时 Content 是后端固定文案而非模型输出，
	// 前端据此换低调样式；它也是后续统计「用户都在问什么不相关的东西」的依据。
	Refused   bool      `gorm:"not null;default:false;comment:判定为跑题,content为固定文案" json:"refused"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}
