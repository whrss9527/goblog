package routers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goblog/internal/config"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
)

// fakeGitHub stands in for api.github.com: a few repositories of "whrss9527".
func fakeGitHub(t *testing.T) *httptest.Server {
	t.Helper()
	repos := map[string]string{
		"goblog": `{"name":"goblog","full_name":"whrss9527/goblog","description":"Go 写的博客","html_url":"https://github.com/whrss9527/goblog",
			"homepage":"https://whrss.com","language":"Go","topics":["blog","go","pwa"],"stargazers_count":1234,"forks_count":3,
			"created_at":"2022-11-20T10:00:00Z","pushed_at":"` + time.Now().Add(-72*time.Hour).UTC().Format(time.RFC3339) + `"}`,
		"proxyswitch": `{"name":"proxyswitch","full_name":"whrss9527/proxyswitch","description":"切换系统代理的小工具","html_url":"https://github.com/whrss9527/proxyswitch",
			"homepage":"","language":"Go","topics":["cli","proxy"],"stargazers_count":41,"forks_count":0,
			"created_at":"2026-08-30T08:00:00Z","pushed_at":"2026-09-24T13:00:37Z"}`,
		"old-thing": `{"name":"old-thing","full_name":"whrss9527/old-thing","html_url":"https://github.com/whrss9527/old-thing","language":"Kotlin",
			"archived":true,"created_at":"2021-01-01T00:00:00Z","pushed_at":"2021-02-01T00:00:00Z"}`,
		"secret": `{"name":"secret","full_name":"whrss9527/secret","html_url":"https://github.com/whrss9527/secret","private":true,"language":"Go"}`,
	}
	list := `[
		{"name":"whrss9527","html_url":"https://github.com/whrss9527/whrss9527","pushed_at":"2026-09-26T02:00:57Z"},
		{"name":"GitHubPoster","html_url":"https://github.com/whrss9527/GitHubPoster","fork":true},
		{"name":"proxyswitch","html_url":"https://github.com/whrss9527/proxyswitch","language":"Go","stargazers_count":41,"description":"切换系统代理的小工具"},
		{"name":"goblog","html_url":"https://github.com/whrss9527/goblog","language":"Go"},
		{"name":"old-thing","html_url":"https://github.com/whrss9527/old-thing","archived":true}
	]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/whrss9527/repos" {
			w.Write([]byte(list))
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/repos/whrss9527/")
		if doc, ok := repos[strings.ToLower(name)]; ok && name != r.URL.Path {
			w.Write([]byte(doc))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server
}

// withProjects writes projects.json into the test data directory.
func withProjects(projects string) func(*config.Config) {
	return func(c *config.Config) {
		if err := os.WriteFile(filepath.Join(c.App.DataDir, "projects.json"), []byte(projects), 0o644); err != nil {
			panic(err)
		}
	}
}

const testProjects = `[
  {"id": 1, "name": "goblog", "description": "这个博客", "highlights": ["离线可读", "Git 做存储"], "repo": "https://github.com/whrss9527/goblog",
   "tech": ["Gin", "PWA"], "status": 1, "post": "hello", "featured": true, "started": "2022-11"},
  {"id": 2, "name": "Gitee 上的旧项目", "description": "早年的东西 & 一些尝试", "repo": "https://gitee.com/whrss9527/old", "url": "https://old.example.com",
   "tech": ["Java"], "status": 3, "started": "2019-03"},
  {"id": 3, "name": "Draft article", "repo": "javascript:alert(1)", "post": "draft", "status": 2, "started": "2024-01"},
  {"id": 4, "name": "私有", "repo": "https://github.com/whrss9527/secret", "status": 1},
  {"id": 5, "name": "已删除", "repo": "https://github.com/whrss9527/deleted-long-ago", "status": 2}
]`

func TestProjectsPage(t *testing.T) {
	api := fakeGitHub(t)
	h := newTestServer(t, withProjects(testProjects), func(c *config.Config) { c.App.GitHubAPI = api.URL })

	// the numbers arrive in the background right after startup
	var body string
	deadline := time.Now().Add(5 * time.Second)
	for {
		body = get(t, h, "/projects").Body.String()
		if strings.Contains(body, "1234 个 Star") || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	for _, want := range []string{
		`<a href="/projects" aria-current="page">项目</a>`, `rel="canonical" href="https://blog.example.com/projects"`, "<title>项目 | 测试博客</title>",
		`id="project-goblog"`, `<a href="https://github.com/whrss9527/goblog">goblog</a>`, "精选", "离线可读",
		"1234 个 Star", "1234", "3 天前更新", "始于 2022 年 11 月",
		`class="is-lang" style="--lang: #00add8"`, ">Go</li>", ">Gin</li>", // the repository language comes first, in its colour
		`href="/posts/hello" title="Hello &#34;Gopher&#34;"`, "介绍文章",
		"早年的东西 &amp; 一些尝试", `href="https://gitee.com/whrss9527/old"`, "Gitee", `href="https://old.example.com"`, "访问", "status-archived",
		`data-filter="3"`, "已归档<small>1</small>", "共 5 个", "进行中 2 个",
		`"@type":"CollectionPage"`, `"@type":"SoftwareSourceCode"`, `"codeRepository":"https://github.com/whrss9527/goblog"`,
		`"url":"https://blog.example.com/projects#project-goblog"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/projects does not contain %q", want)
		}
	}
	for _, unwanted := range []string{
		"javascript:", "alert(1)", // an unsafe address in a hand-edited file is dropped
		`href="/posts/draft"`,                           // unpublished posts are not linked
		`href="https://github.com/whrss9527/secret"`,    // private: visitors would get a 404
		`https://github.com/whrss9527/deleted-long-ago`, // GitHub says it is gone
		"<no value>",
	} {
		if strings.Contains(body, unwanted) {
			t.Errorf("/projects must not contain %q", unwanted)
		}
	}
	if strings.Index(body, `id="project-goblog"`) > strings.Index(body, `id="project-gitee-上的旧项目"`) {
		t.Errorf("featured projects come first, archived ones last")
	}

	home := get(t, h, "/").Body.String()
	for _, want := range []string{`href="/projects"`, "widget-projects", `href="/projects#project-goblog"`, "全部 5 个"} {
		if !strings.Contains(home, want) {
			t.Errorf("home page does not contain %q", want)
		}
	}
	if strings.Contains(home, "Gitee 上的旧项目") {
		t.Errorf("the home sidebar leaves archived projects out")
	}

	post := get(t, h, "/posts/hello").Body.String()
	if !strings.Contains(post, "文中的项目") || !strings.Contains(post, `<h3 class="project-name"><a href="https://github.com/whrss9527/goblog">goblog</a></h3>`) {
		t.Errorf("the post that introduces a project shows it")
	}
	if strings.Contains(post, "介绍文章") {
		t.Errorf("inside the article the card does not link to the article itself")
	}
	if older := get(t, h, "/posts/older").Body.String(); strings.Contains(older, "文中的项目") {
		t.Errorf("other posts have no project section")
	}

	if sitemap := get(t, h, "/sitemap.xml").Body.String(); !strings.Contains(sitemap, "https://blog.example.com/projects") {
		t.Errorf("the sitemap lists /projects")
	}
	if manifest := get(t, h, "/manifest.webmanifest").Body.String(); !strings.Contains(manifest, `"url":"/projects"`) {
		t.Errorf("the installed app offers a shortcut to /projects")
	}
}

