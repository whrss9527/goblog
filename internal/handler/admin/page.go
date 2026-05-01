package admin

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/model"
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

func (h *PageHandler) PageList(ctx *gin.Context) {
	perPage, _ := strconv.Atoi(ctx.Request.URL.Query().Get("per_page"))
	page, _ := strconv.Atoi(ctx.Request.URL.Query().Get("page"))
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 1 {
		page = 1
	}
	pages, err := h.PageRepo.GetPages(repository.PageParams{
		PerPage: perPage,
		Page:    page,
	})
	if err != nil {
		slog.Error("get pages failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	data := make(map[string]any)
	data["pages"] = pages
	data["page"] = page
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "pages/list", h.config.App)
}

func (h *PageHandler) PageAdd(ctx *gin.Context) {
	data := make(map[string]any)
	ident := ctx.Request.FormValue("page_id")
	var page model.Page
	if len(ident) > 0 {
		var err error
		page, err = h.PageRepo.GetPage(ident)
		if err != nil {
			slog.Error("get page failed", "err", err)
			ctx.Writer.WriteHeader(500)
			return
		}
	}
	exists, err := h.PageRepo.PageExist(page.Id)
	if err != nil {
		slog.Error("check page exist failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	if exists {
		data["id"] = page.Id
		data["title"] = page.Title
		data["content"] = page.Content
	}

	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "pages/add", h.config.App)
}

func (h *PageHandler) PageDelete(ctx *gin.Context) {
	var page model.Page
	page.Id = ctx.Param("id")
	_, err := h.PageRepo.PageDelete(page)
	if err != nil {
		data := make(map[string]any)
		data["msg"] = "删除失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/pages", http.StatusFound)
}

func (h *PageHandler) PageSave(ctx *gin.Context) {
	var page model.Page
	page.Id = ctx.Request.FormValue("id")
	page.Title = ctx.Request.FormValue("title")
	page.Content = ctx.Request.FormValue("content")
	_, err := h.PageRepo.PageSave(page)
	if err != nil {
		data := make(map[string]any)
		data["msg"] = "添加或修改失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/pages", http.StatusFound)
}
