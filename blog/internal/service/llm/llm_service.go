package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"blog/pkg/config"
	"blog/pkg/logger"

	"go.uber.org/zap"
)

// ollamaGenerateRequest Ollama /api/generate 请求体
type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaGenerateResponse Ollama /api/generate 响应体
type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

type llmServiceImpl struct {
	cfg    config.LLMConfig
	client *http.Client
}

// NewLLMService 创建 LLM 服务
func NewLLMService(cfg config.LLMConfig) LLMService {
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}

	return &llmServiceImpl{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// stripHTMLTags 去除 HTML 标签，提取纯文本
func stripHTMLTags(html string) string {
	// 去除 HTML 标签
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(html, "")
	// 替换连续空白为单个空格
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	// 去除首尾空白
	text = strings.TrimSpace(text)
	return text
}

// GenerateSummary 根据文章内容生成摘要
func (s *llmServiceImpl) GenerateSummary(content string) (string, error) {
	// 1. 去除 HTML 标签，提取纯文本
	plainText := stripHTMLTags(content)
	if len(plainText) == 0 {
		return "", fmt.Errorf("文章内容为空")
	}

	// 2. 限制文本长度，避免超出模型上下文（按字符数截断，避免切碎 UTF-8 字符）
	runes := []rune(plainText)
	if len(runes) > 8000 {
		plainText = string(runes[:8000])
	}

	// 3. 构造 prompt
	prompt := fmt.Sprintf(`请为以下文章生成一段简洁的摘要，要求：
- 不超过100字
- 概括文章核心内容
- 语言简洁通顺
- 不要出现"本文"、"文章"等开头
- 不要出现任何表情符号
- 只输出一段文本
- 输出格式参考如下：
在开发一个即时通讯聊天室时，为了增强系统的灵活性、稳定性以及扩展性，我们结合了多种缓存和排行榜功能。Redis被用于提升性能并实现特定的功能如用户信息缓存、历史消息缓存、异步消息队列和活跃度排行榜等。

文章内容：
%s

摘要：`, plainText)

	// 4. 调用 Ollama API
	summary, err := s.callOllama(prompt)
	if err != nil {
		logger.Error("调用 Ollama 生成摘要失败", zap.Error(err))
		return "", fmt.Errorf("摘要生成失败: %w", err)
	}

	// 5. 清理和截断（按字符数截断，避免切碎 UTF-8 字符）
	summary = strings.TrimSpace(summary)
	summary = strings.Trim(summary, `"'「」""''`)
	summary = strings.TrimSpace(summary)
	sRunes := []rune(summary)
	if len(sRunes) > 255 {
		summary = string(sRunes[:255])
	}

	return summary, nil
}

// callOllama 调用 Ollama 的 /api/generate 接口
func (s *llmServiceImpl) callOllama(prompt string) (string, error) {
	baseURL := s.cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	reqBody := ollamaGenerateRequest{
		Model:  s.cfg.ModelName,
		Prompt: prompt,
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.TimeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/generate", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 Ollama 失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama 返回异常状态码: %d, body: %s", resp.StatusCode, string(respBytes))
	}

	var result ollamaGenerateResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("Ollama 返回错误: %s", result.Error)
	}

	return result.Response, nil
}
