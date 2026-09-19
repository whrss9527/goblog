package admin

import (
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/filestore"
	"goblog/internal/handler/front"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/utils"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type PostHandler struct {
	PostRepo       repository.PostRepository
	CategoryRepo   repository.CategoryRepository
	TagRepo        repository.TagRepository
	feedHandler    *front.FeedHandler
	sitemapHandler *front.SitemapHandler
	config         *config.Config
}

func NewPostHandler(postRepo repository.PostRepository, categoryRepo repository.CategoryRepository, tagRepo repository.TagRepository, feedHandler *front.FeedHandler, sitemapHandler *front.SitemapHandler, config *config.Config) *PostHandler {
	return &PostHandler{
		PostRepo:       postRepo,
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
	data := make(map[string]interface{})
	data["posts"] = posts
	data["categories"] = categories
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "posts/list", h.config.App)
}

// PostAdd 处理文章添加请求
func (h *PostHandler) PostAdd(ctx *gin.Context) {
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
	h.renderEditor(ctx, http.StatusOK, post, h.getTags(post), "")
}

// renderEditor shows the post editor, either for PostAdd or to hand a rejected
// save back to the author with everything they typed still in place.
func (h *PostHandler) renderEditor(ctx *gin.Context, status int, post model.Post, tags string, problem string) {
	data := make(map[string]interface{})
	categories, _ := h.CategoryRepo.GetCategories()
	for i := range categories {
		categories[i].Cur = post.CategoryId
	}
	data["categories"] = categories
	data["id"] = post.Id
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
	var tagNames []string
	if allTags, err := h.TagRepo.GetTags(); err == nil {
		// most used first: those are the ones worth a tap
		sort.SliceStable(allTags, func(i, j int) bool { return allTags[i].Count > allTags[j].Count })
		for _, tag := range allTags {
			tagNames = append(tagNames, tag.Name)
		}
	}
	data["slugs_json"] = toJSON(slugs)
	data["all_tags_json"] = toJSON(tagNames)

	view.AdminRenderStatus(status, data, ctx.Writer, "posts/add", h.config.App)
}

// PostSave 处理文章保存请求
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
		h.renderEditor(ctx, http.StatusUnprocessableEntity, post, tags, problem)
		return
	}

	post.TagIds = h.getTagIds(tags)
	if _, err := h.PostRepo.PostSave(post); err != nil {
		slog.Error("save post failed", "err", err, "slug", post.Identity)
		h.renderEditor(ctx, http.StatusUnprocessableEntity, post, tags, saveErrorMessage(err))
		return
	}
	h.feedHandler.GenerateFeedXml()
	h.sitemapHandler.GenerateSitemap()

	if err := h.TagRepo.IncrTagCount(""); err != nil {
		slog.Error("recalc tag counts failed", "err", err)
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin?saved="+url.QueryEscape(post.Identity), http.StatusFound)
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

// getTagIds 根据标签名称获取标签 ID 列表
func (h *PostHandler) getTagIds(tags string) (tagIds []int) {
	tagNames := strings.Split(tags, ",")
	tagNames = utils.RemoveDuplicateElement(tagNames)
	allTags, _ := h.TagRepo.GetTags()
	var allTagNames []string
	allTagByName := make(map[string]model.Tag)
	for _, tag := range allTags {
		allTagNames = append(allTagNames, tag.Name)
		allTagByName[tag.Name] = tag
	}
	for _, tagName := range tagNames {
		if utils.StrInArray(tagName, allTagNames) {
			tagIds = append(tagIds, allTagByName[tagName].Id)
		} else {
			var newTag model.Tag
			newTag.Name = tagName
			newTagId, _ := h.TagRepo.AddTag(newTag)
			if newTagId > 0 {
				tagIds = append(tagIds, newTagId)
			}
		}
	}
	return
}
