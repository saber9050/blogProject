package tag

import (
	"blog/internal/model/dto/response"
	tagSvc "blog/internal/service/tag"
	"blog/pkg/response"

	"github.com/gin-gonic/gin"
)

// TagController 标签控制器
type TagController struct {
	tagService tagSvc.TagService
}

// NewTagController 创建标签控制器
func NewTagController(tagService tagSvc.TagService) *TagController {
	return &TagController{
		tagService: tagService,
	}
}

// ListTags 获取标签列表（前台接口，无需认证）
func (ctrl *TagController) ListTags(c *gin.Context) {
	list, err := ctrl.tagService.ListPublic()
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, list)
}
