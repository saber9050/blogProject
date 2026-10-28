package llmconfig

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/llmconfig"
	"blog/pkg/errors"
	"blog/pkg/llmclient"
	"blog/pkg/logger"

	"go.uber.org/zap"
)

const (
	defaultTimeoutSec  = 30
	defaultMaxTokens   = 2048
	defaultTemperature = 0.3
	testTimeoutCapSec  = 15 // 测试连接超时上限，避免拖太久
	testMaxTokens      = 16 // 测试连接只生成极少量 token
)

// llmConfigService 模型配置服务实现
type llmConfigService struct {
	repo    repo.Repository
	ensurer SchemaEnsurer
}

// NewLLMConfigService 创建模型配置服务实例。
//
// ensurer 允许为 nil（P4 之前没有实现），此时新建向量模型只会校验维度，
// 不会去建 article_chunks 表——那种情况下向量化功能还不可用，符合预期。
func NewLLMConfigService(r repo.Repository, ensurer SchemaEnsurer) Service {
	return &llmConfigService{repo: r, ensurer: ensurer}
}

// List 配置列表（密钥脱敏）
func (s *llmConfigService) List() ([]*response.LLMConfigResponse, error) {
	return s.ListByType(0)
}

// ListByType 按类型筛选；t=0 返回全部
func (s *llmConfigService) ListByType(t int8) ([]*response.LLMConfigResponse, error) {
	list, err := s.repo.ListByType(t)
	if err != nil {
		logger.Error("获取模型配置列表失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取模型配置列表失败", err)
	}
	result := make([]*response.LLMConfigResponse, 0, len(list))
	for _, m := range list {
		result = append(result, toResponse(m))
	}
	return result, nil
}

// Get 配置详情（密钥脱敏）
func (s *llmConfigService) Get(id uint) (*response.LLMConfigResponse, error) {
	m, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("获取模型配置失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取模型配置失败", err)
	}
	if m == nil {
		return nil, errors.New(errors.CodeNotFound, "配置不存在")
	}
	return toResponse(m), nil
}

// Test 测试连接（不落库）
//
// 向量模型和对话模型打的是不同接口，所以先看类型再决定测哪个。
// 向量模型还会把实际输出的维度带回去——用户填错维度时能当场看出来，
// 而不是等到第一次向量化才失败。
func (s *llmConfigService) Test(req *request.TestLLMConfigRequest) (*response.TestConnectionResponse, error) {
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" && req.ID > 0 {
		m, err := s.repo.FindByID(req.ID)
		if err != nil {
			logger.Error("读取待测配置失败", zap.Error(err))
			return nil, errors.NewWithErr(errors.CodeInternalError, "读取配置失败", err)
		}
		if m == nil {
			return nil, errors.New(errors.CodeNotFound, "配置不存在")
		}
		apiKey = m.APIKey
	}
	if apiKey == "" {
		return &response.TestConnectionResponse{OK: false, Error: "请填写 api_key"}, nil
	}

	cfg := llmclient.Config{
		BaseURL:     strings.TrimSpace(req.BaseURL),
		APIKey:      apiKey,
		Model:       strings.TrimSpace(req.Model),
		TimeoutSec:  req.TimeoutSec,
		MaxTokens:   defaultMaxTokens,
		Temperature: defaultTemperature,
	}

	if req.Type == request.TypeEmbedding {
		dim, latency, err := s.testEmbedding(cfg)
		if err != nil {
			return &response.TestConnectionResponse{OK: false, LatencyMs: latency, Error: friendlyErr(err)}, nil
		}
		return &response.TestConnectionResponse{OK: true, LatencyMs: latency, Dimension: dim}, nil
	}

	reply, latency, err := s.testConn(cfg)
	if err != nil {
		return &response.TestConnectionResponse{OK: false, LatencyMs: latency, Error: friendlyErr(err)}, nil
	}
	return &response.TestConnectionResponse{OK: true, LatencyMs: latency, Reply: reply}, nil
}

