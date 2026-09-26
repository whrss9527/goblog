package github

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRepo(t *testing.T) {
	tests := []struct {
		in          string
		owner, name string
		ok          bool
	}{
		{"https://github.com/whrss9527/goblog", "whrss9527", "goblog", true},
		{"https://github.com/whrss9527/goblog/", "whrss9527", "goblog", true},
		{"https://github.com/whrss9527/goblog.git", "whrss9527", "goblog", true},
		{"https://www.github.com/whrss9527/goblog/tree/main/internal", "whrss9527", "goblog", true},
		{"http://github.com/whrss9527/proxyswitch-mac", "whrss9527", "proxyswitch-mac", true},
		{"  github.com/whrss9527/goblog  ", "whrss9527", "goblog", true},
		{"git@github.com:whrss9527/blog-data.git", "whrss9527", "blog-data", true},
		{"whrss9527/My-illness-record", "whrss9527", "My-illness-record", true},
		{"whrss9527/draw.io", "whrss9527", "draw.io", true},
		{"https://gitee.com/whrss9527/goblog", "", "", false},
		{"https://github.com/whrss9527", "", "", false},
		{"https://github.com.evil.example/a/b", "", "", false},
		{"ftp://github.com/a/b", "", "", false},
		{"javascript:alert(1)//github.com/a/b", "", "", false},
		{"a/b/c", "", "", false},
		{"goblog", "", "", false},
		{"", "", "", false},
		{"https://github.com/-bad/name", "", "", false},
		{"https://github.com/owner/..", "", "", false},
	}
	for _, tt := range tests {
		owner, name, ok := ParseRepo(tt.in)
		assert.Equal(t, tt.ok, ok, tt.in)
		assert.Equal(t, tt.owner, owner, tt.in)
		assert.Equal(t, tt.name, name, tt.in)
	}
	assert.Equal(t, "whrss9527", OwnerOf("https://github.com/whrss9527/blog-data.git"))
	assert.Equal(t, "", OwnerOf("https://gitlab.com/whrss9527/blog-data.git"))
	assert.Equal(t, "", OwnerOf("whrss9527/blog-data"), "a bare owner/name is not an address")
	assert.Equal(t, "#00add8", LanguageColor(" Go "))
	assert.Equal(t, "", LanguageColor("Brainfuck"))
}

func TestClientRepo(t *testing.T) {
	var seen http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		switch r.URL.Path {
		case "/repos/whrss9527/goblog":
			w.Header().Set("ETag", `"abc"`)
			w.Write([]byte(`{"full_name":"whrss9527/goblog","name":"goblog","description":"Go 博客","html_url":"https://github.com/whrss9527/goblog",
				"homepage":"https://whrss.com","language":"Go","topics":["blog","golang"],"stargazers_count":12,"forks_count":3,
				"created_at":"2022-11-20T10:00:00Z","pushed_at":"2026-09-19T12:19:32Z"}`))
		case "/repos/whrss9527/limited":
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "1900000000")
			w.WriteHeader(http.StatusForbidden)
		case "/repos/whrss9527/slow-down":
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(http.StatusTooManyRequests)
		case "/repos/whrss9527/forbidden":
			w.WriteHeader(http.StatusForbidden)
		case "/repos/whrss9527/broken":
			w.WriteHeader(http.StatusBadGateway)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	ctx := context.Background()

	c := NewClient(server.URL+"/", " secret-token ")
	repo, err := c.Repo(ctx, "whrss9527", "goblog")
	require.NoError(t, err)
	assert.Equal(t, "Go 博客", repo.Description)
	assert.Equal(t, 12, repo.Stars)
	assert.Equal(t, []string{"blog", "golang"}, repo.Topics)
	assert.Equal(t, 2026, repo.PushedAt.Year())
	assert.Equal(t, "Bearer secret-token", seen.Get("Authorization"))
	assert.Equal(t, "application/vnd.github+json", seen.Get("Accept"))
	assert.True(t, strings.HasPrefix(seen.Get("User-Agent"), "goblog/"), "GitHub rejects requests without a User-Agent")

	_, err = NewClient(server.URL, "").Repo(ctx, "whrss9527", "goblog")
	require.NoError(t, err)
	assert.Empty(t, seen.Get("Authorization"), "no token, no Authorization header")

	_, err = c.Repo(ctx, "whrss9527", "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = c.Repo(ctx, "whrss9527", "limited")
	var limited *RateLimitError
	require.ErrorAs(t, err, &limited)
	assert.ErrorIs(t, err, ErrRateLimited)
	assert.Equal(t, int64(1900000000), limited.Reset.Unix())

	_, err = c.Repo(ctx, "whrss9527", "slow-down")
	require.ErrorAs(t, err, &limited)
	assert.WithinDuration(t, time.Now().Add(30*time.Second), limited.Reset, 5*time.Second)

	_, err = c.Repo(ctx, "whrss9527", "forbidden")
	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrRateLimited), "a plain 403 is not a rate limit")

	_, err = c.Repo(ctx, "whrss9527", "broken")
	assert.ErrorContains(t, err, "502")
}

func TestClientUserRepos(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/whrss9527/repos" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		query = r.URL.RawQuery
		json.NewEncoder(w).Encode([]Repo{
			{Name: "proxyswitch-mac"},
			{Name: "GitHubPoster", Fork: true},
			{Name: "old", Archived: true},
			{Name: "secret", Private: true},
			{Name: "proxyswitch"},
			{Name: "goblog"},
		})
	}))
	defer server.Close()

	repos, err := NewClient(server.URL, "").UserRepos(context.Background(), "whrss9527", 2)
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "proxyswitch-mac", repos[0].Name)
	assert.Equal(t, "proxyswitch", repos[1].Name)
	assert.Contains(t, query, "sort=pushed")
	assert.Contains(t, query, "type=owner")

	_, err = NewClient(server.URL, "").UserRepos(context.Background(), "nobody", 5)
	assert.ErrorIs(t, err, ErrNotFound)
}

