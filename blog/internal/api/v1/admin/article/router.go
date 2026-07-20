package article

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台文章管理路由
func (ctrl *ArticleController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/articles", ctrl.ListArticles)
	r.POST("/articles", ctrl.CreateArticle)
	r.POST("/articles/generate-summary", ctrl.GenerateSummary)
	r.PUT("/articles/transfer", ctrl.TransferArticleCategory)
	r.PUT("/articles/:id", ctrl.UpdateArticle)
	r.DELETE("/articles/:id", ctrl.DeleteArticle)
	r.POST("/upload", ctrl.UploadImage)
}
