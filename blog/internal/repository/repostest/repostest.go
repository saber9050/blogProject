// Package repostest 是仓储层集成测试的公共辅助包。
//
// 它只服务于带 `integration` 构建标签的测试文件，不参与任何生产代码路径。
// 集成测试需要一个可写的真实 PostgreSQL，连接串通过 TEST_PG_DSN 传入：
//
//	TEST_PG_DSN='postgres://postgres:test@127.0.0.1:5432/blog_test?sslmode=disable' \
//	  go test -tags=integration ./internal/repository/...
//
// DSN 里的库名只作为前缀，每个测试包会在它后面拼上自己的 schema，
// 实际使用 blog_test_auth、blog_test_user 这样的独立库——原因见 Open 的注释。
package repostest

import (
	"blog/internal/model/entity"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // 注册 pgx，用于建库
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DSNEnv 是测试库连接串所在的环境变量名。
const DSNEnv = "TEST_PG_DSN"

// Open 打开测试库并自动建表。
//
// dsn 为空时返回 (nil, nil)，调用方据此判断"没有测试库"并跳过用例——
// 这样本地不装 PostgreSQL 也能跑 `go test ./...`，不会被集成测试拖累。
//
// schema 必填，实际连接的库是 `<DSN 中的库名>_<schema>`。之所以要按包隔离，
// 是因为 `go test ./...` 默认并行执行不同的测试包：如果 auth 和 user 两个包
// 共用一个库，它们各自的清表操作会互相踩，出现"单跑通过、一起跑就挂"的灵异现象。
// 每包一个库之后，并行执行才是安全的。
func Open(dsn, schema string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", DSNEnv, err)
	}
	if !strings.HasPrefix(u.Scheme, "postgres") {
		return nil, fmt.Errorf("%s 必须是 postgres:// 形式，例如 postgres://postgres:pw@127.0.0.1:5432/blog_test?sslmode=disable", DSNEnv)
	}
	base := strings.TrimPrefix(u.Path, "/")
	if base == "" {
		return nil, fmt.Errorf("%s 必须带库名，例如 postgres://postgres:pw@127.0.0.1:5432/blog_test?sslmode=disable", DSNEnv)
	}
	if schema == "" {
		return nil, fmt.Errorf("schema 不能为空：每个测试包必须使用独立的库，否则并行执行时会互相清表")
	}

	target := base + "_" + schema
	if err := ensureDatabase(u, target); err != nil {
		return nil, err
	}

	targetURL := *u
	targetURL.Path = "/" + target

	db, err := gorm.Open(gormpostgres.Open(targetURL.String()), &gorm.Config{
		// 测试时关掉 GORM 自带日志，失败信息靠断言表达，日志只会淹没输出
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		// 与 pkg/database/postgres.go 保持一致的连接参数，否则测试环境与线上行为有差异
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

	// 建表清单与线上同源（entity.AllModels），否则测试库结构和线上不一致，
	// 测试通过也说明不了问题。
	if err := db.AutoMigrate(entity.AllModels()...); err != nil {
		return nil, fmt.Errorf("测试库建表失败: %w", err)
	}

	return db, nil
}

// ensureDatabase 连接到 PostgreSQL 服务端（postgres 系统库）并创建目标库。
//
// PostgreSQL 的 CREATE DATABASE 既不支持 IF NOT EXISTS，也不允许在事务里执行，
// 所以这里走裸 database/sql，不经过 GORM。
func ensureDatabase(u *url.URL, name string) error {
	if !validIdent(name) {
		return fmt.Errorf("测试库名 %q 含非法字符，只允许字母、数字、下划线与横线", name)
	}

	admin := *u
	admin.Path = "/postgres"

	conn, err := sql.Open("pgx", admin.String())
	if err != nil {
		return fmt.Errorf("连接 PostgreSQL 服务端失败: %w", err)
	}
	defer func() { _ = conn.Close() }()

	var exists int
	err = conn.QueryRow("SELECT 1 FROM pg_database WHERE datname = $1", name).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// 不存在，往下建
	case err != nil:
		return fmt.Errorf("查询测试库 %s 是否存在失败: %w", name, err)
	default:
		return nil
	}

	// 建库语句不支持占位符，name 已通过 validIdent 校验，不接受外部输入
	if _, err := conn.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		return fmt.Errorf("创建测试库 %s 失败（该账号需要 CREATEDB 权限）: %w", name, err)
	}
	return nil
}

// validIdent 粗校验标识符，防止库名里出现引号把 DDL 拼坏。
func validIdent(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
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
