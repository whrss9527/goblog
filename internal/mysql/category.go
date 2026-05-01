package mysql

import (
	"gorm.io/gorm"

	"goblog/internal/pkg/model"
)

// GetCategories 获取所有分类
func (r *GormRepository) GetCategories() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Find(&categories).Error
	return categories, err
}

// GetCategory 根据 ID 获取单个分类
func (r *GormRepository) GetCategory(id int) (model.Category, error) {
	var category model.Category
	err := r.db.First(&category, id).Error
	return category, err
}

// GetCategoryIdsByName 根据名称模糊查询分类 ID
func (r *GormRepository) GetCategoryIdsByName(name string) ([]string, error) {
	var categoryIds []string
	err := r.db.Model(&model.Category{}).
		Where("name LIKE ?", "%"+name+"%").
		Pluck("id", &categoryIds).
		Error
	return categoryIds, err
}

// CategoryDelete 删除分类
func (r *GormRepository) CategoryDelete(category model.Category) (int, error) {
	result := r.db.Delete(&category)
	if result.Error != nil {
		return 0, result.Error
	}
	return category.Id, nil
}

// CategorySave 保存分类，如果分类 ID 存在则更新，否则创建新分类
func (r *GormRepository) CategorySave(category model.Category) (int, error) {
	var existingCategory model.Category
	err := r.db.First(&existingCategory, category.Id).Error

	if err == gorm.ErrRecordNotFound {
		// 分类不存在，创建新分类
		result := r.db.Create(&category)
		if result.Error != nil {
			return 0, result.Error
		}
		return category.Id, nil
	} else if err != nil {
		return 0, err
	}

	// 分类存在，更新分类信息
	result := r.db.Model(&existingCategory).Updates(map[string]interface{}{
		"name": category.Name,
	})
	if result.Error != nil {
		return 0, result.Error
	}
	return category.Id, nil
}
