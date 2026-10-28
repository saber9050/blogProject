package chunktemplate

import (
	"fmt"
	"strings"
	"time"

	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/chunktemplate"
	"blog/pkg/errors"
	"blog/pkg/logger"

	"go.uber.org/zap"
)

type chunkTemplateService struct {
	repo repo.Repository
}

// NewChunkTemplateService 创建切分模板服务实例
func NewChunkTemplateService(r repo.Repository) Service {
	return &chunkTemplateService{repo: r}
}

// List 列表（带使用统计）
//
// 一次查全部统计再回填，而不是每条模板各查一次：模板数量是个位数，
// 聚合一次比 N 次往返便宜得多，也不会在模板变多后悄悄退化成 N+1。
func (s *chunkTemplateService) List(status *int8) ([]*response.ChunkTemplateResponse, error) {
	list, err := s.repo.List(status)
	if err != nil {
		logger.Error("获取切分模板列表失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取切分模板列表失败", err)
	}
	stats, err := s.repo.Stats()
	if err != nil {
		// 统计失败不该让整页打不开，退化成全 0，至少列表能看
		logger.Warn("统计模板使用情况失败，使用数按 0 展示", zap.Error(err))
		stats = map[uint]repo.TemplateStat{}
	}

	result := make([]*response.ChunkTemplateResponse, 0, len(list))
	for _, t := range list {
		result = append(result, toResponse(t, stats[t.ID]))
	}
	return result, nil
}

// Get 详情
func (s *chunkTemplateService) Get(id uint) (*response.ChunkTemplateResponse, error) {
	t, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("获取切分模板失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取切分模板失败", err)
	}
	if t == nil {
		return nil, errors.New(errors.CodeNotFound, "切分模板不存在")
	}
	stats, err := s.repo.Stats()
	if err != nil {
		stats = map[uint]repo.TemplateStat{}
	}
	return toResponse(t, stats[id]), nil
}

// Create 新建
func (s *chunkTemplateService) Create(req *request.CreateChunkTemplateRequest) (*response.ChunkTemplateResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New(errors.CodeBadRequest, "模板名称不能为空")
	}
	overlap := req.ChunkOverlap
	minSize := req.MinChunkSize
	if err := validateParams(req.ChunkSize, overlap, minSize); err != nil {
		return nil, err
	}

	exists, err := s.repo.ExistsByName(name, 0)
	if err != nil {
		logger.Error("检查模板名称失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "检查模板名称失败", err)
	}
	if exists {
		return nil, errors.New(errors.CodeBadRequest, "模板名称已存在")
	}

	stripHTML := true
	if req.StripHTML != nil {
		stripHTML = *req.StripHTML
	}
	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}

	t := &entity.ChunkTemplate{
		Name:         name,
		Description:  strings.TrimSpace(req.Description),
		ChunkSize:    req.ChunkSize,
		ChunkOverlap: overlap,
		MinChunkSize: minSize,
		StripHTML:    stripHTML,
		Status:       status,
	}
	if err := s.repo.Create(t); err != nil {
		logger.Error("创建切分模板失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "创建切分模板失败", err)
	}
	return toResponse(t, repo.TemplateStat{}), nil
}

