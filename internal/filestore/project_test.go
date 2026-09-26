package filestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goblog/internal/pkg/model"
)

func TestProjectCRUD(t *testing.T) {
	r := setupTestRepo(t)

	id, err := r.ProjectSave(model.Project{Name: "goblog", Repo: "https://github.com/whrss9527/goblog", Tech: []string{"Go"}})
	require.NoError(t, err)
	assert.Equal(t, 1, id)

	got, err := r.GetProject(id)
	require.NoError(t, err)
	assert.Equal(t, "goblog", got.Name)
	assert.Equal(t, model.ProjectStatusActive, got.Status, "a missing status means active")
	assert.False(t, got.CreatedAt.IsZero())
	created := got.CreatedAt

	got.Description = "博客程序"
	got.CreatedAt = created.AddDate(-1, 0, 0) // the form never sends it; the stored value wins
	_, err = r.ProjectSave(got)
	require.NoError(t, err)
	again, err := r.GetProject(id)
	require.NoError(t, err)
	assert.Equal(t, "博客程序", again.Description)
	assert.Equal(t, created, again.CreatedAt)

	second, err := r.ProjectSave(model.Project{Name: "proxyswitch"})
	require.NoError(t, err)
	assert.Equal(t, 2, second)

	require.NoError(t, r.ProjectDelete(id))
	_, err = r.GetProject(id)
	assert.ErrorIs(t, err, ErrProjectNotFound)
	assert.ErrorIs(t, r.ProjectDelete(id), ErrProjectNotFound)

	// ids are not handed out again: anchors and bookmarks of a deleted project stay dead
	third, err := r.ProjectSave(model.Project{Name: "new one"})
	require.NoError(t, err)
	assert.Equal(t, 3, third)

	// an id that is gone (deleted in another tab) starts a new project instead of failing
	fourth, err := r.ProjectSave(model.Project{Id: 99, Name: "from a stale form"})
	require.NoError(t, err)
	assert.Equal(t, 4, fourth)
}

func TestProjectOrder(t *testing.T) {
	r := setupTestRepo(t)
	for _, p := range []model.Project{
		{Name: "old", Started: "2021-05", Status: model.ProjectStatusDone},
		{Name: "archived but new", Started: "2026-01", Status: model.ProjectStatusArchived},
		{Name: "recent", Started: "2026-09"},
		{Name: "undated"},
		{Name: "featured", Started: "2020-01", Featured: true},
		{Name: "same month, added later", Started: "2026-09"},
	} {
		_, err := r.ProjectSave(p)
		require.NoError(t, err)
	}
	projects, err := r.GetProjects()
	require.NoError(t, err)
	var names []string
	for _, p := range projects {
		names = append(names, p.Name)
	}
	assert.Equal(t, []string{"featured", "same month, added later", "recent", "old", "undated", "archived but new"}, names)
}

func TestProjectNormalize(t *testing.T) {
	r := setupTestRepo(t)

	_, err := r.ProjectSave(model.Project{Name: "   "})
	assert.ErrorIs(t, err, ErrProjectNameEmpty)
	_, statErr := os.Stat(filepath.Join(r.dataDir, projectsFile))
	assert.True(t, os.IsNotExist(statErr), "a rejected project writes nothing")

	id, err := r.ProjectSave(model.Project{
		Name:       "  spaced  ",
		Repo:       " https://github.com/a/b ",
		Tech:       []string{"Go", " go ", "", "Vue", "GO"},
		Highlights: []string{" 离线可用 ", "", "   "},
		Status:     42,
	})
	require.NoError(t, err)
	got, err := r.GetProject(id)
	require.NoError(t, err)
	assert.Equal(t, "spaced", got.Name)
	assert.Equal(t, "https://github.com/a/b", got.Repo)
	assert.Equal(t, []string{"Go", "Vue"}, got.Tech)
	assert.Equal(t, []string{"离线可用"}, got.Highlights)
	assert.Equal(t, model.ProjectStatusActive, got.Status)

	// callers get copies: changing them does not reach the store
	got.Tech[0] = "changed"
	again, _ := r.GetProject(id)
	assert.Equal(t, "Go", again.Tech[0])
}

