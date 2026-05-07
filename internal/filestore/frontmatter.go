package filestore

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"goblog/internal/pkg/model"
)

func parseFrontmatter(raw string) (meta map[string]string, content string) {
	meta = make(map[string]string)
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return meta, raw
	}
	rest := raw[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return meta, raw
	}
	header := rest[:idx]
	content = strings.TrimSpace(rest[idx+4:])

	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
			val = strings.ReplaceAll(val, `\"`, `"`)
		}
		meta[key] = val
	}
	return
}

func (r *FileRepository) parsePost(raw string, slug string) *model.Post {
	meta, content := parseFrontmatter(raw)
	post := &model.Post{
		Id:          slug,
		Title:       meta["title"],
		Identity:    slug,
		Description: meta["description"],
		Content:     content,
	}

	post.Status, _ = strconv.Atoi(meta["status"])
	post.CategoryId, _ = strconv.Atoi(meta["category_id"])
	post.IsTop, _ = strconv.Atoi(meta["is_top"])
	post.WordCount, _ = strconv.Atoi(meta["word_count"])

	if t, err := time.Parse(time.RFC3339, meta["created_at"]); err == nil {
		post.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, meta["updated_at"]); err == nil {
		post.UpdatedAt = t
	}

	if tagStr := meta["tag_ids"]; tagStr != "" {
		var tagIds []int
		if err := json.Unmarshal([]byte(tagStr), &tagIds); err == nil {
			post.TagIds = tagIds
		}
	}

	tagIds, _ := json.Marshal(post.TagIds)
	post.TagIdString = string(tagIds)

	return post
}

func postToFrontmatter(post *model.Post) string {
	tagIds, _ := json.Marshal(post.TagIds)
	desc := strings.ReplaceAll(post.Description, "\"", "\\\"")
	title := strings.ReplaceAll(post.Title, "\"", "\\\"")

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: \"%s\"\n", title)
	fmt.Fprintf(&b, "status: %d\n", post.Status)
	fmt.Fprintf(&b, "created_at: %s\n", post.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "updated_at: %s\n", post.UpdatedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "category_id: %d\n", post.CategoryId)
	fmt.Fprintf(&b, "is_top: %d\n", post.IsTop)
	fmt.Fprintf(&b, "tag_ids: %s\n", string(tagIds))
	fmt.Fprintf(&b, "description: \"%s\"\n", desc)
	fmt.Fprintf(&b, "word_count: %d\n", post.WordCount)
	b.WriteString("---\n\n")
	b.WriteString(post.Content)

	return b.String()
}

func pageToFrontmatter(page model.Page) string {
	title := strings.ReplaceAll(page.Title, "\"", "\\\"")
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", page.Id)
	fmt.Fprintf(&b, "title: \"%s\"\n", title)
	b.WriteString("---\n\n")
	b.WriteString(page.Content)
	return b.String()
}
