package filestore

import (
	"encoding/json"
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
	views      map[string]int

	nextCategoryId int
	nextTagId      int
	nextBookId     int

	gitEnabled bool
	gitMu      sync.Mutex
	done       chan struct{}
}

func NewFileRepository(dataDir, gitRepo, gitToken string) (*FileRepository, error) {
	r := &FileRepository{
		dataDir:        dataDir,
		postById:       make(map[string]*model.Post),
		postBySlug:     make(map[string]*model.Post),
		views:          make(map[string]int),
		nextCategoryId: 1,
		nextTagId:      1,
		nextBookId:     1,
		done:           make(chan struct{}),
	}

	if err := r.ensureDataDir(gitRepo, gitToken); err != nil {
		return nil, fmt.Errorf("ensure data dir: %w", err)
	}

	if err := r.loadAll(); err != nil {
		return nil, err
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
		cmd := exec.Command("git", "-C", r.dataDir, "pull", "--ff-only")
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Warn("git pull failed, using local data", "err", err, "output", string(out))
		} else {
			slog.Info("git pull completed")
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
	if err := r.loadViews(); err != nil {
		return fmt.Errorf("load views: %w", err)
	}
	if err := r.loadPosts(); err != nil {
		return fmt.Errorf("load posts: %w", err)
	}
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
		}
	}
}

func (r *FileRepository) Done() <-chan struct{} {
	return r.done
}

func (r *FileRepository) Close() {
	close(r.done)
	if err := r.flushViews(); err != nil {
		slog.Error("final flush views failed", "err", err)
	} else {
		slog.Info("views flushed on shutdown")
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
		cmd = exec.Command("git", "-C", r.dataDir, "push")
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Error("git push failed", "err", err, "output", string(out))
		}
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

		cmd = exec.Command("git", "-C", r.dataDir, "push")
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Error("git push failed", "err", err, "output", string(out))
		}
	}()
}
