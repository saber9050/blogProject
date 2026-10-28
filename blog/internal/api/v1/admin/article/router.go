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

	// 向量化：提交与状态查询都挂在 /articles/:id 下。
	// 这两条是两段路径，与上面 PUT/DELETE /articles/:id 深度不同，不会冲突。
	r.POST("/articles/:id/vectorize", ctrl.Vectorize)
	r.GET("/articles/:id/vector-status", ctrl.VectorStatus)

	r.POST("/upload", ctrl.UploadImage)
}
