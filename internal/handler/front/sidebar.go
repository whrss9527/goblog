package front

import (
	"sort"

	"goblog/internal/pkg/model"
)

const (
	sidebarHotPosts = 5
	sidebarTopTags  = 14
)

// Sidebar is the discovery column shown next to the post list on wide screens.
type Sidebar struct {
	PostCount int
	WordCount int
	ViewCount int
	TagCount  int
	SinceYear int
	HotPosts  []*model.Post
	TopTags   []model.Tag
}

func buildSidebar(posts []*model.Post, tagMap map[int]model.Tag) Sidebar {
	sb := Sidebar{PostCount: len(posts)}
	for _, p := range posts {
		sb.WordCount += p.WordCount
		sb.ViewCount += p.Views
		if year := p.CreatedAt.Year(); !p.CreatedAt.IsZero() && (sb.SinceYear == 0 || year < sb.SinceYear) {
			sb.SinceYear = year
		}
	}

	hot := make([]*model.Post, len(posts))
	copy(hot, posts)
	sort.SliceStable(hot, func(i, j int) bool { return hot[i].Views > hot[j].Views })
	if len(hot) > sidebarHotPosts {
		hot = hot[:sidebarHotPosts]
	}
	sb.HotPosts = hot

	tags := make([]model.Tag, 0, len(tagMap))
	for _, tag := range tagMap {
		if tag.Count > 0 {
			tags = append(tags, tag)
		}
	}
	sb.TagCount = len(tags)
	sort.SliceStable(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		return tags[i].Id < tags[j].Id
	})
	if len(tags) > sidebarTopTags {
		tags = tags[:sidebarTopTags]
	}
	sb.TopTags = tags
	return sb
}
