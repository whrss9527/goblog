package filestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"goblog/internal/pkg/model"
)

type FileRepository struct {
	dataDir string

	mu         sync.RWMutex
	posts      []*model.Post
	postById   map[string]*model.Post
	postBySlug map[string]*model.Post
	categories []model.Category
	tags       []model.Tag
	pages      []model.Page
	users      []model.User
	books      []model.Book
	projects   []model.Project
	views      map[string]int
	likes      map[string]int

	nextCategoryId int
	nextTagId      int
	nextBookId     int
	nextProjectId  int

	gitEnabled bool
	gitMu      sync.Mutex
	sync       syncState
	done       chan struct{}
	closeOnce  sync.Once
}

func NewFileRepository(dataDir, gitRepo, gitToken string) (*FileRepository, error) {
	r := &FileRepository{
		dataDir:        dataDir,
		postById:       make(map[string]*model.Post),
		postBySlug:     make(map[string]*model.Post),
		views:          make(map[string]int),
		likes:          make(map[string]int),
		nextCategoryId: 1,
		nextTagId:      1,
		nextBookId:     1,
		nextProjectId:  1,
		done:           make(chan struct{}),
	}

	if err := r.ensureDataDir(gitRepo, gitToken); err != nil {
		return nil, fmt.Errorf("ensure data dir: %w", err)
	}

	if err := r.loadAll(); err != nil {
		return nil, err
	}
	// JSON files are written through a temporary sibling (writeFileAtomic); a
	// "git add -A" at the wrong moment must not commit one
	if err := r.excludeFromGit("*.tmp-*", "half-written files are never committed"); err != nil {
		slog.Warn("could not exclude temporary files from git", "err", err)
	}

	go r.flushViewsLoop()
	go r.pushViewsLoop()

	return r, nil
}

func (r *FileRepository) ensureDataDir(gitRepo, gitToken string) error {
	gitDir := filepath.Join(r.dataDir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		r.gitEnabled = true
		if gitToken != "" {
			r.configureGitToken(gitToken)
		}
		r.configureGitIdentity()
		// what was pushed from elsewhere comes first; commits this server could
		// not push before it stopped are replayed on top (a plain pull --ff-only
		// used to give up here for good once both sides had new commits)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		pulled, _, err := r.integrateLocked(ctx, false)
		switch {
		case errors.Is(err, ErrNoRemote):
		case err != nil:
			slog.Warn("git pull failed, using local data", "err", err)
		default:
			slog.Info("git pull completed", "new_commits", pulled)
		}
		return nil
	}

	if gitRepo == "" {
		if _, err := os.Stat(r.dataDir); err == nil {
			return nil
		}
		return fmt.Errorf("data_dir %s does not exist and git_repo is not configured", r.dataDir)
	}

	cloneURL := gitRepo
	if gitToken != "" {
		cloneURL = strings.Replace(gitRepo, "https://", "https://"+gitToken+"@", 1)
	}
	slog.Info("cloning data repo", "repo", gitRepo, "dir", r.dataDir)
	cmd := exec.Command("git", "clone", cloneURL, r.dataDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
	}
	r.gitEnabled = true
	r.configureGitIdentity()
	slog.Info("data repo cloned successfully")
	return nil
}

// configureGitIdentity sets local git user.name/email if they are not already
// configured. Required for commits made by the application (views push,
// admin operations) to work on a fresh server where global git identity is
// missing.
func (r *FileRepository) configureGitIdentity() {
	for _, key := range []string{"user.name", "user.email"} {
		cmd := exec.Command("git", "-C", r.dataDir, "config", "--local", key)
		if err := cmd.Run(); err == nil {
			continue // already set locally
		}
		var value string
		switch key {
		case "user.name":
			value = "goblog"
		case "user.email":
			value = "goblog@localhost"
		}
		exec.Command("git", "-C", r.dataDir, "config", "--local", key, value).Run()
	}
}

