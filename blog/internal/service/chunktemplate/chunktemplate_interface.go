package chunktemplate

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
)

// Service 切分模板服务接口
type Service interface {
	// List 列表；status 为 nil 表示不过滤
	List(status *int8) ([]*response.ChunkTemplateResponse, error)
	// Get 详情
	Get(id uint) (*response.ChunkTemplateResponse, error)
	// Create 新建
	Create(req *request.CreateChunkTemplateRequest) (*response.ChunkTemplateResponse, error)
	// Update 更新。已产生块的模板不允许改切分参数
	Update(id uint, req *request.UpdateChunkTemplateRequest) error
	// Delete 删除。已被使用的模板不允许删除
	Delete(id uint) error
}
