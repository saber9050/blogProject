package llmconfig

import (
	"blog/internal/model/dto/request"
	llmconfigSvc "blog/internal/service/llmconfig"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller 后台模型配置控制器
type Controller struct {
	llmConfigService llmconfigSvc.Service
}

// NewController 创建后台模型配置控制器
func NewController(llmConfigService llmconfigSvc.Service) *Controller {
	return &Controller{llmConfigService: llmConfigService}
}

// List 配置列表。支持 ?type=1|2 过滤（1=对话模型 2=向量模型）
func (ctrl *Controller) List(c *gin.Context) {
	var t int8
	if v := c.Query("type"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || (n != 1 && n != 2) {
			response.BadRequest(c, "type 只能是 1（对话模型）或 2（向量模型）")
			return
		}
		t = int8(n)
	}

	result, err := ctrl.llmConfigService.ListByType(t)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Get 配置详情
func (ctrl *Controller) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "无效的配置ID")
		return
	}

	result, err := ctrl.llmConfigService.Get(id)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Test 测试连接（不落库）
func (ctrl *Controller) Test(c *gin.Context) {
	var req request.TestLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	result, err := ctrl.llmConfigService.Test(&req)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Create 新建配置（内部复测，未通过则拒绝入库）
func (ctrl *Controller) Create(c *gin.Context) {
	var req request.CreateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	result, err := ctrl.llmConfigService.Create(&req)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Update 更新配置（仅修改运行参数与状态，不测试连接）
func (ctrl *Controller) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "无效的配置ID")
		return
	}

	var req request.UpdateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.llmConfigService.Update(id, &req); err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, nil)
}

// Delete 删除配置
func (ctrl *Controller) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "无效的配置ID")
		return
	}

	if err := ctrl.llmConfigService.Delete(id); err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, nil)
}

// parseID 解析路径参数 id
func parseID(c *gin.Context) (uint, error) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id64), nil
}
