package comment

import (
	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	commentSvc "blog/internal/service/comment"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CommentController 后台评论管理控制器
type CommentController struct {
	commentService commentSvc.CommentService
}

// NewCommentController 创建后台评论管理控制器
func NewCommentController(commentService commentSvc.CommentService) *CommentController {
	return &CommentController{commentService: commentService}
}

// ListComments 获取评论列表（所有未删除评论）
func (ctrl *CommentController) ListComments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := ctrl.commentService.ListAdminComments(page, pageSize)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// DeleteComment 删除评论
func (ctrl *CommentController) DeleteComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	userID := middleware.GetUserID(c)
	if err := ctrl.commentService.DeleteComment(uint(id), userID, 1); err != nil {
		response.BizError(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// BatchDeleteComments 批量删除评论
func (ctrl *CommentController) BatchDeleteComments(c *gin.Context) {
	var req request.AdminBatchDeleteCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	if len(req.IDs) == 0 {
		response.BadRequest(c, "请选择要删除的评论")
		return
	}

	affected, err := ctrl.commentService.BatchDeleteComment(req.IDs)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.SuccessWithMessage(c, "批量删除成功", gin.H{
		"affected_count": affected,
	})
}
