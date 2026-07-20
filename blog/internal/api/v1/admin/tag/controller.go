package tag

import (
	"blog/internal/model/dto/request"
	tagSvc "blog/internal/service/tag"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TagController 后台标签管理控制器
type TagController struct {
	tagService tagSvc.TagService
}

// NewTagController 创建后台标签管理控制器
func NewTagController(tagService tagSvc.TagService) *TagController {
	return &TagController{tagService: tagService}
}

// ListTags 获取后台标签列表
func (ctrl *TagController) ListTags(c *gin.Context) {
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
func (ctrl *TagController) CreateTag(c *gin.Context) {
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
func (ctrl *TagController) UpdateTag(c *gin.Context) {
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
func (ctrl *TagController) DeleteTag(c *gin.Context) {
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
