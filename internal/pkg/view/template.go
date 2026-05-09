package view

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"goblog/internal/config"
)

var funcMap = template.FuncMap{
	"noescape": func(s string) template.HTML {
		return template.HTML(s)
	},
	"formatTime": func(t time.Time, layout string) string {
		return t.Format(layout)
	},
	"tagStyle": func(count int) string {
		sizes := [...]string{"12px", "12px", "15px", "20px", "25px", "30px", "35px", "40px", "45px", "50px", "55px"}
		if count >= len(sizes) {
			return "font-size:60px"
		}
		return fmt.Sprintf("font-size:%s", sizes[count])
	},
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

func Render(data map[string]any, w http.ResponseWriter, tpl string, appConf *config.AppConfig) {
	data["name"] = appConf.Name
	data["cdn"] = appConf.Cdn
	if _, ok := data["title"]; !ok {
		data["title"] = appConf.Name
	}
	if _, ok := data["description"]; !ok {
		data["description"] = appConf.Name
	}

	t, ok := frontTemplates[tpl]
	if !ok {
		slog.Error("front template not found", "tpl", tpl)
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	if err := t.Execute(w, data); err != nil {
		slog.Error("render front template failed", "tpl", tpl, "err", err)
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
