package category

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册分类路由
func (ctrl *CategoryController) RegisterRoutes(r *gin.RouterGroup) {
	// 前台分类接口（无需认证）
	r.GET("/categories", ctrl.ListCategories)
}
