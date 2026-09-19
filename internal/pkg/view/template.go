package view

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
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

		frontPages := []string{"index", "posts", "tags", "pages", "about", "archive", "reading", "404"}
		for _, page := range frontPages {
			tplPaths := []string{
				"tpl/default/layout.html",
				"tpl/default/" + page + ".html",
				"tpl/default/heatmap.html",
				"tpl/default/icons.html",
				"tpl/default/markdown.html",
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
			"tags/list",
			"books/list", "books/add",
		}
		for _, page := range adminPages {
			tplPath := "tpl/admin/" + page + ".html"
			t, err := template.ParseFiles(tplPath)
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
	data["year"] = time.Now().Year()
	if _, ok := data["title"]; !ok {
		data["title"] = appConf.Name
	}
	if _, ok := data["description"]; !ok {
		data["description"] = appConf.Name
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
	data["cdn"] = appConf.Cdn

	t, ok := adminTemplates[tpl]
	if !ok {
		slog.Error("admin template not found", "tpl", tpl)
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	if err := t.Execute(w, data); err != nil {
		slog.Error("render admin template failed", "tpl", tpl, "err", err)
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