// Create 新建配置
//
// 顺序是「校验 → 建表 → 落库」，不能颠倒：
// 先 Insert 再校验的话，校验失败时库里会留下一条「建了模型但用不了」的脏记录，
// 还得回滚删除，多一次出错机会。
func (s *llmConfigService) Create(req *request.CreateLLMConfigRequest) (*response.LLMConfigResponse, error) {
	modelType := req.Type
	if modelType == 0 {
		modelType = entity.ModelTypeChat
	}

	cfg := llmclient.Config{
		BaseURL:     strings.TrimSpace(req.BaseURL),
		APIKey:      strings.TrimSpace(req.APIKey),
		Model:       strings.TrimSpace(req.Model),
		TimeoutSec:  withDefaultInt(req.TimeoutSec, defaultTimeoutSec),
		MaxTokens:   withDefaultInt(req.MaxTokens, defaultMaxTokens),
		Temperature: withDefaultFloat(req.Temperature, defaultTemperature),
	}

	dimension := 0
	if modelType == entity.ModelTypeEmbedding {
		var err error
		if dimension, err = s.prepareEmbedding(req, cfg); err != nil {
			return nil, err
		}
	} else {
		// 对话模型：入库门禁，测试连接不通过则拒绝入库
		if _, _, err := s.testConn(cfg); err != nil {
			return nil, errors.New(errors.CodeBadRequest, "配置不可用，测试连接失败："+friendlyErr(err))
		}
	}

	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}

	m := &entity.AIModelConfig{
		Name:        cfg.Model, // 配置名称不再单独配置，自动取模型 id
		BaseURL:     cfg.BaseURL,
		APIKey:      cfg.APIKey,
		Model:       cfg.Model,
		Type:        modelType,
		Dimension:   dimension,
		TimeoutSec:  cfg.TimeoutSec,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Status:      status,
	}

	if err := s.repo.Create(m); err != nil {
		logger.Error("创建模型配置失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "创建模型配置失败", err)
	}

	return toResponse(m), nil
}

// prepareEmbedding 向量模型的入库前校验，返回校验通过的维度。
//
// 四步都要过：
//  1. 唯一性——全局只允许一条向量模型
//  2. 维度范围——太小无意义，太大浪费存储（4096 维 float4 = 16KB/块）
//  3. 连真实服务测一次，把实际输出维度与用户填的比对
//  4. 向量表维度必须与之一致（表已存在时）
func (s *llmConfigService) prepareEmbedding(req *request.CreateLLMConfigRequest, cfg llmclient.Config) (int, error) {
	n, err := s.repo.CountByType(entity.ModelTypeEmbedding, 0)
	if err != nil {
		logger.Error("统计向量模型数量失败", zap.Error(err))
		return 0, errors.NewWithErr(errors.CodeInternalError, "校验向量模型唯一性失败", err)
	}
	if n > 0 {
		// 给出已存在的那条，让用户知道该删哪个。可能已被禁用，所以不用 FindActiveEmbedding
		list, _ := s.repo.ListByType(entity.ModelTypeEmbedding)
		hint := ""
		if len(list) > 0 {
			hint = fmt.Sprintf("（%s, id=%d）", list[0].Model, list[0].ID)
		}
		return 0, errors.New(errors.CodeBadRequest,
			"已存在向量模型"+hint+"，全局只允许一条，请先删除或改为对话模型")
	}

	if req.Dimension < entity.MinEmbeddingDimension || req.Dimension > entity.MaxEmbeddingDimension {
		return 0, errors.New(errors.CodeBadRequest,
			fmt.Sprintf("向量维度需在 %d~%d 之间，当前填写 %d",
				entity.MinEmbeddingDimension, entity.MaxEmbeddingDimension, req.Dimension))
	}

	actualDim, _, err := s.testEmbedding(cfg)
	if err != nil {
		return 0, errors.New(errors.CodeBadRequest, "配置不可用，测试连接失败："+friendlyErr(err))
	}
	if actualDim != req.Dimension {
		return 0, errors.New(errors.CodeBadRequest,
			fmt.Sprintf("模型实际输出 %d 维，与你填写的 %d 不一致；请核对模型的向量维度后重填",
				actualDim, req.Dimension))
	}

	if s.ensurer == nil {
		// P4 之前没有向量表维护服务；此时只校验不建表，向量化功能尚不可用
		logger.Warn("向量表维护未接入，跳过 article_chunks 初始化",
			zap.Int("dimension", req.Dimension))
		return req.Dimension, nil
	}
	if err := s.ensurer.EnsureFor(req.Dimension); err != nil {
		// EnsureFor 的维度冲突信息本身就是完整的句子（含当前维度、块数、如何切换）
		return 0, errors.New(errors.CodeBadRequest, err.Error())
	}
	return req.Dimension, nil
}

