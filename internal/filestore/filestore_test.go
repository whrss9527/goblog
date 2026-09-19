package filestore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goblog/internal/pkg/model"
	"goblog/internal/repository"
)

func setupTestRepo(t *testing.T) *FileRepository {
	t.Helper()
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "posts"), 0755)
	os.MkdirAll(filepath.Join(dir, "pages"), 0755)

	r := &FileRepository{
		dataDir:        dir,
		postById:       make(map[string]*model.Post),
		postBySlug:     make(map[string]*model.Post),
		views:          make(map[string]int),
		nextCategoryId: 1,
		nextTagId:      1,
		nextBookId:     1,
	}
	return r
}

func TestPostCRUD(t *testing.T) {
	r := setupTestRepo(t)

	post := model.Post{
		Title:       "First Post",
		Identity:    "first-post",
		Description: "My first post",
		Content:     "Hello world",
		CategoryId:  1,
		TagIds:      []int{1, 2},
		Status:      1,
	}

	id, err := r.PostSave(post)
	require.NoError(t, err)
	assert.Equal(t, "first-post", id)

	got, err := r.GetPost("first-post")
	require.NoError(t, err)
	assert.Equal(t, "First Post", got.Title)
	assert.Equal(t, "Hello world", got.Content)

	gotBySlug, err := r.GetPostByIdentity("first-post")
	require.NoError(t, err)
	assert.Equal(t, "First Post", gotBySlug.Title)

	post.Id = "first-post"
	post.Title = "Updated Post"
	_, err = r.PostSave(post)
	require.NoError(t, err)

	got, err = r.GetPost("first-post")
	require.NoError(t, err)
	assert.Equal(t, "Updated Post", got.Title)

	_, err = r.PostDelete(model.Post{Id: "first-post"})
	require.NoError(t, err)

	_, err = r.GetPost("first-post")
	assert.Error(t, err)
}

func TestGetPosts_Pagination(t *testing.T) {
	r := setupTestRepo(t)

	for i := 0; i < 5; i++ {
		post := model.Post{
			Title:    "Post " + string(rune('A'+i)),
			Identity: "post-" + string(rune('a'+i)),
			Content:  "content",
			Status:   1,
		}
		_, err := r.PostSave(post)
		require.NoError(t, err)
	}

	posts, total, err := r.GetPosts(repository.PostParams{PerPage: 2, Page: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, posts, 2)

	posts, total, err = r.GetPosts(repository.PostParams{PerPage: 2, Page: 3})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, posts, 1)

	posts, _, err = r.GetPosts(repository.PostParams{PerPage: 2, Page: 10})
	require.NoError(t, err)
	assert.Len(t, posts, 0)
}

func TestIncrView(t *testing.T) {
	r := setupTestRepo(t)

	post := model.Post{
		Title:    "View Test",
		Identity: "view-test",
		Content:  "content",
		Status:   1,
	}
	_, err := r.PostSave(post)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		require.NoError(t, r.IncrView("view-test"))
	}

	got, err := r.GetPost("view-test")
	require.NoError(t, err)
	assert.Equal(t, 3, got.Views)
}

func TestCategoryCRUD(t *testing.T) {
	r := setupTestRepo(t)

	id, err := r.CategorySave(model.Category{Name: "Go"})
	require.NoError(t, err)
	assert.Equal(t, 1, id)

	id2, err := r.CategorySave(model.Category{Name: "Rust"})
	require.NoError(t, err)
	assert.Equal(t, 2, id2)

	categories, err := r.GetCategories()
	require.NoError(t, err)
	assert.Len(t, categories, 2)

	cat, err := r.GetCategory(1)
	require.NoError(t, err)
	assert.Equal(t, "Go", cat.Name)

	_, err = r.CategoryDelete(model.Category{Id: 1})
	require.NoError(t, err)

	categories, err = r.GetCategories()
	require.NoError(t, err)
	assert.Len(t, categories, 1)
	assert.Equal(t, "Rust", categories[0].Name)
}

func TestPageCRUD(t *testing.T) {
	r := setupTestRepo(t)

	_, err := r.PageSave(model.Page{Id: "about", Title: "About", Content: "About page"})
	require.NoError(t, err)

	page, err := r.GetPage("about")
	require.NoError(t, err)
	assert.Equal(t, "About", page.Title)

	_, err = r.PageSave(model.Page{Id: "about", Title: "About v2", Content: "Updated"})
	require.NoError(t, err)

	page, err = r.GetPage("about")
	require.NoError(t, err)
	assert.Equal(t, "About v2", page.Title)

	_, err = r.PageDelete(model.Page{Id: "about"})
	require.NoError(t, err)

	_, err = r.GetPage("about")
	assert.Error(t, err)
}

