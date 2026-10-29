package database

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"gorm.io/gorm"
)

// chunkTable 向量块表名。写成常量而不是从 entity 取，是为了让本文件的 DDL
// 与 entity.ArticleChunk.TableName() 的对应关系一眼可见（两边改动时必须同步）。
const chunkTable = "article_chunks"

// 维度上限。pgvector 单个 vector 列最多 16000 维；HNSW 索引最多 2000 维。
// 这里只限制列维度，索引还没建（见 ensureChunkTable 的注释）。
const maxVectorDimension = 16000

// ErrVectorUnavailable 目标库没有 pgvector 扩展。
//
// 单列一个错误是为了让上层能区分「装不上扩展」和「其它 SQL 错误」：
// 前者应该降级（向量功能不可用，其余功能照常），后者应该直接报错。
var ErrVectorUnavailable = errors.New("pgvector 扩展不可用")

// DimensionMismatchError 已有的向量表维度与期望维度不一致。
//
// 这个错误不该被自动修复。pgvector 的 vector(n) 维度写在 DDL 里，改维度只能删表重建，
// 而删表意味着丢掉全部向量数据——那是必须由人确认的操作，绝不能由一次「启动自检」静默完成。
type DimensionMismatchError struct {
	Current int   // 表当前维度
	Want    int   // 期望维度（来自当前向量模型的 dimension）
	Chunks  int64 // 表里已有的块数，用于告诉用户会丢多少数据
}

func (e *DimensionMismatchError) Error() string {
	// 块数为 0 时不要说「已有 0 个块」：用户会想「既然没数据，为什么不能换维度」。
	// 真正的原因是表结构已经固定，即使没有数据也只能重建。
	if e.Chunks <= 0 {
		return fmt.Sprintf("向量表当前是 %d 维（表结构已固定，暂无数据），与期望的 %d 维不一致",
			e.Current, e.Want)
	}
	return fmt.Sprintf("向量表当前是 %d 维（已有 %d 个块），与期望的 %d 维不一致",
		e.Current, e.Chunks, e.Want)
}

// EnsureSchema 确保向量表存在且维度与 dim 一致。
//
// 三种分支：
//   - 表不存在      → 按 dim 建表 + 索引
//   - 表存在，维度相同 → 什么都不做（幂等，每次启动都会调）
//   - 表存在，维度不同 → 返回 *DimensionMismatchError，不自动改表
//
// 调用方：应用启动（维度从库里的向量模型读）、新增向量模型（维度从请求里读）。
func EnsureSchema(db *gorm.DB, dim int) error {
	if dim <= 0 {
		return fmt.Errorf("向量维度必须为正数，收到 %d", dim)
	}
	if dim > maxVectorDimension {
		return fmt.Errorf("向量维度 %d 超出 pgvector 上限 %d", dim, maxVectorDimension)
	}

	if err := ensureVectorExtension(db); err != nil {
		return err
	}

	// 先清掉模板概念的遗留结构，再判断表是否存在。
	// 顺序不能反：只删表不删列的话，重建出来的表仍然带着 template_id。
	cleanupLegacyTemplateSchema(db)

	current, exists, err := CurrentDimension(db)
	if err != nil {
		return err
	}
	if exists {
		if current == dim {
			// 索引也要在这个分支补：表可能是更早的版本建的，缺后来新增的索引。
			// CREATE INDEX IF NOT EXISTS 是幂等的，每次启动都跑一遍代价可忽略。
			return ensureChunkIndexes(db)
		}
		var chunks int64
		if err := db.Table(chunkTable).Count(&chunks).Error; err != nil {
			// 数不出来不影响判断，只是错误信息里少个数字
			chunks = -1
		}
		return &DimensionMismatchError{Current: current, Want: dim, Chunks: chunks}
	}

	return ensureChunkTable(db, dim)
}

// CurrentDimension 读取现有向量表的 embedding 列维度。
//
// 第二个返回值表示表是否存在。用 to_regclass 而不是 'article_chunks'::regclass：
// 后者在表不存在时会直接抛 SQL 错误，没法当「不存在」处理。
//
// 用 format_type 而不是直接读 atttypmod：pgvector 的 typmod 编码规则属于实现细节，
// format_type 直接返回 "vector(1024)" 这种可读形式，稳且不用关心内部约定。
func CurrentDimension(db *gorm.DB) (int, bool, error) {
	var colType string
	err := db.Raw(`
		SELECT format_type(atttypid, atttypmod)
		FROM pg_attribute
		WHERE attrelid = to_regclass(?) AND attname = 'embedding'`,
		chunkTable).Scan(&colType).Error
	if err != nil {
		return 0, false, fmt.Errorf("读取向量表维度失败: %w", err)
	}
	if colType == "" {
		return 0, false, nil
	}

	m := dimensionPattern.FindStringSubmatch(colType)
	if m == nil {
		return 0, false, fmt.Errorf("向量表 embedding 列类型异常：%s", colType)
	}
	dim, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false, fmt.Errorf("解析向量维度失败：%s", colType)
	}
	return dim, true, nil
}

