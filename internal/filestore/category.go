package filestore

import (
	"errors"
	"fmt"
	"strings"

	"goblog/internal/pkg/model"
)

// ErrCategoryNameEmpty is returned when a category would end up without a name.
var ErrCategoryNameEmpty = errors.New("category name is empty")

// CategoryInUseError is returned by CategoryDelete while posts are still filed under the category:
// deleting it would leave them pointing at nothing.
type CategoryInUseError struct {
	Posts int
}

func (e *CategoryInUseError) Error() string {
	return fmt.Sprintf("category still has %d posts", e.Posts)
}

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

	inUse := 0
	for _, post := range r.posts {
		if post.CategoryId == category.Id {
			inUse++
		}
	}
	if inUse > 0 {
		return 0, &CategoryInUseError{Posts: inUse}
	}

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

	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return 0, ErrCategoryNameEmpty
	}
	now := timestamp()

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
