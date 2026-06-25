package api

import (
	"blog/internal/api/v1/admin"
	"blog/internal/api/v1/article"
	"blog/internal/api/v1/auth"
	"blog/internal/api/v1/category"
	"blog/internal/api/v1/comment"
	"blog/internal/api/v1/tag"
	"blog/internal/api/v1/user"
	"blog/internal/middleware"
	articleSvc "blog/internal/service/article"
	auth2 "blog/internal/service/auth"
	categorySvc "blog/internal/service/category"
	commentSvc "blog/internal/service/comment"
	tagSvc "blog/internal/service/tag"
	user2 "blog/internal/service/user"

	"github.com/gin-gonic/gin"
)

// Router 路由
type Router struct {
	authCtrl       *auth.Controller
	userCtrl       *user.Controller
	articleCtrl    *article.ArticleController
	adminCtrl      *admin.AdminController
	commentCtrl    *comment.Controller
	categoryCtrl   *category.CategoryController
	tagCtrl        *tag.TagController
	articleService articleSvc.ArticleService
}

// NewRouter 创建路由
func NewRouter(
	authSvc auth2.AuthService,
	userSvc user2.UserService,
	articleService articleSvc.ArticleService,
	commentSvc commentSvc.CommentService,
	categoryService categorySvc.CategoryService,
	tagService tagSvc.TagService,
) *Router {
	return &Router{
		authCtrl:       auth.NewController(authSvc),
		userCtrl:       user.NewController(userSvc),
		articleCtrl:    article.NewArticleController(articleService),
		adminCtrl:      admin.NewAdminController(userSvc, articleService, categoryService, tagService),
		commentCtrl:    comment.NewController(commentSvc),
		categoryCtrl:   category.NewCategoryController(categoryService),
		tagCtrl:        tag.NewTagController(tagService),
		articleService: articleService,
	}
}

// ArticleSvc 获取文章服务（供 worker 使用）
func (r *Router) ArticleSvc() articleSvc.ArticleService {
	return r.articleService
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
		// 认证路由组（无需认证或登录认证）
		r.authCtrl.RegisterRouter(v1)

		// 用户路由组（需登录）
		r.userCtrl.RegisterRouter(v1)

		// 前台文章路由（部分接口可选认证）
		r.articleCtrl.RegisterRoutes(v1)

		// 前台分类路由（无需认证）
		r.categoryCtrl.RegisterRoutes(v1)

		// 前台标签路由（无需认证）
		r.tagCtrl.RegisterRoutes(v1)

		// 评论路由组（挂载在 /articles 路径下）
		r.commentCtrl.RegisterRouter(v1.Group("/articles"))

		// 后台管理路由组（需管理员认证，role_id = 1）
		adminGroup := v1.Group("/admin")
		adminGroup.Use(middleware.Auth(1))
		{
			r.adminCtrl.RegisterRoutes(adminGroup)
		}
	}
}