// fakeAPI serves repositories from a map and records the requests it gets.
type fakeAPI struct {
	mu       sync.Mutex
	repos    map[string]string // path -> JSON document
	requests []string
	limited  bool
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.URL.Path+" "+r.Header.Get("If-None-Match"))
	if f.limited {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "4000000000")
		w.WriteHeader(http.StatusForbidden)
		return
	}
	doc, ok := f.repos[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	etag := fmt.Sprintf(`"%x"`, sha1.Sum([]byte(doc)))
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)
	w.Write([]byte(doc))
}

func (f *fakeAPI) take() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.requests
	f.requests = nil
	return out
}

func TestStatsRefresh(t *testing.T) {
	api := &fakeAPI{repos: map[string]string{
		"/repos/whrss9527/goblog":      `{"name":"goblog","stargazers_count":12,"language":"Go"}`,
		"/repos/whrss9527/proxyswitch": `{"name":"proxyswitch","stargazers_count":5,"language":"Swift"}`,
	}}
	server := httptest.NewServer(api)
	defer server.Close()
	ctx := context.Background()
	stats := NewStats(NewClient(server.URL, ""))

	urls := []string{
		"https://github.com/whrss9527/goblog",
		"https://github.com/whrss9527/proxyswitch/",
		"https://github.com/whrss9527/GOBLOG", // the same repository, fetched once
		"https://github.com/whrss9527/gone",
		"https://gitee.com/whrss9527/elsewhere",
	}
	stats.Refresh(ctx, urls, true)
	assert.Len(t, api.take(), 3, "one request per distinct GitHub repository")

	repo, ok := stats.Lookup("https://github.com/WHRSS9527/goblog.git")
	require.True(t, ok)
	assert.Equal(t, 12, repo.Stars)
	_, ok = stats.Lookup("https://github.com/whrss9527/gone")
	assert.False(t, ok, "a 404 is remembered as missing")
	assert.True(t, stats.Missing("https://github.com/whrss9527/gone"))
	assert.False(t, stats.Missing("https://github.com/whrss9527/goblog"))
	assert.False(t, stats.Missing("https://github.com/whrss9527/never-asked"))
	_, ok = stats.Lookup("https://gitee.com/whrss9527/elsewhere")
	assert.False(t, ok)

	stats.Refresh(ctx, urls, false)
	assert.Empty(t, api.take(), "an early round skips what was just checked")

	api.mu.Lock()
	api.repos["/repos/whrss9527/goblog"] = `{"name":"goblog","stargazers_count":13,"language":"Go"}`
	api.mu.Unlock()
	stats.Refresh(ctx, urls, true)
	conditional := 0
	for _, request := range api.take() {
		if strings.Contains(request, ` "`) {
			conditional++
		}
	}
	assert.Equal(t, 2, conditional, "repositories known from before are asked with If-None-Match")
	repo, _ = stats.Lookup("https://github.com/whrss9527/goblog")
	assert.Equal(t, 13, repo.Stars, "changed documents replace the cached ones")
	repo, ok = stats.Lookup("https://github.com/whrss9527/proxyswitch")
	assert.True(t, ok, "a 304 keeps the cached document")
	assert.Equal(t, 5, repo.Stars)

	// repositories no longer listed are forgotten
	stats.Refresh(ctx, urls[:1], true)
	api.take()
	_, ok = stats.Lookup("https://github.com/whrss9527/proxyswitch")
	assert.False(t, ok)

	// a used-up rate limit stops the round and pauses requests until it resets
	api.mu.Lock()
	api.limited = true
	api.mu.Unlock()
	stats.Refresh(ctx, urls, true)
	assert.Len(t, api.take(), 1, "the first refusal ends the round")
	stats.Refresh(ctx, urls, true)
	assert.Empty(t, api.take(), "no requests while the limit lasts")
	repo, ok = stats.Lookup("https://github.com/whrss9527/goblog")
	assert.True(t, ok, "the numbers already known stay")
	assert.Equal(t, 13, repo.Stars)

	var nilStats *Stats
	_, ok = nilStats.Lookup("https://github.com/whrss9527/goblog")
	assert.False(t, ok, "a nil cache (stats switched off) knows nothing")
	assert.False(t, nilStats.Missing("https://github.com/whrss9527/gone"))
	nilStats.Kick()
}

func TestStatsRunAndKick(t *testing.T) {
	api := &fakeAPI{repos: map[string]string{
		"/repos/a/one": `{"name":"one","stargazers_count":1}`,
		"/repos/a/two": `{"name":"two","stargazers_count":2}`,
	}}
	server := httptest.NewServer(api)
	defer server.Close()
	stats := NewStats(NewClient(server.URL, ""))

	var mu sync.Mutex
	list := []string{"https://github.com/a/one"}
	done := make(chan struct{})
	defer close(done)
	stats.Run(done, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), list...)
	}, time.Hour)

	waitFor := func(url string) bool {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if _, ok := stats.Lookup(url); ok {
				return true
			}
			time.Sleep(10 * time.Millisecond)
		}
		return false
	}
	require.True(t, waitFor("https://github.com/a/one"), "Run refreshes right away")

	mu.Lock()
	list = append(list, "https://github.com/a/two")
	mu.Unlock()
	stats.Kick()
	require.True(t, waitFor("https://github.com/a/two"), "Kick triggers an early round")
}
