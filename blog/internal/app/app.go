package app

import (
	"blog/internal/api"
	auth3 "blog/internal/cache/auth"
	"blog/internal/middleware"
	"blog/internal/model/entity"
	"blog/internal/repository/about"
	"blog/internal/repository/article"
	"blog/internal/repository/auth"
	"blog/internal/repository/category"
	commentRepo "blog/internal/repository/comment"
	llmconfigRepo "blog/internal/repository/llmconfig"
	"blog/internal/repository/tag"
	userRepo "blog/internal/repository/user"
	aboutSvc "blog/internal/service/about"
	articleSvc "blog/internal/service/article"
	auth2 "blog/internal/service/auth"
	categorySvc "blog/internal/service/category"
	commentSvc "blog/internal/service/comment"
	llmSvc "blog/internal/service/llm"
	llmconfigSvc "blog/internal/service/llmconfig"
	tagSvc "blog/internal/service/tag"
	userSvc "blog/internal/service/user"
	"blog/pkg/config"
	"blog/pkg/database"
	"blog/pkg/logger"
	minioPkg "blog/pkg/minio"
	"blog/pkg/utils"
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
	pgDB         *gorm.DB
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

	// 4. 初始化默认管理员
	a.initDefaultAdmin()

	// 5. 初始化 MinIO
	if err := a.initMinIO(); err != nil {
		return err
	}

	// 6. 初始化依赖
	a.initDependencies()

	// 7. 初始化路由
	a.initRouter()

	// 8. 初始化服务器
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
	// 初始化 PostgreSQL（主库：业务表 + 后续的向量表同库，便于放进同一事务）
	pgDB, err := database.InitPostgres(&a.cfg.Database.Postgres)
	if err != nil {
		return fmt.Errorf("PostgreSQL 初始化失败: %w", err)
	}
	a.pgDB = pgDB

	// 同步表结构（AutoMigrate 只建表/加列，不搬数据）
	//
	// 日志特意不写「数据库迁移」：那四个字容易被理解成「把 MySQL 数据搬过来了」，
	// 实际这里只是让表结构跟上 entity 定义。搬数据是 `migrate` 子命令干的事。
	// 建表清单统一由 entity.AllModels() 提供，启动、迁移工具、集成测试三处共用。
	logger.Info("开始同步表结构（AutoMigrate）...")
	if err := a.pgDB.AutoMigrate(entity.AllModels()...); err != nil {
		logger.Warn("表结构同步警告", zap.Error(err))
	} else {
		logger.Info("表结构同步完成（不含数据）")
	}

	// 向量表要等 AutoMigrate 之后：部分唯一索引依赖 ai_model_configs 表已存在
	a.ensureVectorSchema()

	// 初始化 Redis（可选）
	rs, err := database.InitRedis(&a.cfg.Database.Redis)
	if err != nil {
		logger.Warn("Redis 初始化失败，将不影响核心功能", zap.Error(err))
	}
	a.redis = rs

	return nil
}

