package filestore

import (
	"fmt"
	"strings"
	"time"

	"goblog/internal/pkg/model"
)

func (r *FileRepository) GetCategories() ([]model.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.Category, len(r.categories))
	copy(result, r.categories)
	return result, nil
}

func (r *FileRepository) GetCategory(id int) (model.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.categories {
		if c.Id == id {
			return c, nil
		}
	}
	return model.Category{}, fmt.Errorf("category not found: %d", id)
}

func (r *FileRepository) GetCategoryIdsByName(name string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids []string
	kw := strings.ToLower(name)
	for _, c := range r.categories {
		if strings.Contains(strings.ToLower(c.Name), kw) {
			ids = append(ids, fmt.Sprintf("%d", c.Id))
		}
	}
	return ids, nil
}

func (r *FileRepository) CategoryDelete(category model.Category) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, c := range r.categories {
		if c.Id == category.Id {
			r.categories = append(r.categories[:i], r.categories[i+1:]...)
			if err := r.saveJSON("categories.json", r.categories); err != nil {
				return 0, err
			}
			r.gitCommitAndPush(fmt.Sprintf("delete category: %s", c.Name))
			return category.Id, nil
		}
	}
	return 0, fmt.Errorf("category not found: %d", category.Id)
}

func (r *FileRepository) CategorySave(category model.Category) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05")

	for i, c := range r.categories {
		if c.Id == category.Id {
			r.categories[i].Name = category.Name
			r.categories[i].UpdatedAt = now
			if err := r.saveJSON("categories.json", r.categories); err != nil {
				return 0, err
			}
			r.gitCommitAndPush(fmt.Sprintf("update category: %s", category.Name))
			return category.Id, nil
		}
	}

	category.Id = r.nextCategoryId
	r.nextCategoryId++
	category.CreatedAt = now
	category.UpdatedAt = now
	r.categories = append(r.categories, category)
	if err := r.saveJSON("categories.json", r.categories); err != nil {
		return 0, err
	}
	r.gitCommitAndPush(fmt.Sprintf("add category: %s", category.Name))
	return category.Id, nil
}
