package vectorize

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"
	"unicode/utf8"

	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	articleRepo "blog/internal/repository/article"
	articlechunkRepo "blog/internal/repository/articlechunk"
	"blog/internal/service/chunker"
	llmconfigSvc "blog/internal/service/llmconfig"
	vectorSchemaSvc "blog/internal/service/vectorschema"
	"blog/pkg/errors"
	"blog/pkg/llmclient"
	"blog/pkg/logger"

	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// queueCapacity 有界队列。无界的话批量提交会把内存吃光——每篇文章的
	// 全部块向量要同时留在内存里等事务写入。满了直接拒绝，比 OOM 好。
	queueCapacity = 64

	// embeddingBatch 每批送多少条文本去 /embeddings。
	// 太小则请求数爆炸，太大则单次容易超时或被服务端限流。
	embeddingBatch = 10

	// taskTimeout 单篇硬超时。长文 30 块 × 每批 1~2 秒，正常远用不到这个数，
	// 它存在是为了兜住「embedding 服务 hang 住」这种情况，别让 worker 卡死一行。
	taskTimeout = 10 * time.Minute

	// maxErrorLen 写进 vector_error 的最大长度（列是 varchar(255)）
	maxErrorLen = 255
)

// task 一个待处理的向量化任务
//
// 只有文章 id：切分参数是全局固定的，任务里不带任何「怎么切」的信息。
type task struct {
	articleID uint
}

type vectorizeService struct {
	db        *gorm.DB
	articleRp articleRepo.ArticleRepository
	chunkRepo articlechunkRepo.Repository
	llmCfgSvc llmconfigSvc.Service
	schemaSvc vectorSchemaSvc.Service
	queue     chan task
}

// NewVectorizeService 创建向量化服务实例
func NewVectorizeService(
	db *gorm.DB,
	articleRp articleRepo.ArticleRepository,
	chunkRepo articlechunkRepo.Repository,
	llmCfgSvc llmconfigSvc.Service,
	schemaSvc vectorSchemaSvc.Service,
) Service {
	return &vectorizeService{
		db:        db,
		articleRp: articleRp,
		chunkRepo: chunkRepo,
		llmCfgSvc: llmCfgSvc,
		schemaSvc: schemaSvc,
		queue:     make(chan task, queueCapacity),
	}
}

// Submit 提交向量化任务
//
// 顺序：校验 → 抢锁 → 投递。抢锁用的是条件 UPDATE，
// 靠 RowsAffected 判断，不做「先查后改」——那样并发点两次会都通过检查。
func (s *vectorizeService) Submit(articleID uint) (*response.VectorizeSubmitResponse, error) {
	article, err := s.articleRp.FindByID(articleID)
	if err != nil {
		logger.Error("读取文章失败", zap.Uint("article_id", articleID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取文章失败", err)
	}
	if article == nil {
		return nil, errors.New(errors.CodeNotFound, "文章不存在")
	}
	// 草稿也允许向量化：这是「发布前先看看切分效果对不对」的正常诉求。
	// 之前挡草稿的理由是「草稿还在改，切了也是白切」，但切分管理能让作者
	// 直接看到分块原文，提前验证恰恰是它最大的用处。
	// 代价是库里会存在「未发布文章的向量」，所以将来做检索时必须过滤
	// articles.status = 1，不能只查 article_chunks。
	// 提前判空、判过短，别等到进了队列才失败——用户点完就该知道结果。
	// 判断口径与 worker 里的 chunker.Split 保持一致（同一个 MinContentChars），
	// 否则会出现「提交成功、排队几秒后失败」这种让人困惑的体验。
	plain := chunker.ExtractText(article.Content, true)
	if plain == "" {
		return nil, errors.New(errors.CodeBadRequest, "文章正文为空，无法向量化")
	}
	if n := utf8.RuneCountInString(plain); n < chunker.MinContentChars {
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf(
			"文章正文过少（当前 %d 字，至少 %d 字），无法向量化", n, chunker.MinContentChars))
	}

	// 提前确认向量模型可用，避免任务排队半天最后因为没配模型失败
	if _, _, err := s.llmCfgSvc.ResolveEmbedding(); err != nil {
		return nil, err
	}

	// 抢锁：排队中(1)和处理中(2)都算已占用
	res := s.db.Model(&entity.Article{}).
		Where("id = ? AND vector_status NOT IN ?", articleID,
			[]int8{entity.VectorStatusQueued, entity.VectorStatusRunning}).
		Updates(map[string]interface{}{
			"vector_status": entity.VectorStatusQueued,
			"vector_error":  "",
		})
	if res.Error != nil {
		logger.Error("提交向量化任务失败", zap.Uint("article_id", articleID), zap.Error(res.Error))
		return nil, errors.NewWithErr(errors.CodeInternalError, "提交向量化任务失败", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, errors.New(errors.CodeConflict, "该文章正在排队或处理中，请等待完成")
	}

	select {
	case s.queue <- task{articleID: articleID}:
	default:
		// 队列满：把状态退回去。带 vector_status=1 条件，
		// 否则可能覆盖掉刚刚被 worker 改成 2 的状态
		s.db.Model(&entity.Article{}).
			Where("id = ? AND vector_status = ?", articleID, entity.VectorStatusQueued).
			Update("vector_status", entity.VectorStatusNone)
		return nil, errors.New(errors.CodeServiceUnavailable, "任务队列已满，请稍后重试")
	}

	return &response.VectorizeSubmitResponse{
		ArticleID:    articleID,
		VectorStatus: entity.VectorStatusQueued,
	}, nil
}

// Status 查询向量化状态
func (s *vectorizeService) Status(articleID uint) (*response.VectorStatusResponse, error) {
	var a entity.Article
	err := s.db.Select(
		"id, vector_status, vector_chunk_count, vector_error, vectorized_at, updated_at").
		First(&a, articleID).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errors.CodeNotFound, "文章不存在")
		}
		logger.Error("查询向量化状态失败", zap.Uint("article_id", articleID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "查询向量化状态失败", err)
	}

	out := &response.VectorStatusResponse{
		ArticleID:        a.ID,
		VectorStatus:     a.VectorStatus,
		VectorChunkCount: a.VectorChunkCount,
		VectorError:      a.VectorError,
		UpdatedAt:        a.UpdatedAt.Format(time.RFC3339),
	}
	if a.VectorizedAt != nil {
		out.VectorizedAt = a.VectorizedAt.Format(time.RFC3339)
	}
	return out, nil
}

