package routers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goblog/internal/config"
	ginpkg "goblog/internal/pkg/gin"
	"goblog/internal/pkg/view"
)

// testPost is dated relative to "now" (see newTestServer) so age-dependent
// features such as the outdated notice behave the same whenever the test runs.
const testPost = `---
title: "Hello \"Gopher\""
status: 1
created_at: %[1]s
updated_at: %[1]s
category_id: 1
is_top: 0
tag_ids: [1]
description: "第一行摘要
![img](https://example.com/a.png)
第二行摘要"
word_count: 1234
---

## 小标题

正文 **内容**。
`

const olderPost = `---
title: "Older sibling"
status: 1
created_at: 2021-01-02T10:00:00+08:00
updated_at: 2021-01-02T10:00:00+08:00
category_id: 1
is_top: 0
tag_ids: [1]
description: "older"
word_count: 800
---

` + "```go\nfmt.Println(1)\n```" + `
`

const hiddenPost = `---
title: "Draft"
status: 0
created_at: 2024-03-16T10:00:00+08:00
updated_at: 2024-03-16T10:00:00+08:00
category_id: 1
is_top: 0
tag_ids: []
description: "hidden"
word_count: 1
---

secret
`

// newTestServer boots the real router against a throw-away data directory.
// The process working directory becomes a temp dir that links to the real
// tpl/ and static/ trees, so files generated at startup (feed.xml,
// sitemap.xml, heatmap.txt) never touch the repository.
func newTestServer(t *testing.T, options ...func(*config.Config)) http.Handler {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	for _, dir := range []string{"tpl", "static"} {
		if err := os.Symlink(filepath.Join(repoRoot, dir), filepath.Join(work, dir)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(work, "robots.txt"), []byte("User-agent: *\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dataDir := filepath.Join(work, "data")
	files := map[string]string{
		"posts/hello.md":  fmt.Sprintf(testPost, recentDate().Format(time.RFC3339)),
		"posts/older.md":  olderPost,
		"posts/draft.md":  hiddenPost,
		"pages/about.md":  "---\nid: about\ntitle: \"关于我\"\n---\n\n关于页面正文，足够长的一段介绍文字。",
		"pages/flow.md":   "---\nid: flow\ntitle: \"流程图\"\n---\n\n```flow\nst=>start: 开始\n```\n",
		"categories.json": `[{"id":1,"name":"技术"}]`,
		"tags.json":       `[{"id":1,"name":"go","count":1},{"id":2,"name":"unused","count":0}]`,
		"books.json":      `[{"id":1,"title":"Clean Code","cover":"/covers/c.jpg","status":2,"progress":100,"year":2024}]`,
		"covers/c.jpg":    "not-really-a-jpeg",
	}
	for name, content := range files {
		path := filepath.Join(dataDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Chdir(work)
	view.InitTemplates()

	conf := &config.Config{
		App: &config.AppConfig{
			Name:          "测试博客",
			Description:   "写给测试的博客",
			Mode:          "test",
			Host:          "https://blog.example.com",
			Cdn:           "/static",
			SessionSecret: "test-secret",
			DataDir:       dataDir,
		},
		Server: &config.ServerConfig{HttpPort: 0},
	}
	for _, option := range options {
		option(conf)
	}
	engine := ginpkg.InitGinConfig("test")
	cleanup := NewServer(conf).InitRouter(engine)
	t.Cleanup(cleanup)
	return engine
}

// recentDate is a publication date that is always "one month ago".
func recentDate() time.Time {
	return time.Now().AddDate(0, -1, 0)
}

func TestLikeEndpoint(t *testing.T) {
	h := newTestServer(t)

	post := func(target string, headers map[string]string, cookies []*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, target, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	same := map[string]string{"X-Requested-With": "goblog", "Sec-Fetch-Site": "same-origin"}

	if rec := post("/api/posts/hello/like", nil, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("missing custom header must be rejected, got %d", rec.Code)
	}
	if rec := post("/api/posts/hello/like", map[string]string{"X-Requested-With": "goblog", "Sec-Fetch-Site": "cross-site"}, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site request must be rejected, got %d", rec.Code)
	}
	if rec := post("/api/posts/draft/like", same, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("hidden posts cannot be liked, got %d", rec.Code)
	}

	first := post("/api/posts/hello/like", same, nil)
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"likes":1`) {
		t.Fatalf("first like: %d %s", first.Code, first.Body.String())
	}
	cookies := first.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].HttpOnly {
		t.Fatalf("like must set an HttpOnly cookie, got %+v", cookies)
	}

	again := post("/api/posts/hello/like", same, cookies)
	if again.Code != http.StatusOK || !strings.Contains(again.Body.String(), `"likes":1`) || !strings.Contains(again.Body.String(), `"already":true`) {
		t.Fatalf("liking twice from the same browser must be idempotent: %d %s", again.Code, again.Body.String())
	}

	other := post("/api/posts/hello/like", same, nil)
	if !strings.Contains(other.Body.String(), `"likes":2`) {
		t.Fatalf("another browser adds a like: %s", other.Body.String())
	}

	// the post page reflects the counter and the pressed state
	req := httptest.NewRequest(http.MethodGet, "/posts/hello", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `id="like-count">2<`) || !strings.Contains(body, `aria-pressed="true"`) {
		t.Errorf("post page must show likes=2 and the liked state")
	}
}

func TestRandomRedirect(t *testing.T) {
	h := newTestServer(t)

	for i := 0; i < 5; i++ {
		rec := get(t, h, "/random?from=hello")
		if rec.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302", rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/posts/older" {
			t.Fatalf("with from=hello the only other published post must be picked, got %q", loc)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("random redirects must not be cached, Cache-Control = %q", cc)
		}
	}
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestFrontPages(t *testing.T) {
	h := newTestServer(t)

	tests := []struct {
		name        string
		target      string
		wantStatus  int
		wantContain []string
		wantAbsent  []string
	}{
		{
			name: "home lists published posts with a clean excerpt", target: "/", wantStatus: http.StatusOK,
			wantContain: []string{`Hello &#34;Gopher&#34;`, "第一行摘要第二行摘要", `href="/?tag_id=1"`, `aria-current="page"`, "1 / 1", "Older sibling"},
			wantAbsent:  []string{"Draft", "example.com/a.png", "&lt;no value&gt;", "<no value>"},
		},
		{
			name: "search keeps the keyword and is not indexable", target: "/?keyword=gopher", wantStatus: http.StatusOK,
			wantContain: []string{`value="gopher"`, "搜索：", `content="noindex"`, `Hello &#34;Gopher&#34;`},
		},
		{
			name: "empty result shows the empty state", target: "/?keyword=zzzz-nothing", wantStatus: http.StatusOK,
			wantContain: []string{"这里还没有内容"},
		},
		{
			name: "post page", target: "/posts/hello", wantStatus: http.StatusOK,
			wantContain: []string{`<h1 class="article-title">Hello &#34;Gopher&#34;</h1>`, `rel="canonical" href="https://blog.example.com/posts/hello"`, "1234 字", `content="第一行摘要第二行摘要"`},
		},
		{
			name: "post carries social and structured metadata", target: "/posts/hello", wantStatus: http.StatusOK,
			wantContain: []string{
				`<title>Hello &#34;Gopher&#34; | 测试博客</title>`, `<meta property="og:type" content="article"/>`,
				`<meta property="og:image" content="https://blog.example.com/static/logo.png"/>`, `<meta name="twitter:card" content="summary"/>`,
				`<meta property="article:published_time" content="`, `<meta property="article:section" content="技术"/>`, `<meta property="article:tag" content="go"/>`,
				`<script type="application/ld+json">{"@context":"https://schema.org","@type":"BlogPosting"`, `"headline":"Hello \"Gopher\""`,
			},
		},
		{
			name: "home page title is not doubled and describes the site", target: "/", wantStatus: http.StatusOK,
			wantContain: []string{
				"<title>测试博客 - 写给测试的博客</title>", `<meta name="description" content="写给测试的博客"/>`, `rel="canonical" href="https://blog.example.com/"`,
				`"@type":"Blog"`, `"target":"https://blog.example.com/?keyword={search_term_string}"`, `<meta property="og:type" content="website"/>`,
			},
			wantAbsent: []string{"article:published_time", "测试博客 | 测试博客"},
		},
		{
			name: "filtered and paged lists have their own canonical and no site data", target: "/?tag_id=1&page=1", wantStatus: http.StatusOK,
			wantContain: []string{`rel="canonical" href="https://blog.example.com/?tag_id=1"`, "<title>标签：go | 测试博客</title>"},
			wantAbsent:  []string{`"@type":"Blog"`},
		},
		{
			name: "search results have no canonical", target: "/?keyword=gopher", wantStatus: http.StatusOK,
			wantAbsent: []string{`rel="canonical"`},
		},
		{name: "archive canonical", target: "/archive", wantStatus: http.StatusOK, wantContain: []string{`rel="canonical" href="https://blog.example.com/archive"`, "<title>归档 | 测试博客</title>"}},
		{
			name: "articles are rendered on the server, without the editor.md stack", target: "/posts/hello", wantStatus: http.StatusOK,
			wantContain: []string{`data-rendered="server"`, `<h2 id="小标题">小标题</h2>`, "<p>正文 <strong>内容</strong>。</p>", "/static/js/article.js", "prettify.min.js"},
			wantAbsent:  []string{"jquery.min.js", "editormd", "<textarea", "marked.min.js"},
		},
		{
			name: "code blocks are ready for prettify", target: "/posts/older", wantStatus: http.StatusOK,
			wantContain: []string{`<pre class="prettyprint linenums"><code class="language-go">fmt.Println(1)`},
		},
		{
			name: "content that needs editor.md falls back to the browser renderer", target: "/pages/flow", wantStatus: http.StatusOK,
			wantContain: []string{"<textarea", "jquery.min.js", "editormd", "st=&gt;start"},
			wantAbsent:  []string{`data-rendered="server"`},
		},
		{
			name: "post links to its neighbour and related posts", target: "/posts/hello", wantStatus: http.StatusOK,
			wantContain: []string{`rel="prev" href="/posts/older"`, "相关文章", "Older sibling", "CC BY-NC-SA 4.0"},
			wantAbsent:  []string{`rel="next"`, "notice-outdated", "Draft"},
		},
		{
			name: "old technical post carries the outdated notice", target: "/posts/older", wantStatus: http.StatusOK,
			wantContain: []string{"notice-outdated", `rel="next" href="/posts/hello"`},
			wantAbsent:  []string{`rel="prev"`},
		},
		{name: "hidden post is a 404", target: "/posts/draft", wantStatus: http.StatusNotFound, wantContain: []string{"页面不存在", `content="noindex"`}},
		{name: "missing post is a 404", target: "/posts/nope", wantStatus: http.StatusNotFound, wantContain: []string{"页面不存在"}},
		{name: "unknown route is a themed 404", target: "/definitely/not/here", wantStatus: http.StatusNotFound, wantContain: []string{"返回首页"}},
		{name: "page", target: "/pages/about", wantStatus: http.StatusOK, wantContain: []string{"关于我", `id="page-viewer"`, `data-rendered="server"`, "<p>关于页面正文，足够长的一段介绍文字。</p>"}},
		{name: "missing page is a 404", target: "/pages/nope", wantStatus: http.StatusNotFound},
		{
			name: "archive", target: "/archive", wantStatus: http.StatusOK,
			wantContain: []string{fmt.Sprintf(`id="y%d"`, recentDate().Year()), `id="y2021"`, "共 2 篇文章"},
			wantAbsent:  []string{"Draft"},
		},
		{
			name: "tags hide unused ones and embed heatmap JSON as JSON", target: "/tags", wantStatus: http.StatusOK,
			wantContain: []string{`href="/?tag_id=1"`, `<script id="heatmap-data" type="application/json">[{`},
			wantAbsent:  []string{"unused", `type="application/json">"`},
		},
		{name: "reading list", target: "/reading", wantStatus: http.StatusOK, wantContain: []string{"Clean Code", `src="/covers/c.jpg"`}},
		{name: "book covers are served from the data dir", target: "/covers/c.jpg", wantStatus: http.StatusOK, wantContain: []string{"not-really-a-jpeg"}},
		{name: "feed", target: "/feed.xml", wantStatus: http.StatusOK, wantContain: []string{"https://blog.example.com/posts/hello"}, wantAbsent: []string{"Draft"}},
		{name: "sitemap", target: "/sitemap.xml", wantStatus: http.StatusOK, wantContain: []string{"https://blog.example.com/posts/hello"}},
		{name: "health", target: "/ping", wantStatus: http.StatusOK},
		{
			name: "search API ranks and highlights", target: "/api/search?q=gopher", wantStatus: http.StatusOK,
			wantContain: []string{`"total":1`, `"url":"/posts/hello"`, `\u003cmark\u003eGopher\u003c/mark\u003e`},
			wantAbsent:  []string{"Draft"},
		},
		{
			name: "search API with an empty query returns hot posts", target: "/api/search?q=", wantStatus: http.StatusOK,
			wantContain: []string{`"hot":[`, `"url":"/posts/hello"`},
			wantAbsent:  []string{"Draft"},
		},
		{
			name: "stats page", target: "/stats", wantStatus: http.StatusOK,
			wantContain: []string{"博客数据", "最受欢迎的文章", "每年写了多少", `id="time-progress"`, "2021", `<script id="heatmap-data" type="application/json">[`},
			wantAbsent:  []string{"Draft", "<no value>"},
		},
		{
			name: "footer shows the site age", target: "/archive", wantStatus: http.StatusOK,
			wantContain: []string{`id="site-days">已运行 `},
		},
		{
			name: "home sidebar shows stats and hot posts", target: "/", wantStatus: http.StatusOK,
			wantContain: []string{"热门文章", "常用标签", `class="hot-rank">1<`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, h, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("GET %s: status = %d, want %d", tt.target, rec.Code, tt.wantStatus)
			}
			body := rec.Body.String()
			for _, want := range tt.wantContain {
				if !strings.Contains(body, want) {
					t.Errorf("GET %s: body does not contain %q", tt.target, want)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(body, absent) {
					t.Errorf("GET %s: body must not contain %q", tt.target, absent)
				}
			}
		})
	}
}

func TestStaticCacheHeaders(t *testing.T) {
	h := newTestServer(t)

	tests := []struct {
		target, want string
	}{
		{"/static/css/style.css?v=abc123", "public, max-age=31536000, immutable"},
		{"/static/css/style.css", "public, max-age=86400, stale-while-revalidate=604800"},
		{"/covers/c.jpg", "public, max-age=86400, stale-while-revalidate=604800"},
		{"/static/nope.css?v=1", "no-store"},
		{"/posts/hello", ""},
		{"/api/search?q=go&v=1", "public, max-age=60"}, // the API's own policy, untouched
	}
	for _, tt := range tests {
		if got := get(t, h, tt.target).Header().Get("Cache-Control"); got != tt.want {
			t.Errorf("GET %s: Cache-Control = %q, want %q", tt.target, got, tt.want)
		}
	}
}

func TestHeadRequestsDoNotCountAsViews(t *testing.T) {
	h := ginpkg.HeadAsGet(newTestServer(t))

	views := func() string {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/hello", nil))
		body := rec.Body.String()
		i := strings.Index(body, `title="阅读 `)
		if i < 0 {
			t.Fatalf("view counter not found")
		}
		return body[i : i+len(`title="阅读 `)+4]
	}
	first := views() // the page shows the count before this visit is added
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/posts/hello", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("HEAD /posts/hello = %d, want 200", rec.Code)
		}
	}
	if second := views(); !strings.Contains(first, "阅读 0 ") || !strings.Contains(second, "阅读 1 ") {
		t.Errorf("the three HEAD probes must not count as views: first %q, second %q", first, second)
	}
}

func TestPWAEndpoints(t *testing.T) {
	h := newTestServer(t)

	manifest := get(t, h, "/manifest.webmanifest")
	if manifest.Code != http.StatusOK || !strings.HasPrefix(manifest.Header().Get("Content-Type"), "application/manifest+json") {
		t.Fatalf("manifest: status %d, type %q", manifest.Code, manifest.Header().Get("Content-Type"))
	}
	var m struct {
		Name        string `json:"name"`
		ShortName   string `json:"short_name"`
		Description string `json:"description"`
		StartURL    string `json:"start_url"`
		Display     string `json:"display"`
		Icons       []struct {
			Src, Sizes, Purpose string
		} `json:"icons"`
	}
	if err := json.Unmarshal(manifest.Body.Bytes(), &m); err != nil {
		t.Fatalf("manifest is not JSON: %v", err)
	}
	if m.Name != "测试博客" || m.Description != "写给测试的博客" || m.StartURL != "/" || m.Display != "standalone" || len(m.Icons) != 3 {
		t.Errorf("unexpected manifest: %+v", m)
	}
	for _, icon := range m.Icons {
		if rec := get(t, h, icon.Src); rec.Code != http.StatusOK || !strings.Contains(icon.Src, "?v=") {
			t.Errorf("icon %q: status %d, must exist and be fingerprinted", icon.Src, rec.Code)
		}
	}

	sw := get(t, h, "/sw.js")
	body := sw.Body.String()
	if sw.Code != http.StatusOK || !strings.HasPrefix(sw.Header().Get("Content-Type"), "text/javascript") || sw.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("sw.js: status %d, headers %v", sw.Code, sw.Header())
	}
	for _, want := range []string{
		`self.__GOBLOG = {"offline":"/offline","precache":["/static/css/style.css?v=`, `"version":"`, "addEventListener('fetch'",
		`/^\/(admin|api|random|feed|`, // what the worker must never touch
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sw.js does not contain %q", want)
		}
	}

	offline := get(t, h, "/offline")
	if offline.Code != http.StatusOK || !strings.Contains(offline.Body.String(), "现在没有网络") || !strings.Contains(offline.Body.String(), `content="noindex"`) {
		t.Errorf("offline page: status %d", offline.Code)
	}

	home := get(t, h, "/").Body.String()
	for _, want := range []string{`<link rel="manifest" href="/manifest.webmanifest"/>`, `<meta name="goblog:sw" content="/sw.js"/>`, `rel="apple-touch-icon" href="/static/icons/apple-touch-icon.png?v=`} {
		if !strings.Contains(home, want) {
			t.Errorf("home page does not contain %q", want)
		}
	}
}

