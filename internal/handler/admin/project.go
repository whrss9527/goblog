package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/filestore"
	"goblog/internal/handler/front"
	"goblog/internal/pkg/github"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

const (
	maxProjectName       = 60
	maxProjectText       = 300
	maxProjectHighlights = 6
	maxHighlightLength   = 80
	maxProjectTech       = 10
	maxTechLength        = 30
	// suggestionCount is how many of the owner's recent repositories the list page offers.
	suggestionCount = 8
	// suggestionTTL keeps the list of the owner's repositories for a while: the
	// admin reloads the page after every add.
	suggestionTTL = 10 * time.Minute
)

type ProjectHandler struct {
	ProjectRepo repository.ProjectRepository
	PostRepo    repository.PostRepository
	github      *github.Client
	stats       *github.Stats // nil when app.github_stats is off
	sitemap     *front.SitemapHandler
	config      *config.Config

	suggestMu   sync.Mutex
	suggestAt   time.Time
	suggestFrom string
	suggested   []github.Repo
}

func NewProjectHandler(projectRepo repository.ProjectRepository, postRepo repository.PostRepository, client *github.Client,
	stats *github.Stats, sitemap *front.SitemapHandler, config *config.Config) *ProjectHandler {
	return &ProjectHandler{
		ProjectRepo: projectRepo,
		PostRepo:    postRepo,
		github:      client,
		stats:       stats,
		sitemap:     sitemap,
		config:      config,
	}
}

func (h *ProjectHandler) ProjectList(ctx *gin.Context) {
	h.renderList(ctx, http.StatusOK, "")
}

