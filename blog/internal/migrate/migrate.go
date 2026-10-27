// Package migrate 把 MySQL 里的业务数据整体搬到 PostgreSQL。
//
// 为什么做成子命令而不是独立二进制：
// 服务器上只有 alpine + blog-server 一个二进制，没有 Go 工具链，也不该为了
// 一次迁移往服务器装环境。把迁移逻辑编进主程序后，服务器上直接复用已有镜像：
//
//	# 本地
//	go run ./cmd/server migrate -src "root:pw@tcp(127.0.0.1:3306)/blog-project?..." \
//	                            -dst "postgres://postgres:pw@127.0.0.1:5432/blog-project?sslmode=disable"
//
//	# 服务器（用已部署的镜像，不必重新构建）
//	docker run --rm --network blog-network \
//	  -e MIGRATE_SRC=... -e MIGRATE_DST=... \
//	  <镜像> migrate -force
//
// 安全设计：目标库已有数据时默认拒绝执行（防止误把线上新数据冲掉），
// 必须显式加 -force。
package migrate

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"blog/internal/model/entity"

	"github.com/jackc/pgx/v5"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	envSrc = "MIGRATE_SRC"
	envDst = "MIGRATE_DST"

	// 单批插入行数。文章正文较大，200 行一批在 2C4G 上足够稳。
	batchSize = 200
)

// textChecks 是需要额外做「字节数」校验的表与列。
//
// 行数一致不代表内容一致：编码转换出错、text 被截断，行数都还是对的。
// 以字节数比对能抓住这一类问题（MySQL 的 LENGTH 与 PG 的 octet_length 都返回字节数）。
var textChecks = []struct {
	Table  string
	Column string
}{
	{"articles", "content"},
	{"comments", "content"},
}

// Run 执行迁移。args 是子命令之后的参数（不含 "migrate" 本身）。
func Run(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	srcFlag := fs.String("src", os.Getenv(envSrc), "源 MySQL DSN，例如 root:pw@tcp(127.0.0.1:3306)/blog-project?charset=utf8mb4&parseTime=True&loc=Local")
	dstFlag := fs.String("dst", os.Getenv(envDst), "目标 PostgreSQL DSN，URL 或 key=value 均可，例如 \"host=127.0.0.1 port=5432 user=postgres password=pw dbname=blog-project sslmode=disable TimeZone=Asia/Shanghai\"")
	force := fs.Bool("force", false, "目标库已有数据时仍然覆盖（会 TRUNCATE 所有表）")
	skipTruncate := fs.Bool("skip-truncate", false, "不清理目标表，直接追加；仅用于补数据，正常迁移别用")
	dryRun := fs.Bool("dry-run", false, "只连源库统计行数，不写目标库")
	if err := fs.Parse(args); err != nil {
		return err
	}

	srcDSN := strings.TrimSpace(*srcFlag)
	dstDSN := strings.TrimSpace(*dstFlag)
	if srcDSN == "" {
		return fmt.Errorf("必须提供 -src（或设置环境变量 %s）", envSrc)
	}
	if dstDSN == "" && !*dryRun {
		return fmt.Errorf("必须提供 -dst（或设置环境变量 %s）", envDst)
	}

	fmt.Println("========================================")
	fmt.Println("MySQL → PostgreSQL 数据迁移")
	fmt.Printf("  源   %s\n", maskDSN(srcDSN))
	if dstDSN != "" {
		fmt.Printf("  目标 %s\n", maskDSN(dstDSN))
	}
	fmt.Println("========================================")

	src, err := openMySQL(srcDSN)
	if err != nil {
		return err
	}
	defer closeDB(src)

	// 源库连通性 + 行数摸底
	srcCounts, err := countTables(src)
	if err != nil {
		return fmt.Errorf("读取源库行数失败: %w", err)
	}
	fmt.Println("\n源库（MySQL）行数：")
	var srcTotal int64
	for _, t := range entity.AllTables() {
		fmt.Printf("  %-20s %d\n", t, srcCounts[t])
		srcTotal += srcCounts[t]
	}
	fmt.Printf("  %-20s %d\n", "合计", srcTotal)

	if *dryRun {
		fmt.Println("\n-dry-run：仅统计，未写入目标库")
		return nil
	}

	if err := ensureDatabase(dstDSN); err != nil {
		return err
	}

	dst, err := openPostgres(dstDSN)
	if err != nil {
		return err
	}
	defer closeDB(dst)

	fmt.Println("\n目标库建表（AutoMigrate）...")
	if err := dst.AutoMigrate(entity.AllModels()...); err != nil {
		return fmt.Errorf("目标库建表失败: %w", err)
	}
	fmt.Println("  完成")

	if !*skipTruncate {
		occupied, err := countTables(dst)
		if err != nil {
			return fmt.Errorf("读取目标库行数失败: %w", err)
		}
		var dstTotal int64
		for _, t := range entity.AllTables() {
			dstTotal += occupied[t]
		}
		if dstTotal > 0 && !*force {
			return fmt.Errorf(
				"目标库已有 %d 行数据。确认要覆盖请加 -force（先备份！），否则请换一个空库",
				dstTotal,
			)
		}
	}

	fmt.Println("\n开始逐表迁移：")
	start := time.Now()
	for _, table := range entity.AllTables() {
		if !*skipTruncate {
			if err := dst.Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE").Error; err != nil {
				return fmt.Errorf("清理表 %s 失败: %w", table, err)
			}
		}

		n, err := copyTable(src, dst, table)
		if err != nil {
			return fmt.Errorf("迁移表 %s 失败: %w", table, err)
		}

		// 自增序列必须手工校准：迁移保留了原始 id，但 PostgreSQL 的
		// 序列不会因为手工 INSERT 而前进，不修的话下一条新记录会主键冲突。
		if err := resetSequence(dst, table); err != nil {
			fmt.Printf("  %-20s %5d 行  ⚠ 序列校准失败（不影响已迁移数据）: %v\n", table, n, err)
			continue
		}
		fmt.Printf("  %-20s %5d 行\n", table, n)
	}
	fmt.Printf("\n耗时 %s\n", time.Since(start).Round(time.Millisecond))

	fmt.Println("\n校验中...")
	if err := verify(src, dst); err != nil {
		return err
	}

	fmt.Println("\n========================================")
	fmt.Println("迁移完成，校验全部通过")
	fmt.Println("下一步：确认 configs/config.yaml 的 database.postgres 段指向本库，再重启后端")
	fmt.Println("========================================")
	return nil
}

