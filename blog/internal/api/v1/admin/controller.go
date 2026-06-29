package admin

import (
	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	response2 "blog/internal/model/dto/response"
	articleSvc "blog/internal/service/article"
	categorySvc "blog/internal/service/category"
	tagSvc "blog/internal/service/tag"
	userSvc "blog/internal/service/user"
	"blog/pkg/errors"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AdminController 后台管理控制器
type AdminController struct {
	userService     userSvc.UserService
	articleService  articleSvc.ArticleService
	categoryService categorySvc.CategoryService
	tagService      tagSvc.TagService
}

// NewAdminController 创建后台管理控制器
func NewAdminController(
	userService userSvc.UserService,
	articleService articleSvc.ArticleService,
	categoryService categorySvc.CategoryService,
	tagService tagSvc.TagService,
) *AdminController {
	return &AdminController{
		userService:     userService,
		articleService:  articleService,
		categoryService: categoryService,
		tagService:      tagService,
	}
}

// ========== 用户管理 ==========

// ListUsers 获取普通用户列表
func (ctrl *AdminController) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	list, total, err := ctrl.userService.ListNormalUsers(page, pageSize)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, &response2.PaginatedResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// CreateUser 创建用户
func (ctrl *AdminController) CreateUser(c *gin.Context) {
	var req request.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	id, err := ctrl.userService.AdminCreateUser(&req)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, gin.H{"id": id})
}

// UpdateUser 更新用户状态
func (ctrl *AdminController) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	var req request.AdminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.userService.AdminUpdateStatus(uint(id), req.Status); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteUser 删除用户
func (ctrl *AdminController) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := ctrl.userService.AdminDeleteUser(uint(id)); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// ========== 文章管理 ==========

// ListArticles 获取后台文章列表
func (ctrl *AdminController) ListArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var status *int
	if s := c.Query("status"); s != "" {
		sInt, _ := strconv.Atoi(s)
		status = &sInt
	}

	var categoryID uint
	if cid := c.Query("category_id"); cid != "" {
		cidUint, _ := strconv.ParseUint(cid, 10, 32)
		categoryID = uint(cidUint)
	}

	keyword := c.Query("keyword")

	result, err := ctrl.articleService.AdminList(page, pageSize, status, categoryID, nil, keyword)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// CreateArticle 创建文章
func (ctrl *AdminController) CreateArticle(c *gin.Context) {
	var req request.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	result, err := ctrl.articleService.AdminCreate(&req, userID)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// UpdateArticle 更新文章
func (ctrl *AdminController) UpdateArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	var req request.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.articleService.AdminUpdate(uint(id), &req); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteArticle 删除文章
func (ctrl *AdminController) DeleteArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	if err := ctrl.articleService.AdminDelete(uint(id)); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// UploadImage 上传文章图片,返回完整路径
func (ctrl *AdminController) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传头像文件")
		return
	}

	url, err := ctrl.articleService.UploadImage(file)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, gin.H{
		"url": url,
	})
}

// ========== 分类管理 ==========

// ListCategories 获取后台分类列表
func (ctrl *AdminController) ListCategories(c *gin.Context) {
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
func (ctrl *AdminController) CreateCategory(c *gin.Context) {
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
func (ctrl *AdminController) UpdateCategory(c *gin.Context) {
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
func (ctrl *AdminController) DeleteCategory(c *gin.Context) {
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

// ========== 标签管理 ==========

// ListTags 获取后台标签列表
func (ctrl *AdminController) ListTags(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var status *int
	if s := c.Query("status"); s != "" {
		sInt, _ := strconv.Atoi(s)
		status = &sInt
	}
	keyword := c.Query("keyword")

	result, err := ctrl.tagService.List(page, pageSize, status, keyword)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// CreateTag 创建标签
func (ctrl *AdminController) CreateTag(c *gin.Context) {
	var req request.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.tagService.Create(&req); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateTag 更新标签
func (ctrl *AdminController) UpdateTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的标签ID")
		return
	}

	var req request.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.tagService.Update(uint(id), &req); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteTag 删除标签
func (ctrl *AdminController) DeleteTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的标签ID")
		return
	}

	if err := ctrl.tagService.Delete(uint(id)); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// errors 包快捷引用
var _ = errors.CodeNotFound
