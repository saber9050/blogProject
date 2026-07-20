package user

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台用户管理路由
func (ctrl *UserController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/users", ctrl.ListUsers)
	r.POST("/users", ctrl.CreateUser)
	r.PUT("/users/:id", ctrl.UpdateUser)
	r.DELETE("/users/:id", ctrl.DeleteUser)
}
