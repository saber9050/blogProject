package user

import (
	auth2 "blog/internal/cache/auth"
	"blog/internal/constant"
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	repo "blog/internal/repository/user"
	"blog/internal/service/auth"
	"blog/pkg/email"
	"blog/pkg/errors"
	"blog/pkg/jwt"
	"blog/pkg/logger"
	minioPkg "blog/pkg/minio"
	"blog/pkg/utils"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// userService 用户服务实现
type userService struct {
	userRepo  repo.UserRepository
	minio     *minioPkg.Client
	authSvc   auth.AuthService
	authCache auth2.AuthCache
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repo.UserRepository, minio *minioPkg.Client, authSvc auth.AuthService, authCache auth2.AuthCache) UserService {
	return &userService{
		userRepo:  userRepo,
		minio:     minio,
		authSvc:   authSvc,
		authCache: authCache,
	}
}

// GetUserInfo 获取用户信息
func (s *userService) GetUserInfo(userID uint) (*response.UserInfoResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}
	if user == nil {
		return nil, errors.ErrUserNotFound
	}

	fmt.Println("------" + s.minio.GetFileURL(user.AvatarURL))
	return &response.UserInfoResponse{
		UserID:       user.ID,
		UserName:     user.UserName,
		Account:      user.Account,
		Email:        user.Email,
		AvatarURL:    s.minio.GetFileURL(user.AvatarURL),
		Introduction: user.Introduction,
		RoleID:       user.RoleID,
		Staus:        user.Status,
		CreateAt:     user.CreatedAt,
	}, nil
}

// UpdateProfile 编辑用户信息（昵称、个人简介）
func (s *userService) UpdateProfile(userID uint, req *request.UpdateUserProfileRequest) error {
	updates := make(map[string]interface{})

	if req.NickName != "" {
		updates["user_name"] = req.NickName
	}
	if req.Introduction != "" {
		updates["introduction"] = req.Introduction
	}

	if len(updates) == 0 {
		return nil
	}

	if err := s.userRepo.UpdateProfile(userID, updates); err != nil {
		return fmt.Errorf("更新用户资料失败: %w", err)
	}

	return nil
}

// UpdateAvatar 更换头像
func (s *userService) UpdateAvatar(userID uint, fileHeader *multipart.FileHeader) (*response.UpdateUserAvatarResponse, error) {
	// 找到用户
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("获取用户信息失败:%s", err)
	}

	// 上传新头像
	fileKey, err := s.UpLoadImage(fileHeader)
	if err != nil {
		return nil, err
	}

	// 更新用户头像 URL
	if err := s.userRepo.UpdateProfile(userID, map[string]interface{}{"avatar_url": fileKey}); err != nil {
		return nil, fmt.Errorf("更新用户头像失败: %w", err)
	}

	// 删除旧头像
	ctx := context.Background()
	err = s.minio.Delete(ctx, user.AvatarURL)
	if err != nil {
		logger.Error("删除旧头像失败", zap.Error(err))
	}

	return &response.UpdateUserAvatarResponse{
		AvatarURL: s.minio.GetFileURL(fileKey),
	}, nil
}

// UpLoadImage 上传图片文件
// 返回 fileKey , error
func (s *userService) UpLoadImage(fileHeader *multipart.FileHeader) (string, error) {
	// 验证文件大小
	if fileHeader.Size > constant.ImageMaxLength {
		str := fmt.Sprintf("头像大小不能超过 %dMB", constant.ImageMaxLength>>20)
		return "", errors.New(errors.CodeBadRequest, str)
	}

	// 打开上传的文件
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer func() {
		_ = src.Close()
	}()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	// 校验文件格式
	if _, ok := constant.AllowedImageExt[ext]; !ok {
		return "", errors.New(errors.CodeBadRequest, "不支持的文件格式")
	}

	// 生成唯一的对象名称
	objectName := fmt.Sprintf("all/%s/%s%s",
		time.Now().Format("20060102"),
		randomName(),
		ext)

	// 上传到 MinIO
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fileKey, err := s.minio.Upload(ctx, objectName, src, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		return "", fmt.Errorf("上传头像失败: %w", err)
	}
	return fileKey, nil
}

