package category

import (
	categorySvc "blog/internal/service/category"
	"blog/pkg/response"

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
		response.BizError(c, err)
		return
	}

	response.Success(c, list)
}
