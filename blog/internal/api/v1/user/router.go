package user

import (
	"blog/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRouter 注册用户路由组
func (c *Controller) RegisterRouter(r *gin.RouterGroup) {
	userGroup := r.Group("/user", middleware.Auth())
	{
		userGroup.GET("/info", c.GetUserInfo)              // 获取用户信息
		userGroup.POST("/profile", c.UpdateProfile)        // 编辑用户信息
		userGroup.POST("/avatar", c.UpdateAvatar)          // 更换头像
		userGroup.POST("/email_ack", c.UpdateEmailRequest) // 修改邮箱请求
	}
}

// RegisterRouterACK 注册用户修改邮箱路由组
func (c *Controller) RegisterRouterACK(r *gin.RouterGroup) {
	userGroup := r.Group("/user")
	{
		userGroup.GET("/email", c.UpdateEmail) // 修改邮箱
	}
}