// openMySQL 连接源库。源库只读，任何写操作都不该发生。
func openMySQL(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接源库 MySQL 失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("源库 MySQL Ping 失败: %w", err)
	}
	return db, nil
}

// openPostgres 连接目标库。
func openPostgres(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接目标库 PostgreSQL 失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("目标库 PostgreSQL Ping 失败: %w", err)
	}
	return db, nil
}

// ensureDatabase 目标库不存在时创建它。
//
// 用 pgx.ParseConfig 解析 DSN，而不是自己解析 URL：
// 这样 URL 形式和 libpq 的 key=value 形式都能支持，且不用重复实现一遍引号规则。
// PostgreSQL 的 CREATE DATABASE 既不支持 IF NOT EXISTS，也不允许在事务里执行，
// 所以这里走裸 pgx 连接，不经过 GORM。
func ensureDatabase(dstDSN string) error {
	cc, err := pgx.ParseConfig(dstDSN)
	if err != nil {
		return fmt.Errorf("解析目标 DSN 失败: %w", err)
	}
	dbName := cc.Database
	if dbName == "" {
		return errors.New("目标 DSN 里没有库名（dbname / db）")
	}
	if !validIdent(dbName) {
		return fmt.Errorf("库名 %q 含非法字符，只允许字母、数字、下划线与横线", dbName)
	}

	// 连到 postgres 系统库去建目标库
	admin := cc.Copy()
	admin.Database = "postgres"

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := pgx.ConnectConfig(ctx, admin)
	if err != nil {
		return fmt.Errorf("连接 PostgreSQL 服务失败: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	var exists int
	err = conn.QueryRow(ctx, "SELECT 1 FROM pg_database WHERE datname = $1", dbName).Scan(&exists)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// 不存在，往下建
	case err != nil:
		return fmt.Errorf("查询目标库是否存在失败: %w", err)
	default:
		fmt.Printf("\n目标库 %s 已存在\n", dbName)
		return nil
	}

	fmt.Printf("\n目标库 %s 不存在，正在创建\n", dbName)
	// 建库语句不支持占位符，dbName 已通过 validIdent 校验，不接受外部输入
	if _, err := conn.Exec(ctx, `CREATE DATABASE "`+dbName+`"`); err != nil {
		return fmt.Errorf("创建目标库 %s 失败: %w", dbName, err)
	}
	return nil
}

// copyTable 搬一张表。返回搬运行数。
//
// 用 SELECT * 读成 map 再写目标库，而不是复用 entity 结构体：
// entity 之后会加上向量相关的新列（vector_status 等），源库没有这些列，
// 按结构体读会直接报 unknown column。走 map 的话，
// 源库有几个列就搬几个列，目标库多出来的列自动走 DEFAULT。
func copyTable(src, dst *gorm.DB, table string) (int64, error) {
	targetCols, err := columnSet(dst, table)
	if err != nil {
		return 0, err
	}

	var rows []map[string]interface{}
	if err := src.Table(table).Order("id ASC").Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("读取源表失败: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}

	cleaned := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		out := make(map[string]interface{}, len(row))
		for k, v := range row {
			// 目标库没有的列直接丢弃：多为源库的历史遗留列
			if !targetCols[k] {
				continue
			}
			out[k] = normalize(v)
		}
		cleaned = append(cleaned, out)
	}

	if err := dst.Table(table).CreateInBatches(&cleaned, batchSize).Error; err != nil {
		return 0, fmt.Errorf("写入目标表失败（共 %d 行，每批 %d）: %w", len(cleaned), batchSize, err)
	}
	return int64(len(cleaned)), nil
}

// columnSet 取目标表的列名集合。
func columnSet(db *gorm.DB, table string) (map[string]bool, error) {
	cols, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		return nil, fmt.Errorf("读取目标表 %s 的列失败: %w", table, err)
	}
	set := make(map[string]bool, len(cols))
	for _, c := range cols {
		set[c.Name()] = true
	}
	return set, nil
}

