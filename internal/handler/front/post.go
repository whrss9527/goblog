package front

import (
	"encoding/json"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	ginpkg "goblog/internal/pkg/gin"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

var shanghai *time.Location

func init() {
	var err error
	shanghai, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		shanghai = time.FixedZone("CST", 8*3600)
	}
}

type PostHandler struct {
	PostRepo     repository.PostRepository
	CategoryRepo repository.CategoryRepository
	TagRepo      repository.TagRepository
	conf         *config.Config
}

func NewPostHandler(postRepo repository.PostRepository, categoryRepo repository.CategoryRepository, tagRepository repository.TagRepository, config *config.Config) *PostHandler {
	return &PostHandler{
		PostRepo:     postRepo,
		CategoryRepo: categoryRepo,
		TagRepo:      tagRepository,
		conf:         config,
	}
}

func (h *PostHandler) Index(ctx *gin.Context) {
	categoryId := ctx.Request.URL.Query().Get("category_id")
	tagId := ctx.Request.URL.Query().Get("tag_id")
	perPage, _ := strconv.Atoi(ctx.Request.URL.Query().Get("per_page"))
	page, _ := strconv.Atoi(ctx.Request.URL.Query().Get("page"))
	keyword := ctx.Request.URL.Query().Get("keyword")
	if perPage <= 0 {
		perPage = 14
	}
	if page <= 1 {
		page = 1
	}
	prePage := page - 1
	nextPage := page + 1
	if prePage <= 1 {
		prePage = 1
	}
	params := repository.PostParams{
		CategoryId: categoryId,
		TagId:      tagId,
		PerPage:    perPage,
		Page:       page,
	}
	if len(keyword) > 0 {
		params.Keyword = keyword
		params.Ids = make(map[string][]string)
		postIds, err := h.PostRepo.GetPostIdsByContent(keyword)
		if err == nil {
			params.Ids["ids"] = postIds
		}
		categoryIds, err := h.CategoryRepo.GetCategoryIdsByName(keyword)
		if err == nil {
			params.Ids["category_ids"] = categoryIds
		}
		tagIds, err := h.TagRepo.GetTagIdsByName(keyword)
		if err == nil {
			params.Ids["tag_ids"] = tagIds
		}
	}

	posts, total, err := h.PostRepo.GetPosts(params)
	if err != nil {
		slog.Error("get posts failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	hasMore := int64(page*perPage) < total

	categories, err := h.CategoryRepo.GetCategories()
	if err != nil {
		slog.Error("get categories failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	categoryMap := make(map[int]model.Category)
	for _, category := range categories {
		categoryMap[category.Id] = category
	}
	tagMap := make(map[int]model.Tag)
	if allTags, err := h.TagRepo.GetTags(); err == nil {
		for _, tag := range allTags {
			tagMap[tag.Id] = tag
		}
	}
	for index, post := range posts {
		posts[index].CreatedAt = post.CreatedAt.In(shanghai)
		posts[index].UpdatedAt = post.UpdatedAt.In(shanghai)
		posts[index].CategoryName = categoryMap[post.CategoryId].Name
		for _, tagId := range post.TagIds {
			if tag, ok := tagMap[tagId]; ok {
				posts[index].Tags = append(posts[index].Tags, tag)
			}
		}
	}

	// Describe the active filter so the page can say what is being shown.
	var filterKind, filterLabel string
	switch {
	case keyword != "":
		filterKind, filterLabel = "搜索", keyword
	case tagId != "":
		filterKind = "标签"
		if id, err := strconv.Atoi(tagId); err == nil {
			filterLabel = tagMap[id].Name
		}
	case categoryId != "":
		filterKind = "分类"
		if id, err := strconv.Atoi(categoryId); err == nil {
			filterLabel = categoryMap[id].Name
		}
	}
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages < 1 {
		totalPages = 1
	}

	data := make(map[string]any)
	if all, _, err := h.PostRepo.GetPosts(repository.PostParams{Page: 1}); err == nil {
		data["sidebar"] = buildSidebar(all, tagMap)
		data["on_this_day"] = onThisDay(all, time.Now(), shanghai)
	}
	data["nav"] = "home"
	data["filter_kind"] = filterKind
	data["filter_label"] = filterLabel
	data["category_id"] = categoryId
	data["total_pages"] = totalPages
	if filterKind != "" {
		if filterLabel == "" {
			filterLabel = "未知"
			data["filter_label"] = filterLabel
		}
		data["title"] = filterKind + "：" + filterLabel
		data["noindex"] = keyword != ""
	}
	if keyword == "" {
		data["canonical"] = listCanonical(h.conf.App.Host, categoryId, tagId, page)
		if filterKind == "" && page <= 1 {
			data["json_ld"] = siteJSONLD(h.conf.App)
		}
	}
	data["posts"] = posts
	data["categories"] = categories
	data["page"] = page
	data["pre_url"] = h.getPageUrl(categoryId, tagId, keyword, prePage)
	data["next_url"] = h.getPageUrl(categoryId, tagId, keyword, nextPage)
	data["has_more"] = hasMore
	data["total"] = total
	data["keyword"] = keyword
	view.Render(data, ctx.Writer, "index", h.conf.App)
}

// getPageUrl builds a site-relative list URL that keeps the active filters
// (category, tag, search keyword) while moving between pages.
func (h *PostHandler) getPageUrl(categoryId, tagId, keyword string, page int) string {
	q := url.Values{}
	if categoryId != "" {
		q.Set("category_id", categoryId)
	}
	if tagId != "" {
		q.Set("tag_id", tagId)
	}
	if keyword != "" {
		q.Set("keyword", keyword)
	}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if len(q) == 0 {
		return "/"
	}
	return "/?" + q.Encode()
}

func (h *PostHandler) PostInfo(ctx *gin.Context) {
	identity := ctx.Param("identity")

	post, err := h.PostRepo.GetPostByIdentity(identity)
	if err != nil || post.Status != 1 {
		RenderNotFound(ctx, h.conf.App)
		return
	}
	category, err := h.CategoryRepo.GetCategory(post.CategoryId)
	if err != nil {
		RenderNotFound(ctx, h.conf.App)
		return
	}
	tags, err := h.TagRepo.GetTagsByIds(post.TagIds)
	if err != nil {
		RenderNotFound(ctx, h.conf.App)
		return
	}
	h.recordView(ctx, identity, post.Id)
	post.CategoryName = category.Name
	data := make(map[string]any)
	if all, _, err := h.PostRepo.GetPosts(repository.PostParams{Page: 1}); err == nil {
		data["newer"], data["older"] = neighbours(all, post)
		data["related"] = relatedPosts(all, post, 4)
	}
	data["outdated_years"] = outdatedYears(post, category.Name, time.Now())
	data["liked"] = hasLiked(ctx, post.Identity)
	if html, ok := renderOnServer(h.conf.App, post.Content); ok {
		data["content_html"] = html
	}
	data["post"] = post
	data["tags"] = tags
	data["nav"] = "post"
	data["title"] = post.Title
	description := view.Excerpt(post.Description, 160)
	if description == "" {
		description = view.Excerpt(post.Content, 160)
	}
	data["description"] = description
	data["identity"] = post.Identity
	data["pageId"] = "posts-" + post.Id
	data["canonical"] = h.conf.App.Host + "/posts/" + post.Identity
	image := view.AbsoluteURL(h.conf.App.Host, firstImage(post.Content))
	if image != "" {
		data["og_image"] = image
		data["og_image_large"] = true
	}
	data["published_iso"] = post.CreatedAt.Format(time.RFC3339)
	data["modified_iso"] = post.UpdatedAt.Format(time.RFC3339)
	data["json_ld"] = postJSONLD(h.conf.App, post, description, image, tags)
	view.Render(data, ctx.Writer, "posts", h.conf.App)
}

func (h *PostHandler) recordView(ctx *gin.Context, identity string, postId string) {
	if ctx.Request.Method != "GET" || ctx.GetHeader(ginpkg.OriginalMethodHeader) != "" {
		return // HEAD probes (monitors, link checkers) are not readers
	}
	// quicklink / browser prefetch — don't count as a real visit
	if purpose := ctx.GetHeader("Purpose"); purpose == "prefetch" {
		return
	}
	if purpose := ctx.GetHeader("Sec-Purpose"); purpose == "prefetch" {
		return
	}

	var visits []VisitInfo

	cookie, err := ctx.Cookie("visitInfo")
	if err == nil {
		json.Unmarshal([]byte(cookie), &visits)
	}

	shouldCountVisit := true
	for _, visit := range visits {
		if visit.ArticleID == identity {
			if time.Since(visit.LastVisit) < 30*time.Minute {
				shouldCountVisit = false
			}
			break
		}
	}

	if shouldCountVisit {
		h.PostRepo.IncrView(postId)
		now := time.Now()
		found := false
		for i, visit := range visits {
			if visit.ArticleID == identity {
				visits[i].LastVisit = now
				found = true
				break
			}
		}
		if !found {
			visits = append(visits, VisitInfo{ArticleID: identity, LastVisit: now})
		}
		visitBytes, _ := json.Marshal(visits)
		ctx.SetCookie("visitInfo", string(visitBytes), 3600, "/", "", false, true)
	}
}

type VisitInfo struct {
	ArticleID string    `json:"article_id"`
	LastVisit time.Time `json:"last_visit"`
}
