package filestore

import (
	"fmt"
	"os"
	"path/filepath"

	"goblog/internal/pkg/model"
	"goblog/internal/repository"
)

func (r *FileRepository) GetPages(params repository.PageParams) ([]model.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.Page, len(r.pages))
	copy(result, r.pages)

	if params.PerPage > 0 && params.Page > 0 {
		start := (params.Page - 1) * params.PerPage
		if start >= len(result) {
			return nil, nil
		}
		end := start + params.PerPage
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	return result, nil
}

func (r *FileRepository) GetPage(ident string) (model.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.pages {
		if p.Id == ident {
			return p, nil
		}
	}
	return model.Page{}, fmt.Errorf("page not found: %s", ident)
}

func (r *FileRepository) PageDelete(page model.Page) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.pages {
		if p.Id == page.Id {
			os.Remove(filepath.Join(r.dataDir, "pages", page.Id+".md"))
			r.pages = append(r.pages[:i], r.pages[i+1:]...)
			r.gitCommitAndPush(fmt.Sprintf("delete page: %s", p.Title))
			return page.Id, nil
		}
	}
	return page.Id, fmt.Errorf("page not found: %s", page.Id)
}

func (r *FileRepository) PageSave(page model.Page) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.pages {
		if p.Id == page.Id {
			r.pages[i].Title = page.Title
			r.pages[i].Content = page.Content
			if err := r.writePageFile(r.pages[i]); err != nil {
				return "", err
			}
			r.gitCommitAndPush(fmt.Sprintf("update page: %s", page.Title))
			return page.Id, nil
		}
	}

	r.pages = append(r.pages, page)
	if err := r.writePageFile(page); err != nil {
		return "", err
	}
	r.gitCommitAndPush(fmt.Sprintf("add page: %s", page.Title))
	return page.Id, nil
}

func (r *FileRepository) writePageFile(page model.Page) error {
	dir := filepath.Join(r.dataDir, "pages")
	os.MkdirAll(dir, 0755)
	content := pageToFrontmatter(page)
	return os.WriteFile(filepath.Join(dir, page.Id+".md"), []byte(content), 0644)
}

func (r *FileRepository) PageExist(id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.pages {
		if p.Id == id {
			return true, nil
		}
	}
	return false, nil
}
