package admin

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/model"
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

func (h *TagHandler) TagList(ctx *gin.Context) {
	tags, err := h.TagRepo.GetTags()
	if err != nil {
		slog.Error("get tags failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	data := make(map[string]any)
	data["tags"] = tags
	view.AdminRender(data, ctx.Writer, "tags/list", h.config.App)
}

func (h *TagHandler) TagSave(ctx *gin.Context) {
	var tag model.Tag
	tag.Name = ctx.Request.FormValue("name")
	_, err := h.TagRepo.AddTag(tag)
	if err != nil {
		data := make(map[string]any)
		data["msg"] = "添加标签失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/tags/list", http.StatusFound)
}
