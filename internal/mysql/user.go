package mysql

import (
	"goblog/internal/pkg/model"
)

// GetUserByEmail 根据邮箱获取用户信息
func (r *GormRepository) GetUserByEmail(email string) (model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return user, err
}

// AddUser 添加新用户
func (r *GormRepository) AddUser(user model.User) (uint, error) {
	result := r.db.Create(&user)
	if result.Error != nil {
		return 0, result.Error
	}
	return user.Id, nil
}

// UpdateUser 更新用户信息
func (r *GormRepository) UpdateUser(user model.User) error {
	return r.db.Model(&user).Updates(map[string]interface{}{
		"password": user.Password,
		// 若后续有其他字段需要更新，可在此添加
	}).Error
}

// DeleteUserByEmail 根据邮箱删除用户
func (r *GormRepository) DeleteUserByEmail(email string) error {
	return r.db.Where("email = ?", email).Delete(&model.User{}).Error
}

// GetAllUsers 获取所有用户信息
func (r *GormRepository) GetAllUsers() ([]model.User, error) {
	var users []model.User
	err := r.db.Find(&users).Error
	return users, err
}