// Update 更新配置（仅修改运行参数与状态，不测试连接）
func (s *llmConfigService) Update(id uint, req *request.UpdateLLMConfigRequest) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("读取待更新配置失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "读取配置失败", err)
	}
	if existing == nil {
		return errors.New(errors.CodeNotFound, "配置不存在")
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	fields := map[string]interface{}{
		"timeout_sec": withDefaultInt(req.TimeoutSec, defaultTimeoutSec),
		"max_tokens":  withDefaultInt(req.MaxTokens, defaultMaxTokens),
		"temperature": withDefaultFloat(req.Temperature, defaultTemperature),
		"status":      status,
	}

	if err := s.repo.UpdateFields(id, fields); err != nil {
		logger.Error("更新模型配置失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "更新模型配置失败", err)
	}
	return nil
}

// Delete 删除配置
func (s *llmConfigService) Delete(id uint) error {
	m, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("读取待删除配置失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "读取配置失败", err)
	}
	if m == nil {
		return errors.New(errors.CodeNotFound, "配置不存在")
	}
	if err := s.repo.Delete(id); err != nil {
		logger.Error("删除模型配置失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "删除模型配置失败", err)
	}
	return nil
}

// Resolve 解析指定配置（必须是启用的；未找到或已禁用则报错）
func (s *llmConfigService) Resolve(id uint) (llmclient.Config, error) {
	m, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("读取模型配置失败", zap.Error(err))
		return llmclient.Config{}, errors.NewWithErr(errors.CodeInternalError, "读取模型配置失败", err)
	}
	if m == nil {
		return llmclient.Config{}, errors.New(errors.CodeNotFound, "配置不存在")
	}
	if m.Status != 1 {
		return llmclient.Config{}, errors.New(errors.CodeBadRequest, "该模型配置已禁用，请先在「AI 模型」页启用")
	}
	return s.toClientConfig(m), nil
}

// ResolveEmbedding 解析当前启用的向量模型，同时返回它的维度。
//
// 维度一并返回是刻意的：调用方（向量化任务）必须拿它去校验模型实际输出，
// 让调用方再查一次库既多余又容易忘。不需要传 configID——全局只有一条。
func (s *llmConfigService) ResolveEmbedding() (llmclient.Config, int, error) {
	m, err := s.repo.FindActiveEmbedding()
	if err != nil {
		logger.Error("读取向量模型失败", zap.Error(err))
		return llmclient.Config{}, 0, errors.NewWithErr(errors.CodeInternalError, "读取向量模型失败", err)
	}
	if m == nil {
		// 区分「从没配过」和「配了但被禁用」：两种情况下用户的下一步动作完全不同，
		// 返回同一句话会让人在「AI 模型」页里对着一条已有的配置发懵。
		if list, lerr := s.repo.ListByType(entity.ModelTypeEmbedding); lerr == nil && len(list) > 0 {
			return llmclient.Config{}, 0, errors.New(errors.CodeServiceUnavailable,
				fmt.Sprintf("向量模型（%s）已禁用，请先在「AI 模型」页启用", list[0].Model))
		}
		return llmclient.Config{}, 0, errors.New(errors.CodeServiceUnavailable,
			"尚未配置可用的向量模型，请先在「AI 模型」页添加 type=向量模型 的配置")
	}
	if m.Dimension <= 0 {
		return llmclient.Config{}, 0, errors.New(errors.CodeServiceUnavailable,
			"向量模型缺少维度配置，请删除后重新添加")
	}
	return s.toClientConfig(m), m.Dimension, nil
}

// testConn 用最小代价验证对话连通性，返回回复文本与耗时
func (s *llmConfigService) testConn(cfg llmclient.Config) (string, int64, error) {
	if cfg.TimeoutSec <= 0 || cfg.TimeoutSec > testTimeoutCapSec {
		cfg.TimeoutSec = testTimeoutCapSec
	}
	cfg.MaxTokens = testMaxTokens

	client := llmclient.NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	start := time.Now()
	reply, err := client.Chat(ctx, []llmclient.Message{{Role: "user", Content: "ping"}})
	return strings.TrimSpace(reply), time.Since(start).Milliseconds(), err
}

// testEmbedding 验证向量模型连通性，返回实际输出维度与耗时。
//
// 用一句 "ping" 而不是空串：部分服务对空输入直接返回 400，
// 那样会把「服务可用但输入不合法」误判成「配置有误」。
func (s *llmConfigService) testEmbedding(cfg llmclient.Config) (int, int64, error) {
	if cfg.TimeoutSec <= 0 || cfg.TimeoutSec > testTimeoutCapSec {
		cfg.TimeoutSec = testTimeoutCapSec
	}

	client := llmclient.NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	start := time.Now()
	vecs, err := client.Embed(ctx, []string{"ping"})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return 0, latency, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return 0, latency, errors.New(errors.CodeInternalError, "向量模型返回空向量")
	}
	return len(vecs[0]), latency, nil
}

