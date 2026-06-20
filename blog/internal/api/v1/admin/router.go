package admin

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册后台管理路由
func (ctrl *AdminController) RegisterRoutes(r *gin.RouterGroup) {
	// 用户管理
	r.GET("/users", ctrl.ListUsers)
	r.POST("/users", ctrl.CreateUser)
	r.PUT("/users/:id", ctrl.UpdateUser)
	r.DELETE("/users/:id", ctrl.DeleteUser)

	// 文章管理
	r.GET("/articles", ctrl.ListArticles)
	r.POST("/articles", ctrl.CreateArticle)
	r.PUT("/articles/:id", ctrl.UpdateArticle)
	r.DELETE("/articles/:id", ctrl.DeleteArticle)

	// 分类管理
	r.GET("/categories", ctrl.ListCategories)
	r.POST("/categories", ctrl.CreateCategory)
	r.PUT("/categories/:id", ctrl.UpdateCategory)
	r.DELETE("/categories/:id", ctrl.DeleteCategory)

	// 标签管理
	r.GET("/tags", ctrl.ListTags)
	r.POST("/tags", ctrl.CreateTag)
	r.PUT("/tags/:id", ctrl.UpdateTag)
	r.DELETE("/tags/:id", ctrl.DeleteTag)
}
