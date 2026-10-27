package migrate

import "testing"

// TestMaskDSN 锁住脱敏输出。
//
// 之所以值得单测：第一版用 url.Parse 判断，把 MySQL DSN 当成了 scheme=root 的 URL，
// String() 重建后打印成 `root:****@//tcp/(127.0.0.1:3306)/...`，
// 看起来像连接串被改坏了，实际只是打印难看——但这种"看起来坏了"的日志
// 会让人排查错方向，所以固定下来。
func TestMaskDSN(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "MySQL DSN",
			in:   "root:123456@tcp(127.0.0.1:3306)/blog-project?charset=utf8mb4&parseTime=True&loc=Local",
			want: "root:****@tcp(127.0.0.1:3306)/blog-project?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "PostgreSQL URL",
			in:   "postgres://postgres:123456@127.0.0.1:5432/blog-project?sslmode=disable",
			want: "postgres://postgres:****@127.0.0.1:5432/blog-project?sslmode=disable",
		},
		{
			name: "密码含冒号",
			in:   "root:a:b:c@tcp(127.0.0.1:3306)/db",
			want: "root:****@tcp(127.0.0.1:3306)/db",
		},
		{
			name: "无密码",
			in:   "postgres://postgres@127.0.0.1:5432/db",
			want: "postgres://postgres@127.0.0.1:5432/db",
		},
		{
			name: "不是连接串",
			in:   "hello",
			want: "hello",
		},
	}

	for _, c := range cases {
		if got := maskDSN(c.in); got != c.want {
			t.Errorf("%s: maskDSN() = %q, 期望 %q", c.name, got, c.want)
		}
	}
}

// TestValidIdent 校验库名白名单，防止 DDL 拼接被引号打断。
func TestValidIdent(t *testing.T) {
	ok := []string{"blog", "blog_project", "blog-project", "Blog123", "a"}
	bad := []string{"", "blog;drop", `blog"x`, "blog x", "blog/x", "博客"}

	for _, s := range ok {
		if !validIdent(s) {
			t.Errorf("validIdent(%q) = false, 期望 true", s)
		}
	}
	for _, s := range bad {
		if validIdent(s) {
			t.Errorf("validIdent(%q) = true, 期望 false", s)
		}
	}
}
