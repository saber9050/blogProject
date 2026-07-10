package about

import (
	"blog/internal/model/dto/request"
	aboutSvc "blog/internal/service/about"
	"blog/pkg/response"

	"github.com/gin-gonic/gin"
)

// AboutController 关于页面控制器
type AboutController struct {
	aboutService aboutSvc.AboutService
}

// NewAboutController 创建关于页面控制器
func NewAboutController(aboutService aboutSvc.AboutService) *AboutController {
	return &AboutController{aboutService: aboutService}
}

// GetAboutInfo 获取关于页面完整信息
// GET /about
func (ctrl *AboutController) GetAboutInfo(c *gin.Context) {
	info, err := ctrl.aboutService.GetAboutInfo()
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, info)
}

// UpdateAbout 更新关于页面内容
// PUT /about
func (ctrl *AboutController) UpdateAbout(c *gin.Context) {
	var req request.AboutUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	// 检查请求体是否为空（没有提供任何字段）
	if req.TechStack == nil && req.MyStory == nil && req.Why == nil &&
		req.Interest == nil && req.GitHub == nil && req.CSDN == nil {
		response.Success(c, nil)
		return
	}

	err := ctrl.aboutService.UpdateAbout(&req)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, nil)
}
