package articlechunk

import (
	"time"

	"blog/internal/model/entity"

	"gorm.io/gorm"
)

// ChunkRow 切分块 + join 出来的文章标题与模板名。
//
// 用独立的行结构而不是在 entity.ArticleChunk 上加字段：那两个字段不属于块本身，
// 只是查询时为了少一次往返带出来的。
type ChunkRow struct {
	ID             uint
	ArticleID      uint
	ArticleTitle   string
	TemplateID     uint
	TemplateName   string
	ChunkIndex     int
	CharCount      int
	Content        string
	EmbeddingModel string
	CreatedAt      time.Time
}

// Query 列表查询条件
type Query struct {
	TemplateID uint
	ArticleID  uint
	Keyword    string // 在 content 里模糊匹配
	Page       int
	PageSize   int
}

// TemplateStatRow 按模板聚合的原始行
type TemplateStatRow struct {
	TemplateID   uint
	TemplateName string
	Status       int8
	ArticleCount int64
	ChunkCount   int64
	AvgChars     float64
	LastChunkAt  *time.Time
}

// ArticleStatRow 按文章聚合的原始行
type ArticleStatRow struct {
	ArticleID    uint
	ArticleTitle string
	TemplateID   uint
	TemplateName string
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
	// StatByTemplate 按模板聚合。未被使用的模板也会出现（count 为 0）
	StatByTemplate() ([]*TemplateStatRow, error)
	// StatByArticle 按文章聚合
	StatByArticle() ([]*ArticleStatRow, error)
	// TableReady 向量表是否已创建
	TableReady() (bool, error)
}
