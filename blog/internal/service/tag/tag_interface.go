package tag

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
)

// TagService 标签服务接口
type TagService interface {
	// ListPublic 获取所有启用的标签（前台）
	ListPublic() ([]*response.TagPublicResponse, error)
	// List 获取分页的标签列表（后台）
	List(page, pageSize int, status *int, keyword string) (*response.PaginatedResponse, error)
	// Create 创建标签
	Create(req *request.CreateTagRequest) error
	// Update 更新标签
	Update(id uint, req *request.UpdateTagRequest) error
	// Delete 删除标签
	Delete(id uint) error
}
