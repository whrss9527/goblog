package front

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

const statsTopViewed = 10

// StatsHandler renders /stats, a playful "blog in numbers" page.
type StatsHandler struct {
	PostRepo     repository.PostRepository
	CategoryRepo repository.CategoryRepository
	TagRepo      repository.TagRepository
	BookRepo     repository.BookRepository
	config       *config.Config
}

func NewStatsHandler(postRepo repository.PostRepository, categoryRepo repository.CategoryRepository, tagRepo repository.TagRepository, bookRepo repository.BookRepository, config *config.Config) *StatsHandler {
	return &StatsHandler{PostRepo: postRepo, CategoryRepo: categoryRepo, TagRepo: tagRepo, BookRepo: bookRepo, config: config}
}

// Bucket is one bar of a chart; Percent is relative to the tallest bar.
type Bucket struct {
	Label   string
	Count   int
	Percent int
}

// YearStat aggregates one calendar year.
type YearStat struct {
	Year    int
	Posts   int
	Words   int
	Views   int
	Percent int
}

// CategoryStat is one slice of the category distribution.
type CategoryStat struct {
	Id      int
	Name    string
	Count   int
	Percent float64
}

// RankedPost is a post with a bar length relative to the list's maximum.
type RankedPost struct {
	Post    *model.Post
	Percent int
}

// SiteStats is everything /stats shows.
type SiteStats struct {
	PostCount     int
	WordCount     int
	ViewCount     int
	LikeCount     int
	TagCount      int
	AvgWords      int
	ReadingHours  float64
	BooksFinished int
	BooksReading  int

	FirstPost      *model.Post
	LastPost       *model.Post
	DaysSinceFirst int
	DaysSinceLast  int

	LongestGapDays int
	LongestGapFrom *model.Post
	LongestGapTo   *model.Post

	Years      []YearStat
	Months     []Bucket
	Weekdays   []Bucket
	Hours      []Bucket
	NightOwl   int // share of posts published between 22:00 and 05:59, in percent
	BusiestDay string
	PeakHour   int

	Categories []CategoryStat
	TopViewed  []RankedPost
	Longest    *model.Post
	Shortest   *model.Post
}

func percentOf(n, max int) int {
	if max <= 0 || n <= 0 {
		return 0
	}
	p := n * 100 / max
	if p < 2 {
		p = 2 // keep tiny bars visible
	}
	return p
}

// computeStats derives all numbers from the published posts. now and loc are
// parameters so the result is deterministic under test.
func computeStats(posts []*model.Post, categories []model.Category, tagCount int, books []model.Book, now time.Time, loc *time.Location) SiteStats {
	st := SiteStats{PostCount: len(posts), TagCount: tagCount}
	for _, b := range books {
		switch b.Status {
		case model.BookStatusFinished:
			st.BooksFinished++
		case model.BookStatusReading:
			st.BooksReading++
		}
	}
	if len(posts) == 0 {
		return st
	}

	sorted := make([]*model.Post, len(posts))
	copy(sorted, posts)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].CreatedAt.Before(sorted[j].CreatedAt) })

	years := make(map[int]*YearStat)
	months := make([]int, 12)
	weekdays := make([]int, 7) // Monday first
	hours := make([]int, 24)
	perCategory := make(map[int]int)
	night := 0

	for i, p := range sorted {
		t := p.CreatedAt.In(loc)
		st.WordCount += p.WordCount
		st.ViewCount += p.Views
		st.LikeCount += p.Likes

		ys, ok := years[t.Year()]
		if !ok {
			ys = &YearStat{Year: t.Year()}
			years[t.Year()] = ys
		}
		ys.Posts++
		ys.Words += p.WordCount
		ys.Views += p.Views

		months[int(t.Month())-1]++
		weekdays[(int(t.Weekday())+6)%7]++
		hours[t.Hour()]++
		if t.Hour() >= 22 || t.Hour() < 6 {
			night++
		}
		perCategory[p.CategoryId]++

		if st.Longest == nil || p.WordCount > st.Longest.WordCount {
			st.Longest = p
		}
		if st.Shortest == nil || p.WordCount < st.Shortest.WordCount {
			st.Shortest = p
		}
		if i > 0 {
			gap := int(p.CreatedAt.Sub(sorted[i-1].CreatedAt).Hours() / 24)
			if gap > st.LongestGapDays {
				st.LongestGapDays = gap
				st.LongestGapFrom = sorted[i-1]
				st.LongestGapTo = p
			}
		}
	}

	st.FirstPost = sorted[0]
	st.LastPost = sorted[len(sorted)-1]
	st.DaysSinceFirst = int(now.Sub(st.FirstPost.CreatedAt).Hours() / 24)
	st.DaysSinceLast = int(now.Sub(st.LastPost.CreatedAt).Hours() / 24)
	st.AvgWords = st.WordCount / len(posts)
	st.ReadingHours = float64(st.WordCount) / 400 / 60
	st.NightOwl = night * 100 / len(posts)

	// years, oldest first, with bars relative to the most productive year
	maxPosts := 0
	for _, ys := range years {
		if ys.Posts > maxPosts {
			maxPosts = ys.Posts
		}
	}
	for y := st.FirstPost.CreatedAt.In(loc).Year(); y <= now.In(loc).Year(); y++ {
		ys, ok := years[y]
		if !ok {
			ys = &YearStat{Year: y}
		}
		ys.Percent = percentOf(ys.Posts, maxPosts)
		st.Years = append(st.Years, *ys)
	}

	st.Months = buckets(months, func(i int) string { return fmt.Sprintf("%d月", i+1) })
	weekdayNames := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	st.Weekdays = buckets(weekdays, func(i int) string { return weekdayNames[i] })
	st.Hours = buckets(hours, func(i int) string { return fmt.Sprintf("%d", i) })
	st.BusiestDay = weekdayNames[argmax(weekdays)]
	st.PeakHour = argmax(hours)

	for _, c := range categories {
		if n := perCategory[c.Id]; n > 0 {
			st.Categories = append(st.Categories, CategoryStat{Id: c.Id, Name: c.Name, Count: n, Percent: float64(n) * 100 / float64(len(posts))})
		}
	}
	sort.SliceStable(st.Categories, func(i, j int) bool { return st.Categories[i].Count > st.Categories[j].Count })

	byViews := make([]*model.Post, len(posts))
	copy(byViews, posts)
	sort.SliceStable(byViews, func(i, j int) bool { return byViews[i].Views > byViews[j].Views })
	if len(byViews) > statsTopViewed {
		byViews = byViews[:statsTopViewed]
	}
	for _, p := range byViews {
		st.TopViewed = append(st.TopViewed, RankedPost{Post: p, Percent: percentOf(p.Views, byViews[0].Views)})
	}
	return st
}

