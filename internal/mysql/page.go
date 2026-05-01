package mysql

import (
	"log/slog"

	"goblog/internal/pkg/model"
	"goblog/internal/repository"
)

func (r *GormRepository) GetPages(params repository.PageParams) ([]model.Page, error) {
	var pages []model.Page
	query := r.db.Model(&model.Page{})

	query = query.Order("id ASC")

	if params.PerPage > 0 && params.Page > 0 {
		offset := (params.Page - 1) * params.PerPage
		query = query.Offset(offset).Limit(params.PerPage)
	}

	err := query.Find(&pages).Error
	return pages, err
}

// GetPage 根据 ident 获取页面
func (r *GormRepository) GetPage(ident string) (model.Page, error) {
	var page model.Page
	err := r.db.Where("ident = ?", ident).First(&page).Error
	return page, err
}

// PageDelete 删除页面
func (r *GormRepository) PageDelete(page model.Page) (string, error) {
	result := r.db.Where("ident = ?", page.Id).Delete(&model.Page{})
	if result.Error != nil {
		slog.Error("page delete failed", "page_id", page.Id, "err", result.Error)
		return page.Id, result.Error
	}
	return page.Id, nil
}

// PageSave 保存页面
func (r *GormRepository) PageSave(page model.Page) (string, error) {
	var exists bool
	err := r.db.Model(&model.Page{}).Where("ident = ?", page.Id).Select("count(*) > 0").Scan(&exists).Error
	if err != nil {
		return page.Id, err
	}

	if exists {
		result := r.db.Model(&model.Page{}).Where("ident = ?", page.Id).Updates(map[string]interface{}{
			"title":   page.Title,
			"content": page.Content,
		})
		if result.Error != nil {
			slog.Error("page update failed", "page_id", page.Id, "err", result.Error)
			return page.Id, result.Error
		}
	} else {
		result := r.db.Create(&page)
		if result.Error != nil {
			slog.Error("page insert failed", "page_id", page.Id, "err", result.Error)
			return page.Id, result.Error
		}
	}
	return page.Id, nil
}

// PageExist 检查页面是否存在
func (r *GormRepository) PageExist(id string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Page{}).Where("ident = ?", id).Count(&count).Error
	return count > 0, err
}
