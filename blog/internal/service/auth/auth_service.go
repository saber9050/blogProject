package auth

import (
	auth2 "blog/internal/cache/auth"
	"blog/internal/constant"
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/repository/auth"
	"blog/pkg/email"
	"blog/pkg/errors"
	"blog/pkg/jwt"
	"blog/pkg/utils"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
)

// authService 用户认证服务实现
type authService struct {
	authRepo auth.UserAuthRepository
	cache    auth2.AuthCache
}

// NewAuthService 新建用户认证服务
func NewAuthService(authRepo auth.UserAuthRepository, cache auth2.AuthCache) AuthService {
	return &authService{
		authRepo: authRepo,
		cache:    cache,
	}
}

// Register 注册新用户
func (s *authService) Register(req *request.RegisterRequest) error {
	// 验证格式
	if !s.verifyAccount(req.Account) {
		return errors.New(errors.CodeBadRequest, "账号格式错误")
	}
	if !s.verifyPassword(req.Password) {
		return errors.New(errors.CodeBadRequest, "密码格式错误")
	}
	// 验证码密码是否一致
	if req.Password != req.Ack {
		return errors.New(errors.CodeBadRequest, "两次密码不一致")
	}
	// 验证昵称是否存在
	if ok, err := s.IsExistsName(req.UserName); err != nil {
		return err
	} else if ok {
		return errors.New(errors.CodeConflict, "该昵称已经存在")
	}
	// 验证码账号是否存在
	if ok, err := s.IsExistsAccount(req.Account); err != nil {
		return err
	} else if ok {
		return errors.New(errors.CodeConflict, "该账号已经存在")
	}
	// 创建账号
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败%s", err)
	}
	if err := s.authRepo.CreateUser(req.UserName, req.Account, hash, 0); err != nil {
		return fmt.Errorf("注册失败%s", err)
	}
	return nil
}

// Login 登录
func (s *authService) Login(req *request.LoginRequest) (*response.LoginResponse, error) {
	//	验证图形验证码
	if err := s.verifyImageCaptcha(req.CaptchaKey, req.CaptchaCode); err != nil {
		return nil, err
	}
	//	验证账号
	user, err := s.authRepo.FindUserByAccount(req.Account)
	if err != nil {
		return nil, fmt.Errorf("根据账号查找用户失败:%s", err)
	}
	if user == nil {
		return nil, errors.New(errors.CodeNotFound, "该账号不存在")
	}
	if user.Status == 0 {
		return nil, errors.ErrUserDisabled
	}
	// 验证密码
	if ok := utils.CheckPassword(req.Password, user.PasswordHash); !ok {
		return nil, errors.New(errors.CodeBadRequest, "密码错误")
	}
	// 生成jwt token
	roleID := uint(user.RoleID)
	token, err := jwt.GenerateToken(user.ID, user.UserName, roleID)
	if err != nil {
		return nil, fmt.Errorf("生成 JWT TOKEN 失败:%s", err)
	}
	res := &response.LoginResponse{
		UserName:   user.UserName,
		UserID:     user.ID,
		UserRoleID: roleID,
		Token:      token,
	}
	return res, nil
}

// EmailLogin 邮箱登录
func (s *authService) EmailLogin(req *request.EmailLoginRequest) (*response.LoginResponse, error) {
	// 验证邮箱验证码
	err := s.VerifyCaptcha(req.Email, req.Purpose, req.Captcha)
	if err != nil {
		return nil, err
	}
	// 通过邮箱找用户
	user, err := s.authRepo.FindUserByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("根据邮箱查找用户失败:%s", err)
	}
	if user == nil {
		return nil, errors.New(errors.CodeNotFound, "该邮箱不存在")
	}
	// 生成jwt token
	roleID := uint(user.RoleID)
	token, err := jwt.GenerateToken(user.ID, user.UserName, roleID)
	if err != nil {
		return nil, fmt.Errorf("生成 JWT TOKEN 失败:%s", err)
	}
	res := &response.LoginResponse{
		UserName:   user.UserName,
		UserID:     user.ID,
		UserRoleID: roleID,
		Token:      token,
	}
	return res, nil
}

// SendImageCaptcha 发送图形验证码
func (s *authService) SendImageCaptcha() (*response.ImageCaptchaResponse, error) {
	// 设置图形验证码驱动
	driver := base64Captcha.NewDriverDigit(
		80,  // 高度
		240, // 宽度
		6,   //验证码数字位数
		0.7, // 干扰线强度(相对最大强度的比例)
		5,   // 干扰点数量
	)
	// 创建验证码实例
	ImageCaptcha := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)
	// 生成验证码
	id, base64, answer, err := ImageCaptcha.Generate()
	if err != nil {
		return nil, errors.New(errors.CodeInternalError, "生成图形验证码失败"+err.Error())
	}
	// 写入缓存
	if err := s.cache.StoreCaptcha(id, answer, 300); err != nil {
		return nil, errors.New(errors.CodeInternalError, "缓存图片验证码失败:"+err.Error())
	}
	// 处理Base64 图片前缀，如果没有 data:image/png;base64, 前缀就加上
	base64Image := base64
	if len(base64) > 0 && !strings.HasPrefix(base64Image, "data:image/") {
		base64Image = "data:image/png;base64," + base64
	}
	return &response.ImageCaptchaResponse{
		CaptchaID: id,
		Base64:    base64Image,
	}, nil
}

