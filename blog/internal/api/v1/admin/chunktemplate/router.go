package chunktemplate

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台切分模板路由（由 adminGroup 挂载，已有 Auth(1)）
func (ctrl *Controller) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/chunk-templates")
	g.GET("", ctrl.List)
	g.GET("/:id", ctrl.Get)
	g.POST("", ctrl.Create)
	g.PUT("/:id", ctrl.Update)
	g.DELETE("/:id", ctrl.Delete)
}
