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

type CategoryHandler struct {
	CategoryRepo repository.CategoryRepository
	config       *config.Config
}

func NewCategoryHandler(categoryRepo repository.CategoryRepository, config *config.Config) *CategoryHandler {
	return &CategoryHandler{
		CategoryRepo: categoryRepo,
		config:       config,
	}
}

func (h *CategoryHandler) CategoryList(ctx *gin.Context) {
	categories, err := h.CategoryRepo.GetCategories()
	if err != nil {
		slog.Error("get categories failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	data := make(map[string]any)
	data["categories"] = categories
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "categories/list", h.config.App)
}

func (h *CategoryHandler) CategoryAdd(ctx *gin.Context) {
	data := make(map[string]any)
	idStr := ctx.Request.FormValue("id")
	id, _ := strconv.Atoi(idStr)
	var category model.Category
	if id > 0 {
		var err error
		category, err = h.CategoryRepo.GetCategory(id)
		if err != nil {
			slog.Error("get category failed", "err", err)
			ctx.Writer.WriteHeader(500)
			return
		}
	}
	categories, _ := h.CategoryRepo.GetCategories()
	data["categories"] = categories

	if category.Id > 0 {
		data["id"] = category.Id
		data["name"] = category.Name
	}
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "categories/add", h.config.App)
}

func (h *CategoryHandler) CategoryDelete(ctx *gin.Context) {
	var category model.Category
	idStr := ctx.Request.FormValue("id")
	category.Id, _ = strconv.Atoi(idStr)
	_, err := h.CategoryRepo.CategoryDelete(category)
	if err != nil {
		data := make(map[string]any)
		data["msg"] = "删除失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/categories/list", http.StatusFound)
}

func (h *CategoryHandler) CategorySave(ctx *gin.Context) {
	var category model.Category
	idStr := ctx.Request.FormValue("id")
	category.Id, _ = strconv.Atoi(idStr)
	category.Name = ctx.Request.FormValue("name")
	_, err := h.CategoryRepo.CategorySave(category)
	if err != nil {
		data := make(map[string]any)
		data["msg"] = "添加或修改失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/categories", http.StatusFound)
}
