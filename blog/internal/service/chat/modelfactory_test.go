package chat

import (
	"math"
	"testing"

	"blog/pkg/llmclient"
)

// 这组用例盯的是「转换时不报错、但行为全错」的那几个字段。
// eino-ext 的 ChatModelConfig 有几个指针字段，取地址和不取地址语义完全不同：
//   - MaxTokens 传 &0 是「最多生成 0 个 token」，不是「不限制」——模型会一个字都吐不出来
//   - Temperature 传 &0 是「完全确定性」，与「没配」不可区分
//   - Timeout 我们刻意不传（0 = 不超时），超时交给 ctx 控整条链

func TestToOpenAIConfig(t *testing.T) {
	got := toOpenAIConfig(llmclient.Config{
		BaseURL:     "https://api.example.com/v1",
		APIKey:      "sk-test",
		Model:       "deepseek-chat",
		TimeoutSec:  30,
		MaxTokens:   2048,
		Temperature: 0.3,
	})

	if got.BaseURL != "https://api.example.com/v1" || got.APIKey != "sk-test" || got.Model != "deepseek-chat" {
		t.Fatalf("基础字段未正确映射: %+v", got)
	}
	if got.MaxTokens == nil || *got.MaxTokens != 2048 {
		t.Fatalf("MaxTokens 未映射: %v", got.MaxTokens)
	}
	if got.Temperature == nil {
		t.Fatal("Temperature 为 nil，配置里的 0.3 丢了")
	}
	if math.Abs(float64(*got.Temperature)-0.3) > 1e-6 {
		t.Fatalf("Temperature 精度丢失: %v", *got.Temperature)
	}
	// 不用 TimeoutSec：那是「单次 HTTP 超时」，管不住多轮 agent。
	// 我们靠 ctx 的 120s 控整条链，这里必须是 0（eino-ext 的 0 表示不超时）。
	if got.Timeout != 0 {
		t.Fatalf("不应把 TimeoutSec 映射成客户端超时（那会掐断长回答），实际 %v", got.Timeout)
	}
}

func TestToOpenAIConfigZeroMaxTokens(t *testing.T) {
	// 配置里 MaxTokens 为 0（零值）时必须**不设**，而不是设成指针指向 0。
	// 设成 &0 的后果是模型被要求输出 0 个 token，回答永远是空的。
	got := toOpenAIConfig(llmclient.Config{Model: "m"})
	if got.MaxTokens != nil {
		t.Fatalf("MaxTokens=0 时不应设置该字段（&0 等于「最多 0 token」），实际 %d", *got.MaxTokens)
	}
}

func TestToOpenAIConfigTemperatureZeroIsExplicit(t *testing.T) {
	// 温度 0 是合法且有意义的（求确定性输出），与「没配」无法区分，
	// 这里明确选择「照传」：配置表默认值是 0.3，读到 0 说明就是想要 0。
	got := toOpenAIConfig(llmclient.Config{Model: "m", Temperature: 0})
	if got.Temperature == nil || *got.Temperature != 0 {
		t.Fatalf("Temperature=0 应当显式传 0，实际 %v", got.Temperature)
	}
}

func TestExtractQuery(t *testing.T) {
	cases := []struct {
		name string
		args string
		want string
	}{
		{"正常", `{"query":"Stream 消费组"}`, "Stream 消费组"},
		{"带空格", `{ "query" : "Redis 持久化" }`, "Redis 持久化"},
		{"空串", `{"query":""}`, ""},
		{"畸形 JSON", `{"query":`, ""},
		{"没有 query 字段", `{"q":"x"}`, ""},
		{"参数是数字", `{"query":123}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractQuery(c.args); got != c.want {
				t.Fatalf("extractQuery(%q) = %q, want %q", c.args, got, c.want)
			}
		})
	}
}

func TestCountSegments(t *testing.T) {
	content := "[1] 《A》· 小节一\n正文……\n[2] 《B》· 小节二\n正文……\n[不是序号] 这行不算"
	if got := countSegments(content); got != 2 {
		t.Fatalf("countSegments = %d, want 2", got)
	}
}
