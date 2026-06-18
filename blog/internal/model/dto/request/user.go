package request

// UpdateUserProfileRequest 编辑用户信息请求
type UpdateUserProfileRequest struct {
	NickName     string `json:"nick_name"`    // 昵称
	Introduction string `json:"introduction"` // 个人简介
}

// UpdateUserEmailRequest 修改邮箱请求
type UpdateUserEmailRequest struct {
	Password string `json:"password" binding:"required"`        // 密码
	Captcha  string `json:"captcha" binding:"required,len=6"`   // 邮箱验证码
	NewEmail string `json:"new_email" binding:"required,email"` // 新邮箱
}

// AddEmailRequest 添加邮箱请求
type AddEmailRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"` // 新邮箱
	Captcha  string `json:"captcha" binding:"required,len=6"`   // 邮箱验证码
}