func buckets(counts []int, label func(int) string) []Bucket {
	max := 0
	for _, n := range counts {
		if n > max {
			max = n
		}
	}
	out := make([]Bucket, len(counts))
	for i, n := range counts {
		out[i] = Bucket{Label: label(i), Count: n, Percent: percentOf(n, max)}
	}
	return out
}

func argmax(counts []int) int {
	best := 0
	for i, n := range counts {
		if n > counts[best] {
			best = i
		}
	}
	return best
}

// OnThisDay is the "N years ago today" box.
type OnThisDay struct {
	Exact bool // true: same month and day, false: within the surrounding week
	Posts []*model.Post
}

// onThisDay finds posts from earlier years published on today's date; when
// there are none it widens the window to three days either side.
func onThisDay(posts []*model.Post, now time.Time, loc *time.Location) *OnThisDay {
	today := now.In(loc)
	var exact, near []*model.Post
	for _, p := range posts {
		t := p.CreatedAt.In(loc)
		if t.Year() >= today.Year() {
			continue
		}
		if t.Month() == today.Month() && t.Day() == today.Day() {
			exact = append(exact, p)
			continue
		}
		anniversary := time.Date(today.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		diff := anniversary.Sub(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc))
		if diff < 0 {
			diff = -diff
		}
		if diff <= 3*24*time.Hour {
			near = append(near, p)
		}
	}
	pick := exact
	if len(pick) == 0 {
		pick = near
	}
	if len(pick) == 0 {
		return nil
	}
	sort.SliceStable(pick, func(i, j int) bool { return pick[i].CreatedAt.After(pick[j].CreatedAt) })
	if len(pick) > 3 {
		pick = pick[:3]
	}
	return &OnThisDay{Exact: len(exact) > 0, Posts: pick}
}

func (h *StatsHandler) Stats(ctx *gin.Context) {
	posts, _, err := h.PostRepo.GetPosts(repository.PostParams{Page: 1})
	if err != nil {
		slog.Error("stats: get posts failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	categories, _ := h.CategoryRepo.GetCategories()
	tags, _ := h.TagRepo.GetTags()
	tagCount := 0
	for _, t := range tags {
		if t.Count > 0 {
			tagCount++
		}
	}
	books, _ := h.BookRepo.GetBooks()

	now := time.Now()
	heatmap, err := os.ReadFile("./heatmap.txt")
	if err != nil || !json.Valid(heatmap) {
		heatmap = []byte("[]")
	}

	data := make(map[string]any)
	data["nav"] = "stats"
	data["title"] = "博客数据"
	data["description"] = "用数字看这个博客：文章、字数、阅读量、发文习惯与热门文章。"
	data["canonical"] = h.config.App.Host + "/stats"
	data["stats"] = computeStats(posts, categories, tagCount, books, now, shanghai)
	data["on_this_day"] = onThisDay(posts, now, shanghai)
	data["heatmap"] = template.JS(heatmap)
	view.Render(data, ctx.Writer, "stats", h.config.App)
}
