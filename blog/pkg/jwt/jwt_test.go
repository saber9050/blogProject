package jwt

import (
	"blog/pkg/config"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 所有用例共享同一份测试配置，避免 config.Get() panic
func setup(t *testing.T) {
	t.Helper()
	config.MustLoad("testdata/config.yaml")
}

func TestGenerateAndParseToken(t *testing.T) {
	setup(t)

	tokenStr, expireSeconds, err := GenerateToken(1, "alice", 2, "tid-123")
	if err != nil {
		t.Fatalf("GenerateToken 失败: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("生成的 token 为空")
	}
	if expireSeconds != 30*60 {
		t.Errorf("过期秒数 = %d，期望 1800", expireSeconds)
	}

	claims, err := ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken 失败: %v", err)
	}
	if claims.UserID != 1 {
		t.Errorf("UserID = %d，期望 1", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q，期望 alice", claims.Username)
	}
	if claims.UserRoleID != 2 {
		t.Errorf("UserRoleID = %d，期望 2", claims.UserRoleID)
	}
	if claims.TID != "tid-123" {
		t.Errorf("TID = %q，期望 tid-123", claims.TID)
	}
	if claims.Issuer != "blog-project" {
		t.Errorf("Issuer = %q，期望 blog-project", claims.Issuer)
	}
}

func TestParseTokenStrict(t *testing.T) {
	setup(t)

	tokenStr, _, err := GenerateToken(2, "bob", 1, "tid-456")
	if err != nil {
		t.Fatalf("GenerateToken 失败: %v", err)
	}

	claims, err := ParseTokenStrict(tokenStr)
	if err != nil {
		t.Fatalf("合法 token 不应报错: %v", err)
	}
	if claims.Username != "bob" {
		t.Errorf("Username = %q，期望 bob", claims.Username)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	setup(t)

	cases := []struct {
		name  string
		token string
	}{
		{"空字符串", ""},
		{"乱码", "not.a.token"},
		{"结构正确但签名错误", newTokenWithSecret(t, "wrong-secret", time.Hour)},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseToken(tt.token); !errors.Is(err, ErrTokenInvalid) {
				t.Errorf("期望 ErrTokenInvalid，实际: %v", err)
			}
		})
	}
}

func TestParseTokenExpired(t *testing.T) {
	setup(t)

	// 构造一个 1 秒前就已过期的 token
	expired := newTokenWithSecret(t, config.Get().JWT.Secret, -time.Minute)

	// 宽松解析：过期仍返回 claims，用于 refresh 流程
	claims, err := ParseToken(expired)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("期望 ErrTokenExpired，实际: %v", err)
	}
	if claims == nil {
		t.Fatal("过期 token 仍应返回 claims 供刷新使用")
	}
	if claims.Username != "expired-user" {
		t.Errorf("Username = %q，期望 expired-user", claims.Username)
	}

	// 严格解析：过期直接拒绝
	if _, err := ParseTokenStrict(expired); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("ParseTokenStrict 期望 ErrTokenExpired，实际: %v", err)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	setup(t)

	first, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken 失败: %v", err)
	}
	// 32 字节 → 64 位十六进制
	if len(first) != 64 {
		t.Errorf("refresh token 长度 = %d，期望 64", len(first))
	}

	second, _ := GenerateRefreshToken()
	if first == second {
		t.Error("两次生成的 refresh token 相同，随机源可能有问题")
	}
}

func TestGenerateTokenExpire(t *testing.T) {
	setup(t)

	tokenStr, err := GenerateTokenExpire(9, "carol", 3, 60)
	if err != nil {
		t.Fatalf("GenerateTokenExpire 失败: %v", err)
	}

	claims, err := ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken 失败: %v", err)
	}
	if claims.UserID != 9 || claims.UserRoleID != 3 {
		t.Errorf("claims 解析不符: %+v", claims)
	}

	// 过期时间应在 1 分钟附近
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining <= 0 || remaining > 61*time.Second {
		t.Errorf("剩余有效期 %v 不在 0~61 秒区间", remaining)
	}
}

// newTokenWithSecret 用指定密钥和有效期手工签发 token，用于构造异常场景
func newTokenWithSecret(t *testing.T, secret string, ttl time.Duration) string {
	t.Helper()
	claims := CustomClaims{
		UserID:   1,
		Username: "expired-user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "blog-project",
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("签发测试 token 失败: %v", err)
	}
	return signed
}
