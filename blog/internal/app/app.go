package app

import (
	"blog/internal/api"
	auth3 "blog/internal/cache/auth"
	"blog/internal/model/entity"
	"blog/internal/repository/article"
	"blog/internal/repository/auth"
	"blog/internal/repository/category"
	commentRepo "blog/internal/repository/comment"
	"blog/internal/repository/tag"
	userRepo "blog/internal/repository/user"
	articleSvc "blog/internal/service/article"
	auth2 "blog/internal/service/auth"
	categorySvc "blog/internal/service/category"
	commentSvc "blog/internal/service/comment"
	tagSvc "blog/internal/service/tag"
	userSvc "blog/internal/service/user"
	"blog/pkg/config"
	"blog/pkg/database"
	"blog/pkg/logger"
	minioPkg "blog/pkg/minio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// App 应用结构体
type App struct {
	cfg          *config.Config
	mysqlDB      *gorm.DB
	redis        *redis.Client
	minioClient  *minioPkg.Client
	router       *api.Router
	server       *http.Server
	workerCtx    context.Context
	workerCancel context.CancelFunc
}

// NewApp 创建应用实例
func NewApp() *App {
	return &App{}
}

// Initialize 初始化应用
func (a *App) Initialize() error {
	// 1. 加载配置
	if err := a.initConfig(); err != nil {
		return err
	}

	// 2. 初始化日志
	if err := a.initLogger(); err != nil {
		return err
	}

	// 3. 初始化数据库
	if err := a.initDatabase(); err != nil {
		return err
	}

	// 4. 初始化 MinIO
	if err := a.initMinIO(); err != nil {
		return err
	}

	// 5. 初始化依赖
	a.initDependencies()

	// 5. 初始化路由
	a.initRouter()

	// 6. 初始化服务器
	a.initServer()

	return nil
}

// initConfig 加载配置
func (a *App) initConfig() error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	a.cfg = cfg
	return nil
}

// initLogger 初始化日志
func (a *App) initLogger() error {
	if err := logger.Init(&a.cfg.Log); err != nil {
		return fmt.Errorf("日志初始化失败: %w", err)
	}

	// 打印启动横幅
	logger.Info("=========================================")
	logger.Info(fmt.Sprintf("欢迎使用 %s", a.cfg.App.Name))
	logger.Info(fmt.Sprintf("版本: %s", a.cfg.App.Version))
	logger.Info(fmt.Sprintf("模式: %s", a.cfg.App.Mode))
	logger.Info("配置加载成功")
	logger.Info("=========================================")

	return nil
}

// initDatabase 初始化数据库
func (a *App) initDatabase() error {
	// 初始化 MySQL
	mysqlDB, err := database.InitMySQL(&a.cfg.Database.MySQL)
	if err != nil {
		return fmt.Errorf("MySQL 初始化失败: %w", err)
	}
	a.mysqlDB = mysqlDB

	//自动迁移数据库表
	//logger.Info("开始数据库迁移...")
	if err := a.mysqlDB.AutoMigrate(
		// 用户相关
		&entity.User{},

		// 文章相关
		&entity.Article{},
		&entity.Tag{},
		&entity.TagArticle{},
		&entity.Category{},
		&entity.Comment{},
		&entity.Like{},
	); err != nil {
		logger.Warn("数据库迁移警告", zap.Error(err))
	} else {
		logger.Info("数据库迁移完成")
	}
	// 初始化 Redis（可选）
	rs, err := database.InitRedis(&a.cfg.Database.Redis)
	if err != nil {
		logger.Warn("Redis 初始化失败，将不影响核心功能", zap.Error(err))
	}
	a.redis = rs

	return nil
}

// initMinIO 初始化 MinIO 客户端
func (a *App) initMinIO() error {
	client, err := minioPkg.NewClient(a.cfg.Minio)
	if err != nil {
		return fmt.Errorf("MinIO 初始化失败: %w", err)
	}
	a.minioClient = client
	logger.Info("MinIO 客户端初始化成功")
	return nil
}

// initDependencies 初始化依赖注入
func (a *App) initDependencies() {
	a.workerCtx, a.workerCancel = context.WithCancel(context.Background())

	// 创建 Repository
	authCache := auth3.NewLoginCache()
	authRepo := auth.NewAuthRepository(a.mysqlDB)
	uRepo := userRepo.NewUserRepository(a.mysqlDB)
	aRepo := article.NewArticleRepository(a.mysqlDB)
	cRepo := commentRepo.NewCommentRepository(a.mysqlDB)
	catRepo := category.NewCategoryRepository(a.mysqlDB)
	tRepo := tag.NewTagRepository(a.mysqlDB)

	// 创建 Service
	authSvc := auth2.NewAuthService(authRepo, authCache)
	uSvc := userSvc.NewUserService(uRepo, a.minioClient, authSvc, authCache)
	aSvc := articleSvc.NewArticleService(aRepo)
	cSvc := commentSvc.NewCommentService(cRepo)
	catSvc := categorySvc.NewCategoryService(catRepo)
	tSvc := tagSvc.NewTagService(tRepo)

	// 设置文章服务的额外依赖（用户、分类、标签仓库和 MinIO）
	aSvc.SetDeps(uRepo, catRepo, tRepo, a.minioClient)

	// 创建 Router
	a.router = api.NewRouter(authSvc, uSvc, aSvc, cSvc, catSvc, tSvc)
}

// initRouter 初始化路由
func (a *App) initRouter() {
	// 设置 Gin 模式
	gin.SetMode(a.cfg.App.Mode)
}

// initServer 初始化 HTTP 服务器
func (a *App) initServer() {
	engine := gin.New()

	// 注册路由
	a.router.Setup(engine)

	// 创建 HTTP 服务器
	a.server = &http.Server{
		Addr:           fmt.Sprintf(":%d", a.cfg.App.Port),
		Handler:        engine,
		ReadTimeout:    300 * time.Second,
		WriteTimeout:   300 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}
}

// Run 开始（纯http）
func (a *App) Run() {

	// 启动 HTTP 服务器
	go func() {
		logger.Info("HTTP 服务器启动",
			zap.String("addr", a.server.Addr),
			zap.String("mode", a.cfg.App.Mode),
		)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("HTTP 服务器启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	a.gracefulShutdown()
}

// gracefulShutdown 优雅关闭
func (a *App) gracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	if a.workerCancel != nil {
		a.workerCancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭 HTTP 服务器
	if err := a.server.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭失败", zap.Error(err))
	}

	// 关闭数据库连接
	_ = database.CloseMySQL()
	_ = database.CloseRedis()

	// 同步日志
	_ = logger.Sync()

	logger.Info("服务器已关闭")
	logger.Info("=========================================")
}
