//go:build integration

// 关于页仓储的集成测试。它验证的是「单例记录」这种模式：
// 表里没有记录时返回 nil（而不是报错），有记录时 Save 走的是更新而非插入。
//
// 运行方式：
//
//	TEST_MYSQL_DSN='root:test@tcp(127.0.0.1:3306)/blog_test?charset=utf8mb4&parseTime=True&loc=Local' \
//	  go test -tags=integration -v ./internal/repository/about/...
package about_test

import (
	"blog/internal/model/entity"
	"blog/internal/repository/about"
	"blog/internal/repository/repostest"
	"os"
	"testing"

	"gorm.io/gorm"
)

func setup(t *testing.T) (*gorm.DB, about.AboutRepository) {
	t.Helper()

	// 每个测试包用独立的库（blog_test_about），保证 `go test ./...` 并行执行时互不干扰
	db, err := repostest.Open(os.Getenv(repostest.DSNEnv), "about")
	if err != nil {
		t.Fatalf("初始化测试库失败: %v", err)
	}
	if db == nil {
		t.Skipf("未设置 %s，跳过集成测试", repostest.DSNEnv)
	}
	t.Cleanup(func() { _ = repostest.Close(db) })

	if err := repostest.Cleanup(db, "abouts"); err != nil {
		t.Fatalf("清理 abouts 表失败: %v", err)
	}
	return db, about.NewAboutRepository(db)
}

// 空表时必须返回 (nil, nil)。
// service 层依赖这个约定来判断"是否要新建记录"，一旦这里改成返回 ErrRecordNotFound，
// 关于页在全新部署的库上就会直接 500。
func TestGetAbout_EmptyTableReturnsNil(t *testing.T) {
	_, repo := setup(t)

	got, err := repo.GetAbout()
	if err != nil {
		t.Fatalf("期望无错误，实际: %v", err)
	}
	if got != nil {
		t.Fatalf("空表应返回 nil，实际: %+v", got)
	}
}

func TestCreateAbout_ThenGet(t *testing.T) {
	_, repo := setup(t)

	created := &entity.About{
		TechStack: "Go, Gin, Vue 3",
		MyStory:   "把项目真正跑起来",
		Why:       "留下可验证的痕迹",
		Interest:  "分布式系统, 云原生",
		GitHub:    "https://github.com/saber9050",
		CSDN:      "https://blog.csdn.net/example",
	}
	if err := repo.CreateAbout(created); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("创建后应回填自增主键")
	}

	got, err := repo.GetAbout()
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got == nil {
		t.Fatal("创建后应能查到记录")
	}
	if got.ID != created.ID {
		t.Errorf("ID 不符：期望 %d，实际 %d", created.ID, got.ID)
	}
	if got.TechStack != "Go, Gin, Vue 3" {
		t.Errorf("技术栈不符，实际 %q", got.TechStack)
	}
	if got.MyStory != "把项目真正跑起来" {
		t.Errorf("我的故事不符，实际 %q", got.MyStory)
	}
	if got.CSDN != "https://blog.csdn.net/example" {
		t.Errorf("CSDN 地址不符，实际 %q", got.CSDN)
	}
}

// UpdateAbout 底层是 GORM 的 Save：主键非零时走 UPDATE。
// 这里验证"改了字段能落库"，同时确认记录数没有变成两条——
// Save 误判成 INSERT 的话，单例就变成了双份，关于页会随机取到旧数据。
func TestUpdateAbout_PersistsChangesAndKeepsSingleRow(t *testing.T) {
	db, repo := setup(t)

	created := &entity.About{TechStack: "Go", MyStory: "初版"}
	if err := repo.CreateAbout(created); err != nil {
		t.Fatalf("创建失败: %v", err)
	}

	created.TechStack = "Go, Vue 3, Docker"
	created.MyStory = "改过的故事"
	if err := repo.UpdateAbout(created); err != nil {
		t.Fatalf("更新失败: %v", err)
	}

	got, err := repo.GetAbout()
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got.TechStack != "Go, Vue 3, Docker" {
		t.Errorf("技术栈未更新，实际 %q", got.TechStack)
	}
	if got.MyStory != "改过的故事" {
		t.Errorf("故事未更新，实际 %q", got.MyStory)
	}

	var count int64
	if err := db.Model(&entity.About{}).Count(&count).Error; err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if count != 1 {
		t.Errorf("单例记录数应为 1，实际 %d（Save 可能被误判成了 INSERT）", count)
	}
}
