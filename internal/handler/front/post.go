package front

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
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
	for index, post := range posts {
		posts[index].UpdatedAt = post.UpdatedAt.In(shanghai)
		posts[index].CategoryName = categoryMap[post.CategoryId].Name
	}
	data := make(map[string]any)
	data["posts"] = posts
	data["categories"] = categories
	data["page"] = page
	data["pre_url"] = h.getPageUrl(categoryId, tagId, strconv.Itoa(prePage))
	data["next_url"] = h.getPageUrl(categoryId, tagId, strconv.Itoa(nextPage))
	data["has_more"] = hasMore
	data["total"] = total
	view.Render(data, ctx.Writer, "index", h.conf.App)
}

func (h *PostHandler) getPageUrl(categoryId string, tagId string, page string) string {
	var params []string
	if len(categoryId) > 0 {
		params = append(params, "category_id="+categoryId)
	}
	if len(tagId) > 0 {
		params = append(params, "tag_id="+tagId)
	}
	params = append(params, "page="+page)
	return h.conf.App.Host + "?" + strings.Join(params, "&")
}

func (h *PostHandler) PostInfo(ctx *gin.Context) {
	identity := ctx.Param("identity")

	post, err := h.PostRepo.GetPostByIdentity(identity)
	if err != nil {
		view.Render(make(map[string]any), ctx.Writer, "404", h.conf.App)
		return
	}
	category, err := h.CategoryRepo.GetCategory(post.CategoryId)
	if err != nil {
		view.Render(make(map[string]any), ctx.Writer, "404", h.conf.App)
		return
	}
	tags, err := h.TagRepo.GetTagsByIds(post.TagIds)
	if err != nil {
		view.Render(make(map[string]any), ctx.Writer, "404", h.conf.App)
		return
	}
	h.recordView(ctx, identity, post.Id)
	post.CategoryName = category.Name
	data := make(map[string]any)
	data["post"] = post
	data["tags"] = tags
	data["title"] = post.Title
	data["description"] = post.Description
	data["identity"] = post.Identity
	data["pageId"] = "posts-" + post.Id
	data["canonical"] = h.conf.App.Host + "/posts/" + post.Identity
	view.Render(data, ctx.Writer, "posts", h.conf.App)
}

func (h *PostHandler) recordView(ctx *gin.Context, identity string, postId string) {
	if ctx.Request.Method != "GET" {
		return
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
