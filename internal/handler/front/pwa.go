package front

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/view"
	"goblog/internal/version"
)

const (
	serviceWorkerSource = "static/js/sw.js"
	offlinePath         = "/offline"
)

// precachedAssets is what an installed site needs to draw any page offline.
var precachedAssets = []string{
	"/static/css/style.css",
	"/static/js/enhance.js",
	"/static/js/article.js",
	"/static/js/search.js",
	"/static/icons/icon-192.png",
	"/favicon.ico",
}

// retireWorker replaces the service worker when app.pwa is switched off:
// browsers that still have the old worker pick this one up on their next
// update check, it drops every cache and unregisters itself.
const retireWorker = `self.addEventListener('install', function () { self.skipWaiting(); });
self.addEventListener('activate', function (event) {
    event.waitUntil(caches.keys().then(function (names) {
        return Promise.all(names.filter(function (n) { return n.indexOf('goblog-') === 0; }).map(function (n) { return caches.delete(n); }));
    }).then(function () { return self.registration.unregister(); }));
});
`

// PWAHandler serves what makes the blog installable and readable offline:
// the web app manifest, the service worker and the offline fallback page.
type PWAHandler struct {
	config *config.Config
}

func NewPWAHandler(config *config.Config) *PWAHandler {
	return &PWAHandler{config: config}
}

// Manifest is generated so that name and description follow the site config.
func (h *PWAHandler) Manifest(ctx *gin.Context) {
	app := h.config.App
	icon := func(file, sizes, purpose string) map[string]string {
		return map[string]string{"src": view.AssetURL("/static/icons/" + file), "sizes": sizes, "type": "image/png", "purpose": purpose}
	}
	manifest := map[string]any{
		"id":               "/",
		"name":             app.Name,
		"short_name":       shortName(app.Name),
		"lang":             "zh-CN",
		"dir":              "ltr",
		"start_url":        "/",
		"scope":            "/",
		"display":          "standalone",
		"background_color": "#ffffff",
		"theme_color":      "#ffffff",
		"categories":       []string{"blog", "news", "education"},
		"icons": []map[string]string{
			icon("icon-192.png", "192x192", "any"),
			icon("icon-512.png", "512x512", "any"),
			icon("icon-512.png", "512x512", "maskable"),
		},
		"shortcuts": []map[string]any{
			{"name": "归档", "url": "/archive"},
			{"name": "随便看看", "url": "/random"},
			{"name": "博客数据", "url": "/stats"},
		},
	}
	if app.Description != "" {
		manifest["description"] = app.Description
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		slog.Error("marshal manifest failed", "err", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Header("Cache-Control", "public, max-age=3600")
	ctx.Data(http.StatusOK, "application/manifest+json; charset=utf-8", raw)
}

// shortName keeps the home screen label within the ~12 characters launchers show.
func shortName(name string) string {
	if utf8.RuneCountInString(name) <= 12 {
		return name
	}
	return string([]rune(name)[:12])
}

// ServiceWorker serves static/js/sw.js from the site root (a worker only
// controls pages below its own path) and tells it what to precache. The
// version changes whenever a precached file changes, which makes browsers
// install the new worker.
func (h *PWAHandler) ServiceWorker(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Service-Worker-Allowed", "/")
	if !h.config.App.PWAEnabled() {
		ctx.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(retireWorker))
		return
	}
	source, err := os.ReadFile(serviceWorkerSource)
	if err != nil {
		slog.Error("read service worker failed", "err", err)
		ctx.Status(http.StatusNotFound)
		return
	}
	precache := make([]string, 0, len(precachedAssets))
	for _, path := range precachedAssets {
		precache = append(precache, view.AssetURL(path))
	}
	sum := sha1.Sum([]byte(strings.Join(precache, "\n")))
	settings, err := json.Marshal(map[string]any{
		"version":  version.Version + "-" + hex.EncodeToString(sum[:4]),
		"precache": precache,
		"offline":  offlinePath,
	})
	if err != nil {
		slog.Error("marshal service worker settings failed", "err", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	body := append([]byte("self.__GOBLOG = "+string(settings)+";\n"), source...)
	ctx.Data(http.StatusOK, "text/javascript; charset=utf-8", body)
}

// Offline is the page the service worker shows for addresses it has no copy
// of. It lists the articles that are available without a connection.
func (h *PWAHandler) Offline(ctx *gin.Context) {
	data := map[string]any{
		"title":       "离线",
		"description": "当前没有网络连接",
		"noindex":     true,
	}
	ctx.Header("Cache-Control", "no-cache")
	view.Render(data, ctx.Writer, "offline", h.config.App)
}
