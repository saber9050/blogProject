package tag

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/tag"
	"blog/pkg/errors"
)

// tagService 标签服务实现
type tagService struct {
	tagRepo repo.TagRepository
}

// NewTagService 创建标签服务实例
func NewTagService(tagRepo repo.TagRepository) TagService {
	return &tagService{tagRepo: tagRepo}
}

// ListPublic 获取所有启用的标签（前台）
func (s *tagService) ListPublic() ([]*response.TagPublicResponse, error) {
	list, err := s.tagRepo.ListPublic()
	if err != nil {
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取标签列表失败", err)
	}
	var result []*response.TagPublicResponse
	for _, t := range list {
		result = append(result, &response.TagPublicResponse{
			ID:   t.ID,
			Name: t.TagName,
		})
	}
	return result, nil
}

// List 获取分页的标签列表（后台）
func (s *tagService) List(page, pageSize int, status *int, keyword string) (*response.PaginatedResponse, error) {
	list, total, err := s.tagRepo.List(page, pageSize, status, keyword)
	if err != nil {
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取标签列表失败", err)
	}
	var items []*response.TagAdminResponse
	for _, t := range list {
		items = append(items, &response.TagAdminResponse{
			ID:        t.ID,
			Name:      t.TagName,
			Status:    t.Status,
			CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return &response.PaginatedResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Create 创建标签
func (s *tagService) Create(req *request.CreateTagRequest) error {
	// 检查标签名称是否已存在
	exists, err := s.tagRepo.IsExistsByName(req.Name, 0)
	if err != nil {
		return errors.NewWithErr(errors.CodeInternalError, "检查标签名称失败", err)
	}
	if exists {
		return errors.New(errors.CodeBadRequest, "标签名称已存在")
	}
	tag := &entity.Tag{
		TagName: req.Name,
		Status:  req.Status,
	}
	return s.tagRepo.Create(tag)
}

// Update 更新标签
func (s *tagService) Update(id uint, req *request.UpdateTagRequest) error {
	// 检查标签是否存在
	tag, err := s.tagRepo.FindByID(id)
	if err != nil {
		return errors.NewWithErr(errors.CodeInternalError, "查找标签失败", err)
	}
	if tag == nil {
		return errors.New(errors.CodeNotFound, "标签不存在")
	}
	// 检查标签名称是否重复
	exists, err := s.tagRepo.IsExistsByName(req.Name, id)
	if err != nil {
		return errors.NewWithErr(errors.CodeInternalError, "检查标签名称失败", err)
	}
	if exists {
		return errors.New(errors.CodeBadRequest, "标签名称已存在")
	}

	// 构建需要更新的字段
	fields := map[string]interface{}{
		"tag_name": req.Name,
		"status":   req.Status,
	}

	return s.tagRepo.UpdateFields(id, fields)
}

// Delete 删除标签
func (s *tagService) Delete(id uint) error {
	// 检查标签是否存在
	tag, err := s.tagRepo.FindByID(id)
	if err != nil {
		return errors.NewWithErr(errors.CodeInternalError, "查找标签失败", err)
	}
	if tag == nil {
		return errors.New(errors.CodeNotFound, "标签不存在")
	}
	// 删除标签与文章的关联关系
	if err := s.tagRepo.DeleteTagArticles(id); err != nil {
		return errors.NewWithErr(errors.CodeInternalError, "删除标签关联失败", err)
	}
	return s.tagRepo.Delete(id)
}

// CountEnabled 统计启用状态的标签数量
func (s *tagService) CountEnabled() (int64, error) {
	count, err := s.tagRepo.CountEnabled()
	if err != nil {
		return 0, errors.NewWithErr(errors.CodeInternalError, "统计启用标签失败", err)
	}
	return count, nil
}