func TestPWACanBeSwitchedOff(t *testing.T) {
	off := false
	h := newTestServer(t, func(c *config.Config) { c.App.PWA = &off })

	home := get(t, h, "/").Body.String()
	if strings.Contains(home, "goblog:sw") || strings.Contains(home, `rel="manifest"`) {
		t.Errorf("pages must not advertise the worker or the manifest when app.pwa is false")
	}
	sw := get(t, h, "/sw.js").Body.String()
	if !strings.Contains(sw, "registration.unregister()") || strings.Contains(sw, "__GOBLOG") {
		t.Errorf("sw.js must become the retiring worker: %s", sw)
	}
}

func TestClientRenderMode(t *testing.T) {
	h := newTestServer(t, func(c *config.Config) { c.App.MarkdownRender = "client" })

	for _, target := range []string{"/posts/hello", "/pages/about"} {
		rec := get(t, h, target)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status = %d", target, rec.Code)
		}
		body := rec.Body.String()
		for _, want := range []string{"<textarea", "jquery.min.js", "editormd"} {
			if !strings.Contains(body, want) {
				t.Errorf("GET %s: markdown_render=client must ship %q", target, want)
			}
		}
		if strings.Contains(body, `data-rendered="server"`) {
			t.Errorf("GET %s: must not be rendered on the server", target)
		}
	}
}
