// Package repostest 是仓储层集成测试的公共辅助包。
//
// 它只服务于带 `integration` 构建标签的测试文件，不参与任何生产代码路径。
// 集成测试需要一个可写的真实 MySQL，连接串通过 TEST_MYSQL_DSN 传入：
//
//	TEST_MYSQL_DSN='root:test@tcp(127.0.0.1:3306)/blog_test?charset=utf8mb4&parseTime=True&loc=Local' \
//	  go test -tags=integration ./internal/repository/...
//
// DSN 里的库名只作为前缀，每个测试包会在它后面拼上自己的 schema，
// 实际使用 blog_test_auth、blog_test_user 这样的独立库——原因见 Open 的注释。
package repostest

import (
	"blog/internal/model/entity"
	"database/sql"
	"fmt"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DSNEnv 是测试库连接串所在的环境变量名。
const DSNEnv = "TEST_MYSQL_DSN"

// Open 打开测试库并自动建表。
//
// dsn 为空时返回 (nil, nil)，调用方据此判断"没有测试库"并跳过用例——
// 这样本地不装 MySQL 也能跑 `go test ./...`，不会被集成测试拖累。
//
// schema 必填，实际连接的库是 `<DSN 中的库名>_<schema>`。之所以要按包隔离，
// 是因为 `go test ./...` 默认并行执行不同的测试包：如果 auth 和 user 两个包
// 共用一个库，它们各自的清表操作会互相踩，出现"单跑通过、一起跑就挂"的灵异现象。
// 每包一个库之后，并行执行才是安全的。
func Open(dsn, schema string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, nil
	}

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", DSNEnv, err)
	}
	if cfg.DBName == "" {
		return nil, fmt.Errorf("%s 必须带库名，例如 root:pw@tcp(127.0.0.1:3306)/blog_test?parseTime=True", DSNEnv)
	}
	if schema == "" {
		return nil, fmt.Errorf("schema 不能为空：每个测试包必须使用独立的库，否则并行执行时会互相清表")
	}

	target := cfg.DBName + "_" + schema
	if err := ensureDatabase(cfg, target); err != nil {
		return nil, err
	}

	targetCfg := *cfg
	targetCfg.DBName = target

	db, err := gorm.Open(gormmysql.Open(targetCfg.FormatDSN()), &gorm.Config{
		// 测试时关掉 GORM 自带日志，失败信息靠断言表达，日志只会淹没输出
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		// 与 pkg/database/mysql.go 保持一致的连接参数，否则测试环境与线上行为有差异
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接测试数据库 %s 失败: %w", target, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("测试数据库 Ping 失败: %w", err)
	}

	// 建表清单必须与 internal/app/app.go 中的 AutoMigrate 保持一致，
	// 否则测试库结构和线上不一致，测试通过也说明不了问题。
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.Article{},
		&entity.Tag{},
		&entity.TagArticle{},
		&entity.Category{},
		&entity.Comment{},
		&entity.Like{},
		&entity.ArticleImage{},
		&entity.About{},
	); err != nil {
		return nil, fmt.Errorf("测试库建表失败: %w", err)
	}

	return db, nil
}

// ensureDatabase 连接到 MySQL 服务端（不指定库）并创建目标库。
func ensureDatabase(cfg *mysqldriver.Config, name string) error {
	serverCfg := *cfg
	serverCfg.DBName = ""

	conn, err := sql.Open("mysql", serverCfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("连接 MySQL 服务端失败: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// 建库语句不支持占位符，name 由 DSN 库名拼接而来，不接受外部输入
	stmt := "CREATE DATABASE IF NOT EXISTS `" + name + "` " +
		"CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"
	if _, err := conn.Exec(stmt); err != nil {
		return fmt.Errorf("创建测试库 %s 失败（该账号需要 CREATE 权限）: %w", name, err)
	}
	return nil
}

// Cleanup 清空指定的表，保证用例之间互不污染。
//
// 表名由调用方以字面量传入，不接受外部输入，因此这里直接拼接；
// 有外键依赖时需要先清子表再清主表。
func Cleanup(db *gorm.DB, tables ...string) error {
	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			return fmt.Errorf("清理表 %s 失败: %w", table, err)
		}
	}
	return nil
}

// Close 关闭测试库连接。
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SeedUser 插入一个测试用户并返回其 ID，供多个用例复用。
// 注意：这里直接写库，绕过仓储方法，避免"用被测代码造测试数据"的循环依赖。
func SeedUser(db *gorm.DB, name, account, email string, roleID int8, status int8) (*entity.User, error) {
	u := entity.User{
		UserName:     name,
		Account:      account,
		PasswordHash: "$2a$10$testtesttesttesttesttesttesttesttesttesttesttesttesttesttest",
		Email:        email,
		Status:       status,
		RoleID:       roleID,
	}
	if err := db.Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
