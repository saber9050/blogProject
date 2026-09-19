package llmconfig

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/llmconfig"
	"blog/pkg/config"
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
	repo   repo.Repository
	llmCfg config.LLMConfig // 兜底配置（config.yaml 的 llm 段）
}

// NewLLMConfigService 创建模型配置服务实例
func NewLLMConfigService(r repo.Repository, llmCfg config.LLMConfig) Service {
	return &llmConfigService{repo: r, llmCfg: llmCfg}
}

// List 配置列表（密钥脱敏）
func (s *llmConfigService) List() ([]*response.LLMConfigResponse, error) {
	list, err := s.repo.List()
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

	reply, latency, err := s.testConn(cfg)
	if err != nil {
		return &response.TestConnectionResponse{OK: false, LatencyMs: latency, Error: friendlyErr(err)}, nil
	}
	return &response.TestConnectionResponse{OK: true, LatencyMs: latency, Reply: reply}, nil
}

// Create 新建配置（内部复测，未通过则拒绝入库）
func (s *llmConfigService) Create(req *request.CreateLLMConfigRequest) (*response.LLMConfigResponse, error) {
	cfg := llmclient.Config{
		BaseURL:     strings.TrimSpace(req.BaseURL),
		APIKey:      strings.TrimSpace(req.APIKey),
		Model:       strings.TrimSpace(req.Model),
		TimeoutSec:  withDefaultInt(req.TimeoutSec, defaultTimeoutSec),
		MaxTokens:   withDefaultInt(req.MaxTokens, defaultMaxTokens),
		Temperature: withDefaultFloat(req.Temperature, defaultTemperature),
	}

	// 入库门禁：测试连接不通过则拒绝入库
	if _, _, err := s.testConn(cfg); err != nil {
		return nil, errors.New(errors.CodeBadRequest, "配置不可用，测试连接失败："+friendlyErr(err))
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
		TimeoutSec:  cfg.TimeoutSec,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Status:      status,
		IsDefault:   req.IsDefault,
	}

	if req.IsDefault {
		if err := s.repo.Transaction(func(tx repo.Repository) error {
			if err := tx.ClearDefault(); err != nil {
				return err
			}
			return tx.Create(m)
		}); err != nil {
			logger.Error("创建模型配置失败", zap.Error(err))
			return nil, errors.NewWithErr(errors.CodeInternalError, "创建模型配置失败", err)
		}
	} else if err := s.repo.Create(m); err != nil {
		logger.Error("创建模型配置失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "创建模型配置失败", err)
	}

	return toResponse(m), nil
}

// Update 更新配置（内部复测；APIKey 留空表示不修改）
func (s *llmConfigService) Update(id uint, req *request.UpdateLLMConfigRequest) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("读取待更新配置失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "读取配置失败", err)
	}
	if existing == nil {
		return errors.New(errors.CodeNotFound, "配置不存在")
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		apiKey = existing.APIKey
	}

	cfg := llmclient.Config{
		BaseURL:     strings.TrimSpace(req.BaseURL),
		APIKey:      apiKey,
		Model:       strings.TrimSpace(req.Model),
		TimeoutSec:  withDefaultInt(req.TimeoutSec, defaultTimeoutSec),
		MaxTokens:   withDefaultInt(req.MaxTokens, defaultMaxTokens),
		Temperature: withDefaultFloat(req.Temperature, defaultTemperature),
	}

	// 入库门禁：测试连接不通过则拒绝更新
	if _, _, err := s.testConn(cfg); err != nil {
		return errors.New(errors.CodeBadRequest, "配置不可用，测试连接失败："+friendlyErr(err))
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	fields := map[string]interface{}{
		"name":        cfg.Model, // 配置名称自动取模型 id
		"base_url":    cfg.BaseURL,
		"api_key":     cfg.APIKey,
		"model":       cfg.Model,
		"timeout_sec": cfg.TimeoutSec,
		"max_tokens":  cfg.MaxTokens,
		"temperature": cfg.Temperature,
		"status":      status,
		"is_default":  req.IsDefault,
	}

	if req.IsDefault {
		if err := s.repo.Transaction(func(tx repo.Repository) error {
			if err := tx.ClearDefault(); err != nil {
				return err
			}
			return tx.UpdateFields(id, fields)
		}); err != nil {
			logger.Error("更新模型配置失败", zap.Error(err))
			return errors.NewWithErr(errors.CodeInternalError, "更新模型配置失败", err)
		}
		return nil
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

// Activate 设为当前使用
func (s *llmConfigService) Activate(id uint) error {
	m, err := s.repo.FindByID(id)
	if err != nil {
		logger.Error("读取待启用配置失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "读取配置失败", err)
	}
	if m == nil {
		return errors.New(errors.CodeNotFound, "配置不存在")
	}
	if m.Status != 1 {
		return errors.New(errors.CodeBadRequest, "该配置已禁用，请先启用")
	}

	if err := s.repo.Transaction(func(tx repo.Repository) error {
		if err := tx.ClearDefault(); err != nil {
			return err
		}
		return tx.SetDefault(id)
	}); err != nil {
		logger.Error("设为当前使用失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "设为当前使用失败", err)
	}
	return nil
}

// ResolveActive 解析当前生效配置（库→兜底 config.yaml）
func (s *llmConfigService) ResolveActive() (llmclient.Config, uint, error) {
	m, err := s.repo.FindActive()
	if err != nil {
		logger.Error("读取当前模型配置失败", zap.Error(err))
		return llmclient.Config{}, 0, errors.NewWithErr(errors.CodeInternalError, "读取模型配置失败", err)
	}
	if m == nil {
		// 兜底：使用 config.yaml 的 llm 段
		c := s.llmCfg
		return llmclient.Config{
			BaseURL:     c.BaseURL,
			APIKey:      c.APIKey,
			Model:       c.Model,
			TimeoutSec:  withDefaultInt(c.TimeoutSec, defaultTimeoutSec),
			MaxTokens:   withDefaultInt(c.MaxTokens, defaultMaxTokens),
			Temperature: withDefaultFloat(c.Temperature, defaultTemperature),
		}, 0, nil
	}
	return llmclient.Config{
		BaseURL:     m.BaseURL,
		APIKey:      m.APIKey,
		Model:       m.Model,
		TimeoutSec:  withDefaultInt(m.TimeoutSec, defaultTimeoutSec),
		MaxTokens:   withDefaultInt(m.MaxTokens, defaultMaxTokens),
		Temperature: withDefaultFloat(m.Temperature, defaultTemperature),
	}, m.ID, nil
}

// testConn 用最小代价验证连通性，返回回复文本与耗时
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

// toResponse 实体转响应（密钥脱敏）
func toResponse(m *entity.AIModelConfig) *response.LLMConfigResponse {
	return &response.LLMConfigResponse{
		ID:           m.ID,
		BaseURL:      m.BaseURL,
		APIKeyMasked: maskKey(m.APIKey),
		Model:        m.Model,
		TimeoutSec:   m.TimeoutSec,
		MaxTokens:    m.MaxTokens,
		Temperature:  m.Temperature,
		Status:       m.Status,
		IsDefault:    m.IsDefault,
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
