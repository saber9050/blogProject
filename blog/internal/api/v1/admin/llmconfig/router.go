package llmconfig

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台模型配置路由（挂载在 /admin 分组下，需管理员认证）
func (ctrl *Controller) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/llm-configs")
	{
		g.GET("", ctrl.List)                   // 列表
		g.GET("/:id", ctrl.Get)                // 详情
		g.POST("/test", ctrl.Test)             // 测试连接（不落库）
		g.POST("", ctrl.Create)                // 新建（复测后入库）
		g.PUT("/:id", ctrl.Update)             // 更新（复测）
		g.DELETE("/:id", ctrl.Delete)          // 删除
		g.POST("/:id/activate", ctrl.Activate) // 设为当前使用
	}
}