// SendEmailCaptcha 发送邮箱验证码
func (s *authService) SendEmailCaptcha(req *request.SendEmailCaptchaRequest) error {
	// 创建验证码
	captcha, err := utils.GenerateRandomString(constant.CaptchaLength)
	if err != nil {
		return fmt.Errorf("创建验证码失败:%s", err)
	}
	// 缓存验证码
	err = s.cache.StoreEmailCaptcha(req.Email, captcha, req.Purpose, constant.CaptchaExpire)
	if err != nil {
		return fmt.Errorf("缓存验证码失败:%s", err)
	}
	// 检查频率限制
	ok, err := s.cache.CheckEmailSendLimit(req.Email)
	if err != nil {
		return fmt.Errorf("检查邮箱发送频率失败:%s", err)
	}
	if !ok {
		return errors.New(errors.CodeBadRequest, "请勿频繁发送验证码，请60秒后再试")
	}
	// 发送验证码
	err = email.SendCaptchaEmail(req.Email, captcha, req.Purpose)
	if err != nil {
		// 删除频率限制
		if err := s.cache.DeleteEmailSendLimit(req.Email); err != nil {
			return fmt.Errorf("删除邮箱频率限制失败:%s", err)
		}
		return fmt.Errorf("发送邮箱验证码失败:%s", err)
	}
	return nil
}

// ReSetPassword 重设密码
func (s *authService) ReSetPassword(req *request.ResetPasswordRequest) error {
	// 找到用户
	user, err := s.authRepo.FindUserByEmail(req.Email)
	if err != nil {
		return fmt.Errorf("通过邮箱找到用户失败:%s", err)
	}
	if user == nil {
		return errors.New(errors.CodeNotFound, "没有绑定该邮箱的用户")
	}
	// 核验验证码
	res, err := s.cache.GetEmailCaptcha(req.Email, constant.CaptchaPurposeResetPassword)
	if err != nil {
		return fmt.Errorf("获取邮箱验证码失败:%s", err)
	}
	if res != req.Captcha {
		return errors.New(errors.CodeBadRequest, "验证码错误")
	}
	// 核验密码
	if req.NewPassword == req.ACK {
		return errors.New(errors.CodeBadRequest, "两次密码不同")
	}
	// 更新密码
	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败:%s", err)
	}
	err = s.authRepo.UpdateUserPassword(int(user.ID), hash)
	if err != nil {
		return fmt.Errorf("更新密码失败:%s", err)
	}
	return nil
}

// Logout 登出服务
func (s *authService) Logout(token string) error {
	// 解析token，得到过期时间
	claim, err := jwt.ParseToken(token)
	if err != nil {
		return fmt.Errorf("解析token失败:%s", err)
	}
	now := time.Now()
	// 得到剩余时间
	seconds := claim.ExpiresAt.Sub(now).Seconds()
	// 加入黑名单
	err = s.cache.BlacklistToken(token, int64(seconds))
	if err != nil {
		return fmt.Errorf("将token加入黑名单失败:%s", err)
	}
	return nil
}

// IsExistsName 检测名称是否存在
func (s *authService) IsExistsName(name string) (bool, error) {
	ok, err := s.authRepo.IsExistsByName(name)
	if err != nil {
		return false, fmt.Errorf("检测名称是否存在失败:%s", err)
	}
	return ok, nil
}

// IsExistsAccount 检测账号是否存在
func (s *authService) IsExistsAccount(account string) (bool, error) {
	ok, err := s.authRepo.IsExistsByAccount(account)
	if err != nil {
		return false, fmt.Errorf("检测账号是否存在失败:%s", err)
	}
	return ok, nil
}

// verifyImageCaptcha 验证并删除图形验证码
func (s *authService) verifyImageCaptcha(captchaKey, captchaCode string) error {
	code, err := s.cache.GetCaptcha(captchaKey)
	if err != nil {
		return errors.New(errors.CodeNotFound, "验证码不存在或已经过期")
	}
	if code != captchaCode {
		return errors.New(errors.CodeBadRequest, "验证码错误")
	}
	err = s.cache.DeleteCaptcha(captchaKey)
	if err != nil {
		return errors.New(errors.CodeInternalError, "删除图形验证码失败")
	}
	return nil
}

// VerifyCaptcha 验证并删除邮箱验证码
func (s *authService) VerifyCaptcha(email, purpose, captcha string) error {
	captchaCode, err := s.cache.GetEmailCaptcha(email, purpose)
	if err != nil {
		return errors.New(errors.CodeNotFound, "验证码不存在或已经过期")
	}
	if captchaCode != captcha {
		return errors.New(errors.CodeBadRequest, "验证码错误")
	}
	err = s.cache.DeleteEmailCaptcha(email, purpose)
	if err != nil {
		return errors.New(errors.CodeInternalError, "删除邮箱验证码失败")
	}
	return nil
}

// verifyAccount 验证码账号格式
func (s *authService) verifyAccount(account string) bool {
	// 账号：严格 11 位纯数字
	regAccount := regexp.MustCompile(`^[0-9]{11}$`)
	if !regAccount.MatchString(account) {
		return false
	}
	return true
}

// verifyPassword 验证码密码格式
func (s *authService) verifyPassword(password string) bool {
	// 密码：11-20位，必须同时包含 数字 + 大小写字母
	if len(password) < 11 || len(password) > 20 {
		return false
	}
	// 2. 只允许数字和字母（大小写）
	regPassword := regexp.MustCompile(`^[0-9a-zA-Z]+$`)
	if !regPassword.MatchString(password) {
		return false
	}
	// 3. 必须同时包含数字和字母
	hasDigit := false
	hasLetter := false
	for _, ch := range password {
		if ch >= '0' && ch <= '9' {
			hasDigit = true
		} else if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			hasLetter = true
		}
		if hasDigit && hasLetter {
			return true
		}
	}
	return false
}
