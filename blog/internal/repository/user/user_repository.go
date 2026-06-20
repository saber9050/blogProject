package user

import (
	"blog/internal/model/entity"
	"errors"

	"gorm.io/gorm"
)

// userRepository 用户数据访问实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// FindByID 通过 ID 查找用户
func (r *userRepository) FindByID(id uint) (*entity.User, error) {
	var user entity.User
	err := r.db.First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateProfile 更新用户资料
func (r *userRepository) UpdateProfile(id uint, updates map[string]interface{}) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).Updates(updates).Error
}

// IsExistsEmail 判断邮箱是否存在
func (r *userRepository) IsExistsEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&entity.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByRole 根据角色获取用户列表
func (r *userRepository) ListByRole(roleID int8, page, pageSize int) ([]*entity.User, int64, error) {
	var list []*entity.User
	var total int64

	query := r.db.Model(&entity.User{}).Where("role_id = ?", roleID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Create 创建用户
func (r *userRepository) Create(name, account, passwordHash string, status int8) error {
	user := entity.User{
		UserName:     name,
		Account:      account,
		PasswordHash: passwordHash,
		Status:       status,
		RoleID:       0,
	}
	return r.db.Create(&user).Error
}

// Delete 删除用户（软删除）
func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&entity.User{}, id).Error
}
