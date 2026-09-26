package view

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"goblog/internal/config"
	"goblog/internal/version"
)

var funcMap = template.FuncMap{
	"noescape": func(s string) template.HTML {
		return template.HTML(s)
	},
	"formatTime": func(t time.Time, layout string) string {
		return t.Format(layout)
	},
	// tagStyle scales a tag in the cloud by how many posts use it (13px–25px).
	"tagStyle": func(count int) template.CSS {
		steps := count - 1
		if steps < 0 {
			steps = 0
		}
		if steps > 6 {
			steps = 6
		}
		return template.CSS(fmt.Sprintf("font-size:%dpx", 13+steps*2))
	},
	"inc":            func(i int) int { return i + 1 },
	"dict":           Dict,
	"pathEscape":     url.PathEscape,
	"asset":          AssetURL,
	"excerpt":        Excerpt,
	"readingMinutes": ReadingMinutes,
	"humanCount":     HumanCount,
}

// assetHashes caches content fingerprints of local static files, keyed by URL path.
var assetHashes sync.Map

// AssetURL appends a content fingerprint (?v=<hash>) to a local /static/ URL so
// browsers and CDNs can cache it aggressively while still picking up new
// releases immediately. Unknown or unreadable files are returned unchanged.
func AssetURL(urlPath string) string {
	if v, ok := assetHashes.Load(urlPath); ok {
		return v.(string)
	}
	result := urlPath
	if strings.HasPrefix(urlPath, "/static/") {
		if data, err := os.ReadFile(strings.TrimPrefix(urlPath, "/")); err == nil {
			sum := sha1.Sum(data)
			result = urlPath + "?v=" + hex.EncodeToString(sum[:])[:10]
		} else {
			slog.Warn("asset fingerprint failed", "path", urlPath, "err", err)
		}
	}
	assetHashes.Store(urlPath, result)
	return result
}

var (
	frontTemplates map[string]*template.Template
	adminTemplates map[string]*template.Template
	introTemplate  *template.Template
	tplOnce        sync.Once
)

func InitTemplates() {
	tplOnce.Do(func() {
		frontTemplates = make(map[string]*template.Template)
		adminTemplates = make(map[string]*template.Template)

		frontPages := []string{"index", "posts", "tags", "pages", "about", "archive", "reading", "projects", "stats", "404", "offline"}
		for _, page := range frontPages {
			tplPaths := []string{
				"tpl/default/layout.html",
				"tpl/default/" + page + ".html",
				"tpl/default/heatmap.html",
				"tpl/default/icons.html",
				"tpl/default/markdown.html",
				"tpl/default/project-card.html",
			}
			t, err := template.New("layout.html").Funcs(funcMap).ParseFiles(tplPaths...)
			if err != nil {
				slog.Error("parse front template failed", "page", page, "err", err)
				continue
			}
			frontTemplates[page] = t
		}

		adminPages := []string{
			"login", "register", "401", "404", "500", "password",
			"posts/list", "posts/add",
			"pages/list", "pages/add",
			"categories/list", "categories/add",
			"tags/list", "tags/edit",
			"books/list", "books/add",
			"projects/list", "projects/add",
		}
		for _, page := range adminPages {
			tplPath := "tpl/admin/" + page + ".html"
			// every admin page shares the chrome defined in _partials.html
			t, err := template.New(filepath.Base(tplPath)).Funcs(funcMap).ParseFiles(tplPath, "tpl/admin/_partials.html")
			if err != nil {
				slog.Error("parse admin template failed", "page", page, "err", err)
				continue
			}
			adminTemplates[page] = t
		}

		var err error
		introTemplate, err = template.ParseFiles("tpl/intro/index.html")
		if err != nil {
			slog.Error("parse intro template failed", "err", err)
		}

		slog.Info("templates initialized", "front", len(frontTemplates), "admin", len(adminTemplates))
	})
}

// siteSince is the publication time of the oldest post, used for the footer's
// "running for N days". Stored as UnixNano so it can be read without locking.
var siteSince atomic.Int64

// SetSiteSince records when the blog started.
func SetSiteSince(t time.Time) {
	if !t.IsZero() {
		siteSince.Store(t.UnixNano())
	}
}

// projectCount is the number of projects: the navigation only links to
// /projects once there is something to show there.
var projectCount atomic.Int64

// SetProjectCount records how many projects exist.
func SetProjectCount(n int) {
	projectCount.Store(int64(n))
}

