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
