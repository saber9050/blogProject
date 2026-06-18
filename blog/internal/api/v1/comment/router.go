package comment

import (
	"blog/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRouter 注册评论路由组（挂载在 /articles 路径下）
func (c *Controller) RegisterRouter(r *gin.RouterGroup) {
	// 公开接口（无需登录）
	r.GET("/:id/comments", c.ListComments)                   // 一级评论列表
	r.GET("/:id/comments/:commentId/replies", c.ListReplies) // 二级回复列表

	// 需要登录的接口
	articlesAuth := r.Group("", middleware.Auth())
	{
		articlesAuth.POST("/:id/comments", c.CreateComment)              // 发表评论
		articlesAuth.DELETE("/:id/comments/:commentId", c.DeleteComment) // 删除评论
	}
}
