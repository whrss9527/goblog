package routers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goblog/internal/config"
)

// laptop is a clone of the content repository somewhere else: the author's
// laptop, pushing to the same "GitHub" remote as the server.
type laptop struct {
	t   *testing.T
	dir string
}

func (l *laptop) git(args ...string) {
	l.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = l.dir
	if out, err := cmd.CombinedOutput(); err != nil {
		l.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// push writes files (relative to the repository), commits and pushes them.
func (l *laptop) push(message string, files map[string]string) {
	l.t.Helper()
	l.git("pull", "-q", "--rebase")
	for name, content := range files {
		path := filepath.Join(l.dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			l.t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			l.t.Fatal(err)
		}
	}
	l.git("add", "-A")
	l.git("commit", "-q", "-m", message)
	l.git("push", "-q")
}

func (l *laptop) pushPost(slug, title string) {
	l.t.Helper()
	content := fmt.Sprintf("---\ntitle: %q\nstatus: 1\ncreated_at: %s\nupdated_at: %[2]s\ncategory_id: 1\nis_top: 0\ntag_ids: [1]\ndescription: \"pushed\"\nword_count: 3\n---\n\n写在笔记本上。\n",
		title, time.Now().Format(time.RFC3339))
	l.push("post: "+slug, map[string]string{"posts/" + slug + ".md": content})
}

// gitBacked turns the test data directory into a clone of a bare "GitHub"
// repository and sets author to a laptop that pushes to that remote.
func gitBacked(t *testing.T, author **laptop) func(*config.Config) {
	return func(c *config.Config) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not installed")
		}
		t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
		t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
		run := func(dir string, args ...string) {
			t.Helper()
			cmd := exec.Command("git", args...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
		}
		root := filepath.Dir(c.App.DataDir)
		remote := filepath.Join(root, "remote.git")
		data := c.App.DataDir
		run(root, "init", "-q", "--bare", "-b", "main", remote)
		run(data, "init", "-q", "-b", "main")
		run(data, "config", "user.name", "server")
		run(data, "config", "user.email", "server@example.com")
		run(data, "add", "-A")
		run(data, "commit", "-q", "-m", "initial content")
		run(data, "remote", "add", "origin", remote)
		run(data, "push", "-q", "-u", "origin", "main")
		l := &laptop{t: t, dir: filepath.Join(root, "laptop")}
		run(root, "clone", "-q", remote, l.dir)
		l.git("config", "user.name", "laptop")
		l.git("config", "user.email", "laptop@example.com")
		*author = l
	}
}

func signedHook(h http.Handler, secret, event, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/hooks/git", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", event)
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(body))
		req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGitWebhookSyncsRightAway(t *testing.T) {
	var author *laptop
	h := newTestServer(t, gitBacked(t, &author), func(c *config.Config) {
		c.App.GitWebhookSecret = "hook-secret"
		c.App.GitSync = "off" // only the webhook may bring the post in
	})

	if rec := signedHook(h, "", "push", `{}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("unsigned hook = %d, want 401", rec.Code)
	}
	if rec := signedHook(h, "wrong-secret", "push", `{}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("hook signed with another secret = %d, want 401", rec.Code)
	}
	if rec := signedHook(h, "hook-secret", "ping", `{"zen":"hi"}`); rec.Code != http.StatusOK {
		t.Errorf("ping = %d, want 200", rec.Code)
	}

	author.pushPost("from-laptop", "笔记本上写的")
	if rec := signedHook(h, "hook-secret", "push", `{"ref":"refs/heads/main"}`); rec.Code != http.StatusAccepted {
		t.Fatalf("push hook = %d, want 202", rec.Code)
	}
	deadline := time.Now().Add(20 * time.Second)
	for get(t, h, "/posts/from-laptop").Code != http.StatusOK {
		if time.Now().After(deadline) {
			t.Fatal("the pushed post never showed up")
		}
		time.Sleep(50 * time.Millisecond)
	}
	// derived documents are rebuilt right after the reload
	for target, want := range map[string]string{"/feed.xml": "笔记本上写的", "/sitemap.xml": "/posts/from-laptop"} {
		for !strings.Contains(get(t, h, target).Body.String(), want) {
			if time.Now().After(deadline) {
				t.Fatalf("%s never listed the pulled post", target)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	if home := get(t, h, "/").Body.String(); !strings.Contains(home, "笔记本上写的") {
		t.Errorf("the home page lists the pulled post")
	}
}

func TestGitWebhookNeedsASecret(t *testing.T) {
	h := newTestServer(t)
	if rec := signedHook(h, "", "push", `{}`); rec.Code != http.StatusNotFound {
		t.Errorf("without git_webhook_secret the hook does not exist, got %d", rec.Code)
	}
}

func TestAdminSyncButton(t *testing.T) {
	var author *laptop
	a := newAdminClient(t, gitBacked(t, &author), func(c *config.Config) { c.App.GitSync = "off" })

	_, body, _ := a.get("/admin/")
	if !strings.Contains(body, `action="/admin/sync"`) {
		t.Fatalf("a git-backed blog offers the sync button")
	}

	author.pushPost("synced", "同步进来的文章")
	status, _, location := a.post("/admin/sync", url.Values{"_csrf": {a.token("/admin/")}})
	if status != http.StatusFound || !strings.HasPrefix(location, "/admin?") || !strings.Contains(location, "sync=ok") || !strings.Contains(location, "pulled=1") {
		t.Fatalf("sync = %d -> %q", status, location)
	}
	list := func(location string) string { // the browser follows /admin -> /admin/ with the query
		_, body, _ := a.get(strings.Replace(location, "/admin?", "/admin/?", 1))
		return body
	}
	if body = list(location); !strings.Contains(body, "已同步：拉取了 1 个新提交") || !strings.Contains(body, "同步进来的文章") {
		t.Errorf("the list reports the sync and shows the new post")
	}

	status, _, location = a.post("/admin/sync", url.Values{"_csrf": {a.token("/admin/")}})
	if status != http.StatusFound || !strings.Contains(location, "pulled=0") {
		t.Errorf("second sync = %d -> %q", status, location)
	}
	if !strings.Contains(list(location), "内容仓库已经是最新的") {
		t.Errorf("an idle sync says so")
	}
	if status, _, _ = a.post("/admin/sync", url.Values{}); status != http.StatusForbidden {
		t.Errorf("sync without CSRF token = %d, want 403", status)
	}
}

func TestAdminSyncButtonHiddenWithoutGit(t *testing.T) {
	a := newAdminClient(t)
	if _, body, _ := a.get("/admin/"); strings.Contains(body, `action="/admin/sync"`) {
		t.Errorf("a data directory without git has nothing to sync")
	}
}

// Content pushed from elsewhere that cannot be loaded keeps the site on the
// previous version and stops the admin from saving over it, visibly.
func TestAdminWhileContentIsStale(t *testing.T) {
	var author *laptop
	a := newAdminClient(t, gitBacked(t, &author), func(c *config.Config) { c.App.GitSync = "off" })
	author.push("broken categories", map[string]string{"categories.json": `[{"id":1,"name":"技术"`})

	status, _, location := a.post("/admin/sync", url.Values{"_csrf": {a.token("/admin/")}})
	if status != http.StatusFound || !strings.Contains(location, "sync=stale") {
		t.Fatalf("sync = %d -> %q", status, location)
	}
	_, body, _ := a.get("/admin/?sync=stale")
	for _, want := range []string{"拉取到了新内容，但没能加载", "后台暂时不能保存：内容仓库里有别处推送来的改动没能加载", "could not be loaded"} {
		if !strings.Contains(body, want) {
			t.Errorf("the post list shows %q", want)
		}
	}
	if _, body, _ = a.get("/admin/tags"); !strings.Contains(body, "admin-content-problem") {
		t.Errorf("every admin list page carries the banner")
	}

	status, body, _ = a.post("/admin/categories/save", url.Values{"_csrf": {a.token("/admin/categories/add")}, "name": {"新分类"}})
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, "后台暂时不能保存") {
		t.Errorf("saving a category = %d, want 422 with the reason", status)
	}
	form := url.Values{"_csrf": {a.token("/admin/posts/add")}, "title": {"写不进去"}, "identity": {"stale-save"}, "content": {"正文"}, "category": {"1"}}
	status, body, _ = a.post("/admin/posts/save", form)
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, "后台暂时不能保存") || !strings.Contains(body, "写不进去") {
		t.Errorf("saving a post = %d, want the editor again with the reason and the text", status)
	}

	author.push("fix categories", map[string]string{"categories.json": `[{"id":1,"name":"技术"}]`})
	status, _, location = a.post("/admin/sync", url.Values{"_csrf": {a.token("/admin/")}})
	if status != http.StatusFound || !strings.Contains(location, "sync=ok") {
		t.Fatalf("sync after the fix = %d -> %q", status, location)
	}
	if _, body, _ = a.get("/admin/"); strings.Contains(body, "admin-content-problem") {
		t.Errorf("the banner goes away once the content loads")
	}
	status, _, _ = a.post("/admin/categories/save", url.Values{"_csrf": {a.token("/admin/categories/add")}, "name": {"新分类"}})
	if status != http.StatusFound {
		t.Errorf("saving works again, got %d", status)
	}
}
