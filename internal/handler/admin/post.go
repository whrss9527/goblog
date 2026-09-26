package admin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/filestore"
	"goblog/internal/handler/front"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type PostHandler struct {
	PostRepo     repository.PostRepository
	DraftRepo    repository.DraftRepository
	CategoryRepo repository.CategoryRepository
	TagRepo      repository.TagRepository
	// Sync links the content repository with its remote (optional).
	Sync           ContentSync
	feedHandler    *front.FeedHandler
	sitemapHandler *front.SitemapHandler
	config         *config.Config
}

// ContentSync is the git side of the content repository.
type ContentSync interface {
	GitEnabled() bool
	Sync(ctx context.Context) (filestore.SyncResult, error)
}

func NewPostHandler(postRepo repository.PostRepository, draftRepo repository.DraftRepository, categoryRepo repository.CategoryRepository, tagRepo repository.TagRepository, feedHandler *front.FeedHandler, sitemapHandler *front.SitemapHandler, config *config.Config) *PostHandler {
	return &PostHandler{
		PostRepo:       postRepo,
		DraftRepo:      draftRepo,
		CategoryRepo:   categoryRepo,
		TagRepo:        tagRepo,
		feedHandler:    feedHandler,
		sitemapHandler: sitemapHandler,
		config:         config,
	}
}

