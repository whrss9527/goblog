package admin

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
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

// PostList 处理文章列表请求
func (h *PostHandler) PostList(ctx *gin.Context) {
	categoryId := ctx.Request.URL.Query().Get("category_id")
	tagId := ctx.Request.URL.Query().Get("tag_id")
	pageSize, _ := strconv.Atoi(ctx.Request.URL.Query().Get("per_page"))
	page, _ := strconv.Atoi(ctx.Request.URL.Query().Get("page"))
	if pageSize <= 0 {
		pageSize = 11
	}
	if page <= 1 {
		page = 1
	}
	prePage := page - 1
	nextPage := page + 1
	if prePage <= 1 {
		prePage = 1
	}
	posts, _, err := h.PostRepo.GetPosts(repository.PostParams{
		CategoryId: categoryId,
		TagId:      tagId,
		PerPage:    pageSize,
		Page:       page,
	})
	if err != nil {
		slog.Error("get posts failed", "err", err)
		return
	}
	categories, err := h.CategoryRepo.GetCategories()
	if err != nil {
		slog.Error("get categories failed", "err", err)
		return
	}
	categoryMap := make(map[int]model.Category)
	for _, category := range categories {
		categoryMap[category.Id] = category
	}
	for index, post := range posts {
		posts[index].CategoryName = categoryMap[post.CategoryId].Name
	}
	data := make(map[string]interface{})
	data["posts"] = posts
	data["categories"] = categories
	data["page"] = page
	data["pre_url"] = h.getPageUrl(categoryId, tagId, strconv.Itoa(prePage))
	data["next_url"] = h.getPageUrl(categoryId, tagId, strconv.Itoa(nextPage))
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "posts/list", h.config.App)
}

// PostAdd 处理文章添加请求
func (h *PostHandler) PostAdd(ctx *gin.Context) {
	data := make(map[string]interface{})
	id := ctx.Request.FormValue("id")
	var post model.Post
	if len(id) > 0 {
		var err error
		post, err = h.PostRepo.GetPost(id)
		if err != nil {
			slog.Error("get post failed", "err", err)
			return
		}
	}
	categories, _ := h.CategoryRepo.GetCategories()
	data["categories"] = categories

	for i := range categories {
		categories[i].Cur = post.CategoryId
	}
	data["id"] = post.Id
	data["title"] = post.Title
	data["description"] = post.Description
	data["content"] = post.Content
	data["category_id"] = post.CategoryId
	data["tag_ids"] = post.TagIds
	data["identity"] = post.Identity
	data["tags"] = h.getTags(post)
	data["csrf_token"], _ = ctx.Get("csrf_token")

	view.AdminRender(data, ctx.Writer, "posts/add", h.config.App)
}

// PostSave 处理文章保存请求
func (h *PostHandler) PostSave(ctx *gin.Context) {
	var post model.Post
	post.Id = ctx.Request.FormValue("id")
	post.Title = ctx.Request.FormValue("title")
	post.Description = ctx.Request.FormValue("description")
	post.Content = ctx.Request.FormValue("content")
	post.CategoryId, _ = strconv.Atoi(ctx.Request.FormValue("category"))
	tags := ctx.Request.FormValue("tags")
	post.Identity = ctx.Request.FormValue("identity")
	post.TagIds = h.getTagIds(tags)
	post.Status = 1

	_, err := h.PostRepo.PostSave(post)
	if err != nil {
		data := make(map[string]interface{})
		data["msg"] = "添加或修改失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	h.feedHandler.GenerateFeedXml()
	h.sitemapHandler.GenerateSitemap()

	for _, tagId := range post.TagIds {
		if err := h.TagRepo.IncrTagCount(strconv.Itoa(tagId)); err != nil {
			slog.Error("incr tag count failed", "tag_id", tagId, "err", err)
		}
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusFound)
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

// getPageUrl 生成页面链接
func (h *PostHandler) getPageUrl(categoryId, tagId, page string) string {
	params := make([]string, 0)
	if categoryId != "" {
		params = append(params, "category_id="+categoryId)
	}
	if tagId != "" {
		params = append(params, "tag_id="+tagId)
	}
	params = append(params, "page="+page)
	return "/admin?" + strings.Join(params, "&")
}
