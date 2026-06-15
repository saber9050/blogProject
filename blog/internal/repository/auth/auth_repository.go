package auth

import (
	"blog/internal/model/entity"
	"errors"

	"gorm.io/gorm"
)

// authRepository 用户认证仓储实现
type authRepository struct {
	db *gorm.DB
}

// NewAuthRepository 新建用户认证仓储
func NewAuthRepository(db *gorm.DB) UserAuthRepository {
	return &authRepository{db: db}
}

// FindUserByAccount 通过账号找用户
func (r *authRepository) FindUserByAccount(account string) (*entity.User, error) {
	var user entity.User
	err := r.db.Model(&entity.User{}).Where("account = ?", account).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindUserByEmail 通过邮箱找用户
func (r *authRepository) FindUserByEmail(email string) (*entity.User, error) {
	var user entity.User
	err := r.db.Model(&entity.User{}).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUserPassword 修改密码
func (r *authRepository) UpdateUserPassword(id int, newPasswordHash string) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).
		Update("password_hash = ?", newPasswordHash).Error
}

// IsExistsByName 验证该名字是否存在
func (r *authRepository) IsExistsByName(name string) (bool, error) {
	var count int64
	err := r.db.Model(&entity.User{}).Where("user_name = ?", name).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsExistsByAccount 验证该账号是否存在
func (r *authRepository) IsExistsByAccount(account string) (bool, error) {
	var count int64
	err := r.db.Model(&entity.User{}).Where("account = ?", account).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateUser 创建新用户
func (r *authRepository) CreateUser(name, account, passwordHash string, roleID int8) error {
	user := entity.User{
		UserName:     name,
		Account:      account,
		PasswordHash: passwordHash,
		RoleID:       roleID,
	}
	return r.db.Create(&user).Error
}
