package front

import (
	"testing"
	"time"

	"goblog/internal/pkg/model"
)

func day(d int) time.Time { return time.Date(2024, 1, d, 0, 0, 0, 0, time.UTC) }

func TestNeighbours(t *testing.T) {
	a := &model.Post{Id: "a", CreatedAt: day(1)}
	b := &model.Post{Id: "b", CreatedAt: day(2)}
	c := &model.Post{Id: "c", CreatedAt: day(3)}
	posts := []*model.Post{b, c, a} // deliberately unsorted

	newer, older := neighbours(posts, *b)
	if newer != c || older != a {
		t.Errorf("middle post: newer=%v older=%v", newer, older)
	}
	if newer, older = neighbours(posts, *c); newer != nil || older != b {
		t.Errorf("newest post: newer=%v older=%v", newer, older)
	}
	if newer, older = neighbours(posts, *a); newer != b || older != nil {
		t.Errorf("oldest post: newer=%v older=%v", newer, older)
	}
	if newer, older = neighbours(posts, model.Post{Id: "zzz"}); newer != nil || older != nil {
		t.Errorf("unknown post must have no neighbours")
	}
}

func TestRelatedPosts(t *testing.T) {
	current := model.Post{Id: "cur", CategoryId: 1, TagIds: []int{1, 2}}
	twoTags := &model.Post{Id: "two-tags", CategoryId: 2, TagIds: []int{1, 2}, CreatedAt: day(1)}
	oneTagSameCat := &model.Post{Id: "one-tag-same-cat", CategoryId: 1, TagIds: []int{2, 9}, CreatedAt: day(2)}
	sameCatOld := &model.Post{Id: "same-cat-old", CategoryId: 1, CreatedAt: day(3)}
	sameCatNew := &model.Post{Id: "same-cat-new", CategoryId: 1, CreatedAt: day(4)}
	unrelated := &model.Post{Id: "unrelated", CategoryId: 3, TagIds: []int{7}, CreatedAt: day(5)}
	self := &model.Post{Id: "cur", CategoryId: 1, TagIds: []int{1, 2}, CreatedAt: day(6)}

	got := relatedPosts([]*model.Post{unrelated, sameCatOld, self, oneTagSameCat, sameCatNew, twoTags}, current, 3)

	want := []string{"two-tags", "one-tag-same-cat", "same-cat-new"}
	if len(got) != len(want) {
		t.Fatalf("got %d related posts, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].Id != id {
			t.Errorf("related[%d] = %s, want %s", i, got[i].Id, id)
		}
	}
}

func TestOutdatedYears(t *testing.T) {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	code := "intro\n```go\nfmt.Println()\n```"

	tests := []struct {
		name     string
		created  time.Time
		content  string
		category string
		want     int
	}{
		{"fresh tech post", now.AddDate(-1, 0, 0), code, "技术", 0},
		{"three year old post with code", time.Date(2023, 7, 29, 0, 0, 0, 0, time.UTC), code, "折腾", 3},
		{"anniversary not reached yet", time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC), code, "技术", 2},
		{"old tech category post without code", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), "plain", "技术", 4},
		{"old essay never gets a notice", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), "plain", "生活和思考", 0},
		{"missing date", time.Time{}, code, "技术", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := model.Post{CreatedAt: tt.created, Content: tt.content}
			if got := outdatedYears(post, tt.category, now); got != tt.want {
				t.Errorf("outdatedYears() = %d, want %d", got, tt.want)
			}
		})
	}
}
