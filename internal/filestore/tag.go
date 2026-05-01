package filestore

import (
	"fmt"
	"strings"
	"time"

	"goblog/internal/pkg/model"
)

func (r *FileRepository) GetTags() ([]model.Tag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.Tag, len(r.tags))
	copy(result, r.tags)
	return result, nil
}

func (r *FileRepository) GetTagsByIds(ids []int) ([]model.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	idSet := make(map[int]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []model.Tag
	for _, t := range r.tags {
		if idSet[t.Id] {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *FileRepository) GetTagIdsByName(name string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids []string
	kw := strings.ToLower(name)
	for _, t := range r.tags {
		if strings.Contains(strings.ToLower(t.Name), kw) {
			ids = append(ids, fmt.Sprintf("%d", t.Id))
		}
	}
	return ids, nil
}

func (r *FileRepository) AddTag(tag model.Tag) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05")
	tag.Id = r.nextTagId
	r.nextTagId++
	tag.CreatedAt = now
	tag.UpdatedAt = now
	r.tags = append(r.tags, tag)
	if err := r.saveJSON("tags.json", r.tags); err != nil {
		return 0, err
	}
	r.gitCommitAndPush(fmt.Sprintf("add tag: %s", tag.Name))
	return tag.Id, nil
}

func (r *FileRepository) IncrTagCount(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, post := range r.posts {
		if post.Status != 1 {
			continue
		}
		if r.postHasTag(post, id) {
			count++
		}
	}

	for i, t := range r.tags {
		if fmt.Sprintf("%d", t.Id) == id {
			r.tags[i].Count = count
			break
		}
	}
	return r.saveJSON("tags.json", r.tags)
}
