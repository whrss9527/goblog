// Package github reads public repository information from the GitHub REST API:
// what the admin imports when adding a project, the owner's recent repositories
// offered as suggestions, and the stars / language / last push shown on
// /projects. Everything here is read-only and works without a token (60
// requests an hour per IP); a token raises that limit.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"goblog/internal/version"
)

// DefaultAPI is the address of the public GitHub REST API.
const DefaultAPI = "https://api.github.com"

// maxBody caps what is read from a response; a repository document is a few KB.
const maxBody = 4 << 20

var (
	// ErrNotFound is returned for a repository or user that does not exist or is
	// not visible (private repositories look the same without a token).
	ErrNotFound = errors.New("github: not found")
	// ErrRateLimited matches a *RateLimitError.
	ErrRateLimited = errors.New("github: API rate limit exceeded")
)

// RateLimitError says the API refuses further requests until Reset.
type RateLimitError struct {
	Reset time.Time
}

func (e *RateLimitError) Error() string {
	if e.Reset.IsZero() {
		return ErrRateLimited.Error()
	}
	return fmt.Sprintf("%s (resets at %s)", ErrRateLimited, e.Reset.Format(time.RFC3339))
}

func (e *RateLimitError) Is(target error) bool { return target == ErrRateLimited }

// Repo is the part of a GitHub repository document goblog uses.
type Repo struct {
	FullName    string    `json:"full_name"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	HTMLURL     string    `json:"html_url"`
	Homepage    string    `json:"homepage"`
	Language    string    `json:"language"`
	Topics      []string  `json:"topics"`
	Stars       int       `json:"stargazers_count"`
	Forks       int       `json:"forks_count"`
	Archived    bool      `json:"archived"`
	Fork        bool      `json:"fork"`
	Private     bool      `json:"private"`
	CreatedAt   time.Time `json:"created_at"`
	PushedAt    time.Time `json:"pushed_at"`
}

// Client talks to the GitHub REST API.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// NewClient returns a client for the API at base (DefaultAPI when empty). The
// token is optional.
func NewClient(base, token string) *Client {
	if base == "" {
		base = DefaultAPI
	}
	return &Client{
		base:  strings.TrimRight(base, "/"),
		token: strings.TrimSpace(token),
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Repo fetches one repository.
func (c *Client) Repo(ctx context.Context, owner, name string) (Repo, error) {
	var repo Repo
	_, _, err := c.get(ctx, "/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), "", &repo)
	return repo, err
}

// repoIfChanged fetches a repository unless it still matches etag. GitHub does
// not count such "304 Not Modified" answers against the rate limit.
func (c *Client) repoIfChanged(ctx context.Context, owner, name, etag string) (repo Repo, newETag string, changed bool, err error) {
	newETag, notModified, err := c.get(ctx, "/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), etag, &repo)
	return repo, newETag, !notModified, err
}

// UserRepos lists the public repositories a user owns, most recently pushed
// first, leaving out forks and archived ones. At most limit are returned.
func (c *Client) UserRepos(ctx context.Context, user string, limit int) ([]Repo, error) {
	var all []Repo
	path := "/users/" + url.PathEscape(user) + "/repos?type=owner&sort=pushed&direction=desc&per_page=100"
	if _, _, err := c.get(ctx, path, "", &all); err != nil {
		return nil, err
	}
	repos := make([]Repo, 0, limit)
	for _, r := range all {
		if r.Fork || r.Archived || r.Private {
			continue
		}
		if len(repos) == limit {
			break
		}
		repos = append(repos, r)
	}
	return repos, nil
}

func (c *Client) get(ctx context.Context, path, etag string, v any) (newETag string, notModified bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "goblog/"+version.Version)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return "", false, err
	}
	defer res.Body.Close()
	body := io.LimitReader(res.Body, maxBody)

	switch {
	case res.StatusCode == http.StatusNotModified:
		return etag, true, nil
	case res.StatusCode == http.StatusOK:
		if err := json.NewDecoder(body).Decode(v); err != nil {
			return "", false, fmt.Errorf("github: decode %s: %w", path, err)
		}
		return res.Header.Get("ETag"), false, nil
	case res.StatusCode == http.StatusNotFound:
		return "", false, ErrNotFound
	case res.StatusCode == http.StatusTooManyRequests,
		res.StatusCode == http.StatusForbidden && (res.Header.Get("X-RateLimit-Remaining") == "0" || res.Header.Get("Retry-After") != ""):
		return "", false, &RateLimitError{Reset: resetTime(res.Header, time.Now())}
	}
	io.Copy(io.Discard, body)
	return "", false, fmt.Errorf("github: GET %s: %s", path, res.Status)
}

// resetTime reads when a rate limit ends: Retry-After (seconds) for secondary
// limits, X-RateLimit-Reset (epoch seconds) for the hourly one.
func resetTime(h http.Header, now time.Time) time.Time {
	if s, err := strconv.Atoi(h.Get("Retry-After")); err == nil && s > 0 {
		return now.Add(time.Duration(s) * time.Second)
	}
	if epoch, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64); err == nil && epoch > 0 {
		return time.Unix(epoch, 0)
	}
	return now.Add(time.Hour)
}

var (
	ownerPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)
	namePattern  = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
)

// ParseRepo recognises what people paste for a GitHub repository:
// "https://github.com/owner/name" (also with www., .git, a trailing slash or
// a deeper path such as /tree/main), "github.com/owner/name",
// "git@github.com:owner/name.git" and plain "owner/name".
func ParseRepo(s string) (owner, name string, ok bool) {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "git@github.com:"):
		s = s[len("git@github.com:"):]
	case strings.Contains(lower, "://"):
		u, err := url.Parse(s)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
			return "", "", false
		}
		if host := strings.ToLower(u.Hostname()); host != "github.com" && host != "www.github.com" {
			return "", "", false
		}
		s = u.Path
	case strings.HasPrefix(lower, "github.com/"), strings.HasPrefix(lower, "www.github.com/"):
		s = s[strings.Index(s, "/")+1:]
	case strings.Count(s, "/") != 1:
		return "", "", false // a bare reference has to be exactly owner/name
	}
	parts := strings.Split(strings.Trim(s, "/"), "/")
	if len(parts) < 2 {
		return "", "", false
	}
	owner, name = parts[0], strings.TrimSuffix(parts[1], ".git")
	if !ownerPattern.MatchString(owner) || !namePattern.MatchString(name) || name == "." || name == ".." {
		return "", "", false
	}
	return owner, name, true
}

// Key is the case-insensitive identity of a repository ("owner/name").
func Key(owner, name string) string {
	return strings.ToLower(owner + "/" + name)
}

// RepoURL is the canonical web address of a repository.
func RepoURL(owner, name string) string {
	return "https://github.com/" + owner + "/" + name
}

// OwnerOf returns the account of a GitHub repository address, "" for anything else.
func OwnerOf(repoURL string) string {
	owner, _, ok := ParseRepo(repoURL)
	if !ok || !strings.Contains(repoURL, "github.com") {
		return ""
	}
	return owner
}
