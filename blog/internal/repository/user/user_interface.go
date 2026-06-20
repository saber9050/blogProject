package user

import "blog/internal/model/entity"

// UserRepository 用户数据访问接口
type UserRepository interface {
	// FindByID 通过 ID 查找用户
	FindByID(id uint) (*entity.User, error)
	// UpdateProfile 更新用户资料
	UpdateProfile(id uint, updates map[string]interface{}) error
	// IsExistsEmail 判断邮箱是否存在
	IsExistsEmail(email string) (bool, error)
	// ListByRole 根据角色获取用户列表
	ListByRole(roleID int8, page, pageSize int) ([]*entity.User, int64, error)
	// Create 创建用户
	Create(name, account, passwordHash string, status int8) error
	// Delete 删除用户（软删除）
	Delete(id uint) error
}