func (r *FileRepository) configureGitToken(token string) {
	cmd := exec.Command("git", "-C", r.dataDir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	remoteURL := strings.TrimSpace(string(out))
	if strings.Contains(remoteURL, "@") {
		return
	}
	newURL := strings.Replace(remoteURL, "https://", "https://"+token+"@", 1)
	exec.Command("git", "-C", r.dataDir, "remote", "set-url", "origin", newURL).Run()
	slog.Info("git remote configured with token")
}

func (r *FileRepository) loadAll() error {
	if err := r.loadCategories(); err != nil {
		return fmt.Errorf("load categories: %w", err)
	}
	if err := r.loadTags(); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	if err := r.loadPages(); err != nil {
		return fmt.Errorf("load pages: %w", err)
	}
	if err := r.loadUsers(); err != nil {
		return fmt.Errorf("load users: %w", err)
	}
	if err := r.loadBooks(); err != nil {
		return fmt.Errorf("load books: %w", err)
	}
	if err := r.loadProjects(); err != nil {
		return fmt.Errorf("load projects: %w", err)
	}
	if err := r.loadViews(); err != nil {
		return fmt.Errorf("load views: %w", err)
	}
	if err := r.loadLikes(); err != nil {
		return fmt.Errorf("load likes: %w", err)
	}
	if err := r.loadPosts(); err != nil {
		return fmt.Errorf("load posts: %w", err)
	}
	// Posts may have been added or retagged with an editor and git instead of the admin: the numbers
	// stored in tags.json are only as fresh as the last save made here. (In memory only; the file
	// follows with the next change.)
	r.recountTags()
	return nil
}

func (r *FileRepository) loadCategories() error {
	r.categories = nil
	data, err := os.ReadFile(filepath.Join(r.dataDir, "categories.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.categories); err != nil {
		return err
	}
	for _, c := range r.categories {
		if c.Id >= r.nextCategoryId {
			r.nextCategoryId = c.Id + 1
		}
	}
	return nil
}

func (r *FileRepository) loadTags() error {
	r.tags = nil
	data, err := os.ReadFile(filepath.Join(r.dataDir, "tags.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.tags); err != nil {
		return err
	}
	for _, t := range r.tags {
		if t.Id >= r.nextTagId {
			r.nextTagId = t.Id + 1
		}
	}
	return nil
}

func (r *FileRepository) loadPages() error {
	r.pages = nil
	dir := filepath.Join(r.dataDir, "pages")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		meta, content := parseFrontmatter(string(data))
		page := model.Page{
			Id:      meta["id"],
			Title:   meta["title"],
			Content: content,
		}
		r.pages = append(r.pages, page)
	}
	return nil
}

func (r *FileRepository) loadUsers() error {
	r.users = nil
	data, err := os.ReadFile(filepath.Join(r.dataDir, "users.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &r.users)
}

func (r *FileRepository) loadBooks() error {
	r.books = nil
	data, err := os.ReadFile(filepath.Join(r.dataDir, "books.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.books); err != nil {
		return err
	}
	for _, b := range r.books {
		if b.Id >= r.nextBookId {
			r.nextBookId = b.Id + 1
		}
	}
	return nil
}

func (r *FileRepository) loadViews() error {
	data, err := os.ReadFile(filepath.Join(r.dataDir, "views.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &r.views)
}

func (r *FileRepository) loadLikes() error {
	r.likes = make(map[string]int)
	data, err := os.ReadFile(filepath.Join(r.dataDir, "likes.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &r.likes)
}

func (r *FileRepository) loadPosts() error {
	r.posts = nil
	r.postById = make(map[string]*model.Post)
	r.postBySlug = make(map[string]*model.Post)

	dir := filepath.Join(r.dataDir, "posts")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		post := r.parsePost(string(data), slug)
		if v, ok := r.views[post.Id]; ok {
			post.Views = v
		}
		post.Likes = r.likes[post.Id]
		r.posts = append(r.posts, post)
		r.postById[post.Id] = post
		r.postBySlug[post.Identity] = post
	}

	sort.Slice(r.posts, func(i, j int) bool {
		if r.posts[i].IsTop != r.posts[j].IsTop {
			return r.posts[i].IsTop > r.posts[j].IsTop
		}
		return r.posts[i].CreatedAt.After(r.posts[j].CreatedAt)
	})

	return nil
}

// saveJSON writes v to filename inside dataDir atomically: marshal to a
// sibling temp file, fsync, then rename over the target. This avoids leaving a
// half-written file behind if the process is killed mid-write.
func (r *FileRepository) saveJSON(filename string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return r.writeFileAtomic(filename, data)
}

// writeFileAtomic replaces filename inside dataDir with data (temp file, fsync, rename).
func (r *FileRepository) writeFileAtomic(filename string, data []byte) error {
	target := filepath.Join(r.dataDir, filename)
	tmp, err := os.CreateTemp(r.dataDir, filename+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, target)
}

func (r *FileRepository) flushViews() error {
	r.mu.RLock()
	viewsCopy := make(map[string]int, len(r.views))
	for k, v := range r.views {
		viewsCopy[k] = v
	}
	r.mu.RUnlock()
	return r.saveJSON("views.json", viewsCopy)
}

// flushLikes persists the like counters. Blogs that never received a like do
// not get an empty likes.json added to their data repository.
func (r *FileRepository) flushLikes() error {
	r.mu.RLock()
	likesCopy := make(map[string]int, len(r.likes))
	for k, v := range r.likes {
		likesCopy[k] = v
	}
	r.mu.RUnlock()
	if len(likesCopy) == 0 {
		return nil
	}
	return r.saveJSON("likes.json", likesCopy)
}

func (r *FileRepository) flushViewsLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			if err := r.flushViews(); err != nil {
				slog.Error("flush views failed", "err", err)
			}
			if err := r.flushLikes(); err != nil {
				slog.Error("flush likes failed", "err", err)
			}
		}
	}
}

func (r *FileRepository) pushViewsLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			if err := r.flushViews(); err != nil {
				slog.Error("flush views failed", "err", err)
				continue
			}
			r.gitCommitAndPushPath("views.json", "chore: update views.json")
			if err := r.flushLikes(); err != nil {
				slog.Error("flush likes failed", "err", err)
				continue
			}
			if _, err := os.Stat(filepath.Join(r.dataDir, "likes.json")); err == nil {
				r.gitCommitAndPushPath("likes.json", "chore: update likes.json")
			}
		}
	}
}

func (r *FileRepository) Done() <-chan struct{} {
	return r.done
}

// Close stops the background loops and writes the counters one last time. It
// may be called more than once.
func (r *FileRepository) Close() {
	r.closeOnce.Do(func() {
		close(r.done)
		if err := r.flushViews(); err != nil {
			slog.Error("final flush views failed", "err", err)
		} else {
			slog.Info("views flushed on shutdown")
		}
		if err := r.flushLikes(); err != nil {
			slog.Error("final flush likes failed", "err", err)
		}
	})
}

// gitPushWithRetry runs `git push` from r.dataDir. A push rejected because the
// remote has commits this server does not (a post pushed from a laptop) is
// followed by a sync — fetch, rebase, reload — and pushed again right away;
// other failures are retried with exponential backoff. The push includes any
// previously-failed commits, so we never lose data — at worst the next
// successful push catches up. Logged as Warn (not Error) on each attempt and
// only Error after all retries are exhausted. The caller holds gitMu.
func (r *FileRepository) gitPushWithRetry() {
	const maxAttempts = 3
	backoff := 5 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		out, err := r.git(context.Background(), "push")
		if err == nil {
			return
		}
		if attempt == maxAttempts {
			slog.Error("git push failed after retries", "attempts", maxAttempts, "err", err)
			return
		}
		if strings.Contains(out, "[rejected]") {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			_, reloaded, syncErr := r.syncLocked(ctx)
			cancel()
			if reloaded {
				go r.runReloadHooks()
			}
			if syncErr != nil {
				slog.Error("git push rejected and the remote changes could not be merged", "err", syncErr)
				return
			}
			continue
		}
		slog.Warn("git push transient failure, will retry", "attempt", attempt, "err", err, "next_delay", backoff)
		time.Sleep(backoff)
		backoff *= 3
	}
}

// gitCommitAndPushPath stages exactly the given path (relative to dataDir),
// commits it if there are changes, and pushes. Use this when you want a clean
// commit scoped to a single file rather than gitCommitAndPush which adds -A.
func (r *FileRepository) gitCommitAndPushPath(path, message string) {
	if !r.gitEnabled {
		return
	}
	go func() {
		r.gitMu.Lock()
		defer r.gitMu.Unlock()
		cmd := exec.Command("git", "-C", r.dataDir, "add", "--", path)
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Error("git add failed", "err", err, "output", string(out), "path", path)
			return
		}
		cmd = exec.Command("git", "-C", r.dataDir, "diff", "--cached", "--quiet", "--", path)
		if err := cmd.Run(); err == nil {
			return
		}
		cmd = exec.Command("git", "-C", r.dataDir, "commit", "-m", message, "--", path)
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Error("git commit failed", "err", err, "output", string(out))
			return
		}
		r.gitPushWithRetry()
	}()
}

func (r *FileRepository) gitCommitAndPush(message string) {
	if !r.gitEnabled {
		return
	}
	go func() {
		r.gitMu.Lock()
		defer r.gitMu.Unlock()
		cmd := exec.Command("git", "-C", r.dataDir, "add", "-A")
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Error("git add failed", "err", err, "output", string(out))
			return
		}

		cmd = exec.Command("git", "-C", r.dataDir, "diff", "--cached", "--quiet")
		if err := cmd.Run(); err == nil {
			return
		}

		cmd = exec.Command("git", "-C", r.dataDir, "commit", "-m", message)
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Error("git commit failed", "err", err, "output", string(out))
			return
		}

		r.gitPushWithRetry()
	}()
}
