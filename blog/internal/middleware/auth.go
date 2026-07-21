package middleware

import (
	"blog/internal/cache/auth"
	"blog/pkg/jwt"
	"blog/pkg/response"
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	// ContextUserID 用户 ID 上下文键
	ContextUserID = "user_id"
	// ContextUsername 用户名 上下文键
	ContextUsername = "username"
	// ContextRoleID 角色ID 上下文键
	ContextRoleID = "role_id"
)

var (
	authMw     *authMiddleware
	authMwOnce sync.Once
)

// InitAuth 初始化认证中间件（应用启动时调用一次）
func InitAuth(cache auth.AuthCache) {
	authMwOnce.Do(func() {
		authMw = &authMiddleware{cache: cache}
	})
}

// authMiddleware 认证中间件
type authMiddleware struct {
	cache auth.AuthCache
}

// Auth JWT 认证中间件
func Auth(userRole ...uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := authMw.parseBearerToken(c)
		if err != nil {
			response.Unauthorized(c, err.Error())
			c.Abort()
			return
		}

		if len(userRole) > 0 && userRole[0] != 0 && claims.UserRoleID != userRole[0] {
			response.Unauthorized(c, "令牌类型不匹配")
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.GetUserID())
		c.Set(ContextUsername, claims.GetUsername())
		c.Set(ContextRoleID, claims.UserRoleID)
		c.Next()
	}
}

// OptionalAuth 可选认证中间件
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := authMw.parseBearerToken(c)
		if err == nil {
			c.Set(ContextUserID, claims.GetUserID())
			c.Set(ContextUsername, claims.GetUsername())
			c.Set(ContextRoleID, claims.UserRoleID)
		}
		c.Next()
	}
}

// parseBearerToken 从 Authorization header 解析并验证 JWT token
func (m *authMiddleware) parseBearerToken(c *gin.Context) (*jwt.CustomClaims, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("请提供认证令牌")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("令牌格式错误")
	}

	tokenString := parts[1]
	claims, err := jwt.ParseTokenStrict(tokenString)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get(ContextUserID); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(ContextUsername); exists {
		if name, ok := username.(string); ok {
			return name
		}
	}
	return ""
}

// GetRoleID 从上下文获取角色ID
func GetRoleID(c *gin.Context) int64 {
	if roleID, exists := c.Get(ContextRoleID); exists {
		if id, ok := roleID.(uint); ok {
			return int64(id)
		}
	}
	return 0
}
