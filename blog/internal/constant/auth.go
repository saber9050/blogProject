package constant

// 验证码限制
const (
	// CaptchaLength 验证码长度（6位数字）
	CaptchaLength = 6

	// CaptchaExpire 验证码有效时间（秒）
	CaptchaExpire = 300

	// CaptchaSendLimit 验证码发送间隔
	CaptchaSendLimit = 60
)

// 验证码用途
const (
	// CaptchaPurposeLogin 登录用途
	CaptchaPurposeLogin = "login"

	// CaptchaPurposeResetPassword 重置密码用途
	CaptchaPurposeResetPassword = "reset_password"

	//	CaptchaPurposeResetEmail 重置有邮箱用途
	CaptchaPurposeResetEmail = "reset_email"

	// CaptchaPurposeAddEmail 添加邮箱用途
	CaptchaPurposeAddEmail = "add_email"
)
