package vectorschema

import (
	stderrors "errors"
	"fmt"

	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	"blog/pkg/database"
	"blog/pkg/errors"
	"blog/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// chunkTable 向量块表名（与 entity.ArticleChunk.TableName() 一致）
const chunkTable = "article_chunks"

type vectorSchemaService struct {
	db *gorm.DB
}

// NewVectorSchemaService 创建向量表维护服务实例
func NewVectorSchemaService(db *gorm.DB) Service {
	return &vectorSchemaService{db: db}
}

// EnsureFor 按给定维度确保向量表就绪
//
// 维度冲突的错误直接透传下层：database.DimensionMismatchError 的 Error()
// 已经是一句完整的话（含当前维度、块数、差异），在这里再包装一层反而丢信息。
func (s *vectorSchemaService) EnsureFor(dim int) error {
	return database.EnsureSchema(s.db, dim)
}

// EnsureFromDB 读当前向量模型的维度再建表
func (s *vectorSchemaService) EnsureFromDB() error {
	m, err := s.activeEmbeddingModel()
	if err != nil {
		return err
	}
	if m == nil {
		logger.Info("未配置向量模型，跳过向量表初始化")
		return nil
	}
	if m.Dimension <= 0 {
		logger.Warn("向量模型未填写维度，跳过向量表初始化",
			zap.Uint("config_id", m.ID), zap.String("model", m.Model))
		return nil
	}
	if err := database.EnsureSchema(s.db, m.Dimension); err != nil {
		var mismatch *database.DimensionMismatchError
		if stderrors.As(err, &mismatch) {
			logger.Warn("向量表维度与向量模型不一致，向量化功能不可用；如需切换维度请调用 /admin/vector/reset",
				zap.Int("table_dimension", mismatch.Current),
				zap.Int("model_dimension", mismatch.Want),
				zap.Int64("existing_chunks", mismatch.Chunks))
			return err
		}
		// 走这里的通常是没装 pgvector 扩展，或建表权限不足
		logger.Warn("向量表初始化失败，向量化功能将不可用", zap.Error(err))
		return err
	}
	logger.Info("向量表就绪", zap.Int("dimension", m.Dimension), zap.String("model", m.Model))
	return nil
}

// Info 当前向量表与向量模型的匹配情况
//
// 每个字段都有明确的用途，不做「顺手多查几个」。这个接口会在
// 「AI 模型」新增弹窗打开时被调用，太重的查询会拖慢表单渲染。
func (s *vectorSchemaService) Info() (*response.VectorSchemaResponse, error) {
	out := &response.VectorSchemaResponse{}

	ready, err := s.extensionReady()
	if err != nil {
		return nil, err
	}
	out.ExtensionReady = ready

	dim, exists, err := database.CurrentDimension(s.db)
	if err != nil {
		return nil, err
	}
	out.TableExists = exists
	out.Dimension = dim

	if exists {
		if err := s.db.Table(chunkTable).Count(&out.ChunkTotal).Error; err != nil {
			return nil, err
		}
		if err := s.db.Table(chunkTable).
			Distinct("article_id").Count(&out.ArticleTotal).Error; err != nil {
			return nil, err
		}
	}

	// 查模型时不限 status：要让用户看到「配了但被禁用」这个状态，
	// 只查启用的会显示成「未配置」，反而更容易误解
	m, err := s.anyEmbeddingModel()
	if err != nil {
		return nil, err
	}
	if m != nil {
		out.Model = &response.VectorSchemaModel{
			ID: m.ID, Model: m.Model, Dimension: m.Dimension, Status: m.Status,
		}
		out.Match = exists && m.Dimension == dim
	}

	out.Message = s.describe(out)
	return out, nil
}

// Reset 清空向量数据。
//
// **只删表，不重建。**
//
// 这是实现时才发现的一个逻辑漏洞：如果按「删表 + 用当前模型维度重建」来做，
// 「换成 768 维模型」这个操作会陷入死循环——
//
//  1. reset → 表被旧模型维度（1024）重建
//  2. 删掉旧模型 → 表还是 1024
//  3. 新建 768 维模型 → EnsureFor(768) 撞上 1024 的表 → 报维度冲突
//
// 所以把重建交给「新增向量模型」那一步：Reset 之后表不存在，任何合法维度都能建。
// 代价是 reset 之后如果不重新配模型，向量化会失败——但前置校验里有
// 「表不存在则 EnsureFromDB 自动建表」，实际能自愈。
func (s *vectorSchemaService) Reset() (*response.VectorResetResponse, error) {
	// 有任务正在排队或处理时不允许清空：正在跑的任务会把旧维度的向量写进新表，
	// 结果是 INSERT 失败 + 一批无意义的失败记录
	var running int64
	if err := s.db.Model(&entity.Article{}).
		Where("vector_status IN ?", []int8{entity.VectorStatusQueued, entity.VectorStatusRunning}).
		Count(&running).Error; err != nil {
		return nil, errors.NewWithErr(errors.CodeInternalError, "检查进行中的向量化任务失败", err)
	}
	if running > 0 {
		return nil, errors.New(errors.CodeConflict, fmt.Sprintf(
			"有 %d 篇文章正在排队或处理中，请等待完成后再清空", running))
	}

	var deleted int64
	if ok, err := s.tableExists(); err != nil {
		return nil, err
	} else if ok {
		if err := s.db.Table(chunkTable).Count(&deleted).Error; err != nil {
			return nil, err
		}
	}

	var resetArticles int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := database.DropSchema(tx); err != nil {
			return fmt.Errorf("删除向量表失败: %w", err)
		}
		// 向量字段归零。vector_template_id 也清掉——它指向的模板已经不再
		// 对应该文章的任何块，留着会误导「这篇文章是用 X 模板切的」
		res := tx.Model(&entity.Article{}).Where("vector_status <> ?", entity.VectorStatusNone).
			Updates(map[string]interface{}{
				"vector_status":      entity.VectorStatusNone,
				"vector_template_id": 0,
				"vector_chunk_count": 0,
				"vector_error":       "",
				"vectorized_at":      nil,
			})
		if res.Error != nil {
			return fmt.Errorf("重置文章向量状态失败: %w", res.Error)
		}
		resetArticles = res.RowsAffected
		return nil
	})
	if err != nil {
		logger.Error("清空向量数据失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "清空向量数据失败", err)
	}

	logger.Warn("向量数据已清空", zap.Int64("deleted_chunks", deleted), zap.Int64("reset_articles", resetArticles))
	return &response.VectorResetResponse{
		DeletedChunks: deleted,
		ResetArticles: resetArticles,
	}, nil
}

