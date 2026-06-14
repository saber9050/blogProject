package jwt

import "github.com/golang-jwt/jwt/v5"

// CustomClaims 自定义 JWT Claims
type CustomClaims struct {
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	UserRoleID uint   `json:"user_role_id"` // 1:admin 或 0:normal
	jwt.RegisteredClaims
}

// GetUserID 获取用户 ID
func (c *CustomClaims) GetUserID() uint {
	return c.UserID
}

// GetUsername 获取用户名
func (c *CustomClaims) GetUsername() string {
	return c.Username
}

// GetUserRole 获取用户角色id
func (c *CustomClaims) GetUserRole() uint {
	return c.UserRoleID
}
