package admin

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/filestore"
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
	h.renderList(ctx, http.StatusOK, "")
}

// renderList shows the list, optionally with the reason why the last action was refused.
func (h *CategoryHandler) renderList(ctx *gin.Context, status int, problem string) {
	categories, err := h.CategoryRepo.GetCategories()
	if err != nil {
		slog.Error("get categories failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	data := make(map[string]any)
	data["categories"] = categories
	data["error"] = problem
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRenderStatus(status, data, ctx.Writer, "categories/list", h.config.App)
}

// renderForm shows the add / edit form again with what was typed and what is wrong with it.
func (h *CategoryHandler) renderForm(ctx *gin.Context, status int, category model.Category, problem string) {
	data := make(map[string]any)
	if category.Id > 0 {
		data["id"] = category.Id
	}
	data["name"] = category.Name
	data["error"] = problem
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRenderStatus(status, data, ctx.Writer, "categories/add", h.config.App)
}

func (h *CategoryHandler) CategoryAdd(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Request.FormValue("id"))
	var category model.Category
	if id > 0 {
		var err error
		category, err = h.CategoryRepo.GetCategory(id)
		if err != nil { // deleted in another tab
			http.Redirect(ctx.Writer, ctx.Request, "/admin/categories", http.StatusFound)
			return
		}
	}
	h.renderForm(ctx, http.StatusOK, category, "")
}

func (h *CategoryHandler) CategoryDelete(ctx *gin.Context) {
	var category model.Category
	idStr := ctx.Request.FormValue("id")
	category.Id, _ = strconv.Atoi(idStr)
	_, err := h.CategoryRepo.CategoryDelete(category)
	if err != nil {
		var inUse *filestore.CategoryInUseError
		if errors.As(err, &inUse) {
			h.renderList(ctx, http.StatusConflict,
				fmt.Sprintf("这个分类下还有 %d 篇文章，先把它们改到别的分类，再来删除。", inUse.Posts))
			return
		}
		slog.Error("delete category failed", "err", err)
		h.renderList(ctx, http.StatusUnprocessableEntity, "删除失败：这个分类可能已经不存在了，刷新页面看看。")
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/categories", http.StatusFound)
}

func (h *CategoryHandler) CategorySave(ctx *gin.Context) {
	var category model.Category
	idStr := ctx.Request.FormValue("id")
	category.Id, _ = strconv.Atoi(idStr)
	category.Name = strings.TrimSpace(ctx.Request.FormValue("name"))
	if problem := checkLabel(category.Name, "分类名"); problem != "" {
		h.renderForm(ctx, http.StatusUnprocessableEntity, category, problem)
		return
	}
	_, err := h.CategoryRepo.CategorySave(category)
	if err != nil {
		slog.Error("save category failed", "err", err)
		h.renderForm(ctx, http.StatusUnprocessableEntity, category, "保存失败，请稍后重试。")
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/categories", http.StatusFound)
}
