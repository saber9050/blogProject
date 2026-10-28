package vectorize

import (
	"context"

	"blog/internal/model/dto/response"
)

// Service 文章向量化服务
type Service interface {
	// Submit 提交向量化任务。只做校验 + 抢锁 + 投递，毫秒级返回，
	// 真正的切分与 embedding 在后台 worker 里串行执行。
	Submit(articleID, templateID uint) (*response.VectorizeSubmitResponse, error)
	// Status 查询向量化状态（轮询用，只查 articles 一行）
	Status(articleID uint) (*response.VectorStatusResponse, error)
	// RecoverZombies 把进程重启前遗留的「排队中/处理中」重置为失败。
	// 必须在 Start 之前调用，否则那些文章会永远卡住且无法重新提交。
	RecoverZombies() (int64, error)
	// Start 启动 worker。ctx 取消时 worker 退出。
	Start(ctx context.Context)
}
