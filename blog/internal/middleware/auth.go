package middleware

import (
	"blog/pkg/jwt"
	"blog/pkg/response"
	"fmt"
	"strings"

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

// parseBearerToken 从 Authorization header 中解析并验证 JWT token
func parseBearerToken(c *gin.Context) (*jwt.CustomClaims, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("请提供认证令牌")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("令牌格式错误")
	}

	claims, err := jwt.ParseToken(parts[1])
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// Auth JWT 认证中间件
func Auth(userRole ...uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := parseBearerToken(c)
		if err != nil {
			response.Unauthorized(c, err.Error())
			c.Abort()
			return
		}

		// 判断是否是管理员
		if len(userRole) > 0 && userRole[0] != 0 && claims.UserRoleID != userRole[0] {
			response.Unauthorized(c, "令牌类型不匹配")
			c.Abort()
			return
		}

		// 注入上下文信息
		c.Set(ContextUserID, claims.GetUserID())
		c.Set(ContextUsername, claims.GetUsername())

		c.Next()
	}
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
		if id, ok := roleID.(int64); ok {
			return id
		}
	}
	return 0
}
