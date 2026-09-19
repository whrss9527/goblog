package front

import (
	"testing"
	"time"

	"goblog/internal/pkg/model"
)

func TestComputeStats(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	at := func(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, loc) }
	posts := []*model.Post{
		{Id: "a", Title: "A", CreatedAt: at(2022, 11, 24, 23), WordCount: 1000, Views: 50, Likes: 1, CategoryId: 1}, // Thursday night
		{Id: "b", Title: "B", CreatedAt: at(2023, 1, 3, 10), WordCount: 3000, Views: 200, Likes: 2, CategoryId: 1},  // Tuesday
		{Id: "c", Title: "C", CreatedAt: at(2023, 1, 10, 10), WordCount: 200, Views: 10, CategoryId: 2},             // Tuesday
		{Id: "d", Title: "D", CreatedAt: at(2025, 6, 1, 2), WordCount: 800, Views: 120, CategoryId: 1},              // Sunday night
	}
	categories := []model.Category{{Id: 1, Name: "技术"}, {Id: 2, Name: "生活"}, {Id: 3, Name: "空"}}
	books := []model.Book{{Status: model.BookStatusFinished}, {Status: model.BookStatusFinished}, {Status: model.BookStatusReading}, {Status: model.BookStatusWish}}
	now := at(2025, 6, 11, 12)

	st := computeStats(posts, categories, 7, books, now, loc)

	if st.PostCount != 4 || st.WordCount != 5000 || st.ViewCount != 380 || st.LikeCount != 3 || st.TagCount != 7 {
		t.Errorf("totals wrong: %+v", st)
	}
	if st.AvgWords != 1250 || st.BooksFinished != 2 || st.BooksReading != 1 {
		t.Errorf("avg/books wrong: avg=%d finished=%d reading=%d", st.AvgWords, st.BooksFinished, st.BooksReading)
	}
	if st.FirstPost.Id != "a" || st.LastPost.Id != "d" || st.DaysSinceLast != 10 {
		t.Errorf("first/last wrong: %s %s days=%d", st.FirstPost.Id, st.LastPost.Id, st.DaysSinceLast)
	}
	if st.Longest.Id != "b" || st.Shortest.Id != "c" {
		t.Errorf("longest/shortest wrong: %s %s", st.Longest.Id, st.Shortest.Id)
	}
	if st.LongestGapFrom.Id != "c" || st.LongestGapTo.Id != "d" || st.LongestGapDays < 870 {
		t.Errorf("longest gap wrong: %+v", st)
	}
	// years are contiguous from the first post to now, empty years included
	if len(st.Years) != 4 || st.Years[0].Year != 2022 || st.Years[2].Year != 2024 || st.Years[2].Posts != 0 || st.Years[1].Percent != 100 {
		t.Errorf("years wrong: %+v", st.Years)
	}
	if st.NightOwl != 50 {
		t.Errorf("night owl share = %d, want 50", st.NightOwl)
	}
	if st.BusiestDay != "周二" || st.PeakHour != 10 {
		t.Errorf("busiest day/hour wrong: %s %d", st.BusiestDay, st.PeakHour)
	}
	if len(st.Categories) != 2 || st.Categories[0].Name != "技术" || st.Categories[0].Percent != 75 {
		t.Errorf("categories wrong: %+v", st.Categories)
	}
	if len(st.TopViewed) != 4 || st.TopViewed[0].Post.Id != "b" || st.TopViewed[0].Percent != 100 {
		t.Errorf("top viewed wrong: %+v", st.TopViewed)
	}
	if st.Weekdays[1].Count != 2 || st.Months[0].Count != 2 || st.Hours[23].Count != 1 {
		t.Errorf("buckets wrong")
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	st := computeStats(nil, nil, 0, nil, time.Now(), time.UTC)
	if st.PostCount != 0 || st.FirstPost != nil || len(st.Years) != 0 {
		t.Errorf("empty blog must not panic or invent data: %+v", st)
	}
}

func TestOnThisDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	at := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 9, 0, 0, 0, loc) }
	posts := []*model.Post{
		{Id: "exact-2023", CreatedAt: at(2023, 9, 19)},
		{Id: "exact-2022", CreatedAt: at(2022, 9, 19)},
		{Id: "near", CreatedAt: at(2023, 9, 21)},
		{Id: "this-year", CreatedAt: at(2026, 9, 19)},
		{Id: "far", CreatedAt: at(2023, 3, 1)},
	}

	got := onThisDay(posts, at(2026, 9, 19), loc)
	if got == nil || !got.Exact || len(got.Posts) != 2 || got.Posts[0].Id != "exact-2023" {
		t.Fatalf("exact matches from earlier years expected, got %+v", got)
	}

	got = onThisDay(posts, at(2026, 9, 23), loc)
	if got == nil || got.Exact || len(got.Posts) != 1 || got.Posts[0].Id != "near" {
		t.Fatalf("±3 day window expected, got %+v", got)
	}

	if got = onThisDay(posts, at(2026, 6, 1), loc); got != nil {
		t.Fatalf("nothing around June 1st, got %+v", got)
	}
}