// UpdateEmail 更换邮箱确认
func (s *userService) UpdateEmail(userID uint, req *request.UpdateUserEmailRequest) (*response.UpdateUserEmailResponse, error) {
	// 获取用户
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}
	// 验证并删除邮箱验证码
	err = s.authSvc.VerifyCaptcha(user.Email, constant.CaptchaPurposeResetEmail, req.Captcha)
	if err != nil {
		return nil, fmt.Errorf("验证邮箱验证码失败:%s", err)
	}
	// 验证密码
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.New(errors.CodeBadRequest, "密码错误")
	}
	// 验证邮箱唯一性
	ok, err := s.IsExistsEmail(req.NewEmail)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, errors.New(errors.CodeConflict, "该邮箱已经被绑定")
	}
	// 生成token
	token, err := jwt.GenerateToken(user.ID, req.NewEmail, uint(user.RoleID))
	if err != nil {
		return nil, fmt.Errorf("生成Token失败:%s", err)
	}
	// 限制频率
	res, err := s.authCache.CheckEmailSendLimit(req.NewEmail)
	if err != nil {
		return nil, fmt.Errorf("检查邮箱发送频率失败:%s", err)
	}
	if res {
		return nil, errors.New(errors.CodeTooManyRequests, "请勿频繁发送验证码，请60秒后再试")
	}
	err = email.SendVerificationEmail(req.NewEmail, token)
	if err != nil {
		// 删除频率限制
		err2 := s.authCache.DeleteEmailSendLimit(req.NewEmail)
		if err2 != nil {
			return nil, fmt.Errorf("删除新邮箱频率限制失败:%w", err2)
		}
		return nil, fmt.Errorf("给新邮箱发送邮件失败:%s", err)
	}
	return &response.UpdateUserEmailResponse{
		Message: "已发送确认邮件到新邮箱，请到邮箱确认",
	}, nil
}

// AddEmail 添加邮箱接口，仅在无邮箱时能添加成功
func (s *userService) AddEmail(userID uint, req *request.AddEmailRequest) error {
	// 检查有无邮箱
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}
	if user.Email != "" {
		return errors.New(errors.CodeForbidden, "用户已存在邮箱，不可再添加")
	}
	// 检查邮箱验证码
	err = s.authSvc.VerifyCaptcha(req.NewEmail, constant.CaptchaPurposeAddEmail, req.Captcha)
	if err != nil {
		return fmt.Errorf("验证邮箱验证码失败:%s", err)
	}
	// 检查新邮箱是否存在
	ok, err := s.IsExistsEmail(req.NewEmail)
	if err != nil {
		return err
	}
	if ok {
		return errors.New(errors.CodeConflict, "该邮箱已经被绑定")
	}
	// 添加邮箱
	update := make(map[string]interface{})
	update["email"] = req.NewEmail
	err = s.userRepo.UpdateProfile(userID, update)
	if err != nil {
		return fmt.Errorf("添加邮箱失败:%s", err)
	}
	return nil
}

// UpdateAdminEmail 修改邮箱
func (s *userService) UpdateAdminEmail(id uint, token, newEmail string) error {
	// 检查是否在黑名单
	ok, err := s.authCache.CheckBlacklist(token)
	if err != nil {
		return fmt.Errorf("检查token是否在黑名单失败:%s", err)
	}
	if ok {
		return errors.New(errors.CodeForbidden, "该token已无效")
	}
	// 更新邮箱
	update := make(map[string]interface{})
	update["email"] = newEmail
	err = s.userRepo.UpdateProfile(id, update)
	if err != nil {
		return fmt.Errorf("更新邮箱失败:%s", err)
	}
	// token加入黑名单
	err = s.authCache.BlacklistToken(token, 300)
	if err != nil {
		return fmt.Errorf("将token加入黑名单失败:%w", err)
	}
	return nil
}

// IsExistsEmail 检查邮箱是否存在
func (s *userService) IsExistsEmail(rqEmail string) (bool, error) {
	ok, err := s.userRepo.IsExistsEmail(rqEmail)
	if err != nil {
		return false, fmt.Errorf("查询新邮箱是否存在失败:%s", err)
	}
	return ok, nil
}

// randomName 随机名字
func randomName() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%d_%s", time.Now().Unix(), hex.EncodeToString(buf))
}
