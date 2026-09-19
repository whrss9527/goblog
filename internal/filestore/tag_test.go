package filestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goblog/internal/pkg/model"
)

// tagRepo has the tags "blog"(1), " blog"(2, the padded duplicate older versions could create), "go"(3)
// and three posts: a → [2, 3], b → [1, 2], c (hidden) → [2].
func tagRepo(t *testing.T) *FileRepository {
	t.Helper()
	r := setupTestRepo(t)
	for _, name := range []string{"blog", " blog", "go"} {
		_, err := r.AddTag(model.Tag{Name: name})
		require.NoError(t, err)
	}
	for slug, tagIds := range map[string][]int{"a": {2, 3}, "b": {1, 2}, "c": {2}} {
		_, err := r.PostSave(model.Post{Title: slug, Identity: slug, Content: "正文 " + slug, TagIds: tagIds})
		require.NoError(t, err)
	}
	r.mu.Lock()
	r.postById["c"].Status = 0
	require.NoError(t, r.writePostFile(r.postById["c"]))
	r.mu.Unlock()
	require.NoError(t, r.RecalcTagCounts())
	return r
}

func tagNames(r *FileRepository) map[int]string {
	names := make(map[int]string)
	tags, _ := r.GetTags()
	for _, tag := range tags {
		names[tag.Id] = tag.Name
	}
	return names
}

func reload(t *testing.T, r *FileRepository) *FileRepository {
	t.Helper()
	fresh, err := NewFileRepository(r.dataDir, "", "")
	require.NoError(t, err)
	t.Cleanup(fresh.Close)
	return fresh
}

func TestRenameTag(t *testing.T) {
	r := tagRepo(t)

	id, err := r.RenameTag(3, "  golang ")
	require.NoError(t, err)
	assert.Equal(t, 3, id)
	assert.Equal(t, "golang", tagNames(r)[3], "the new name is stored trimmed")

	// nothing but tags.json changes: posts refer to tags by id
	post, _ := r.GetPost("a")
	assert.Equal(t, []int{2, 3}, post.TagIds)

	fresh := reload(t, r)
	assert.Equal(t, "golang", tagNames(fresh)[3])

	tags, _ := fresh.GetTags()
	for _, tag := range tags {
		for _, stamp := range []string{tag.CreatedAt, tag.UpdatedAt} {
			_, err := time.Parse(time.RFC3339, stamp)
			assert.NoError(t, err, "dates in tags.json are RFC 3339, like the migrated entries")
		}
	}
}