// RecoverZombies 重置进程重启遗留的任务
//
// 两个状态都要重置：
//   - 排队中(1)：队列在内存里，重启后队列空了，这些任务永久丢失
//   - 处理中(2)：正在跑的 goroutine 被杀，事务会回滚但状态留在 2
//
// 不重置的话，抢锁条件是 NOT IN (1,2)，这些文章会永远卡住且无法重新提交。
func (s *vectorizeService) RecoverZombies() (int64, error) {
	res := s.db.Model(&entity.Article{}).
		Where("vector_status IN ?", []int8{entity.VectorStatusQueued, entity.VectorStatusRunning}).
		Updates(map[string]interface{}{
			"vector_status": entity.VectorStatusFailed,
			"vector_error":  "进程重启中断，请重新提交",
		})
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected > 0 {
		logger.Warn("已重置进程重启遗留的向量化任务", zap.Int64("count", res.RowsAffected))
	}
	return res.RowsAffected, nil
}

// Start 启动 worker
func (s *vectorizeService) Start(ctx context.Context) {
	go s.loop(ctx)
	logger.Info("向量化 worker 已启动", zap.Int("queue_capacity", queueCapacity))
}

// loop 单 worker 串行消费
//
// 串行而不是并发：embedding 服务有速率限制，且串行天然避开了「同一篇文章被
// 两个 goroutine 同时写」的问题。当前量级（几十篇文章）排队几十秒完全可接受。
func (s *vectorizeService) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logger.Info("向量化 worker 退出")
			return
		case t := <-s.queue:
			s.process(ctx, t)
		}
	}
}

// process 处理单个任务
func (s *vectorizeService) process(parent context.Context, t task) {
	// 出队后立刻置「处理中」，带状态条件：从入队到出队之间文章可能被删、
	// 或被 reset 清空了状态，那样就不该再把它推进到 2
	res := s.db.Model(&entity.Article{}).
		Where("id = ? AND vector_status = ?", t.articleID, entity.VectorStatusQueued).
		Update("vector_status", entity.VectorStatusRunning)
	if res.Error != nil {
		logger.Error("更新任务状态失败", zap.Uint("article_id", t.articleID), zap.Error(res.Error))
		return
	}
	if res.RowsAffected == 0 {
		logger.Info("任务状态已变化，跳过", zap.Uint("article_id", t.articleID))
		return
	}

	ctx, cancel := context.WithTimeout(parent, taskTimeout)
	defer cancel()

	start := time.Now()
	if err := s.doVectorize(ctx, t); err != nil {
		s.markFailed(t.articleID, err)
		return
	}
	logger.Info("文章向量化完成",
		zap.Uint("article_id", t.articleID),
		zap.Duration("cost", time.Since(start)))
}

