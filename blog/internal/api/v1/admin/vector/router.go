package vector

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台向量表维护路由（由 adminGroup 挂载，已有 Auth(1)）
func (ctrl *Controller) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/vector")
	g.GET("/schema", ctrl.Schema)
	g.POST("/reset", ctrl.Reset)
}
