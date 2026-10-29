package entity

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

// ArticleChunk 文章切分块（含向量）
//
// 单表存储，不按模板分表：只允许一个向量模型 ⇒ 维度固定 ⇒ 分表唯一的好处
// （不同表不同维度）不存在，而「按文章查」在分表方案下要跨 N 张表 UNION 分页，
// 成本远高于收益。切分参数也已固化为 chunker.DefaultParams，没有模板概念。
//
// 这张表**不进 entity.AllModels()**，由 pkg/database/vector.go 的 EnsureSchema 手写 DDL：
// vector(n) 的维度来自运行时配置（ai_model_configs.dimension），AutoMigrate 表达不了；
// 而且一旦把 vector 列交给 AutoMigrate，目标库没装 pgvector 扩展时整个应用都起不来。
//
// 不嵌 BaseEntity：块是「整篇替换」的产物，没有独立生命周期，不需要 updated_at /
// deleted_at（加了反而会在删旧写新时被软删除逻辑干扰）。
type ArticleChunk struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	ArticleID      uint   `gorm:"not null;index:idx_article_chunks_article" json:"article_id"`
	ChunkIndex     int    `gorm:"not null;comment:块序号,从0开始" json:"chunk_index"`
	Content        string `gorm:"type:text;not null;comment:切分后的原文" json:"content"`
	CharCount      int    `gorm:"not null;default:0;comment:字符数" json:"char_count"`
	EmbeddingModel string `gorm:"type:varchar(128);comment:产出该向量的模型id" json:"embedding_model"`
	// Embedding 只用 column 显式指定列名，不写 type:vector(n)。
	//   ① 这张表不进 AutoMigrate，type 永远不会被用到，写了只会与实际列维度不符、误导人；
	//   ② 但必须显式写 column，否则 GORM 见到 struct 类型字段会先怀疑是关联关系。
	//      pgvector.Vector 实现了 driver.Valuer，加上 column 后按普通列处理，读写走
	//      Value()/Scan()，不会误判成 has-one。
	Embedding pgvector.Vector `gorm:"column:embedding" json:"-"`
	CreatedAt time.Time       `gorm:"not null" json:"created_at"`
}

func (ArticleChunk) TableName() string {
	return "article_chunks"
}