func TestSaveJSON_AtomicWrite(t *testing.T) {
	r := setupTestRepo(t)

	data := map[string]int{"a": 1, "b": 2}
	require.NoError(t, r.saveJSON("test.json", data))

	content, err := os.ReadFile(filepath.Join(r.dataDir, "test.json"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `"a": 1`)
}

func TestFlushViews(t *testing.T) {
	r := setupTestRepo(t)

	r.mu.Lock()
	r.views["post-1"] = 10
	r.views["post-2"] = 20
	r.mu.Unlock()

	require.NoError(t, r.flushViews())

	content, err := os.ReadFile(filepath.Join(r.dataDir, "views.json"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `"post-1": 10`)
	assert.Contains(t, string(content), `"post-2": 20`)
}

func TestIncrLike(t *testing.T) {
	r := setupTestRepo(t)
	_, err := r.PostSave(model.Post{Title: "Liked", Identity: "liked", Content: "c", Status: 1})
	require.NoError(t, err)

	n, err := r.IncrLike("liked")
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	n, err = r.IncrLike("liked")
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	got, err := r.GetPost("liked")
	require.NoError(t, err)
	assert.Equal(t, 2, got.Likes)

	_, err = r.IncrLike("missing")
	assert.Error(t, err)

	// persisted and reloaded
	require.NoError(t, r.flushLikes())
	require.NoError(t, r.loadLikes())
	assert.Equal(t, 2, r.likes["liked"])
}

func TestFlushLikes_NoFileWithoutLikes(t *testing.T) {
	r := setupTestRepo(t)
	require.NoError(t, r.flushLikes())
	_, err := os.Stat(filepath.Join(r.dataDir, "likes.json"))
	assert.True(t, os.IsNotExist(err), "likes.json must not be created for a blog without likes")
}

func TestPostRenameKeepsCounters(t *testing.T) {
	r := setupTestRepo(t)
	_, err := r.PostSave(model.Post{Title: "Old", Identity: "old-slug", Content: "c", Status: 1})
	require.NoError(t, err)
	require.NoError(t, r.IncrView("old-slug"))
	_, err = r.IncrLike("old-slug")
	require.NoError(t, err)

	_, err = r.PostSave(model.Post{Id: "old-slug", Title: "Old", Identity: "new-slug", Content: "c", Status: 1})
	require.NoError(t, err)

	got, err := r.GetPostByIdentity("new-slug")
	require.NoError(t, err)
	assert.Equal(t, 1, got.Views, "views must survive a slug change")
	assert.Equal(t, 1, got.Likes, "likes must survive a slug change")
	_, stale := r.views["old-slug"]
	assert.False(t, stale)
}

func TestPostSaveNeverOverwritesAnotherPost(t *testing.T) {
	r := setupTestRepo(t)
	_, err := r.PostSave(model.Post{Title: "First", Identity: "first", Content: "one", Status: 1})
	require.NoError(t, err)
	_, err = r.PostSave(model.Post{Title: "Second", Identity: "second", Content: "two", Status: 1})
	require.NoError(t, err)

	// a new post with the address of an existing one
	_, err = r.PostSave(model.Post{Title: "Impostor", Identity: "first", Content: "gone?", Status: 1})
	assert.ErrorIs(t, err, ErrSlugTaken)

	// renaming a post onto another post
	_, err = r.PostSave(model.Post{Id: "second", Title: "Second", Identity: "first", Content: "two", Status: 1})
	assert.ErrorIs(t, err, ErrSlugTaken)

	first, err := r.GetPost("first")
	require.NoError(t, err)
	assert.Equal(t, "one", first.Content, "the existing post must be untouched")
	second, err := r.GetPost("second")
	require.NoError(t, err)
	assert.Equal(t, "two", second.Content)
	raw, err := os.ReadFile(filepath.Join(r.dataDir, "posts", "first.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "one")
}

func TestPostSaveRejectsUnsafeSlugs(t *testing.T) {
	r := setupTestRepo(t)
	for _, slug := range []string{"", " ", "../escape", "a/b", `a\b`, ".hidden", "trailing ", "nul\x00byte"} {
		_, err := r.PostSave(model.Post{Title: "T", Identity: slug, Content: "c", Status: 1})
		assert.ErrorIs(t, err, ErrInvalidSlug, "slug %q", slug)
	}
	entries, err := os.ReadDir(filepath.Join(r.dataDir, "posts"))
	require.NoError(t, err)
	assert.Empty(t, entries, "nothing may be written for rejected slugs")
	_, err = os.Stat(filepath.Join(r.dataDir, "escape.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestPostSaveKeepsUnusualSlugOfExistingPost(t *testing.T) {
	r := setupTestRepo(t)
	// loaded from disk, written long before slugs were validated
	legacy := &model.Post{Id: "api-design-openapi&grpc", Identity: "api-design-openapi&grpc", Title: "Old", Content: "v1", Status: 1}
	r.posts = append(r.posts, legacy)
	r.postById[legacy.Id] = legacy
	r.postBySlug[legacy.Identity] = legacy

	_, err := r.PostSave(model.Post{Id: legacy.Id, Identity: legacy.Identity, Title: "Old", Content: "v2", Status: 1})
	require.NoError(t, err)
	got, err := r.GetPost(legacy.Id)
	require.NoError(t, err)
	assert.Equal(t, "v2", got.Content)
}

func TestPostSaveWithUnknownIdStartsANewPost(t *testing.T) {
	r := setupTestRepo(t)
	id, err := r.PostSave(model.Post{Id: "deleted-meanwhile", Identity: "fresh", Title: "T", Content: "c", Status: 1})
	require.NoError(t, err)
	assert.Equal(t, "fresh", id)
	got, err := r.GetPost("fresh")
	require.NoError(t, err)
	assert.False(t, got.CreatedAt.IsZero(), "a new post needs a creation time")
}

func TestPageSaveRejectsUnsafeIds(t *testing.T) {
	r := setupTestRepo(t)
	for _, id := range []string{"", "../x", "a/b", ".env"} {
		_, err := r.PageSave(model.Page{Id: id, Title: "T", Content: "c"})
		assert.ErrorIs(t, err, ErrInvalidSlug, "id %q", id)
	}
	entries, err := os.ReadDir(filepath.Join(r.dataDir, "pages"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTagsAndCategoriesKeepTheirFileFormat(t *testing.T) {
	r := setupTestRepo(t)
	// the format the database migration produced
	require.NoError(t, os.WriteFile(filepath.Join(r.dataDir, "tags.json"), []byte(`[
  {"id": 7, "name": "go", "count": 3, "created_at": "2021-08-15T14:09:06+08:00", "updated_at": "2023-04-04T16:31:19+08:00"}
]`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(r.dataDir, "categories.json"), []byte(`[
  {"id": 1, "name": "技术", "created_at": "2021-08-15T14:09:06+08:00", "updated_at": "2021-09-05T19:17:53+08:00"}
]`), 0o644))
	require.NoError(t, r.loadTags())
	require.NoError(t, r.loadCategories())
	require.Len(t, r.tags, 1)
	assert.Equal(t, "2021-08-15T14:09:06+08:00", r.tags[0].CreatedAt, "timestamps must be read, not dropped")

	_, err := r.AddTag(model.Tag{Name: "mysql"})
	require.NoError(t, err)
	_, err = r.CategorySave(model.Category{Name: "生活", Cur: 5})
	require.NoError(t, err)

	tags, err := os.ReadFile(filepath.Join(r.dataDir, "tags.json"))
	require.NoError(t, err)
	for _, want := range []string{`"id": 7`, `"name": "go"`, `"count": 3`, `"created_at": "2021-08-15T14:09:06+08:00"`, `"updated_at": "2023-04-04T16:31:19+08:00"`, `"name": "mysql"`} {
		assert.Contains(t, string(tags), want)
	}
	assert.NotContains(t, string(tags), `"Id"`)
	assert.NotContains(t, string(tags), `"CreatedAt"`)

	categories, err := os.ReadFile(filepath.Join(r.dataDir, "categories.json"))
	require.NoError(t, err)
	assert.Contains(t, string(categories), `"created_at": "2021-08-15T14:09:06+08:00"`)
	assert.Contains(t, string(categories), `"name": "生活"`)
	assert.NotContains(t, string(categories), "Cur", "the form helper field is not data")

	// files an earlier version already rewrote with Go field names still load
	require.NoError(t, os.WriteFile(filepath.Join(r.dataDir, "tags.json"), []byte(`[{"Id": 9, "Name": "legacy", "Count": 1, "CreatedAt": "", "UpdatedAt": ""}]`), 0o644))
	require.NoError(t, r.loadTags())
	require.Len(t, r.tags, 1)
	assert.Equal(t, 9, r.tags[0].Id)
	assert.Equal(t, "legacy", r.tags[0].Name)
}
