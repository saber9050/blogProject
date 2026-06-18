package comment

import (
	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	commentSvc "blog/internal/service/comment"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller 评论控制器
type Controller struct {
	commentService commentSvc.CommentService
}

// NewController 创建评论控制器
func NewController(commentService commentSvc.CommentService) *Controller {
	return &Controller{commentService: commentService}
}

// ListComments 获取一级评论列表
func (c *Controller) ListComments(ctx *gin.Context) {
	articleID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || articleID == 0 {
		response.BadRequest(ctx, "文章ID无效")
		return
	}

	var req request.ListCommentsReq
	_ = ctx.ShouldBindQuery(&req) // 忽略绑定错误，使用默认值

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	result, err := c.commentService.ListComments(uint(articleID), req.Page, req.PageSize)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, result)
}

// ListReplies 获取二级评论（回复）列表
func (c *Controller) ListReplies(ctx *gin.Context) {
	articleID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || articleID == 0 {
		response.BadRequest(ctx, "文章ID无效")
		return
	}

	commentID, err := strconv.ParseUint(ctx.Param("commentId"), 10, 64)
	if err != nil || commentID == 0 {
		response.BadRequest(ctx, "评论ID无效")
		return
	}

	var req request.ListRepliesReq
	_ = ctx.ShouldBindQuery(&req)

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	result, err := c.commentService.ListReplies(uint(commentID), req.Page, req.PageSize)
	if err != nil {
		response.BizError(ctx, err)
		return
	}

	// 防止未使用警告
	_ = articleID

	response.Success(ctx, result)
}

// CreateComment 发表评论
func (c *Controller) CreateComment(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	articleID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || articleID == 0 {
		response.BadRequest(ctx, "文章ID无效")
		return
	}

	var req request.CreateCommentReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}

	comment, err := c.commentService.CreateComment(uint(articleID), userID, &req)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, comment)
}

// DeleteComment 删除评论
func (c *Controller) DeleteComment(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	commentID, err := strconv.ParseUint(ctx.Param("commentId"), 10, 64)
	if err != nil || commentID == 0 {
		response.BadRequest(ctx, "评论ID无效")
		return
	}

	roleID := middleware.GetRoleID(ctx)

	if err := c.commentService.DeleteComment(uint(commentID), userID, int8(roleID)); err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, "删除成功")
}
