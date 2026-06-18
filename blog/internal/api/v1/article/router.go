package article

import (
	"blog/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRouter 注册文章路由组
func (c *Controller) RegisterRouter(r *gin.RouterGroup) {
	articleGroup := r.Group("/articles")
	{
		// 公开接口 — 可选认证（有token自动注入用户上下文，用于点赞状态等）
		publicGroup := articleGroup.Group("", middleware.OptionalAuth())
		{
			publicGroup.GET("", c.ListArticles)         // 文章列表
			publicGroup.GET("/:id", c.GetArticleDetail) // 文章详情
		}

		// 需要登录的接口
		authGroup := articleGroup.Group("", middleware.Auth())
		{
			authGroup.POST("/:id/like", c.LikeArticle)     // 点赞
			authGroup.DELETE("/:id/like", c.UnlikeArticle) // 取消点赞
			authGroup.GET("/me/likes", c.BatchLikeStatus)  // 批量查询点赞状态
		}
	}
}

// RegisterPublicRouter 注册公开的分类/标签路由
func (c *Controller) RegisterPublicRouter(r *gin.RouterGroup) {
	r.GET("/categories", c.ListCategories) // 分类列表
	r.GET("/tags", c.ListTags)             // 标签列表
}
