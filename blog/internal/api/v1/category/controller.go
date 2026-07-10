package category

import (
	dtoResp "blog/internal/model/dto/response"
	categorySvc "blog/internal/service/category"
	resp "blog/pkg/response"

	"github.com/gin-gonic/gin"
)

// CategoryController 分类控制器
type CategoryController struct {
	categoryService categorySvc.CategoryService
}

// NewCategoryController 创建分类控制器
func NewCategoryController(categoryService categorySvc.CategoryService) *CategoryController {
	return &CategoryController{
		categoryService: categoryService,
	}
}

// ListCategories 获取分类列表（前台接口，无需认证）
func (ctrl *CategoryController) ListCategories(c *gin.Context) {
	list, err := ctrl.categoryService.ListPublic()
	if err != nil {
		resp.BizError(c, err)
		return
	}

	resp.Success(c, list)
}

// CountEnabledCategories 统计启用分类数量（前台接口，无需认证）
func (ctrl *CategoryController) CountEnabledCategories(c *gin.Context) {
	count, err := ctrl.categoryService.CountEnabled()
	if err != nil {
		resp.BizError(c, err)
		return
	}

	resp.Success(c, &dtoResp.CountResponse{Count: count})
}
