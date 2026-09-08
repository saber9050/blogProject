package utils

import (
	"strings"
	"testing"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"验证码长度6", 6},
		{"长度为1", 1},
		{"长度为0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRandomString(tt.length)
			if err != nil {
				t.Fatalf("GenerateRandomString(%d) 返回错误: %v", tt.length, err)
			}
			if len(got) != tt.length {
				t.Errorf("期望长度 %d，实际 %d", tt.length, len(got))
			}
			for _, c := range got {
				if !strings.ContainsRune("0123456789", c) {
					t.Errorf("结果包含非数字字符: %q", c)
				}
			}
		})
	}
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	// 连续生成 100 次，碰撞概率极低，用于发现随机数实现错误
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		code, err := GenerateRandomString(8)
		if err != nil {
			t.Fatalf("生成失败: %v", err)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("出现重复随机串: %s", code)
		}
		seen[code] = struct{}{}
	}
}

func TestContains(t *testing.T) {
	slice := []string{"go", "vue", "docker"}

	if !Contains(slice, "go") {
		t.Error("期望包含 'go'")
	}
	if Contains(slice, "rust") {
		t.Error("期望不包含 'rust'")
	}
	if Contains(nil, "go") {
		t.Error("nil 切片不应包含任何元素")
	}
}

func TestTrimSpaceAndIsEmpty(t *testing.T) {
	tests := []struct {
		input   string
		trimmed string
		isEmpty bool
	}{
		{"  hello  ", "hello", false},
		{"\t\n", "", true},
		{"", "", true},
		{"   ", "", true},
		{"a", "a", false},
	}

	for _, tt := range tests {
		if got := TrimSpace(tt.input); got != tt.trimmed {
			t.Errorf("TrimSpace(%q) = %q，期望 %q", tt.input, got, tt.trimmed)
		}
		if got := IsEmpty(tt.input); got != tt.isEmpty {
			t.Errorf("IsEmpty(%q) = %v，期望 %v", tt.input, got, tt.isEmpty)
		}
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	password := "MyP@ssw0rd"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if hash == password {
		t.Fatal("哈希值不应等于明文")
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		t.Errorf("哈希格式不符合 bcrypt: %s", hash)
	}

	if !CheckPassword(password, hash) {
		t.Error("正确密码应通过校验")
	}
	if CheckPassword("wrong-password", hash) {
		t.Error("错误密码不应通过校验")
	}

	// 同一明文两次哈希结果应不同（盐值随机），但都应校验通过
	hash2, _ := HashPassword(password)
	if hash == hash2 {
		t.Error("两次哈希结果相同，说明未使用随机盐")
	}
	if !CheckPassword(password, hash2) {
		t.Error("第二次哈希应同样校验通过")
	}
}

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"有重复", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"无重复", []string{"x", "y"}, []string{"x", "y"}},
		{"空切片", []string{}, []string{}},
		{"nil 输入", nil, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueStrings(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("长度不符：got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("位置 %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := HashPassword("benchmark-password"); err != nil {
			b.Fatal(err)
		}
	}
}
