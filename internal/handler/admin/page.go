package admin

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/handler/front"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type PageHandler struct {
	PageRepo repository.PageRepository
	// Sitemap is regenerated after a page is added or removed (optional).
	Sitemap *front.SitemapHandler
	config  *config.Config
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
	ident := ctx.Request.FormValue("page_id")
	var page model.Page
	if len(ident) > 0 {
		var err error
		page, err = h.PageRepo.GetPage(ident)
		if err != nil {
			slog.Error("get page failed", "err", err)
			data := map[string]any{"msg": "没有找到这个页面，它可能已经被删除"}
			view.AdminRenderStatus(http.StatusNotFound, data, ctx.Writer, "401", h.config.App)
			return
		}
	}
	h.renderEditor(ctx, http.StatusOK, page.Id, page, "")
}

// renderEditor shows the page editor. savedId is the id of the stored page
// ("" while it is being created); page carries what should be in the form.
func (h *PageHandler) renderEditor(ctx *gin.Context, status int, savedId string, page model.Page, problem string) {
	data := make(map[string]any)
	data["id"] = savedId
	data["page_id"] = page.Id
	data["title"] = page.Title
	data["content"] = page.Content
	data["error"] = problem
	data["csrf_token"], _ = ctx.Get("csrf_token")

	var taken []string
	if pages, err := h.PageRepo.GetPages(repository.PageParams{PerPage: 1000, Page: 1}); err == nil {
		for _, p := range pages {
			if p.Id != savedId {
				taken = append(taken, p.Id)
			}
		}
	}
	data["slugs_json"] = toJSON(taken)
	view.AdminRenderStatus(status, data, ctx.Writer, "pages/add", h.config.App)
}

func (h *PageHandler) PageDelete(ctx *gin.Context) {
	var page model.Page
	page.Id = ctx.Param("id")
	_, err := h.PageRepo.PageDelete(page)
	if err != nil {
		data := make(map[string]any)
		slog.Error("delete page failed", "err", err)
		data["msg"] = failureMessage(err, "删除失败，请重试")
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	h.refreshSitemap()
	http.Redirect(ctx.Writer, ctx.Request, "/admin/pages", http.StatusFound)
}

func (h *PageHandler) PageSave(ctx *gin.Context) {
	// "id" is the stored page being edited (empty for a new one), "page_id" the
	// address typed into the form. The form used to send only page_id while
	// this handler read only id, so every save ended up in a page without id.
	savedId := strings.TrimSpace(ctx.Request.FormValue("id"))
	var page model.Page
	page.Id = strings.TrimSpace(ctx.Request.FormValue("page_id"))
	page.Title = strings.TrimSpace(ctx.Request.FormValue("title"))
	page.Content = ctx.Request.FormValue("content")

	problem := ""
	switch {
	case page.Title == "":
		problem = "请填写标题"
	case savedId != "":
		// pages are linked from the navigation by id: editing never renames
		if exists, _ := h.PageRepo.PageExist(savedId); !exists {
			problem = "这个页面已经不存在了，请回到列表重新创建"
		}
		page.Id = savedId
	default:
		problem = checkSlug(page.Id)
		if problem == "" {
			if exists, _ := h.PageRepo.PageExist(page.Id); exists {
				problem = "这个地址已经被另一个页面占用了，换一个吧"
			}
		}
	}
	if problem != "" {
		h.renderEditor(ctx, http.StatusUnprocessableEntity, savedId, page, problem)
		return
	}

	if _, err := h.PageRepo.PageSave(page); err != nil {
		slog.Error("save page failed", "err", err, "id", page.Id)
		h.renderEditor(ctx, http.StatusUnprocessableEntity, savedId, page, saveErrorMessage(err))
		return
	}
	h.refreshSitemap()
	http.Redirect(ctx.Writer, ctx.Request, "/admin/pages?saved="+url.QueryEscape(page.Id), http.StatusFound)
}

func (h *PageHandler) refreshSitemap() {
	if h.Sitemap != nil {
		h.Sitemap.GenerateSitemap()
	}
}