// contentProblem says why the admin cannot save at the moment ("" when it
// can); the router points it at the repository.
var contentProblem atomic.Pointer[func() string]

// SetContentProblemSource tells the admin pages where to ask whether saving is
// refused (filestore.FileRepository.ContentProblem).
func SetContentProblemSource(fn func() string) {
	contentProblem.Store(&fn)
}

// HasProjects reports whether at least one project exists.
func HasProjects() bool {
	return projectCount.Load() > 0
}

func siteDays(now time.Time) int {
	since := siteSince.Load()
	if since == 0 {
		return 0
	}
	return int(now.Sub(time.Unix(0, since)).Hours()/24) + 1
}

// Render renders a front template with HTTP 200.
func Render(data map[string]any, w http.ResponseWriter, tpl string, appConf *config.AppConfig) {
	RenderStatus(http.StatusOK, data, w, tpl, appConf)
}

// RenderStatus renders a front template with the given HTTP status code. The
// template is executed into a buffer first so a failing template never leaves
// a half-written page behind.
func RenderStatus(status int, data map[string]any, w http.ResponseWriter, tpl string, appConf *config.AppConfig) {
	data["name"] = appConf.Name
	data["cdn"] = appConf.Cdn
	data["host"] = appConf.Host
	data["version"] = version.Version
	data["pwa"] = appConf.PWAEnabled()
	now := time.Now()
	data["year"] = now.Year()
	data["site_days"] = siteDays(now)
	data["nav_projects"] = HasProjects()
	if _, ok := data["title"]; !ok {
		data["title"] = appConf.Name
	}
	if _, ok := data["description"]; !ok {
		data["description"] = appConf.Name
		if appConf.Description != "" {
			data["description"] = appConf.Description
		}
	}
	// <title>: "page | site", and just the site (plus its tagline) on the home page
	if title, _ := data["title"].(string); title != "" && title != appConf.Name {
		data["page_title"] = title + " | " + appConf.Name
	} else if appConf.Description != "" {
		data["page_title"] = appConf.Name + " - " + appConf.Description
	} else {
		data["page_title"] = appConf.Name
	}
	if _, ok := data["og_image"]; !ok {
		data["og_image"] = SiteLogo(appConf.Host, appConf.Cdn)
	}
	// Keys printed unconditionally by the layout must exist, otherwise
	// html/template prints "<no value>".
	for _, key := range []string{"keyword", "nav"} {
		if _, ok := data[key]; !ok {
			data[key] = ""
		}
	}

	t, ok := frontTemplates[tpl]
	if !ok {
		slog.Error("front template not found", "tpl", tpl)
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		slog.Error("render front template failed", "tpl", tpl, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("write response failed", "tpl", tpl, "err", err)
	}
}

func AdminRenderWithCSRF(data map[string]any, w http.ResponseWriter, tpl string, appConf *config.AppConfig, csrfToken string) {
	data["csrf_token"] = csrfToken
	AdminRender(data, w, tpl, appConf)
}

func AdminRender(data map[string]any, w http.ResponseWriter, tpl string, appConf *config.AppConfig) {
	AdminRenderStatus(http.StatusOK, data, w, tpl, appConf)
}

// AdminRenderStatus renders an admin template with the given status code. Like
// the front renderer it executes into a buffer first, so a template error
// never leaves half a page behind.
func AdminRenderStatus(status int, data map[string]any, w http.ResponseWriter, tpl string, appConf *config.AppConfig) {
	// prefixed: handlers use plain keys such as "name" and "year" for form values
	data["cdn"] = appConf.Cdn
	data["site_name"] = appConf.Name
	data["site_version"] = version.Version
	data["this_year"] = time.Now().Year()
	data["account_warning"] = appConf.AccountInContentRepo()
	data["content_problem"] = ""
	if fn := contentProblem.Load(); fn != nil {
		data["content_problem"] = (*fn)()
	}

	t, ok := adminTemplates[tpl]
	if !ok {
		slog.Error("admin template not found", "tpl", tpl)
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		slog.Error("render admin template failed", "tpl", tpl, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("write response failed", "tpl", tpl, "err", err)
	}
}

func IntroRender(data map[string]any, w http.ResponseWriter, appConf *config.AppConfig) {
	data["cdn"] = appConf.Cdn
	if introTemplate == nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	if err := introTemplate.Execute(w, data); err != nil {
		slog.Error("render intro template failed", "err", err)
	}
}
