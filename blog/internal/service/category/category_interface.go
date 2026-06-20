package category

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
)

// CategoryService 分类服务接口
type CategoryService interface {
	// ListPublic 获取所有启用的分类（前台）
	ListPublic() ([]*response.CategoryPublicResponse, error)
	// List 获取分页的分类列表（后台）
	List(page, pageSize int, status *int, keyword string) (*response.PaginatedResponse, error)
	// Create 创建分类
	Create(req *request.CreateCategoryRequest) error
	// Update 更新分类
	Update(id uint, req *request.UpdateCategoryRequest) error
	// Delete 删除分类
	Delete(id uint) error
}
