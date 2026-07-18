package response

// LoginResponse 登录响应
type LoginResponse struct {
	UserName     string `json:"user_name"`    // 用户名
	UserID       uint   `json:"user_id"`      // 用户id
	UserRoleID   uint   `json:"user_role_id"` // 用户角色id
	AccessToken  string `json:"access_token"` // Access Token (JWT)
	RefreshToken string `json:"-"`            // Refresh Token（仅服务层传递，通过 Cookie 返回）
}

// RefreshTokenResponse 刷新令牌响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"` // 新的 Access Token
	RefreshToken string `json:"-"`            // 新的 Refresh Token（仅服务层传递，通过 Cookie 返回）
}

// ImageCaptchaResponse 图形验证码响应
type ImageCaptchaResponse struct {
	CaptchaID string `json:"captcha_id"`
	Base64    string `json:"base_64"`
}

// IsExistsResponse 检测是否存在响应
type IsExistsResponse struct {
	IsExists bool `json:"is_exists"` // false 不存在
}