// Update 更新
//
// 两个约束：
//   - 名称可改（但必须唯一）
//   - 切分参数一旦产生过块就锁死。改了参数，库里已存的块与新参数不再对应，
//     既无法解释也无法修复，只能让所有文章重新向量化——那不该是「改个数字」的副作用。
func (s *chunkTemplateService) Update(id uint, req *request.UpdateChunkTemplateRequest) error {
	t, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("读取待更新模板失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "读取切分模板失败", err)
	}
	if t == nil {
		return errors.New(errors.CodeNotFound, "切分模板不存在")
	}

	chunkCount, err := s.repo.ChunkCount(id)
	if err != nil {
		logger.Error("统计模板块数失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "统计模板使用情况失败", err)
	}

	// 取出「生效后」的参数，用它们做完整校验——只校验改动的字段会漏掉
	// 「改大了 chunk_size 导致原有的 overlap 超限」这种组合问题
	size := t.ChunkSize
	if req.ChunkSize != nil {
		size = *req.ChunkSize
	}
	overlap := t.ChunkOverlap
	if req.ChunkOverlap != nil {
		overlap = *req.ChunkOverlap
	}
	minSize := t.MinChunkSize
	if req.MinChunkSize != nil {
		minSize = *req.MinChunkSize
	}

	if chunkCount > 0 {
		changed := (req.ChunkSize != nil && *req.ChunkSize != t.ChunkSize) ||
			(req.ChunkOverlap != nil && *req.ChunkOverlap != t.ChunkOverlap) ||
			(req.MinChunkSize != nil && *req.MinChunkSize != t.MinChunkSize)
		if changed {
			stat, _ := s.repo.Stats()
			articles := stat[id].ArticleCount
			return errors.New(errors.CodeBadRequest, fmt.Sprintf(
				"该模板已被 %d 篇文章使用（共 %d 块），切分参数不可修改；如需调整请新建模板",
				articles, chunkCount))
		}
	}

	if err := validateParams(size, overlap, minSize); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if name := strings.TrimSpace(req.Name); name != "" && name != t.Name {
		exists, err := s.repo.ExistsByName(name, id)
		if err != nil {
			logger.Error("检查模板名称失败", zap.Error(err))
			return errors.NewWithErr(errors.CodeInternalError, "检查模板名称失败", err)
		}
		if exists {
			return errors.New(errors.CodeBadRequest, "模板名称已存在")
		}
		fields["name"] = name
	}
	if req.Description != "" {
		fields["description"] = strings.TrimSpace(req.Description)
	}
	if req.ChunkSize != nil {
		fields["chunk_size"] = *req.ChunkSize
	}
	if req.ChunkOverlap != nil {
		fields["chunk_overlap"] = *req.ChunkOverlap
	}
	if req.MinChunkSize != nil {
		fields["min_chunk_size"] = *req.MinChunkSize
	}
	if req.StripHTML != nil {
		fields["strip_html"] = *req.StripHTML
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if len(fields) == 0 {
		return nil
	}

	if err := s.repo.UpdateFields(id, fields); err != nil {
		logger.Error("更新切分模板失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "更新切分模板失败", err)
	}
	return nil
}

// Delete 删除。已被使用的模板不允许删除。
//
// 不做级联删除：删模板顺手删块，会造成「文章显示已完成但查不到任何块」的
// 不一致状态，而且块是花 API 调用换来的，不该被一次误点清掉。
func (s *chunkTemplateService) Delete(id uint) error {
	t, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("读取待删除模板失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "读取切分模板失败", err)
	}
	if t == nil {
		return errors.New(errors.CodeNotFound, "切分模板不存在")
	}

	chunkCount, err := s.repo.ChunkCount(id)
	if err != nil {
		logger.Error("统计模板块数失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "统计模板使用情况失败", err)
	}
	if chunkCount > 0 {
		stat, _ := s.repo.Stats()
		return errors.New(errors.CodeConflict, fmt.Sprintf(
			"该模板已被 %d 篇文章使用（共 %d 块），无法删除；请先将相关文章改用其他模板重新向量化",
			stat[id].ArticleCount, chunkCount))
	}

	if err := s.repo.Delete(id); err != nil {
		logger.Error("删除切分模板失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "删除切分模板失败", err)
	}
	return nil
}

// validateParams 参数校验。上下限来自 entity，与切分算法共用同一组常量。
func validateParams(size, overlap, minSize int) error {
	if size < entity.MinChunkSizeLimit || size > entity.MaxChunkSizeLimit {
		return errors.New(errors.CodeBadRequest, fmt.Sprintf(
			"块大小需在 %d~%d 之间，当前填写 %d",
			entity.MinChunkSizeLimit, entity.MaxChunkSizeLimit, size))
	}
	if overlap < 0 || overlap > size/2 {
		return errors.New(errors.CodeBadRequest, fmt.Sprintf(
			"重叠长度不能超过块大小的一半（当前 %d > %d）", overlap, size/2))
	}
	if minSize < 0 || minSize > size {
		return errors.New(errors.CodeBadRequest, "碎块阈值不能超过块大小")
	}
	return nil
}

func toResponse(t *entity.ChunkTemplate, stat repo.TemplateStat) *response.ChunkTemplateResponse {
	return &response.ChunkTemplateResponse{
		ID:               t.ID,
		Name:             t.Name,
		Description:      t.Description,
		ChunkSize:        t.ChunkSize,
		ChunkOverlap:     t.ChunkOverlap,
		MinChunkSize:     t.MinChunkSize,
		StripHTML:        t.StripHTML,
		Status:           t.Status,
		UsedArticleCount: stat.ArticleCount,
		ChunkCount:       stat.ChunkCount,
		CreatedAt:        t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        t.UpdatedAt.Format(time.RFC3339),
	}
}
