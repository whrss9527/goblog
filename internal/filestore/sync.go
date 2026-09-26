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
	"syscall"
	"time"
)

// The content repository is not only written by this server: posts are also
// pushed from a laptop or edited on GitHub. Sync brings such changes in while
// the server runs, and makes sure the server's own commits (views.json,
// likes.json, whatever the admin saved) still reach the remote afterwards.
//
// Locks, always taken in this order: gitMu (one git operation at a time) →
// wmu (the working tree: saves never interleave with a commit, rebase, abort or
// reload) → mu (the data in memory).

var (
	// ErrSyncConflict means local and remote changes touch the same lines; the
	// local state is kept and someone has to merge by hand.
	ErrSyncConflict = errors.New("the content repository and its remote changed the same lines")
	// ErrNoRemote means the data directory is not a git checkout with an upstream branch.
	ErrNoRemote = errors.New("the data directory has no remote to sync with")
	// ErrWorktreeBusy means the checkout is in the middle of something else — a
	// rebase or merge being finished by hand, a lock left by a crashed git, a
	// detached HEAD — and nothing is committed or synced until that is resolved.
	ErrWorktreeBusy = errors.New("the content repository is in the middle of another git operation")
	// ErrContentStale means the checked-out files could not be loaded (a JSON
	// typo pushed from elsewhere, say). The site keeps serving what it had in
	// memory, and saving is refused until a fixed version arrives: anything
	// written from the outdated memory would overwrite what came in.
	ErrContentStale = errors.New("the content repository has changes that could not be loaded")
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
	// loadedHead is the commit the content in memory was read from (guarded by gitMu).
	loadedHead string
	// stale is set while the checked-out content could not be loaded (guarded by mu).
	stale error
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

// ContentProblem says why saving is refused at the moment ("" when it is not).
func (r *FileRepository) ContentProblem() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.sync.stale == nil {
		return ""
	}
	return r.sync.stale.Error()
}

// lockWrite takes what a change to the content files needs: the working tree
// (no sync commits, rebases or reloads underneath a half-done save) and the
// data. It refuses while the content in memory is older than the checkout.
func (r *FileRepository) lockWrite() (unlock func(), err error) {
	r.wmu.Lock()
	r.mu.Lock()
	if r.sync.stale != nil {
		err = r.sync.stale
		r.mu.Unlock()
		r.wmu.Unlock()
		return nil, err
	}
	return func() {
		r.mu.Unlock()
		r.wmu.Unlock()
	}, nil
}