// toClientConfig 实体转客户端配置
func (s *llmConfigService) toClientConfig(m *entity.AIModelConfig) llmclient.Config {
	return llmclient.Config{
		BaseURL:     m.BaseURL,
		APIKey:      m.APIKey,
		Model:       m.Model,
		TimeoutSec:  withDefaultInt(m.TimeoutSec, defaultTimeoutSec),
		MaxTokens:   withDefaultInt(m.MaxTokens, defaultMaxTokens),
		Temperature: withDefaultFloat(m.Temperature, defaultTemperature),
	}
}

// toResponse 实体转响应（密钥脱敏）
func toResponse(m *entity.AIModelConfig) *response.LLMConfigResponse {
	return &response.LLMConfigResponse{
		ID:           m.ID,
		BaseURL:      m.BaseURL,
		APIKeyMasked: maskKey(m.APIKey),
		Model:        m.Model,
		Type:         m.Type,
		Dimension:    m.Dimension,
		TimeoutSec:   m.TimeoutSec,
		MaxTokens:    m.MaxTokens,
		Temperature:  m.Temperature,
		Status:       m.Status,
		CreatedAt:    m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    m.UpdatedAt.Format(time.RFC3339),
	}
}

// maskKey 密钥脱敏：sk-xxxxxxxxa1b2 → sk-****a1b2
func maskKey(k string) string {
	if k == "" {
		return ""
	}
	r := []rune(k)
	if len(r) <= 4 {
		return "****"
	}
	prefix := r
	if len(r) > 3 {
		prefix = r[:3]
	}
	return string(prefix) + "****" + string(r[len(r)-4:])
}

func withDefaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func withDefaultFloat(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

// friendlyErr 把底层错误转成对用户友好的提示（不含供应商原始响应体）
func friendlyErr(err error) string {
	var be *errors.BizError
	if stderrors.As(err, &be) {
		if be.Err != nil {
			msg := strings.ToLower(be.Err.Error())
			if stderrors.Is(be.Err, context.DeadlineExceeded) ||
				strings.Contains(msg, "deadline exceeded") ||
				strings.Contains(msg, "timeout") {
				return "连接超时，请检查 base_url 与网络"
			}
		}
		switch {
		case strings.Contains(be.Message, "401"), strings.Contains(be.Message, "403"):
			return "鉴权失败，请检查 api_key"
		case strings.Contains(be.Message, "404"):
			return "接口或模型不存在，请检查 base_url 与 model"
		case strings.Contains(be.Message, "429"):
			return "触发服务商限流，请稍后重试"
		}
		return be.Message
	}
	return err.Error()
}
