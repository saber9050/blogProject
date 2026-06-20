package article

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册前台文章路由
func (ctrl *ArticleController) RegisterRoutes(r *gin.RouterGroup) {
	articleGroup := r.Group("/articles")
	{
		articleGroup.GET("", ctrl.ListArticles)              // 获取文章列表
		articleGroup.GET("/:id", ctrl.GetArticleDetail)      // 获取文章详情
		articleGroup.POST("/:id/like", ctrl.LikeArticle)     // 点赞
		articleGroup.DELETE("/:id/like", ctrl.UnlikeArticle) // 取消点赞
	}
}
