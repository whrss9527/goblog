package front

import (
	"encoding/xml"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/repository"
)

type SitemapHandler struct {
	PostRepo repository.PostRepository
	// Pages adds every page (about, …); without it only /pages/about is listed.
	Pages repository.PageRepository
	// Projects adds /projects once there is at least one project (optional).
	Projects repository.ProjectRepository
	host     string

	doc cachedDoc
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

// GenerateSitemap rebuilds the sitemap. It is called at startup and whenever
// posts or projects change.
func (h *SitemapHandler) GenerateSitemap() {
	host := strings.TrimRight(h.host, "/")
	posts, err := h.PostRepo.GetPostsArchive()
	if err != nil {
		slog.Error("sitemap: get posts failed", "err", err)
		return
	}
	var latest time.Time
	for _, post := range posts {
		if t := lastModified(post.CreatedAt, post.UpdatedAt); t.After(latest) {
			latest = t
		}
	}
	date := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format(time.DateOnly)
	}

	// lists change whenever a post does
	urls := []sitemapURL{
		{Loc: host + "/", Lastmod: date(latest), Changefreq: "daily", Priority: "1.0"}, // as in the home page's canonical
		{Loc: host + "/tags", Lastmod: date(latest), Changefreq: "weekly", Priority: "0.8"},
		{Loc: host + "/archive", Lastmod: date(latest), Changefreq: "weekly", Priority: "0.8"},
	}
	if h.Projects != nil {
		if projects, err := h.Projects.GetProjects(); err == nil && len(projects) > 0 {
			var changed time.Time
			for _, p := range projects {
				if p.UpdatedAt.After(changed) {
					changed = p.UpdatedAt
				}
			}
			urls = append(urls, sitemapURL{Loc: host + "/projects", Lastmod: date(changed), Changefreq: "weekly", Priority: "0.7"})
		}
	}
	urls = append(urls,
		sitemapURL{Loc: host + "/reading", Changefreq: "weekly", Priority: "0.6"},
		sitemapURL{Loc: host + "/stats", Changefreq: "weekly", Priority: "0.5"},
	)
	pageIds := []string{"about"}
	if h.Pages != nil {
		if pages, err := h.Pages.GetPages(repository.PageParams{}); err == nil {
			pageIds = pageIds[:0]
			for _, page := range pages {
				pageIds = append(pageIds, page.Id)
			}
		}
	}
	for _, id := range pageIds {
		urls = append(urls, sitemapURL{Loc: host + "/pages/" + url.PathEscape(id), Changefreq: "monthly", Priority: "0.6"})
	}

	for _, post := range posts {
		urls = append(urls, sitemapURL{
			Loc:        host + "/posts/" + url.PathEscape(post.Identity),
			Lastmod:    date(lastModified(post.CreatedAt, post.UpdatedAt)),
			Changefreq: "monthly",
			Priority:   "0.7",
		})
	}

	data, err := xml.MarshalIndent(urlset{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls}, "", "  ")
	if err != nil {
		slog.Error("sitemap: marshal failed", "err", err)
		return
	}
	h.doc.set(append([]byte(xml.Header), data...), latest)
}

// lastModified is the later of a post's two dates (either may be missing).
func lastModified(created, updated time.Time) time.Time {
	if updated.After(created) {
		return updated
	}
	return created
}

func (h *SitemapHandler) GetSitemap(ctx *gin.Context) {
	h.doc.serve(ctx, "application/xml; charset=utf-8")
}
