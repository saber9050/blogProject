package category

import (
	"blog/internal/model/dto/request"
	categorySvc "blog/internal/service/category"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CategoryController 后台分类管理控制器
type CategoryController struct {
	categoryService categorySvc.CategoryService
}

// NewCategoryController 创建后台分类管理控制器
func NewCategoryController(categoryService categorySvc.CategoryService) *CategoryController {
	return &CategoryController{categoryService: categoryService}
}

// ListCategories 获取后台分类列表
func (ctrl *CategoryController) ListCategories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var status *int
	if s := c.Query("status"); s != "" {
		sInt, _ := strconv.Atoi(s)
		status = &sInt
	}
	keyword := c.Query("keyword")

	result, err := ctrl.categoryService.List(page, pageSize, status, keyword)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// CreateCategory 创建分类
func (ctrl *CategoryController) CreateCategory(c *gin.Context) {
	var req request.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.categoryService.Create(&req); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateCategory 更新分类
func (ctrl *CategoryController) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的分类ID")
		return
	}

	var req request.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.categoryService.Update(uint(id), &req); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteCategory 删除分类
func (ctrl *CategoryController) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的分类ID")
		return
	}

	if err := ctrl.categoryService.Delete(uint(id)); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}
