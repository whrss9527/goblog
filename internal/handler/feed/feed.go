package feed

import (
	"sort"
	"time"

	"github.com/gorilla/feeds"
)

type Article struct {
	Title     string
	Link      string
	Id        string
	Published time.Time
	Created   time.Time
	Updated   time.Time
	Content   string
	Summary   string
}

type FeedConfig struct {
	Title string
	Host  string
	// Description becomes the feed's subtitle.
	Description string
	// Updated is when the feed's content last changed; the generation time when zero.
	Updated time.Time
}

func GenerateFeed(articles []Article, cfg FeedConfig) (string, error) {
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Published.After(articles[j].Published)
	})

	now := time.Now()

	feed := &feeds.Feed{
		Title:       cfg.Title,
		Link:        &feeds.Link{Href: cfg.Host + "/"},
		Description: cfg.Description,
		Author:      &feeds.Author{Name: cfg.Title},
		Created:     now,
		Updated:     cfg.Updated,
	}

	// 添加文章到 feed
	feed.Items = make([]*feeds.Item, len(articles))
	for i, article := range articles {
		feed.Items[i] = &feeds.Item{
			Title:       article.Title,
			Link:        &feeds.Link{Href: article.Link},
			Id:          article.Id,
			Created:     article.Created,
			Updated:     article.Updated,
			Content:     article.Content,
			Description: article.Summary,
		}
	}

	// 生成 Atom 格式的 feed
	return feed.ToAtom()
}
