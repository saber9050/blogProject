package auth

import (
	"blog/internal/model/dto/request"
	response2 "blog/internal/model/dto/response"
	"blog/internal/service/auth"
	"blog/pkg/response"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	authService auth.AuthService
}

func NewController(authService auth.AuthService) *Controller {
	return &Controller{
		authService: authService,
	}
}

// Register 注册
func (c *Controller) Register(ctx *gin.Context) {
	var req request.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}
	if err := c.authService.Register(&req); err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{
		"message": "注册成功",
	})
}

// Login 账号密码登录
func (c *Controller) Login(ctx *gin.Context) {
	var req request.LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}

	authResponse, err := c.authService.Login(&req)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, authResponse)
}

// EmailLogin 邮箱登录
func (c *Controller) EmailLogin(ctx *gin.Context) {
	var req request.EmailLoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}

	authResponse, err := c.authService.EmailLogin(&req)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, authResponse)
}

// SendImageCaptcha 发送图形验证码
func (c *Controller) SendImageCaptcha(ctx *gin.Context) {

	authResponse, err := c.authService.SendImageCaptcha()
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, authResponse)
}

// SendEmailCaptcha 发送邮箱验证码
func (c *Controller) SendEmailCaptcha(ctx *gin.Context) {
	var req request.SendEmailCaptchaRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}

	err := c.authService.SendEmailCaptcha(&req)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{
		"message": "已向你的邮箱发送验证码",
	})
}

// ResetPassword 修改密码接口
func (c *Controller) ResetPassword(ctx *gin.Context) {
	var req request.ResetPasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}

	err := c.authService.ReSetPassword(&req)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{
		"message": "成功修改密码",
	})
}

// Logout 登出
func (c *Controller) Logout(ctx *gin.Context) {
	token := ctx.GetHeader("Authorization")

	err := c.authService.Logout(token)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, nil)
}

// IsExistName 查验名称是否存在
func (c *Controller) IsExistName(ctx *gin.Context) {
	name := ctx.Query("user_name")
	ok, err := c.authService.IsExistsName(name)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, &response2.IsExistsResponse{IsExists: ok})
}

// IsExistAccount 查验账号是否存在
func (c *Controller) IsExistAccount(ctx *gin.Context) {
	account := ctx.Query("account")
	ok, err := c.authService.IsExistsAccount(account)
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, &response2.IsExistsResponse{IsExists: ok})
}
