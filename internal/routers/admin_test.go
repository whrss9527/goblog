package routers

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"goblog/internal/config"
)

const (
	adminEmail    = "admin@example.com"
	adminPassword = "correct horse battery staple"
)

var csrfField = regexp.MustCompile(`name="_csrf" value="([^"]+)"`)

// adminClient is a logged-in browser-like client for the test server.
type adminClient struct {
	t       *testing.T
	base    string
	client  *http.Client
	dataDir string
}

func newAdminClient(t *testing.T) *adminClient {
	t.Helper()
	var dataDir string
	handler := newTestServer(t, func(c *config.Config) {
		dataDir = c.App.DataDir
		c.App.Host = "http://blog.example.com" // an https host marks the session cookie Secure, the test server speaks http
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		users := `[{"id":1,"email":"` + adminEmail + `","password":"` + string(hash) + `"}]`
		if err := os.WriteFile(filepath.Join(dataDir, "users.json"), []byte(users), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	jar, _ := cookiejar.New(nil)
	a := &adminClient{t: t, base: server.URL, dataDir: dataDir, client: &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}

	if status, _, location := a.get("/admin/"); status != http.StatusFound || location != "/admin/login" {
		t.Fatalf("anonymous /admin/ = %d -> %q, want a redirect to the login page", status, location)
	}
	status, _, location := a.post("/admin/sign-in", url.Values{"email": {adminEmail}, "password": {adminPassword}})
	if status != http.StatusFound || location != "/admin" {
		t.Fatalf("sign-in = %d -> %q", status, location)
	}
	return a
}

func (a *adminClient) do(req *http.Request) (int, string, string) {
	a.t.Helper()
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(body), res.Header.Get("Location")
}

func (a *adminClient) get(path string) (int, string, string) {
	req, _ := http.NewRequest(http.MethodGet, a.base+path, nil)
	return a.do(req)
}

func (a *adminClient) post(path string, form url.Values) (int, string, string) {
	req, _ := http.NewRequest(http.MethodPost, a.base+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return a.do(req)
}

// token fetches a page and returns the CSRF token of its form.
func (a *adminClient) token(path string) string {
	a.t.Helper()
	_, body, _ := a.get(path)
	m := csrfField.FindStringSubmatch(body)
	if m == nil {
		a.t.Fatalf("no CSRF field on %s", path)
	}
	return m[1]
}

func TestAdminPostList(t *testing.T) {
	a := newAdminClient(t)

	status, body, _ := a.get("/admin/")
	if status != http.StatusOK {
		t.Fatalf("GET /admin/ = %d", status)
	}
	for _, want := range []string{
		"2 篇", `href="/admin/logout"`, `href="/admin/posts/add?id=hello"`, `action="/admin/posts/delete/hello"`,
		"Older sibling", "2021-01-02", "/static/admin/css/goblog-admin.css?v=", `name="viewport"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("post list does not contain %q", want)
		}
	}
	if strings.Contains(body, "+0800 CST") || strings.Contains(body, "Draft") {
		t.Errorf("dates must be formatted and hidden posts stay out of the list")
	}
}

func TestAdminPostSaveValidation(t *testing.T) {
	a := newAdminClient(t)
	postFile := func(slug string) string { return filepath.Join(a.dataDir, "posts", slug+".md") }

	form := func(id, title, slug, content string) url.Values {
		return url.Values{"_csrf": {a.token("/admin/posts/add")}, "id": {id}, "title": {title}, "identity": {slug},
			"category": {"1"}, "tags": {"go"}, "description": {"摘要"}, "content": {content}}
	}

	tests := []struct {
		name, slug, title, wantMessage string
	}{
		{"empty slug", "", "T", "请填写地址"},
		{"path traversal", "../escape", "T", "小写字母"},
		{"upper case and spaces", "My Post", "T", "小写字母"},
		{"taken by another post", "older", "T", "占用"},
		{"missing title", "fine-slug", "  ", "请填写标题"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body, _ := a.post("/admin/posts/save", form("", tt.title, tt.slug, "写了很久的正文"))
			if status != http.StatusUnprocessableEntity || !strings.Contains(body, tt.wantMessage) {
				t.Fatalf("status %d, want 422 with %q", status, tt.wantMessage)
			}
			if !strings.Contains(body, "写了很久的正文") || !strings.Contains(body, `id="editor-form"`) {
				t.Errorf("a rejected save must hand the text back inside the editor")
			}
		})
	}
	if _, err := os.Stat(filepath.Join(a.dataDir, "escape.md")); !os.IsNotExist(err) {
		t.Errorf("path traversal wrote a file")
	}
	older, _ := os.ReadFile(postFile("older"))
	if !strings.Contains(string(older), "fmt.Println(1)") {
		t.Errorf("the post whose address was requested again must be untouched: %s", older)
	}

	// a valid new post
	status, _, location := a.post("/admin/posts/save", form("", "新文章", "brand-new", "## 你好"))
	if status != http.StatusFound || location != "/admin?saved=brand-new" {
		t.Fatalf("valid save = %d -> %q", status, location)
	}
	if raw, err := os.ReadFile(postFile("brand-new")); err != nil || !strings.Contains(string(raw), "## 你好") {
		t.Fatalf("new post file: %v", err)
	}
	if status, body, _ := a.get("/posts/brand-new"); status != http.StatusOK || !strings.Contains(body, "新文章") {
		t.Errorf("the new post must be public, got %d", status)
	}

	// editing keeps working, renaming moves the file
	status, _, location = a.post("/admin/posts/save", form("brand-new", "新文章（改）", "brand-new-2", "## 改过"))
	if status != http.StatusFound || location != "/admin?saved=brand-new-2" {
		t.Fatalf("rename = %d -> %q", status, location)
	}
	if _, err := os.Stat(postFile("brand-new")); !os.IsNotExist(err) {
		t.Errorf("the old file must be gone after a rename")
	}
	if status, _, _ := a.get("/posts/brand-new-2"); status != http.StatusOK {
		t.Errorf("renamed post = %d", status)
	}
}

func TestAdminEditorPage(t *testing.T) {
	a := newAdminClient(t)
	status, body, _ := a.get("/admin/posts/add?id=hello")
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	for _, want := range []string{
		`data-kind="post"`, `data-id="hello"`, `data-slugs="[&#34;older&#34;]"`, `data-tags="[&#34;go&#34;,&#34;unused&#34;]"`,
		`name="identity" value="hello" data-initial="hello"`, "goblog-editor.js?v=", `value="Hello &#34;Gopher&#34;"`, "正文 **内容**。",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("editor does not contain %q", want)
		}
	}
	if status, _, _ := a.get("/admin/posts/add?id=nope"); status != http.StatusNotFound {
		t.Errorf("editing a missing post = %d, want 404", status)
	}
}

func TestAdminPageSave(t *testing.T) {
	a := newAdminClient(t)
	pageFile := func(id string) string { return filepath.Join(a.dataDir, "pages", id+".md") }

	// editing an existing page: the form sends id (stored page) and page_id (address field)
	status, _, location := a.post("/admin/pages/save", url.Values{
		"_csrf": {a.token("/admin/pages/add?page_id=about")}, "id": {"about"}, "page_id": {"about"}, "title": {"关于我"}, "content": {"新的关于页"},
	})
	if status != http.StatusFound || location != "/admin/pages?saved=about" {
		t.Fatalf("page save = %d -> %q", status, location)
	}
	if raw, _ := os.ReadFile(pageFile("about")); !strings.Contains(string(raw), "新的关于页") {
		t.Errorf("the about page must be updated: %s", raw)
	}
	if _, err := os.Stat(pageFile("")); !os.IsNotExist(err) {
		t.Errorf("saving a page must never write pages/.md")
	}

	for _, id := range []string{"", "../x", "About Me", "about"} {
		status, body, _ := a.post("/admin/pages/save", url.Values{
			"_csrf": {a.token("/admin/pages/add")}, "id": {""}, "page_id": {id}, "title": {"T"}, "content": {"别丢"},
		})
		if status != http.StatusUnprocessableEntity || !strings.Contains(body, "别丢") {
			t.Errorf("new page with id %q = %d, want 422 with the text handed back", id, status)
		}
	}

	status, _, location = a.post("/admin/pages/save", url.Values{
		"_csrf": {a.token("/admin/pages/add")}, "id": {""}, "page_id": {"links"}, "title": {"友链"}, "content": {"朋友们"},
	})
	if status != http.StatusFound || location != "/admin/pages?saved=links" {
		t.Fatalf("new page = %d -> %q", status, location)
	}
	if status, body, _ := a.get("/pages/links"); status != http.StatusOK || !strings.Contains(body, "朋友们") {
		t.Errorf("new page must be public, got %d", status)
	}
}

func TestAdminLogout(t *testing.T) {
	a := newAdminClient(t)
	if status, _, location := a.get("/admin/logout"); status != http.StatusFound || location != "/admin/login" {
		t.Fatalf("logout = %d -> %q", status, location)
	}
	if status, _, location := a.get("/admin/"); status != http.StatusFound || location != "/admin/login" {
		t.Errorf("after logout /admin/ = %d -> %q", status, location)
	}
}
