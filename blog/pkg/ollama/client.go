package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"blog/pkg/errors"
)

// generateRequest Ollama /api/generate 请求体
type generateRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	Stream    bool   `json:"stream"`
	KeepAlive string `json:"keep_alive,omitempty"`
}

// generateResponse Ollama /api/generate 响应体
type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// Config Ollama 客户端配置
type Config struct {
	BaseURL    string
	ModelName  string
	KeepAlive  string
	TimeoutSec int
}

// Client Ollama API 客户端
type Client struct {
	cfg    Config
	client *http.Client
}

// NewClient 创建 Ollama API 客户端
func NewClient(cfg Config) *Client {
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}

	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Generate 调用 Ollama 的 /api/generate 接口
func (c *Client) Generate(prompt string) (string, error) {
	baseURL := c.cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	reqBody := generateRequest{
		Model:     c.cfg.ModelName,
		Prompt:    prompt,
		Stream:    false,
		KeepAlive: c.cfg.KeepAlive,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "序列化请求体失败", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.cfg.TimeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/generate", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "创建请求失败", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "请求 Ollama 失败", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "读取响应失败", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.NewWithErr(errors.CodeInternalError, fmt.Sprintf("Ollama 返回异常状态码: %d", resp.StatusCode), nil)
	}

	var result generateResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", errors.NewWithErr(errors.CodeInternalError, "解析响应失败", err)
	}

	if result.Error != "" {
		return "", errors.NewWithErr(errors.CodeInternalError, fmt.Sprintf("Ollama 返回错误: %s", result.Error), nil)
	}

	return result.Response, nil
}
