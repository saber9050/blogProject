package user

import "blog/internal/model/entity"

// UserRepository 用户数据访问接口
type UserRepository interface {
	// FindByID 通过 ID 查找用户
	FindByID(id uint) (*entity.User, error)
	// UpdateProfile 更新用户资料
	UpdateProfile(id uint, updates map[string]interface{}) error
}
