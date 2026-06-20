package tag

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册标签路由
func (ctrl *TagController) RegisterRoutes(r *gin.RouterGroup) {
	// 前台标签接口（无需认证）
	r.GET("/tags", ctrl.ListTags)
}
