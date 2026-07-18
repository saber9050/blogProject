package jwt

import (
	"blog/pkg/config"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid = errors.New("token 无效")
	ErrTokenExpired = errors.New("token 已过期")
)

// GenerateToken 生成 Access Token（短有效期，分钟级）
// tid 存储 refresh token
func GenerateToken(userID uint, username string, userRole uint, tid string) (string, int64, error) {
	cfg := config.Get().JWT
	expireSeconds := int64(cfg.AccessExpireMinutes) * 60

	claims := CustomClaims{
		UserID:     userID,
		Username:   username,
		UserRoleID: userRole,
		TID:        tid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "blog-project",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", 0, err
	}
	return tokenString, expireSeconds, nil
}

// ParseToken 解析 JWT Token（允许过期，用于刷新流程获取claims）
func ParseToken(tokenString string) (*CustomClaims, error) {
	cfg := config.Get().JWT
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			// 过期也返回 claims，用于刷新流程
			if claims, ok := token.Claims.(*CustomClaims); ok {
				return claims, ErrTokenExpired
			}
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// ParseTokenStrict 严格解析 JWT Token（过期直接返回错误）
func ParseTokenStrict(tokenString string) (*CustomClaims, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		// 如果是因为过期而返回的 claims，这里拒绝
		if errors.Is(err, ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}
	return claims, nil
}

// GenerateRefreshToken 生成随机 Refresh Token
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateTokenExpire 生成指定过期时间 Token（单位秒）
// 用于短时间临时token（如邮箱确认链接）
func GenerateTokenExpire(userID uint, username string, userRole uint, expireSecond uint) (string, error) {
	cfg := config.Get().JWT
	claims := CustomClaims{
		UserID:     userID,
		Username:   username,
		UserRoleID: userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireSecond) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "blog-project",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}
