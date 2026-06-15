package auth

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
)

// AuthService 用户认证服务
type AuthService interface {
	// Register 注册新用户
	Register(req *request.RegisterRequest) error
	// Login 登录
	Login(req *request.LoginRequest) (*response.LoginResponse, error)
	// EmailLogin 邮箱登录
	EmailLogin(req *request.EmailLoginRequest) (*response.LoginResponse, error)
	// SendImageCaptcha 发送图形验证码
	SendImageCaptcha() (*response.ImageCaptchaResponse, error)
	// SendEmailCaptcha 发送邮箱验证码
	SendEmailCaptcha(req *request.SendEmailCaptchaRequest) error
	// RefreshJWTToken 刷新 JWT Token
	RefreshJWTToken(req *request.RefreshJWTTokenRequest) (*response.RefreshJWTTokenResponse, error)
	// ReSetPassword 重设密码
	ReSetPassword(req *request.ResetPasswordRequest) error
	// Logout 登出服务
	Logout(token string) error
	// IsExistsName 检测名称是否存在
	IsExistsName(name string) (bool, error)
	// IsExistsAccount 检测账号是否存在
	IsExistsAccount(account string) (bool, error)
	// VerifyCaptcha 验证并删除邮箱验证码
	VerifyCaptcha(email, purpose, captcha string) error
}
