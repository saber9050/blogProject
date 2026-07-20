package category

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册后台分类管理路由
func (ctrl *CategoryController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/categories", ctrl.ListCategories)
	r.POST("/categories", ctrl.CreateCategory)
	r.PUT("/categories/:id", ctrl.UpdateCategory)
	r.DELETE("/categories/:id", ctrl.DeleteCategory)
}
