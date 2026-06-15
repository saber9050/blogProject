package response

import "time"

// LoginResponse 登录响应
type LoginResponse struct {
	UserName   string `json:"user_name"`    // 用户名
	UserID     uint   `json:"user_id"`      // 用户id
	UserRoleID uint   `json:"user_role_id"` //用户角色id
	Token      string `json:"token"`        // token
}

// ImageCaptchaResponse 图形验证码响应
type ImageCaptchaResponse struct {
	CaptchaID string `json:"captcha_id"`
	Base64    string `json:"base_64"`
}

// RefreshJWTTokenResponse 刷新 JWT TOKEN 响应
type RefreshJWTTokenResponse struct {
	Token string `json:"token"`
}

// IsExistsResponse 检测是否存在响应
type IsExistsResponse struct {
	IsExists bool `json:"is_exists"` // false 不存在
}

// UserInfoResponse 用户信息响应
type UserInfoResponse struct {
	UserID    uint      `json:"user_id"`    // id
	UserName  string    `json:"user_name"`  // 昵称
	Account   string    `json:"account"`    // 账号
	Email     string    `json:"email"`      // 邮箱
	AvatarURL string    `json:"avatar_url"` // 头像
	RoleID    int8      `json:"role_id"`    // 角色ID
	Staus     int8      `json:"staus"`      // 状态
	CreateAt  time.Time `json:"create_at"`  // 创建时间
}
