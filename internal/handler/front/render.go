package front

import (
	"html/template"
	"log/slog"
	"strings"

	"goblog/internal/config"
	"goblog/internal/pkg/md2html"
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
