package front

import (
	"bytes"
	"html"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/handler/feed"
	"goblog/internal/pkg/md2html"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

// feedEntries is how many of the newest posts the feed carries, in full.
const feedEntries = 20

type FeedHandler struct {
	PostRepo repository.PostRepository
	host     string
	name     string
	// Description is the feed's subtitle (app.description).
	Description string

	doc cachedDoc
}

func NewFeedHandler(postRepo repository.PostRepository, host, name string) *FeedHandler {
	return &FeedHandler{
		PostRepo: postRepo,
		host:     host,
		name:     name,
	}
}

// GetFeedXml serves the Atom feed from memory. Readers that poll it get
// "304 Not Modified" until a post is published or changed.
func (h *FeedHandler) GetFeedXml(ctx *gin.Context) {
	h.doc.serve(ctx, "application/xml; charset=utf-8")
}

func (h *FeedHandler) GetRobotTxt(ctx *gin.Context) {
	file, err := os.ReadFile("./robots.txt")
	if err != nil {
		slog.Error("read robots.txt failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	// the sitemap address depends on the configured host, so it is added here
	if !bytes.Contains(bytes.ToLower(file), []byte("sitemap:")) && h.host != "" {
		file = append(bytes.TrimRight(file, "\n"), []byte("\n\nSitemap: "+strings.TrimRight(h.host, "/")+"/sitemap.xml\n")...)
	}
	ctx.Header("Content-Type", "text/plain; charset=utf-8")
	ctx.Header("Cache-Control", "public, max-age=3600")
	ctx.Writer.Write(file)
}

// GenerateFeedXml rebuilds the feed from the newest published posts. It is
// called at startup and whenever a post is saved or deleted.
func (h *FeedHandler) GenerateFeedXml() {
	posts, err := h.PostRepo.GetPostsWithContent()
	if err != nil {
		slog.Error("feed: get posts failed", "err", err)
		return
	}
	sort.SliceStable(posts, func(i, j int) bool { return posts[i].CreatedAt.After(posts[j].CreatedAt) })
	if len(posts) > feedEntries {
		posts = posts[:feedEntries]
	}

	var updated time.Time
	articles := make([]feed.Article, 0, len(posts))
	for _, post := range posts {
		for _, t := range []time.Time{post.CreatedAt, post.UpdatedAt} {
			if t.After(updated) {
				updated = t
			}
		}
		articles = append(articles, feed.Article{
			Title:     post.Title,
			Link:      h.host + "/posts/" + post.Identity,
			Id:        post.Identity,
			Published: post.CreatedAt,
			Created:   post.CreatedAt,
			Updated:   post.UpdatedAt,
			Content:   md2html.FeedHTML(post.Content, h.host),
			// the summary is HTML too: plain text, escaped
			Summary: html.EscapeString(view.Excerpt(post.Description, 200)),
		})
	}

	feedXml, err := feed.GenerateFeed(articles, feed.FeedConfig{Title: h.name, Host: h.host, Description: h.Description, Updated: updated})
	if err != nil {
		slog.Error("feed: generate failed", "err", err)
		return
	}
	h.doc.set([]byte(feedXml), updated)
}
