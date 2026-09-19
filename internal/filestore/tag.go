package filestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"goblog/internal/pkg/model"
)

var (
	// ErrTagNotFound is returned when the tag to rename or delete does not exist (any more).
	ErrTagNotFound = errors.New("tag does not exist")
	// ErrTagNameEmpty is returned when a tag would end up without a name.
	ErrTagNameEmpty = errors.New("tag name is empty")
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

	now := timestamp()
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

func (r *FileRepository) IncrTagCount(_ string) error {
	return r.RecalcTagCounts()
}

func (r *FileRepository) RecalcTagCounts() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.recountTags()
	return r.saveJSON("tags.json", r.tags)
}

// recountTags refreshes Tag.Count from the published posts. The caller holds r.mu.
func (r *FileRepository) recountTags() {
	counts := make(map[int]int)
	for _, post := range r.posts {
		if post.Status != 1 {
			continue
		}
		for _, tid := range post.TagIds {
			counts[tid]++
		}
	}
	for i := range r.tags {
		r.tags[i].Count = counts[r.tags[i].Id]
	}
}

// RenameTag gives a tag a new name. When another tag already carries that name the two are merged: every
// post using the renamed tag moves over to the tag that owns the name, and the renamed tag disappears.
// That is how duplicates such as " blog" / "blog" or "Go" / "go" get cleaned up. The returned id is the
// tag the posts are filed under afterwards.
func (r *FileRepository) RenameTag(id int, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, ErrTagNameEmpty
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	index := r.tagIndex(id)
	if index < 0 {
		return 0, ErrTagNotFound
	}
	oldName := r.tags[index].Name

	target := -1
	for i, tag := range r.tags {
		if i != index && strings.TrimSpace(tag.Name) == name {
			target = i
			break
		}
	}

	if target < 0 {
		if oldName == name {
			return id, nil
		}
		r.tags[index].Name = name
		r.tags[index].UpdatedAt = timestamp()
		if err := r.saveJSON("tags.json", r.tags); err != nil {
			return 0, err
		}
		r.gitCommitAndPush(fmt.Sprintf("rename tag: %s -> %s", strings.TrimSpace(oldName), name))
		return id, nil
	}

	targetId := r.tags[target].Id
	if err := r.retagPosts(id, targetId); err != nil {
		return 0, err
	}
	r.tags[target].Name = name // the name that was asked for, without any padding the old entry had
	r.tags[target].UpdatedAt = timestamp()
	r.tags = append(r.tags[:index:index], r.tags[index+1:]...)
	r.recountTags()
	if err := r.saveJSON("tags.json", r.tags); err != nil {
		return 0, err
	}
	r.gitCommitAndPush(fmt.Sprintf("merge tag: %s -> %s", strings.TrimSpace(oldName), name))
	return targetId, nil
}

// DeleteTag removes a tag and takes it off every post that used it.
func (r *FileRepository) DeleteTag(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := r.tagIndex(id)
	if index < 0 {
		return ErrTagNotFound
	}
	name := r.tags[index].Name
	if err := r.retagPosts(id, 0); err != nil {
		return err
	}
	r.tags = append(r.tags[:index:index], r.tags[index+1:]...)
	r.recountTags()
	if err := r.saveJSON("tags.json", r.tags); err != nil {
		return err
	}
	r.gitCommitAndPush(fmt.Sprintf("delete tag: %s", strings.TrimSpace(name)))
	return nil
}

// timestamp is how created_at / updated_at of tags and categories are stored: RFC 3339 like every
// other date in the content repository (the migrated entries have it; "2006-01-02 15:04:05" without a
// zone is what this code wrote before, nothing reads either).
func timestamp() string {
	return time.Now().Format(time.RFC3339)
}

func (r *FileRepository) tagIndex(id int) int {
	for i, tag := range r.tags {
		if tag.Id == id {
			return i
		}
	}
	return -1
}

// retagPosts replaces tag `from` by tag `to` on every post (to == 0: just drops it). Files are written
// before anything changes in memory, so a failed write leaves the repository as it was; updated_at is left
// alone because the article itself did not change. The caller holds r.mu.
func (r *FileRepository) retagPosts(from, to int) error {
	type change struct {
		post   *model.Post
		tagIds []int
	}
	var changes []change
	for _, post := range r.posts {
		uses := false
		for _, tagId := range post.TagIds {
			if tagId == from {
				uses = true
				break
			}
		}
		if !uses {
			continue
		}
		tagIds := make([]int, 0, len(post.TagIds))
		seen := make(map[int]bool, len(post.TagIds))
		for _, tagId := range post.TagIds {
			if tagId == from {
				tagId = to
			}
			if tagId == 0 || seen[tagId] {
				continue
			}
			seen[tagId] = true
			tagIds = append(tagIds, tagId)
		}
		changes = append(changes, change{post: post, tagIds: tagIds})
	}

	for _, c := range changes {
		if err := r.setTagIdsInFile(c.post, c.tagIds); err != nil {
			return fmt.Errorf("retag %s: %w", c.post.Identity, err)
		}
	}
	for _, c := range changes {
		c.post.TagIds = c.tagIds
		c.post.TagIdString = tagIdString(c.tagIds)
	}
	return nil
}

// setTagIdsInFile changes the tag_ids line of a post file and nothing else: the commit in the content
// repository then shows the retagging, one line per post. Writing the post the regular way would also
// normalise whatever whitespace the author left around the text.
func (r *FileRepository) setTagIdsInFile(post *model.Post, tagIds []int) error {
	path := filepath.Join(r.dataDir, "posts", post.Identity+".md")
	if raw, err := os.ReadFile(path); err == nil {
		if updated, ok := replaceTagIdsLine(string(raw), tagIds); ok {
			return os.WriteFile(path, []byte(updated), 0644)
		}
	}
	// no such line, an unusual header, or the file is gone: write the whole post
	updated := *post
	updated.TagIds = tagIds
	return r.writePostFile(&updated)
}

var tagIdsLine = regexp.MustCompile(`(?m)^tag_ids:[^\r\n]*`)

// replaceTagIdsLine swaps the tag_ids line of a post file. The result is checked with the real parser:
// the ids must be the new ones and nothing else may have changed (a "tag_ids:" line inside a quoted
// multi-line description would otherwise be mistaken for the real one).
func replaceTagIdsLine(raw string, tagIds []int) (string, bool) {
	loc := tagIdsLine.FindStringIndex(raw)
	if loc == nil {
		return "", false
	}
	want := tagIdString(tagIds)
	updated := raw[:loc[0]] + "tag_ids: " + want + raw[loc[1]:]

	before, bodyBefore := parseFrontmatter(raw)
	after, bodyAfter := parseFrontmatter(updated)
	if bodyBefore != bodyAfter || len(before) != len(after) || after["tag_ids"] != want {
		return "", false
	}
	for key, value := range before {
		if key != "tag_ids" && after[key] != value {
			return "", false
		}
	}
	return updated, true
}
