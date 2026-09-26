package md2html

import (
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/russross/blackfriday/v2"
)

// FeedHTML renders Markdown for the feed: the same HTML the site shows (see
// RenderArticle), with site-relative links and images made absolute against
// base — readers resolve relative addresses against the feed, or not at all.
// Readers bring their own styles for paragraphs, lists and code, so there are
// no inline styles (they used to be a third of the feed's size).
func FeedHTML(markdown, base string) string {
	rendered, err := RenderArticle(markdown)
	if err != nil {
		slog.Error("md2html render failed", "err", err)
		rendered = string(blackfriday.Run([]byte(markdown)))
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rendered))
	if err != nil {
		slog.Error("md2html parse failed", "err", err)
		return rendered
	}
	body := doc.Find("body")
	base = strings.TrimRight(base, "/")
	absolute := func(attr string) func(int, *goquery.Selection) {
		return func(_ int, s *goquery.Selection) {
			if v, _ := s.Attr(attr); strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "//") {
				s.SetAttr(attr, base+v)
			}
		}
	}
	body.Find("a[href]").Each(absolute("href"))
	body.Find("img[src]").Each(absolute("src"))

	out, err := body.Html()
	if err != nil {
		slog.Error("md2html output failed", "err", err)
		return rendered
	}
	return strings.ReplaceAll(out, "/>", ">")
}