// PostList 处理文章列表请求。The table searches, sorts and pages in the browser, so
// it gets every post; with the old server-side page size of 11 anything older
// than the newest eleven posts could not be reached from the admin.
func (h *PostHandler) PostList(ctx *gin.Context) {
	posts, _, err := h.PostRepo.GetPosts(repository.PostParams{
		CategoryId: ctx.Query("category_id"),
		TagId:      ctx.Query("tag_id"),
		Page:       1,
	})
	if err != nil {
		slog.Error("get posts failed", "err", err)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	categories, err := h.CategoryRepo.GetCategories()
	if err != nil {
		slog.Error("get categories failed", "err", err)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	categoryMap := make(map[int]model.Category)
	for _, category := range categories {
		categoryMap[category.Id] = category
	}
	for index, post := range posts {
		posts[index].CategoryName = categoryMap[post.CategoryId].Name
		posts[index].CreatedAt = post.CreatedAt.In(adminZone)
		posts[index].UpdatedAt = post.UpdatedAt.In(adminZone)
	}
	drafts, err := h.DraftRepo.GetDrafts()
	if err != nil {
		slog.Error("get drafts failed", "err", err)
	}
	for index, draft := range drafts {
		drafts[index].UpdatedAt = draft.UpdatedAt.In(adminZone)
	}
	data := make(map[string]interface{})
	data["posts"] = posts
	data["drafts"] = drafts
	data["categories"] = categories
	data["git_sync"] = h.Sync != nil && h.Sync.GitEnabled()
	// the outcome of "同步内容仓库", e.g. /admin?sync=ok&pulled=2&head=1a2b3c4
	data["sync"] = ctx.Query("sync")
	data["sync_pulled"], _ = strconv.Atoi(ctx.Query("pulled"))
	data["sync_head"] = ctx.Query("head")
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "posts/list", h.config.App)
}

// SyncNow handles POST /admin/sync: bring in what was pushed to the content
// repository from elsewhere (a laptop, GitHub's editor) without waiting for
// the next scheduled sync, and push what this server has not pushed yet.
func (h *PostHandler) SyncNow(ctx *gin.Context) {
	if h.Sync == nil || !h.Sync.GitEnabled() {
		http.Redirect(ctx.Writer, ctx.Request, "/admin?sync=noremote", http.StatusFound)
		return
	}
	c, cancel := context.WithTimeout(ctx.Request.Context(), time.Minute)
	defer cancel()
	result, err := h.Sync.Sync(c)
	outcome := "ok"
	switch {
	case errors.Is(err, filestore.ErrSyncConflict):
		outcome = "conflict"
	case errors.Is(err, filestore.ErrNoRemote):
		outcome = "noremote"
	case err != nil:
		slog.Error("sync content repository failed", "err", err)
		outcome = "failed"
	}
	q := url.Values{"sync": {outcome}}
	if outcome == "ok" {
		q.Set("pulled", strconv.Itoa(result.Pulled))
		q.Set("head", result.Head)
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin?"+q.Encode(), http.StatusFound)
}

// PostAdd 处理文章添加请求: a new post, an existing one (?id=) or a draft (?draft=).
func (h *PostHandler) PostAdd(ctx *gin.Context) {
	if slug := ctx.Query("draft"); slug != "" {
		draft, err := h.DraftRepo.GetDraft(slug)
		if err != nil {
			data := map[string]interface{}{"msg": "没有找到这份草稿，它可能已经发布或被删除"}
			view.AdminRenderStatus(http.StatusNotFound, data, ctx.Writer, "401", h.config.App)
			return
		}
		tags := strings.Join(draft.TagNames, ",")
		if tags == "" {
			tags = h.getTags(draft)
		}
		draft.Id = "" // not published yet
		h.renderEditor(ctx, http.StatusOK, draft, tags, slug, "")
		return
	}

	id := ctx.Request.FormValue("id")
	var post model.Post
	if len(id) > 0 {
		var err error
		post, err = h.PostRepo.GetPost(id)
		if err != nil {
			slog.Error("get post failed", "err", err)
			data := map[string]interface{}{"msg": "没有找到这篇文章，它可能已经被删除或改了地址"}
			view.AdminRenderStatus(http.StatusNotFound, data, ctx.Writer, "401", h.config.App)
			return
		}
	}
	h.renderEditor(ctx, http.StatusOK, post, h.getTags(post), "", "")
}

// renderEditor shows the post editor, either for PostAdd or to hand a rejected
// save back to the author with everything they typed still in place. draft is
// the slug of the stored draft being edited ("" if there is none).
func (h *PostHandler) renderEditor(ctx *gin.Context, status int, post model.Post, tags string, draft string, problem string) {
	data := make(map[string]interface{})
	categories, _ := h.CategoryRepo.GetCategories()
	for i := range categories {
		categories[i].Cur = post.CategoryId
	}
	data["categories"] = categories
	data["id"] = post.Id
	data["draft"] = draft
	data["title"] = post.Title
	data["description"] = post.Description
	data["content"] = post.Content
	data["category_id"] = post.CategoryId
	data["tag_ids"] = post.TagIds
	data["identity"] = post.Identity
	data["tags"] = tags
	data["error"] = problem
	data["csrf_token"], _ = ctx.Get("csrf_token")

	// for the editor's hints: addresses that are taken, tags that exist
	var slugs []string
	if all, _, err := h.PostRepo.GetPosts(repository.PostParams{Page: 1}); err == nil {
		for _, p := range all {
			if p.Id != post.Id {
				slugs = append(slugs, p.Identity)
			}
		}
	}
	if drafts, err := h.DraftRepo.GetDrafts(); err == nil {
		for _, d := range drafts {
			if d.Identity != draft {
				slugs = append(slugs, d.Identity)
			}
		}
	}
	var tagNames []string
	if allTags, err := h.TagRepo.GetTags(); err == nil {
		// most used first: those are the ones worth a tap
		sort.SliceStable(allTags, func(i, j int) bool { return allTags[i].Count > allTags[j].Count })
		for _, tag := range allTags {
			if name := strings.TrimSpace(tag.Name); name != "" {
				tagNames = append(tagNames, name)
			}
		}
	}
	data["slugs_json"] = toJSON(slugs)
	data["all_tags_json"] = toJSON(uniqueStrings(tagNames))

	view.AdminRenderStatus(status, data, ctx.Writer, "posts/add", h.config.App)
}

// PostSave 处理文章保存请求。action=draft keeps an unpublished post as a private
// draft, anything else publishes.
func (h *PostHandler) PostSave(ctx *gin.Context) {
	var post model.Post
	post.Id = ctx.Request.FormValue("id")
	post.Title = strings.TrimSpace(ctx.Request.FormValue("title"))
	post.Description = ctx.Request.FormValue("description")
	post.Content = ctx.Request.FormValue("content")
	post.CategoryId, _ = strconv.Atoi(ctx.Request.FormValue("category"))
	tags := ctx.Request.FormValue("tags")
	post.Identity = strings.TrimSpace(ctx.Request.FormValue("identity"))
	post.Status = 1
	draft := strings.TrimSpace(ctx.Request.FormValue("draft"))
	asDraft := ctx.Request.FormValue("action") == "draft" && post.Id == "" // a published post is never turned back into a draft

	// Validate before anything is written. A slug that stays as it is always
	// passes: a few old posts have addresses the rules below would refuse.
	problem := ""
	if post.Title == "" {
		problem = "请填写标题"
	} else if post.Id == "" || post.Identity != post.Id {
		problem = checkSlug(post.Identity)
		if problem == "" {
			if _, err := h.PostRepo.GetPost(post.Identity); err == nil {
				problem = saveErrorMessage(filestore.ErrSlugTaken)
			}
		}
	}
	if problem != "" {
		h.renderEditor(ctx, http.StatusUnprocessableEntity, post, tags, draft, problem)
		return
	}

	if asDraft {
		post.TagNames = splitTags(tags)
		post.TagIds = h.existingTagIds(post.TagNames)
		if err := h.DraftRepo.SaveDraft(post, draft); err != nil {
			slog.Error("save draft failed", "err", err, "slug", post.Identity)
			h.renderEditor(ctx, http.StatusUnprocessableEntity, post, tags, draft, saveErrorMessage(err))
			return
		}
		target := "/admin/posts/add?draft=" + url.QueryEscape(post.Identity) + "&saved=" + url.QueryEscape(post.Identity)
		http.Redirect(ctx.Writer, ctx.Request, target, http.StatusFound)
		return
	}

	post.TagIds = h.getTagIds(tags)
	if _, err := h.PostRepo.PostSave(post); err != nil {
		slog.Error("save post failed", "err", err, "slug", post.Identity)
		h.renderEditor(ctx, http.StatusUnprocessableEntity, post, tags, draft, saveErrorMessage(err))
		return
	}
	if draft != "" {
		if err := h.DraftRepo.DeleteDraft(draft); err != nil {
			slog.Error("delete published draft failed", "err", err, "slug", draft)
		}
	}
	h.feedHandler.GenerateFeedXml()
	h.sitemapHandler.GenerateSitemap()

	if err := h.TagRepo.IncrTagCount(""); err != nil {
		slog.Error("recalc tag counts failed", "err", err)
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin?saved="+url.QueryEscape(post.Identity), http.StatusFound)
}

// DraftDelete removes a draft.
func (h *PostHandler) DraftDelete(ctx *gin.Context) {
	if err := h.DraftRepo.DeleteDraft(ctx.Param("slug")); err != nil {
		slog.Error("delete draft failed", "err", err)
		data := map[string]interface{}{"msg": "删除草稿失败，请重试"}
		view.AdminRenderStatus(http.StatusInternalServerError, data, ctx.Writer, "401", h.config.App)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusFound)
}

// DraftPreview shows a draft the way the published post will look.
func (h *PostHandler) DraftPreview(ctx *gin.Context) {
	draft, err := h.DraftRepo.GetDraft(ctx.Query("draft"))
	if err != nil {
		data := map[string]interface{}{"msg": "没有找到这份草稿，它可能已经发布或被删除"}
		view.AdminRenderStatus(http.StatusNotFound, data, ctx.Writer, "401", h.config.App)
		return
	}
	if category, err := h.CategoryRepo.GetCategory(draft.CategoryId); err == nil {
		draft.CategoryName = category.Name
	}
	names := draft.TagNames
	if len(names) == 0 {
		names = splitTags(h.getTags(draft))
	}
	tags := make([]model.Tag, 0, len(names))
	for _, name := range names {
		tags = append(tags, model.Tag{Name: name})
	}
	front.RenderPostPreview(ctx, h.config.App, draft, tags)
}

// PostDelete 处理文章删除请求
func (h *PostHandler) PostDelete(ctx *gin.Context) {
	var post model.Post
	post.Id = ctx.Param("id")
	_, err := h.PostRepo.PostDelete(post)
	if err != nil {
		data := make(map[string]interface{})
		data["msg"] = "删除失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}

	h.feedHandler.GenerateFeedXml()
	h.sitemapHandler.GenerateSitemap()
	if err := h.TagRepo.IncrTagCount(""); err != nil {
		slog.Error("recalc tag counts failed", "err", err)
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusFound)
}

// getTags 获取文章标签
func (h *PostHandler) getTags(post model.Post) string {
	var tags []string
	if len(post.TagIds) > 0 {
		allTags, _ := h.TagRepo.GetTags()

		tagsById := make(map[int]model.Tag)
		for _, tag := range allTags {
			tagsById[tag.Id] = tag
		}
		for _, tagId := range post.TagIds {
			tags = append(tags, tagsById[tagId].Name)
		}

	}
	return strings.Join(tags, ",")
}

// splitTags parses the tags field: separated by commas (also the full-width
// one), trimmed, without empties and duplicates. It used to be split on ","
// only and not trimmed, so "go, mysql" created a second tag named " mysql" and
// an empty field created a tag without a name.
func splitTags(tags string) []string {
	parts := strings.FieldsFunc(tags, func(r rune) bool { return r == ',' || r == '，' })
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.TrimSpace(part); name != "" {
			names = append(names, name)
		}
	}
	return uniqueStrings(names)
}

func uniqueStrings(list []string) []string {
	seen := make(map[string]bool, len(list))
	out := list[:0:0]
	for _, item := range list {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

// tagsByName indexes the existing tags by their trimmed name; of two tags with
// the same name (" mysql" and "mysql") the one without padding wins.
func (h *PostHandler) tagsByName() map[string]model.Tag {
	allTags, _ := h.TagRepo.GetTags()
	byName := make(map[string]model.Tag, len(allTags))
	for _, tag := range allTags {
		name := strings.TrimSpace(tag.Name)
		if current, ok := byName[name]; !ok || current.Name != name {
			byName[name] = tag
		}
	}
	return byName
}

// existingTagIds resolves names to ids without creating anything.
func (h *PostHandler) existingTagIds(names []string) (tagIds []int) {
	byName := h.tagsByName()
	for _, name := range names {
		if tag, ok := byName[name]; ok {
			tagIds = append(tagIds, tag.Id)
		}
	}
	return
}

// getTagIds 根据标签名称获取标签 ID 列表，不存在的标签会被创建
func (h *PostHandler) getTagIds(tags string) (tagIds []int) {
	byName := h.tagsByName()
	for _, name := range splitTags(tags) {
		if tag, ok := byName[name]; ok {
			tagIds = append(tagIds, tag.Id)
			continue
		}
		if newTagId, _ := h.TagRepo.AddTag(model.Tag{Name: name}); newTagId > 0 {
			tagIds = append(tagIds, newTagId)
		}
	}
	return
}
