package github

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// recheckAfter is how long a repository counts as fresh: an early round (after
// the admin saved a project) only fetches what is new or older than this.
const recheckAfter = 10 * time.Minute

// Stats keeps the live numbers of the projects' repositories in memory. Pages
// only ever read the cache, so a slow or unreachable GitHub never delays them;
// a failed refresh keeps the previous numbers.
type Stats struct {
	client *Client

	mu           sync.RWMutex
	entries      map[string]*statsEntry
	blockedUntil time.Time

	refreshMu sync.Mutex // one round at a time
	kick      chan struct{}
}

type statsEntry struct {
	repo    Repo
	etag    string
	checked time.Time
	missing bool // the repository is gone or private
}

// NewStats returns an empty cache that fetches through client.
func NewStats(client *Client) *Stats {
	return &Stats{client: client, entries: make(map[string]*statsEntry), kick: make(chan struct{}, 1)}
}

// Lookup returns what is known about the GitHub repository at repoURL.
func (s *Stats) Lookup(repoURL string) (Repo, bool) {
	if s == nil {
		return Repo{}, false
	}
	owner, name, ok := ParseRepo(repoURL)
	if !ok {
		return Repo{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[Key(owner, name)]
	if !ok || e.missing || e.checked.IsZero() {
		return Repo{}, false
	}
	return e.repo, true
}

// Missing reports whether GitHub answered "not found" for the repository at
// repoURL the last time it was asked: deleted, renamed away or private.
func (s *Stats) Missing(repoURL string) bool {
	if s == nil {
		return false
	}
	owner, name, ok := ParseRepo(repoURL)
	if !ok {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[Key(owner, name)]
	return ok && e.missing
}

// Refresh brings the repositories behind repoURLs up to date, one request after
// another. Unless all is set, repositories checked within the last few minutes
// are skipped. It stops early when GitHub says the rate limit is used up and
// makes no requests until the limit resets. Repositories that are no longer in
// the list are forgotten.
func (s *Stats) Refresh(ctx context.Context, repoURLs []string, all bool) {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	wanted := make(map[string][2]string)
	for _, u := range repoURLs {
		owner, name, ok := ParseRepo(u)
		if _, seen := wanted[Key(owner, name)]; ok && !seen { // the first spelling wins
			wanted[Key(owner, name)] = [2]string{owner, name}
		}
	}

	s.mu.Lock()
	for key := range s.entries {
		if _, ok := wanted[key]; !ok {
			delete(s.entries, key)
		}
	}
	blocked := time.Now().Before(s.blockedUntil)
	s.mu.Unlock()
	if blocked {
		return
	}

	var failures int
	var firstErr error
	for key, ref := range wanted {
		if ctx.Err() != nil {
			return
		}
		s.mu.RLock()
		previous := s.entries[key]
		s.mu.RUnlock()
		var etag string
		if previous != nil {
			if !all && time.Since(previous.checked) < recheckAfter {
				continue
			}
			etag = previous.etag
		}

		repo, newETag, changed, err := s.client.repoIfChanged(ctx, ref[0], ref[1], etag)
		var limited *RateLimitError
		switch {
		case errors.As(err, &limited):
			s.mu.Lock()
			s.blockedUntil = limited.Reset
			s.mu.Unlock()
			slog.Warn("github rate limit reached, project stats paused", "until", limited.Reset.Format(time.RFC3339))
			return
		case errors.Is(err, ErrNotFound):
			s.store(key, &statsEntry{missing: true, checked: time.Now()})
		case err != nil:
			failures++
			if firstErr == nil {
				firstErr = err
			}
		case changed:
			s.store(key, &statsEntry{repo: repo, etag: newETag, checked: time.Now()})
		default:
			s.store(key, &statsEntry{repo: previous.repo, etag: previous.etag, checked: time.Now()})
		}
	}
	if failures > 0 {
		slog.Warn("github project stats refresh failed", "failed", failures, "err", firstErr)
	}
}

func (s *Stats) store(key string, e *statsEntry) {
	s.mu.Lock()
	s.entries[key] = e
	s.mu.Unlock()
}

// Kick asks Run for an early round, e.g. after a project was added.
func (s *Stats) Kick() {
	if s == nil {
		return
	}
	select {
	case s.kick <- struct{}{}:
	default: // a round is already pending
	}
}

// Run refreshes the repositories named by list() right away, then every
// interval and whenever Kick is called, until done is closed.
func (s *Stats) Run(done <-chan struct{}, list func() []string, interval time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-done
		cancel()
	}()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		s.Refresh(ctx, list(), true)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.Refresh(ctx, list(), true)
			case <-s.kick:
				s.Refresh(ctx, list(), false)
			}
		}
	}()
}
