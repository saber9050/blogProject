package chunk

import (
	stderrors "errors"
	"time"

	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/articlechunk"
	"blog/pkg/errors"
	"blog/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// previewRunes 列表里 content 的预览长度。
//
// 截断是刻意的：一页 20 条 × 每块最多 900 字，不截断单个响应能到 30KB 以上，
// 列表页渲染也会明显卡顿。要看全文点开详情。
const previewRunes = 200

// maxPageSize 单页上限
const maxPageSize = 100

type chunkService struct {
	db   *gorm.DB
	repo repo.Repository
}

// NewChunkService 创建切分块查询服务
//
// 这里持 db 只为了「整篇删除」那一个写操作：删块要同时重置 articles 上的
// 向量化状态，两件事必须在一个事务里，而 repo 的写方法都要求调用方传 tx。
func NewChunkService(db *gorm.DB, r repo.Repository) Service {
	return &chunkService{db: db, repo: r}
}

// List 分页查询
func (s *chunkService) List(
	templateID, articleID uint, keyword string, page, pageSize int,
) (*response.ChunkListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	rows, total, err := s.repo.List(repo.Query{
		TemplateID: templateID,
		ArticleID:  articleID,
		Keyword:    keyword,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		logger.Error("查询切分块失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "查询切分块失败", err)
	}

	list := make([]*response.ChunkItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, toItem(r, true))
	}
	return &response.ChunkListResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     list,
	}, nil
}

// Get 单块详情
func (s *chunkService) Get(id uint) (*response.ChunkItem, error) {
	row, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("查询切分块详情失败", zap.Uint("chunk_id", id), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "查询切分块失败", err)
	}
	if row == nil {
		return nil, errors.New(errors.CodeNotFound, "切分块不存在")
	}
	return toItem(row, false), nil
}