// normalize 把 MySQL 驱动返回的、pgx 不接受的类型转成等价类型。
func normalize(v interface{}) interface{} {
	switch x := v.(type) {
	case []byte:
		// MySQL 的 text/blob 列扫进 map 是 []byte，直接写 text 列 pgx 会报类型不匹配
		return string(x)
	case uint64:
		// PostgreSQL 没有无符号整型，pgx 不接受 uint64 写 bigint
		return int64(x)
	default:
		return v
	}
}

// resetSequence 把自增序列推到当前最大 id，避免新记录主键冲突。
func resetSequence(db *gorm.DB, table string) error {
	// 表名来自 entity.AllTables() 的常量，不接受外部输入，拼接是安全的
	stmt := fmt.Sprintf(
		`SELECT setval(
			pg_get_serial_sequence('%s', 'id'),
			COALESCE((SELECT MAX(id) FROM %s), 1),
			EXISTS(SELECT 1 FROM %s)
		)`,
		table, table, table,
	)
	return db.Exec(stmt).Error
}

// countTables 统计各表行数。
func countTables(db *gorm.DB) (map[string]int64, error) {
	result := make(map[string]int64, len(entity.AllTables()))
	for _, t := range entity.AllTables() {
		var n int64
		if err := db.Table(t).Count(&n).Error; err != nil {
			return nil, fmt.Errorf("统计表 %s 失败: %w", t, err)
		}
		result[t] = n
	}
	return result, nil
}

// verify 校验行数与文本字节数。
func verify(src, dst *gorm.DB) error {
	srcCounts, err := countTables(src)
	if err != nil {
		return err
	}
	dstCounts, err := countTables(dst)
	if err != nil {
		return err
	}

	fmt.Println("行数比对：")
	failed := false
	for _, t := range entity.AllTables() {
		mark := "OK"
		if srcCounts[t] != dstCounts[t] {
			mark = "不一致"
			failed = true
		}
		fmt.Printf("  %-20s 源 %5d / 目标 %5d  %s\n", t, srcCounts[t], dstCounts[t], mark)
	}

	fmt.Println("文本内容比对（字节数）：")
	for _, c := range textChecks {
		var srcBytes, dstBytes int64
		// MySQL 的 LENGTH 与 PG 的 octet_length 都返回字节数，可直接比
		if err := src.Raw(fmt.Sprintf("SELECT COALESCE(SUM(LENGTH(%s)),0) FROM %s", c.Column, c.Table)).Scan(&srcBytes).Error; err != nil {
			return fmt.Errorf("统计源表 %s.%s 失败: %w", c.Table, c.Column, err)
		}
		if err := dst.Raw(fmt.Sprintf("SELECT COALESCE(SUM(octet_length(%s)),0) FROM %s", c.Column, c.Table)).Scan(&dstBytes).Error; err != nil {
			return fmt.Errorf("统计目标表 %s.%s 失败: %w", c.Table, c.Column, err)
		}
		mark := "OK"
		if srcBytes != dstBytes {
			mark = "不一致"
			failed = true
		}
		fmt.Printf("  %-20s 源 %d / 目标 %d 字节  %s\n", c.Table+"."+c.Column, srcBytes, dstBytes, mark)
	}

	if failed {
		return errors.New("校验未通过：行数或文本内容与源库不一致，请勿切换流量")
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

// maskDSN 打印连接串前把密码打码。
//
// 刻意不用 url.Parse 来判断：`root:pw@tcp(host:port)/db` 这种 MySQL DSN 会被
// 当成 scheme=root 的 URL 解析，再用 String() 重建后变成
// `root:****@//tcp/(127.0.0.1:3306)/...`——打印出来像连接串被改坏了（真踩过）。
// 纯字符串处理最可控。
func maskDSN(dsn string) string {
	// postgres://user:pass@host:port/db：只处理 "://" 之后、首个 "@" 之前的凭证段
	if i := strings.Index(dsn, "://"); i > 0 {
		rest := dsn[i+3:]
		at := strings.Index(rest, "@")
		if at <= 0 {
			return dsn
		}
		cred := rest[:at]
		colon := strings.Index(cred, ":")
		if colon <= 0 {
			return dsn
		}
		return dsn[:i+3] + cred[:colon] + ":****" + rest[at:]
	}
	// user:pass@tcp(host:port)/db：用户名与密码的分界是 "@" 之前的**第一个** ":"，
	// 与 go-sql-driver 的 ParseDSN 一致（它也用 strings.Index），
	// 这样密码里含 ":" 时也能正确打码。
	if at := strings.Index(dsn, "@"); at > 0 {
		if colon := strings.Index(dsn[:at], ":"); colon > 0 {
			return dsn[:colon+1] + "****" + dsn[at:]
		}
	}
	return dsn
}

func closeDB(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
