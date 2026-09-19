package front

import (
	"strings"
	"testing"

	"goblog/internal/pkg/model"
)

func searchFixture() ([]*model.Post, map[int]string, map[int]string) {
	posts := []*model.Post{
		{Id: "deadlock", Identity: "golang-deadlock", Title: "死锁日记：手写 GoLang 上报队列", Description: "一次内存溢出", Content: "我们用 channel 实现队列。\n```go\nfunc main() {}\n```\n后来发现 goroutine 泄漏。", CategoryId: 1, TagIds: []int{1}, CreatedAt: day(3), Views: 40},
		{Id: "gorm", Identity: "gorm-save", Title: "【Gorm】Save 方法更新踩坑记录", Description: "gorm 的 Save", Content: "Save 会更新所有字段，和 golang 零值有关。", CategoryId: 1, TagIds: []int{2}, CreatedAt: day(2), Views: 90},
		{Id: "bike", Identity: "my bike & me", Title: "我的自行车", Description: "通勤 <b>神器</b>", Content: "周末骑车去了 go 公园。", CategoryId: 2, CreatedAt: day(1), Views: 10},
	}
	return posts, map[int]string{1: "技术", 2: "生活"}, map[int]string{1: "golang", 2: "gorm"}
}

func TestSearchPostsRanking(t *testing.T) {
	posts, cats, tags := searchFixture()

	items, total := searchPosts(posts, cats, tags, "golang", 10)
	if total != 2 || len(items) != 2 {
		t.Fatalf("total=%d len=%d, want 2", total, len(items))
	}
	if items[0].URL != "/posts/golang-deadlock" {
		t.Errorf("title+tag match must rank first, got %s", items[0].URL)
	}
	if !strings.Contains(items[0].TitleHTML, "<mark>GoLang</mark>") {
		t.Errorf("title must be highlighted case-insensitively: %s", items[0].TitleHTML)
	}
	if !strings.Contains(items[1].SnippetHTML, "<mark>golang</mark>") {
		t.Errorf("snippet should come from the body around the match: %s", items[1].SnippetHTML)
	}
}

func TestSearchPostsAndSemantics(t *testing.T) {
	posts, cats, tags := searchFixture()

	if _, total := searchPosts(posts, cats, tags, "save golang", 10); total != 1 {
		t.Errorf("all terms must match: total=%d, want 1", total)
	}
	if _, total := searchPosts(posts, cats, tags, "save 自行车", 10); total != 0 {
		t.Errorf("terms matching different posts must not produce a hit: total=%d", total)
	}
	if _, total := searchPosts(posts, cats, tags, "生活", 10); total != 1 {
		t.Errorf("category names are searchable: total=%d", total)
	}
	if _, total := searchPosts(posts, cats, tags, "zzz", 10); total != 0 {
		t.Errorf("no match expected, total=%d", total)
	}
}

func TestSearchEscapesHTMLAndRegexp(t *testing.T) {
	posts, cats, tags := searchFixture()

	items, total := searchPosts(posts, cats, tags, "神器", 10)
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if strings.Contains(items[0].SnippetHTML, "<b>") {
		t.Errorf("post text must be escaped: %s", items[0].SnippetHTML)
	}
	if items[0].URL != "/posts/my%20bike%20&%20me" {
		t.Errorf("identity must be path-escaped, got %s", items[0].URL)
	}
	// regexp metacharacters in the query are literals, not patterns
	if _, total := searchPosts(posts, cats, tags, ".*", 10); total != 0 {
		t.Errorf("'.*' must be treated literally, total=%d", total)
	}
	if _, total := searchPosts(posts, cats, tags, "main(", 10); total != 1 {
		t.Errorf("'main(' is a literal substring of one post, total=%d", total)
	}
	if _, total := searchPosts(posts, cats, tags, "[a-z", 10); total != 0 {
		t.Errorf("an unbalanced bracket must neither panic nor match, total=%d", total)
	}
}

func TestSearchLimitAndHot(t *testing.T) {
	posts, cats, tags := searchFixture()

	items, total := searchPosts(posts, cats, tags, "o", 2)
	if total != 3 || len(items) != 2 {
		t.Errorf("total=%d len=%d, want 3 and 2", total, len(items))
	}
	hot := hotPosts(posts, cats, tags, 2)
	if len(hot) != 2 || hot[0].URL != "/posts/gorm-save" {
		t.Errorf("hot posts must be ordered by views: %+v", hot)
	}
}

func TestNormalizeQuery(t *testing.T) {
	if got := normalizeQuery("  go   优雅 \n 退出 "); got != "go 优雅 退出" {
		t.Errorf("normalizeQuery = %q", got)
	}
	if got := normalizeQuery(strings.Repeat("长", 80)); len([]rune(got)) != searchMaxQueryRunes {
		t.Errorf("query must be capped at %d runes, got %d", searchMaxQueryRunes, len([]rune(got)))
	}
}

func TestHighlightHTML(t *testing.T) {
	if got := highlightHTML("a<b>&c", nil); got != "a&lt;b&gt;&amp;c" {
		t.Errorf("nil regexp must still escape: %q", got)
	}
}
