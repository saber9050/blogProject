package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"blog/pkg/errors"
)

// Embed 批量把文本转成向量。
//
// 走 OpenAI 兼容的 /embeddings。ollama 也实现了这个协议（base_url 填
// http://host:11434/v1），所以本地模型和云端模型用同一份代码。
//
// 调用方负责两件事，客户端不猜：
//   - 每批的条数（建议 10 条）：太大容易单次超时，太小则请求数爆炸
//   - 结果的维度校验：客户端不知道库表要求多少维，只有调用方知道
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if c.cfg.APIKey == "" {
		return nil, errors.New(errors.CodeInternalError, "向量模型 API Key 未配置")
	}
	if c.cfg.BaseURL == "" {
		return nil, errors.New(errors.CodeInternalError, "向量模型 BaseURL 未配置")
	}
	if c.cfg.Model == "" {
		return nil, errors.New(errors.CodeInternalError, "向量模型名未配置")
	}
	if len(texts) == 0 {
		return nil, errors.New(errors.CodeBadRequest, "待向量化文本为空")
	}

	body, err := json.Marshal(embeddingRequest{Model: c.cfg.Model, Input: texts})
	if err != nil {
		return nil, errors.NewWithErr(errors.CodeInternalError, "序列化向量化请求失败", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.embeddingEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.NewWithErr(errors.CodeInternalError, "创建向量化请求失败", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewWithErr(errors.CodeInternalError, "请求向量模型失败", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取向量模型响应失败", err)
	}

	if resp.StatusCode != http.StatusOK {
		// 同 Chat：只提取 message 用于提示，不透传供应商完整响应体
		var errResp embeddingResponse
		if json.Unmarshal(respBytes, &errResp) == nil && errResp.Error != nil && errResp.Error.Message != "" {
			return nil, errors.New(errors.CodeInternalError,
				fmt.Sprintf("向量模型返回异常状态码 %d: %s", resp.StatusCode, errResp.Error.Message))
		}
		return nil, errors.New(errors.CodeInternalError,
			fmt.Sprintf("向量模型返回异常状态码: %d", resp.StatusCode))
	}

	var result embeddingResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, errors.NewWithErr(errors.CodeInternalError, "解析向量模型响应失败", err)
	}
	if result.Error != nil && result.Error.Message != "" {
		return nil, errors.New(errors.CodeInternalError, "向量模型返回错误: "+result.Error.Message)
	}
	if len(result.Data) != len(texts) {
		return nil, errors.New(errors.CodeInternalError,
			fmt.Sprintf("向量模型返回 %d 条结果，与请求的 %d 条不一致", len(result.Data), len(texts)))
	}

	// 按 index 放回原位。不能假设返回顺序与输入一致——OpenAI 的规范里
	// index 就是用来重排的，有的服务（尤其是并发实现的网关）确实会乱序。
	out := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < 0 || d.Index >= len(texts) {
			return nil, errors.New(errors.CodeInternalError,
				fmt.Sprintf("向量模型返回的 index %d 越界（共 %d 条）", d.Index, len(texts)))
		}
		if len(d.Embedding) == 0 {
			return nil, errors.New(errors.CodeInternalError,
				fmt.Sprintf("第 %d 条向量为空", d.Index))
		}
		out[d.Index] = d.Embedding
	}
	for i, v := range out {
		if v == nil {
			return nil, errors.New(errors.CodeInternalError,
				fmt.Sprintf("向量模型返回结果缺少第 %d 条", i))
		}
	}
	return out, nil
}

// embeddingRequest /v1/embeddings 请求体
type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embeddingResponse /v1/embeddings 响应体
type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// embeddingEndpointOf 由 BaseURL 拼出向量化地址。
// base_url 约定写到 /v1 为止，与 Chat 一致。
func embeddingEndpointOf(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/embeddings"
}