// git runs a git command in the data directory. It never waits for a
// password prompt (a missing credential fails right away), and when ctx ends
// git is asked to stop (SIGTERM, so it removes its lock files) before it is killed.
func (r *FileRepository) git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.dataDir}, args...)...)
	// GIT_OPTIONAL_LOCKS=0: "git status" does not take .git/index.lock just to refresh the index;
	// GIT_EDITOR=true: "rebase --continue" keeps the message instead of waiting for an editor
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_EDITOR=true")
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 10 * time.Second
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
	// ErrContentStale: the rebase went through, only loading the result failed
	if (err == nil || errors.Is(err, ErrContentStale)) && result.Pushed > 0 {
		if _, pushErr := r.pushLocked(ctx); pushErr != nil {
			err = errors.Join(err, pushErr)
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
// gitMu. reloaded says whether new content was loaded into memory; result is
// filled in as soon as the rebase is done, so it is valid with ErrContentStale.
func (r *FileRepository) syncLocked(ctx context.Context) (result SyncResult, reloaded bool, err error) {
	if err := r.checkWorktree(ctx); err != nil {
		return result, false, err
	}
	if _, err := r.git(ctx, "rev-parse", "--verify", "--quiet", "@{u}"); err != nil {
		return result, false, ErrNoRemote
	}
	if _, err := r.git(ctx, "fetch", "--quiet"); err != nil {
		return result, false, err
	}

	r.wmu.Lock()
	defer r.wmu.Unlock()
	pulled, err := r.integrateLocked(ctx, true)
	if err != nil {
		return result, false, err
	}
	r.commitLeftoversLocked(ctx)
	result.Pulled = pulled
	result.Pushed, _ = r.countCommits(ctx, "@{u}..HEAD")
	result.Head, _ = r.git(ctx, "rev-parse", "--short", "HEAD")
	reloaded, err = r.reloadIfMovedLocked(ctx)
	return result, reloaded, err
}

// integrateAtStartup is the first sync, before the content is loaded: it
// cleans up after an interrupted one, fetches, and replays the commits this
// server could not push before it stopped on top of what was pushed meanwhile.
func (r *FileRepository) integrateAtStartup(ctx context.Context) (pulled int, err error) {
	r.gitMu.Lock()
	defer r.gitMu.Unlock()
	r.recoverWorktree(ctx)
	if err := r.checkWorktree(ctx); err != nil {
		return 0, err
	}
	if _, err := r.git(ctx, "rev-parse", "--verify", "--quiet", "@{u}"); err != nil {
		return 0, ErrNoRemote
	}
	if _, err := r.git(ctx, "fetch", "--quiet"); err != nil {
		return 0, err
	}
	r.wmu.Lock()
	defer r.wmu.Unlock()
	if pulled, err = r.integrateLocked(ctx, false); err == nil {
		r.commitLeftoversLocked(ctx) // e.g. a save whose commit the last shutdown cut off
	}
	return pulled, err
}

// integrateLocked puts the server's own commits on top of the freshly fetched
// upstream. Its own commits are those after the fork point: the newest
// version of the upstream this branch was built on, as recorded in the
// upstream's reflog. A plain "rebase @{u}" starts from the merge base instead,
// and after a force push that replays what the author removed from the remote
// (a leaked secret, say) — even when the remote only went back and a plain
// push would bring the commit straight back. flush writes the in-memory
// counters first (not at startup, before they are loaded). pulled counts the
// commits that came in. The caller holds gitMu and wmu.
func (r *FileRepository) integrateLocked(ctx context.Context, flush bool) (pulled int, err error) {
	upstream, err := r.git(ctx, "rev-parse", "@{u}")
	if err != nil {
		return 0, ErrNoRemote
	}
	base, err := r.git(ctx, "merge-base", "--fork-point", "@{u}", "HEAD")
	if err != nil { // the reflog does not know (expired, or no fetch yet)
		if base, err = r.git(ctx, "merge-base", "@{u}", "HEAD"); err != nil {
			return 0, fmt.Errorf("the remote history shares no commit with the local one: %w", err)
		}
	}
	if base == upstream {
		return 0, nil // the branch already builds on the upstream
	}
	if _, err := r.git(ctx, "merge-base", "--is-ancestor", base, upstream); err != nil {
		slog.Warn("content repository: the remote history was rewritten (force push); only this server's own commits are replayed on top of it")
	}
	if pulled, err = r.countCommits(ctx, "HEAD.."+upstream); err != nil {
		return 0, err
	}
	// Everything local becomes a commit first, so the rebase replays it like any other local commit and a conflict
	// ends in a clean abort. (An autostash could not: a change the admin just saved, whose commit is still
	// queued, would come back as conflict markers in the working tree.)
	r.commitPendingLocked(ctx, flush)
	if err := r.rebaseLocked(ctx, upstream, base); err != nil {
		abortCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if _, abortErr := r.git(abortCtx, "rebase", "--abort"); abortErr != nil {
			slog.Error("content repository: git rebase --abort failed", "err", abortErr)
		}
		if ctx.Err() != nil { // interrupted, not a conflict
			return 0, fmt.Errorf("git rebase: %w", ctx.Err())
		}
		slog.Error("content repository: local and remote changes conflict, keeping the local state", "err", err)
		return 0, ErrSyncConflict
	}
	return pulled, nil
}

// counterFiles are written from memory, by this server only.
var counterFiles = map[string]bool{"views.json": true, "likes.json": true}

// rebaseLocked replays base..HEAD onto upstream. A conflict in the counters is
// resolved with the numbers in memory, which are the newest: such conflicts
// come from a force push that dropped the server's counter commits, or from a
// second server writing the same repository, and would otherwise stop every
// sync from then on. Any other conflict ends the rebase with an error (the
// caller aborts it). The caller holds gitMu and wmu.
func (r *FileRepository) rebaseLocked(ctx context.Context, upstream, base string) error {
	replayed, err := r.countCommits(ctx, base+"..HEAD")
	if err != nil {
		return err
	}
	_, err = r.git(ctx, "rebase", "--onto", upstream, base)
	for stops := 0; err != nil && stops < replayed && ctx.Err() == nil; stops++ {
		out, diffErr := r.git(ctx, "diff", "--name-only", "--diff-filter=U")
		if diffErr != nil || out == "" {
			return err
		}
		conflicted := strings.Split(out, "\n")
		for _, name := range conflicted {
			if !counterFiles[name] {
				return err
			}
		}
		for _, name := range conflicted {
			if writeErr := r.writeCounterLocked(name); writeErr != nil {
				return writeErr
			}
		}
		if _, addErr := r.git(ctx, append([]string{"add", "--"}, conflicted...)...); addErr != nil {
			return addErr
		}
		slog.Info("content repository: counters conflicted while rebasing, kept the numbers in memory", "files", conflicted)
		_, err = r.git(ctx, "rebase", "--continue")
	}
	return err
}

// writeCounterLocked writes views.json or likes.json from memory. The caller holds wmu.
func (r *FileRepository) writeCounterLocked(name string) error {
	r.mu.RLock()
	counts := maps.Clone(r.views)
	if name == "likes.json" {
		counts = maps.Clone(r.likes)
	}
	r.mu.RUnlock()
	if counts == nil {
		counts = map[string]int{}
	}
	return r.saveJSON(name, counts)
}

// reloadIfMovedLocked loads the content again when the checkout is not what
// memory was read from: after a pull, and also after a reload that failed or a
// pull done by hand. The caller holds gitMu and wmu.
func (r *FileRepository) reloadIfMovedLocked(ctx context.Context) (bool, error) {
	head, err := r.git(ctx, "rev-parse", "HEAD")
	if err != nil || head == r.sync.loadedHead {
		return false, err
	}
	if err := r.reloadContent(); err != nil {
		stale := fmt.Errorf("%w: %v", ErrContentStale, err)
		r.mu.Lock()
		r.sync.stale = stale
		r.mu.Unlock()
		slog.Error("content repository: the checked-out content could not be loaded; the site keeps the previous version and saving is refused until it is fixed", "err", err)
		return false, stale
	}
	r.sync.loadedHead = head
	return true, nil
}

// staleLockAge is how old a .git/index.lock must be before it is taken for
// the leftover of a git that was killed: no git command runs that long here.
const staleLockAge = 10 * time.Minute

// checkWorktree refuses a checkout that is in the middle of something else.
// Committing or rebasing over a merge someone is finishing by hand could
// publish conflict markers; a detached HEAD would collect commits no branch has.
func (r *FileRepository) checkWorktree(ctx context.Context) error {
	gitDir := filepath.Join(r.dataDir, ".git")
	r.removeStaleLock(staleLockAge)
	for _, marker := range []struct{ file, what string }{
		{"rebase-merge", "a rebase"}, {"rebase-apply", "a rebase"}, {"MERGE_HEAD", "a merge"},
		{"CHERRY_PICK_HEAD", "a cherry-pick"}, {"REVERT_HEAD", "a revert"}, {"index.lock", "another git command"},
	} {
		if _, err := os.Stat(filepath.Join(gitDir, marker.file)); err == nil {
			return fmt.Errorf("%w: %s is in progress (.git/%s)", ErrWorktreeBusy, marker.what, marker.file)
		}
	}
	if _, err := r.git(ctx, "symbolic-ref", "-q", "HEAD"); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 { // 1: not a symbolic ref; anything else is git failing
			return fmt.Errorf("%w: HEAD is detached, check out the branch again", ErrWorktreeBusy)
		}
		return err
	}
	out, err := r.git(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return err
	}
	if out != "" {
		return fmt.Errorf("%w: unmerged files: %s", ErrWorktreeBusy, strings.ReplaceAll(out, "\n", ", "))
	}
	return nil
}

// removeStaleLock removes a .git/index.lock that has not changed for age: the
// git that took it was killed (SIGKILL, power loss) and every commit would fail on it.
func (r *FileRepository) removeStaleLock(age time.Duration) {
	lock := filepath.Join(r.dataDir, ".git", "index.lock")
	info, err := os.Stat(lock)
	if err != nil || time.Since(info.ModTime()) < age {
		return
	}
	slog.Warn("content repository: removing the .git/index.lock an interrupted git left behind", "age", time.Since(info.ModTime()).Round(time.Second))
	if err := os.Remove(lock); err != nil && !os.IsNotExist(err) {
		slog.Error("content repository: could not remove .git/index.lock", "err", err)
	}
}

// recoverWorktree cleans up after a sync a crash or kill interrupted. It runs
// at startup only, before this server touches git: a rebase left behind is
// aborted (the branch returns to where it was, no commit is lost) and a lock
// file is removed — the process that took it is gone, unless it is a git
// someone runs by hand right now, which gets a few seconds to finish.
func (r *FileRepository) recoverWorktree(ctx context.Context) {
	gitDir := filepath.Join(r.dataDir, ".git")
	for wait := 0; wait < 50; wait++ {
		info, err := os.Stat(filepath.Join(gitDir, "index.lock"))
		if err != nil || time.Since(info.ModTime()) > 10*time.Second {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	r.removeStaleLock(0)
	for _, dir := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(gitDir, dir)); err == nil {
			slog.Warn("content repository: aborting the rebase an interrupted sync left behind")
			if _, err := r.git(ctx, "rebase", "--abort"); err != nil {
				slog.Error("content repository: git rebase --abort failed", "err", err)
			}
			return
		}
	}
}

// errBehindUpstream means the branch does not contain the upstream as last
// fetched: it needs a sync before it can be pushed.
var errBehindUpstream = errors.New("the branch is not based on the fetched upstream")

// pushLocked pushes the branch to its upstream, but only as a fast-forward of
// the upstream as this checkout last fetched it (--force-with-lease pins that
// value, the ancestry check rules out anything but a fast-forward). A remote
// that moved since — a post pushed from the laptop, or a force push that took
// a commit out — rejects the push instead of getting the commit back: a
// plain push would happily fast-forward a remote that was reset to an older
// commit. The caller holds gitMu; on a rejection it syncs and tries again.
func (r *FileRepository) pushLocked(ctx context.Context) (string, error) {
	if _, err := r.git(ctx, "rev-parse", "--verify", "--quiet", "@{u}"); err != nil {
		return "", ErrNoRemote
	}
	if _, err := r.git(ctx, "merge-base", "--is-ancestor", "@{u}", "HEAD"); err != nil {
		return "", errBehindUpstream
	}
	return r.git(ctx, "push", "--quiet", "--force-with-lease")
}

func (r *FileRepository) countCommits(ctx context.Context, revisions string) (int, error) {
	out, err := r.git(ctx, "rev-list", "--count", revisions)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

// commitLocked stages paths (everything when there are none) and commits them
// with message when anything changed. Memory stays in step with the checkout:
// if it was loaded from the commit this one builds on, it now matches the new
// one. The caller holds gitMu.
func (r *FileRepository) commitLocked(ctx context.Context, message string, paths ...string) (committed bool, err error) {
	before, _ := r.git(ctx, "rev-parse", "HEAD")
	add := []string{"add", "-A"}
	if len(paths) > 0 {
		add = append([]string{"add", "--"}, paths...)
	}
	if _, err := r.git(ctx, add...); err != nil {
		return false, err
	}
	if _, err := r.git(ctx, append([]string{"diff", "--cached", "--quiet", "--"}, paths...)...); err == nil {
		return false, nil // nothing new
	}
	commit := []string{"commit", "--quiet", "-m", message}
	if len(paths) > 0 {
		commit = append(append(commit, "--"), paths...)
	}
	if _, err := r.git(ctx, commit...); err != nil {
		return false, err
	}
	if after, err := r.git(ctx, "rev-parse", "HEAD"); err == nil && before == r.sync.loadedHead {
		r.sync.loadedHead = after
	}
	return true, nil
}

// commitPendingLocked commits what is not committed yet: the view and like
// counters in their usual commit (written from memory first when flush is set
// — not at startup, before they are loaded), then anything else, e.g. an admin
// save whose own commit is still queued. The caller holds gitMu and wmu.
func (r *FileRepository) commitPendingLocked(ctx context.Context, flush bool) {
	if flush {
		if err := r.flushViewsLocked(); err != nil {
			slog.Error("flush views failed", "err", err)
		}
		if err := r.flushLikesLocked(); err != nil {
			slog.Error("flush likes failed", "err", err)
		}
	}
	if _, err := os.Stat(filepath.Join(r.dataDir, "views.json")); err == nil {
		paths := []string{"views.json"}
		if _, err := os.Stat(filepath.Join(r.dataDir, "likes.json")); err == nil {
			paths = append(paths, "likes.json")
		}
		if _, err := r.commitLocked(ctx, "chore: update views.json", paths...); err != nil {
			slog.Error("git commit counters failed", "err", err)
		}
	}
	if out, err := r.git(ctx, "status", "--porcelain"); err != nil || out == "" {
		return
	}
	if _, err := r.commitLocked(ctx, "chore: commit local changes before syncing"); err != nil {
		slog.Error("git commit failed", "err", err)
	}
}

// commitLeftoversLocked commits changes an earlier commit did not pick up (it
// was skipped while the checkout was busy, say), so a sync always pushes
// everything the admin saved. The counters alone are no reason for a commit:
// they have their hourly one. The caller holds gitMu and wmu.
func (r *FileRepository) commitLeftoversLocked(ctx context.Context) {
	out, err := r.git(ctx, "status", "--porcelain", "--", ".", ":(exclude)views.json", ":(exclude)likes.json")
	if err != nil || out == "" {
		return
	}
	if _, err := r.commitLocked(ctx, "chore: commit local changes"); err != nil {
		slog.Error("git commit failed", "err", err)
	}
}

// reloadContent reads posts, pages, tags, categories, books, projects and
// users from disk again. Everything is loaded into a scratch repository first
// and only swapped in when all of it parsed, so a broken file pushed from
// elsewhere never leaves the site half loaded. View and like counters are not
// read: the ones in memory are newer. The caller holds wmu, so no save can
// land between reading the files and swapping them in.
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
	r.sync.stale = nil
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
			case errors.Is(err, ErrWorktreeBusy), errors.Is(err, ErrContentStale), errors.Is(err, ErrSyncConflict):
				slog.Error("content repository sync needs attention", "err", err)
			case err != nil:
				slog.Warn("content repository sync failed", "err", err)
			case result.Pulled > 0:
				slog.Info("content repository synced", "pulled", result.Pulled, "pushed", result.Pushed, "head", result.Head)
			}
		}
	}()
}
