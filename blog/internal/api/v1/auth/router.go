package auth

import "github.com/gin-gonic/gin"

// RegisterRouter 注册认证路由组
func (c *Controller) RegisterRouter(r *gin.RouterGroup) {
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", c.Register)               // 注册
		authGroup.GET("/image_captcha", c.SendImageCaptcha)   // 图形验证码
		authGroup.POST("/login", c.Login)                     // 登录
		authGroup.POST("/email_login", c.EmailLogin)          // 邮箱登录
		authGroup.POST("/captcha", c.SendEmailCaptcha)        // 发送邮箱验证码
		authGroup.POST("/reset_password", c.ResetPassword)    // 重设密码
		authGroup.POST("/logout", c.Logout)                   // 登出
		authGroup.GET("/is_exists_name", c.IsExistName)       // 检测名字唯一性
		authGroup.GET("/is_exists_account", c.IsExistAccount) // 检测账号唯一性
	}
}
