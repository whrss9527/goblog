package md2html

import (
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/russross/blackfriday/v2"
)

// Md2Html renders Markdown for the feed. It is the same HTML the site shows
// (see RenderArticle) plus inline styles, because feed readers drop stylesheets.
func Md2Html(markdown []byte) string {
	rendered, err := RenderArticle(string(markdown))
	if err != nil {
		slog.Error("md2html render failed", "err", err)
		rendered = string(blackfriday.Run(markdown))
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rendered))
	if err != nil {
		slog.Error("md2html parse failed", "err", err)
		return rendered
	}
	body := doc.Find("body")

	body.Find("p, h1, h2, h3, h4, h5, h6, ul, ol, table, pre").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("style", "max-width: 1300px; display: block; margin-left: auto; margin-right: auto; text-align: left;")
	})
	// no display:block here: list items would lose their bullets and numbers
	body.Find("li").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("style", "max-width: 1300px; margin-left: auto; margin-right: auto; text-align: left;")
	})

	body.Find("img").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("style", "max-width: 500px; max-height: 500px; display: block; margin-left: auto; margin-right: auto;")
	})

	body.Find("code").Each(func(i int, s *goquery.Selection) {
		if goquery.NodeName(s.Parent()) == "pre" {
			s.SetAttr("style", "display: block; white-space: pre; tab-size: 4; border: 1px solid #ccc; padding: 6px 10px; color: #333; background-color: #f9f9f9; border-radius: 3px;")
		} else {
			s.ReplaceWithHtml("<b>" + escapeText(s.Text()) + "</b>")
		}
	})

	out, err := body.Html()
	if err != nil {
		slog.Error("md2html output failed", "err", err)
		return rendered
	}
	return strings.ReplaceAll(out, "/>", ">")
}

var textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func escapeText(s string) string { return textEscaper.Replace(s) }