// DropSchema 删除向量表。**危险操作**，只应由 /admin/vector/reset 在用户显式确认后调用。
//
// 只删表，不碰 articles 的向量字段——状态归零由 service 层负责，因为它要写业务表。
func DropSchema(db *gorm.DB) error {
	return db.Exec("DROP TABLE IF EXISTS " + chunkTable).Error
}

// EnsureSingleEmbeddingModelIndex 给 ai_model_configs 建部分唯一索引，
// 保证全局只有一条 type=2（向量模型）的记录。
//
// 这是业务校验的兜底：服务层已经在 Create 时挡了一次，但手工改库能绕过服务层。
// 索引建不上（库里已有多条 type=2）时只报错不阻断启动——启动失败比多一条脏数据严重得多。
//
// 注意：MySQL 不支持带 WHERE 的索引，这是迁到 PostgreSQL 之后的额外收益。
func EnsureSingleEmbeddingModelIndex(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS uk_ai_model_configs_single_embedding
		ON ai_model_configs (type) WHERE type = 2`).Error
}

var dimensionPattern = regexp.MustCompile(`^vector\((\d+)\)$`)

// ensureVectorExtension 建扩展。失败即目标库不是 pgvector 镜像。
func ensureVectorExtension(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return fmt.Errorf("%w: %v", ErrVectorUnavailable, err)
	}
	return nil
}

// ensureChunkTable 按 dim 建表。
//
// 索引说明：只建 article_id 与 (article_id, chunk_index) 两个普通索引，**不建 HNSW**。
// 本模块没有检索出口，HNSW 只会在每次 INSERT/DELETE 时维护，纯开销；
// 等做语义检索时再执行
//
//	CREATE INDEX ... USING hnsw (embedding vector_cosine_ops) WITH (m=16, ef_construction=64)
//
// 那时才有 EXPLAIN 能验证效果。另外 HNSW 有 2000 维上限，维度配得大时还得换 halfvec。
func ensureChunkTable(db *gorm.DB, dim int) error {
	// 只有维度是数字（前面已校验为正整数且有上限），不存在注入面
	ddl := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id              bigserial PRIMARY KEY,
			article_id      bigint      NOT NULL,
			chunk_index     integer     NOT NULL,
			content         text        NOT NULL,
			char_count      integer     NOT NULL DEFAULT 0,
			embedding_model varchar(128),
			embedding       vector(%d),
			created_at      timestamptz NOT NULL DEFAULT now()
		)`, chunkTable, dim)
	if err := db.Exec(ddl).Error; err != nil {
		return fmt.Errorf("创建向量表失败: %w", err)
	}
	return ensureChunkIndexes(db)
}

// ensureChunkIndexes 幂等地建表索引。
//
// 单独拆出来是因为「表已存在且维度一致」那条分支也要跑：表可能是更早的版本建的，
// 缺后来新增的索引，而 EnsureSchema 在那条分支上原本直接 return，永远补不上。
func ensureChunkIndexes(db *gorm.DB) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_article_chunks_article ON ` + chunkTable + ` (article_id)`,
		// 明细列表按「文章倒序 + 块序号正序」分页，这个组合索引直接覆盖排序
		`CREATE INDEX IF NOT EXISTS idx_article_chunks_article_index ON ` + chunkTable + ` (article_id, chunk_index)`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return fmt.Errorf("创建向量表索引失败: %w", err)
		}
	}
	return nil
}

// cleanupLegacyTemplateSchema 清掉「切分模板」下线后遗留的结构。
//
// 模板概念已在 2026-10-29 移除，但已有库里还留着 article_chunks.template_id 列和
// chunk_templates 表。EnsureSchema 用的是 CREATE TABLE IF NOT EXISTS，改 DDL
// 影响不到已经存在的表——而 template_id 是 NOT NULL 且没有默认值，代码又不再写它，
// 不清理的话下一次插入块会直接报 not-null violation。
//
// 失败只忽略不报错：表可能本来就不存在，或者当前库用户没有 DDL 权限；
// 真要出问题会在插入时暴露，不必让整个启动流程卡在「清理历史结构」这一步。
func cleanupLegacyTemplateSchema(db *gorm.DB) {
	_ = db.Exec("ALTER TABLE " + chunkTable + " DROP COLUMN IF EXISTS template_id").Error
	_ = db.Exec("DROP TABLE IF EXISTS chunk_templates").Error
}
