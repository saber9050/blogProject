package response

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

// IsExistsResponse 检测是否存在响应
type IsExistsResponse struct {
	IsExists bool `json:"is_exists"` // false 不存在
}