// ensureVectorSchema 按当前启用的向量模型维度初始化向量表。
//
// 全程只 warn 不阻断启动：向量功能不可用，不该让整站打不开。
// 这也是把 article_chunks 排除在 AutoMigrate 之外的目的——没装 pgvector 扩展时，
// 失败的只是这一段，登录、文章、评论全都照常。
func (a *App) ensureVectorSchema() {
	// 唯一索引优先：即便下面因为没配模型而跳过，这个约束也该尽量建上
	if err := database.EnsureSingleEmbeddingModelIndex(a.pgDB); err != nil {
		logger.Warn("向量模型唯一索引创建失败（可能库内已有多条向量模型记录），已跳过",
			zap.Error(err))
	}

	var m entity.AIModelConfig
	err := a.pgDB.Where("type = ? AND status = ?", entity.ModelTypeEmbedding, 1).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Info("未配置向量模型，跳过向量表初始化")
		return
	}
	if err != nil {
		logger.Warn("读取向量模型失败，跳过向量表初始化", zap.Error(err))
		return
	}
	if m.Dimension <= 0 {
		logger.Warn("向量模型未填写维度，跳过向量表初始化",
			zap.Uint("config_id", m.ID), zap.String("model", m.Model))
		return
	}

	if err := database.EnsureSchema(a.pgDB, m.Dimension); err != nil {
		var mismatch *database.DimensionMismatchError
		if errors.As(err, &mismatch) {
			// 这是最需要人说清楚的一种错：库里的表和数据都是好的，只是与新配的模型维度不符
			logger.Warn("向量表维度与向量模型不一致，向量化功能不可用；如需切换维度请调用 /admin/vector/reset",
				zap.Int("table_dimension", mismatch.Current),
				zap.Int("model_dimension", mismatch.Want),
				zap.Int64("existing_chunks", mismatch.Chunks))
			return
		}
		logger.Warn("向量表初始化失败，向量化功能将不可用", zap.Error(err))
		return
	}
	logger.Info("向量表就绪",
		zap.Int("dimension", m.Dimension), zap.String("model", m.Model))
}

// initDefaultAdmin 初始化默认管理员
func (a *App) initDefaultAdmin() {
	if !a.cfg.Admin.Enabled {
		logger.Info("默认管理员创建已禁用")
		return
	}

	var count int64
	a.pgDB.Model(&entity.User{}).Where("role_id = ?", 1).Count(&count)
	if count > 0 {
		logger.Info("管理员账号已存在，跳过创建")
		return
	}

	hash, err := utils.HashPassword(a.cfg.Admin.Password)
	if err != nil {
		logger.Error("默认管理员密码加密失败", zap.Error(err))
		return
	}

	admin := entity.User{
		UserName:     a.cfg.Admin.Username,
		Account:      a.cfg.Admin.Account,
		PasswordHash: hash,
		RoleID:       1,
		Status:       1,
	}
	if err := a.pgDB.Create(&admin).Error; err != nil {
		logger.Error("创建默认管理员失败", zap.Error(err))
		return
	}
	logger.Info("默认管理员账号已创建")
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
	authRepo := auth.NewAuthRepository(a.pgDB)
	uRepo := userRepo.NewUserRepository(a.pgDB)
	aRepo := article.NewArticleRepository(a.pgDB)
	cRepo := commentRepo.NewCommentRepository(a.pgDB)
	catRepo := category.NewCategoryRepository(a.pgDB)
	tRepo := tag.NewTagRepository(a.pgDB)
	aboutRepo := about.NewAboutRepository(a.pgDB)
	llmCfgRepo := llmconfigRepo.NewLLMConfigRepository(a.pgDB)

	// 创建 Service
	authSvc := auth2.NewAuthService(authRepo, authCache)
	uSvc := userSvc.NewUserService(uRepo, a.minioClient, authSvc, authCache)

	// 初始化认证中间件
	middleware.InitAuth(authCache)
	aSvc := articleSvc.NewArticleService(aRepo, uRepo, catRepo, tRepo, cRepo, uSvc, a.minioClient)
	cSvc := commentSvc.NewCommentService(cRepo, a.minioClient, uRepo)
	catSvc := categorySvc.NewCategoryService(catRepo)
	tSvc := tagSvc.NewTagService(tRepo)
	aboutSvc := aboutSvc.NewAboutService(aboutRepo, a.minioClient, uRepo)
	llmCfgSvc := llmconfigSvc.NewLLMConfigService(llmCfgRepo)
	lSvc := llmSvc.NewLLMService(llmCfgSvc)

	// 创建 Router
	a.router = api.NewRouter(authSvc, uSvc, aSvc, cSvc, catSvc, tSvc, aboutSvc, lSvc, llmCfgSvc)
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
	_ = database.ClosePostgres()
	_ = database.CloseRedis()

	// 同步日志
	_ = logger.Sync()

	logger.Info("服务器已关闭")
	logger.Info("=========================================")
}