func (h *ProjectHandler) renderList(ctx *gin.Context, status int, problem string) {
	projects, err := h.ProjectRepo.GetProjects()
	if err != nil {
		slog.Error("get projects failed", "err", err)
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	data := make(map[string]any)
	data["projects"] = projects
	data["error"] = problem
	data["done"] = ctx.Query("done")
	data["done_name"] = ctx.Query("name")
	data["github_user"] = h.githubUser()
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRenderStatus(status, data, ctx.Writer, "projects/list", h.config.App)
}

// ProjectAdd shows the form: empty, for ?id= an existing project, and for
// ?repo= a new one whose details the page fetches from GitHub right away.
func (h *ProjectHandler) ProjectAdd(ctx *gin.Context) {
	var project model.Project
	if id, _ := strconv.Atoi(ctx.Query("id")); id > 0 {
		var err error
		project, err = h.ProjectRepo.GetProject(id)
		if err != nil { // deleted in another tab
			http.Redirect(ctx.Writer, ctx.Request, "/admin/projects", http.StatusFound)
			return
		}
	} else {
		project.Repo = strings.TrimSpace(ctx.Query("repo"))
		project.Status = model.ProjectStatusActive
	}
	h.renderForm(ctx, http.StatusOK, project, "")
}

func (h *ProjectHandler) renderForm(ctx *gin.Context, status int, project model.Project, problem string) {
	data := make(map[string]any)
	data["project"] = project
	data["highlights"] = strings.Join(project.Highlights, "\n")
	data["tech"] = strings.Join(project.Tech, ", ")
	data["error"] = problem
	data["posts"] = h.postChoices(project.Post)
	data["github_import"] = project.Id == 0 && project.Name == "" && project.Repo != ""
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRenderStatus(status, data, ctx.Writer, "projects/add", h.config.App)
}

// postChoice is one option of the "介绍文章" select.
type postChoice struct {
	Slug, Title, Date string
	Selected          bool
}

func (h *ProjectHandler) postChoices(selected string) []postChoice {
	posts, _, err := h.PostRepo.GetPosts(repository.PostParams{Page: 1})
	if err != nil {
		return nil
	}
	choices := make([]postChoice, 0, len(posts))
	for _, p := range posts {
		choices = append(choices, postChoice{Slug: p.Identity, Title: p.Title, Date: p.CreatedAt.In(adminZone).Format("2006-01-02"), Selected: p.Identity == selected})
	}
	return choices
}

// ProjectSave validates the form and stores the project. A rejected form comes
// back with everything that was typed and what is wrong with it.
func (h *ProjectHandler) ProjectSave(ctx *gin.Context) {
	project := model.Project{
		Name:        strings.TrimSpace(ctx.PostForm("name")),
		Description: strings.TrimSpace(ctx.PostForm("description")),
		Highlights:  splitLines(ctx.PostForm("highlights")),
		Repo:        strings.TrimSpace(ctx.PostForm("repo")),
		URL:         strings.TrimSpace(ctx.PostForm("url")),
		Tech:        splitTags(ctx.PostForm("tech")),
		Post:        strings.TrimSpace(ctx.PostForm("post")),
		Cover:       strings.TrimSpace(ctx.PostForm("cover")),
		Featured:    ctx.PostForm("featured") != "",
		Started:     strings.TrimSpace(ctx.PostForm("started")),
	}
	project.Id, _ = strconv.Atoi(ctx.PostForm("id"))
	project.Status, _ = strconv.Atoi(ctx.PostForm("status"))
	// "owner/name" is accepted for GitHub and stored as the full address
	if project.Repo != "" && !strings.Contains(project.Repo, "://") {
		if owner, name, ok := github.ParseRepo(project.Repo); ok {
			project.Repo = github.RepoURL(owner, name)
		}
	}

	if problem := h.checkProject(project); problem != "" {
		h.renderForm(ctx, http.StatusUnprocessableEntity, project, problem)
		return
	}
	if _, err := h.ProjectRepo.ProjectSave(project); err != nil {
		slog.Error("save project failed", "err", err, "name", project.Name)
		h.renderForm(ctx, http.StatusUnprocessableEntity, project, "保存失败，请稍后重试。")
		return
	}
	h.afterChange()
	http.Redirect(ctx.Writer, ctx.Request, "/admin/projects?done=saved&name="+url.QueryEscape(project.Name), http.StatusFound)
}

// ProjectDelete removes a project.
func (h *ProjectHandler) ProjectDelete(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	project, _ := h.ProjectRepo.GetProject(id)
	err := h.ProjectRepo.ProjectDelete(id)
	switch {
	case errors.Is(err, filestore.ErrProjectNotFound):
		h.renderList(ctx, http.StatusNotFound, "这个项目已经不存在了，刷新页面看看。")
		return
	case err != nil:
		slog.Error("delete project failed", "id", id, "err", err)
		h.renderList(ctx, http.StatusUnprocessableEntity, fmt.Sprintf("删除「%s」失败，请稍后重试。", project.Name))
		return
	}
	h.afterChange()
	http.Redirect(ctx.Writer, ctx.Request, "/admin/projects?done=deleted&name="+url.QueryEscape(project.Name), http.StatusFound)
}

// afterChange updates what depends on the list of projects.
func (h *ProjectHandler) afterChange() {
	if projects, err := h.ProjectRepo.GetProjects(); err == nil {
		view.SetProjectCount(len(projects))
	}
	h.sitemap.GenerateSitemap()
	h.stats.Kick() // numbers for a newly added repository
}

// checkProject returns what is wrong with a project for its author ("" when nothing is).
func (h *ProjectHandler) checkProject(p model.Project) string {
	switch {
	case p.Name == "":
		return "请填写项目名称"
	case utf8.RuneCountInString(p.Name) > maxProjectName:
		return fmt.Sprintf("名称太长了，最多 %d 个字", maxProjectName)
	case strings.ContainsFunc(p.Name, unicode.IsControl):
		return "名称里不能有换行或控制字符"
	case utf8.RuneCountInString(p.Description) > maxProjectText:
		return fmt.Sprintf("简介最多 %d 个字；更长的介绍可以写成一篇文章，再在「介绍文章」里选上它", maxProjectText)
	case len(p.Highlights) > maxProjectHighlights:
		return fmt.Sprintf("亮点最多 %d 条", maxProjectHighlights)
	case len(p.Tech) > maxProjectTech:
		return fmt.Sprintf("技术栈最多 %d 项", maxProjectTech)
	case p.Status != 0 && (p.Status < model.ProjectStatusActive || p.Status > model.ProjectStatusArchived):
		return "请选择项目状态"
	}
	for _, line := range p.Highlights {
		if utf8.RuneCountInString(line) > maxHighlightLength {
			return fmt.Sprintf("每条亮点最多 %d 个字：「%s」", maxHighlightLength, line)
		}
	}
	for _, t := range p.Tech {
		if utf8.RuneCountInString(t) > maxTechLength {
			return fmt.Sprintf("技术栈每项最多 %d 个字：「%s」", maxTechLength, t)
		}
	}
	if p.Repo != "" && !isWebAddress(p.Repo) {
		return "源码地址要以 https:// 或 http:// 开头（GitHub 仓库也可以写成 owner/name）"
	}
	if p.URL != "" && !isWebAddress(p.URL) {
		return "访问地址要以 https:// 或 http:// 开头"
	}
	if p.Cover != "" && !isWebAddress(p.Cover) && !(strings.HasPrefix(p.Cover, "/") && !strings.HasPrefix(p.Cover, "//")) {
		return "封面图要以 https:// 开头，或者是站内地址（如 /covers/goblog.png）"
	}
	if p.Started != "" {
		if _, err := time.Parse("2006-01", p.Started); err != nil {
			return "开始时间的格式是「年-月」，例如 2026-09"
		}
	}
	if p.Post != "" {
		if post, err := h.PostRepo.GetPostByIdentity(p.Post); err != nil || post.Status != 1 {
			return fmt.Sprintf("没有找到地址为「%s」的已发布文章", p.Post)
		}
	}
	if other := h.sameRepo(p); other != "" {
		return fmt.Sprintf("这个仓库已经在项目「%s」里了", other)
	}
	return ""
}

// sameRepo returns the name of another project with the same repository.
func (h *ProjectHandler) sameRepo(p model.Project) string {
	if p.Repo == "" {
		return ""
	}
	key := repoKey(p.Repo)
	projects, _ := h.ProjectRepo.GetProjects()
	for _, other := range projects {
		if other.Id != p.Id && other.Repo != "" && repoKey(other.Repo) == key {
			return other.Name
		}
	}
	return ""
}

// repoKey identifies a repository address: GitHub addresses by owner/name, others as written.
func repoKey(repo string) string {
	if owner, name, ok := github.ParseRepo(repo); ok {
		return github.Key(owner, name)
	}
	return strings.TrimRight(strings.ToLower(repo), "/")
}

func isWebAddress(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}

// listMarker is a list bullet someone may type or paste in front of a highlight.
var listMarker = regexp.MustCompile(`^(?:[-*+]\s+|[•·]\s*)`)

// splitLines turns a textarea into its non-empty, trimmed lines, without list bullets.
func splitLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(listMarker.ReplaceAllString(strings.TrimSpace(line), ""))
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// importedProject is what the form fills in from a GitHub repository.
type importedProject struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Repo        string   `json:"repo"`
	URL         string   `json:"url"`
	Tech        []string `json:"tech"`
	Started     string   `json:"started"`
	Status      int      `json:"status"`
	Stars       int      `json:"stars"`
	Private     bool     `json:"private"`
	// Existing names the project that already has this repository.
	Existing string `json:"existing,omitempty"`
}

