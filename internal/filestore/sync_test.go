package filestore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goblog/internal/pkg/model"
)

const syncPost = `---
title: "%s"
status: 1
created_at: 2024-03-16T10:00:00+08:00
updated_at: 2024-03-16T10:00:00+08:00
category_id: 1
is_top: 0
tag_ids: [1]
description: "d"
word_count: 1
---

%s
`

// gitWorld is a bare "GitHub" repository, a laptop clone of it and the data
// directory the server will clone into.
type gitWorld struct {
	t       *testing.T
	remote  string
	laptop  string
	dataDir string
}

func newGitWorld(t *testing.T) *gitWorld {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	// the developer's own git settings (signing, hooks, pull.rebase…) must not leak in
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	root := t.TempDir()
	w := &gitWorld{t: t, remote: filepath.Join(root, "remote.git"), laptop: filepath.Join(root, "laptop"), dataDir: filepath.Join(root, "data")}
	w.run(root, "init", "-q", "--bare", "-b", "main", w.remote)
	w.run(root, "clone", "-q", w.remote, w.laptop)
	w.run(w.laptop, "config", "user.name", "laptop")
	w.run(w.laptop, "config", "user.email", "laptop@example.com")
	w.write("categories.json", `[{"id":1,"name":"技术"}]`)
	w.write("tags.json", `[{"id":1,"name":"go","count":0}]`)
	w.write("posts/first.md", strings.ReplaceAll(strings.Replace(syncPost, "%s", "First", 1), "%s", "first body"))
	w.push("initial content")
	return w
}

