package routers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goblog/internal/config"
	ginpkg "goblog/internal/pkg/gin"
	"goblog/internal/pkg/view"
)

const testPost = `---
title: "Hello \"Gopher\""
status: 1
created_at: 2024-03-15T10:00:00+08:00
updated_at: 2024-03-15T10:00:00+08:00
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
func newTestServer(t *testing.T) http.Handler {
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
		"posts/hello.md":  testPost,
		"posts/draft.md":  hiddenPost,
		"pages/about.md":  "---\nid: about\ntitle: \"关于我\"\n---\n\n关于页面正文，足够长的一段介绍文字。",
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
			Mode:          "test",
			Host:          "https://blog.example.com",
			Cdn:           "/static",
			SessionSecret: "test-secret",
			DataDir:       dataDir,
		},
		Server: &config.ServerConfig{HttpPort: 0},
	}
	engine := ginpkg.InitGinConfig("test")
	cleanup := NewServer(conf).InitRouter(engine)
	t.Cleanup(cleanup)
	return engine
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
			wantContain: []string{`Hello &#34;Gopher&#34;`, "第一行摘要第二行摘要", `href="/?tag_id=1"`, `aria-current="page"`, "1 / 1"},
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
			wantContain: []string{`<h1 class="article-title">Hello &#34;Gopher&#34;</h1>`, `rel="canonical" href="https://blog.example.com/posts/hello"`, "/static/js/jquery.min.js", "1234 字", `content="第一行摘要第二行摘要"`},
		},
		{name: "hidden post is a 404", target: "/posts/draft", wantStatus: http.StatusNotFound, wantContain: []string{"页面不存在", `content="noindex"`}},
		{name: "missing post is a 404", target: "/posts/nope", wantStatus: http.StatusNotFound, wantContain: []string{"页面不存在"}},
		{name: "unknown route is a themed 404", target: "/definitely/not/here", wantStatus: http.StatusNotFound, wantContain: []string{"返回首页"}},
		{name: "page", target: "/pages/about", wantStatus: http.StatusOK, wantContain: []string{"关于我", `id="page-viewer"`}},
		{name: "missing page is a 404", target: "/pages/nope", wantStatus: http.StatusNotFound},
		{name: "archive", target: "/archive", wantStatus: http.StatusOK, wantContain: []string{`id="y2024"`, "共 1 篇文章"}, wantAbsent: []string{"Draft"}},
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
