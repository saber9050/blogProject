package entity

import "time"

// ChatSession 文章页 AI 助手的会话。
//
// 一个会话绑定「一个用户 + 一篇文章」：会话创建时写入 article_id，之后不可改。
// 用户在提问时可以用 `/` 额外引用别的文章，但那只是「本轮补充资料」，落在
// chat_messages.article_refs 上，不改变这里的归属——否则引用几篇就会把会话
// 从原文下拖走，「不同文章的对话不互串」就形同虚设。
//
// 不嵌 BaseEntity：字段里没有 DeletedAt，删除是硬删（消息一起删），
// 嵌了反而要时刻提防 gorm 把 Delete 变成软删。
//
// 也不嵌 gorm.DeletedAt，理由同上——设计上没有「已删除但仍存在」的会话，
// 一旦加上，db.Delete 会静默变成 UPDATE deleted_at，而所有查询又都不带
// deleted_at IS NULL，结果是「删了还在、列表里还看得见」，且没人想得到去查。
type ChatSession struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// ArticleID 会话所属文章。索引有两处用途：
	//   idx_chat_sessions_user_article 的第二列（前台按用户+文章列会话）
	//   idx_chat_sessions_article 的第一列（后台按文章筛选）
	ArticleID uint `gorm:"type:bigint;not null;comment:所属文章id;index:idx_chat_sessions_user_article,priority:2;index:idx_chat_sessions_article,priority:1" json:"article_id"`
	// UserID 会话归属用户。游客不支持，因此 not null。
	UserID uint `gorm:"type:bigint;not null;comment:所属用户id;index:idx_chat_sessions_user_article,priority:1" json:"user_id"`
	// ModelConfigID 本会话使用的对话模型配置（ai_model_configs.id）。
	// 放在会话级而不是全局：A 会话用便宜的模型、B 会话用强的，切换后历史不受影响。
	ModelConfigID uint `gorm:"type:bigint;not null;comment:本会话使用的对话模型配置id" json:"model_config_id"`
	// Title 会话标题，取首条提问前 30 个 rune。空串表示还没发过消息。
	Title string `gorm:"type:varchar(120);not null;default:'';comment:会话标题" json:"title"`
	// MessageCount 该会话的全部消息行数（user + assistant，含失败/拒答/中断）。
	// 语义等于 count(chat_messages where session_id=?)，每发一次消息恒定 +2。
	// ⚠️ 不是「提问次数」，拿它和「200 轮」比之前先除以 2。
	MessageCount int `gorm:"not null;default:0;comment:消息总条数(user+assistant)" json:"message_count"`
	// LastMessageAt 会话列表排序键，也是前端显示「3 分钟前」的依据。
	// 两个索引都把它放在末位并倒序，让「最近活跃优先」直接走索引顺序。
	LastMessageAt *time.Time `gorm:"comment:最后一条消息时间;index:idx_chat_sessions_user_article,priority:3,sort:desc;index:idx_chat_sessions_article,priority:2,sort:desc" json:"last_message_at"`
	CreatedAt     time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

func (ChatSession) TableName() string {
	return "chat_sessions"
}
