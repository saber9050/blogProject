package category

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/category"
	"blog/pkg/errors"
	"fmt"
)

// categoryService 分类服务实现
type categoryService struct {
	categoryRepo repo.CategoryRepository
}

// NewCategoryService 创建分类服务实例
func NewCategoryService(categoryRepo repo.CategoryRepository) CategoryService {
	return &categoryService{categoryRepo: categoryRepo}
}

// ListPublic 获取所有启用的分类（前台）
func (s *categoryService) ListPublic() ([]*response.CategoryPublicResponse, error) {
	list, err := s.categoryRepo.ListPublic()
	if err != nil {
		return nil, fmt.Errorf("获取分类列表失败: %w", err)
	}
	var result []*response.CategoryPublicResponse
	for _, c := range list {
		result = append(result, &response.CategoryPublicResponse{
			ID:   c.ID,
			Name: c.CategoryName,
		})
	}
	return result, nil
}

// List 获取分页的分类列表（后台）
func (s *categoryService) List(page, pageSize int, status *int, keyword string) (*response.PaginatedResponse, error) {
	list, total, err := s.categoryRepo.List(page, pageSize, status, keyword)
	if err != nil {
		return nil, fmt.Errorf("获取分类列表失败: %w", err)
	}
	var items []*response.CategoryAdminResponse
	for _, c := range list {
		items = append(items, &response.CategoryAdminResponse{
			ID:        c.ID,
			Name:      c.CategoryName,
			Status:    c.Status,
			CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return &response.PaginatedResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Create 创建分类
func (s *categoryService) Create(req *request.CreateCategoryRequest) error {
	// 检查分类名称是否已存在
	exists, err := s.categoryRepo.IsExistsByName(req.Name, 0)
	if err != nil {
		return fmt.Errorf("检查分类名称失败: %w", err)
	}
	if exists {
		return errors.New(errors.CodeBadRequest, "分类名称已存在")
	}
	category := &entity.Category{
		CategoryName: req.Name,
		Status:       req.Status,
	}
	return s.categoryRepo.Create(category)
}

// Update 更新分类
func (s *categoryService) Update(id uint, req *request.UpdateCategoryRequest) error {
	// 检查分类是否存在
	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("查找分类失败: %w", err)
	}
	if category == nil {
		return errors.New(errors.CodeNotFound, "分类不存在")
	}
	// 检查分类名称是否重复
	exists, err := s.categoryRepo.IsExistsByName(req.Name, id)
	if err != nil {
		return fmt.Errorf("检查分类名称失败: %w", err)
	}
	if exists {
		return errors.New(errors.CodeBadRequest, "分类名称已存在")
	}
	category.CategoryName = req.Name
	category.Status = req.Status
	return s.categoryRepo.Update(category)
}

// Delete 删除分类
func (s *categoryService) Delete(id uint) error {
	// 检查分类是否存在
	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("查找分类失败: %w", err)
	}
	if category == nil {
		return errors.New(errors.CodeNotFound, "分类不存在")
	}
	// 检查分类下是否有关联文章
	count, err := s.categoryRepo.CountByCategoryID(id)
	if err != nil {
		return fmt.Errorf("统计关联文章失败: %w", err)
	}
	if count > 0 {
		return errors.New(errors.CodeBadRequest, "该分类下存在文章，无法删除")
	}
	return s.categoryRepo.Delete(id)
}