// Stat 聚合统计
func (s *chunkService) Stat(groupBy string) (*response.ChunkStatResponse, error) {
	if groupBy == "" {
		groupBy = "template"
	}
	if groupBy != "template" && groupBy != "article" {
		return nil, errors.New(errors.CodeBadRequest, "group_by 只能是 template 或 article")
	}

	chunkTotal, articleTotal, err := s.repo.Summary()
	if err != nil {
		logger.Error("统计切分块总数失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "统计切分数据失败", err)
	}

	out := &response.ChunkStatResponse{GroupBy: groupBy}
	out.Summary = response.ChunkStatSummary{
		ChunkTotal:   chunkTotal,
		ArticleTotal: articleTotal,
	}
	if articleTotal > 0 {
		out.Summary.AvgChunkPerArticle = round1(float64(chunkTotal) / float64(articleTotal))
	}

	switch groupBy {
	case "template":
		rows, err := s.repo.StatByTemplate()
		if err != nil {
			logger.Error("按模板统计失败", zap.Error(err))
			return nil, errors.NewWithErr(errors.CodeInternalError, "按模板统计失败", err)
		}
		list := make([]*response.ChunkTemplateStat, 0, len(rows))
		for _, r := range rows {
			list = append(list, &response.ChunkTemplateStat{
				TemplateID:     r.TemplateID,
				TemplateName:   r.TemplateName,
				TemplateStatus: r.Status,
				ArticleCount:   r.ArticleCount,
				ChunkCount:     r.ChunkCount,
				AvgChars:       int(r.AvgChars + 0.5),
				LastChunkAt:    formatTime(r.LastChunkAt),
			})
		}
		out.List = list
	case "article":
		rows, err := s.repo.StatByArticle()
		if err != nil {
			logger.Error("按文章统计失败", zap.Error(err))
			return nil, errors.NewWithErr(errors.CodeInternalError, "按文章统计失败", err)
		}
		list := make([]*response.ChunkArticleStat, 0, len(rows))
		for _, r := range rows {
			list = append(list, &response.ChunkArticleStat{
				ArticleID:    r.ArticleID,
				ArticleTitle: r.ArticleTitle,
				TemplateID:   r.TemplateID,
				TemplateName: r.TemplateName,
				ChunkCount:   r.ChunkCount,
				AvgChars:     int(r.AvgChars + 0.5),
				LastChunkAt:  formatTime(r.LastChunkAt),
			})
		}
		out.List = list
	}
	return out, nil
}

// DeleteByArticle 删除整篇文章的向量
//
// 允许的场景很具体：切出来发现效果不对，想清掉重来。所以它必须同时做两件事——
// 删块、把文章状态退回「未处理」，少做任何一件都会让下一轮向量化判断错：
//   - 只删块不重置状态：文章仍显示「是（N 块）」，但库里一块都没有
//   - 只重置状态不删块：重新向量化时旧块还在，统计会翻倍
func (s *chunkService) DeleteByArticle(articleID uint) (*response.ChunkDeleteResponse, error) {
	var article entity.Article
	err := s.db.Select("id, vector_status").First(&article, articleID).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New(errors.CodeNotFound, "文章不存在")
	}
	if err != nil {
		logger.Error("读取文章失败", zap.Uint("article_id", articleID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取文章失败", err)
	}

	// 排队中/处理中必须挡住：worker 完成时会把整批块连同状态一起写回来，
	// 这时候删掉，用户会看到「删了又自己冒出来」，比报错更难理解。
	if article.VectorStatus == entity.VectorStatusQueued ||
		article.VectorStatus == entity.VectorStatusRunning {
		return nil, errors.New(errors.CodeConflict, "该文章正在向量化，请等处理完成后再删除")
	}

	var deleted int64
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		// 表可能还没建（没配过向量模型），此时没有块可删，只重置文章状态
		ready, err := s.repo.TableReady()
		if err != nil {
			return err
		}
		if ready {
			if err := tx.Model(&entity.ArticleChunk{}).
				Where("article_id = ?", articleID).
				Count(&deleted).Error; err != nil {
				return err
			}
			if err := s.repo.DeleteByArticle(tx, articleID); err != nil {
				return err
			}
		}
		return tx.Model(&entity.Article{}).Where("id = ?", articleID).
			Updates(map[string]interface{}{
				"vector_status":      entity.VectorStatusNone,
				"vector_chunk_count": 0,
				"vector_error":       "",
				"vectorized_at":      nil,
			}).Error
	})
	if txErr != nil {
		logger.Error("删除文章向量失败", zap.Uint("article_id", articleID), zap.Error(txErr))
		return nil, errors.NewWithErr(errors.CodeInternalError, "删除文章向量失败", txErr)
	}

	logger.Info("已删除文章向量",
		zap.Uint("article_id", articleID), zap.Int64("deleted_chunks", deleted))
	return &response.ChunkDeleteResponse{
		ArticleID:     articleID,
		DeletedChunks: deleted,
		VectorStatus:  entity.VectorStatusNone,
	}, nil
}

// toItem 行转 DTO。truncate=true 时 content 截断到 previewRunes。
func toItem(r *repo.ChunkRow, truncate bool) *response.ChunkItem {
	item := &response.ChunkItem{
		ID:             r.ID,
		ArticleID:      r.ArticleID,
		ArticleTitle:   r.ArticleTitle,
		TemplateID:     r.TemplateID,
		TemplateName:   r.TemplateName,
		ChunkIndex:     r.ChunkIndex,
		CharCount:      r.CharCount,
		Content:        r.Content,
		EmbeddingModel: r.EmbeddingModel,
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
	}
	if truncate {
		// 按 rune 截断，不能用切片下标——中文会被切成半个字符
		if runes := []rune(item.Content); len(runes) > previewRunes {
			item.Content = string(runes[:previewRunes])
			item.ContentTruncated = true
		}
	}
	return item
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// round1 保留一位小数
func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