// doVectorize 真正的处理流程
func (s *vectorizeService) doVectorize(ctx context.Context, t task) error {
	article, err := s.articleRp.FindByID(t.articleID)
	if err != nil {
		return fmt.Errorf("读取文章失败: %w", err)
	}
	if article == nil {
		return stderrors.New("文章已被删除")
	}

	cfg, dim, err := s.llmCfgSvc.ResolveEmbedding()
	if err != nil {
		return err
	}

	// 表可能刚被 /vector/reset 删掉，这里补一次。平时是幂等的（维度一致直接返回）
	if err := s.schemaSvc.EnsureFor(dim); err != nil {
		return err
	}

	chunks, err := chunker.Split(article.Content, article.Title, chunker.DefaultParams())
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return stderrors.New("切分结果为空")
	}

	vectors, err := s.embedAll(ctx, cfg, dim, chunks)
	if err != nil {
		return err
	}

	rows := make([]*entity.ArticleChunk, 0, len(chunks))
	for i, c := range chunks {
		rows = append(rows, &entity.ArticleChunk{
			ArticleID:    t.articleID,
			SectionIndex: c.SectionIndex,
			HeadingPath:  c.HeadingPath,
			// TemplateID 留零值：模板概念已下线，这一列随 DDL 一起等下一轮清理。
			// 不写死 0 而是不赋值，是为了让「这里没这个概念」在代码里也是一致的。
			ChunkIndex:     i,
			Content:        c.Content,
			CharCount:      c.CharCount,
			EmbeddingModel: cfg.Model,
			Embedding:      pgvector.NewVector(vectors[i]),
		})
	}

	// 一个事务里完成「删旧块 + 插新块 + 回写文章状态」。
	// 这正是当初把主库迁到 PG 的收益：跨库做不到这件事。
	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.chunkRepo.DeleteByArticle(tx, t.articleID); err != nil {
			return fmt.Errorf("删除旧块失败: %w", err)
		}
		if err := s.chunkRepo.BatchCreate(tx, rows); err != nil {
			return fmt.Errorf("写入新块失败: %w", err)
		}
		return tx.Model(&entity.Article{}).Where("id = ?", t.articleID).
			Updates(map[string]interface{}{
				"vector_status":      entity.VectorStatusDone,
				"vector_chunk_count": len(rows),
				"vector_error":       "",
				"vectorized_at":      now,
			}).Error
	})
}

// embedAll 分批调 embedding，返回与 chunks 一一对应的向量
//
// 全部拿到向量之后才开事务写库：把外部 HTTP 调用放进事务里会长时间持有
// PG 连接和行锁。内存开销可以忽略（30 块 × 1024 维 × 4B ≈ 120KB）。
func (s *vectorizeService) embedAll(
	ctx context.Context, cfg llmclient.Config, dim int, chunks []chunker.Result,
) ([][]float32, error) {
	client := llmclient.NewClient(cfg)
	out := make([][]float32, len(chunks))

	for i := 0; i < len(chunks); i += embeddingBatch {
		end := i + embeddingBatch
		if end > len(chunks) {
			end = len(chunks)
		}

		texts := make([]string, 0, end-i)
		for _, c := range chunks[i:end] {
			texts = append(texts, c.Content)
		}

		vecs, err := client.Embed(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("第 %d~%d 块向量化失败: %w", i, end-1, err)
		}
		for j, v := range vecs {
			// 维度校验必须在这里做：PG 报的是 "expected 1024 dimensions, not 768"，
			// 信息生硬且不指出根因（是模型配错了，不是数据坏了）
			if len(v) != dim {
				return nil, fmt.Errorf("模型输出 %d 维，与向量表要求的 %d 维不一致；请核对向量模型配置", len(v), dim)
			}
			out[i+j] = v
		}
	}
	return out, nil
}

// markFailed 标记失败并记录原因
func (s *vectorizeService) markFailed(articleID uint, cause error) {
	msg := cause.Error()
	if r := []rune(msg); len(r) > maxErrorLen {
		msg = string(r[:maxErrorLen])
	}
	res := s.db.Model(&entity.Article{}).Where("id = ?", articleID).
		Updates(map[string]interface{}{
			"vector_status": entity.VectorStatusFailed,
			"vector_error":  msg,
		})
	if res.Error != nil {
		logger.Error("回写失败状态失败",
			zap.Uint("article_id", articleID), zap.Error(res.Error))
	}
	logger.Warn("文章向量化失败",
		zap.Uint("article_id", articleID), zap.String("reason", msg))
}
