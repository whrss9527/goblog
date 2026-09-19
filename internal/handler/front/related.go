package front

import (
	"sort"
	"strings"
	"time"

	"goblog/internal/pkg/model"
)

// outdatedAfter is the age after which technical posts get a "may be outdated" notice.
const outdatedAfter = 2 * 365 * 24 * time.Hour

// neighbours returns the chronologically adjacent posts of current: newer is
// the post published right after it, older the one right before it. posts may
// be in any order; nil means there is no such neighbour.
func neighbours(posts []*model.Post, current model.Post) (newer, older *model.Post) {
	sorted := make([]*model.Post, len(posts))
	copy(sorted, posts)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].CreatedAt.After(sorted[j].CreatedAt) })

	for i, p := range sorted {
		if p.Id != current.Id {
			continue
		}
		if i > 0 {
			newer = sorted[i-1]
		}
		if i+1 < len(sorted) {
			older = sorted[i+1]
		}
		break
	}
	return newer, older
}

// relatedPosts ranks the other posts by similarity to current: every shared
// tag counts 3, the same category 1. Ties go to the more recent post. Posts
// without any overlap are never suggested.
func relatedPosts(posts []*model.Post, current model.Post, limit int) []*model.Post {
	tagSet := make(map[int]struct{}, len(current.TagIds))
	for _, id := range current.TagIds {
		tagSet[id] = struct{}{}
	}

	type scored struct {
		post  *model.Post
		score int
	}
	var candidates []scored
	for _, p := range posts {
		if p.Id == current.Id {
			continue
		}
		score := 0
		for _, id := range p.TagIds {
			if _, ok := tagSet[id]; ok {
				score += 3
			}
		}
		if p.CategoryId == current.CategoryId {
			score++
		}
		if score > 0 {
			candidates = append(candidates, scored{post: p, score: score})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].post.CreatedAt.After(candidates[j].post.CreatedAt)
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	related := make([]*model.Post, 0, len(candidates))
	for _, c := range candidates {
		related = append(related, c.post)
	}
	return related
}

// outdatedYears reports how many full years old a technical post is once it
// passed outdatedAfter, or 0 when no notice should be shown. A post counts as
// technical when it contains a fenced code block or sits in a category whose
// name mentions 技术; essays and reading notes age just fine.
func outdatedYears(post model.Post, categoryName string, now time.Time) int {
	if post.CreatedAt.IsZero() || now.Sub(post.CreatedAt) < outdatedAfter {
		return 0
	}
	if !strings.Contains(post.Content, "```") && !strings.Contains(categoryName, "技术") {
		return 0
	}
	years := now.Year() - post.CreatedAt.Year()
	anniversary := post.CreatedAt.AddDate(years, 0, 0)
	if now.Before(anniversary) {
		years--
	}
	return years
}
