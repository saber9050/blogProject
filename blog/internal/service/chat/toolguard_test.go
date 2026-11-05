package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/compose"
)

// 这几条验证的是「硬闸」——不是提示词，是代码。
//
// 之所以要单测：闸门失效的表现是「模型一直查、请求越来越慢、账单越来越高」，
// 这类问题不会报错，也不会有明显异常，只能靠测试在这个函数上钉死行为。

// countingEndpoint 模拟真实 tool：每执行一次自增 ToolCalls（与 runSearch 一致）。
func countingEndpoint(calls *int, sc *Scope) compose.InvokableToolEndpoint {
	return func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
		*calls++
		sc.ToolCalls++
		return &compose.ToolOutput{Result: "检索结果"}, nil
	}
}

func TestToolGuardBlocksAfterMaxCalls(t *testing.T) {
	sc := &Scope{SessionID: 1}
	ctx := withScope(context.Background(), sc)
	calls := 0
	guarded := toolGuard(countingEndpoint(&calls, sc))

	// 前 5 次都能真正执行
	for i := 0; i < maxToolCalls; i++ {
		out, err := guarded(ctx, &compose.ToolInput{})
		if err != nil {
			t.Fatalf("第 %d 次调用不应报错: %v", i+1, err)
		}
		if out == nil {
			t.Fatalf("第 %d 次调用没有输出", i+1)
		}
	}
	if calls != maxToolCalls {
		t.Fatalf("应执行 %d 次，实际 %d", maxToolCalls, calls)
	}

	// 第 6 次：不执行检索，返回一句让模型收尾的话
	out, err := guarded(ctx, &compose.ToolInput{})
	if err != nil {
		t.Fatalf("超次数必须返回 nil error（返回 error 会被框架当成工具失败）: %v", err)
	}
	if calls != maxToolCalls {
		t.Fatalf("第 %d 次不应真正执行检索", maxToolCalls+1)
	}
	if !strings.Contains(out.Result, "检索次数已用完") {
		t.Fatalf("应提示次数用完，实际: %q", out.Result)
	}
	// 提示里必须有「基于现有资料作答」——不然模型可能改去编答案
	if !strings.Contains(out.Result, "基于现有资料作答") {
		t.Fatalf("提示缺少「基于现有资料作答」: %q", out.Result)
	}
}

func TestToolGuardMarksLastCall(t *testing.T) {
	sc := &Scope{SessionID: 1}
	ctx := withScope(context.Background(), sc)
	calls := 0
	guarded := toolGuard(countingEndpoint(&calls, sc))

	// 第 5 次（= maxToolCalls）的结果要带「已是最后一次」提示
	for i := 0; i < maxToolCalls; i++ {
		out, _ := guarded(ctx, &compose.ToolInput{})
		isLast := i == maxToolCalls-1
		hasMark := strings.Contains(out.Result, "已是最后一次")
		if isLast && !hasMark {
			t.Fatalf("最后一次应带提示，实际: %q", out.Result)
		}
		if !isLast && hasMark {
			t.Fatalf("第 %d 次不该带「最后一次」提示", i+1)
		}
	}
}

func TestToolGuardRespectsParentCancel(t *testing.T) {
	// 用户点「停止」后：必须把 ctx 错误原样抛出去让循环停下，
	// 而不是包装成「检索超时」继续跑——那会让 agent 在已取消的请求上继续烧 token。
	sc := &Scope{SessionID: 1}
	parent, cancel := context.WithCancel(withScope(context.Background(), sc))
	blocks := func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	guarded := toolGuard(blocks)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := guarded(parent, &compose.ToolInput{})
	if err == nil {
		t.Fatal("父 ctx 取消时必须返回错误")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("应透出取消错误，实际: %v", err)
	}
}

func TestToolGuardNilScopeStillWorks(t *testing.T) {
	// 万一 ctx 里的 scope 丢了（框架换了 ctx），也不能 panic 或卡死：
	// 退化成「计数不生效」，检索本身照常可用。
	calls := 0
	endpoint := func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
		calls++
		return &compose.ToolOutput{Result: "ok"}, nil
	}
	guarded := toolGuard(endpoint)

	out, err := guarded(context.Background(), &compose.ToolInput{})
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if calls != 1 || out.Result != "ok" {
		t.Fatalf("无 scope 时应正常执行，calls=%d out=%v", calls, out)
	}
}
