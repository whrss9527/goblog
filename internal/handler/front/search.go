package front

import (
	"html"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

const (
	searchMaxQueryRunes = 50
	searchMaxTerms      = 5
	searchLimit         = 20
	searchHotLimit      = 6
	snippetBefore       = 28
	snippetAfter        = 72
)

// SearchHandler serves the JSON API behind the instant-search palette.
type SearchHandler struct {
	PostRepo     repository.PostRepository
	CategoryRepo repository.CategoryRepository
	TagRepo      repository.TagRepository
}

func NewSearchHandler(postRepo repository.PostRepository, categoryRepo repository.CategoryRepository, tagRepo repository.TagRepository) *SearchHandler {
	return &SearchHandler{PostRepo: postRepo, CategoryRepo: categoryRepo, TagRepo: tagRepo}
}

// SearchItem is one hit. The *_html fields are already HTML-escaped with the
// matched terms wrapped in <mark>, so the client can insert them verbatim.
type SearchItem struct {
	Title       string   `json:"title"`
	TitleHTML   string   `json:"title_html"`
	URL         string   `json:"url"`
	Date        string   `json:"date"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	SnippetHTML string   `json:"snippet_html"`
	Views       int      `json:"views"`
}

// Search handles GET /api/search?q=. An empty query returns the most viewed
// posts so the palette has something useful to show before typing.
func (h *SearchHandler) Search(ctx *gin.Context) {
	posts, _, err := h.PostRepo.GetPosts(repository.PostParams{Page: 1})
	if err != nil {
		slog.Error("search: get posts failed", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	categoryNames := make(map[int]string)
	if categories, err := h.CategoryRepo.GetCategories(); err == nil {
		for _, c := range categories {
			categoryNames[c.Id] = c.Name
		}
	}
	tagNames := make(map[int]string)
	if tags, err := h.TagRepo.GetTags(); err == nil {
		for _, t := range tags {
			tagNames[t.Id] = t.Name
		}
	}

	query := normalizeQuery(ctx.Query("q"))
	ctx.Header("Cache-Control", "public, max-age=60")
	if query == "" {
		ctx.JSON(http.StatusOK, gin.H{"query": "", "total": 0, "items": []SearchItem{}, "hot": hotPosts(posts, categoryNames, tagNames, searchHotLimit)})
		return
	}
	items, total := searchPosts(posts, categoryNames, tagNames, query, searchLimit)
	ctx.JSON(http.StatusOK, gin.H{"query": query, "total": total, "items": items})
}

// Random handles GET /random: redirect to a random published post, avoiding
// the one named by ?from= so "another one" never reloads the same page.
func (h *PostHandler) Random(ctx *gin.Context) {
	posts, _, err := h.PostRepo.GetPosts(repository.PostParams{Page: 1})
	if err != nil || len(posts) == 0 {
		ctx.Redirect(http.StatusFound, "/")
		return
	}
	from := ctx.Query("from")
	candidates := posts[:0:0]
	for _, p := range posts {
		if p.Identity != from {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		candidates = posts
	}
	pick := candidates[rand.IntN(len(candidates))]
	ctx.Header("Cache-Control", "no-store")
	ctx.Redirect(http.StatusFound, postURL(pick.Identity))
}

func postURL(identity string) string {
	return "/posts/" + url.PathEscape(identity)
}

func normalizeQuery(q string) string {
	q = strings.Join(strings.Fields(q), " ")
	if utf8.RuneCountInString(q) > searchMaxQueryRunes {
		q = string([]rune(q)[:searchMaxQueryRunes])
	}
	return q
}

func hotPosts(posts []*model.Post, categoryNames, tagNames map[int]string, limit int) []SearchItem {
	sorted := make([]*model.Post, len(posts))
	copy(sorted, posts)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Views > sorted[j].Views })
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	items := make([]SearchItem, 0, len(sorted))
	for _, p := range sorted {
		items = append(items, newSearchItem(p, categoryNames, tagNames, nil))
	}
	return items
}

// searchPosts ranks posts against query. Every whitespace separated term has
// to match somewhere (AND); where it matches decides the score:
// title 10, tag 6, description 4, category 3, body 1 per hit (max 5).
func searchPosts(posts []*model.Post, categoryNames, tagNames map[int]string, query string, limit int) ([]SearchItem, int) {
	terms := strings.Fields(query)
	if len(terms) > searchMaxTerms {
		terms = terms[:searchMaxTerms]
	}
	if len(terms) == 0 {
		return []SearchItem{}, 0
	}
	termRes := make([]*regexp.Regexp, len(terms))
	quoted := make([]string, len(terms))
	for i, term := range terms {
		quoted[i] = regexp.QuoteMeta(term)
		termRes[i] = regexp.MustCompile("(?i)" + quoted[i])
	}
	// longest alternative first so "golang" wins over "go" when both are terms
	sort.SliceStable(quoted, func(i, j int) bool { return len(quoted[i]) > len(quoted[j]) })
	anyTerm := regexp.MustCompile("(?i)(" + strings.Join(quoted, "|") + ")")

	type hit struct {
		post  *model.Post
		score int
	}
	var hits []hit
	for _, p := range posts {
		var tags []string
		for _, id := range p.TagIds {
			if name, ok := tagNames[id]; ok {
				tags = append(tags, name)
			}
		}
		tagText := strings.Join(tags, " ")
		category := categoryNames[p.CategoryId]

		score := 0
		matchedAll := true
		for _, re := range termRes {
			termScore := 0
			if re.MatchString(p.Title) {
				termScore += 10
			}
			if re.MatchString(tagText) {
				termScore += 6
			}
			if re.MatchString(p.Description) {
				termScore += 4
			}
			if re.MatchString(category) {
				termScore += 3
			}
			if n := len(re.FindAllStringIndex(p.Content, 5)); n > 0 {
				termScore += n
			}
			if termScore == 0 {
				matchedAll = false
				break
			}
			score += termScore
		}
		if matchedAll {
			hits = append(hits, hit{post: p, score: score})
		}
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].post.CreatedAt.After(hits[j].post.CreatedAt)
	})

	total := len(hits)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	items := make([]SearchItem, 0, len(hits))
	for _, h := range hits {
		items = append(items, newSearchItem(h.post, categoryNames, tagNames, anyTerm))
	}
	return items, total
}

func newSearchItem(p *model.Post, categoryNames, tagNames map[int]string, re *regexp.Regexp) SearchItem {
	tags := make([]string, 0, len(p.TagIds))
	for _, id := range p.TagIds {
		if name, ok := tagNames[id]; ok {
			tags = append(tags, name)
		}
	}
	return SearchItem{
		Title:       p.Title,
		TitleHTML:   highlightHTML(p.Title, re),
		URL:         postURL(p.Identity),
		Date:        p.CreatedAt.Format("2006-01-02"),
		Category:    categoryNames[p.CategoryId],
		Tags:        tags,
		SnippetHTML: highlightHTML(snippet(p, re), re),
		Views:       p.Views,
	}
}

// snippet returns a short plain-text window around the first match in the
// body, falling back to the description / the opening of the post.
func snippet(p *model.Post, re *regexp.Regexp) string {
	fallback := view.Excerpt(p.Description, snippetBefore+snippetAfter)
	if fallback == "" {
		fallback = view.Excerpt(p.Content, snippetBefore+snippetAfter)
	}
	if re == nil {
		return fallback
	}
	plain := view.Excerpt(p.Content, 0)
	loc := re.FindStringIndex(plain)
	if loc == nil {
		return fallback
	}

	start := loc[0]
	for i := 0; i < snippetBefore && start > 0; i++ {
		_, size := utf8.DecodeLastRuneInString(plain[:start])
		start -= size
	}
	end := loc[1]
	for i := 0; i < snippetAfter && end < len(plain); i++ {
		_, size := utf8.DecodeRuneInString(plain[end:])
		end += size
	}
	out := strings.TrimSpace(plain[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(plain) {
		out += "…"
	}
	return out
}

// highlightHTML escapes text for HTML and wraps every match of re in <mark>.
func highlightHTML(text string, re *regexp.Regexp) string {
	if re == nil {
		return html.EscapeString(text)
	}
	var b strings.Builder
	last := 0
	for _, loc := range re.FindAllStringIndex(text, -1) {
		if loc[0] == loc[1] {
			continue
		}
		b.WriteString(html.EscapeString(text[last:loc[0]]))
		b.WriteString("<mark>")
		b.WriteString(html.EscapeString(text[loc[0]:loc[1]]))
		b.WriteString("</mark>")
		last = loc[1]
	}
	b.WriteString(html.EscapeString(text[last:]))
	return b.String()
}
