package category

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/category"
	"blog/pkg/errors"
	"blog/pkg/logger"

	"go.uber.org/zap"
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
	enabled := 1
	list, _, err := s.categoryRepo.List(1, 10000, &enabled, "", "", "")
	if err != nil {
		logger.Error("获取分类列表失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取分类列表失败", err)
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
func (s *categoryService) List(page, pageSize int, status *int, keyword, startTime, endTime string) (*response.PaginatedResponse, error) {
	list, total, err := s.categoryRepo.List(page, pageSize, status, keyword, startTime, endTime)
	if err != nil {
		logger.Error("获取分类列表失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取分类列表失败", err)
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
		logger.Error("检查分类名称失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "检查分类名称失败", err)
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
		logger.Error("查找分类失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "查找分类失败", err)
	}
	if category == nil {
		return errors.New(errors.CodeNotFound, "分类不存在")
	}
	// 检查分类名称是否重复
	exists, err := s.categoryRepo.IsExistsByName(req.Name, id)
	if err != nil {
		logger.Error("检查分类名称失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "检查分类名称失败", err)
	}
	if exists {
		return errors.New(errors.CodeBadRequest, "分类名称已存在")
	}

	// 如果状态改为禁用，检查分类下是否有关联文章
	if req.Status == 0 && category.Status != 0 {
		count, err := s.categoryRepo.CountByCategoryID(id)
		if err != nil {
			logger.Error("统计关联文章失败", zap.Error(err))
			return errors.NewWithErr(errors.CodeInternalError, "统计关联文章失败", err)
		}
		if count > 0 {
			return errors.New(errors.CodeBadRequest, "该分类下存在文章，不可禁用")
		}
	}

	// 构建需要更新的字段
	fields := map[string]interface{}{
		"category_name": req.Name,
		"status":        req.Status,
	}

	return s.categoryRepo.UpdateFields(id, fields)
}

// Delete 删除分类
func (s *categoryService) Delete(id uint) error {
	// 检查分类是否存在
	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		logger.Error("查找分类失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "查找分类失败", err)
	}
	if category == nil {
		return errors.New(errors.CodeNotFound, "分类不存在")
	}
	// 检查分类下是否有关联文章
	count, err := s.categoryRepo.CountByCategoryID(id)
	if err != nil {
		logger.Error("统计关联文章失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "统计关联文章失败", err)
	}
	if count > 0 {
		return errors.New(errors.CodeBadRequest, "该分类下存在文章，无法删除")
	}
	return s.categoryRepo.Delete(id)
}

// CountEnabled 统计启用状态的分类数量
func (s *categoryService) CountEnabled() (int64, error) {
	count, err := s.categoryRepo.CountEnabled()
	if err != nil {
		logger.Error("统计启用分类失败", zap.Error(err))
		return 0, errors.NewWithErr(errors.CodeInternalError, "统计启用分类失败", err)
	}
	return count, nil
}
