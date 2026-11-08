package chat

import (
	"errors"
	"strings"
	"time"

	"blog/internal/model/entity"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// ErrNotFound 目标记录不存在。上层统一映射成 404，不区分「没有」和「不是你的」。
var ErrNotFound = gorm.ErrRecordNotFound

// 表名常量：多处裸 SQL 拼表名，集中一处避免改表时漏改。
const (
	sessionTable = "chat_sessions"
	messageTable = "chat_messages"
)

type chatRepository struct {
	db *gorm.DB
}

// NewChatRepository 创建助手会话仓储实例。
func NewChatRepository(db *gorm.DB) Repository {
	return &chatRepository{db: db}
}

// ===== 会话 =====

func (r *chatRepository) CreateSession(s *entity.ChatSession) error {
	return r.db.Create(s).Error
}

func (r *chatRepository) FindSession(id, userID uint) (*entity.ChatSession, error) {
	var s entity.ChatSession
	// 归属校验和取记录合成一次查询：先查再比对会让「别人的会话」多一次无用往返，
	// 而且两条路径容易写出不一致的返回。
	err := r.db.Where("id = ? AND user_id = ?", id, userID).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *chatRepository) ListSessionsByArticle(userID, articleID uint, limit int) ([]*entity.ChatSession, error) {
	if limit <= 0 {
		limit = 50
	}
	var list []*entity.ChatSession
	// NULLS LAST：还没发过消息的会话 last_message_at 为 NULL，倒在最后，
	// 否则 PG 默认把 NULL 排在最前（DESC 时），新建的空会话会盖住正在聊的那个。
	err := r.db.
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Order("last_message_at DESC NULLS LAST, id DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (r *chatRepository) UpdateSession(id uint, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	// 用 map 而不是 struct：struct 的零值（title=""）会被 GORM 当成「不更新」，
	// 而改标题恰好可能就是想清空。map 能精确表达「这一列要写成空串」。
	res := r.db.Model(&entity.ChatSession{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *chatRepository) TouchSession(id uint, addMessages int, now time.Time) error {
	// 用表达式自增，不用「读出来 +1 再写回」：并发两条消息时后者会覆盖前者，计数直接丢。
	res := r.db.Model(&entity.ChatSession{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"message_count":   gorm.Expr("message_count + ?", addMessages),
			"last_message_at": now,
			"updated_at":      now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *chatRepository) DeleteSession(id uint) error {
	return r.hardDelete(id)
}

func (r *chatRepository) AdminDeleteSession(id uint) error {
	return r.hardDelete(id)
}

// hardDelete 硬删会话：工具记录 → 消息 → 会话，同一事务。
//
// 顺序不能反：反了会留下孤儿行（指向已不存在的父），越积越多且永远查不出来。
// 必须同事务：中途失败会留下「消息删了、会话还在」的空壳，用户点进去一片空白。
//
// ⚠️ chat_tool_messages 是**最深的一层**（消息的从属），必须最先删。
// 它按 session_id 删而不是按 message_id：一条 DELETE 清完，不用先查 message_id 列表。
//
// 没有依赖物理外键的 ON DELETE CASCADE——表上不建外键（与 article_chunks、comments
// 的风格一致，也避免批量删除时的锁等待）。所以这里的顺序就是唯一的保障。
func (r *chatRepository) hardDelete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ?", id).Delete(&entity.ChatToolMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", id).Delete(&entity.ChatMessage{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", id).Delete(&entity.ChatSession{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// ===== 工具消息 =====

func (r *chatRepository) CreateToolMessage(m *entity.ChatToolMessage) error {
	return r.db.Create(m).Error
}

func (r *chatRepository) MessageExistsInSession(sessionID, messageID uint) (bool, error) {
	if sessionID == 0 || messageID == 0 {
		return false, nil
	}
	var n int64
	err := r.db.Model(&entity.ChatMessage{}).
		Where("id = ? AND session_id = ?", messageID, sessionID).
		Count(&n).Error
	return n > 0, err
}

func (r *chatRepository) ListToolMessagesByMessage(messageID uint) ([]*entity.ChatToolMessage, error) {
	if messageID == 0 {
		return nil, nil
	}
	var rows []*entity.ChatToolMessage
	err := r.db.Where("message_id = ?", messageID).Order("round ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *chatRepository) ListToolMessagesBySession(sessionID uint) (map[uint][]*entity.ChatToolMessage, error) {
	out := make(map[uint][]*entity.ChatToolMessage)
	if sessionID == 0 {
		return out, nil
	}
	var rows []*entity.ChatToolMessage
	// 排序保证前端按 round 顺序展示，不用再排一次
	err := r.db.Where("session_id = ?", sessionID).
		Order("message_id ASC, round ASC, id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, m := range rows {
		out[m.MessageID] = append(out[m.MessageID], m)
	}
	return out, nil
}

func (r *chatRepository) LastMessageIDWithTools(sessionID, beforeMessageID uint) (uint, error) {
	var ids []uint
	q := r.db.Model(&entity.ChatToolMessage{}).Where("session_id = ?", sessionID)
	if beforeMessageID > 0 {
		// 只看当前提问之前的消息：当前这条 assistant 是刚建的占位，还没有工具记录
		q = q.Where("message_id < ?", beforeMessageID)
	}
	// Distinct 而不是 GROUP BY：同一个 message 有多轮检索，只关心"哪条消息有记录"
	err := q.Distinct().Order("message_id DESC").Limit(1).Pluck("message_id", &ids).Error
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	return ids[0], nil
}

// ===== 消息 =====

func (r *chatRepository) CreateMessage(m *entity.ChatMessage) error {
	return r.db.Create(m).Error
}

func (r *chatRepository) ListMessages(sessionID, beforeID uint, limit int) ([]*entity.ChatMessage, error) {
	if limit <= 0 {
		limit = 30
	}
	q := r.db.Where("session_id = ?", sessionID)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}
	var list []*entity.ChatMessage
	err := q.Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

func (r *chatRepository) UpdateMessage(id uint, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	res := r.db.Model(&entity.ChatMessage{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *chatRepository) ListAllMessages(sessionID uint) ([]*entity.ChatMessage, error) {
	var list []*entity.ChatMessage
	err := r.db.Where("session_id = ?", sessionID).Order("id ASC").Find(&list).Error
	return list, err
}

// ===== 文章 =====

func (r *chatRepository) FindArticle(id uint) (*ArticleBrief, error) {
	var rows []*ArticleBrief
	// 用 Raw + 显式列，不用 db.Where(&entity.Article{})：实体的 gorm.DeletedAt 会自动加
	// deleted_at IS NULL，看着更省事，但那样也把「软删」语义带进来了；这里就要一条
	// 一眼能看懂、不依赖任何隐式行为的 SQL。
	err := r.db.Raw(`
		SELECT id, title, status, vector_status
		FROM articles
		WHERE id = ? AND deleted_at IS NULL`, id).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *chatRepository) SearchPublishedArticles(keyword string, excludeID uint, limit int) ([]*ArticleBrief, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	kw := strings.TrimSpace(keyword)
	var rows []*ArticleBrief
	// status = 1 是硬条件：草稿标题绝不能出现在前台的候选列表里。
	// 这也是为什么不复用 /articles?keyword= ——那个接口返回完整列表项、不能排除当前文章，
	// 而且它的字段一改，聊天这边就跟着变重。
	sql := `SELECT id, title, status, vector_status FROM articles
		WHERE status = 1 AND deleted_at IS NULL`
	args := []any{}
	if excludeID > 0 {
		sql += " AND id <> ?"
		args = append(args, excludeID)
	}
	if kw != "" {
		sql += " AND title ILIKE ?"
		args = append(args, "%"+kw+"%")
	}
	sql += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *chatRepository) VectorTableReady() (bool, error) {
	var exists bool
	// to_regclass 不存在时返回 NULL，不报错；直接查 information_schema 也行，
	// 但 regclass 更短且与模块 02 里的写法保持一致。
	err := r.db.Raw("SELECT to_regclass('article_chunks') IS NOT NULL").Scan(&exists).Error
	return exists, err
}

// ===== 后台 =====

// adminSelectSession 后台列表与详情共用的列。
//
// first_user_content 用 LATERAL 取首条 user 消息：管理员在列表上最想看到的就是
// 「这个人到底问了什么」，标题可能被改过，原文不会。
// LEFT JOIN users/articles 不用 INNER：用户或文章被硬删后，会话仍在表里，
// 用 INNER 会让这些会话从后台「消失」，管理员就查不到异常数据了。
const adminSelectSession = `
	SELECT s.id, s.article_id,
	       COALESCE(a.title, '') AS article_title,
	       (a.id IS NULL)        AS article_deleted,
	       s.user_id,
	       COALESCE(u.user_name, '') AS user_name,
	       s.title, s.message_count, s.last_message_at, s.created_at,
	       COALESCE(m.content, '') AS first_user_content
	FROM ` + sessionTable + ` s
	LEFT JOIN users u ON u.id = s.user_id
	LEFT JOIN articles a ON a.id = s.article_id
	LEFT JOIN LATERAL (
	    SELECT content FROM ` + messageTable + `
	    WHERE session_id = s.id AND role = 'user'
	    ORDER BY id LIMIT 1
	) m ON true`

func (r *chatRepository) AdminListSessions(q AdminQuery) ([]*SessionRow, int64, error) {
	page, size := q.Page, q.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	// 筛选条件抽出来，列表与计数两处共用一份 SQL 和一份参数，避免改一处漏一处。
	// 不用「闭包传 *gorm.DB 再 Where」：Raw 之后再 Where，GORM 的拼接顺序不保证，
	// 计数 SQL 与列表 SQL 稍有偏差就会出现「列表 3 条、总数 7 条」这种极难察觉的错。
	var conds []string
	var args []any
	if kw := strings.TrimSpace(q.User); kw != "" {
		if id, ok := parseUint(kw); ok {
			// 纯数字：既可能是 user_id，也可能是用户名里的数字，两个都试。
			conds = append(conds, "(s.user_id = ? OR u.user_name ILIKE ?)")
			args = append(args, id, "%"+kw+"%")
		} else {
			// 非数字：只按用户名模糊匹配（拿 "zxp" 去转 int 没意义）。
			conds = append(conds, "u.user_name ILIKE ?")
			args = append(args, "%"+kw+"%")
		}
	}
	if kw := strings.TrimSpace(q.Article); kw != "" {
		conds = append(conds, "a.title ILIKE ?")
		args = append(args, "%"+kw+"%")
	}
	whereSQL := " WHERE 1 = 1"
	if len(conds) > 0 {
		whereSQL += " AND " + strings.Join(conds, " AND ")
	}

	const joinedFrom = ` FROM ` + sessionTable + ` s
		LEFT JOIN users u ON u.id = s.user_id
		LEFT JOIN articles a ON a.id = s.article_id`

	var total int64
	if err := r.db.Raw("SELECT count(*)"+joinedFrom+whereSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序带上 id 兜底：last_message_at 大量为 NULL 时（刚建的会话），只按它排序
	// 结果不稳定，翻页会重复或漏行。
	sql := adminSelectSession + whereSQL +
		` ORDER BY s.last_message_at DESC NULLS LAST, s.id DESC LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), size, (page-1)*size)

	var rows []*SessionRow
	if err := r.db.Raw(sql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *chatRepository) AdminFindSession(id uint) (*SessionRow, error) {
	var row SessionRow
	err := r.db.Raw(adminSelectSession+` WHERE s.id = ?`, id).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

// parseUint 只接受纯数字（不带正负号、不带空格），失败返回 ok=false。
func parseUint(s string) (uint, bool) {
	if s == "" {
		return 0, false
	}
	var n uint
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + uint(c-'0')
	}
	return n, true
}

// ===== 检索 =====

// chunkSelectColumns 两路检索共用的列，保证返回结构一致。
const chunkSelectColumns = `
	c.id, c.article_id, a.title AS article_title,
	c.section_index, COALESCE(c.heading_path, '') AS heading_path,
	c.chunk_index, c.char_count, c.content`

func (r *chatRepository) SemanticSearch(vec pgvector.Vector, idsLiteral string,
	perArticle int, minScore float64) ([]*RetrievedChunk, error) {

	if perArticle <= 0 {
		perArticle = 3
	}
	// 两路各最多 8 篇 × perArticle 条，取满也用不上更多
	limit := perArticle * 8

	// 分组限额在库里做：8 篇各 top-3 会让单篇（通常是当前文章）把名额占满，
	// 其它引用文章一条都进不来。row_number() 在同一趟扫描里顺手算，不额外开销。
	//
	// 为什么 join articles 带 status=1：模块 02 允许草稿向量化，不加这条草稿内容会漏给前台。
	sql := `
	WITH sem AS (
		SELECT ` + chunkSelectColumns + `,
		       c.embedding <=> ? AS dist,
		       row_number() OVER (PARTITION BY c.article_id ORDER BY c.embedding <=> ?) AS rn
		FROM article_chunks c
		JOIN articles a ON a.id = c.article_id AND a.status = 1 AND a.deleted_at IS NULL
		WHERE c.article_id = ANY(?::bigint[])
		  AND c.embedding IS NOT NULL
	)
	SELECT id, article_id, article_title, section_index, heading_path,
	       chunk_index, char_count, content, 1 - dist AS score
	FROM sem
	WHERE rn <= ? AND 1 - dist >= ?
	ORDER BY score DESC
	LIMIT ?`

	var rows []*RetrievedChunk
	err := r.db.Raw(sql, vec, vec, idsLiteral, perArticle, minScore, limit).Scan(&rows).Error
	return rows, err
}

func (r *chatRepository) KeywordSearch(query, idsLiteral string,
	perArticle int, minScore float64) ([]*RetrievedChunk, error) {

	if perArticle <= 0 {
		perArticle = 3
	}
	limit := perArticle * 8

	// ⚠️ 必须用 word_similarity 而不是 similarity：
	//   similarity 是「整段像不像整段」，分母包含整块长度。我们是「几十字查询 vs 几百到
	//   两千字的块」，分母被撑大后分值恒在 0.00x，任何阈值都会把命中全滤掉——
	//   表现是「代码不报错、关键词路永远返回 0 条」，这是最难查的一类静默失效。
	//   word_similarity 只拿块里最像的一小段做比较，才是这个场景该用的。
	//
	// 也刻意不用 `q <% t` 操作符：它依赖会话级的 pg_trgm.word_similarity_threshold，
	// 而我们的连接来自连接池，SET 会残留到下一个使用者。阈值写成显式比较最可控；
	// 代价是将来建 GIN 索引时用不上（那时再改，现在几百行全表扫无所谓）。
	sql := `
	WITH kw AS (
		SELECT ` + chunkSelectColumns + `,
		       word_similarity(?, c.content) AS score,
		       row_number() OVER (
		           PARTITION BY c.article_id
		           ORDER BY word_similarity(?, c.content) DESC
		       ) AS rn
		FROM article_chunks c
		JOIN articles a ON a.id = c.article_id AND a.status = 1 AND a.deleted_at IS NULL
		WHERE c.article_id = ANY(?::bigint[])
	)
	SELECT id, article_id, article_title, section_index, heading_path,
	       chunk_index, char_count, content, score
	FROM kw
	WHERE rn <= ? AND score >= ?
	ORDER BY score DESC
	LIMIT ?`

	var rows []*RetrievedChunk
	err := r.db.Raw(sql, query, query, idsLiteral, perArticle, minScore, limit).Scan(&rows).Error
	return rows, err
}

func (r *chatRepository) ListSectionChunks(keys []SectionKey) ([]*RetrievedChunk, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	// 用 (a = ? AND b = ?) OR (...) 而不是行值 IN ((?,?),(?,?))：
	// 行值 IN 里的占位符在 pgx 下**类型推断不可靠**，会报
	// "could not determine data type of parameter"，而且这个错只在特定组合下出现。
	// 最多 8 组 OR，表达式长一点但类型完全明确。
	conds := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)*2)
	for _, k := range keys {
		conds = append(conds, "(c.article_id = ? AND c.section_index = ?)")
		args = append(args, k.ArticleID, k.SectionIndex)
	}

	sql := `
	SELECT ` + chunkSelectColumns + `, 0 AS score
	FROM article_chunks c
	JOIN articles a ON a.id = c.article_id AND a.status = 1 AND a.deleted_at IS NULL
	WHERE ` + strings.Join(conds, " OR ") + `
	ORDER BY c.article_id, c.section_index, c.chunk_index`

	var rows []*RetrievedChunk
	err := r.db.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

func (r *chatRepository) ArticleText(id uint) (string, error) {
	var rows []struct {
		Content string
	}
	err := r.db.Raw(`SELECT content FROM articles WHERE id = ? AND deleted_at IS NULL`, id).Scan(&rows).Error
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}
	return rows[0].Content, nil
}