func TestProjectFileFormat(t *testing.T) {
	r := setupTestRepo(t)
	_, err := r.ProjectSave(model.Project{Name: "A & B <tool>", Description: "读 -> 写", Started: "2026-09"})
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(r.dataDir, projectsFile))
	require.NoError(t, err)
	text := string(raw)
	assert.Contains(t, text, `"name": "A & B <tool>"`, "prose stays readable, no \\u0026")
	assert.Contains(t, text, `"description": "读 -> 写"`)
	assert.Contains(t, text, `"tech": []`, "empty lists are written as []")
	assert.Contains(t, text, `"highlights": []`)
	assert.True(t, strings.HasSuffix(text, "}\n]\n"), "the file ends with one newline")

	// a fresh repository reads the file back to the same projects
	reloaded := setupTestRepo(t)
	reloaded.dataDir = r.dataDir
	require.NoError(t, reloaded.loadProjects())
	want, _ := r.GetProjects()
	got, _ := reloaded.GetProjects()
	require.Len(t, got, 1)
	assert.Equal(t, want[0].Name, got[0].Name)
	assert.True(t, want[0].CreatedAt.Equal(got[0].CreatedAt))
	next, err := reloaded.ProjectSave(model.Project{Name: "next"})
	require.NoError(t, err)
	assert.Equal(t, 2, next, "ids continue after the highest one in the file")
}

func TestProjectHandEditedFile(t *testing.T) {
	r := setupTestRepo(t)
	file := `[
  {"id": 7, "name": " hand made ", "tech": null, "status": 0},
  {"id": 3, "name": "second", "repo": "https://github.com/a/b", "featured": true}
]`
	require.NoError(t, os.WriteFile(filepath.Join(r.dataDir, projectsFile), []byte(file), 0o644))
	require.NoError(t, r.loadProjects())

	projects, err := r.GetProjects()
	require.NoError(t, err)
	require.Len(t, projects, 2)
	assert.Equal(t, "second", projects[0].Name, "featured first")
	assert.Equal(t, "hand made", projects[1].Name)
	assert.Equal(t, model.ProjectStatusActive, projects[1].Status)

	// saving anything rewrites the file in the canonical form, without invented timestamps
	_, err = r.ProjectSave(model.Project{Name: "added"})
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(r.dataDir, projectsFile))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "null")
	assert.NotContains(t, string(raw), "0001-01-01")
	id, err := r.ProjectSave(model.Project{Name: "after"})
	require.NoError(t, err)
	assert.Equal(t, 9, id, "new ids start above the highest one found in the file")
}

func TestProjectWriteFailureKeepsMemory(t *testing.T) {
	r := setupTestRepo(t)
	id, err := r.ProjectSave(model.Project{Name: "kept"})
	require.NoError(t, err)

	// a directory in place of the file makes the final rename fail, even for root
	target := filepath.Join(r.dataDir, projectsFile)
	require.NoError(t, os.Remove(target))
	require.NoError(t, os.MkdirAll(filepath.Join(target, "blocker"), 0o755))

	_, err = r.ProjectSave(model.Project{Id: id, Name: "renamed"})
	assert.Error(t, err)
	_, err = r.ProjectSave(model.Project{Name: "added"})
	assert.Error(t, err)
	assert.Error(t, r.ProjectDelete(id))

	projects, _ := r.GetProjects()
	require.Len(t, projects, 1)
	assert.Equal(t, "kept", projects[0].Name)
	assert.Equal(t, 2, r.nextProjectId, "a failed add does not use up an id")
}

func TestAtomicWritesAreOrdinaryFiles(t *testing.T) {
	r := setupTestRepo(t)
	_, err := r.ProjectSave(model.Project{Name: "x"})
	require.NoError(t, err)
	require.NoError(t, r.SaveImage("2026/09/1758854400123.png", []byte("png")))
	for _, name := range []string{projectsFile, "images/2026/09/1758854400123.png"} {
		info, err := os.Stat(filepath.Join(r.dataDir, name))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm(), name)
	}
	assert.ErrorIs(t, r.SaveImage("2026/09/1758854400123.png", []byte("other")), ErrImageExists, "images are never replaced")
	for _, bad := range []string{"../x.png", "2026/09/a.svg", "2026/9/1.png", "x.png", "2026/09/../../posts/x.png"} {
		assert.Error(t, r.SaveImage(bad, []byte("x")), bad)
	}
}
