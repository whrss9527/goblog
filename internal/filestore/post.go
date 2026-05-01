package filestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"goblog/internal/pkg/model"
	"goblog/internal/repository"
	"goblog/pkg/utils"
)

func (r *FileRepository) GetPosts(params repository.PostParams) ([]*model.Post, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*model.Post
	for _, post := range r.posts {
		if post.Status != 1 {
			continue
		}
		if params.CategoryId != "" {
			cid := fmt.Sprintf("%d", post.CategoryId)
			if cid != params.CategoryId {
				continue
			}
		}
		if params.TagId != "" {
			if !r.postHasTag(post, params.TagId) {
				continue
			}
		}
		if params.Keyword != "" {
			if !r.postMatchesKeyword(post, params) {
				continue
			}
		}
		filtered = append(filtered, post)
	}

	total := int64(len(filtered))

	if params.PerPage > 0 && params.Page > 0 {
		start := (params.Page - 1) * params.PerPage
		if start >= len(filtered) {
			return nil, total, nil
		}
		end := start + params.PerPage
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[start:end]
	}

	result := make([]*model.Post, len(filtered))
	for i, p := range filtered {
		cp := *p
		result[i] = &cp
	}

	for _, post := range result {
		var tagIds []int
		if err := json.Unmarshal([]byte(post.TagIdString), &tagIds); err == nil {
			post.TagIds = tagIds
		}
	}

	return result, total, nil
}

func (r *FileRepository) postHasTag(post *model.Post, tagId string) bool {
	for _, tid := range post.TagIds {
		if fmt.Sprintf("%d", tid) == tagId {
			return true
		}
	}
	return false
}

func (r *FileRepository) postMatchesKeyword(post *model.Post, params repository.PostParams) bool {
	kw := strings.ToLower(params.Keyword)
	if strings.Contains(strings.ToLower(post.Title), kw) ||
		strings.Contains(strings.ToLower(post.Description), kw) {
		return true
	}

	if ids, ok := params.Ids["ids"]; ok {
		for _, id := range ids {
			if post.Id == id {
				return true
			}
		}
	}
	if ids, ok := params.Ids["category_ids"]; ok {
		cid := fmt.Sprintf("%d", post.CategoryId)
		for _, id := range ids {
			if cid == id {
				return true
			}
		}
	}
	if ids, ok := params.Ids["tag_ids"]; ok {
		for _, tid := range ids {
			if r.postHasTag(post, tid) {
				return true
			}
		}
	}
	return false
}

func (r *FileRepository) GetPostsWithContent() ([]*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*model.Post
	for _, p := range r.posts {
		if p.Status != 1 {
			continue
		}
		cp := *p
		var tagIds []int
		if err := json.Unmarshal([]byte(cp.TagIdString), &tagIds); err == nil {
			cp.TagIds = tagIds
		}
		result = append(result, &cp)
	}
	return result, nil
}

