package vector

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台向量表维护路由（由 adminGroup 挂载，已有 Auth(1)）
func (ctrl *Controller) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/vector")
	g.GET("/schema", ctrl.Schema)
	// /reset 的前端入口已移除（2026-10-29），保留接口仅供「换维度」时 curl 运维调用，
	// 详见 Reset 方法上的注释。
	g.POST("/reset", ctrl.Reset)
}
