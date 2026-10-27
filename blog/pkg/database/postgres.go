package database

import (
	"blog/pkg/config"
	"blog/pkg/logger"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var pgDB *gorm.DB

const (
	defaultPGPort     = 5432
	defaultPGSSLMode  = "disable"
	defaultPGTimeZone = "Asia/Shanghai"
)

// BuildDSN 由配置拼出 PostgreSQL 连接串。
//
// 用 libpq 的 key=value 形式，不用 URL 形式。原因不是风格偏好，而是踩过坑：
// URL 形式下 url.Values.Encode() 会把 TimeZone 的值 "Asia/Shanghai" 里的 "/"
// 编码成 "%2F"，而 pgx 把这个值原样当启动参数发给服务端，PostgreSQL 直接报
//
//	unknown time zone Asia%2FShanghai
//
// 启动即 panic。key=value 形式没有这层编码，密码里的空格、单引号、反斜杠
// 按 libpq 的引号规则转义即可（见 quoteLibpq）。
func BuildDSN(cfg *config.PostgresConfig) string {
	port := cfg.Port
	if port == 0 {
		port = defaultPGPort
	}
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = defaultPGSSLMode
	}
	tz := cfg.TimeZone
	if tz == "" {
		tz = defaultPGTimeZone
	}

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		quoteLibpq(cfg.Host),
		port,
		quoteLibpq(cfg.Username),
		quoteLibpq(cfg.Password),
		quoteLibpq(cfg.Database),
		quoteLibpq(sslMode),
		quoteLibpq(tz),
	)
}

// quoteLibpq 给需要转义的值加单引号。
//
// 只在值含空白、引号或反斜杠时才加。**不能无脑全加**：GORM 的 postgres 驱动
// 用正则 `(time_zone|TimeZone|timezone)=(.*?)($|&| )` 从原始 DSN 字符串里
// 抠出时区再覆盖 RuntimeParams（driver 源码 postgres.go:107），它不会剥引号。
// 一旦写成 TimeZone='Asia/Shanghai'，被原样塞进启动参数，
// 服务端就报 `invalid value for parameter "TimeZone": "'Asia/Shanghai'"`。
func quoteLibpq(v string) string {
	if v == "" {
		return "''"
	}
	if !strings.ContainsAny(v, " \t\r\n'\"\\") {
		return v
	}
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(v) + "'"
}

// InitPostgres 初始化 PostgreSQL 连接
func InitPostgres(cfg *config.PostgresConfig) (*gorm.DB, error) {
	// 配错时给一句人话，而不是让 pgx 抛 "dial tcp :0: connect: connection refused"
	if cfg.Host == "" || cfg.Database == "" {
		return nil, fmt.Errorf("database.postgres 未配置：请在 configs/config.yaml 填写 host / port / username / password / database")
	}

	gormConfig := &gorm.Config{
		// 与 MySQL 时期保持一致：不建外键，删除顺序由业务保证
		DisableForeignKeyConstraintWhenMigrating: true,
		// 跳过默认事务，单条写入不需要包事务
		SkipDefaultTransaction: true,
	}

	if config.Get().App.Mode == "debug" {
		gormConfig.Logger = gormlogger.Default.LogMode(gormlogger.Info)
	} else {
		gormConfig.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	db, err := gorm.Open(postgres.Open(BuildDSN(cfg)), gormConfig)
	if err != nil {
		logger.Error("PostgreSQL 连接失败", zap.Error(err))
		return nil, fmt.Errorf("PostgreSQL 连接失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("获取 SQL DB 失败", zap.Error(err))
		return nil, fmt.Errorf("获取 SQL DB 失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		logger.Error("PostgreSQL Ping 失败", zap.Error(err))
		return nil, fmt.Errorf("PostgreSQL Ping 失败: %w", err)
	}

	pgDB = db
	logger.Info("PostgreSQL 连接成功",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
	)

	return db, nil
}

// GetPostgres 获取 PostgreSQL 实例
func GetPostgres() *gorm.DB {
	if pgDB == nil {
		panic("PostgreSQL 未初始化")
	}
	return pgDB
}

// ClosePostgres 关闭 PostgreSQL 连接
func ClosePostgres() error {
	if pgDB != nil {
		sqlDB, err := pgDB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
