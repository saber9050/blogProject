//go:build integration

// 用户仓储的集成测试。重点覆盖 ListByRole 这条拼了分页、关键字、状态、时间范围
// 四个条件的动态查询——它是单测最够不着、又最容易出错的一类代码。
//
// 运行方式：
//
//	TEST_MYSQL_DSN='root:test@tcp(127.0.0.1:3306)/blog_test?charset=utf8mb4&parseTime=True&loc=Local' \
//	  go test -tags=integration -v ./internal/repository/user/...
package user_test

import (
	"blog/internal/repository/repostest"
	"blog/internal/repository/user"
	"fmt"
	"os"
	"testing"

	"gorm.io/gorm"
)

func setup(t *testing.T) (*gorm.DB, user.UserRepository) {
	t.Helper()

	// 每个测试包用独立的库（blog_test_user），保证 `go test ./...` 并行执行时互不干扰
	db, err := repostest.Open(os.Getenv(repostest.DSNEnv), "user")
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
	return db, user.NewUserRepository(db)
}

// seed 批量造用户，账号从 seedAccountBase 起逐个递增保证唯一。
func seed(t *testing.T, db *gorm.DB, prefix string, n int, roleID, status int8, seedAccountBase int) {
	t.Helper()
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s%02d", prefix, i)
		account := fmt.Sprintf("199%08d", seedAccountBase+i)
		email := fmt.Sprintf("%s%02d@example.com", prefix, i)
		if _, err := repostest.SeedUser(db, name, account, email, roleID, status); err != nil {
			t.Fatalf("造第 %d 条数据失败: %v", i, err)
		}
	}
}

func TestFindByID_NotFoundReturnsNil(t *testing.T) {
	_, repo := setup(t)

	u, err := repo.FindByID(999999)
	if err != nil {
		t.Fatalf("期望无错误，实际: %v", err)
	}
	if u != nil {
		t.Fatalf("期望返回 nil，实际: %+v", u)
	}
}

func TestFindByID(t *testing.T) {
	db, repo := setup(t)

	seeded, err := repostest.SeedUser(db, "叶子", "19900000001", "leaf@example.com", 1, 1)
	if err != nil {
		t.Fatalf("造数据失败: %v", err)
	}

	got, err := repo.FindByID(seeded.ID)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got == nil {
		t.Fatal("期望查到用户，实际为 nil")
	}
	if got.Account != "19900000001" {
		t.Errorf("账号不符：期望 19900000001，实际 %s", got.Account)
	}
}

// 25 条数据、每页 10 条：前两页各 10 条，第三页 5 条，第四页 0 条。
// total 必须恒等于 25——它是分页组件的总页数来源，算错会导致翻页丢数据。
func TestListByRole_Pagination(t *testing.T) {
	db, repo := setup(t)
	seed(t, db, "普通用户", 25, 0, 1, 0)

	seen := make(map[uint]bool)

	for page := 1; page <= 3; page++ {
		list, total, err := repo.ListByRole(0, page, 10, "", nil, "", "")
		if err != nil {
			t.Fatalf("第 %d 页查询失败: %v", page, err)
		}
		if total != 25 {
			t.Fatalf("第 %d 页 total 应为 25，实际 %d", page, total)
		}

		wantLen := 10
		if page == 3 {
			wantLen = 5
		}
		if len(list) != wantLen {
			t.Fatalf("第 %d 页应有 %d 条，实际 %d", page, wantLen, len(list))
		}

		for _, u := range list {
			if seen[u.ID] {
				t.Errorf("ID %d 在不同页重复出现，分页有重叠", u.ID)
			}
			seen[u.ID] = true
		}
	}

	if len(seen) != 25 {
		t.Errorf("翻完三页应覆盖 25 条不同记录，实际 %d 条", len(seen))
	}

	// 越界的第 4 页应返回空列表，而不是报错
	list, total, err := repo.ListByRole(0, 4, 10, "", nil, "", "")
	if err != nil {
		t.Fatalf("第 4 页查询失败: %v", err)
	}
	if total != 25 {
		t.Errorf("第 4 页 total 仍应为 25，实际 %d", total)
	}
	if len(list) != 0 {
		t.Errorf("越界页应返回空列表，实际 %d 条", len(list))
	}
}

