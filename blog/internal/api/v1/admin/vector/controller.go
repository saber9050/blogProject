package vector

import (
	"fmt"

	"blog/internal/model/dto/request"
	vectorSvc "blog/internal/service/vectorschema"
	"blog/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller 后台向量表维护控制器
type Controller struct {
	vectorService vectorSvc.Service
}

// NewController 创建后台向量表维护控制器
func NewController(vectorService vectorSvc.Service) *Controller {
	return &Controller{vectorService: vectorService}
}

// Schema 向量表现状（维度、块数、与模型是否匹配）
func (ctrl *Controller) Schema(c *gin.Context) {
	result, err := ctrl.vectorService.Info()
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Reset 清空向量数据（危险操作）
//
// confirm 校验放在 controller 而不是 service：它属于「接口层的误操作防护」，
// 换个调用方（比如将来的批量任务）不该被这个字符串绑住。
func (ctrl *Controller) Reset(c *gin.Context) {
	var req request.ResetVectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if req.Confirm != "RESET" {
		// 提示里带上将被删除的块数：只说「请确认」的话，用户不知道代价有多大
		var n int64
		if info, err := ctrl.vectorService.Info(); err == nil && info != nil {
			n = info.ChunkTotal
		}
		response.BadRequest(c, fmt.Sprintf(
			"此操作会永久删除全部 %d 个向量块且不可恢复，请传 confirm:\"RESET\" 确认", n))
		return
	}

	result, err := ctrl.vectorService.Reset()
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}
