package comment

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台评论管理路由
func (ctrl *CommentController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/comments", ctrl.ListComments)
	r.DELETE("/comments", ctrl.BatchDeleteComments)
	r.DELETE("/comments/:id", ctrl.DeleteComment)
}
