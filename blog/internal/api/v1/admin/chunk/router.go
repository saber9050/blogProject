package chunk

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台切分管理路由（由 adminGroup 挂载，已有 Auth(1)）
//
// /chunks/stat 与 /chunks/:id 在同层级混了静态段与参数段。项目用的 gin v1.12
// 支持这种共存（v1.7 起静态优先匹配）。这里把静态路由写在前面只是为了让阅读顺序
// 与匹配优先级一致，gin 本身不依赖注册顺序。
func (ctrl *Controller) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/chunks")
	g.GET("", ctrl.List)
	g.GET("/stat", ctrl.Stat)
	g.GET("/:id", ctrl.Get)
	// 整篇删除。gin 按 method 分树，DELETE 树里只有这一条，
	// 不会与 GET 树的 :id 参数段冲突。
	g.DELETE("/article/:article_id", ctrl.DeleteByArticle)
}
