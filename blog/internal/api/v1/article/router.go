package article

import (
	"blog/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册前台文章路由
func (ctrl *ArticleController) RegisterRoutes(r *gin.RouterGroup) {
	articleGroup := r.Group("/articles")
	{
		articleGroup.GET("", middleware.OptionalAuth(), ctrl.ListArticles)         // 获取文章列表（可选认证）
		articleGroup.GET("/:id", middleware.OptionalAuth(), ctrl.GetArticleDetail) // 获取文章详情（可选认证）
		articleGroup.POST("/:id/like", middleware.Auth(), ctrl.LikeArticle)        // 点赞（需认证）
		articleGroup.DELETE("/:id/like", middleware.Auth(), ctrl.UnlikeArticle)    // 取消点赞（需认证）
	}
}
