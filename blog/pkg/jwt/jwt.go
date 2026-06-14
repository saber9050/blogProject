package jwt

import (
	"blog/pkg/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid = errors.New("token 无效")
	ErrTokenExpired = errors.New("token 已过期")
)

// GenerateToken 生成 Token
func GenerateToken(userID uint, username string, userRole uint) (string, error) {
	cfg := config.Get().JWT

	claims := CustomClaims{
		UserID:     userID,
		Username:   username,
		UserRoleID: userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.ExpireHours) * time.Hour)), // 过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),                                                 // 令牌签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),                                                 // 令牌生效时间
			Issuer:    "core-coach",                                                                   // 签发者
		},
	}
	// 创建jwt 令牌对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// 对令牌签名并返回
	return token.SignedString([]byte(cfg.Secret))
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*CustomClaims, error) {
	cfg := config.Get().JWT
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	// 若令牌有效
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// RefreshToken 刷新 Token
func RefreshToken(tokenString string) (string, error) {
	// 解析原有令牌
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	// 用原有数据生成新令牌
	return GenerateToken(claims.UserID, claims.Username, claims.UserRoleID)
}
