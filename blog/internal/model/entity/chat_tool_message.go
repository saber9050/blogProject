package entity

import "time"

// ChatToolMessage 一次工具调用的完整记录，独立于 chat_messages 存放。
//
// 为什么不塞进 chat_messages 的 jsonb：
//   - Content 是检索回来的一整段资料（最多 8 段 × 2000 字），单行 6KB 起步。
//     塞进消息表会让每次历史列表查询都拖着它走，行变大、缓存命中率下降。
//   - 独立表可以按需加载：历史列表只查元信息，用户点开某一轮才拉原文。
//   - 删会话时按 session_id 一条 DELETE 就能清完，不用先查出所有 message_id。
//
// 为什么必须存 Content：这是「它当时到底读到了什么」的唯一凭证。
// refs 里只有 chunk_id，而模块 02 的重新向量化是「删旧写新」，chunk_id 会整体变化，
// 历史 refs 迟早指向不存在的块——那时想复盘「为什么这么答」就彻底没证据了。
type ChatToolMessage struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// SessionID 冗余一份而不是只靠 message_id 关联：删会话要按 session 一条 DELETE，
	// 只靠 message_id 得先查出该会话的所有 message_id 再删，多一次往返且可能漏。
	SessionID uint `gorm:"type:bigint;not null;comment:所属会话id;index:idx_chat_tool_messages_session,priority:1" json:"session_id"`
	// MessageID 属于哪条 assistant 消息（也就是那一轮回答）。
	MessageID uint `gorm:"type:bigint;not null;comment:所属助手消息id;index:idx_chat_tool_messages_msg,priority:1" json:"message_id"`
	// Round 第几轮检索，与 chat_messages.steps 里的 round 对应。
	Round int `gorm:"not null;default:0;comment:第几轮检索;index:idx_chat_tool_messages_msg,priority:2" json:"round"`

	ToolName string `gorm:"type:varchar(64);not null;default:'';comment:工具名" json:"tool_name"`
	// ToolCallID 模型给的调用 id。
	//
	// ⚠️ 回放历史时必须与 assistant 消息的 ToolCalls[].ID **一致**：
	// OpenAI 协议要求 tool 消息紧跟在带 tool_calls 的 assistant 消息之后，
	// 且靠这个 id 配对。对不上会被 provider 直接拒绝（400）。
	ToolCallID string `gorm:"type:varchar(64);not null;default:'';comment:模型给的调用id" json:"tool_call_id"`
	// Arguments 完整参数 JSON。回放时要原样用它，不能只存 query——
	// 工具签名将来加参数时，只存 query 就没法还原调用。
	Arguments string `gorm:"type:text;not null;default:'';comment:工具参数JSON" json:"arguments"`
	// Query 从 Arguments 里提出来的检索词，单独存一份便于排查时扫。
	Query string `gorm:"type:varchar(512);not null;default:'';comment:检索词" json:"query"`
	// Content 工具返回给模型的资料原文（含 `[1] 《标题》· 小节` 这样的段头）。
	//
	// 回放与审计都靠它：回放让模型看到「上一轮实际读到的东西」，
	// 审计让管理员能判断答案是否忠于文章。
	Content string `gorm:"type:text;not null;default:'';comment:工具返回给模型的原文" json:"content"`

	Hits  int   `gorm:"not null;default:0;comment:命中段数" json:"hits"`
	Chars int   `gorm:"not null;default:0;comment:返回字数" json:"chars"`
	Ms    int64 `gorm:"not null;default:0;comment:耗时毫秒" json:"ms"`
	// Articles 命中的文章标题。用 jsonb 而不是逗号拼接：标题里可能就有逗号。
	Articles []string `gorm:"serializer:json;type:jsonb;comment:命中的文章标题" json:"articles"`
	Degraded bool     `gorm:"not null;default:false;comment:本次检索是否降级" json:"degraded"`
	Note     string   `gorm:"type:varchar(255);not null;default:'';comment:降级或异常说明" json:"note"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

func (ChatToolMessage) TableName() string {
	return "chat_tool_messages"
}
