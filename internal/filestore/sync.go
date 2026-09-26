package filestore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The content repository is not only written by this server: posts are also
// pushed from a laptop or edited on GitHub. Sync brings such changes in while
// the server runs, and makes sure the server's own commits (views.json,
// likes.json, whatever the admin saved) still reach the remote afterwards.

var (
	// ErrSyncConflict means local and remote changes touch the same lines; the
	// local state is kept and someone has to merge by hand.
	ErrSyncConflict = errors.New("the content repository and its remote changed the same lines")
	// ErrNoRemote means the data directory is not a git checkout with an upstream branch.
	ErrNoRemote = errors.New("the data directory has no remote to sync with")
)

// SyncResult says what a sync did.
type SyncResult struct {
	// Pulled is how many commits came in from the remote.
	Pulled int
	// Pushed is how many local commits went out.
	Pushed int
	// Head is the short id of the checked-out commit afterwards.
	Head string
}

// syncState is the part of FileRepository that coordinates syncs.
type syncState struct {
	hooksMu sync.Mutex
	hooks   []func()
	request chan struct{}
}

// OnReload registers fn to run after content pulled from the remote was loaded
// (feeds, sitemap and other derived data need rebuilding then).
func (r *FileRepository) OnReload(fn func()) {
	r.sync.hooksMu.Lock()
	defer r.sync.hooksMu.Unlock()
	r.sync.hooks = append(r.sync.hooks, fn)
}

// GitEnabled reports whether the data directory is a git checkout.
func (r *FileRepository) GitEnabled() bool {
	return r.gitEnabled
}

// git runs a git command in the data directory. It never waits for a
// password prompt: a missing credential fails right away.
func (r *FileRepository) git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.dataDir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Sync fetches the remote and, when it has new commits, replays this server's
// unpushed commits on top of them, loads the new content into memory and
// pushes. It is safe to call at any time; concurrent calls run one after another.
func (r *FileRepository) Sync(ctx context.Context) (SyncResult, error) {
	if !r.gitEnabled {
		return SyncResult{}, ErrNoRemote
	}
	r.gitMu.Lock()
	result, reloaded, err := r.syncLocked(ctx)
	if err == nil && result.Pushed > 0 {
		if _, pushErr := r.git(ctx, "push", "--quiet"); pushErr != nil {
			err = pushErr
			result.Pushed = 0
		}
	}
	r.gitMu.Unlock()
	if reloaded {
		r.runReloadHooks()
	}
	return result, err
}

// syncLocked does the fetch / rebase / reload part of Sync. The caller holds
// gitMu. reloaded says whether new content was loaded into memory.
func (r *FileRepository) syncLocked(ctx context.Context) (result SyncResult, reloaded bool, err error) {
	pulled, moved, err := r.integrateLocked(ctx, true)
	if err != nil {
		return result, false, err
	}
	if moved {
		if err := r.reloadContent(); err != nil {
			// keep serving what is in memory; the next sync or restart tries again
			slog.Error("content repository: the pulled content could not be loaded", "err", err)
			return result, false, err
		}
		reloaded = true
	}
	result.Pulled = pulled
	result.Pushed, _ = r.countCommits(ctx, "@{u}..HEAD")
	result.Head, _ = r.git(ctx, "rev-parse", "--short", "HEAD")
	return result, reloaded, nil
}

