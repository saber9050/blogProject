package articlechunk

import (
	"time"

	"blog/internal/model/entity"

	"gorm.io/gorm"
)

// ChunkRow 切分块 + join 出来的文章标题。
//
// 用独立的行结构而不是在 entity.ArticleChunk 上加字段：标题不属于块本身，
// 只是查询时为了少一次往返带出来的。
type ChunkRow struct {
	ID             uint
	ArticleID      uint
	ArticleTitle   string
	SectionIndex   int
	HeadingPath    string
	ChunkIndex     int
	CharCount      int
	Content        string
	EmbeddingModel string
	CreatedAt      time.Time
}

// Query 列表查询条件
type Query struct {
	ArticleID uint
	Keyword   string // 在 content 里模糊匹配
	Page      int
	PageSize  int
}

// ArticleStatRow 按文章聚合的原始行
type ArticleStatRow struct {
	ArticleID    uint
	ArticleTitle string
	ChunkCount   int64
	AvgChars     float64
	LastChunkAt  *time.Time
}

// Repository 文章切分块数据访问接口
//
// 写方法显式接受 tx：向量化的「删旧块 + 插新块 + 回写文章状态」必须在同一个事务里，
// 而事务对象由 service 层的 db.Transaction 提供。repository 自己持有 db 的话，
// 这些方法就跑不到那个事务里，中途失败会留下「旧块删了、新块没写」的空档。
type Repository interface {
	// DeleteByArticle 删除某文章的全部块
	DeleteByArticle(tx *gorm.DB, articleID uint) error
	// BatchCreate 批量插入块
	BatchCreate(tx *gorm.DB, chunks []*entity.ArticleChunk) error
	// CountByArticle 某文章的块数
	CountByArticle(articleID uint) (int64, error)

	// ===== 查询（切分管理用）=====
	// List 分页查询块，不带 content 的完整内容由调用方决定是否截断
	List(q Query) ([]*ChunkRow, int64, error)
	// FindByID 单块详情（含完整 content），未找到返回 (nil, nil)
	FindByID(id uint) (*ChunkRow, error)
	// Summary 总块数与涉及文章数
	Summary() (chunkTotal int64, articleTotal int64, err error)
	// StatByArticle 按文章聚合
	StatByArticle() ([]*ArticleStatRow, error)
	// ListBySection 某文章某小节的全部块，按块序号升序。
	// 同一 section_index 的块拼起来就是整节原文——父子块方案里「大块」的来源。
	ListBySection(articleID uint, sectionIndex int) ([]*ChunkRow, error)
	// TableReady 向量表是否已创建
	TableReady() (bool, error)
}
