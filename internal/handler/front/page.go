package front

import (
	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type PageHandler struct {
	PageRepo repository.PageRepository
	config   *config.Config
}

func NewPageHandler(pageRepo repository.PageRepository, config *config.Config) *PageHandler {
	return &PageHandler{
		PageRepo: pageRepo,
		config:   config,
	}
}

func (h *PageHandler) Page(ctx *gin.Context) {
	pageId := ctx.Param("id")
	page, err := h.PageRepo.GetPage(pageId)
	if err != nil {
		view.Render(make(map[string]any), ctx.Writer, "404", h.config.App)
		return
	}
	data := make(map[string]any)
	data["title"] = page.Title
	data["description"] = page.Title
	data["page"] = page
	data["page_id"] = "page-" + pageId
	data["canonical"] = h.config.App.Host + "/pages/" + pageId
	view.Render(data, ctx.Writer, "pages", h.config.App)
}