func TestRenameTagOntoExistingNameMerges(t *testing.T) {
	r := tagRepo(t)
	before, _ := r.GetPost("a")

	id, err := r.RenameTag(2, "blog")
	require.NoError(t, err)
	assert.Equal(t, 1, id, "the tag that already owned the name survives")
	assert.Equal(t, map[int]string{1: "blog", 3: "go"}, tagNames(r))

	a, _ := r.GetPost("a")
	b, _ := r.GetPost("b")
	c, _ := r.GetPost("c")
	assert.Equal(t, []int{1, 3}, a.TagIds, "position of the tag is kept")
	assert.Equal(t, []int{1}, b.TagIds, "a post that had both tags keeps one")
	assert.Equal(t, []int{1}, c.TagIds, "hidden posts are retagged too")
	assert.True(t, a.UpdatedAt.Equal(before.UpdatedAt), "retagging is not an edit of the article")

	tags, _ := r.GetTags()
	for _, tag := range tags {
		if tag.Id == 1 {
			assert.Equal(t, 2, tag.Count, "counts cover published posts only: a and b")
		}
	}

	// and all of it is on disk
	fresh := reload(t, r)
	assert.Equal(t, map[int]string{1: "blog", 3: "go"}, tagNames(fresh))
	a, _ = fresh.GetPost("a")
	assert.Equal(t, []int{1, 3}, a.TagIds)
	assert.Equal(t, "正文 a", a.Content)
	raw, err := os.ReadFile(filepath.Join(r.dataDir, "posts", "a.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "tag_ids: [1, 3]\n")
}

func TestRenamePaddedTagWithoutTwin(t *testing.T) {
	r := tagRepo(t)
	require.NoError(t, r.DeleteTag(1)) // only " blog" is left

	id, err := r.RenameTag(2, "blog")
	require.NoError(t, err)
	assert.Equal(t, 2, id)
	assert.Equal(t, "blog", tagNames(r)[2])
}

func TestRenameTagErrors(t *testing.T) {
	r := tagRepo(t)

	_, err := r.RenameTag(3, "   ")
	assert.ErrorIs(t, err, ErrTagNameEmpty)
	_, err = r.RenameTag(99, "x")
	assert.ErrorIs(t, err, ErrTagNotFound)

	id, err := r.RenameTag(3, "go")
	require.NoError(t, err, "saving the form without a change is fine")
	assert.Equal(t, 3, id)
	assert.Len(t, tagNames(r), 3)
}

func TestDeleteTag(t *testing.T) {
	r := tagRepo(t)

	require.NoError(t, r.DeleteTag(2))
	assert.Equal(t, map[int]string{1: "blog", 3: "go"}, tagNames(r))
	a, _ := r.GetPost("a")
	b, _ := r.GetPost("b")
	c, _ := r.GetPost("c")
	assert.Equal(t, []int{3}, a.TagIds)
	assert.Equal(t, []int{1}, b.TagIds)
	assert.Empty(t, c.TagIds)

	assert.ErrorIs(t, r.DeleteTag(2), ErrTagNotFound)

	fresh := reload(t, r)
	c, _ = fresh.GetPost("c")
	assert.Empty(t, c.TagIds)
	raw, _ := os.ReadFile(filepath.Join(r.dataDir, "posts", "c.md"))
	assert.Contains(t, string(raw), "tag_ids: []\n", "never \"null\"")

	// ids are not handed out again: an old /?tag_id=2 link must not show an unrelated tag later
	id, err := fresh.AddTag(model.Tag{Name: "new"})
	require.NoError(t, err)
	assert.Equal(t, 4, id)
}

// A failed write must not leave memory and disk telling different stories.
func TestRetagLeavesMemoryAloneWhenWritingFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	r := tagRepo(t)
	posts := filepath.Join(r.dataDir, "posts")
	for _, name := range []string{"a.md", "b.md", "c.md"} {
		require.NoError(t, os.Chmod(filepath.Join(posts, name), 0o444))
	}
	t.Cleanup(func() {
		for _, name := range []string{"a.md", "b.md", "c.md"} {
			os.Chmod(filepath.Join(posts, name), 0o644)
		}
	})

	_, err := r.RenameTag(2, "blog")
	require.Error(t, err)
	assert.Len(t, tagNames(r), 3, "the tag is still there")
	a, _ := r.GetPost("a")
	assert.Equal(t, []int{2, 3}, a.TagIds)
}

// Saving must not reformat a file: the content repository's history should show what the author changed.
func TestPostFilesAreStableTextFiles(t *testing.T) {
	r := setupTestRepo(t)
	_, err := r.PostSave(model.Post{Title: "T", Identity: "stable", Content: "正文", TagIds: []int{61, 62}})
	require.NoError(t, err)
	path := filepath.Join(r.dataDir, "posts", "stable.md")
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(written), "tag_ids: [61, 62]\n", "same separator as the migrated content files")
	assert.True(t, strings.HasSuffix(string(written), "\n\n正文\n"), "a text file ends with exactly one newline")

	again := postToFrontmatter(r.parsePost(string(written), "stable"))
	assert.Equal(t, string(written), again, "parse → write is a fixed point")
}

func TestCategoryDeleteRefusesWhileInUse(t *testing.T) {
	r := setupTestRepo(t)
	used, err := r.CategorySave(model.Category{Name: " 技术 "})
	require.NoError(t, err)
	empty, err := r.CategorySave(model.Category{Name: "随笔"})
	require.NoError(t, err)
	_, err = r.PostSave(model.Post{Title: "p", Identity: "p", Content: "x", CategoryId: used})
	require.NoError(t, err)

	category, _ := r.GetCategory(used)
	assert.Equal(t, "技术", category.Name, "names are stored trimmed")

	_, err = r.CategoryDelete(model.Category{Id: used})
	var inUse *CategoryInUseError
	require.ErrorAs(t, err, &inUse)
	assert.Equal(t, 1, inUse.Posts)
	_, err = r.GetCategory(used)
	assert.NoError(t, err, "still there")

	_, err = r.CategoryDelete(model.Category{Id: empty})
	assert.NoError(t, err)

	_, err = r.CategorySave(model.Category{Name: "  "})
	assert.ErrorIs(t, err, ErrCategoryNameEmpty)
}

