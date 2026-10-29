package vectorschema

import (
	"blog/internal/model/dto/response"
)

// Service 向量表维护服务
type Service interface {
	// EnsureFor 按给定维度确保向量表就绪（不存在则建表，存在则校验维度）。
	// 供「新增向量模型」调用——维度必须显式传入，不能从库里读，
	// 因为那时新模型还没落库。
	EnsureFor(dim int) error
	// EnsureFromDB 从库里当前启用的向量模型读维度再 EnsureFor。
	// 供应用启动、向量化前置校验调用。
	EnsureFromDB() error
	// Info 当前向量表与向量模型的匹配情况
	Info() (*response.VectorSchemaResponse, error)
	// Reset 清空向量数据：删表 + 重置所有文章的向量状态。
	// **不重建表**，重建交给下一次 EnsureFor。
	Reset() (*response.VectorResetResponse, error)
}
