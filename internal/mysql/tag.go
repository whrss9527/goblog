package mysql

import (
	"goblog/internal/pkg/model"
)

// GetTags 获取所有标签，按 count 降序排列
func (r *GormRepository) GetTags() ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Order("count DESC").Find(&tags).Error
	return tags, err
}

// GetTagIdsByName 根据标签名称模糊查询标签 ID
func (r *GormRepository) GetTagsByIds(ids []int) ([]model.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var tags []model.Tag
	err := r.db.Where("id IN ?", ids).Find(&tags).Error
	return tags, err
}

func (r *GormRepository) GetTagIdsByName(name string) ([]string, error) {
	var tagIds []string
	err := r.db.Model(&model.Tag{}).
		Where("name LIKE ?", "%"+name+"%").
		Pluck("id", &tagIds).
		Error
	return tagIds, err
}

// AddTag 添加新标签
func (r *GormRepository) AddTag(tag model.Tag) (int, error) {
	result := r.db.Create(&tag)
	if result.Error != nil {
		return 0, result.Error
	}
	return tag.Id, nil
}

// IncrTagCount 增加指定标签的 count 值
func (r *GormRepository) IncrTagCount(id string) error {
	// 先获取当前文章数量
	var count int64
	err := r.db.Model(&model.Post{}).
		Where("JSON_CONTAINS(tag_ids, ?)", id).
		Count(&count).
		Error
	if err != nil {
		return err
	}

	// 更新标签的 count 值
	return r.db.Model(&model.Tag{}).
		Where("id = ?", id).
		Update("count", count).
		Error
}