// Posts written with an editor and pushed with git never pass through the admin: the counts kept in
// tags.json go stale, the ones shown on the site must not.
func TestTagCountsAreRecountedAtStartup(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "posts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tags.json"),
		[]byte(`[{"id":1,"name":"go","count":7},{"id":2,"name":"db","count":0}]`), 0o644))
	for slug, header := range map[string]string{
		"one":    "status: 1\ntag_ids: [1, 2]",
		"two":    "status: 1\ntag_ids: [2]",
		"hidden": "status: 0\ntag_ids: [1]",
	} {
		raw := "---\ntitle: \"" + slug + "\"\n" + header + "\n---\n\nbody\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, "posts", slug+".md"), []byte(raw), 0o644))
	}

	r, err := NewFileRepository(dir, "", "")
	require.NoError(t, err)
	t.Cleanup(r.Close)

	counts := make(map[string]int)
	tags, _ := r.GetTags()
	for _, tag := range tags {
		counts[tag.Name] = tag.Count
	}
	assert.Equal(t, map[string]int{"go": 1, "db": 2}, counts)
}

// A merge shows up in the content repository as one changed line per post, whatever the file looks like.
func TestRetagChangesOnlyTheTagLine(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "posts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tags.json"),
		[]byte(`[{"id":1,"name":"blog","count":0},{"id":2,"name":" blog","count":1},{"id":3,"name":"go","count":1}]`), 0o644))

	files := map[string]string{
		// leading blank lines, a description with trailing blanks, no newline at the end, old "[a,b]" style
		"odd": "---\ntitle: \"Odd\"\nstatus: 1\ncreated_at: 2024-01-02T10:00:00+08:00\nupdated_at: 2024-01-02T10:00:00+08:00\n" +
			"category_id: 1\nis_top: 0\ntag_ids: [2,3]\ndescription: \"两行\n摘要  \"\nword_count: 2\n---\n\n\n\n正文  \n\n```\ncode\n```",
		// Windows line endings
		"crlf": "---\r\ntitle: \"CRLF\"\r\nstatus: 1\r\ntag_ids: [2]\r\ndescription: \"d\"\r\n---\r\n\r\nbody\r\n",
		// a line that only looks like the header field, inside the quoted description, before the real one
		"decoy": "---\ntitle: \"Decoy\"\nstatus: 1\ndescription: \"first\ntag_ids: [2]\nlast\"\ntag_ids: [2, 3]\n---\n\nbody\n",
	}
	for slug, raw := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "posts", slug+".md"), []byte(raw), 0o644))
	}
	r, err := NewFileRepository(dir, "", "")
	require.NoError(t, err)
	t.Cleanup(r.Close)

	_, err = r.RenameTag(2, "blog")
	require.NoError(t, err)

	read := func(slug string) string {
		raw, err := os.ReadFile(filepath.Join(dir, "posts", slug+".md"))
		require.NoError(t, err)
		return string(raw)
	}
	assert.Equal(t, strings.Replace(files["odd"], "tag_ids: [2,3]", "tag_ids: [1, 3]", 1), read("odd"))
	assert.Equal(t, strings.Replace(files["crlf"], "tag_ids: [2]", "tag_ids: [1]", 1), read("crlf"))

	// the decoy cannot be patched in place, so the post is written the regular way — and still correct
	decoy, _ := reload(t, r).GetPost("decoy")
	assert.Equal(t, []int{1, 3}, decoy.TagIds)
	assert.Equal(t, "first\ntag_ids: [2]\nlast", decoy.Description)
	assert.Equal(t, "body", decoy.Content)
}
