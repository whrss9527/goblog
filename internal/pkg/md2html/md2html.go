package md2html

import (
	"bytes"
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/russross/blackfriday/v2"
)

func Md2Html(markdown []byte) string {
	html := blackfriday.Run(markdown)

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		slog.Error("md2html parse failed", "err", err)
		return string(html)
	}

	doc.Find("p, h1, h2, h3, h4, h5, h6, ul, ol, li, table, pre").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("style", "max-width: 1300px; display: block; margin-left: auto; margin-right: auto; text-align: left;")
	})

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("style", "max-width: 500px; max-height: 500px; display: block; margin-left: auto; margin-right: auto;")
	})

	doc.Find("code").Each(func(i int, s *goquery.Selection) {
		if goquery.NodeName(s.Parent()) == "pre" {
			s.SetAttr("style", "display: block; white-space: pre; border: 1px solid #ccc; padding: 6px 10px; color: #333; background-color: #f9f9f9; border-radius: 3px;")
		} else {
			s.ReplaceWithHtml("<b>" + s.Text() + "</b>")
		}
	})

	modifiedHtml, err := doc.Html()
	if err != nil {
		slog.Error("md2html output failed", "err", err)
		return string(html)
	}

	modifiedHtml = strings.Replace(modifiedHtml, "/>", ">", -1)
	return modifiedHtml
}
