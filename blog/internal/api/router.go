package api

import (
	"blog/internal/api/v1/auth"
	"blog/internal/middleware"
	auth2 "blog/internal/service/auth"
	"github.com/gin-gonic/gin"
)

// Router 路由
type Router struct {
	authCtrl *auth.Controller
}

// NewRouter 创建路由
func NewRouter(
	authSvc auth2.AuthService,
) *Router {
	return &Router{
		authCtrl: auth.NewController(authSvc),
	}
}

// Setup 设置路由
func (r *Router) Setup(engine *gin.Engine) {
	// 全局中间件
	engine.Use(middleware.Recovery())
	engine.Use(middleware.Logger())
	engine.Use(middleware.CORS())

	// 健康检查
	engine.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "core-coach API is running",
		})
	})

	// API v1 路由组
	v1 := engine.Group("/api/v1")
	{
		// 认证路由组
		r.authCtrl.RegisterRouter(v1)
	}

}
