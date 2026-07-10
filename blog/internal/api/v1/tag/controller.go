package tag

import (
	dtoResp "blog/internal/model/dto/response"
	tagSvc "blog/internal/service/tag"
	resp "blog/pkg/response"

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
		resp.BizError(c, err)
		return
	}

	resp.Success(c, list)
}

// CountEnabledTags 统计启用标签数量（前台接口，无需认证）
func (ctrl *TagController) CountEnabledTags(c *gin.Context) {
	count, err := ctrl.tagService.CountEnabled()
	if err != nil {
		resp.BizError(c, err)
		return
	}

	resp.Success(c, &dtoResp.CountResponse{Count: count})
}
