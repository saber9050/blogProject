package about

import (
	"blog/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册关于页面路由
func (ctrl *AboutController) RegisterRoutes(r *gin.RouterGroup) {
	// 前台无需认证的接口
	r.GET("/admin/info", ctrl.GetAdminInfo) // 获取管理员信息
	r.GET("/about", ctrl.GetAboutInfo)      // 获取关于页面完整信息

	// 后台需要管理员认证的接口
	adminGroup := r.Group("")
	adminGroup.Use(middleware.Auth(1)) // 要求管理员角色（role_id = 1）
	{
		adminGroup.PUT("/about", ctrl.UpdateAbout) // 更新关于页面内容
	}
}
