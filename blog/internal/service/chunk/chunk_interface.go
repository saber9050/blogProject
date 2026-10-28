package chunk

import (
	"blog/internal/model/dto/response"
)

// Service 切分块查询服务（切分管理页用）
type Service interface {
	// List 分页查询。content 截断到 200 字，全文走 Get
	List(templateID, articleID uint, keyword string, page, pageSize int) (*response.ChunkListResponse, error)
	// Get 单块详情（完整 content）
	Get(id uint) (*response.ChunkItem, error)
	// Stat 聚合统计。groupBy 只能是 template / article
	Stat(groupBy string) (*response.ChunkStatResponse, error)
}