// GitHubRepo handles GET /admin/projects/github?repo=: the details of a
// repository, shaped as form values.
func (h *ProjectHandler) GitHubRepo(ctx *gin.Context) {
	owner, name, ok := github.ParseRepo(ctx.Query("repo"))
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "这不像是一个 GitHub 仓库地址，应该是 https://github.com/用户名/仓库名"})
		return
	}
	c, cancel := context.WithTimeout(ctx.Request.Context(), 8*time.Second)
	defer cancel()
	repo, err := h.github.Repo(c, owner, name)
	if err != nil {
		status, message := githubErrorMessage(err)
		ctx.JSON(status, gin.H{"error": message})
		return
	}

	project := importedProject{
		Name:        repo.Name,
		Description: strings.TrimSpace(repo.Description),
		Repo:        github.RepoURL(owner, name),
		Status:      model.ProjectStatusActive,
		Stars:       repo.Stars,
		Private:     repo.Private,
		Tech:        []string{},
	}
	if repo.HTMLURL != "" {
		project.Repo = repo.HTMLURL
	}
	if isWebAddress(strings.TrimSpace(repo.Homepage)) {
		project.URL = strings.TrimSpace(repo.Homepage)
	}
	if repo.Language != "" {
		project.Tech = append(project.Tech, repo.Language)
	}
	for _, topic := range repo.Topics {
		if len(project.Tech) == maxProjectTech {
			break
		}
		if !strings.EqualFold(topic, repo.Language) { // "Go" and the topic "go"
			project.Tech = append(project.Tech, topic)
		}
	}
	if !repo.CreatedAt.IsZero() {
		project.Started = repo.CreatedAt.In(adminZone).Format("2006-01")
	}
	if repo.Archived {
		project.Status = model.ProjectStatusArchived
	}
	project.Existing = h.sameRepo(model.Project{Id: -1, Repo: project.Repo})
	ctx.JSON(http.StatusOK, project)
}

