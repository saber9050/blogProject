package auth

type AuthCache interface {
	// StoreCaptcha 存储图形验证码
	StoreCaptcha(captchaKey, captchaCode string, expireSecond int64) error
	// GetCaptcha 获取图形验证码
	GetCaptcha(captchaKey string) (string, error)
	// DeleteCaptcha 删除图形验证码
	DeleteCaptcha(captchaKey string) error
	// StoreEmailCaptcha 存储邮箱验证码
	StoreEmailCaptcha(email string, captcha string, purpose string, expireSeconds int64) error
	// GetEmailCaptcha 获取邮箱验证码
	GetEmailCaptcha(email string, purpose string) (string, error)
	// DeleteEmailCaptcha 删除邮箱验证码
	DeleteEmailCaptcha(email string, purpose string) error
	// CheckEmailSendLimit 检查邮箱发送频率限制（60秒）
	CheckEmailSendLimit(email string) (bool, error)
	// DeleteEmailSendLimit 删除邮箱发送频率限制
	DeleteEmailSendLimit(email string) error
	// BlacklistToken 将 JWT TOKEN 加入黑名单
	BlacklistToken(token string, expireSecond int64) error
	// CheckBlacklist 检查 JWT TOKEN 是否在黑名单中
	CheckBlacklist(token string) (bool, error)
}