// describe 生成一句人话说明，前端可以直接展示
func (s *vectorSchemaService) describe(r *response.VectorSchemaResponse) string {
	switch {
	case !r.ExtensionReady:
		return "当前数据库未安装 pgvector 扩展，向量化功能不可用"
	case r.Model == nil:
		return "尚未配置向量模型，向量表未初始化"
	case r.Model.Dimension <= 0:
		return "向量模型缺少维度配置，请删除后重新添加"
	case r.Model.Status != 1:
		return "向量模型已禁用，请先在「AI 模型」页启用"
	case !r.TableExists:
		return "向量表尚未创建，将在下次启动或配置模型时自动创建"
	case !r.Match:
		return fmt.Sprintf("向量表是 %d 维，与向量模型的 %d 维不一致；如需切换请先清空向量数据",
			r.Dimension, r.Model.Dimension)
	}
	return "向量表与向量模型一致"
}

func (s *vectorSchemaService) extensionReady() (bool, error) {
	var n int64
	err := s.db.Raw("SELECT count(*) FROM pg_available_extensions WHERE name = 'vector'").Scan(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *vectorSchemaService) tableExists() (bool, error) {
	var name *string
	if err := s.db.Raw("SELECT to_regclass(?)::text", chunkTable).Scan(&name).Error; err != nil {
		return false, err
	}
	return name != nil, nil
}

// anyEmbeddingModel 任意一条向量模型（不限状态）
func (s *vectorSchemaService) anyEmbeddingModel() (*entity.AIModelConfig, error) {
	var m entity.AIModelConfig
	err := s.db.Where("type = ?", entity.ModelTypeEmbedding).First(&m).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// activeEmbeddingModel 启用中的向量模型
func (s *vectorSchemaService) activeEmbeddingModel() (*entity.AIModelConfig, error) {
	var m entity.AIModelConfig
	err := s.db.Where("type = ? AND status = ?", entity.ModelTypeEmbedding, 1).First(&m).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