func (w *gitWorld) run(dir string, args ...string) string {
	w.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(w.t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func (w *gitWorld) write(name, content string) {
	w.t.Helper()
	path := filepath.Join(w.laptop, name)
	require.NoError(w.t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(w.t, os.WriteFile(path, []byte(content), 0o644))
}

// push commits everything on the laptop and pushes it, as the author would.
func (w *gitWorld) push(message string) {
	w.t.Helper()
	w.run(w.laptop, "add", "-A")
	w.run(w.laptop, "commit", "-q", "-m", message)
	if w.run(w.laptop, "ls-remote", "--heads", "origin", "main") != "" {
		w.run(w.laptop, "pull", "-q", "--rebase")
	}
	w.run(w.laptop, "push", "-q", "-u", "origin", "main")
}

func (w *gitWorld) remoteLog() string {
	return w.run(w.remote, "log", "--format=%s", "main")
}

// server starts a repository on the data directory (cloning it the first time).
func (w *gitWorld) server() *FileRepository {
	w.t.Helper()
	r, err := NewFileRepository(w.dataDir, w.remote, "")
	require.NoError(w.t, err)
	w.t.Cleanup(r.Close)
	return r
}

func (w *gitWorld) waitForRemote(want string) {
	w.t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !strings.Contains(w.remoteLog(), want) {
		if time.Now().After(deadline) {
			w.t.Fatalf("the remote never got %q:\n%s", want, w.remoteLog())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func post(title, body string) string {
	return strings.Replace(strings.Replace(syncPost, "%s", title, 1), "%s", body, 1)
}

func TestSyncBringsInPushedPosts(t *testing.T) {
	w := newGitWorld(t)
	r := w.server()
	var reloads atomic.Int32
	r.OnReload(func() { reloads.Add(1) })
	require.NoError(t, r.IncrView("first"))
	require.NoError(t, r.IncrView("first"))

	result, err := r.Sync(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Pulled, "nothing new yet")
	assert.Equal(t, int32(0), reloads.Load())

	w.write("posts/second.md", post("Second", "written on the laptop"))
	w.push("post: second")
	// a JSON file being written right now (writeFileAtomic) must not end up in a commit
	require.NoError(t, os.WriteFile(filepath.Join(w.dataDir, "views.json.tmp-123"), []byte("{"), 0o644))

	result, err = r.Sync(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Pulled)
	assert.NotEmpty(t, result.Head)
	assert.Equal(t, int32(1), reloads.Load(), "derived data is rebuilt once")

	second, err := r.GetPostByIdentity("second")
	require.NoError(t, err, "the pushed post is served without a restart")
	assert.Contains(t, second.Content, "written on the laptop")
	first, _ := r.GetPostByIdentity("first")
	assert.Equal(t, 2, first.Views, "view counts in memory survive the reload")
	tags, _ := r.GetTags()
	assert.Equal(t, 2, tags[0].Count, "tag counts follow the new posts")

	assert.Equal(t, 1, result.Pushed, "the counters went out on top of the laptop's commit")
	log := w.remoteLog()
	assert.True(t, strings.HasPrefix(log, "chore: update views.json\npost: second"), "history stays linear:\n%s", log)
	assert.Empty(t, w.run(w.dataDir, "status", "--porcelain"))
	assert.NotContains(t, w.run(w.dataDir, "ls-files"), ".tmp-")
}

func TestRejectedPushIsRebasedAndRetried(t *testing.T) {
	w := newGitWorld(t)
	r := w.server()
	var reloads atomic.Int32
	r.OnReload(func() { reloads.Add(1) })

	w.write("posts/laptop.md", post("From the laptop", "x"))
	w.push("post: laptop")

	// the server does not know about the laptop's commit: its push is rejected first
	_, err := r.PostSave(model.Post{Title: "From the admin", Identity: "admin", Content: "y", CategoryId: 1})
	require.NoError(t, err)
	w.waitForRemote("add post: From the admin")

	log := w.remoteLog()
	assert.Less(t, strings.Index(log, "add post: From the admin"), strings.Index(log, "post: laptop"), "the admin's commit was replayed on top:\n%s", log)
	deadline := time.Now().Add(5 * time.Second)
	for reloads.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	_, err = r.GetPostByIdentity("laptop")
	assert.NoError(t, err, "what the rebase brought in is loaded as well")
	assert.Equal(t, int32(1), reloads.Load())
}

func TestSyncConflictKeepsTheLocalState(t *testing.T) {
	w := newGitWorld(t)
	r := w.server()

	w.write("posts/first.md", post("First", "the laptop's version"))
	w.push("edit on the laptop")
	first, err := r.GetPost("first")
	require.NoError(t, err)
	first.Content = "the admin's version"
	_, err = r.PostSave(first) // its background push runs into the same conflict
	require.NoError(t, err)

	_, err = r.Sync(context.Background())
	assert.ErrorIs(t, err, ErrSyncConflict)
	for _, dir := range []string{"rebase-merge", "rebase-apply"} {
		_, statErr := os.Stat(filepath.Join(w.dataDir, ".git", dir))
		assert.True(t, os.IsNotExist(statErr), "no rebase is left half done")
	}
	assert.Empty(t, w.run(w.dataDir, "status", "--porcelain"), "the working tree is clean")
	got, _ := r.GetPost("first")
	assert.Equal(t, "the admin's version", strings.TrimSpace(got.Content), "the server keeps serving its own version")
	assert.Contains(t, w.run(w.remote, "show", "main:posts/first.md"), "the laptop's version", "nothing was forced onto the remote")
}

func TestSyncWithBrokenContentKeepsServing(t *testing.T) {
	w := newGitWorld(t)
	r := w.server()
	var reloads atomic.Int32
	r.OnReload(func() { reloads.Add(1) })

	w.write("tags.json", `[{"id":1,"name":"go"`)
	w.write("posts/third.md", post("Third", "z"))
	w.push("broken tags")

	_, err := r.Sync(context.Background())
	assert.Error(t, err)
	tags, _ := r.GetTags()
	require.Len(t, tags, 1, "the loaded tags stay")
	_, err = r.GetPostByIdentity("third")
	assert.Error(t, err, "nothing of the broken state is half loaded")
	assert.Equal(t, int32(0), reloads.Load())
}

func TestStartupReplaysUnpushedCommits(t *testing.T) {
	w := newGitWorld(t)
	r := w.server()
	r.Close()

	// the server stopped with a commit it could not push; meanwhile the author pushed a post
	require.NoError(t, os.WriteFile(filepath.Join(w.dataDir, "views.json"), []byte(`{"first":7}`), 0o644))
	w.run(w.dataDir, "add", "views.json")
	w.run(w.dataDir, "commit", "-q", "-m", "chore: update views.json")
	w.write("posts/while-down.md", post("While down", "w"))
	w.push("post: while down")

	restarted := w.server()
	_, err := restarted.GetPostByIdentity("while-down")
	assert.NoError(t, err, "a pull --ff-only used to give up here")
	first, _ := restarted.GetPostByIdentity("first")
	assert.Equal(t, 7, first.Views)
	assert.Equal(t, "chore: update views.json\npost: while down", strings.Join(strings.SplitN(w.run(w.dataDir, "log", "--format=%s"), "\n", 3)[:2], "\n"))
}

func TestRequestSync(t *testing.T) {
	w := newGitWorld(t)
	r := w.server()
	r.StartSync(0) // no schedule: only on request

	w.write("posts/hooked.md", post("Hooked", "h"))
	w.push("post: hooked")
	r.RequestSync()

	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := r.GetPostByIdentity("hooked"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("a requested sync brings the post in")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestSyncWithoutGit(t *testing.T) {
	r := setupTestRepo(t)
	_, err := r.Sync(context.Background())
	assert.ErrorIs(t, err, ErrNoRemote)
	r.RequestSync() // no-op, must not block
	assert.False(t, r.GitEnabled())
}