func TestListByRole_FilterByRole(t *testing.T) {
	db, repo := setup(t)
	seed(t, db, "管理员", 2, 1, 1, 0)
	seed(t, db, "普通用户", 3, 0, 1, 100)

	admins, total, err := repo.ListByRole(1, 1, 10, "", nil, "", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if total != 2 || len(admins) != 2 {
		t.Fatalf("管理员应有 2 条，实际 total=%d len=%d", total, len(admins))
	}
	for _, u := range admins {
		if u.RoleID != 1 {
			t.Errorf("按 role_id=1 过滤却查出 role_id=%d 的记录", u.RoleID)
		}
	}
}

func TestListByRole_KeywordMatchesUserName(t *testing.T) {
	db, repo := setup(t)
	seed(t, db, "张三", 2, 0, 1, 0) // 张三00 张三01
	seed(t, db, "李四", 3, 0, 1, 100)

	list, total, err := repo.ListByRole(0, 1, 10, "张三", nil, "", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("关键字「张三」应命中 2 条，实际 total=%d len=%d", total, len(list))
	}

	none, total, err := repo.ListByRole(0, 1, 10, "王五", nil, "", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if total != 0 || len(none) != 0 {
		t.Fatalf("关键字「王五」不应命中任何记录，实际 total=%d len=%d", total, len(none))
	}
}

func TestListByRole_StatusFilter(t *testing.T) {
	db, repo := setup(t)
	seed(t, db, "启用用户", 3, 0, 1, 0)
	seed(t, db, "禁用用户", 2, 0, 0, 100)

	enabled := 1
	list, total, err := repo.ListByRole(0, 1, 10, "", &enabled, "", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if total != 3 || len(list) != 3 {
		t.Fatalf("状态=1 应有 3 条，实际 total=%d len=%d", total, len(list))
	}
	for _, u := range list {
		if u.Status != 1 {
			t.Errorf("按 status=1 过滤却查出 status=%d 的记录", u.Status)
		}
	}

	disabled := 0
	_, total, err = repo.ListByRole(0, 1, 10, "", &disabled, "", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if total != 2 {
		t.Errorf("状态=0 应有 2 条，实际 %d", total)
	}
}

// 列表按 created_at 倒序返回。同秒创建的记录时间相同，
// 所以断言"后一条不晚于前一条"，而不是断言严格的先后关系。
func TestListByRole_OrderByCreatedAtDesc(t *testing.T) {
	db, repo := setup(t)
	seed(t, db, "排序用户", 5, 0, 1, 0)

	list, _, err := repo.ListByRole(0, 1, 10, "", nil, "", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("数据量不足以验证排序，实际 %d 条", len(list))
	}

	for i := 1; i < len(list); i++ {
		if list[i].CreatedAt.After(list[i-1].CreatedAt) {
			t.Errorf("结果未按 created_at 倒序：第 %d 条(%v) 晚于 第 %d 条(%v)",
				i, list[i].CreatedAt, i-1, list[i-1].CreatedAt)
		}
	}
}

func TestIsExistsEmail(t *testing.T) {
	db, repo := setup(t)

	if _, err := repostest.SeedUser(db, "叶子", "19900000001", "leaf@example.com", 0, 1); err != nil {
		t.Fatalf("造数据失败: %v", err)
	}

	exists, err := repo.IsExistsEmail("leaf@example.com")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if !exists {
		t.Error("leaf@example.com 应判定为已存在")
	}

	exists, err = repo.IsExistsEmail("nobody@example.com")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if exists {
		t.Error("nobody@example.com 应判定为不存在")
	}
}

// UpdateProfile 是部分更新：只改传入的字段，其余必须原样保留。
func TestUpdateProfile_OnlyTouchesGivenFields(t *testing.T) {
	db, repo := setup(t)

	seeded, err := repostest.SeedUser(db, "叶子", "19900000001", "leaf@example.com", 0, 1)
	if err != nil {
		t.Fatalf("造数据失败: %v", err)
	}

	if err := repo.UpdateProfile(seeded.ID, map[string]interface{}{
		"introduction": "把项目真正跑起来",
		"avatar_url":   "https://example.com/avatar.png",
	}); err != nil {
		t.Fatalf("更新资料失败: %v", err)
	}

	got, err := repo.FindByID(seeded.ID)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got.Introduction != "把项目真正跑起来" {
		t.Errorf("简介未更新，实际 %q", got.Introduction)
	}
	if got.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("头像未更新，实际 %q", got.AvatarURL)
	}
	if got.UserName != "叶子" || got.Account != "19900000001" || got.Email != "leaf@example.com" {
		t.Errorf("未传入的字段被意外修改：%+v", got)
	}
}

func TestDelete(t *testing.T) {
	db, repo := setup(t)

	seeded, err := repostest.SeedUser(db, "待删除", "19900000009", "gone@example.com", 0, 1)
	if err != nil {
		t.Fatalf("造数据失败: %v", err)
	}

	if err := repo.Delete(seeded.ID); err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	got, err := repo.FindByID(seeded.ID)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got != nil {
		t.Errorf("删除后不应再查到该用户，实际: %+v", got)
	}
}