func TestProjectsWithoutGitHubStats(t *testing.T) {
	off := false
	h := newTestServer(t, withProjects(testProjects), func(c *config.Config) { c.App.GitHubStats = &off })
	body := get(t, h, "/projects").Body.String()
	if strings.Contains(body, "Star") || strings.Contains(body, "更新</span>") {
		t.Errorf("without github_stats there are no numbers")
	}
	for _, want := range []string{"goblog", ">Gin</li>", `href="https://github.com/whrss9527/goblog"`, `href="https://github.com/whrss9527/secret"`} {
		if !strings.Contains(body, want) {
			t.Errorf("/projects does not contain %q", want)
		}
	}
}

func TestNoProjects(t *testing.T) {
	h := newTestServer(t)
	home := get(t, h, "/").Body.String()
	if strings.Contains(home, `href="/projects"`) {
		t.Errorf("the navigation only links to /projects when there are projects")
	}
	rec := get(t, h, "/projects")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "这里还没有项目") || !strings.Contains(rec.Body.String(), `content="noindex"`) {
		t.Errorf("an empty /projects shows the empty state and is not indexed: %d", rec.Code)
	}
	if strings.Contains(get(t, h, "/sitemap.xml").Body.String(), "/projects") {
		t.Errorf("an empty /projects is not in the sitemap")
	}
	if strings.Contains(get(t, h, "/manifest.webmanifest").Body.String(), "/projects") {
		t.Errorf("no shortcut to an empty page")
	}
}

