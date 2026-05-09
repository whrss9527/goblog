package front

import (
	"encoding/xml"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/repository"
)

type SitemapHandler struct {
	PostRepo repository.PostRepository
	host     string
}

func NewSitemapHandler(postRepo repository.PostRepository, host string) *SitemapHandler {
	return &SitemapHandler{PostRepo: postRepo, host: host}
}

type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	Lastmod    string `xml:"lastmod,omitempty"`
	Changefreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func (h *SitemapHandler) GenerateSitemap() {
	host := h.host
	posts, err := h.PostRepo.GetPostsArchive()
	if err != nil {
		slog.Error("sitemap: get posts failed", "err", err)
		return
	}

	urls := []sitemapURL{
		{Loc: host, Changefreq: "daily", Priority: "1.0"},
		{Loc: host + "/tags", Changefreq: "weekly", Priority: "0.8"},
		{Loc: host + "/archive", Changefreq: "weekly", Priority: "0.8"},
		{Loc: host + "/pages/about", Changefreq: "monthly", Priority: "0.6"},
	}

	for _, post := range posts {
		lastmod := post.UpdatedAt
		if lastmod.IsZero() {
			lastmod = post.CreatedAt
		}
		urls = append(urls, sitemapURL{
			Loc:        host + "/posts/" + post.Identity,
			Lastmod:    lastmod.Format(time.DateOnly),
			Changefreq: "monthly",
			Priority:   "0.7",
		})
	}

	sitemap := urlset{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	data, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		slog.Error("sitemap: marshal failed", "err", err)
		return
	}

	content := append([]byte(xml.Header), data...)
	if err := os.WriteFile("./sitemap.xml", content, 0644); err != nil {
		slog.Error("sitemap: write file failed", "err", err)
	}
}

func (h *SitemapHandler) GetSitemap(ctx *gin.Context) {
	file, err := os.ReadFile("./sitemap.xml")
	if err != nil {
		slog.Error("read sitemap.xml failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	ctx.Header("Content-Type", "application/xml; charset=utf-8")
	ctx.Writer.Write(file)
}