// suggestion is one of the owner's repositories that is not a project yet.
type suggestion struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Language    string    `json:"language"`
	Stars       int       `json:"stars"`
	PushedAt    time.Time `json:"pushed_at"`
}

// GitHubSuggestions handles GET /admin/projects/github/suggestions: the
// owner's recently pushed public repositories that are not projects yet.
func (h *ProjectHandler) GitHubSuggestions(ctx *gin.Context) {
	user := h.githubUser()
	if user == "" {
		ctx.JSON(http.StatusOK, gin.H{"user": "", "repos": []suggestion{}})
		return
	}
	repos, err := h.ownerRepos(ctx.Request.Context(), user)
	if err != nil {
		status, message := githubErrorMessage(err)
		ctx.JSON(status, gin.H{"user": user, "error": message})
		return
	}
	known := make(map[string]bool)
	if projects, err := h.ProjectRepo.GetProjects(); err == nil {
		for _, p := range projects {
			if p.Repo != "" {
				known[repoKey(p.Repo)] = true
			}
		}
	}
	list := make([]suggestion, 0, suggestionCount)
	for _, r := range repos {
		if len(list) == suggestionCount {
			break
		}
		owner, name, ok := github.ParseRepo(r.HTMLURL)
		if !ok || known[github.Key(owner, name)] || strings.EqualFold(name, owner) { // owner/owner is the profile README
			continue
		}
		list = append(list, suggestion{Name: r.Name, Description: r.Description, URL: r.HTMLURL, Language: r.Language, Stars: r.Stars, PushedAt: r.PushedAt})
	}
	ctx.JSON(http.StatusOK, gin.H{"user": user, "repos": list})
}

// ownerRepos returns the owner's recent repositories, cached for a few minutes.
func (h *ProjectHandler) ownerRepos(ctx context.Context, user string) ([]github.Repo, error) {
	h.suggestMu.Lock()
	defer h.suggestMu.Unlock()
	if h.suggestFrom == user && time.Since(h.suggestAt) < suggestionTTL {
		return h.suggested, nil
	}
	c, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repos, err := h.github.UserRepos(c, user, 30)
	if err != nil {
		return nil, err
	}
	h.suggested, h.suggestAt, h.suggestFrom = repos, time.Now(), user
	return repos, nil
}

// githubUser is whose repositories are offered: app.github_user, else the
// owner of the content repository on GitHub.
func (h *ProjectHandler) githubUser() string {
	if user := strings.TrimSpace(h.config.App.GitHubUser); user != "" {
		return user
	}
	return github.OwnerOf(h.config.App.GitRepo)
}

func githubErrorMessage(err error) (int, string) {
	var limited *github.RateLimitError
	switch {
	case errors.Is(err, github.ErrNotFound):
		return http.StatusNotFound, "GitHub 上没有找到这个仓库（私有仓库需要在配置里填 github_token）"
	case errors.As(err, &limited):
		return http.StatusTooManyRequests, fmt.Sprintf("GitHub 接口调用次数暂时用完了，%s 之后再试；配置 github_token 可以提高上限", limited.Reset.In(adminZone).Format("15:04"))
	}
	slog.Warn("github request failed", "err", err)
	return http.StatusBadGateway, "连不上 GitHub，请稍后再试（也可以直接手动填写）"
}
