package routers

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
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

func newAdminClient(t *testing.T, options ...func(*config.Config)) *adminClient {
	t.Helper()
	var dataDir string
	handler := newTestServer(t, append([]func(*config.Config){func(c *config.Config) {
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
	}}, options...)...)
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

func TestAdminDrafts(t *testing.T) {
	a := newAdminClient(t)
	draftFile := func(slug string) string { return filepath.Join(a.dataDir, ".drafts", slug+".md") }
	form := func(action, draft, title, slug, tags, content string) url.Values {
		return url.Values{"_csrf": {a.token("/admin/posts/add")}, "id": {""}, "draft": {draft}, "action": {action}, "title": {title},
			"identity": {slug}, "category": {"1"}, "tags": {tags}, "description": {""}, "content": {content}}
	}

	// save a draft: stays in the editor, nothing becomes public
	status, _, location := a.post("/admin/posts/save", form("draft", "", "半成品", "wip", "go, 全新标签", "## 还没写完"))
	if status != http.StatusFound || location != "/admin/posts/add?draft=wip&saved=wip" {
		t.Fatalf("save draft = %d -> %q", status, location)
	}
	if raw, err := os.ReadFile(draftFile("wip")); err != nil || !strings.Contains(string(raw), "还没写完") || !strings.Contains(string(raw), "status: 0") {
		t.Fatalf("draft file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(a.dataDir, "posts", "wip.md")); !os.IsNotExist(err) {
		t.Errorf("a draft must not be written to posts/")
	}
	for _, public := range []string{"/posts/wip", "/", "/archive", "/feed.xml", "/sitemap.xml", "/stats"} {
		if status, body, _ := a.get(public); (public == "/posts/wip" && status != http.StatusNotFound) || strings.Contains(body, "半成品") || strings.Contains(body, "还没写完") {
			t.Errorf("draft leaks into %s (status %d)", public, status)
		}
	}
	if _, body, _ := a.get("/api/search?q=" + url.QueryEscape("还没写完")); !strings.Contains(body, `"total":0`) {
		t.Errorf("draft leaks into the search API: %s", body)
	}
	if raw, _ := os.ReadFile(filepath.Join(a.dataDir, "tags.json")); strings.Contains(string(raw), "全新标签") {
		t.Errorf("a draft must not create tags in the public tags.json")
	}

	// the list shows it, the editor loads it, the preview renders it
	if _, body, _ := a.get("/admin/"); !strings.Contains(body, "半成品") || !strings.Contains(body, `href="/admin/posts/add?draft=wip"`) || !strings.Contains(body, `action="/admin/drafts/delete/wip"`) {
		t.Errorf("draft missing from the admin list")
	}
	if status, body, _ := a.get("/admin/posts/add?draft=wip"); status != http.StatusOK || !strings.Contains(body, `name="draft" value="wip"`) ||
		!strings.Contains(body, `value="go,全新标签"`) || !strings.Contains(body, "还没写完") || !strings.Contains(body, `id="save-draft"`) {
		t.Errorf("editor for the draft = %d", status)
	}
	if status, body, _ := a.get("/admin/posts/preview?draft=wip"); status != http.StatusOK || !strings.Contains(body, "草稿预览") ||
		!strings.Contains(body, `<h2 id="还没写完">`) || !strings.Contains(body, `content="noindex"`) || strings.Contains(body, `id="like-btn"`) || strings.Contains(body, "giscus") {
		t.Errorf("preview = %d, want the rendered draft without likes and comments", status)
	}

	// a draft cannot sit on the address of a published post; renaming works
	if status, body, _ := a.post("/admin/posts/save", form("draft", "wip", "半成品", "older", "", "x")); status != http.StatusUnprocessableEntity || !strings.Contains(body, "占用") {
		t.Errorf("draft on a published address = %d", status)
	}
	status, _, location = a.post("/admin/posts/save", form("draft", "wip", "半成品", "wip-2", "go", "## 第二版"))
	if status != http.StatusFound || !strings.HasPrefix(location, "/admin/posts/add?draft=wip-2") {
		t.Fatalf("rename draft = %d -> %q", status, location)
	}
	if _, err := os.Stat(draftFile("wip")); !os.IsNotExist(err) {
		t.Errorf("the old draft file must be gone after a rename")
	}

	// publishing turns the draft into a post and removes the draft
	status, _, location = a.post("/admin/posts/save", form("publish", "wip-2", "成品", "finished", "go,全新标签", "## 写完了"))
	if status != http.StatusFound || location != "/admin?saved=finished" {
		t.Fatalf("publish = %d -> %q", status, location)
	}
	if _, err := os.Stat(draftFile("wip-2")); !os.IsNotExist(err) {
		t.Errorf("publishing must remove the draft")
	}
	if status, body, _ := a.get("/posts/finished"); status != http.StatusOK || !strings.Contains(body, "写完了") || !strings.Contains(body, "全新标签") {
		t.Errorf("published post = %d", status)
	}
	if raw, _ := os.ReadFile(filepath.Join(a.dataDir, "tags.json")); !strings.Contains(string(raw), `"name": "全新标签"`) || strings.Contains(string(raw), `" 全新标签"`) {
		t.Errorf("publishing creates the new tag, trimmed: %s", raw)
	}

	// an already published post cannot be pushed back into a draft through the form
	values := form("draft", "", "成品", "finished", "go", "改了")
	values.Set("id", "finished")
	if status, _, location := a.post("/admin/posts/save", values); status != http.StatusFound || location != "/admin?saved=finished" {
		t.Errorf("action=draft on a published post = %d -> %q, want a normal save", status, location)
	}

	// delete
	a.post("/admin/posts/save", form("draft", "", "要删的", "to-delete", "", "x"))
	if status, _, location := a.post("/admin/drafts/delete/to-delete", url.Values{"_csrf": {a.token("/admin/posts/add")}}); status != http.StatusFound || location != "/admin" {
		t.Errorf("delete draft = %d -> %q", status, location)
	}
	if _, err := os.Stat(draftFile("to-delete")); !os.IsNotExist(err) {
		t.Errorf("draft file must be deleted")
	}
}

func TestAdminTagParsing(t *testing.T) {
	a := newAdminClient(t)
	status, _, _ := a.post("/admin/posts/save", url.Values{"_csrf": {a.token("/admin/posts/add")}, "id": {""}, "title": {"标签测试"}, "identity": {"tag-test"},
		"category": {"1"}, "tags": {" go ， mysql,,mysql , "}, "description": {""}, "content": {"x"}})
	if status != http.StatusFound {
		t.Fatalf("save = %d", status)
	}
	raw, _ := os.ReadFile(filepath.Join(a.dataDir, "tags.json"))
	tags := string(raw)
	if strings.Count(tags, `"mysql"`) != 1 || strings.Contains(tags, `" mysql"`) || strings.Contains(tags, `"name": ""`) {
		t.Errorf("tags must be trimmed, deduplicated and never empty: %s", tags)
	}
	post, _ := os.ReadFile(filepath.Join(a.dataDir, "posts", "tag-test.md"))
	if !strings.Contains(string(post), "tag_ids: [1,") {
		t.Errorf("the existing tag go (id 1) must be reused: %s", post)
	}

	// a post without tags gets none, and no nameless tag appears
	a.post("/admin/posts/save", url.Values{"_csrf": {a.token("/admin/posts/add")}, "id": {""}, "title": {"无标签"}, "identity": {"no-tags"},
		"category": {"1"}, "tags": {""}, "description": {""}, "content": {"x"}})
	raw, _ = os.ReadFile(filepath.Join(a.dataDir, "tags.json"))
	if strings.Contains(string(raw), `"name": ""`) {
		t.Errorf("an empty tags field must not create a tag: %s", raw)
	}
}

func TestAdminTagRenameMergeDelete(t *testing.T) {
	a := newAdminClient(t)
	readTags := func() string {
		raw, _ := os.ReadFile(filepath.Join(a.dataDir, "tags.json"))
		return string(raw)
	}
	postFile := func(slug string) string {
		raw, _ := os.ReadFile(filepath.Join(a.dataDir, "posts", slug+".md"))
		return string(raw)
	}

	// the list offers both actions, the form explains what a known name does
	_, body, _ := a.get("/admin/tags")
	for _, want := range []string{`href="/admin/tags/edit?id=1"`, `action="/admin/tags/delete"`, "改名", "用到它的 2 篇文章会去掉这个标签"} {
		if !strings.Contains(body, want) {
			t.Errorf("tag list does not contain %q", want)
		}
	}
	_, body, _ = a.get("/admin/tags/edit?id=2")
	if !strings.Contains(body, `value="unused"`) || !strings.Contains(body, `data-other-names="[&#34;go&#34;]"`) {
		t.Errorf("rename form must carry the name and the names that would merge")
	}
	if status, _, location := a.get("/admin/tags/edit?id=99"); status != http.StatusFound || location != "/admin/tags" {
		t.Errorf("unknown tag = %d -> %q, want a redirect to the list", status, location)
	}

	// refused names come back with the form, nothing is written
	for _, name := range []string{"   ", "a,b", "a，b", strings.Repeat("长", 41), "two\nlines"} {
		status, body, _ := a.post("/admin/tags/save", url.Values{"_csrf": {a.token("/admin/tags/edit?id=2")}, "id": {"2"}, "name": {name}})
		if status != http.StatusUnprocessableEntity || !strings.Contains(body, "alert-danger") {
			t.Errorf("name %q = %d, want 422 with an explanation", name, status)
		}
	}
	if !strings.Contains(readTags(), `"unused"`) {
		t.Fatalf("a refused rename must not change tags.json: %s", readTags())
	}

	// plain rename
	status, _, location := a.post("/admin/tags/save", url.Values{"_csrf": {a.token("/admin/tags/edit?id=2")}, "id": {"2"}, "name": {" golang "}})
	if status != http.StatusFound || !strings.HasPrefix(location, "/admin/tags?done=renamed&name=golang") {
		t.Fatalf("rename = %d -> %q", status, location)
	}
	if tags := readTags(); !strings.Contains(tags, `"golang"`) || strings.Contains(tags, `"unused"`) || !strings.Contains(tags, `"created_at"`) {
		t.Errorf("tags.json after rename: %s", tags)
	}
	if _, body, _ := a.get(location); !strings.Contains(body, "已改名为「golang」") {
		t.Error("the list must confirm the rename")
	}

	// renaming onto an existing name merges: posts of "go" (id 1) move to "golang" (id 2)
	status, _, location = a.post("/admin/tags/save", url.Values{"_csrf": {a.token("/admin/tags/edit?id=1")}, "id": {"1"}, "name": {"golang"}})
	if status != http.StatusFound || !strings.HasPrefix(location, "/admin/tags?done=merged") {
		t.Fatalf("merge = %d -> %q", status, location)
	}
	if tags := readTags(); strings.Contains(tags, `"go"`) || !strings.Contains(tags, `"count": 2`) {
		t.Errorf("tags.json after merge: %s", tags)
	}
	for _, slug := range []string{"hello", "older"} {
		if post := postFile(slug); !strings.Contains(post, "tag_ids: [2]\n") {
			t.Errorf("%s must be filed under the surviving tag: %.300s", slug, post)
		}
	}
	if post := postFile("older"); !strings.Contains(post, "updated_at: 2021-01-02T10:00:00+08:00") {
		t.Errorf("retagging is not an edit, updated_at must stay: %.300s", post)
	}
	if _, body, _ := a.get("/posts/hello"); !strings.Contains(body, `href="/?tag_id=2">golang</a>`) {
		t.Error("the article must show the surviving tag")
	}

	// delete
	status, _, location = a.post("/admin/tags/delete", url.Values{"_csrf": {a.token("/admin/tags")}, "id": {"2"}})
	if status != http.StatusFound || !strings.HasPrefix(location, "/admin/tags?done=deleted&name=golang") {
		t.Fatalf("delete = %d -> %q", status, location)
	}
	if post := postFile("hello"); !strings.Contains(post, "tag_ids: []\n") {
		t.Errorf("a deleted tag must leave the posts: %.300s", post)
	}
	if _, body, _ := a.get("/posts/hello"); strings.Contains(body, "tag_id=") {
		t.Error("the article still links a deleted tag")
	}
	// (no tag is left, so the list has no form any more: any other page carries the session's token)
	if status, body, _ := a.post("/admin/tags/delete", url.Values{"_csrf": {a.token("/admin/categories/add")}, "id": {"2"}}); status != http.StatusNotFound || !strings.Contains(body, "已经不存在") {
		t.Errorf("deleting twice = %d, want 404 with an explanation", status)
	}

	// anonymous visitors and forged forms get nowhere
	if status, _, _ := a.post("/admin/tags/delete", url.Values{"_csrf": {"forged"}, "id": {"1"}}); status != http.StatusForbidden {
		t.Errorf("forged CSRF token = %d, want 403", status)
	}
}

func TestAdminCategories(t *testing.T) {
	a := newAdminClient(t)

	// a category with posts cannot be deleted: the posts would point at nothing
	status, body, _ := a.post("/admin/categories/delete", url.Values{"_csrf": {a.token("/admin/categories")}, "id": {"1"}})
	if status != http.StatusConflict || !strings.Contains(body, "还有 3 篇文章") {
		t.Errorf("deleting a used category = %d, want 409 with the number of posts", status)
	}
	raw, _ := os.ReadFile(filepath.Join(a.dataDir, "categories.json"))
	if !strings.Contains(string(raw), "技术") {
		t.Fatalf("the category is gone: %s", raw)
	}

	// names are validated, the form comes back with what was typed
	status, body, _ = a.post("/admin/categories/save", url.Values{"_csrf": {a.token("/admin/categories/add")}, "id": {""}, "name": {"  "}})
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, "分类名不能为空") {
		t.Errorf("empty name = %d", status)
	}

	// add, rename, delete
	status, _, location := a.post("/admin/categories/save", url.Values{"_csrf": {a.token("/admin/categories/add")}, "id": {""}, "name": {" 随笔 "}})
	if status != http.StatusFound || location != "/admin/categories" {
		t.Fatalf("add = %d -> %q", status, location)
	}
	status, _, _ = a.post("/admin/categories/save", url.Values{"_csrf": {a.token("/admin/categories/add?id=2")}, "id": {"2"}, "name": {"生活"}})
	if status != http.StatusFound {
		t.Fatalf("rename = %d", status)
	}
	raw, _ = os.ReadFile(filepath.Join(a.dataDir, "categories.json"))
	if !strings.Contains(string(raw), `"生活"`) || strings.Contains(string(raw), "随笔") {
		t.Errorf("categories.json after rename: %s", raw)
	}
	status, _, location = a.post("/admin/categories/delete", url.Values{"_csrf": {a.token("/admin/categories")}, "id": {"2"}})
	if status != http.StatusFound || location != "/admin/categories" {
		t.Fatalf("delete = %d -> %q, want a redirect to the list (it used to lead to a 404)", status, location)
	}
	if status, _, _ := a.get(location); status != http.StatusOK {
		t.Errorf("the page after deleting = %d", status)
	}
	if status, _, location := a.get("/admin/categories/add?id=2"); status != http.StatusFound || location != "/admin/categories" {
		t.Errorf("editing a deleted category = %d -> %q", status, location)
	}
}

func TestAdminLoginFailuresLookAlike(t *testing.T) {
	a := newAdminClient(t)
	a.get("/admin/logout")

	var bodies []string
	for _, creds := range [][2]string{{adminEmail, "wrong password"}, {"nobody@example.com", adminPassword}, {"", ""}} {
		status, body, _ := a.post("/admin/sign-in", url.Values{"email": {creds[0]}, "password": {creds[1]}})
		if status != http.StatusUnauthorized || !strings.Contains(body, "邮箱或密码不正确") {
			t.Errorf("login as %q = %d, want 401 with the generic message", creds[0], status)
		}
		bodies = append(bodies, body)
	}
	if bodies[0] != bodies[1] {
		t.Errorf("a wrong password and an unknown account must be indistinguishable")
	}
	if status, _, location := a.get("/admin/"); status != http.StatusFound || location != "/admin/login" {
		t.Errorf("failed logins must not open the admin: %d -> %q", status, location)
	}
}

func TestAdminAccountFromConfig(t *testing.T) {
	const configPassword = "only in the config file"
	hash, err := bcrypt.GenerateFromPassword([]byte(configPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	var dataDir string
	handler := newTestServer(t, func(c *config.Config) {
		dataDir = c.App.DataDir
		c.App.Host = "http://blog.example.com"
		c.App.AdminEmail = "Owner@Example.com"
		c.App.AdminPasswordHash = string(hash)
		// an account in the (public) content repository must not work any more
		old, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.MinCost)
		users := `[{"id":1,"email":"` + adminEmail + `","password":"` + string(old) + `"}]`
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

	if status, _, _ := a.post("/admin/sign-in", url.Values{"email": {adminEmail}, "password": {adminPassword}}); status != http.StatusUnauthorized {
		t.Errorf("users.json account = %d, want 401 once the config defines the admin", status)
	}
	if status, _, _ := a.post("/admin/sign-in", url.Values{"email": {"owner@example.com"}, "password": {"wrong"}}); status != http.StatusUnauthorized {
		t.Errorf("wrong password = %d, want 401", status)
	}
	status, _, location := a.post("/admin/sign-in", url.Values{"email": {"owner@example.com"}, "password": {configPassword}})
	if status != http.StatusFound || location != "/admin" {
		t.Fatalf("config account = %d -> %q, want a redirect into the admin", status, location)
	}
	if status, _, _ := a.get("/admin/"); status != http.StatusOK {
		t.Errorf("admin after login = %d", status)
	}
}

// The account in users.json travels with the (usually public) content repository. The admin keeps saying so
// until the account has moved into the config file.
func TestAdminAccountWarning(t *testing.T) {
	const warning = "admin-account-warning"

	t.Run("local data directory", func(t *testing.T) {
		a := newAdminClient(t)
		if _, body, _ := a.get("/admin/"); strings.Contains(body, warning) {
			t.Error("nothing is published from a plain data directory, there is nothing to warn about")
		}
	})

	t.Run("git-backed data directory", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git is not installed")
		}
		a := newAdminClient(t, func(c *config.Config) {
			if out, err := exec.Command("git", "init", "--quiet", c.App.DataDir).CombinedOutput(); err != nil {
				t.Fatalf("git init: %v\n%s", err, out)
			}
		})
		for _, path := range []string{"/admin/", "/admin/tags", "/admin/categories/add"} {
			status, body, _ := a.get(path)
			if status != http.StatusOK || !strings.Contains(body, warning) || !strings.Contains(body, "-hash-password") {
				t.Errorf("%s = %d, the users.json warning (with the way out) is missing", path, status)
			}
		}
		// the two full-screen editors stay free of banners
		if _, body, _ := a.get("/admin/posts/add"); strings.Contains(body, warning) {
			t.Error("the editor must not carry the warning")
		}
	})
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
