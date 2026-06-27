package request

// RegisterRequest 注册请求
type RegisterRequest struct {
	UserName string `json:"user_name" binding:"required,min=1,max=15"` // 昵称，不能重复，无其他要求
	Account  string `json:"account" binding:"required,len=11"`         // 账号，11位数字
	Password string `json:"password" binding:"required,min=11,max=20"` // 密码，11-20位数字和字母的组合
	Ack      string `json:"ack" binding:"required,min=11,max=20"`      // 密码二次确认
}

// LoginRequest 账号登录请求
type LoginRequest struct {
	Account     string `json:"account" binding:"required"`            // 账号
	Password    string `json:"password" binding:"required"`           // 密码
	CaptchaKey  string `json:"captcha_key" binding:"required"`        // 图形验证码id
	CaptchaCode string `json:"captcha_code" binding:"required,len=6"` // 图形验证码
}

// EmailLoginRequest 邮箱登录请求
type EmailLoginRequest struct {
	Email   string `json:"email" binding:"required,email"`                                    // 邮箱
	Purpose string `json:"purpose" binding:"required,oneof=login reset_password reset_email"` // 目的：登录，重设密码，重设邮箱
	Captcha string `json:"captcha" binding:"required,len=6"`                                  // 验证码
}

// SendEmailCaptchaRequest 发送邮箱验证码请求
type SendEmailCaptchaRequest struct {
	Email   string `json:"email" binding:"required,email"`                                              // 邮箱
	Purpose string `json:"purpose" binding:"required,oneof=login reset_password reset_email add_email"` // 目的：登录，重设密码，重设邮箱
}

// ResetPasswordRequest 重设密码请求
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required"`         // 邮箱
	Captcha     string `json:"captcha" binding:"required,len=6"` // 验证码
	NewPassword string `json:"new_password" binding:"required"`  // 新密码
	ACK         string `json:"ack" binding:"required"`           // 密码二次确认
}
