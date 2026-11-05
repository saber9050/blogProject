package chat

import (
	"strconv"

	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	chatSvc "blog/internal/service/chat"
	"blog/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller 前台 AI 助手接口。
//
// 所有路由都挂了 middleware.Auth()：游客不能使用该功能。
// ⚠️ 绝不能换成 OptionalAuth —— 未登录时 user_id 会是 0，所有匿名请求会落进
// 同一个归属桶，等于把所有人的对话串在一起。而单人测试时永远只有一个「匿名用户」，
// 这个 bug 在开发期测不出来，上线才会炸。
type Controller struct {
	svc chatSvc.Service
}

// NewController 创建控制器。
func NewController(svc chatSvc.Service) *Controller {
	return &Controller{svc: svc}
}

// RegisterRoutes 注册路由。
//
// 分两个维度挂载：
//   - /articles/:id/chat/context 语义上属于「这篇文章的聊天室」，与现有 /articles 树同级
//   - /chat/... 是全新的会话维度前缀，不与任何现有组冲突
//
// 路由冲突已核对：现有 /articles 树下已有 "" / /random / /stats / /:id / /:id/like /
// /:id/comments，参数名统一是 :id。新增 /:id/chat/context 与它们同层级、同参数名、
// 静态段不同，gin 可以共存。
func (ctrl *Controller) RegisterRoutes(r *gin.RouterGroup) {
	ag := r.Group("/articles", middleware.Auth())
	ag.GET("/:id/chat/context", ctrl.Context)

	cg := r.Group("/chat", middleware.Auth())
	cg.POST("/sessions", ctrl.CreateSession)
	cg.GET("/sessions/:sessionId/messages", ctrl.ListMessages)
	cg.PATCH("/sessions/:sessionId", ctrl.UpdateSession)
	cg.DELETE("/sessions/:sessionId", ctrl.DeleteSession)
	cg.GET("/article-candidates", ctrl.ListArticleCandidates)
}

// Context 面板初始化：能不能聊、可用模型、该文章下的会话列表。
func (ctrl *Controller) Context(c *gin.Context) {
	articleID, ok := parseIDParam(c, "id")
	if !ok {
		response.BadRequest(c, "无效的文章ID")
		return
	}
	res, err := ctrl.svc.Context(middleware.GetUserID(c), articleID)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, res)
}

// CreateSession 新建会话。
func (ctrl *Controller) CreateSession(c *gin.Context) {
	var req request.CreateChatSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	res, err := ctrl.svc.CreateSession(middleware.GetUserID(c), &req)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, res)
}

// ListMessages 历史消息（游标分页）。
func (ctrl *Controller) ListMessages(c *gin.Context) {
	sessionID, ok := parseIDParam(c, "sessionId")
	if !ok {
		response.BadRequest(c, "无效的会话ID")
		return
	}
	var q request.ChatMessageListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	res, err := ctrl.svc.ListMessages(middleware.GetUserID(c), sessionID, q.BeforeID, q.Limit)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, res)
}

// UpdateSession 改标题 / 换模型。
func (ctrl *Controller) UpdateSession(c *gin.Context) {
	sessionID, ok := parseIDParam(c, "sessionId")
	if !ok {
		response.BadRequest(c, "无效的会话ID")
		return
	}
	var req request.UpdateChatSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	if err := ctrl.svc.UpdateSession(middleware.GetUserID(c), sessionID, &req); err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, nil)
}

// DeleteSession 硬删会话及其消息。
func (ctrl *Controller) DeleteSession(c *gin.Context) {
	sessionID, ok := parseIDParam(c, "sessionId")
	if !ok {
		response.BadRequest(c, "无效的会话ID")
		return
	}
	if err := ctrl.svc.DeleteSession(middleware.GetUserID(c), sessionID); err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, nil)
}

// ListArticleCandidates `/` 选择器的候选文章。
func (ctrl *Controller) ListArticleCandidates(c *gin.Context) {
	var q request.ChatArticleCandidateQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	res, err := ctrl.svc.ListArticleCandidates(q.Keyword, q.ExcludeID, q.Limit)
	if err != nil {
		response.BizError(c, err)
		return
	}
	// 直接返回数组（不是 {list: [...]}）：候选浮层只关心顺序和标题，
	// 没有分页语义，套一层 list 只会让前端多写一次解构。
	response.Success(c, res)
}

// parseIDParam 解析路径上的正整数 id。0 与非法值都返回 false。
func parseIDParam(c *gin.Context, name string) (uint, bool) {
	n, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}
