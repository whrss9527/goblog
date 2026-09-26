package front

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goblog/internal/pkg/model"
	"goblog/internal/repository"
)

// postsOnly is a PostRepository that only knows how to list posts.
type postsOnly struct {
	repository.PostRepository
	posts []*model.Post
}

func (p postsOnly) GetPostsWithContent() ([]*model.Post, error) {
	out := make([]*model.Post, len(p.posts))
	for i, post := range p.posts {
		cp := *post
		out[i] = &cp
	}
	return out, nil
}

func TestFeedKeepsTheNewestPosts(t *testing.T) {
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	var posts []*model.Post
	for i := 0; i < 25; i++ {
		created := base.AddDate(0, 0, i)
		posts = append(posts, &model.Post{Identity: fmt.Sprintf("post-%02d", i), Title: fmt.Sprintf("第 %d 篇", i),
			Content: "[另一篇](/posts/post-00)", CreatedAt: created, UpdatedAt: created, Status: 1})
	}
	posts[3].UpdatedAt = base.AddDate(1, 0, 0) // an old post edited later does not decide the feed's date: it is not in it

	h := NewFeedHandler(postsOnly{posts: posts}, "https://blog.example.com", "测试博客")
	h.GenerateFeedXml()
	body, etag, modified := h.doc.get()
	feed := string(body)

	assert.Equal(t, feedEntries, strings.Count(feed, "<entry>"))
	assert.Contains(t, feed, "<id>post-24</id>")
	assert.Contains(t, feed, "<id>post-05</id>")
	assert.NotContains(t, feed, "<id>post-04</id>")
	assert.Contains(t, feed, "https://blog.example.com/posts/post-00", "links inside entries are absolute")
	assert.True(t, modified.Equal(base.AddDate(0, 0, 24)), "the feed changed when its newest entry did, got %s", modified)

	h.GenerateFeedXml()
	_, again, _ := h.doc.get()
	require.NotEmpty(t, etag)
	assert.Equal(t, etag, again, "regenerating unchanged posts keeps the validator (restarts do not look like news)")
}
