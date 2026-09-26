package front

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"sort"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type TagHandler struct {
	TagRepo repository.TagRepository
	// Heatmap returns the posts-per-day data of the heatmap (a JSON array).
	Heatmap func() []byte
	config  *config.Config
}

func NewTagHandler(tagRepo repository.TagRepository, config *config.Config) *TagHandler {
	return &TagHandler{
		TagRepo: tagRepo,
		config:  config,
	}
}

func (handler *TagHandler) Tag(ctx *gin.Context) {
	heatmap := []byte("[]")
	if handler.Heatmap != nil {
		heatmap = handler.Heatmap()
	}
	if !json.Valid(heatmap) {
		slog.Error("heatmap data is not valid JSON, ignoring")
		heatmap = []byte("[]")
	}
	tags, err := handler.TagRepo.GetTags()
	if err != nil {
		slog.Error("get tags failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	// Hide unused tags and put the most used ones first.
	visible := tags[:0]
	for _, tag := range tags {
		if tag.Count > 0 {
			visible = append(visible, tag)
		}
	}
	tags = visible
	sort.SliceStable(tags, func(i, j int) bool { return tags[i].Count > tags[j].Count })

	data := make(map[string]any)
	data["nav"] = "tags"
	data["title"] = "标签"
	data["description"] = "了迹奇有没的博客标签"
	data["canonical"] = handler.config.App.Host + "/tags"
	data["tags"] = tags
	// template.JS keeps html/template from re-quoting the JSON document as a JS
	// string (which made JSON.parse return a string and broke the heatmap).
	// The payload is produced by encoding/json, which escapes <, > and &.
	data["heatmap"] = template.JS(heatmap)
	view.Render(data, ctx.Writer, "tags", handler.config.App)
}
