package chat

import (
	"context"
	"fmt"
	"time"

	"blog/pkg/errors"
	"blog/pkg/logger"

	"go.uber.org/zap"
)

// 限流阈值。按「每用户每分钟」计数。
//
// 为什么是 6 而不是更高：agentic 单条提问实际是 2~3 次模型调用（先 tool_call 再作答），
// token 消耗与延迟都是固定流水线的 2~3 倍。6 条/分钟对正常用户绰绰有余，
// 对刷接口的人则是硬天花板。
const (
	chatRateLimit  = 6
	chatRateWindow = time.Minute
	// key 的过期时间比窗口略长：窗口结束后还留一点余量，
	// 避免「刚好在整分钟边界」的计数被提前清掉而少拦一次。
	chatRateKeyTTL = 90 * time.Second
)

// allow 判断本次提问是否放行。
//
// ⚠️ 三条刻意的设计：
//  1. **Redis 不可用就放行**。限流是防刷手段，不是安全边界；缓存挂了让整个功能不可用
//     是本末倒置（登录、发文都不受 Redis 影响）。
//  2. **固定窗口**（按 unix 分钟分桶）而不是滑动窗口：实现只有 INCR + EXPIRE 两条命令，
//     不需要 Lua。代价是边界处可能放行接近 2 倍（59 秒和 01 秒各 6 次），
//     对「防刷」这个目标完全够用。
//  3. key 里带分钟数，天然按窗口隔离；EXPIRE 只是防止 key 泄漏堆积。
func (s *chatService) allow(ctx context.Context, userID uint) error {
	if s.redis == nil {
		return nil
	}
	window := time.Now().Unix() / int64(chatRateWindow/time.Second)
	key := fmt.Sprintf("chat:rate:u:%d:%d", userID, window)

	n, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		logger.Warn("限流计数失败，本次放行", zap.Uint("user_id", userID), zap.Error(err))
		return nil
	}
	if n == 1 {
		// 只在第一次设置过期，避免每个请求都发一次 EXPIRE
		if err := s.redis.Expire(ctx, key, chatRateKeyTTL).Err(); err != nil {
			logger.Warn("设置限流 key 过期失败", zap.Error(err))
		}
	}
	if n > chatRateLimit {
		return errors.New(errors.CodeTooManyRequests, "提问太频繁了，休息一下再问")
	}
	return nil
}
