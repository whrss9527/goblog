package admin

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/filestore"
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
	h.renderList(ctx, http.StatusOK, "")
}

func (h *TagHandler) renderList(ctx *gin.Context, status int, problem string) {
	tags, err := h.TagRepo.GetTags()
	if err != nil {
		slog.Error("get tags failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	data := make(map[string]any)
	data["tags"] = tags
	data["error"] = problem
	// what the previous action did, e.g. /admin/tags?done=merged&name=blog
	data["done"] = ctx.Query("done")
	data["done_name"] = ctx.Query("name")
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRenderStatus(status, data, ctx.Writer, "tags/list", h.config.App)
}

// TagEdit shows the rename form.
func (h *TagHandler) TagEdit(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Query("id"))
	tag, ok := h.find(id)
	if !ok { // deleted or merged in another tab
		http.Redirect(ctx.Writer, ctx.Request, "/admin/tags", http.StatusFound)
		return
	}
	h.renderForm(ctx, http.StatusOK, tag, tag.Name, "")
}

func (h *TagHandler) renderForm(ctx *gin.Context, status int, tag model.Tag, typed, problem string) {
	tags, _ := h.TagRepo.GetTags()
	names := make([]string, 0, len(tags))
	for _, other := range tags {
		if name := strings.TrimSpace(other.Name); other.Id != tag.Id && name != "" {
			names = append(names, name)
		}
	}
	data := make(map[string]any)
	data["tag"] = tag
	data["typed"] = typed
	data["other_names"] = toJSON(uniqueStrings(names))
	data["error"] = problem
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRenderStatus(status, data, ctx.Writer, "tags/edit", h.config.App)
}

// TagSave renames a tag. A name that already belongs to another tag merges the two.
func (h *TagHandler) TagSave(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	tag, ok := h.find(id)
	if !ok {
		h.renderList(ctx, http.StatusNotFound, "这个标签已经不存在了（可能刚在别的页面被合并或删除）。")
		return
	}
	name := strings.TrimSpace(ctx.PostForm("name"))
	if problem := checkLabel(name, "标签名"); problem != "" {
		h.renderForm(ctx, http.StatusUnprocessableEntity, tag, ctx.PostForm("name"), problem)
		return
	}

	keptId, err := h.TagRepo.RenameTag(id, name)
	switch {
	case errors.Is(err, filestore.ErrTagNotFound):
		h.renderList(ctx, http.StatusNotFound, "这个标签已经不存在了（可能刚在别的页面被合并或删除）。")
		return
	case err != nil:
		slog.Error("rename tag failed", "id", id, "err", err)
		h.renderForm(ctx, http.StatusUnprocessableEntity, tag, name, failureMessage(err, "保存失败，请稍后重试。"))
		return
	}
	done := "renamed"
	if keptId != id {
		done = "merged"
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/tags?done="+done+"&name="+url.QueryEscape(name), http.StatusFound)
}

// TagDelete removes a tag and takes it off every post.
func (h *TagHandler) TagDelete(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	tag, _ := h.find(id)
	err := h.TagRepo.DeleteTag(id)
	switch {
	case errors.Is(err, filestore.ErrTagNotFound):
		h.renderList(ctx, http.StatusNotFound, "这个标签已经不存在了，刷新页面看看。")
		return
	case err != nil:
		slog.Error("delete tag failed", "id", id, "err", err)
		h.renderList(ctx, http.StatusUnprocessableEntity, failureMessage(err, fmt.Sprintf("删除「%s」失败，请稍后重试。", strings.TrimSpace(tag.Name))))
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/tags?done=deleted&name="+url.QueryEscape(strings.TrimSpace(tag.Name)), http.StatusFound)
}

func (h *TagHandler) find(id int) (model.Tag, bool) {
	if id <= 0 {
		return model.Tag{}, false
	}
	tags, err := h.TagRepo.GetTagsByIds([]int{id})
	if err != nil || len(tags) == 0 {
		return model.Tag{}, false
	}
	return tags[0], true
}
