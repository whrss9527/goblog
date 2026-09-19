package front

import (
	"html/template"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/md2html"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/pkg/utils"
)

// renderOnServer returns the article HTML when it can (and should) be produced
// on the server. ok is false when the site is configured for client-side
// rendering, the content needs editor.md-only features, or rendering failed —
// the templates then fall back to the in-browser renderer.
func renderOnServer(appConf *config.AppConfig, markdown string) (template.HTML, bool) {
	if strings.EqualFold(appConf.MarkdownRender, "client") || md2html.NeedsClientRenderer(markdown) {
		return "", false
	}
	out, err := md2html.RenderArticle(markdown)
	if err != nil {
		slog.Error("server-side markdown render failed, falling back to client", "err", err)
		return "", false
	}
	// RenderArticle strips scripts, event handlers and script: URLs.
	return template.HTML(out), true
}

// RenderPostPreview shows an unpublished post with the public article
// template, for its author only (the admin routes call this): marked as a
// preview, not indexable, without likes, comments or view counting.
func RenderPostPreview(ctx *gin.Context, appConf *config.AppConfig, post model.Post, tags []model.Tag) {
	data := make(map[string]any)
	if html, ok := renderOnServer(appConf, post.Content); ok {
		data["content_html"] = html
	}
	if post.CreatedAt.IsZero() {
		post.CreatedAt = time.Now()
	}
	post.CreatedAt = post.CreatedAt.In(shanghai)
	if post.WordCount == 0 {
		post.WordCount = utils.GetTotalWords(post.Content)
	}
	description := view.Excerpt(post.Description, 160)
	if description == "" {
		description = view.Excerpt(post.Content, 160)
	}
	data["post"] = post
	data["tags"] = tags
	data["nav"] = "post"
	data["title"] = "[草稿预览] " + post.Title
	data["description"] = description
	data["identity"] = post.Identity
	data["pageId"] = "preview-" + post.Identity
	data["noindex"] = true
	data["preview"] = true
	data["outdated_years"] = 0
	ctx.Header("Cache-Control", "no-store")
	view.Render(data, ctx.Writer, "posts", appConf)
}
