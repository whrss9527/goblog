package front

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/github"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

// sidebarProjects is how many projects the home sidebar shows.
const sidebarProjects = 3

// ProjectCatalog joins the stored projects with the live numbers of their
// repositories and the articles that introduce them: what /projects, the home
// sidebar and article pages show.
type ProjectCatalog struct {
	Projects repository.ProjectRepository
	Posts    repository.PostRepository
	// Stats is nil when app.github_stats is off.
	Stats *github.Stats
}

// ProjectCard is a project ready for display.
type ProjectCard struct {
	model.Project
	// Anchor is the element id on /projects ("project-goblog").
	Anchor string
	// Link is where the name points: the project's site, else its source, else its article.
	Link string
	// RepoURL is the source address ("" for private repositories), RepoLabel names its host.
	RepoURL   string
	RepoLabel string
	// PostTitle is set when Post names a published article.
	PostTitle    string
	StatusLabel  string
	StatusClass  string
	StartedLabel string
	// Stack is what the project is built with: the repository's main language
	// first when the author did not list it, then Tech, languages in their GitHub colour.
	Stack []TechLabel
	// Live numbers from GitHub; HasStats is false when nothing is known (yet).
	HasStats bool
	Stars    int
	Forks    int
	PushedAt time.Time
	// PushedLabel says when the last push happened ("3 天前").
	PushedLabel string
}

// TechLabel is one entry of a project's technology list.
type TechLabel struct {
	Name  string
	Color string
}

// PrimaryTech is the first entry of Stack (the zero value when there is none).
func (c ProjectCard) PrimaryTech() TechLabel {
	if len(c.Stack) == 0 {
		return TechLabel{}
	}
	return c.Stack[0]
}

// Cards returns every project ready for display, in display order.
func (c *ProjectCatalog) Cards() []ProjectCard {
	if c == nil || c.Projects == nil {
		return nil
	}
	projects, err := c.Projects.GetProjects()
	if err != nil {
		slog.Error("get projects failed", "err", err)
		return nil
	}
	cards := make([]ProjectCard, 0, len(projects))
	anchors := make(map[string]int, len(projects))
	for _, p := range projects {
		card := c.card(p)
		anchors[card.Anchor]++
		if n := anchors[card.Anchor]; n > 1 {
			card.Anchor += "-" + strconv.Itoa(n)
		}
		cards = append(cards, card)
	}
	return cards
}

// ForPost returns the projects an article introduces.
func (c *ProjectCatalog) ForPost(slug string) []ProjectCard {
	var cards []ProjectCard
	for _, card := range c.Cards() {
		if card.Post == slug && card.PostTitle != "" {
			cards = append(cards, card)
		}
	}
	return cards
}

// highlightProjects picks what the home sidebar shows: featured projects first,
// then the most recent ones that are not archived.
func highlightProjects(cards []ProjectCard, limit int) []ProjectCard {
	var picked []ProjectCard
	for _, card := range cards {
		if len(picked) == limit {
			break
		}
		if card.Status != model.ProjectStatusArchived {
			picked = append(picked, card)
		}
	}
	return picked
}

func (c *ProjectCatalog) card(p model.Project) ProjectCard {
	// hand edits of projects.json skip the admin's checks: only web addresses become links
	p.URL = webAddress(p.URL)
	if !strings.HasPrefix(p.Cover, "/") || strings.HasPrefix(p.Cover, "//") {
		p.Cover = webAddress(p.Cover)
	}
	card := ProjectCard{
		Project:      p,
		Anchor:       projectAnchor(p),
		RepoURL:      repoAddress(p.Repo),
		StatusLabel:  model.ProjectStatusLabel(p.Status),
		StatusClass:  projectStatusClass(p.Status),
		StartedLabel: monthLabel(p.Started),
	}
	card.RepoLabel = repoLabel(card.RepoURL)
	if p.Post != "" && c.Posts != nil {
		if post, err := c.Posts.GetPostByIdentity(p.Post); err == nil && post.Status == 1 {
			card.PostTitle = post.Title
		}
	}

	language := ""
	if repo, ok := c.Stats.Lookup(p.Repo); ok {
		card.HasStats = true
		card.Stars, card.Forks, card.PushedAt = repo.Stars, repo.Forks, repo.PushedAt.In(shanghai)
		card.PushedLabel = agoLabel(repo.PushedAt, time.Now())
		language = repo.Language
		if repo.Private {
			card.RepoURL, card.RepoLabel = "", "" // visitors would only get a 404
		}
	} else if c.Stats.Missing(p.Repo) {
		card.RepoURL, card.RepoLabel = "", "" // deleted or private: a dead link helps nobody
	}
	card.Stack = techLabels(language, p.Tech)

	switch {
	case p.URL != "":
		card.Link = p.URL
	case card.RepoURL != "":
		card.Link = card.RepoURL
	case card.PostTitle != "":
		card.Link = postURL(p.Post)
	}
	return card
}

// techLabels lists the technologies of a project: the repository's main
// language first unless the author already named it, then the author's list.
func techLabels(language string, tech []string) []TechLabel {
	labels := make([]TechLabel, 0, len(tech)+1)
	if language != "" {
		listed := false
		for _, t := range tech {
			listed = listed || strings.EqualFold(t, language)
		}
		if !listed {
			labels = append(labels, TechLabel{Name: language, Color: github.LanguageColor(language)})
		}
	}
	for _, t := range tech {
		labels = append(labels, TechLabel{Name: t, Color: github.LanguageColor(t)})
	}
	return labels
}

