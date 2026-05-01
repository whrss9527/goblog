package front

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type ArchiveHandler struct {
	PostRepo repository.PostRepository
	config   *config.Config
}

func NewArchiveHandler(postRepo repository.PostRepository, config *config.Config) *ArchiveHandler {
	return &ArchiveHandler{
		PostRepo: postRepo,
		config:   config,
	}
}

type ArchiveYear struct {
	Year  int
	Posts []*model.Post
}

func (h *ArchiveHandler) Archive(ctx *gin.Context) {
	posts, err := h.PostRepo.GetPostsArchive()
	if err != nil {
		slog.Error("get archive posts failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}

	var archives []ArchiveYear
	var current *ArchiveYear
	for _, post := range posts {
		year := post.CreatedAt.Year()
		if current == nil || current.Year != year {
			archives = append(archives, ArchiveYear{Year: year})
			current = &archives[len(archives)-1]
		}
		current.Posts = append(current.Posts, post)
	}

	data := make(map[string]any)
	data["title"] = "归档"
	data["description"] = "文章归档"
	data["archives"] = archives
	data["total"] = len(posts)
	view.Render(data, ctx.Writer, "archive", h.config.App)
}
