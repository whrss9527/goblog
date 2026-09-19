package filestore

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goblog/internal/pkg/model"
	"goblog/internal/repository"
)

func TestDraftLifecycle(t *testing.T) {
	r := setupTestRepo(t)

	drafts, err := r.GetDrafts()
	require.NoError(t, err)
	assert.Empty(t, drafts, "no drafts directory yet is not an error")

	require.NoError(t, r.SaveDraft(model.Post{Title: "想法", Identity: "idea", Content: "第一版", CategoryId: 2, TagIds: []int{1}}, ""))
	got, err := r.GetDraft("idea")
	require.NoError(t, err)
	assert.Equal(t, "想法", got.Title)
	assert.Equal(t, "第一版", got.Content)
	assert.Equal(t, 0, got.Status, "a draft is never published")
	assert.Equal(t, []int{1}, got.TagIds)
	created := got.CreatedAt

	// drafts are invisible to everything that lists posts
	posts, total, err := r.GetPosts(repositoryAll())
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, posts)
	_, err = r.GetPostByIdentity("idea")
	assert.Error(t, err)

	// saving again under a new slug renames the draft and keeps its creation time
	time.Sleep(1100 * time.Millisecond) // timestamps are stored with second precision
	require.NoError(t, r.SaveDraft(model.Post{Title: "想法", Identity: "better-idea", Content: "第二版"}, "idea"))
	_, err = r.GetDraft("idea")
	assert.ErrorIs(t, err, ErrDraftNotFound)
	got, err = r.GetDraft("better-idea")
	require.NoError(t, err)
	assert.Equal(t, "第二版", got.Content)
	assert.True(t, got.CreatedAt.Equal(created), "creation time survives: %v vs %v", got.CreatedAt, created)
	assert.True(t, got.UpdatedAt.After(created))

	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, r.SaveDraft(model.Post{Title: "另一个", Identity: "second", Content: "x"}, ""))
	drafts, err = r.GetDrafts()
	require.NoError(t, err)
	require.Len(t, drafts, 2)
	assert.Equal(t, "second", drafts[0].Identity, "most recently saved first")

	require.NoError(t, r.DeleteDraft("second"))
	require.NoError(t, r.DeleteDraft("second"), "deleting twice is fine")
	drafts, _ = r.GetDrafts()
	assert.Len(t, drafts, 1)
}

func TestDraftKeepsTagNamesWithoutCreatingTags(t *testing.T) {
	r := setupTestRepo(t)
	require.NoError(t, r.SaveDraft(model.Post{Title: "T", Identity: "tagged", Content: "c\n\n---\n\nmore", TagIds: []int{3}, TagNames: []string{"go", "全新, 标签"}}, ""))

	got, err := r.GetDraft("tagged")
	require.NoError(t, err)
	assert.Equal(t, []string{"go", "全新  标签"}, got.TagNames, "names survive, a comma inside a name cannot split it")
	assert.Equal(t, []int{3}, got.TagIds)
	assert.Equal(t, "c\n\n---\n\nmore", got.Content, "a rule inside the text is not mistaken for the frontmatter end")
	assert.Empty(t, r.tags, "saving a draft must not create tags")
	_, err = os.Stat(filepath.Join(r.dataDir, "tags.json"))
	assert.True(t, os.IsNotExist(err))
}

func TestDraftSlugRules(t *testing.T) {
	r := setupTestRepo(t)
	_, err := r.PostSave(model.Post{Title: "Published", Identity: "published", Content: "c", Status: 1})
	require.NoError(t, err)
	require.NoError(t, r.SaveDraft(model.Post{Title: "A", Identity: "draft-a", Content: "a"}, ""))

	assert.ErrorIs(t, r.SaveDraft(model.Post{Title: "X", Identity: "published", Content: "x"}, ""), ErrSlugTaken, "address of a published post")
	assert.ErrorIs(t, r.SaveDraft(model.Post{Title: "X", Identity: "draft-a", Content: "x"}, ""), ErrSlugTaken, "address of another draft")
	assert.NoError(t, r.SaveDraft(model.Post{Title: "A2", Identity: "draft-a", Content: "a2"}, "draft-a"), "saving a draft over itself")
	for _, slug := range []string{"", "../posts/published", "a/b", ".hidden"} {
		assert.ErrorIs(t, r.SaveDraft(model.Post{Title: "X", Identity: slug, Content: "x"}, ""), ErrInvalidSlug, "slug %q", slug)
		_, err := r.GetDraft(slug)
		assert.ErrorIs(t, err, ErrDraftNotFound)
	}
	raw, err := os.ReadFile(filepath.Join(r.dataDir, "posts", "published.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "status: 1")
}

func TestDraftsNeverReachGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	r := setupTestRepo(t)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", r.dataDir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}
	git("init", "-q")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "test")

	// published while git is still "off": PostSave would otherwise commit and
	// try to push in the background, and this repository has no remote
	_, err := r.PostSave(model.Post{Title: "Public", Identity: "public", Content: "c", Status: 1})
	require.NoError(t, err)

	r.gitEnabled = true
	require.NoError(t, r.SaveDraft(model.Post{Title: "Secret", Identity: "secret-plan", Content: "not yet"}, ""))
	require.NoError(t, r.SaveDraft(model.Post{Title: "Secret", Identity: "secret-plan", Content: "still not"}, "secret-plan"))
	r.gitEnabled = false

	git("add", "-A") // what gitCommitAndPush runs
	status := git("status", "--porcelain", "--ignored=no")
	assert.NotContains(t, status, ".drafts", "drafts must be invisible to git add -A")
	tracked := git("ls-files")
	assert.NotContains(t, tracked, "secret-plan")
	assert.Contains(t, tracked, "posts/public.md", "published posts are still committed")

	exclude, err := os.ReadFile(filepath.Join(r.dataDir, ".git", "info", "exclude"))
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(exclude), ".drafts/"), "the exclude rule is written once")
	_, err = os.Stat(filepath.Join(r.dataDir, ".gitignore"))
	assert.True(t, os.IsNotExist(err), "the repository content itself stays untouched")
}

func repositoryAll() repository.PostParams { return repository.PostParams{Page: 1} }
