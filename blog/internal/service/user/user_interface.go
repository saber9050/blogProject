package user

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"mime/multipart"
)

// UserService 用户服务接口
type UserService interface {
	// GetUserInfo 获取用户信息
	GetUserInfo(userID uint) (*response.UserInfoResponse, error)
	// UpdateProfile 编辑用户信息（昵称、个人简介）
	UpdateProfile(userID uint, req *request.UpdateUserProfileRequest) error
	// UpdateAvatar 更换头像
	UpdateAvatar(userID uint, fileHeader *multipart.FileHeader) (*response.UpdateUserAvatarResponse, error)
	// UpdateEmail 更换邮箱确认
	UpdateEmail(userID uint, req *request.UpdateUserEmailRequest) (*response.UpdateUserEmailResponse, error)
	//	UpdateAdminEmail 修改邮箱
	UpdateAdminEmail(id uint, token, newEmail string) error
}