// repoAddress makes the stored source address safe to link: GitHub shorthands
// ("owner/name") become full addresses, anything that is not http(s) is dropped.
func repoAddress(repo string) string {
	if repo != "" && !strings.Contains(repo, "://") {
		if owner, name, ok := github.ParseRepo(repo); ok {
			return github.RepoURL(owner, name)
		}
		return ""
	}
	return webAddress(repo)
}

// webAddress returns s when it is an http(s) address and "" otherwise.
func webAddress(s string) string {
	if u, err := url.Parse(s); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return s
}

// repoLabel names where the source lives, for the button that links to it.
func repoLabel(repoURL string) string {
	u, err := url.Parse(repoURL)
	if repoURL == "" || err != nil {
		return ""
	}
	switch host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www."); host {
	case "github.com":
		return "GitHub"
	case "gitee.com":
		return "Gitee"
	case "gitlab.com":
		return "GitLab"
	case "codeberg.org":
		return "Codeberg"
	}
	return "源码"
}

func projectStatusClass(status int) string {
	switch status {
	case model.ProjectStatusDone:
		return "done"
	case model.ProjectStatusArchived:
		return "archived"
	}
	return "active"
}

// monthLabel turns "2026-09" into "2026 年 9 月" ("" for anything else).
func monthLabel(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d 年 %d 月", t.Year(), int(t.Month()))
}

// projectAnchor derives a readable element id from the project name:
// "My Tool" -> "project-my-tool"; names without letters or digits fall back to the id.
func projectAnchor(p model.Project) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(p.Name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		default:
			dash = true
		}
	}
	if b.Len() == 0 {
		return "project-" + strconv.Itoa(p.Id)
	}
	return "project-" + b.String()
}

// agoLabel describes how long ago t was the way the site talks: 今天 / 3 天前 / 2 个月前 / 1 年前.
func agoLabel(t, now time.Time) string {
	days := int(now.Sub(t).Hours() / 24)
	switch {
	case t.IsZero():
		return ""
	case days < 1:
		return "今天"
	case days < 2:
		return "昨天"
	case days < 31:
		return fmt.Sprintf("%d 天前", days)
	case days < 365:
		return fmt.Sprintf("%d 个月前", days/30)
	}
	return fmt.Sprintf("%d 年前", days/365)
}

// ProjectHandler serves /projects.
type ProjectHandler struct {
	Catalog *ProjectCatalog
	config  *config.Config
}

func NewProjectHandler(catalog *ProjectCatalog, config *config.Config) *ProjectHandler {
	return &ProjectHandler{Catalog: catalog, config: config}
}

// ProjectCounts sums up the list for the page header and the filter.
type ProjectCounts struct {
	Total, Active, Done, Archived, Stars int
}

// Statuses is how many different statuses occur (the filter is only worth showing for more than one).
func (c ProjectCounts) Statuses() int {
	n := 0
	for _, count := range []int{c.Active, c.Done, c.Archived} {
		if count > 0 {
			n++
		}
	}
	return n
}

func countProjects(cards []ProjectCard) ProjectCounts {
	counts := ProjectCounts{Total: len(cards)}
	for _, card := range cards {
		switch card.Status {
		case model.ProjectStatusDone:
			counts.Done++
		case model.ProjectStatusArchived:
			counts.Archived++
		default:
			counts.Active++
		}
		counts.Stars += card.Stars
	}
	return counts
}

// Projects renders /projects.
func (h *ProjectHandler) Projects(ctx *gin.Context) {
	cards := h.Catalog.Cards()
	names := make([]string, 0, len(cards))
	for _, card := range cards {
		names = append(names, card.Name)
	}
	description := "我做过的项目"
	if len(names) > 0 {
		description = view.Excerpt(fmt.Sprintf("我做过的 %d 个项目：%s", len(names), strings.Join(names, "、")), 160)
	}

	data := make(map[string]any)
	data["nav"] = "projects"
	data["title"] = "项目"
	data["description"] = description
	data["canonical"] = strings.TrimRight(h.config.App.Host, "/") + "/projects"
	data["projects"] = cards
	data["counts"] = countProjects(cards)
	if len(cards) > 0 {
		data["json_ld"] = projectsJSONLD(h.config.App, cards)
	} else {
		data["noindex"] = true
	}
	view.Render(data, ctx.Writer, "projects", h.config.App)
}

// projectsJSONLD describes the page as a collection of software projects.
func projectsJSONLD(conf *config.AppConfig, cards []ProjectCard) template.JS {
	page := strings.TrimRight(conf.Host, "/") + "/projects"
	items := make([]map[string]any, 0, len(cards))
	for i, card := range cards {
		item := map[string]any{
			"@type": "SoftwareSourceCode",
			"name":  card.Name,
			"url":   page + "#" + card.Anchor,
		}
		if card.Description != "" {
			item["description"] = card.Description
		}
		if card.RepoURL != "" {
			item["codeRepository"] = card.RepoURL
		}
		if card.URL != "" {
			item["sameAs"] = card.URL
		}
		if len(card.Stack) > 0 {
			names := make([]string, 0, len(card.Stack))
			for _, t := range card.Stack {
				names = append(names, t.Name)
			}
			item["keywords"] = strings.Join(names, ",")
			if card.Stack[0].Color != "" {
				item["programmingLanguage"] = card.Stack[0].Name
			}
		}
		if card.Started != "" {
			item["dateCreated"] = card.Started
		}
		items = append(items, map[string]any{"@type": "ListItem", "position": i + 1, "item": item})
	}
	return jsonLD(map[string]any{
		"@context":   "https://schema.org",
		"@type":      "CollectionPage",
		"name":       "项目",
		"url":        page,
		"inLanguage": "zh-CN",
		"author":     publisherLD(conf),
		"mainEntity": map[string]any{"@type": "ItemList", "numberOfItems": len(items), "itemListElement": items},
	})
}
