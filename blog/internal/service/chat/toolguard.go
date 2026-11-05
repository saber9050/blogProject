package chat

import (
	"context"
	stderrors "errors"
	"time"

	"blog/pkg/logger"

	"github.com/cloudwego/eino/compose"
	"go.uber.org/zap"
)

// 边界控制的数值。全部硬编码在代码里，**不写在提示词里**：
// 提示词只能降低「模型一直查」的概率，不能保证；真正兜住的是这几道闸。
const (
	// maxToolCalls 单次提问内同一工具最多执行几次。
	//
	// 为什么是 5：正常问答 1~2 次就够；复杂问题（跨多篇引用）3~4 次也够了。
	// 5 是「够用 + 不会失控」的折中。超了不报错，返回一句「请基于现有资料作答」，
	// 模型看到这句会收尾——让它以为「查不到了」比让它以为「系统坏了」安全得多，
	// 后者可能促使它改用通用知识编答案。
	maxToolCalls = 5
	// maxToolOnce 单次工具调用的超时。
	//
	// 检索 = 1 次 embedding + 2 条 SQL，正常几十毫秒。15s 说明下游（embed 服务 /
	// 数据库）出问题了，继续等没有意义，不如让模型基于已有资料先答。
	maxToolOnce = 15 * time.Second
)

// toolGuard 工具级中间件：次数闸 + 单次超时 + 「最后一次」提示。
//
// 为什么把次数闸放在中间件而不是靠 MaxIterations：
// MaxIterations 数的是「模型调用轮次」，而它默认 20，只用来防 agent 彻底转死。
// 业务上的「最多查 5 次」必须在 tool 层表达，语义才清晰，也不受「一轮里调几个工具」影响。
func toolGuard(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
	return func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
		sc := scopeFromCtx(ctx)
		if sc.SessionID == 0 {
			// scope 没传到工具层 = 中间件拿不到计数对象，这道闸等于失效。
			// 不阻断（检索本身还能跑），但要留痕，否则这种静默失效永远查不出来。
			logger.Warn("工具中间件未取到请求 scope，次数限制本次不生效",
				zap.Uint("session_id", sc.SessionID))
		}

		if sc.ToolCalls >= maxToolCalls {
			// ⚠️ 返回 nil error 而不是 error：
			// 返回 error 会被框架当成工具失败，可能触发重试或打乱循环。
			// 返回一段正常文本，模型只会理解成「没有更多资料了」，然后基于已有内容作答。
			return &compose.ToolOutput{
				Result: "检索次数已用完，请基于现有资料作答。不要再调用检索工具。",
			}, nil
		}

		callCtx, cancel := context.WithTimeout(ctx, maxToolOnce)
		defer cancel()

		out, err := next(callCtx, in)
		if err != nil {
			// 父 ctx 已被取消（用户点停止 / 整体超时）→ 必须原样抛出去让循环停下。
			// 这时候「假装成功」是错的：用户已经不想等了，继续跑只是白烧 token。
			if ctx.Err() != nil {
				return nil, err
			}
			// 单次超时不中断整条链：把它变成一句可理解的话，让模型自己决定怎么办
			if stderrors.Is(err, context.DeadlineExceeded) {
				sc.Degraded = true
				return &compose.ToolOutput{Result: "检索超时，请基于已有资料作答。"}, nil
			}
			// 其他错误原样返回（tool 内部已经做过分类，能到这里的都是真异常）
			return nil, err
		}
		if out == nil {
			return &compose.ToolOutput{Result: "没有检索到相关内容。"}, nil
		}

		// 第 5 次（最后一次）在结果开头加一句提示。
		// 这不是提示词层的劝说，而是工具输出里的一个字段——它一定进模型上下文，
		// 比在 Instruction 里写「最多查 5 次」可靠得多。
		if sc.ToolCalls == maxToolCalls {
			out.Result = "（本次检索已是最后一次，请基于已有资料作答）\n" + out.Result
		}
		return out, nil
	}
}
