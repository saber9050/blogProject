package chat

import (
	stderrors "errors"
	"strconv"

	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	chatSvc "blog/internal/service/chat"
	"blog/pkg/errors"
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
	cg.POST("/sessions/:sessionId/messages", ctrl.SendMessage)
	// 工具消息按需拉：历史列表只给元信息（查了什么、命中几段），
	// 用户点开某一轮才取原文。原文单条 6KB 起，跟着列表走会让响应膨胀几十倍。
	cg.GET("/sessions/:sessionId/messages/:messageId/tools", ctrl.ListToolMessages)
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

// SendMessage 发消息。stream=true 走 SSE，false 走一次性 JSON。
func (ctrl *Controller) SendMessage(c *gin.Context) {
	sessionID, ok := parseIDParam(c, "sessionId")
	if !ok {
		response.BadRequest(c, "无效的会话ID")
		return
	}
	var req request.SendChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	// 用请求自带的 ctx：客户端断开 / 浏览器关页面时它会被取消，
	// 从而中断 agent 循环和正在执行的检索——否则用户点了停止，后端还在烧 token。
	ctx := c.Request.Context()

	if !req.WantsStream() {
		res, err := ctrl.svc.SendMessage(ctx, userID, sessionID, &req, nil)
		if err != nil {
			response.BizError(c, err)
			return
		}
		response.Success(c, res)
		return
	}

	sink := newSSESink(c)
	res, err := ctrl.svc.SendMessage(ctx, userID, sessionID, &req, sink)
	if err != nil {
		if !sink.started {
			// meta 还没发出去，HTTP 状态码还没定型，仍能返回正常的 404/400/503。
			// 这正是 sseSink 延迟写 header 的目的。
			response.BizError(c, err)
			return
		}
		// 已经开始推流，只能靠 error 事件表达；状态码无法再改。
		_ = sink.OnError(bizCode(err), bizMessage(err), sink.wroteDelta)
		return
	}
	_ = sink.done(res)
}

// ListToolMessages 取某条助手消息的工具调用记录（含检索到的原文）。
func (ctrl *Controller) ListToolMessages(c *gin.Context) {
	sessionID, ok := parseIDParam(c, "sessionId")
	if !ok {
		response.BadRequest(c, "无效的会话ID")
		return
	}
	messageID, ok := parseIDParam(c, "messageId")
	if !ok {
		response.BadRequest(c, "无效的消息ID")
		return
	}
	list, err := ctrl.svc.ListToolMessages(middleware.GetUserID(c), sessionID, messageID)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
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

// bizCode / bizMessage 从 service 返回的错误里取出对外的业务码与文案。
// 非 BizError 一律按 500 + 通用文案处理，绝不把内部错误原文透给前端
// （里面可能有 SQL 片段、上游地址之类）。
func bizCode(err error) int {
	var be *errors.BizError
	if stderrors.As(err, &be) {
		return be.Code
	}
	return errors.CodeInternalError
}

func bizMessage(err error) string {
	var be *errors.BizError
	if stderrors.As(err, &be) {
		return be.Message
	}
	return errors.GetMessage(errors.CodeInternalError)
}