// integrateLocked fetches and, when the remote has new commits, replays the
// local ones on top of them (they have not been pushed, so nobody else has
// them). withCounters commits the in-memory view and like counters first —
// not at startup, before they are loaded. The caller holds gitMu.
func (r *FileRepository) integrateLocked(ctx context.Context, withCounters bool) (pulled int, moved bool, err error) {
	if _, err := r.git(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err != nil {
		return 0, false, ErrNoRemote
	}
	if _, err := r.git(ctx, "fetch", "--quiet"); err != nil {
		return 0, false, err
	}
	behind, err := r.countCommits(ctx, "HEAD..@{u}")
	if err != nil || behind == 0 {
		return 0, false, err
	}
	before, _ := r.git(ctx, "rev-parse", "HEAD")
	// Everything local becomes a commit first, so the rebase replays it like any other local commit and a conflict
	// ends in a clean abort. (An autostash could not: a change the admin just saved, whose commit is still
	// waiting for gitMu, would come back as conflict markers in the working tree.)
	r.commitPendingLocked(ctx, withCounters)
	if _, err := r.git(ctx, "rebase", "@{u}"); err != nil {
		slog.Error("content repository: local and remote changes conflict, keeping the local state", "err", err)
		if _, abortErr := r.git(context.Background(), "rebase", "--abort"); abortErr != nil {
			slog.Error("content repository: git rebase --abort failed", "err", abortErr)
		}
		return 0, false, ErrSyncConflict
	}
	after, _ := r.git(ctx, "rev-parse", "HEAD")
	return behind, before != after, nil
}

func (r *FileRepository) countCommits(ctx context.Context, revisions string) (int, error) {
	out, err := r.git(ctx, "rev-list", "--count", revisions)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

// commitPendingLocked commits what is not committed yet: the view and like
// counters in their usual commit (written from memory first when flush is set
// — not at startup, before they are loaded), then anything else, e.g. an admin
// save whose own commit is still queued. The caller holds gitMu.
func (r *FileRepository) commitPendingLocked(ctx context.Context, flush bool) {
	if flush {
		if err := r.flushViews(); err != nil {
			slog.Error("flush views failed", "err", err)
		}
		if err := r.flushLikes(); err != nil {
			slog.Error("flush likes failed", "err", err)
		}
	}
	r.commitCountersLocked(ctx)
	if out, err := r.git(ctx, "status", "--porcelain"); err != nil || out == "" {
		return
	}
	if _, err := r.git(ctx, "add", "-A"); err != nil {
		slog.Error("git add failed", "err", err)
		return
	}
	if _, err := r.git(ctx, "diff", "--cached", "--quiet"); err == nil {
		return
	}
	if _, err := r.git(ctx, "commit", "--quiet", "-m", "chore: commit local changes before syncing"); err != nil {
		slog.Error("git commit failed", "err", err)
	}
}

// commitCountersLocked commits views.json / likes.json when they changed on
// disk. The caller holds gitMu.
func (r *FileRepository) commitCountersLocked(ctx context.Context) {
	if _, err := os.Stat(filepath.Join(r.dataDir, "views.json")); err != nil {
		return
	}
	paths := []string{"views.json"}
	if _, err := os.Stat(filepath.Join(r.dataDir, "likes.json")); err == nil {
		paths = append(paths, "likes.json")
	}
	if _, err := r.git(ctx, append([]string{"add", "--"}, paths...)...); err != nil {
		slog.Error("git add counters failed", "err", err)
		return
	}
	if _, err := r.git(ctx, append([]string{"diff", "--cached", "--quiet", "--"}, paths...)...); err == nil {
		return // nothing new
	}
	if _, err := r.git(ctx, append([]string{"commit", "--quiet", "-m", "chore: update views.json", "--"}, paths...)...); err != nil {
		slog.Error("git commit counters failed", "err", err)
	}
}

// reloadContent reads posts, pages, tags, categories, books, projects and
// users from disk again. Everything is loaded into a scratch repository first
// and only swapped in when all of it parsed, so a broken file pushed from
// elsewhere never leaves the site half loaded. View and like counters are not
// read: the ones in memory are newer.
func (r *FileRepository) reloadContent() error {
	r.mu.RLock()
	fresh := &FileRepository{
		dataDir:        r.dataDir,
		views:          maps.Clone(r.views),
		likes:          maps.Clone(r.likes),
		nextCategoryId: 1,
		nextTagId:      1,
		nextBookId:     1,
		nextProjectId:  1,
	}
	r.mu.RUnlock()
	for _, load := range []func() error{fresh.loadCategories, fresh.loadTags, fresh.loadPages, fresh.loadUsers, fresh.loadBooks, fresh.loadProjects, fresh.loadPosts} {
		if err := load(); err != nil {
			return err
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.categories, r.tags, r.pages, r.users, r.books, r.projects = fresh.categories, fresh.tags, fresh.pages, fresh.users, fresh.books, fresh.projects
	r.posts, r.postById, r.postBySlug = fresh.posts, fresh.postById, fresh.postBySlug
	// ids only ever grow: an id deleted on the remote is not handed out again
	r.nextCategoryId = max(r.nextCategoryId, fresh.nextCategoryId)
	r.nextTagId = max(r.nextTagId, fresh.nextTagId)
	r.nextBookId = max(r.nextBookId, fresh.nextBookId)
	r.nextProjectId = max(r.nextProjectId, fresh.nextProjectId)
	// counters may have moved while the files were read
	for _, p := range r.posts {
		p.Views = r.views[p.Id]
		p.Likes = r.likes[p.Id]
	}
	r.recountTags()
	return nil
}

func (r *FileRepository) runReloadHooks() {
	r.sync.hooksMu.Lock()
	hooks := append([]func(){}, r.sync.hooks...)
	r.sync.hooksMu.Unlock()
	for _, hook := range hooks {
		hook()
	}
}

// RequestSync asks for a sync soon (a webhook said the remote changed). It
// never blocks; requests arriving while one is pending are merged.
func (r *FileRepository) RequestSync() {
	if !r.gitEnabled || r.sync.request == nil {
		return
	}
	select {
	case r.sync.request <- struct{}{}:
	default:
	}
}

// StartSync syncs every interval (never when interval <= 0) and whenever
// RequestSync is called, until the repository is closed. Call it after the
// reload hooks are registered.
func (r *FileRepository) StartSync(interval time.Duration) {
	if !r.gitEnabled {
		return
	}
	r.sync.request = make(chan struct{}, 1)
	go func() {
		var tick <-chan time.Time
		if interval > 0 {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			tick = ticker.C
		}
		for {
			select {
			case <-r.done:
				return
			case <-tick:
			case <-r.sync.request:
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			result, err := r.Sync(ctx)
			cancel()
			switch {
			case errors.Is(err, ErrNoRemote):
			case err != nil:
				slog.Warn("content repository sync failed", "err", err)
			case result.Pulled > 0:
				slog.Info("content repository synced", "pulled", result.Pulled, "pushed", result.Pushed, "head", result.Head)
			}
		}
	}()
}
