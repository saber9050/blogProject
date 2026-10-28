package chunktemplate

import (
	"blog/internal/model/dto/request"
	chunkTplSvc "blog/internal/service/chunktemplate"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller 后台切分模板控制器
type Controller struct {
	tplService chunkTplSvc.Service
}

// NewController 创建后台切分模板控制器
func NewController(tplService chunkTplSvc.Service) *Controller {
	return &Controller{tplService: tplService}
}

// List 模板列表。支持 ?status=1|0 过滤
func (ctrl *Controller) List(c *gin.Context) {
	var status *int8
	if s := c.Query("status"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || (n != 0 && n != 1) {
			response.BadRequest(c, "status 只能是 0 或 1")
			return
		}
		v := int8(n)
		status = &v
	}

	result, err := ctrl.tplService.List(status)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Get 模板详情
func (ctrl *Controller) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "无效的模板ID")
		return
	}

	result, err := ctrl.tplService.Get(id)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Create 新建模板
func (ctrl *Controller) Create(c *gin.Context) {
	var req request.CreateChunkTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误：名称必填且不超过 64 字，块大小必填")
		return
	}

	result, err := ctrl.tplService.Create(&req)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Update 更新模板（已使用的模板不允许改切分参数）
func (ctrl *Controller) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "无效的模板ID")
		return
	}

	var req request.UpdateChunkTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.tplService.Update(id, &req); err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, nil)
}

// Delete 删除模板（已被使用的模板不允许删除）
func (ctrl *Controller) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "无效的模板ID")
		return
	}

	if err := ctrl.tplService.Delete(id); err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, nil)
}

func parseID(c *gin.Context) (uint, error) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id64), nil
}
