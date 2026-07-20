package tag

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台标签管理路由
func (ctrl *TagController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/tags", ctrl.ListTags)
	r.POST("/tags", ctrl.CreateTag)
	r.PUT("/tags/:id", ctrl.UpdateTag)
	r.DELETE("/tags/:id", ctrl.DeleteTag)
}
