package front

import (
	"log/slog"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"goblog/internal/handler/feed"
	"goblog/internal/pkg/md2html"
	"goblog/internal/repository"
)

type FeedHandler struct {
	PostRepo repository.PostRepository
	host     string
}

func NewFeedHandler(postRepo repository.PostRepository, host string) *FeedHandler {
	return &FeedHandler{
		PostRepo: postRepo,
		host:     host,
	}
}

func (h *FeedHandler) GetFeedXml(ctx *gin.Context) {
	file, err := os.ReadFile("./feed.xml")
	if err != nil {
		slog.Error("read feed.xml failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	ctx.Header("Content-Type", "application/xml; charset=utf-8")
	ctx.Writer.Write(file)
}

func (h *FeedHandler) GetRobotTxt(ctx *gin.Context) {
	file, err := os.ReadFile("./robot.txt")
	if err != nil {
		slog.Error("read robot.txt failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	ctx.Header("Content-Type", "text/plain; charset=utf-8")
	ctx.Writer.Write(file)
}

func (h *FeedHandler) GenerateFeedXml() {
	posts, err := h.PostRepo.GetPostsWithContent()
	if err != nil {
		slog.Error("feed: get posts failed", "err", err)
		return
	}

	articles := make([]feed.Article, 0, len(posts))
	for _, post := range posts {
		articles = append(articles, feed.Article{
			Title:     post.Title,
			Link:      h.host + "/posts/" + post.Identity,
			Id:        post.Identity,
			Published: post.CreatedAt,
			Created:   post.CreatedAt,
			Updated:   post.UpdatedAt,
			Content:   md2html.Md2Html([]byte(removeUnwantedChars(post.Content))),
			Summary:   post.Description,
		})
	}

	feedXml, err := feed.GenerateFeed(articles)
	if err != nil {
		slog.Error("feed: generate failed", "err", err)
		return
	}
	if err := os.WriteFile("./feed.xml", []byte(feedXml), 0644); err != nil {
		slog.Error("feed: write file failed", "err", err)
	}
}

func removeUnwantedChars(s string) string {
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
