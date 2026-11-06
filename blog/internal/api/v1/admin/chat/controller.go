package chat

import (
	"strconv"

	"blog/internal/model/dto/request"
	chatSvc "blog/internal/service/chat"
	"blog/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller 后台「助手会话」管理。
//
// 单独一个包（admin/chat）而不是塞进前台 api/v1/chat：
// 前台是「用户看自己的数据」（带归属校验），后台是「管理员看全站 + join 用户表」。
// 两套语义混在一个 controller 里，迟早会因为复制粘贴时漏掉鉴权中间件出事。
//
// 鉴权由 adminGroup 统一挂载的 middleware.Auth(1) 提供，这里不再重复。
type Controller struct {
	svc chatSvc.Service
}

// NewController 创建后台控制器。
func NewController(svc chatSvc.Service) *Controller {
	return &Controller{svc: svc}
}

// RegisterRoutes 注册后台路由。
func (ctrl *Controller) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/chat-sessions")
	g.GET("", ctrl.List)
	g.GET("/:id", ctrl.Detail)
	g.DELETE("/:id", ctrl.Delete)
}

// List 会话列表：会话 id / 文章名 / 用户 id+名 / 提问数 / 最后活跃 / 摘要。
//
// 筛选用一个输入框实现两种匹配（纯数字按 user_id，其余按用户名模糊），
// 文章名单独一个输入框。
func (ctrl *Controller) List(c *gin.Context) {
	var q request.AdminChatSessionQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	res, err := ctrl.svc.AdminList(&q)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, res)
}

// Detail 会话详情：会话信息 + 全部消息（含 refs / refused / error 徽章数据）。
func (ctrl *Controller) Detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		response.BadRequest(c, "无效的会话ID")
		return
	}
	res, err := ctrl.svc.AdminDetail(uint(id))
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, res)
}

// Delete 硬删会话。不可恢复，前端必须二次确认。
func (ctrl *Controller) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		response.BadRequest(c, "无效的会话ID")
		return
	}
	if err := ctrl.svc.AdminDelete(uint(id)); err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, nil)
}
