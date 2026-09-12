//go:build integration

// 认证仓储的集成测试。覆盖「按账号/邮箱查用户 → 判断是否已注册 → 创建用户 → 改密码」
// 这条完整链路，全部对着真实 MySQL 执行。
//
// 运行方式：
//
//	TEST_MYSQL_DSN='root:test@tcp(127.0.0.1:3306)/blog_test?charset=utf8mb4&parseTime=True&loc=Local' \
//	  go test -tags=integration -v ./internal/repository/auth/...
package auth_test

import (
	"blog/internal/repository/auth"
	"blog/internal/repository/repostest"
	"os"
	"testing"

	"gorm.io/gorm"
)

func setup(t *testing.T) (*gorm.DB, auth.UserAuthRepository) {
	t.Helper()

	// 每个测试包用独立的库（blog_test_auth），保证 `go test ./...` 并行执行时互不干扰
	db, err := repostest.Open(os.Getenv(repostest.DSNEnv), "auth")
	if err != nil {
		t.Fatalf("初始化测试库失败: %v", err)
	}
	if db == nil {
		t.Skipf("未设置 %s，跳过集成测试", repostest.DSNEnv)
	}
	t.Cleanup(func() { _ = repostest.Close(db) })

	if err := repostest.Cleanup(db, "users"); err != nil {
		t.Fatalf("清理 users 表失败: %v", err)
	}
	return db, auth.NewAuthRepository(db)
}

// 查不到用户时应当返回 (nil, nil)，而不是把 gorm.ErrRecordNotFound 抛给上层。
// 这是仓储层对 service 的契约，用 mock 验证等于自说自话，只有真库能证明。
func TestFindUserByAccount_NotFoundReturnsNil(t *testing.T) {
	_, repo := setup(t)

	u, err := repo.FindUserByAccount("19900000000")
	if err != nil {
		t.Fatalf("期望无错误，实际: %v", err)
	}
	if u != nil {
		t.Fatalf("期望返回 nil，实际: %+v", u)
	}
}

func TestFindUserByAccount(t *testing.T) {
	db, repo := setup(t)

	seeded, err := repostest.SeedUser(db, "叶子", "19900000001", "leaf@example.com", 0, 1)
	if err != nil {
		t.Fatalf("造数据失败: %v", err)
	}

	got, err := repo.FindUserByAccount("19900000001")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got == nil {
		t.Fatal("期望查到用户，实际为 nil")
	}
	if got.ID != seeded.ID {
		t.Errorf("ID 不符：期望 %d，实际 %d", seeded.ID, got.ID)
	}
	if got.UserName != "叶子" {
		t.Errorf("用户名不符：期望 叶子，实际 %s", got.UserName)
	}
	if got.RoleID != 0 {
		t.Errorf("角色不符：期望 0，实际 %d", got.RoleID)
	}
}

func TestFindUserByEmail(t *testing.T) {
	db, repo := setup(t)

	if _, err := repostest.SeedUser(db, "叶子", "19900000002", "leaf@example.com", 0, 1); err != nil {
		t.Fatalf("造数据失败: %v", err)
	}

	got, err := repo.FindUserByEmail("leaf@example.com")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got == nil {
		t.Fatal("期望查到用户，实际为 nil")
	}

	missing, err := repo.FindUserByEmail("nobody@example.com")
	if err != nil {
		t.Fatalf("期望无错误，实际: %v", err)
	}
	if missing != nil {
		t.Fatalf("期望 nil，实际: %+v", missing)
	}
}

func TestIsExistsByNameAndAccount(t *testing.T) {
	db, repo := setup(t)

	if _, err := repostest.SeedUser(db, "叶子", "19900000003", "leaf@example.com", 0, 1); err != nil {
		t.Fatalf("造数据失败: %v", err)
	}

	exists, err := repo.IsExistsByName("叶子")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if !exists {
		t.Error("用户名「叶子」应判定为已存在")
	}

	exists, err = repo.IsExistsByAccount("19900000003")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if !exists {
		t.Error("账号 19900000003 应判定为已存在")
	}

	exists, err = repo.IsExistsByAccount("19900000099")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if exists {
		t.Error("账号 19900000099 应判定为不存在")
	}
}

// 注册后必须能立刻查到，且默认状态为启用。
// 用户表在 account / user_name 上有唯一索引——这两个约束只在真实 MySQL 上生效。
func TestCreateUser(t *testing.T) {
	_, repo := setup(t)

	if err := repo.CreateUser("新用户", "19900000004", "hash", 0); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	got, err := repo.FindUserByAccount("19900000004")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got == nil {
		t.Fatal("创建后应当能查到用户")
	}
	if got.UserName != "新用户" {
		t.Errorf("用户名不符：期望 新用户，实际 %s", got.UserName)
	}
	if got.Status != 1 {
		t.Errorf("新用户状态应为 1（启用），实际 %d", got.Status)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at 未自动填充")
	}
}

// 重复账号必须被数据库唯一索引挡住。
// 单测里 mock 仓储永远发现不了这条约束是否真的建起来了。
func TestCreateUser_DuplicateAccountIsRejected(t *testing.T) {
	_, repo := setup(t)

	if err := repo.CreateUser("用户甲", "19900000005", "hash", 0); err != nil {
		t.Fatalf("首次创建失败: %v", err)
	}

	err := repo.CreateUser("用户乙", "19900000005", "hash", 0)
	if err == nil {
		t.Fatal("重复账号应当被唯一索引拒绝，实际创建成功")
	}
}

func TestUpdateUserPassword(t *testing.T) {
	db, repo := setup(t)

	seeded, err := repostest.SeedUser(db, "叶子", "19900000006", "leaf@example.com", 0, 1)
	if err != nil {
		t.Fatalf("造数据失败: %v", err)
	}

	if err := repo.UpdateUserPassword(int(seeded.ID), "new-hash-value"); err != nil {
		t.Fatalf("改密码失败: %v", err)
	}

	got, err := repo.FindUserByAccount("19900000006")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got.PasswordHash != "new-hash-value" {
		t.Errorf("密码未更新：期望 new-hash-value，实际 %s", got.PasswordHash)
	}
	// 其他字段不应被这次更新波及
	if got.UserName != "叶子" {
		t.Errorf("更新密码不应影响用户名，实际变成了 %s", got.UserName)
	}
}
