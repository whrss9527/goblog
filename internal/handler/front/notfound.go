package front

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/view"
)

// RenderNotFound renders the themed 404 page with a real 404 status code so
// crawlers don't index missing pages as soft-404s.
func RenderNotFound(ctx *gin.Context, appConf *config.AppConfig) {
	data := map[string]any{
		"title":       "页面不存在",
		"description": "404 - 页面不存在",
		"noindex":     true,
	}
	view.RenderStatus(http.StatusNotFound, data, ctx.Writer, "404", appConf)
}

// NotFound returns a gin handler suitable for router.NoRoute.
func NotFound(appConf *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		RenderNotFound(ctx, appConf)
	}
}
