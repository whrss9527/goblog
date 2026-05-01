package front

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type TagHandler struct {
	TagRepo repository.TagRepository
	config  *config.Config
}

func NewTagHandler(tagRepo repository.TagRepository, config *config.Config) *TagHandler {
	return &TagHandler{
		TagRepo: tagRepo,
		config:  config,
	}
}

func (handler *TagHandler) Tag(ctx *gin.Context) {
	heatmap, err := os.ReadFile("./heatmap.txt")
	if err != nil {
		slog.Error("read heatmap failed", "err", err)
		heatmap = []byte("[]")
	}
	tags, err := handler.TagRepo.GetTags()
	if err != nil {
		slog.Error("get tags failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	data := make(map[string]any)
	data["title"] = "标签"
	data["description"] = "了迹奇有没的博客标签"
	data["tags"] = tags
	data["heatmap"] = string(heatmap)
	view.Render(data, ctx.Writer, "tags", handler.config.App)
}
