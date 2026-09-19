package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"blog/pkg/errors"
)

// Config 模型客户端配置
type Config struct {
	BaseURL     string  // 形如 https://api.deepseek.com/v1（到 /v1 为止）
	APIKey      string  // 服务商密钥
	Model       string  // 模型名，如 deepseek-chat
	TimeoutSec  int     // 单次请求超时秒数，默认 30
	MaxTokens   int     // 单次生成上限
	Temperature float64 // 采样温度
}

// Message 一条对话消息
type Message struct {
	Role    string `json:"role"` // system / user / assistant
	Content string `json:"content"`
}

// Client OpenAI 兼容协议的模型客户端
type Client struct {
	cfg        Config
	endpoint   string
	httpClient *http.Client
}

// NewClient 创建模型客户端
func NewClient(cfg Config) *Client {
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	return &Client{
		cfg:        cfg,
		endpoint:   strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions",
		httpClient: &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

// chatRequest /v1/chat/completions 请求体
type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// chatResponse /v1/chat/completions 响应体
type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Chat 发起一次对话补全，返回模型回复文本
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	if c.cfg.APIKey == "" {
		return "", errors.New(errors.CodeInternalError, "LLM API Key 未配置")
	}
	if c.cfg.BaseURL == "" {
		return "", errors.New(errors.CodeInternalError, "LLM BaseURL 未配置")
	}
	if c.cfg.Model == "" {
		return "", errors.New(errors.CodeInternalError, "LLM 模型名未配置")
	}

	body, err := json.Marshal(chatRequest{
		Model:       c.cfg.Model,
		Messages:    messages,
		Temperature: c.cfg.Temperature,
		MaxTokens:   c.cfg.MaxTokens,
	})
	if err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "序列化请求体失败", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "创建请求失败", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "请求 LLM 失败", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "读取响应失败", err)
	}

	if resp.StatusCode != http.StatusOK {
		// 注意：不把供应商原始响应体透传给前端，仅提取其 message 用于日志/提示
		var errResp chatResponse
		if json.Unmarshal(respBytes, &errResp) == nil && errResp.Error != nil && errResp.Error.Message != "" {
			return "", errors.New(errors.CodeInternalError, fmt.Sprintf("LLM 返回异常状态码 %d: %s", resp.StatusCode, errResp.Error.Message))
		}
		return "", errors.New(errors.CodeInternalError, fmt.Sprintf("LLM 返回异常状态码: %d", resp.StatusCode))
	}

	var result chatResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "解析响应失败", err)
	}
	if result.Error != nil && result.Error.Message != "" {
		return "", errors.New(errors.CodeInternalError, "LLM 返回错误: "+result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", errors.New(errors.CodeInternalError, "LLM 返回内容为空")
	}
	return result.Choices[0].Message.Content, nil
}
