package llm

import (
	"fmt"
	"regexp"
	"strings"

	"blog/pkg/config"
	"blog/pkg/errors"
	"blog/pkg/logger"
	"blog/pkg/ollama"

	"go.uber.org/zap"
)

type llmServiceImpl struct {
	ollamaClient *ollama.Client
}

// NewLLMService 创建 LLM 服务
func NewLLMService(cfg config.LLMConfig) LLMService {
	return &llmServiceImpl{
		ollamaClient: ollama.NewClient(ollama.Config{
			BaseURL:    cfg.BaseURL,
			ModelName:  cfg.ModelName,
			KeepAlive:  cfg.KeepAlive,
			TimeoutSec: cfg.TimeoutSec,
		}),
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

// GenerateSummary 根据文章标题和内容生成摘要
func (s *llmServiceImpl) GenerateSummary(title string, content string) (string, error) {
	// 1. 去除 HTML 标签，提取纯文本
	plainText := stripHTMLTags(content)
	if len(plainText) == 0 {
		return "", errors.New(errors.CodeBadRequest, "文章内容为空")
	}

	// 2. 限制文本长度，避免超出模型上下文（按字符数截断，避免切碎 UTF-8 字符）
	runes := []rune(plainText)
	if len(runes) > 8000 {
		plainText = string(runes[:8000])
	}

	// 3. 构造 prompt（包含标题和内容）
	prompt := fmt.Sprintf(`请为以下文章生成一段简洁的摘要，要求：
- 不超过100字
- 概括文章核心内容
- 语言简洁通顺
- 不要出现"本文"、"文章"等开头
- 不要出现任何表情符号
- 只输出一段话的文本
- 不要有任何形式的分段和分条回答
- 生成完摘要后审查是否超过了150词，如果超过重新按照要求生成

文章标题：
%s

文章内容：
%s

摘要：`, title, plainText)

	// 4. 调用 Ollama API
	summary, err := s.ollamaClient.Generate(prompt)
	if err != nil {
		logger.Error("调用 Ollama 生成摘要失败", zap.Error(err))
		return "", err
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