func (r *FileRepository) GetPostsArchive() ([]*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*model.Post
	for _, p := range r.posts {
		if p.Status != 1 {
			continue
		}
		result = append(result, &model.Post{
			Id:        p.Id,
			Title:     p.Title,
			CreatedAt: p.CreatedAt,
			Identity:  p.Identity,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (r *FileRepository) GetPost(id string) (model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if p, ok := r.postById[id]; ok {
		cp := *p
		var tagIds []int
		if err := json.Unmarshal([]byte(cp.TagIdString), &tagIds); err == nil {
			cp.TagIds = tagIds
		}
		return cp, nil
	}
	return model.Post{}, fmt.Errorf("post not found: %s", id)
}

func (r *FileRepository) GetPostByIdentity(identity string) (model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if p, ok := r.postBySlug[identity]; ok {
		cp := *p
		var tagIds []int
		if err := json.Unmarshal([]byte(cp.TagIdString), &tagIds); err == nil {
			cp.TagIds = tagIds
		}
		return cp, nil
	}
	return model.Post{}, fmt.Errorf("post not found: %s", identity)
}

func (r *FileRepository) IncrView(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.views[id]++
	if p, ok := r.postById[id]; ok {
		p.Views = r.views[id]
	}
	return nil
}

func (r *FileRepository) PostSave(post model.Post) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	isNew := post.Id == ""
	if isNew {
		post.Id = post.Identity
		post.CreatedAt = time.Now()
		post.Status = 1
	}
	post.UpdatedAt = time.Now()
	post.WordCount = utils.GetTotalWords(post.Content)
	tagIds, _ := json.Marshal(post.TagIds)
	post.TagIdString = string(tagIds)

	if existing, ok := r.postById[post.Id]; ok {
		oldSlug := existing.Identity
		existing.Title = post.Title
		existing.Description = post.Description
		existing.CategoryId = post.CategoryId
		existing.TagIds = post.TagIds
		existing.TagIdString = post.TagIdString
		existing.WordCount = post.WordCount
		existing.Content = post.Content
		existing.UpdatedAt = post.UpdatedAt

		if oldSlug != post.Identity {
			existing.Id = post.Identity
			existing.Identity = post.Identity
			delete(r.postById, oldSlug)
			delete(r.postBySlug, oldSlug)
			r.postById[post.Identity] = existing
			r.postBySlug[post.Identity] = existing
			os.Remove(filepath.Join(r.dataDir, "posts", oldSlug+".md"))
		}

		if err := r.writePostFile(existing); err != nil {
			return "", err
		}
	} else {
		p := &post
		if v, ok := r.views[p.Id]; ok {
			p.Views = v
		}
		r.posts = append(r.posts, p)
		r.postById[p.Id] = p
		r.postBySlug[p.Identity] = p

		sort.Slice(r.posts, func(i, j int) bool {
			if r.posts[i].IsTop != r.posts[j].IsTop {
				return r.posts[i].IsTop > r.posts[j].IsTop
			}
			return r.posts[i].CreatedAt.After(r.posts[j].CreatedAt)
		})

		if err := r.writePostFile(p); err != nil {
			return "", err
		}
	}

	action := "update"
	if isNew {
		action = "add"
	}
	r.gitCommitAndPush(fmt.Sprintf("%s post: %s", action, post.Title))

	return post.Id, nil
}

func (r *FileRepository) writePostFile(post *model.Post) error {
	dir := filepath.Join(r.dataDir, "posts")
	os.MkdirAll(dir, 0755)
	content := postToFrontmatter(post)
	return os.WriteFile(filepath.Join(dir, post.Identity+".md"), []byte(content), 0644)
}

func (r *FileRepository) PostDelete(post model.Post) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.postById[post.Id]
	if !ok {
		return post.Id, fmt.Errorf("post not found: %s", post.Id)
	}

	os.Remove(filepath.Join(r.dataDir, "posts", existing.Identity+".md"))

	delete(r.postById, post.Id)
	delete(r.postBySlug, existing.Identity)
	for i, p := range r.posts {
		if p.Id == post.Id {
			r.posts = append(r.posts[:i], r.posts[i+1:]...)
			break
		}
	}

	r.gitCommitAndPush(fmt.Sprintf("delete post: %s", existing.Title))

	return post.Id, nil
}

func (r *FileRepository) GetPostCountByTagId(id string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, post := range r.posts {
		if post.Status != 1 {
			continue
		}
		if r.postHasTag(post, id) {
			count++
		}
	}
	return count, nil
}

func (r *FileRepository) GetPostIdsByContent(content string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	kw := strings.ToLower(content)
	var ids []string
	for _, post := range r.posts {
		if strings.Contains(strings.ToLower(post.Content), kw) {
			ids = append(ids, post.Id)
		}
	}
	return ids, nil
}

func (r *FileRepository) PostExist(id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.postById[id]
	return ok, nil
}
