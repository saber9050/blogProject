package chat

import (
	"context"
	"encoding/json"
	"testing"
)

// TestSearchToolInfo 检查检索工具对外暴露的 schema。
//
// 为什么值得单独测：ADK 在构建 agent 时会把每个 tool 的 Info 转成 toolInfos 传给模型。
// 如果 Info 的 ParamsOneOf 缺失或参数名不对，模型就**永远看不到这个工具**——
// 表现是「能正常回答、但从不检索」，而且不报任何错。这是最难定位的一类故障：
// 没有异常、没有日志，只能靠对照 schema 才能发现。
func TestSearchToolInfo(t *testing.T) {
	// searcher 传 nil 即可：这个测试只关心 schema，不会真正执行检索。
	tl := newSearchTool(nil)

	info, err := tl.Info(context.Background())
	if err != nil {
		t.Fatalf("取工具 Info 失败: %v", err)
	}
	if info.Name != searchToolName {
		t.Fatalf("工具名 = %q，期望 %q", info.Name, searchToolName)
	}
	if info.Desc == "" {
		t.Fatal("工具描述为空：没有描述模型不知道什么时候该用它")
	}

	params, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatalf("参数 schema 生成失败: %v", err)
	}
	if params == nil {
		t.Fatal("参数 schema 为 nil：模型会认为这个工具没有参数，通常就不会调它")
	}

	b, _ := json.Marshal(params)
	t.Logf("参数 schema: %s", string(b))

	// 关键断言：必须有一个名为 query 的 string 参数。
	// 参数名对不上时模型调用会被 tool 侧解析失败，表现为「调了但一直报参数错」。
	props, _ := params.Properties.Get("query")
	if props == nil {
		t.Fatalf("schema 里没有 query 参数，模型无法正确调用：%s", string(b))
	}
	if props.Type != "string" {
		t.Fatalf("query 参数类型 = %q，期望 string", props.Type)
	}
}