func TestAdminProjects(t *testing.T) {
	api := fakeGitHub(t)
	a := newAdminClient(t, func(c *config.Config) {
		c.App.GitHubAPI = api.URL
		c.App.GitHubUser = "whrss9527" // in production usually the owner of git_repo
	})
	if view.HasProjects() {
		t.Fatalf("the test data has no projects")
	}

	status, body, _ := a.get("/admin/projects")
	if status != http.StatusOK || !strings.Contains(body, "添加项目") || !strings.Contains(body, `data-user="whrss9527"`) || !strings.Contains(body, `href="/admin/projects"`) {
		t.Fatalf("project list = %d", status)
	}

	// import from GitHub
	var imported map[string]any
	status, body, _ = a.get("/admin/projects/github?repo=" + url.QueryEscape("whrss9527/proxyswitch"))
	if status != http.StatusOK || json.Unmarshal([]byte(body), &imported) != nil {
		t.Fatalf("import = %d %s", status, body)
	}
	if imported["name"] != "proxyswitch" || imported["repo"] != "https://github.com/whrss9527/proxyswitch" || imported["started"] != "2026-08" ||
		imported["status"] != float64(model.ProjectStatusActive) || strings.Join(toStrings(imported["tech"]), ",") != "Go,cli,proxy" {
		t.Errorf("imported = %v", imported)
	}
	status, body, _ = a.get("/admin/projects/github?repo=" + url.QueryEscape("git@github.com:whrss9527/goblog.git"))
	imported = nil
	if status != http.StatusOK || json.Unmarshal([]byte(body), &imported) != nil ||
		strings.Join(toStrings(imported["tech"]), ",") != "Go,blog,pwa" || imported["url"] != "https://whrss.com" {
		t.Errorf("import of goblog = %d %v (the topic \"go\" repeats the language)", status, imported)
	}
	status, body, _ = a.get("/admin/projects/github?repo=" + url.QueryEscape("https://github.com/whrss9527/old-thing"))
	if status != http.StatusOK || !strings.Contains(body, `"status":3`) {
		t.Errorf("an archived repository is imported as archived: %s", body)
	}
	if status, _, _ = a.get("/admin/projects/github?repo=not%20a%20repo"); status != http.StatusBadRequest {
		t.Errorf("a malformed address = %d, want 400", status)
	}
	if status, body, _ = a.get("/admin/projects/github?repo=whrss9527/nope"); status != http.StatusNotFound || !strings.Contains(body, "没有找到") {
		t.Errorf("a missing repository = %d %s", status, body)
	}

	// validation: the form comes back with what was typed
	form := func(values map[string]string) url.Values {
		v := url.Values{"_csrf": {a.token("/admin/projects/add")}, "status": {"1"}}
		for key, value := range values {
			v.Set(key, value)
		}
		return v
	}
	for _, tt := range []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"missing name", map[string]string{"description": "留着的简介"}, "请填写项目名称"},
		{"unsafe address", map[string]string{"name": "x", "repo": "javascript:alert(1)"}, "源码地址要以"},
		{"site without scheme", map[string]string{"name": "x", "url": "example.com"}, "访问地址要以"},
		{"cover", map[string]string{"name": "x", "cover": "//evil.example/a.png"}, "封面图"},
		{"month", map[string]string{"name": "x", "started": "2026/9"}, "开始时间"},
		{"unpublished post", map[string]string{"name": "x", "post": "draft"}, "没有找到地址为「draft」的已发布文章"},
		{"too many highlights", map[string]string{"name": "x", "highlights": "1\n2\n3\n4\n5\n6\n7"}, "亮点最多 6 条"},
		{"long name", map[string]string{"name": strings.Repeat("长", 61)}, "名称太长了"},
	} {
		status, body, _ := a.post("/admin/projects/save", form(tt.values))
		if status != http.StatusUnprocessableEntity || !strings.Contains(body, tt.want) {
			t.Errorf("%s: status %d, message %q missing", tt.name, status, tt.want)
		}
		if tt.name == "missing name" && !strings.Contains(body, "留着的简介") {
			t.Errorf("a rejected form keeps what was typed")
		}
	}
	if status, _, _ := a.post("/admin/projects/save", url.Values{"name": {"x"}}); status != http.StatusForbidden {
		t.Errorf("save without CSRF token = %d, want 403", status)
	}

	// save: "owner/name" becomes the full address, lists are split and trimmed
	status, _, location := a.post("/admin/projects/save", form(map[string]string{
		"name": "proxyswitch", "repo": "whrss9527/proxyswitch", "description": "切换代理", "tech": "Go， CLI,go",
		"highlights": "- 一条命令切换\n\n  • 支持规则 ", "post": "hello", "started": "2026-08", "featured": "1",
	}))
	if status != http.StatusFound || !strings.HasPrefix(location, "/admin/projects?done=saved") {
		t.Fatalf("save = %d -> %q", status, location)
	}
	raw, err := os.ReadFile(filepath.Join(a.dataDir, "projects.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored []model.Project
	if err := json.Unmarshal(raw, &stored); err != nil || len(stored) != 1 {
		t.Fatalf("projects.json = %s", raw)
	}
	p := stored[0]
	if p.Repo != "https://github.com/whrss9527/proxyswitch" || strings.Join(p.Tech, "|") != "Go|CLI" || strings.Join(p.Highlights, "|") != "一条命令切换|支持规则" ||
		!p.Featured || p.Post != "hello" || p.Started != "2026-08" || p.Status != model.ProjectStatusActive {
		t.Errorf("stored project = %+v", p)
	}
	if !view.HasProjects() {
		t.Errorf("the navigation learns about the first project right away")
	}
	_, list, _ := a.get(location)
	if !strings.Contains(list, "已保存「proxyswitch」") {
		t.Errorf("the list confirms the save")
	}
	if status, sitemap, _ := a.get("/sitemap.xml"); status != http.StatusOK || !strings.Contains(sitemap, "/projects") {
		t.Errorf("the sitemap is regenerated after a save")
	}

	// the same repository cannot be added twice
	status, body, _ = a.post("/admin/projects/save", form(map[string]string{"name": "again", "repo": "https://github.com/whrss9527/proxyswitch/"}))
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, "已经在项目「proxyswitch」里了") {
		t.Errorf("duplicate repository = %d", status)
	}
	if _, body, _ = a.get("/admin/projects/github?repo=whrss9527/proxyswitch"); !strings.Contains(body, `"existing":"proxyswitch"`) {
		t.Errorf("the import warns about a repository that is already a project: %s", body)
	}

	// suggestions leave out what is already a project, forks, archived repositories and the profile README
	status, body, _ = a.get("/admin/projects/github/suggestions")
	if status != http.StatusOK || !strings.Contains(body, `"name":"goblog"`) {
		t.Errorf("suggestions = %d %s", status, body)
	}
	for _, unwanted := range []string{`"name":"proxyswitch"`, "GitHubPoster", "old-thing", `"name":"whrss9527"`} {
		if strings.Contains(body, unwanted) {
			t.Errorf("suggestions must not contain %s", unwanted)
		}
	}

	// edit
	status, body, _ = a.get("/admin/projects/add?id=1")
	if status != http.StatusOK || !strings.Contains(body, `value="proxyswitch"`) || !strings.Contains(body, "一条命令切换\n支持规则") ||
		!strings.Contains(body, `<option value="hello" selected>`) || !strings.Contains(body, "checked") {
		t.Errorf("edit form does not show the stored values")
	}
	status, _, _ = a.post("/admin/projects/save", form(map[string]string{"id": "1", "name": "ProxySwitch", "repo": p.Repo, "status": "2"}))
	if status != http.StatusFound {
		t.Fatalf("update = %d", status)
	}
	if _, front, _ := a.get("/projects"); !strings.Contains(front, ">ProxySwitch</a>") || !strings.Contains(front, "已完成") {
		t.Errorf("/projects shows the update")
	}
	if status, _, location = a.get("/admin/projects/add?id=99"); status != http.StatusFound || location != "/admin/projects" {
		t.Errorf("editing a project that is gone = %d -> %q", status, location)
	}

	// delete
	status, _, location = a.post("/admin/projects/delete/1", url.Values{"_csrf": {a.token("/admin/projects")}})
	if status != http.StatusFound || !strings.Contains(location, "done=deleted") {
		t.Fatalf("delete = %d -> %q", status, location)
	}
	if raw, _ := os.ReadFile(filepath.Join(a.dataDir, "projects.json")); strings.TrimSpace(string(raw)) != "[]" {
		t.Errorf("projects.json after delete = %s", raw)
	}
	if view.HasProjects() {
		t.Errorf("the navigation drops 项目 with the last project")
	}
	if status, _, _ = a.post("/admin/projects/delete/1", url.Values{"_csrf": {a.token("/admin/projects/add")}}); status != http.StatusNotFound {
		t.Errorf("deleting twice = %d, want 404", status)
	}
}

func TestAdminProjectsNeedLogin(t *testing.T) {
	h := newTestServer(t)
	for _, target := range []string{"/admin/projects", "/admin/projects/add", "/admin/projects/github?repo=a/b", "/admin/projects/github/suggestions"} {
		rec := get(t, h, target)
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/admin/login" {
			t.Errorf("anonymous %s = %d", target, rec.Code)
		}
	}
}

func toStrings(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, _ := item.(string)
		out = append(out, s)
	}
	return out
}
