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
	// DeleteArticles 批量删除多篇文章的全部块，并把这些文章的向量状态重置为「未处理」。
	//
	// 逐篇独立事务：勾选 10 篇里有 1 篇正在向量化时，跳过它、其余照删，
	// 而不是整批回滚让用户自己去猜是哪一篇卡住了。
	//
	// 只支持整篇删除，不提供单块删除：块是「整篇重新切分、整体替换」的产物，
	// 删掉其中一块之后这篇文章的块集合就既不是这次切分的结果、也不是上次的，
	// 之后无论重新切分还是做检索都会得到无法解释的状态。
	DeleteArticles(articleIDs []uint) (*response.ChunkDeleteBatchResponse, error)
}
