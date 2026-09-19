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
		RenderNotFound(ctx, h.config.App)
		return
	}
	data := make(map[string]any)
	if pageId == "about" {
		data["nav"] = "about"
	}
	data["title"] = page.Title
	description := view.Excerpt(page.Content, 160)
	if description == "" {
		description = page.Title
	}
	data["description"] = description
	data["page"] = page
	if html, ok := renderOnServer(h.config.App, page.Content); ok {
		data["content_html"] = html
	}
	data["page_id"] = "page-" + pageId
	data["canonical"] = h.config.App.Host + "/pages/" + pageId
	view.Render(data, ctx.Writer, "pages", h.config.App)
}
