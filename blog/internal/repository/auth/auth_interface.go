package auth

import "blog/internal/model/entity"

// UserAuthRepository 用户认证仓储
type UserAuthRepository interface {
	// FindUserByAccount 通过账号找用户
	FindUserByAccount(account string) (*entity.User, error)
	// FindUserByEmail 通过邮箱找用户
	FindUserByEmail(email string) (*entity.User, error)
	// FindUserByID 通过ID找用户
	FindUserByID(id uint) (*entity.User, error)
	// UpdateUserPassword 修改密码
	UpdateUserPassword(id int, newPassword string) error
	// IsExistsByName 验证该名字是否存在
	IsExistsByName(name string) (bool, error)
	// IsExistsByAccount 验证该账号是否存在
	IsExistsByAccount(account string) (bool, error)
	// CreateUser 创建新用户
	CreateUser(name, account, passwordHash string, roleID int8) error
}
