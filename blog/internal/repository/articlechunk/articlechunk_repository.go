package articlechunk

import (
	"strings"

	"blog/internal/model/entity"

	"gorm.io/gorm"
)

const chunkTable = "article_chunks"

// selectChunkRow 列表与详情共用的列。
//
// join articles 只为带出标题：块单独看只有 article_id，界面上必须显示人看得懂的标题。
// join 条件带 deleted_at IS NULL，否则软删除文章的块会让列表出现「标题为空的孤儿行」。
const selectChunkRow = `
	SELECT c.id, c.article_id, COALESCE(a.title, '') AS article_title,
	       c.section_index, COALESCE(c.heading_path, '') AS heading_path,
	       c.chunk_index, c.char_count, c.content,
	       COALESCE(c.embedding_model, '') AS embedding_model,
	       c.created_at
	FROM ` + chunkTable + ` c
	LEFT JOIN articles a ON a.id = c.article_id AND a.deleted_at IS NULL`

type articleChunkRepository struct {
	db *gorm.DB
}

// NewArticleChunkRepository 创建文章块仓储实例
func NewArticleChunkRepository(db *gorm.DB) Repository {
	return &articleChunkRepository{db: db}
}

func (r *articleChunkRepository) DeleteByArticle(tx *gorm.DB, articleID uint) error {
	// 用 Exec 而不是 tx.Delete：块表没有 DeletedAt，本该是硬删除，
	// 但走 ORM 的 Delete 要依赖「模型上没有软删除字段」这个隐含前提，
	// 万一以后有人给 ArticleChunk 加了 DeletedAt，这里会静默变成软删除——
	// 而 chunk 的语义是「整篇替换」，留软删除记录只会让数据越积越多。
	return tx.Exec("DELETE FROM "+chunkTable+" WHERE article_id = ?", articleID).Error
}

func (r *articleChunkRepository) BatchCreate(tx *gorm.DB, chunks []*entity.ArticleChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	// 批大小 100：每行含一个 vector(n) 列，太大容易撞 PG 的参数上限
	return tx.CreateInBatches(chunks, 100).Error
}

func (r *articleChunkRepository) CountByArticle(articleID uint) (int64, error) {
	var n int64
	if err := r.db.Table(chunkTable).Where("article_id = ?", articleID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// List 分页查询
func (r *articleChunkRepository) List(q Query) ([]*ChunkRow, int64, error) {
	ok, err := r.TableReady()
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, nil
	}

	where, args := buildWhere(q)

	var total int64
	countSQL := "SELECT count(*) FROM " + chunkTable + " c " + where
	if err := r.db.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	listSQL := selectChunkRow + " " + where +
		" ORDER BY c.article_id DESC, c.chunk_index ASC LIMIT ? OFFSET ?"
	listArgs := append(append([]interface{}{}, args...), q.PageSize, (q.Page-1)*q.PageSize)

	var rows []*ChunkRow
	if err := r.db.Raw(listSQL, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindByID 单块详情
func (r *articleChunkRepository) FindByID(id uint) (*ChunkRow, error) {
	ok, err := r.TableReady()
	if err != nil || !ok {
		return nil, err
	}

	var rows []*ChunkRow
	if err := r.db.Raw(selectChunkRow+" WHERE c.id = ?", id).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Summary 总块数与涉及文章数
func (r *articleChunkRepository) Summary() (int64, int64, error) {
	ok, err := r.TableReady()
	if err != nil || !ok {
		return 0, 0, err
	}

	var row struct {
		ChunkTotal   int64
		ArticleTotal int64
	}
	err = r.db.Raw("SELECT count(*) AS chunk_total, count(DISTINCT article_id) AS article_total FROM " +
		chunkTable).Scan(&row).Error
	return row.ChunkTotal, row.ArticleTotal, err
}

// StatByArticle 按文章聚合
func (r *articleChunkRepository) StatByArticle() ([]*ArticleStatRow, error) {
	ok, err := r.TableReady()
	if err != nil || !ok {
		return nil, err
	}

	var rows []*ArticleStatRow
	err = r.db.Raw(`
		SELECT a.id AS article_id, a.title AS article_title,
		       count(c.id) AS chunk_count,
		       COALESCE(AVG(c.char_count), 0)::float8 AS avg_chars,
		       max(c.created_at) AS last_chunk_at
		FROM ` + chunkTable + ` c
		JOIN articles a ON a.id = c.article_id AND a.deleted_at IS NULL
		GROUP BY a.id, a.title
		ORDER BY chunk_count DESC, a.id DESC`).Scan(&rows).Error
	return rows, err
}

// ListBySection 某文章某小节的全部块，按块序号升序
func (r *articleChunkRepository) ListBySection(articleID uint, sectionIndex int) ([]*ChunkRow, error) {
	ok, err := r.TableReady()
	if err != nil || !ok {
		return nil, err
	}
	var rows []*ChunkRow
	err = r.db.Raw(selectChunkRow+" WHERE c.article_id = ? AND c.section_index = ? ORDER BY c.chunk_index",
		articleID, sectionIndex).Scan(&rows).Error
	return rows, err
}

// TableReady 向量表是否已创建
func (r *articleChunkRepository) TableReady() (bool, error) {
	var name *string
	if err := r.db.Raw("SELECT to_regclass(?)::text", chunkTable).Scan(&name).Error; err != nil {
		return false, err
	}
	return name != nil, nil
}

// buildWhere 拼查询条件。参数全部走占位符，不拼进 SQL 字符串。
func buildWhere(q Query) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	sb.WriteString(" WHERE 1=1")

	if q.ArticleID > 0 {
		sb.WriteString(" AND c.article_id = ?")
		args = append(args, q.ArticleID)
	}
	if q.Keyword != "" {
		// ILIKE：与项目其它搜索一致（PG 的 LIKE 区分大小写）
		sb.WriteString(" AND c.content ILIKE ?")
		args = append(args, "%"+q.Keyword+"%")
	}
	return sb.String(), args
}
